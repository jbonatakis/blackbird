package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func TestBuildContextIncludesTaskAndDeps(t *testing.T) {
	tempDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	snapshotPath := filepath.Join(tempDir, ".blackbird", "snapshot.md")
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
		t.Fatalf("mkdir snapshot: %v", err)
	}
	if err := os.WriteFile(snapshotPath, []byte("snapshot"), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}

	now := time.Date(2026, 1, 28, 18, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"dep": {
				ID:        "dep",
				Title:     "Dependency",
				Status:    plan.StatusDone,
				CreatedAt: now,
				UpdatedAt: now,
			},
			"task": {
				ID:                 "task",
				Title:              "Task",
				Description:        "desc",
				AcceptanceCriteria: []string{"a", "b"},
				Prompt:             "do it",
				Deps:               []string{"dep"},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}

	ctx, err := BuildContext(g, "task")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.Task.ID != "task" || ctx.Task.Title != "Task" || ctx.Task.Prompt != "do it" {
		t.Fatalf("unexpected task context: %#v", ctx.Task)
	}
	if ctx.SystemPrompt == "" {
		t.Fatalf("expected system prompt")
	}
	if len(ctx.Dependencies) != 1 || ctx.Dependencies[0].ID != "dep" {
		t.Fatalf("unexpected deps: %#v", ctx.Dependencies)
	}
	if ctx.ProjectSnapshot != "snapshot" {
		t.Fatalf("unexpected snapshot: %q", ctx.ProjectSnapshot)
	}
}

func TestBuildContextErrorsOnUnknownTask(t *testing.T) {
	g := plan.WorkGraph{SchemaVersion: plan.SchemaVersion, Items: map[string]plan.WorkItem{}}
	_, err := BuildContext(g, "missing")
	if err == nil {
		t.Fatalf("expected error for unknown task")
	}
}

func TestBuildContextErrorsOnUnknownDep(t *testing.T) {
	now := time.Now().UTC()
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:        "task",
				Title:     "Task",
				Deps:      []string{"missing"},
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
	_, err := BuildContext(g, "task")
	if err == nil {
		t.Fatalf("expected error for unknown dependency")
	}
}

func TestBuildContextWithMemoryPackForCodex(t *testing.T) {
	tempDir := t.TempDir()
	session, err := memory.CreateSession(memory.SessionPath(tempDir), "Ship memory")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	now := time.Date(2026, 1, 29, 9, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:        "task",
				Title:     "Task",
				Prompt:    "do it",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	resolved := config.DefaultResolvedConfig()
	ctx, err := BuildContextWithOptions(g, "task", ContextBuildOptions{
		BaseDir:  tempDir,
		Provider: "codex",
		Memory:   &resolved.Memory,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("BuildContextWithOptions: %v", err)
	}
	if ctx.SessionID != session.SessionID {
		t.Fatalf("session id = %q, want %q", ctx.SessionID, session.SessionID)
	}
	if ctx.Memory == nil {
		t.Fatalf("expected memory context pack")
	}
	if ctx.Memory.SessionID != session.SessionID {
		t.Fatalf("memory session id = %q, want %q", ctx.Memory.SessionID, session.SessionID)
	}
}

func TestBuildContextWithMemoryPackNoopForClaude(t *testing.T) {
	tempDir := t.TempDir()
	now := time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC)
	g := plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			"task": {
				ID:        "task",
				Title:     "Task",
				Prompt:    "do it",
				Status:    plan.StatusTodo,
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	resolved := config.DefaultResolvedConfig()
	ctx, err := BuildContextWithOptions(g, "task", ContextBuildOptions{
		BaseDir:  tempDir,
		Provider: "claude",
		Memory:   &resolved.Memory,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("BuildContextWithOptions: %v", err)
	}
	if ctx.Memory != nil {
		t.Fatalf("expected no memory context pack for claude")
	}
	if ctx.SessionID != "" {
		t.Fatalf("expected empty session id for claude, got %q", ctx.SessionID)
	}
}
