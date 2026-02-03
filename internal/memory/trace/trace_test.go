package trace

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func TestReplayOrderAcrossRotations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.wal")

	clock := &testClock{now: time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)}
	writer, err := NewWALWriter(path, Options{
		MaxSizeBytes:    200,
		MaxAge:          10 * time.Second,
		FsyncOnWrite:    false,
		FsyncOnWriteSet: true,
		Now:             clock.Now,
	})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}

	body := bytes.Repeat([]byte("a"), 120)
	events := []Event{
		{Type: EventRequestBody, RequestID: "req-1", Seq: 1, Body: body},
		{Type: EventRequestBody, RequestID: "req-1", Seq: 2, Body: body},
	}
	for _, ev := range events {
		if err := writer.Append(ev); err != nil {
			_ = writer.Close()
			t.Fatalf("append: %v", err)
		}
	}

	clock.Advance(20 * time.Second)
	if err := writer.Append(Event{Type: EventRequestBody, RequestID: "req-1", Seq: 3, Body: []byte("b")}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := Replay(path)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}
	for i, ev := range got {
		wantSeq := i + 1
		if ev.Seq != wantSeq {
			t.Fatalf("event %d seq=%d, want %d", i, ev.Seq, wantSeq)
		}
	}
}

func TestRedactionAndPrivacyMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.wal")

	writer, err := NewWALWriter(path, Options{
		PrivacyMode:     true,
		FsyncOnWrite:    false,
		FsyncOnWriteSet: true,
		MaxAge:          -1,
	})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}

	reqHeaders := map[string][]string{
		"Authorization": {"Bearer secret"},
		"Content-Type":  {"application/json"},
	}
	respHeaders := map[string][]string{
		"Set-Cookie":   {"session=secret"},
		"Content-Type": {"application/json"},
	}

	if err := writer.Append(Event{Type: EventRequestStart, RequestID: "req-1", Method: "POST", Path: "/v1/responses", Headers: reqHeaders}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}
	if err := writer.Append(Event{Type: EventRequestBody, RequestID: "req-1", Seq: 1, Body: []byte("top secret")}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}
	if err := writer.Append(Event{Type: EventResponseStart, RequestID: "req-1", Status: 200, Headers: respHeaders}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}
	if err := writer.Append(Event{Type: EventResponseBody, RequestID: "req-1", Seq: 1, Body: []byte("secret")}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	got, err := Replay(path)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}

	for _, ev := range got {
		switch ev.Type {
		case EventRequestStart:
			if ev.Headers["Authorization"][0] != DefaultRedactionReplacement {
				t.Fatalf("authorization header not redacted")
			}
			if ev.Headers["Content-Type"][0] != "application/json" {
				t.Fatalf("content-type header changed")
			}
		case EventResponseStart:
			if ev.Headers["Set-Cookie"][0] != DefaultRedactionReplacement {
				t.Fatalf("set-cookie header not redacted")
			}
			if ev.Headers["Content-Type"][0] != "application/json" {
				t.Fatalf("content-type header changed")
			}
		default:
			t.Fatalf("unexpected event type %s", ev.Type)
		}
	}
}

func TestRetentionPruning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.wal")
	baseName := "session"

	start := time.Date(2026, 2, 3, 9, 0, 0, 0, time.UTC)
	clock := &testClock{now: start}

	oldPath := filepath.Join(dir, rotationName(baseName, start.Add(-2*time.Hour)))
	recentPath := filepath.Join(dir, rotationName(baseName, start.Add(-30*time.Minute)))

	if err := writeEventFile(oldPath, Event{Type: EventRequestStart, RequestID: "old", Timestamp: start.Add(-2 * time.Hour)}); err != nil {
		t.Fatalf("write old wal: %v", err)
	}
	if err := writeEventFile(recentPath, Event{Type: EventRequestStart, RequestID: "recent", Timestamp: start.Add(-30 * time.Minute)}); err != nil {
		t.Fatalf("write recent wal: %v", err)
	}

	writer, err := NewWALWriter(path, Options{
		MaxSizeBytes:    200,
		MaxAge:          10 * time.Second,
		Retention:       time.Hour,
		FsyncOnWrite:    false,
		FsyncOnWriteSet: true,
		Now:             clock.Now,
	})
	if err != nil {
		t.Fatalf("open writer: %v", err)
	}

	body := bytes.Repeat([]byte("a"), 120)
	if err := writer.Append(Event{Type: EventRequestBody, RequestID: "req-1", Seq: 1, Body: body}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}
	if err := writer.Append(Event{Type: EventRequestBody, RequestID: "req-1", Seq: 2, Body: body}); err != nil {
		_ = writer.Close()
		t.Fatalf("append: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(oldPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected old wal to be pruned, err=%v", err)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Fatalf("expected recent wal to remain, err=%v", err)
	}
}

func writeEventFile(path string, event Event) error {
	event.SchemaVersion = SchemaVersion
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Unix(0, 0).UTC()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	return os.WriteFile(path, payload, 0o644)
}
