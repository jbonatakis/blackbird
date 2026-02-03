package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"github.com/jbonatakis/blackbird/internal/memory/canonical"
)

type messageRef struct {
	index int
	item  canonical.Item
}

type lineSpan struct {
	text  string
	start int
	end   int
}

// ExtractArtifacts derives artifacts from canonical logs without deduplication.
func ExtractArtifacts(logs []canonical.Log) []Artifact {
	sorted := append([]canonical.Log(nil), logs...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RunID < sorted[j].RunID
	})

	var artifacts []Artifact
	for _, log := range sorted {
		artifacts = append(artifacts, extractLogArtifacts(log)...)
	}
	return artifacts
}

// BuildArtifacts extracts artifacts from logs and merges them with existing artifacts.
func BuildArtifacts(existing []Artifact, logs []canonical.Log) []Artifact {
	incoming := ExtractArtifacts(logs)
	return MergeArtifacts(existing, incoming)
}

func extractLogArtifacts(log canonical.Log) []Artifact {
	var artifacts []Artifact
	artifacts = append(artifacts, extractTranscriptArtifacts(log)...)
	artifacts = append(artifacts, extractDecisionArtifacts(log)...)
	artifacts = append(artifacts, extractConstraintArtifacts(log)...)
	artifacts = append(artifacts, extractOpenThreadArtifacts(log)...)
	if outcome := extractOutcomeArtifact(log); outcome != nil {
		artifacts = append(artifacts, *outcome)
	}
	return artifacts
}

func extractTranscriptArtifacts(log canonical.Log) []Artifact {
	var artifacts []Artifact
	for _, ref := range messageRefs(log) {
		msg := ref.item.Message
		if msg == nil || strings.TrimSpace(msg.Content) == "" {
			continue
		}
		content := msg.Content
		artifact := Artifact{
			SchemaVersion:  SchemaVersion,
			ArtifactID:     transcriptID(log, ref.index, msg.Role, content),
			SessionID:      log.SessionID,
			TaskID:         log.TaskID,
			RunID:          log.RunID,
			Type:           ArtifactTranscript,
			Content:        Content{Text: content, Role: msg.Role},
			Provenance:     []Provenance{messageProvenance(ref, 0, len(content))},
			BuilderVersion: BuilderVersion,
		}
		artifacts = append(artifacts, artifact)
	}
	return artifacts
}

func extractDecisionArtifacts(log canonical.Log) []Artifact {
	var artifacts []Artifact
	for _, ref := range messageRefs(log) {
		msg := ref.item.Message
		if msg == nil || msg.Role != "assistant" {
			continue
		}
		for _, line := range splitLinesWithSpan(msg.Content) {
			statement, rationale, start, end, ok := parseDecision(line.text)
			if !ok {
				continue
			}
			artifact := Artifact{
				SchemaVersion:  SchemaVersion,
				ArtifactID:     decisionID(log, statement),
				SessionID:      log.SessionID,
				TaskID:         log.TaskID,
				RunID:          log.RunID,
				Type:           ArtifactDecision,
				Content:        Content{Text: statement, Rationale: rationale},
				Provenance:     []Provenance{messageProvenance(ref, line.start+start, line.start+end)},
				BuilderVersion: BuilderVersion,
			}
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

func extractConstraintArtifacts(log canonical.Log) []Artifact {
	var artifacts []Artifact
	for _, ref := range messageRefs(log) {
		msg := ref.item.Message
		if msg == nil || msg.Role != "assistant" {
			continue
		}
		for _, line := range splitLinesWithSpan(msg.Content) {
			statement, rationale, start, end, ok := parseConstraint(line.text)
			if !ok {
				continue
			}
			artifact := Artifact{
				SchemaVersion:  SchemaVersion,
				ArtifactID:     constraintID(log, statement),
				SessionID:      log.SessionID,
				TaskID:         log.TaskID,
				RunID:          log.RunID,
				Type:           ArtifactConstraint,
				Content:        Content{Text: statement, Rationale: rationale, Scope: "task"},
				Provenance:     []Provenance{messageProvenance(ref, line.start+start, line.start+end)},
				BuilderVersion: BuilderVersion,
			}
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

func extractOpenThreadArtifacts(log canonical.Log) []Artifact {
	var artifacts []Artifact
	for _, ref := range messageRefs(log) {
		msg := ref.item.Message
		if msg == nil || msg.Role != "assistant" {
			continue
		}
		for _, line := range splitLinesWithSpan(msg.Content) {
			statement, start, end, ok := parseOpenThread(line.text)
			if !ok {
				continue
			}
			artifact := Artifact{
				SchemaVersion:  SchemaVersion,
				ArtifactID:     openThreadID(log, statement),
				SessionID:      log.SessionID,
				TaskID:         log.TaskID,
				RunID:          log.RunID,
				Type:           ArtifactOpenThread,
				Content:        Content{Text: statement},
				Provenance:     []Provenance{messageProvenance(ref, line.start+start, line.start+end)},
				BuilderVersion: BuilderVersion,
			}
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

func extractOutcomeArtifact(log canonical.Log) *Artifact {
	status := detectOutcomeStatus(log)
	summary := summarizeOutcome(log)
	files := extractFiles(log)
	commands, commandSources := extractCommands(log)
	errors, errorSources := extractErrors(log)

	if status == "" && len(summary) == 0 && len(files) == 0 && len(commands) == 0 && len(errors) == 0 {
		return nil
	}

	provenance := []Provenance{}
	if msgRef, ok := lastAssistantMessage(log); ok {
		provenance = append(provenance, messageProvenance(msgRef, 0, len(msgRef.item.Message.Content)))
	}
	provenance = append(provenance, commandSources...)
	provenance = append(provenance, errorSources...)

	artifact := Artifact{
		SchemaVersion: SchemaVersion,
		ArtifactID:    outcomeID(log),
		SessionID:     log.SessionID,
		TaskID:        log.TaskID,
		RunID:         log.RunID,
		Type:          ArtifactOutcome,
		Content: Content{
			Status:   status,
			Summary:  summary,
			Files:    files,
			Commands: commands,
			Errors:   errors,
		},
		Provenance:     provenance,
		BuilderVersion: BuilderVersion,
	}
	return &artifact
}

func messageRefs(log canonical.Log) []messageRef {
	refs := make([]messageRef, 0, len(log.Items))
	for idx, item := range log.Items {
		if item.Message == nil {
			continue
		}
		refs = append(refs, messageRef{index: idx, item: item})
	}
	return refs
}

func lastAssistantMessage(log canonical.Log) (messageRef, bool) {
	for idx := len(log.Items) - 1; idx >= 0; idx-- {
		item := log.Items[idx]
		if item.Message != nil && item.Message.Role == "assistant" {
			return messageRef{index: idx, item: item}, true
		}
	}
	return messageRef{}, false
}

func splitLinesWithSpan(content string) []lineSpan {
	var lines []lineSpan
	start := 0
	for i, r := range content {
		if r != '\n' {
			continue
		}
		lines = append(lines, lineSpan{text: trimLineEnding(content[start:i]), start: start, end: i})
		start = i + 1
	}
	if start <= len(content) {
		lines = append(lines, lineSpan{text: trimLineEnding(content[start:]), start: start, end: len(content)})
	}
	return lines
}

func trimLineEnding(line string) string {
	return strings.TrimSuffix(line, "\r")
}

func parseDecision(line string) (string, string, int, int, bool) {
	trimmed, offset := stripBullet(line)
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "decision:"):
		return extractAfterPrefix(trimmed, offset, len("decision:"))
	case strings.HasPrefix(lower, "decision -"):
		return extractAfterPrefix(trimmed, offset, len("decision -"))
	case strings.HasPrefix(lower, "decided:"):
		return extractAfterPrefix(trimmed, offset, len("decided:"))
	case strings.HasPrefix(lower, "we decided to"):
		return extractAfterPrefix(trimmed, offset, len("we decided to"))
	case strings.HasPrefix(lower, "we decided"):
		return extractAfterPrefix(trimmed, offset, len("we decided"))
	default:
		return "", "", 0, 0, false
	}
}

func parseConstraint(line string) (string, string, int, int, bool) {
	trimmed, offset := stripBullet(line)
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "constraint:"):
		return extractAfterPrefix(trimmed, offset, len("constraint:"))
	case strings.HasPrefix(lower, "must not"):
		return extractEntire(trimmed, offset)
	case strings.HasPrefix(lower, "must "):
		return extractEntire(trimmed, offset)
	case strings.HasPrefix(lower, "do not"):
		return extractEntire(trimmed, offset)
	case strings.HasPrefix(lower, "don't "):
		return extractEntire(trimmed, offset)
	default:
		return "", "", 0, 0, false
	}
}

func parseOpenThread(line string) (string, int, int, bool) {
	trimmed, offset := stripBullet(line)
	lower := strings.ToLower(trimmed)

	switch {
	case strings.HasPrefix(lower, "[ ]"):
		text, _, start, end, ok := extractAfterPrefix(trimmed, offset, len("[ ]"))
		return text, start, end, ok
	case strings.HasPrefix(lower, "todo"):
		return extractAfterTodo(trimmed, offset)
	case strings.HasPrefix(lower, "open question"):
		text, _, start, end, ok := extractAfterPrefix(trimmed, offset, len("open question"))
		return text, start, end, ok
	case strings.HasPrefix(lower, "need to "):
		text, _, start, end, ok := extractEntire(trimmed, offset)
		return text, start, end, ok
	case strings.HasPrefix(lower, "needs to "):
		text, _, start, end, ok := extractEntire(trimmed, offset)
		return text, start, end, ok
	case strings.HasPrefix(lower, "follow up"):
		text, _, start, end, ok := extractEntire(trimmed, offset)
		return text, start, end, ok
	case strings.HasPrefix(lower, "blocked by"):
		text, _, start, end, ok := extractEntire(trimmed, offset)
		return text, start, end, ok
	default:
		return "", 0, 0, false
	}
}

func extractAfterTodo(trimmed string, offset int) (string, int, int, bool) {
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "todo:") {
		text, _, start, end, ok := extractAfterPrefix(trimmed, offset, len("todo:"))
		return text, start, end, ok
	}
	if strings.HasPrefix(lower, "todo") {
		text, _, start, end, ok := extractAfterPrefix(trimmed, offset, len("todo"))
		return text, start, end, ok
	}
	return "", 0, 0, false
}

func extractAfterPrefix(trimmed string, offset int, prefixLen int) (string, string, int, int, bool) {
	if len(trimmed) < prefixLen {
		return "", "", 0, 0, false
	}
	rest := trimmed[prefixLen:]
	rest, restOffset := trimLeftSpaceWithOffset(rest)
	if rest == "" {
		return "", "", 0, 0, false
	}
	statement, rationale := splitRationale(rest)
	start := offset + prefixLen + restOffset
	end := start + len(statement)
	return statement, rationale, start, end, true
}

func extractEntire(trimmed string, offset int) (string, string, int, int, bool) {
	statement, rationale := splitRationale(trimmed)
	if statement == "" {
		return "", "", 0, 0, false
	}
	start := offset
	end := start + len(statement)
	return statement, rationale, start, end, true
}

func splitRationale(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	lower := strings.ToLower(text)
	markers := []string{" because ", " since ", " due to ", " so that "}
	for _, marker := range markers {
		if idx := strings.Index(lower, marker); idx > 0 {
			statement := strings.TrimSpace(text[:idx])
			rationale := strings.TrimSpace(text[idx+len(marker):])
			return statement, rationale
		}
	}
	return text, ""
}

func trimLeftSpaceWithOffset(s string) (string, int) {
	trimmed := strings.TrimLeftFunc(s, unicode.IsSpace)
	return trimmed, len(s) - len(trimmed)
}

func stripBullet(line string) (string, int) {
	trimmed, offset := trimLeftSpaceWithOffset(line)
	checkboxes := []string{"- [ ]", "- [x]", "- [X]", "* [ ]", "* [x]", "* [X]"}
	for _, prefix := range checkboxes {
		if strings.HasPrefix(trimmed, prefix) {
			trimmed = trimmed[len(prefix):]
			offset += len(prefix)
			trimmed, extra := trimLeftSpaceWithOffset(trimmed)
			offset += extra
			return trimmed, offset
		}
	}
	simple := []string{"- ", "* "}
	for _, prefix := range simple {
		if strings.HasPrefix(trimmed, prefix) {
			trimmed = trimmed[len(prefix):]
			offset += len(prefix)
			trimmed, extra := trimLeftSpaceWithOffset(trimmed)
			offset += extra
			return trimmed, offset
		}
	}
	if len(trimmed) > 2 {
		i := 0
		for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
			i++
		}
		if i > 0 && i+1 < len(trimmed) {
			if trimmed[i] == '.' || trimmed[i] == ')' {
				if trimmed[i+1] == ' ' || trimmed[i+1] == '\t' {
					trimmed = trimmed[i+2:]
					offset += i + 2
					trimmed, extra := trimLeftSpaceWithOffset(trimmed)
					offset += extra
					return trimmed, offset
				}
			}
		}
	}
	return trimmed, offset
}

func messageProvenance(ref messageRef, start int, end int) Provenance {
	spans := convertProvenanceSpans(nil)
	if ref.item.Message != nil {
		spans = convertProvenanceSpans(ref.item.Message.Provenance)
	}
	role := ""
	if ref.item.Message != nil {
		role = ref.item.Message.Role
	}
	return Provenance{
		ItemIndex:    ref.index,
		ItemType:     string(ref.item.Type),
		Role:         role,
		ContentStart: start,
		ContentEnd:   end,
		Spans:        spans,
	}
}

func convertProvenanceSpans(spans []canonical.ProvenanceSpan) []ProvenanceSpan {
	if len(spans) == 0 {
		return nil
	}
	out := make([]ProvenanceSpan, 0, len(spans))
	for _, span := range spans {
		out = append(out, ProvenanceSpan{
			ContentStart: span.ContentStart,
			ContentEnd:   span.ContentEnd,
			Trace:        convertTraceSpans(span.Trace),
		})
	}
	return out
}

func convertTraceSpans(spans []canonical.TraceSpan) []TraceSpan {
	if len(spans) == 0 {
		return nil
	}
	out := make([]TraceSpan, 0, len(spans))
	for _, span := range spans {
		out = append(out, TraceSpan{
			RequestID:  span.RequestID,
			EventIndex: span.EventIndex,
			EventType:  span.EventType,
			Seq:        span.Seq,
			ByteStart:  span.ByteStart,
			ByteEnd:    span.ByteEnd,
		})
	}
	return out
}

func transcriptID(log canonical.Log, index int, role string, content string) string {
	hasher := sha256.New()
	hasher.Write([]byte("transcript"))
	hasher.Write([]byte{0})
	hasher.Write([]byte(log.SessionID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(log.TaskID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(log.RunID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(role))
	hasher.Write([]byte{0})
	payload, _ := json.Marshal(struct {
		Index   int    `json:"index"`
		Content string `json:"content"`
	}{Index: index, Content: content})
	hasher.Write(payload)
	return "transcript_" + hex.EncodeToString(hasher.Sum(nil))
}

func decisionID(log canonical.Log, statement string) string {
	return hashedStatementID("decision", log, statement)
}

func constraintID(log canonical.Log, statement string) string {
	return hashedStatementID("constraint", log, statement)
}

func openThreadID(log canonical.Log, statement string) string {
	return hashedStatementID("open_thread", log, statement)
}

func outcomeID(log canonical.Log) string {
	return hashedStatementID("outcome", log, "")
}

func hashedStatementID(prefix string, log canonical.Log, statement string) string {
	hasher := sha256.New()
	hasher.Write([]byte(prefix))
	hasher.Write([]byte{0})
	hasher.Write([]byte(log.SessionID))
	hasher.Write([]byte{0})
	hasher.Write([]byte(log.TaskID))
	hasher.Write([]byte{0})
	if log.SessionID == "" && log.TaskID == "" {
		hasher.Write([]byte(log.RunID))
		hasher.Write([]byte{0})
	}
	if statement != "" {
		hasher.Write([]byte(normalizeStatement(statement)))
	}
	return prefix + "_" + hex.EncodeToString(hasher.Sum(nil))
}

func normalizeStatement(statement string) string {
	statement = strings.TrimSpace(statement)
	statement = strings.Trim(statement, ".;:")
	statement = strings.ToLower(statement)
	return strings.Join(strings.Fields(statement), " ")
}
