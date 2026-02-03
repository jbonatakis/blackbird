package memory

import "testing"

func TestLoadOrCreateSession(t *testing.T) {
	root := t.TempDir()
	path := SessionPath(root)

	created, wasCreated, err := LoadOrCreateSession(path, "Ship memory capture")
	if err != nil {
		t.Fatalf("load or create: %v", err)
	}
	if !wasCreated {
		t.Fatalf("expected session to be created")
	}
	if created.SessionID == "" {
		t.Fatalf("expected session id to be set")
	}
	if created.Goal != "Ship memory capture" {
		t.Fatalf("goal = %q, want %q", created.Goal, "Ship memory capture")
	}

	loaded, wasCreated, err := LoadOrCreateSession(path, "Ignore this")
	if err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if wasCreated {
		t.Fatalf("expected session to be loaded")
	}
	if loaded.SessionID != created.SessionID {
		t.Fatalf("session id = %s, want %s", loaded.SessionID, created.SessionID)
	}
	if loaded.Goal != created.Goal {
		t.Fatalf("goal = %q, want %q", loaded.Goal, created.Goal)
	}
}
