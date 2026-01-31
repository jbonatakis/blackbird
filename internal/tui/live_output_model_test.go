package tui

import "testing"

func TestLiveOutputMsgAppendsAndContinuesListening(t *testing.T) {
	streamCh := make(chan liveOutputMsg, 1)
	streamCh <- liveOutputMsg{stream: "stdout", data: "next"}

	model := Model{liveOutputChan: streamCh}
	updated, cmd := model.Update(liveOutputMsg{stream: "stdout", data: "hello"})
	next := updated.(Model)

	if next.liveStdout != "hello" {
		t.Fatalf("expected liveStdout to append data, got %q", next.liveStdout)
	}
	if cmd == nil {
		t.Fatalf("expected listen command, got nil")
	}

	msg := cmd()
	if _, ok := msg.(liveOutputMsg); !ok {
		t.Fatalf("expected follow-up liveOutputMsg, got %T", msg)
	}
}
