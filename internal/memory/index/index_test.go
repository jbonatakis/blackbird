package index

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/artifact"
)

func newArtifact(id, session, task, run string, typ artifact.ArtifactType, text string) artifact.Artifact {
	return artifact.Artifact{
		SchemaVersion:  artifact.SchemaVersion,
		ArtifactID:     id,
		SessionID:      session,
		TaskID:         task,
		RunID:          run,
		Type:           typ,
		Content:        artifact.Content{Text: text},
		BuilderVersion: artifact.BuilderVersion,
	}
}

func buildIndex(t *testing.T, artifacts []artifact.Artifact, times map[string]time.Time) *Index {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "index.db")
	idx, err := Open(path)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	opts := RebuildOptions{
		TimestampFor: func(art artifact.Artifact) time.Time {
			if ts, ok := times[art.ArtifactID]; ok {
				return ts
			}
			return time.Time{}
		},
		Now: time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC),
	}
	if err := idx.Rebuild(artifacts, opts); err != nil {
		_ = idx.Close()
		t.Fatalf("rebuild index: %v", err)
	}
	t.Cleanup(func() {
		_ = idx.Close()
	})
	return idx
}
