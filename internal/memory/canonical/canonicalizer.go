package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

type runState struct {
	log        Log
	requestIDs map[string]struct{}
}

type requestState struct {
	requestID string
	runID     string
	sessionID string
	taskID    string
	method    string
	path      string

	requestBody  bytes.Buffer
	responseBody bytes.Buffer

	responseOffset int
	responseChunks []chunkInfo

	sse sseParser

	assistant       messageBuilder
	toolCalls       map[string]*toolCallBuilder
	toolCallOrder   []*toolCallBuilder
	outputTextSeen  bool
	responseHandled bool

	requestModel       string
	requestTemperature *float64
	requestMaxTokens   *int
	responseModel      string
	responseUsage      Usage
	responseUsageSet   bool
}

type chunkInfo struct {
	requestID   string
	eventIndex  int
	eventType   string
	seq         int
	streamStart int
	streamEnd   int
}

type messageBuilder struct {
	role    string
	content strings.Builder
	length  int
	spans   []ProvenanceSpan
}

type toolCallBuilder struct {
	ID        string
	Name      string
	Arguments string
	length    int
	spans     []ProvenanceSpan
}

func Canonicalize(events []trace.Event) ([]Log, error) {
	if events == nil {
		return nil, nil
	}

	runStates := make(map[string]*runState)
	runOrder := []string{}
	requests := make(map[string]*requestState)

	for idx, ev := range events {
		req := ensureRequestState(requests, ev)
		if req != nil {
			updateRequestIDs(req, ev)
		}

		switch ev.Type {
		case trace.EventRequestStart:
			handleRequestStart(req, ev)
		case trace.EventRequestBody:
			handleRequestBody(req, ev)
		case trace.EventRequestEnd:
			if req != nil {
				handleRequestEnd(req, runStates, &runOrder)
			}
		case trace.EventResponseStart:
			handleResponseStart(req, ev, runStates, &runOrder)
		case trace.EventResponseBody:
			handleResponseBody(req, ev, idx, runStates, &runOrder)
		case trace.EventResponseEnd:
			if req != nil {
				handleResponseEnd(req, runStates, &runOrder)
			}
		}
	}

	for _, req := range requests {
		if req == nil || req.responseHandled {
			continue
		}
		handleResponseEnd(req, runStates, &runOrder)
	}

	logs := make([]Log, 0, len(runOrder))
	for _, runID := range runOrder {
		if run := runStates[runID]; run != nil {
			logs = append(logs, run.log)
		}
	}
	return logs, nil
}

func CanonicalizeWAL(path string) ([]Log, error) {
	events, err := trace.Replay(path)
	if err != nil {
		return nil, err
	}
	return Canonicalize(events)
}

func ensureRequestState(requests map[string]*requestState, ev trace.Event) *requestState {
	if strings.TrimSpace(ev.RequestID) == "" {
		return nil
	}
	req := requests[ev.RequestID]
	if req == nil {
		req = &requestState{requestID: ev.RequestID}
		requests[ev.RequestID] = req
	}
	updateRequestIDs(req, ev)
	return req
}

func updateRequestIDs(req *requestState, ev trace.Event) {
	if req == nil {
		return
	}
	if req.runID == "" && ev.RunID != "" {
		req.runID = ev.RunID
	}
	if req.sessionID == "" && ev.SessionID != "" {
		req.sessionID = ev.SessionID
	}
	if req.taskID == "" && ev.TaskID != "" {
		req.taskID = ev.TaskID
	}
}

func ensureRunState(runStates map[string]*runState, runOrder *[]string, runID string) *runState {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil
	}
	if runStates[runID] != nil {
		return runStates[runID]
	}
	state := &runState{
		log: Log{
			SchemaVersion: SchemaVersion,
			RunID:         runID,
		},
		requestIDs: make(map[string]struct{}),
	}
	runStates[runID] = state
	*runOrder = append(*runOrder, runID)
	return state
}

func handleRequestStart(req *requestState, ev trace.Event) {
	if req == nil {
		return
	}
	req.method = ev.Method
	req.path = ev.Path
}

func handleRequestBody(req *requestState, ev trace.Event) {
	if req == nil {
		return
	}
	if len(ev.Body) == 0 {
		return
	}
	_, _ = req.requestBody.Write(ev.Body)
}

func handleRequestEnd(req *requestState, runStates map[string]*runState, runOrder *[]string) {
	if req == nil {
		return
	}
	run := ensureRunState(runStates, runOrder, req.runID)
	if run == nil {
		return
	}
	applyRunIDs(run, req)
	addRequestID(run, req.requestID)

	messages, results, meta := parseRequestBody(req.requestBody.Bytes())
	applyRequestMetadata(run, req, meta)

	for _, msg := range messages {
		run.log.Items = append(run.log.Items, Item{
			Type:    ItemMessage,
			Message: &msg,
		})
	}
	for _, res := range results {
		run.log.Items = append(run.log.Items, Item{
			Type:       ItemToolResult,
			ToolResult: &res,
		})
	}
}

func handleResponseStart(req *requestState, ev trace.Event, runStates map[string]*runState, runOrder *[]string) {
	if req == nil {
		return
	}
	run := ensureRunState(runStates, runOrder, req.runID)
	if run == nil {
		return
	}
	applyRunIDs(run, req)
	addRequestID(run, req.requestID)
	if run.log.Metadata.RequestMethod == "" && req.method != "" {
		run.log.Metadata.RequestMethod = req.method
	}
	if run.log.Metadata.RequestPath == "" && req.path != "" {
		run.log.Metadata.RequestPath = req.path
	}
	if ev.Status != 0 {
		run.log.Metadata.Status = ev.Status
	}
}

func handleResponseBody(req *requestState, ev trace.Event, eventIndex int, runStates map[string]*runState, runOrder *[]string) {
	if req == nil {
		return
	}
	if len(ev.Body) == 0 {
		return
	}

	chunk := chunkInfo{
		requestID:   req.requestID,
		eventIndex:  eventIndex,
		eventType:   ev.Type,
		seq:         ev.Seq,
		streamStart: req.responseOffset,
		streamEnd:   req.responseOffset + len(ev.Body),
	}
	req.responseChunks = append(req.responseChunks, chunk)

	_, _ = req.responseBody.Write(ev.Body)

	data := req.sse.Feed(ev.Body, req.responseOffset)
	req.responseOffset += len(ev.Body)

	run := ensureRunState(runStates, runOrder, req.runID)
	if run != nil {
		applyRunIDs(run, req)
		addRequestID(run, req.requestID)
	}
	for _, entry := range data {
		handleSSEData(req, run, entry)
	}
}

func handleResponseEnd(req *requestState, runStates map[string]*runState, runOrder *[]string) {
	if req == nil || req.responseHandled {
		return
	}
	req.responseHandled = true
	run := ensureRunState(runStates, runOrder, req.runID)
	if run != nil {
		applyRunIDs(run, req)
		addRequestID(run, req.requestID)
	}

	for _, entry := range req.sse.Flush() {
		handleSSEData(req, run, entry)
	}

	if req.assistant.length == 0 {
		if content, ok := parseNonStreamingResponse(req.responseBody.Bytes()); ok {
			req.assistant.append(content, nil)
		}
	}

	if run != nil {
		applyResponseMetadata(run, req)
		appendResponseItems(run, req)
	}
}

func applyRunIDs(run *runState, req *requestState) {
	if run == nil || req == nil {
		return
	}
	if run.log.SessionID == "" && req.sessionID != "" {
		run.log.SessionID = req.sessionID
	}
	if run.log.TaskID == "" && req.taskID != "" {
		run.log.TaskID = req.taskID
	}
	if run.log.RunID == "" && req.runID != "" {
		run.log.RunID = req.runID
	}
}

func addRequestID(run *runState, requestID string) {
	if run == nil || requestID == "" {
		return
	}
	if _, ok := run.requestIDs[requestID]; ok {
		return
	}
	run.requestIDs[requestID] = struct{}{}
	run.log.RequestIDs = append(run.log.RequestIDs, requestID)
}

func applyRequestMetadata(run *runState, req *requestState, meta requestMeta) {
	if run == nil {
		return
	}
	if req != nil {
		if meta.Model != "" {
			req.requestModel = meta.Model
		}
		if meta.Temperature != nil {
			req.requestTemperature = meta.Temperature
		}
		if meta.MaxTokens != nil {
			req.requestMaxTokens = meta.MaxTokens
		}
	}
	if run.log.Metadata.Model == "" && meta.Model != "" {
		run.log.Metadata.Model = meta.Model
	}
	if run.log.Metadata.Temperature == nil && meta.Temperature != nil {
		run.log.Metadata.Temperature = meta.Temperature
	}
	if run.log.Metadata.MaxTokens == nil && meta.MaxTokens != nil {
		run.log.Metadata.MaxTokens = meta.MaxTokens
	}
}

func applyResponseMetadata(run *runState, req *requestState) {
	if run == nil || req == nil {
		return
	}
	if req.responseModel != "" {
		run.log.Metadata.Model = req.responseModel
	} else if run.log.Metadata.Model == "" && req.requestModel != "" {
		run.log.Metadata.Model = req.requestModel
	}
	if run.log.Metadata.Temperature == nil && req.requestTemperature != nil {
		run.log.Metadata.Temperature = req.requestTemperature
	}
	if run.log.Metadata.MaxTokens == nil && req.requestMaxTokens != nil {
		run.log.Metadata.MaxTokens = req.requestMaxTokens
	}
	if req.responseUsageSet {
		run.log.Metadata.Usage = req.responseUsage
	}
}

func appendResponseItems(run *runState, req *requestState) {
	if run == nil || req == nil {
		return
	}
	if req.assistant.length > 0 {
		msg := Message{
			Role:       "assistant",
			Content:    req.assistant.content.String(),
			Provenance: req.assistant.spans,
		}
		run.log.Items = append(run.log.Items, Item{
			Type:    ItemMessage,
			Message: &msg,
		})
	}

	if len(req.toolCallOrder) > 0 {
		for _, builder := range req.toolCallOrder {
			if builder == nil {
				continue
			}
			call := ToolCall{
				ID:         strings.TrimSpace(builder.ID),
				Name:       strings.TrimSpace(builder.Name),
				Arguments:  builder.Arguments,
				Provenance: builder.spans,
			}
			run.log.Items = append(run.log.Items, Item{
				Type:     ItemToolCall,
				ToolCall: &call,
			})
		}
	}
}

func handleSSEData(req *requestState, run *runState, entry sseData) {
	data := strings.TrimSpace(entry.Data)
	if data == "" || data == "[DONE]" {
		return
	}

	var probe struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(data), &probe); err != nil {
		return
	}
	if probe.Type != "" {
		handleResponsesEvent(req, run, data, probe.Type, entry.Span)
		return
	}

	handleChatCompletionEvent(req, run, data, entry.Span)
}

func handleResponsesEvent(req *requestState, run *runState, data string, eventType string, span streamSpan) {
	switch eventType {
	case "response.output_text.delta":
		var ev struct {
			Delta string `json:"delta"`
			Text  string `json:"text"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			return
		}
		text := ev.Delta
		if text == "" {
			text = ev.Text
		}
		if text == "" {
			return
		}
		req.outputTextSeen = true
		req.assistant.append(text, req.traceSpans(span))
	case "response.output_text.done":
		if req.outputTextSeen {
			return
		}
		var ev struct {
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			return
		}
		if ev.Text == "" {
			return
		}
		req.assistant.append(ev.Text, req.traceSpans(span))
	case "response.function_call_arguments.delta", "response.function_call_arguments.done":
		var ev struct {
			Delta       string `json:"delta"`
			Arguments   string `json:"arguments"`
			ItemID      string `json:"item_id"`
			CallID      string `json:"call_id"`
			Name        string `json:"name"`
			OutputIndex *int   `json:"output_index"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			return
		}
		id := ev.ItemID
		if id == "" {
			id = ev.CallID
		}
		builder := req.toolCallBuilder(id, ev.OutputIndex)
		if ev.Name != "" {
			builder.Name = ev.Name
		}
		args := ev.Delta
		if args == "" {
			args = ev.Arguments
		}
		if args != "" {
			builder.append(args, req.traceSpans(span))
		}
	case "response.completed", "response.failed":
		var ev struct {
			Response struct {
				Model string `json:"model"`
				Usage Usage  `json:"usage"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(data), &ev) != nil {
			return
		}
		if ev.Response.Model != "" {
			req.responseModel = ev.Response.Model
		}
		if hasUsage(ev.Response.Usage) {
			req.responseUsage = ev.Response.Usage
			req.responseUsageSet = true
		}
		if run != nil {
			applyResponseMetadata(run, req)
		}
	}
}

func handleChatCompletionEvent(req *requestState, run *runState, data string, span streamSpan) {
	var chunk struct {
		Model   string `json:"model"`
		Choices []struct {
			Delta struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					Index    int    `json:"index"`
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
		Usage *Usage `json:"usage"`
	}
	if json.Unmarshal([]byte(data), &chunk) != nil {
		return
	}
	if chunk.Model != "" {
		req.responseModel = chunk.Model
	}
	if chunk.Usage != nil && hasUsage(*chunk.Usage) {
		req.responseUsage = *chunk.Usage
		req.responseUsageSet = true
	}
	if run != nil {
		applyResponseMetadata(run, req)
	}

	if len(chunk.Choices) == 0 {
		return
	}
	choice := chunk.Choices[0]
	if choice.Delta.Content != "" {
		req.outputTextSeen = true
		req.assistant.append(choice.Delta.Content, req.traceSpans(span))
	}
	for _, call := range choice.Delta.ToolCalls {
		idx := call.Index
		builder := req.toolCallBuilder(call.ID, &idx)
		if call.Function.Name != "" {
			builder.Name = call.Function.Name
		}
		if call.Function.Arguments != "" {
			builder.append(call.Function.Arguments, req.traceSpans(span))
		}
	}
}

func (req *requestState) traceSpans(span streamSpan) []TraceSpan {
	if req == nil {
		return nil
	}
	if span.End <= span.Start {
		return nil
	}
	if len(req.responseChunks) == 0 {
		return nil
	}
	var spans []TraceSpan
	for _, chunk := range req.responseChunks {
		if span.End <= chunk.streamStart || span.Start >= chunk.streamEnd {
			continue
		}
		start := span.Start
		if start < chunk.streamStart {
			start = chunk.streamStart
		}
		end := span.End
		if end > chunk.streamEnd {
			end = chunk.streamEnd
		}
		spans = append(spans, TraceSpan{
			RequestID:  chunk.requestID,
			EventIndex: chunk.eventIndex,
			EventType:  chunk.eventType,
			Seq:        chunk.seq,
			ByteStart:  start - chunk.streamStart,
			ByteEnd:    end - chunk.streamStart,
		})
	}
	return spans
}

func (b *messageBuilder) append(text string, trace []TraceSpan) {
	if b == nil || text == "" {
		return
	}
	if b.role == "" {
		b.role = "assistant"
	}
	start := b.length
	b.content.WriteString(text)
	b.length += len(text)
	if len(trace) == 0 {
		return
	}
	b.spans = append(b.spans, ProvenanceSpan{
		ContentStart: start,
		ContentEnd:   b.length,
		Trace:        trace,
	})
}

func (req *requestState) toolCallBuilder(id string, index *int) *toolCallBuilder {
	if req.toolCalls == nil {
		req.toolCalls = make(map[string]*toolCallBuilder)
	}
	if id != "" {
		if builder := req.toolCalls["id:"+id]; builder != nil {
			return builder
		}
	}
	if index != nil {
		key := fmt.Sprintf("index:%d", *index)
		if builder := req.toolCalls[key]; builder != nil {
			if id != "" {
				req.toolCalls["id:"+id] = builder
				builder.ID = id
			}
			return builder
		}
	}

	key := ""
	if id != "" {
		key = "id:" + id
	} else if index != nil {
		key = fmt.Sprintf("index:%d", *index)
	} else {
		key = fmt.Sprintf("anon:%d", len(req.toolCallOrder))
	}

	builder := &toolCallBuilder{}
	if id != "" {
		builder.ID = id
	}
	req.toolCalls[key] = builder
	if id != "" && index != nil {
		req.toolCalls[fmt.Sprintf("index:%d", *index)] = builder
	}
	req.toolCallOrder = append(req.toolCallOrder, builder)
	return builder
}

func (b *toolCallBuilder) append(text string, trace []TraceSpan) {
	if b == nil || text == "" {
		return
	}
	start := b.length
	b.Arguments += text
	b.length += len(text)
	if len(trace) == 0 {
		return
	}
	b.spans = append(b.spans, ProvenanceSpan{
		ContentStart: start,
		ContentEnd:   b.length,
		Trace:        trace,
	})
}

type requestMeta struct {
	Model       string
	Temperature *float64
	MaxTokens   *int
}

func parseRequestBody(body []byte) ([]Message, []ToolResult, requestMeta) {
	var meta requestMeta
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, nil, meta
	}

	var raw map[string]any
	if json.Unmarshal(body, &raw) != nil {
		return nil, nil, meta
	}
	meta.Model = getString(raw, "model")
	if temp, ok := getFloat(raw, "temperature"); ok {
		meta.Temperature = &temp
	}
	if maxTokens, ok := getInt(raw, "max_tokens"); ok {
		meta.MaxTokens = &maxTokens
	} else if maxTokens, ok := getInt(raw, "max_output_tokens"); ok {
		meta.MaxTokens = &maxTokens
	}

	var messages []Message
	var results []ToolResult
	if rawMessages, ok := raw["messages"]; ok {
		msgs, res := parseMessageArray(rawMessages)
		messages = append(messages, msgs...)
		results = append(results, res...)
	}
	if rawInput, ok := raw["input"]; ok {
		msgs, res := parseInput(rawInput)
		messages = append(messages, msgs...)
		results = append(results, res...)
	}
	return messages, results, meta
}

func parseMessageArray(value any) ([]Message, []ToolResult) {
	arr, ok := value.([]any)
	if !ok {
		return nil, nil
	}
	var messages []Message
	var results []ToolResult
	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role := strings.TrimSpace(getString(obj, "role"))
		content := extractContent(obj["content"])
		if role == "tool" {
			if content == "" {
				continue
			}
			toolID := getString(obj, "tool_call_id")
			results = append(results, ToolResult{
				ID:     toolID,
				Result: content,
			})
			continue
		}
		if role == "" || content == "" {
			continue
		}
		messages = append(messages, Message{
			Role:    role,
			Content: content,
		})
	}
	return messages, results
}

func parseInput(value any) ([]Message, []ToolResult) {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return []Message{{Role: "user", Content: v}}, nil
	case []any:
		var messages []Message
		var results []ToolResult
		for _, item := range v {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			itemType := strings.TrimSpace(getString(obj, "type"))
			role := strings.TrimSpace(getString(obj, "role"))
			switch {
			case itemType == "message" || (itemType == "" && role != ""):
				content := extractContent(obj["content"])
				if content == "" {
					continue
				}
				if role == "" {
					role = "user"
				}
				messages = append(messages, Message{
					Role:    role,
					Content: content,
				})
			case isToolResultType(itemType) || role == "tool":
				result := extractContent(obj["output"])
				if result == "" {
					result = extractContent(obj["content"])
				}
				if result == "" {
					continue
				}
				toolID := getString(obj, "call_id")
				if toolID == "" {
					toolID = getString(obj, "tool_call_id")
				}
				results = append(results, ToolResult{
					ID:     toolID,
					Result: result,
				})
			}
		}
		return messages, results
	default:
		return nil, nil
	}
}

func parseNonStreamingResponse(body []byte) (string, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return "", false
	}
	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
				Role    string `json:"role"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(body, &chat) == nil {
		if len(chat.Choices) > 0 && chat.Choices[0].Message.Content != "" {
			return chat.Choices[0].Message.Content, true
		}
	}
	return "", false
}

func extractContent(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case []any:
		var sb strings.Builder
		for _, part := range v {
			sb.WriteString(extractContent(part))
		}
		return sb.String()
	case map[string]any:
		if text := getString(v, "text"); text != "" {
			return text
		}
		if text := getString(v, "content"); text != "" {
			return text
		}
		if text := getString(v, "value"); text != "" {
			return text
		}
		if inner, ok := v["content"]; ok {
			return extractContent(inner)
		}
	}
	return ""
}

func isToolResultType(kind string) bool {
	switch kind {
	case "function_call_output", "tool_call_output", "tool_result", "tool_output":
		return true
	default:
		return false
	}
}

func getString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}

func getFloat(m map[string]any, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func getInt(m map[string]any, key string) (int, bool) {
	if m == nil {
		return 0, false
	}
	value, ok := m[key]
	if !ok || value == nil {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func hasUsage(usage Usage) bool {
	return usage.InputTokens != 0 || usage.OutputTokens != 0 || usage.TotalTokens != 0
}

func ValidateLog(log Log) error {
	if log.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d", log.SchemaVersion)
	}
	return nil
}

func MergeLogs(logs []Log) (Log, error) {
	if len(logs) == 0 {
		return Log{}, errors.New("no logs provided")
	}
	sort.SliceStable(logs, func(i, j int) bool {
		return logs[i].RunID < logs[j].RunID
	})
	merged := logs[0]
	for i := 1; i < len(logs); i++ {
		merged.Items = append(merged.Items, logs[i].Items...)
	}
	return merged, nil
}
