package execution

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestLaunchAgentClaudeSetsSessionID(t *testing.T) {
	script, argsPath, _ := writeLaunchCaptureScript(t)

	runtime := agent.Runtime{
		Provider: "claude",
		Command:  script,
		Timeout:  2 * time.Second,
	}
	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task:          TaskContext{ID: "task-7-claude", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}
	if record.ProviderSessionRef == "" {
		t.Fatalf("expected provider session ref to be set")
	}

	args := readLinesFromFile(t, argsPath)
	wantArgs := []string{"--permission-mode", "bypassPermissions", "--session-id", record.ProviderSessionRef}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args mismatch: got %v want %v", args, wantArgs)
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

func writeLaunchCaptureScript(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	stdinPath := filepath.Join(dir, "stdin.txt")
	scriptPath := filepath.Join(dir, "capture.sh")

	content := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"" + argsPath + "\"\ncat - > \"" + stdinPath + "\"\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return scriptPath, argsPath, stdinPath
}

func readLinesFromFile(t *testing.T, path string) []string {
	t.Helper()
	data := readFileString(t, path)
	if data == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSpace(data), "\n")
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(data)
}
