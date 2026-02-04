package derive

import (
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
	"github.com/jbonatakis/blackbird/internal/memory/index"
	"github.com/jbonatakis/blackbird/internal/memory/trace"
)

func TestFromWALDerivesArtifactsAndRebuildsIndex(t *testing.T) {
	tempDir := t.TempDir()
	walPath := memory.TraceWALPath(tempDir, "")

	writer, err := trace.NewWALWriter(walPath, trace.Options{FsyncOnWrite: false, FsyncOnWriteSet: true})
	if err != nil {
		t.Fatalf("new wal writer: %v", err)
	}

	data := `{"choices":[{"delta":{"content":"Decision: use sqlite."}}]}`
	events := []trace.Event{
		{Type: trace.EventResponseStart, RequestID: "req-1", RunID: "run-1", SessionID: "s1", TaskID: "t1", Status: 200},
		{Type: trace.EventResponseBody, RequestID: "req-1", RunID: "run-1", SessionID: "s1", TaskID: "t1", Seq: 1, Body: []byte("data: " + data + "\n\n")},
		{Type: trace.EventResponseEnd, RequestID: "req-1", RunID: "run-1", SessionID: "s1", TaskID: "t1"},
	}

	for _, ev := range events {
		if err := writer.Append(ev); err != nil {
			_ = writer.Close()
			t.Fatalf("append wal: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close wal: %v", err)
	}

	if err := FromWAL(Options{ProjectRoot: tempDir, RunID: "run-1"}); err != nil {
		t.Fatalf("derive: %v", err)
	}

	store, found, err := artifact.LoadStoreForProject(tempDir)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if !found {
		t.Fatalf("expected artifact store to be created")
	}
	if len(store.Artifacts) == 0 {
		t.Fatalf("expected artifacts, got none")
	}

	foundDecision := false
	for _, art := range store.Artifacts {
		if art.Type == artifact.ArtifactDecision && strings.Contains(strings.ToLower(art.Content.Text), "sqlite") {
			foundDecision = true
			break
		}
	}
	if !foundDecision {
		t.Fatalf("expected decision artifact mentioning sqlite")
	}

	results, err := index.SearchForProject(tempDir, index.SearchOptions{Query: "sqlite"})
	if err != nil {
		t.Fatalf("search index: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected index results")
	}
}

func TestFromWALNoopWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	if err := FromWAL(Options{ProjectRoot: tempDir}); err != nil {
		t.Fatalf("derive: %v", err)
	}
	_, found, err := artifact.LoadStoreForProject(tempDir)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if found {
		t.Fatalf("expected no artifact store when wal is empty")
	}
}
