package tui

import "testing"

func TestListenLiveOutputCmdReturnsChunk(t *testing.T) {
	ch := make(chan liveOutputMsg, 1)
	ch <- liveOutputMsg{stream: "stdout", data: "hello"}

	cmd := listenLiveOutputCmd(ch)
	if cmd == nil {
		t.Fatalf("expected cmd, got nil")
	}

	msg := cmd()
	chunk, ok := msg.(liveOutputMsg)
	if !ok {
		t.Fatalf("expected liveOutputMsg, got %T", msg)
	}
	if chunk.stream != "stdout" || chunk.data != "hello" {
		t.Fatalf("unexpected chunk: %#v", chunk)
	}
}

func TestListenLiveOutputCmdReturnsDoneOnClose(t *testing.T) {
	ch := make(chan liveOutputMsg)
	close(ch)

	cmd := listenLiveOutputCmd(ch)
	if cmd == nil {
		t.Fatalf("expected cmd, got nil")
	}

	msg := cmd()
	if _, ok := msg.(liveOutputDoneMsg); !ok {
		t.Fatalf("expected liveOutputDoneMsg, got %T", msg)
	}
}
