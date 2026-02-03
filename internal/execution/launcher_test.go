package execution

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/config"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
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
	if record.Context.RunID == "" || record.Context.RunID != record.ID {
		t.Fatalf("expected run id to be set, got %q (record id %q)", record.Context.RunID, record.ID)
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

func TestLaunchAgentSetsCodexProxyEnv(t *testing.T) {
	restore := clearEnv(t, []string{envOpenAIBaseURL, envOpenAIAPIBase, envOpenAIDefaultHeaders})
	defer restore()

	runtime := agent.Runtime{
		Provider: "codex",
		Command:  "env",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		SessionID:     "session-123",
		Task:          TaskContext{ID: "task-7", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}

	env := parseEnvOutput(record.Stdout)
	expectedBase := proxyBaseURL(config.DefaultResolvedConfig().Memory.Proxy.ListenAddr, memprovider.CodexAdapter{}.BaseURLPrefix())
	if got := env[envOpenAIBaseURL]; got != expectedBase {
		t.Fatalf("base url = %q, want %q", got, expectedBase)
	}
	if got := env[envOpenAIAPIBase]; got != expectedBase {
		t.Fatalf("api base url = %q, want %q", got, expectedBase)
	}

	rawHeaders := env[envOpenAIDefaultHeaders]
	if rawHeaders == "" {
		t.Fatalf("expected default headers env")
	}
	headers := map[string]string{}
	if err := json.Unmarshal([]byte(rawHeaders), &headers); err != nil {
		t.Fatalf("parse headers: %v", err)
	}
	if got := headers[memprovider.HeaderBlackbirdSessionID]; got != "session-123" {
		t.Fatalf("session header = %q, want %q", got, "session-123")
	}
	if got := headers[memprovider.HeaderBlackbirdTaskID]; got != "task-7" {
		t.Fatalf("task header = %q, want %q", got, "task-7")
	}
	if got := headers[memprovider.HeaderBlackbirdRunID]; got != record.ID {
		t.Fatalf("run header = %q, want %q", got, record.ID)
	}
}

func TestLaunchAgentSkipsProxyEnvForClaude(t *testing.T) {
	restore := clearEnv(t, []string{envOpenAIBaseURL, envOpenAIAPIBase, envOpenAIDefaultHeaders})
	defer restore()

	runtime := agent.Runtime{
		Provider: "claude",
		Command:  "env",
		UseShell: true,
		Timeout:  2 * time.Second,
	}

	ctx := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		SessionID:     "session-claude",
		Task:          TaskContext{ID: "task-8", Title: "Task"},
	}

	record, err := LaunchAgent(context.Background(), runtime, ctx)
	if err != nil {
		t.Fatalf("LaunchAgent: %v", err)
	}

	env := parseEnvOutput(record.Stdout)
	if _, ok := env[envOpenAIBaseURL]; ok {
		t.Fatalf("expected no base url env for claude")
	}
	if _, ok := env[envOpenAIDefaultHeaders]; ok {
		t.Fatalf("expected no default headers env for claude")
	}
}

func parseEnvOutput(output string) map[string]string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	env := make(map[string]string, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		if eq := strings.Index(line, "="); eq != -1 {
			env[line[:eq]] = line[eq+1:]
		}
	}
	return env
}

func clearEnv(t *testing.T, keys []string) func() {
	t.Helper()
	type savedValue struct {
		value string
		ok    bool
	}
	saved := make(map[string]savedValue, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		saved[key] = savedValue{value: value, ok: ok}
		if ok {
			_ = os.Unsetenv(key)
		}
	}
	return func() {
		for key, entry := range saved {
			if entry.ok {
				_ = os.Setenv(key, entry.value)
			} else {
				_ = os.Unsetenv(key)
			}
		}
	}
}
