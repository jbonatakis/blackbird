package memory

import (
	"path/filepath"
	"testing"
)

func TestMemoryPaths(t *testing.T) {
	root := t.TempDir()
	memoryRoot := MemoryRoot(root)
	expectedRoot := filepath.Join(root, ".blackbird", "memory")
	if memoryRoot != expectedRoot {
		t.Fatalf("memory root = %s, want %s", memoryRoot, expectedRoot)
	}

	sessionPath := SessionPath(root)
	expectedSession := filepath.Join(expectedRoot, "session.json")
	if sessionPath != expectedSession {
		t.Fatalf("session path = %s, want %s", sessionPath, expectedSession)
	}

	walPath := TraceWALPath(root, "session123")
	expectedWal := filepath.Join(expectedRoot, "trace", "session123.wal")
	if walPath != expectedWal {
		t.Fatalf("wal path = %s, want %s", walPath, expectedWal)
	}

	defaultWal := TraceWALPath(root, "")
	expectedDefaultWal := filepath.Join(expectedRoot, "trace", "trace.wal")
	if defaultWal != expectedDefaultWal {
		t.Fatalf("default wal path = %s, want %s", defaultWal, expectedDefaultWal)
	}

	canonicalPath := CanonicalLogPath(root, "run456")
	expectedCanonical := filepath.Join(expectedRoot, "canonical", "run456.json")
	if canonicalPath != expectedCanonical {
		t.Fatalf("canonical log path = %s, want %s", canonicalPath, expectedCanonical)
	}

	defaultCanonical := CanonicalLogPath(root, "")
	expectedDefaultCanonical := filepath.Join(expectedRoot, "canonical", "canonical.json")
	if defaultCanonical != expectedDefaultCanonical {
		t.Fatalf("default canonical log path = %s, want %s", defaultCanonical, expectedDefaultCanonical)
	}

	artifactsPath := ArtifactsDBPath(root)
	expectedArtifacts := filepath.Join(expectedRoot, "artifacts.db")
	if artifactsPath != expectedArtifacts {
		t.Fatalf("artifacts db path = %s, want %s", artifactsPath, expectedArtifacts)
	}

	indexPath := IndexDBPath(root)
	expectedIndex := filepath.Join(expectedRoot, "index.db")
	if indexPath != expectedIndex {
		t.Fatalf("index db path = %s, want %s", indexPath, expectedIndex)
	}
}
