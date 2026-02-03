package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

type routeCaptureRoundTripper struct {
	apiHost  string
	chatHost string
	apiCh    chan string
	chatCh   chan string
}

func (rt *routeCaptureRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	discardRequestBody(req)
	if req.URL.Host == rt.apiHost {
		rt.apiCh <- req.URL.Path
	} else if req.URL.Host == rt.chatHost {
		rt.chatCh <- req.URL.Path
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}, nil
}

type streamingRoundTripper struct {
	firstChunk  []byte
	secondChunk []byte
	allowSecond chan struct{}
}

func (rt *streamingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	discardRequestBody(req)
	reader, writer := io.Pipe()
	go func() {
		_, _ = writer.Write(rt.firstChunk)
		<-rt.allowSecond
		_, _ = writer.Write(rt.secondChunk)
		_ = writer.Close()
	}()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       reader,
		Request:    req,
	}, nil
}

type staticRoundTripper struct {
	header http.Header
	status int
	body   string
}

func (rt staticRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	discardRequestBody(req)
	return &http.Response{
		StatusCode: rt.status,
		Header:     cloneHeader(rt.header),
		Body:       io.NopCloser(strings.NewReader(rt.body)),
		Request:    req,
	}, nil
}

type streamingRecorder struct {
	header http.Header
	status int
	writes chan []byte
}

func newStreamingRecorder() *streamingRecorder {
	return &streamingRecorder{
		header: make(http.Header),
		writes: make(chan []byte, 4),
	}
}

func (r *streamingRecorder) Header() http.Header {
	return r.header
}

func (r *streamingRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *streamingRecorder) Write(p []byte) (int, error) {
	buf := make([]byte, len(p))
	copy(buf, p)
	r.writes <- buf
	return len(p), nil
}

func (r *streamingRecorder) Flush() {}

func discardRequestBody(req *http.Request) {
	if req.Body == nil || req.Body == http.NoBody {
		return
	}
	_, _ = io.Copy(io.Discard, req.Body)
	_ = req.Body.Close()
}

func TestProxyRoutesCodex(t *testing.T) {
	apiPathCh := make(chan string, 1)
	chatPathCh := make(chan string, 1)
	rt := &routeCaptureRoundTripper{
		apiHost:  "api.test",
		chatHost: "chat.test",
		apiCh:    apiPathCh,
		chatCh:   chatPathCh,
	}

	walPath := filepath.Join(t.TempDir(), "trace.wal")
	proxy, err := New(Config{
		Adapter:        memprovider.CodexAdapter{},
		APIBaseURL:     "http://api.test",
		ChatGPTBaseURL: "http://chat.test",
		TracePath:      walPath,
		TraceOptions: trace.Options{
			FsyncOnWrite:    false,
			FsyncOnWriteSet: true,
			MaxAge:          -1,
		},
		Client: &http.Client{Transport: rt},
	})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "http://proxy.test/responses", strings.NewReader("{}"))
	resp := httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)

	select {
	case path := <-apiPathCh:
		if path != "/v1/responses" {
			t.Fatalf("api path = %q, want %q", path, "/v1/responses")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for api upstream")
	}

	req = httptest.NewRequest(http.MethodPost, "http://proxy.test/responses", strings.NewReader("{}"))
	req.Header.Set("Chatgpt-Account-Id", "acct")
	resp = httptest.NewRecorder()
	proxy.ServeHTTP(resp, req)

	select {
	case path := <-chatPathCh:
		if path != "/backend-api/codex/responses" {
			t.Fatalf("chatgpt path = %q, want %q", path, "/backend-api/codex/responses")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for chatgpt upstream")
	}
}

func TestProxyStreamsResponse(t *testing.T) {
	firstChunk := []byte("chunk1\n")
	secondChunk := []byte("chunk2\n")
	allowSecond := make(chan struct{})

	rt := &streamingRoundTripper{
		firstChunk:  firstChunk,
		secondChunk: secondChunk,
		allowSecond: allowSecond,
	}

	walPath := filepath.Join(t.TempDir(), "trace.wal")
	proxy, err := New(Config{
		Adapter:    memprovider.CodexAdapter{},
		APIBaseURL: "http://api.test",
		TracePath:  walPath,
		TraceOptions: trace.Options{
			FsyncOnWrite:    false,
			FsyncOnWriteSet: true,
			MaxAge:          -1,
		},
		Client: &http.Client{Transport: rt},
	})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}
	defer proxy.Close()

	recorder := newStreamingRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://proxy.test/responses", nil)

	done := make(chan struct{})
	go func() {
		proxy.ServeHTTP(recorder, req)
		close(done)
	}()

	select {
	case got := <-recorder.writes:
		if string(got) != string(firstChunk) {
			close(allowSecond)
			t.Fatalf("first chunk = %q, want %q", string(got), string(firstChunk))
		}
	case <-time.After(2 * time.Second):
		close(allowSecond)
		t.Fatal("first chunk was not streamed")
	}

	close(allowSecond)
	select {
	case got := <-recorder.writes:
		if string(got) != string(secondChunk) {
			t.Fatalf("second chunk = %q, want %q", string(got), string(secondChunk))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second chunk not received")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("proxy did not finish")
	}
}

func TestProxyCapturesIDsAndRedactsHeaders(t *testing.T) {
	rt := staticRoundTripper{
		header: http.Header{"Set-Cookie": []string{"session=secret"}},
		status: http.StatusOK,
		body:   "ok",
	}

	walPath := filepath.Join(t.TempDir(), "trace.wal")
	proxy, err := New(Config{
		Adapter:    memprovider.CodexAdapter{},
		APIBaseURL: "http://api.test",
		TracePath:  walPath,
		TraceOptions: trace.Options{
			FsyncOnWrite:    false,
			FsyncOnWriteSet: true,
			MaxAge:          -1,
		},
		Client: &http.Client{Transport: rt},
	})
	if err != nil {
		t.Fatalf("new proxy: %v", err)
	}
	defer proxy.Close()

	req := httptest.NewRequest(http.MethodPost, "http://proxy.test/responses", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(memprovider.HeaderBlackbirdSessionID, "session-123")
	req.Header.Set(memprovider.HeaderBlackbirdTaskID, "task-456")
	req.Header.Set(memprovider.HeaderBlackbirdRunID, "run-789")

	recorder := httptest.NewRecorder()
	proxy.ServeHTTP(recorder, req)

	if err := proxy.Close(); err != nil {
		t.Fatalf("close proxy: %v", err)
	}

	events, err := trace.Replay(walPath)
	if err != nil {
		t.Fatalf("replay wal: %v", err)
	}

	var reqStart *trace.Event
	var respStart *trace.Event
	for i := range events {
		ev := &events[i]
		switch ev.Type {
		case trace.EventRequestStart:
			reqStart = ev
		case trace.EventResponseStart:
			respStart = ev
		}
	}
	if reqStart == nil {
		t.Fatal("missing request start event")
	}
	if respStart == nil {
		t.Fatal("missing response start event")
	}
	if reqStart.RequestID == "" {
		t.Fatal("request_id not set")
	}
	if respStart.RequestID != reqStart.RequestID {
		t.Fatalf("response request_id = %q, want %q", respStart.RequestID, reqStart.RequestID)
	}

	if values := reqStart.Headers["Authorization"]; len(values) == 0 {
		t.Fatal("missing authorization header")
	} else if values[0] != trace.DefaultRedactionReplacement {
		t.Fatalf("authorization header = %q, want %q", values[0], trace.DefaultRedactionReplacement)
	}
	if values := respStart.Headers["Set-Cookie"]; len(values) == 0 {
		t.Fatal("missing set-cookie header")
	} else if values[0] != trace.DefaultRedactionReplacement {
		t.Fatalf("set-cookie header = %q, want %q", values[0], trace.DefaultRedactionReplacement)
	}

	for _, ev := range events {
		if ev.RequestID != reqStart.RequestID {
			continue
		}
		if ev.SessionID != "session-123" {
			t.Fatalf("session id = %q, want %q", ev.SessionID, "session-123")
		}
		if ev.TaskID != "task-456" {
			t.Fatalf("task id = %q, want %q", ev.TaskID, "task-456")
		}
		if ev.RunID != "run-789" {
			t.Fatalf("run id = %q, want %q", ev.RunID, "run-789")
		}
	}
}

func TestStripPrefix(t *testing.T) {
	path, ok := stripPrefix("/base/v1/responses", "/base")
	if !ok {
		t.Fatal("expected prefix to match")
	}
	if path != "/v1/responses" {
		t.Fatalf("path = %q, want %q", path, "/v1/responses")
	}

	if _, ok := stripPrefix("/other", "/base"); ok {
		t.Fatal("expected prefix mismatch")
	}
}

func TestBodyRecorderCopiesData(t *testing.T) {
	input := "hello"
	records := 0
	recorder := newBodyRecorder(io.NopCloser(strings.NewReader(input)), func(seq int, data []byte) {
		records++
		if seq != 1 {
			t.Fatalf("seq = %d, want 1", seq)
		}
		if string(data) != input {
			t.Fatalf("data = %q, want %q", string(data), input)
		}
	})

	if _, err := io.ReadAll(recorder); err != nil {
		t.Fatalf("read: %v", err)
	}
	if records != 1 {
		t.Fatalf("records = %d, want 1", records)
	}
	if recorder.Bytes() != int64(len(input)) {
		t.Fatalf("bytes = %d, want %d", recorder.Bytes(), len(input))
	}
	if err := recorder.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestParseBaseURL(t *testing.T) {
	if _, err := parseBaseURL("", true); err == nil {
		t.Fatal("expected error for empty required base url")
	}
	if _, err := parseBaseURL("http://example.com", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := parseBaseURL("example.com", true); err == nil {
		t.Fatal("expected error for missing scheme")
	}
	if _, err := parseBaseURL("", false); err != nil {
		t.Fatalf("unexpected error for optional empty: %v", err)
	}
}

func TestNextRequestIDFallback(t *testing.T) {
	proxy := &Proxy{
		requestID: func() (string, error) { return "", errors.New("no") },
		now:       func() time.Time { return time.Unix(1, 2) },
	}
	id := proxy.nextRequestID()
	if !strings.HasPrefix(id, "req-") {
		t.Fatalf("fallback request id = %q", id)
	}
}
