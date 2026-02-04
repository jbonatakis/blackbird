package execution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestResumeWithFeedbackCodex(t *testing.T) {
	script, argsPath, stdinPath := writeCaptureScript(t)

	prev := RunRecord{
		ID:                 "run-1",
		TaskID:             "task-1",
		Provider:           "codex",
		ProviderSessionRef: "session-123",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	runtime := agent.Runtime{
		Provider: "codex",
		Command:  script,
		Timeout:  2 * time.Second,
	}

	record, err := ResumeWithFeedback(context.Background(), runtime, prev, "fix this", StreamConfig{})
	if err != nil {
		t.Fatalf("ResumeWithFeedback: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", record.Status)
	}

	args := readLines(t, argsPath)
	wantArgs := []string{"exec", "--full-auto", "resume", "session-123"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args mismatch: got %v want %v", args, wantArgs)
	}
	stdin := readFile(t, stdinPath)
	if stdin != "fix this" {
		t.Fatalf("stdin mismatch: got %q", stdin)
	}
}

func TestResumeWithFeedbackClaude(t *testing.T) {
	script, argsPath, stdinPath := writeCaptureScript(t)

	prev := RunRecord{
		ID:                 "run-2",
		TaskID:             "task-2",
		Provider:           "claude",
		ProviderSessionRef: "session-456",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-2", Title: "Task"},
		},
	}

	runtime := agent.Runtime{
		Provider: "claude",
		Command:  script,
		Timeout:  2 * time.Second,
	}

	record, err := ResumeWithFeedback(context.Background(), runtime, prev, "review and fix", StreamConfig{})
	if err != nil {
		t.Fatalf("ResumeWithFeedback: %v", err)
	}
	if record.Status != RunStatusSuccess {
		t.Fatalf("expected success, got %s", record.Status)
	}

	args := readLines(t, argsPath)
	wantArgs := []string{"--permission-mode", "bypassPermissions", "--resume", "session-456"}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args mismatch: got %v want %v", args, wantArgs)
	}
	stdin := readFile(t, stdinPath)
	if stdin != "review and fix" {
		t.Fatalf("stdin mismatch: got %q", stdin)
	}
}

func TestResumeWithFeedbackRequiresFeedback(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "codex",
		Command:  "cat",
		Timeout:  2 * time.Second,
	}
	prev := RunRecord{
		ID:                 "run-blank",
		TaskID:             "task-blank",
		Provider:           "codex",
		ProviderSessionRef: "session-blank",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-blank", Title: "Task"},
		},
	}

	_, err := ResumeWithFeedback(context.Background(), runtime, prev, "   ", StreamConfig{})
	if err == nil || !strings.Contains(err.Error(), "resume feedback required") {
		t.Fatalf("expected resume feedback required error, got %v", err)
	}
}

func TestResumeWithFeedbackErrors(t *testing.T) {
	runtime := agent.Runtime{
		Provider: "codex",
		Command:  "cat",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	_, err := ResumeWithFeedback(context.Background(), runtime, RunRecord{
		ID:       "run-3",
		TaskID:   "task-3",
		Provider: "codex",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-3", Title: "Task"},
		},
	}, "feedback", StreamConfig{})
	if err == nil || !strings.Contains(err.Error(), "session ref") {
		t.Fatalf("expected missing session ref error, got %v", err)
	}

	runtimeUnsupported := agent.Runtime{
		Command:  "cat",
		UseShell: true,
		Timeout:  2 * time.Second,
	}
	_, err = ResumeWithFeedback(context.Background(), runtimeUnsupported, RunRecord{
		ID:                 "run-4",
		TaskID:             "task-4",
		Provider:           "unknown",
		ProviderSessionRef: "session-999",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-4", Title: "Task"},
		},
	}, "feedback", StreamConfig{})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported provider error, got %v", err)
	}

	runtimeMismatch := agent.Runtime{
		Provider: "claude",
		Command:  "cat",
		UseShell: true,
		Timeout:  2 * time.Second,
	}
	_, err = ResumeWithFeedback(context.Background(), runtimeMismatch, RunRecord{
		ID:                 "run-5",
		TaskID:             "task-5",
		Provider:           "codex",
		ProviderSessionRef: "session-777",
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-5", Title: "Task"},
		},
	}, "feedback", StreamConfig{})
	if err == nil || !strings.Contains(err.Error(), "provider mismatch") {
		t.Fatalf("expected provider mismatch error, got %v", err)
	}
}

func writeCaptureScript(t *testing.T) (string, string, string) {
	t.Helper()
	dir := t.TempDir()
	argsPath := filepath.Join(dir, "args.txt")
	stdinPath := filepath.Join(dir, "stdin.txt")
	scriptPath := filepath.Join(dir, "capture.sh")

	content := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\ncat - > %q\n", argsPath, stdinPath)
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return scriptPath, argsPath, stdinPath
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	data := readFile(t, path)
	if data == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSpace(data), "\n")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(data)
}
