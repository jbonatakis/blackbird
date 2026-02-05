package execution

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// LaunchAgent executes the agent command with the provided context pack.
func LaunchAgent(ctx context.Context, runtime agent.Runtime, contextPack ContextPack) (RunRecord, error) {
	return LaunchAgentWithStream(ctx, runtime, contextPack, StreamConfig{})
}

// LaunchAgentWithStream executes the agent command with optional live output streaming.
func LaunchAgentWithStream(ctx context.Context, runtime agent.Runtime, contextPack ContextPack, stream StreamConfig) (RunRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runtime.Timeout == 0 {
		runtime.Timeout = agent.DefaultTimeout
	}
	runtime = applySelectedProvider(runtime)
	if runtime.Command == "" {
		return RunRecord{}, fmt.Errorf("agent command required")
	}
	if contextPack.Task.ID == "" {
		return RunRecord{}, fmt.Errorf("context pack task id required")
	}

	payload, err := json.Marshal(contextPack)
	if err != nil {
		return RunRecord{}, fmt.Errorf("encode context pack: %w", err)
	}

	start := time.Now().UTC()
	record := RunRecord{
		ID:        newRunID(),
		TaskID:    contextPack.Task.ID,
		Provider:  runtime.Provider,
		StartedAt: start,
		Status:    RunStatusRunning,
		Context:   contextPack,
	}
	sessionRef := ""
	if supportsResumeProvider(record.Provider) {
		switch normalizeProvider(record.Provider) {
		case "claude":
			if !runtime.UseShell {
				sessionRef = newSessionID()
				record.ProviderSessionRef = sessionRef
			}
		case "codex":
			record.ProviderSessionRef = record.ID
		}
	}

	ctx, cancel := context.WithTimeout(ctx, runtime.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.UseShell {
		cmd = exec.CommandContext(ctx, "sh", "-c", runtime.Command)
	} else {
		args := append([]string{}, runtime.Args...)
		args = buildLaunchArgs(runtime.Provider, args, sessionRef)
		cmd = exec.CommandContext(ctx, runtime.Command, args...)
	}
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = streamWriter(&stdout, stream.Stdout, os.Stdout)
	cmd.Stderr = streamWriter(&stderr, stream.Stderr, os.Stderr)

	execErr := cmd.Run()
	completed := time.Now().UTC()
	record.CompletedAt = &completed
	record.Stdout = stdout.String()
	record.Stderr = stderr.String()

	questions, qErr := ParseQuestions(record.Stdout)
	if qErr != nil {
		execErr = errors.Join(execErr, qErr)
	}

	exitCode := extractExitCode(execErr)
	if exitCode != nil {
		record.ExitCode = exitCode
	}

	if len(questions) > 0 {
		record.Status = RunStatusWaitingUser
		return record, nil
	}

	if execErr != nil {
		record.Status = RunStatusFailed
		record.Error = execErr.Error()
		return record, execErr
	}

	record.Status = RunStatusSuccess
	return record, nil
}

type StreamConfig struct {
	Stdout io.Writer
	Stderr io.Writer
}

func streamWriter(buf *bytes.Buffer, cfgWriter io.Writer, envWriter io.Writer) io.Writer {
	writers := []io.Writer{buf}
	if cfgWriter != nil {
		writers = append(writers, cfgWriter)
	}
	if os.Getenv(agent.EnvStream) == "1" && envWriter != nil {
		writers = append(writers, envWriter)
	}
	if len(writers) == 1 {
		return buf
	}
	return io.MultiWriter(writers...)
}

func newRunID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	// RFC 4122 variant and version 4.
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	encoded := hex.EncodeToString(b[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		encoded[0:8],
		encoded[8:12],
		encoded[12:16],
		encoded[16:20],
		encoded[20:32],
	)
}

func extractExitCode(err error) *int {
	if err == nil {
		code := 0
		return &code
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		return &code
	}
	return nil
}

func buildLaunchArgs(provider string, args []string, sessionRef string) []string {
	switch normalizeProvider(provider) {
	case "codex":
		// Prefer headless auto-approve for execution runs.
		return append([]string{"exec", "--full-auto"}, args...)
	case "claude":
		// Claude Code permission mode to bypass prompts for edits and commands.
		prefix := []string{"--permission-mode", "bypassPermissions"}
		if strings.TrimSpace(sessionRef) != "" {
			prefix = append(prefix, "--session-id", sessionRef)
		}
		return append(prefix, args...)
	default:
		return args
	}
}

func applySelectedProvider(runtime agent.Runtime) agent.Runtime {
	if strings.TrimSpace(runtime.Provider) != "" {
		return runtime
	}
	selection, err := agent.LoadAgentSelection(agent.AgentSelectionPath())
	if err != nil && selection.Agent.ID == "" {
		return runtime
	}
	if selection.Agent.ID != "" {
		runtime.Provider = string(selection.Agent.ID)
	}
	return runtime
}
