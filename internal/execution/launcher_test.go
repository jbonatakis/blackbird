package execution

import (
	"bytes"
	"context"
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
