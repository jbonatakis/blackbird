package canonical

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

func TestSSEDeltaAssembly(t *testing.T) {
	data1 := `{"choices":[{"delta":{"content":"Hello "}}]}`
	data2 := `{"choices":[{"delta":{"content":"world"}}]}`

	events := []trace.Event{
		{Type: trace.EventResponseStart, RequestID: "req-1", RunID: "run-1", Status: 200},
		{Type: trace.EventResponseBody, RequestID: "req-1", RunID: "run-1", Seq: 1, Body: []byte("data: " + data1 + "\n\n")},
		{Type: trace.EventResponseBody, RequestID: "req-1", RunID: "run-1", Seq: 2, Body: []byte("data: " + data2 + "\n\n")},
		{Type: trace.EventResponseEnd, RequestID: "req-1", RunID: "run-1"},
	}

	logs, err := Canonicalize(events)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	log := logs[0]
	if len(log.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(log.Items))
	}
	item := log.Items[0]
	if item.Message == nil {
		t.Fatalf("expected message item")
	}
	if item.Message.Content != "Hello world" {
		t.Fatalf("message content = %q, want %q", item.Message.Content, "Hello world")
	}
}

func TestProvenanceMapping(t *testing.T) {
	data1 := `{"choices":[{"delta":{"content":"Hello "}}]}`
	data2 := `{"choices":[{"delta":{"content":"world"}}]}`

	events := []trace.Event{
		{Type: trace.EventResponseStart, RequestID: "req-1", RunID: "run-1", Status: 200},
		{Type: trace.EventResponseBody, RequestID: "req-1", RunID: "run-1", Seq: 1, Body: []byte("data: " + data1 + "\n\n")},
		{Type: trace.EventResponseBody, RequestID: "req-1", RunID: "run-1", Seq: 2, Body: []byte("data: " + data2 + "\n\n")},
		{Type: trace.EventResponseEnd, RequestID: "req-1", RunID: "run-1"},
	}

	logs, err := Canonicalize(events)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	item := logs[0].Items[0]
	msg := item.Message
	if msg == nil {
		t.Fatalf("expected message item")
	}
	if len(msg.Provenance) != 2 {
		t.Fatalf("expected 2 provenance spans, got %d", len(msg.Provenance))
	}

	span1 := msg.Provenance[0]
	if span1.ContentStart != 0 || span1.ContentEnd != len("Hello ") {
		t.Fatalf("span1 content range = %d-%d, want 0-%d", span1.ContentStart, span1.ContentEnd, len("Hello "))
	}
	if len(span1.Trace) != 1 {
		t.Fatalf("span1 trace spans = %d, want 1", len(span1.Trace))
	}
	trace1 := span1.Trace[0]
	if trace1.EventIndex != 1 {
		t.Fatalf("span1 event index = %d, want 1", trace1.EventIndex)
	}
	if trace1.Seq != 1 {
		t.Fatalf("span1 seq = %d, want 1", trace1.Seq)
	}
	wantStart := len("data: ")
	wantEnd := wantStart + len(data1)
	if trace1.ByteStart != wantStart || trace1.ByteEnd != wantEnd {
		t.Fatalf("span1 byte range = %d-%d, want %d-%d", trace1.ByteStart, trace1.ByteEnd, wantStart, wantEnd)
	}

	span2 := msg.Provenance[1]
	if span2.ContentStart != len("Hello ") || span2.ContentEnd != len("Hello world") {
		t.Fatalf("span2 content range = %d-%d, want %d-%d", span2.ContentStart, span2.ContentEnd, len("Hello "), len("Hello world"))
	}
	if len(span2.Trace) != 1 {
		t.Fatalf("span2 trace spans = %d, want 1", len(span2.Trace))
	}
	trace2 := span2.Trace[0]
	if trace2.EventIndex != 2 {
		t.Fatalf("span2 event index = %d, want 2", trace2.EventIndex)
	}
	if trace2.Seq != 2 {
		t.Fatalf("span2 seq = %d, want 2", trace2.Seq)
	}
	wantStart2 := len("data: ")
	wantEnd2 := wantStart2 + len(data2)
	if trace2.ByteStart != wantStart2 || trace2.ByteEnd != wantEnd2 {
		t.Fatalf("span2 byte range = %d-%d, want %d-%d", trace2.ByteStart, trace2.ByteEnd, wantStart2, wantEnd2)
	}
}
