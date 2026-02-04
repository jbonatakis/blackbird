package execution

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestLaunchAgentSuccess(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "test",
		Command:  "cat",
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-1", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", record.Status)
	}
	if record.TaskID != "task-1" {
		t.Fatalf("expected task id, got %s", record.TaskID)
	}
	if record.Stdout == "" {
		t.Fatalf("expected stdout capture")
	}
}

func TestLaunchAgentWaitingUser(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "test",
		Command:  "printf '{\"tool\":\"AskUserQuestion\",\"id\":\"q1\",\"prompt\":\"Continue?\"}'",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-2", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.Status != RunStatusWaitingUser {
		t.Fatalf("expected waiting_user, got %s", record.Status)
	}
}

func TestLaunchAgentFailure(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "test",
		Command:  "exit 2",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-3", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err == nil {
		t.Fatalf("expected error")
	}
	if record.Status != RunStatusFailed {
		t.Fatalf("expected failed, got %s", record.Status)
	}
	if record.ExitCode == nil || *record.ExitCode != 2 {
		t.Fatalf("expected exit code 2, got %#v", record.ExitCode)
	}
}

func TestLaunchAgentWithStreamWritesOutput(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "test",
		Command:  "printf 'streamed-output'",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-4", Title: "Task"},
	}

	var streamed bytes.Buffer
	record, err := LaunchAgentWithStream(context.Background(), runtime, ctx, StreamConfig{
		Stdout: &streamed,
	})
	if err != nil {
		t.Fatalf("LaunchAgentWithStream: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", record.Status)
	}
	if streamed.String() == "" {
		t.Fatalf("expected streamed stdout output")
	}
	if record.Stdout == "" {
		t.Fatalf("expected captured stdout output")
	}
}

func TestLaunchAgentDefaultsProviderToSelectedAgent(t *testing.T) {
	dir := t.TempDir()
	if err := agent.SaveAgentSelection(filepath.Join(dir, ".blackbird", "agent.json"), "codex"); err != nil {
		t.Fatalf("SaveAgentSelection: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	runtime := agent.Runtime{
		Command:  "cat",
		UseShell: true, // avoid provider args (e.g. codex "exec --full-auto") being passed to cat
		Timeout:  2 * time.Second,
	}
	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-5", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.Provider != "codex" {
		t.Fatalf("expected provider codex, got %q", record.Provider)
	}
}

func TestLaunchAgentKeepsExplicitProvider(t *testing.T) {
	dir := t.TempDir()
	if err := agent.SaveAgentSelection(filepath.Join(dir, ".blackbird", "agent.json"), "codex"); err != nil {
		t.Fatalf("SaveAgentSelection: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	runtime := agent.Runtime{
		Provider: "claude",
		Command:  "cat",
		UseShell: true, // avoid provider args being passed to cat
		Timeout:  2 * time.Second,
	}
	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-6", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.Provider != "claude" {
		t.Fatalf("expected provider claude, got %q", record.Provider)
	}
}

func TestLaunchAgentSetsProviderSessionRef(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "codex",
		Command:  "cat",
		UseShell: true, // avoid provider args being passed to cat
		Timeout:  2 * time.Second,
	}
	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-7", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.ProviderSessionRef == "" {
		t.Fatalf("expected provider session ref to be set")
	}
	if record.ProviderSessionRef != record.ID {
		t.Fatalf("expected provider session ref to match run id, got %q", record.ProviderSessionRef)
	}
}
