package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

const defaultBufferSize = 32 * 1024

var hopHeaders = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"TE":                  {},
	"Trailers":            {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

type Config struct {
	Adapter           memprovider.Adapter
	APIBaseURL        string
	ChatGPTBaseURL    string
	BaseURLPrefix     string
	TracePath         string
	TraceOptions      trace.Options
	Client            *http.Client
	Now               func() time.Time
	RequestID         func() (string, error)
	ErrorHandler      func(error)
	ResponseBufSize   int
	DisableHopHeaders bool
}

type Proxy struct {
	adapter        memprovider.Adapter
	apiBase        *url.URL
	chatBase       *url.URL
	prefix         string
	client         *http.Client
	wal            *trace.WALWriter
	now            func() time.Time
	requestID      func() (string, error)
	onError        func(error)
	responseBufLen int
	stripHop       bool
}

func New(cfg Config) (*Proxy, error) {
	if cfg.Adapter == nil {
		return nil, errors.New("adapter is required")
	}
	if strings.TrimSpace(cfg.TracePath) == "" {
		return nil, errors.New("trace path is required")
	}

	apiBase, err := parseBaseURL(cfg.APIBaseURL, true)
	if err != nil {
		return nil, fmt.Errorf("parse api upstream: %w", err)
	}
	chatBase, err := parseBaseURL(cfg.ChatGPTBaseURL, false)
	if err != nil {
		return nil, fmt.Errorf("parse chatgpt upstream: %w", err)
	}

	wal, err := trace.NewWALWriter(cfg.TracePath, cfg.TraceOptions)
	if err != nil {
		return nil, err
	}

	client := cfg.Client
	if client == nil {
		client = &http.Client{}
	}

	now := cfg.Now
	if now == nil {
		now = time.Now
	}

	requestID := cfg.RequestID
	if requestID == nil {
		requestID = newRequestID
	}

	bufSize := cfg.ResponseBufSize
	if bufSize <= 0 {
		bufSize = defaultBufferSize
	}

	prefix := strings.TrimSpace(cfg.BaseURLPrefix)
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	return &Proxy{
		adapter:        cfg.Adapter,
		apiBase:        apiBase,
		chatBase:       chatBase,
		prefix:         prefix,
		client:         client,
		wal:            wal,
		now:            now,
		requestID:      requestID,
		onError:        cfg.ErrorHandler,
		responseBufLen: bufSize,
		stripHop:       !cfg.DisableHopHeaders,
	}, nil
}

func (p *Proxy) Close() error {
	if p == nil || p.wal == nil {
		return nil
	}
	return p.wal.Close()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p == nil {
		http.Error(w, "proxy unavailable", http.StatusServiceUnavailable)
		return
	}

	start := p.now()
	requestID := p.nextRequestID()
	ids := p.adapter.RequestIDs(r.Header)

	path := r.URL.Path
	if p.prefix != "" {
		var ok bool
		path, ok = stripPrefix(path, p.prefix)
		if !ok {
			http.NotFound(w, r)
			return
		}
	}

	route := p.adapter.Route(path, r.Header)
	upstream := p.upstreamFor(route.Upstream)
	if upstream == nil {
		p.append(trace.Event{
			Type:      trace.EventError,
			RequestID: requestID,
			SessionID: ids.SessionID,
			TaskID:    ids.TaskID,
			RunID:     ids.RunID,
			Error:     fmt.Sprintf("unknown upstream for path %s", path),
			ErrorKind: "upstream",
		})
		http.Error(w, "unknown upstream", http.StatusBadGateway)
		return
	}

	requestHeaders := cloneHeader(r.Header)
	requestHeaders.Del("X-Forwarded-For")
	if p.stripHop {
		removeHopHeaders(requestHeaders)
	}

	p.append(trace.Event{
		Type:      trace.EventRequestStart,
		RequestID: requestID,
		SessionID: ids.SessionID,
		TaskID:    ids.TaskID,
		RunID:     ids.RunID,
		Method:    r.Method,
		Path:      route.Path,
		Headers:   requestHeaders,
	})

	var recorder *bodyRecorder
	var body io.Reader
	if r.Body != nil && r.Body != http.NoBody {
		recorder = newBodyRecorder(r.Body, func(seq int, data []byte) {
			p.append(trace.Event{
				Type:      trace.EventRequestBody,
				RequestID: requestID,
				SessionID: ids.SessionID,
				TaskID:    ids.TaskID,
				RunID:     ids.RunID,
				Seq:       seq,
				Body:      data,
			})
		})
		body = recorder
	}

	upstreamURL := *upstream
	upstreamURL.Path = route.Path
	upstreamURL.RawQuery = r.URL.RawQuery

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, upstreamURL.String(), body)
	if err != nil {
		p.append(trace.Event{
			Type:      trace.EventError,
			RequestID: requestID,
			SessionID: ids.SessionID,
			TaskID:    ids.TaskID,
			RunID:     ids.RunID,
			Error:     err.Error(),
			ErrorKind: "build_request",
		})
		http.Error(w, "failed to build request", http.StatusBadGateway)
		return
	}

	outReq.Header = requestHeaders
	outReq.Host = upstreamURL.Host
	outReq.ContentLength = r.ContentLength
	outReq.TransferEncoding = append([]string(nil), r.TransferEncoding...)

	resp, err := p.client.Do(outReq)
	requestBytes := int64(0)
	if recorder != nil {
		requestBytes = recorder.Bytes()
	}
	p.append(trace.Event{
		Type:       trace.EventRequestEnd,
		RequestID:  requestID,
		SessionID:  ids.SessionID,
		TaskID:     ids.TaskID,
		RunID:      ids.RunID,
		BodyBytes:  requestBytes,
		DurationMs: p.durationMs(start),
	})

	if err != nil {
		p.append(trace.Event{
			Type:      trace.EventError,
			RequestID: requestID,
			SessionID: ids.SessionID,
			TaskID:    ids.TaskID,
			RunID:     ids.RunID,
			Error:     err.Error(),
			ErrorKind: "upstream_request",
		})
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseHeaders := cloneHeader(resp.Header)
	if p.stripHop {
		removeHopHeaders(responseHeaders)
	}
	p.append(trace.Event{
		Type:      trace.EventResponseStart,
		RequestID: requestID,
		SessionID: ids.SessionID,
		TaskID:    ids.TaskID,
		RunID:     ids.RunID,
		Status:    resp.StatusCode,
		Headers:   responseHeaders,
	})

	copyHeader(w.Header(), responseHeaders)
	w.WriteHeader(resp.StatusCode)

	flusher, _ := w.(http.Flusher)
	buf := make([]byte, p.responseBufLen)
	var responseBytes int64
	respSeq := 0
	var copyErr error

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			respSeq++
			p.append(trace.Event{
				Type:      trace.EventResponseBody,
				RequestID: requestID,
				SessionID: ids.SessionID,
				TaskID:    ids.TaskID,
				RunID:     ids.RunID,
				Seq:       respSeq,
				Body:      chunk,
			})
			if _, err := w.Write(chunk); err != nil {
				copyErr = err
				break
			}
			responseBytes += int64(n)
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				copyErr = readErr
			}
			break
		}
	}

	if copyErr != nil {
		p.append(trace.Event{
			Type:      trace.EventError,
			RequestID: requestID,
			SessionID: ids.SessionID,
			TaskID:    ids.TaskID,
			RunID:     ids.RunID,
			Error:     copyErr.Error(),
			ErrorKind: "response_copy",
		})
	}

	p.append(trace.Event{
		Type:       trace.EventResponseEnd,
		RequestID:  requestID,
		SessionID:  ids.SessionID,
		TaskID:     ids.TaskID,
		RunID:      ids.RunID,
		Status:     resp.StatusCode,
		BodyBytes:  responseBytes,
		DurationMs: p.durationMs(start),
	})
}

func (p *Proxy) append(ev trace.Event) {
	if p == nil || p.wal == nil {
		return
	}
	if err := p.wal.Append(ev); err != nil && p.onError != nil {
		p.onError(err)
	}
}

func (p *Proxy) upstreamFor(upstream memprovider.Upstream) *url.URL {
	switch upstream {
	case memprovider.UpstreamAPI:
		return p.apiBase
	case memprovider.UpstreamChatGPT:
		return p.chatBase
	default:
		return nil
	}
}

func (p *Proxy) nextRequestID() string {
	id, err := p.requestID()
	if err == nil && strings.TrimSpace(id) != "" {
		return id
	}
	return fmt.Sprintf("req-%d", p.now().UnixNano())
}

func (p *Proxy) durationMs(start time.Time) int64 {
	d := p.now().Sub(start)
	if d < 0 {
		return 0
	}
	return d.Milliseconds()
}

func parseBaseURL(raw string, required bool) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if required {
			return nil, errors.New("base url is required")
		}
		return nil, nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid base url %q", trimmed)
	}
	return parsed, nil
}

func stripPrefix(path string, prefix string) (string, bool) {
	if prefix == "" {
		return path, true
	}
	if !strings.HasPrefix(path, prefix) {
		return path, false
	}
	trimmed := strings.TrimPrefix(path, prefix)
	if trimmed == "" {
		return "/", true
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed, true
}

func cloneHeader(header http.Header) http.Header {
	clone := make(http.Header, len(header))
	for key, values := range header {
		if values == nil {
			clone[key] = nil
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		clone[key] = copied
	}
	return clone
}

func removeHopHeaders(header http.Header) {
	if header == nil {
		return
	}
	connection := header.Get("Connection")
	for _, hop := range strings.Split(connection, ",") {
		if hop == "" {
			continue
		}
		if name := strings.TrimSpace(hop); name != "" {
			header.Del(name)
		}
	}
	for name := range hopHeaders {
		header.Del(name)
	}
}

func copyHeader(dst http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func newRequestID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
