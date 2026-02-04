package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// ResumeWithFeedback resumes a provider session and injects feedback as the follow-up prompt.
func ResumeWithFeedback(ctx context.Context, runtime agent.Runtime, previous RunRecord, feedback string, stream StreamConfig) (RunRecord, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if strings.TrimSpace(feedback) == "" {
		return RunRecord{}, fmt.Errorf("resume feedback required")
	}

	provider := normalizeProvider(previous.Provider)
	if provider == "" {
		return RunRecord{}, fmt.Errorf("resume with feedback requires provider on previous run")
	}
	if runtime.Provider != "" && normalizeProvider(runtime.Provider) != provider {
		return RunRecord{}, fmt.Errorf("resume with feedback provider mismatch: run uses %q, runtime uses %q", previous.Provider, runtime.Provider)
	}
	if !supportsResumeProvider(provider) {
		return RunRecord{}, fmt.Errorf("resume with feedback unsupported for provider %q", previous.Provider)
	}

	sessionRef := strings.TrimSpace(previous.ProviderSessionRef)
	if sessionRef == "" {
		return RunRecord{}, fmt.Errorf("resume with feedback requires provider session ref for run %q", previous.ID)
	}

	if runtime.Timeout == 0 {
		runtime.Timeout = agent.DefaultTimeout
	}
	if runtime.Command == "" {
		cmd, ok := defaultProviderCommand(provider)
		if !ok {
			return RunRecord{}, fmt.Errorf("resume with feedback unsupported for provider %q", previous.Provider)
		}
		runtime.Command = cmd
	}

	args, err := resumeArgs(provider, sessionRef, runtime.Args)
	if err != nil {
		return RunRecord{}, err
	}

	start := time.Now().UTC()
	ctxPack := previous.Context
	if ctxPack.Task.ID == "" {
		ctxPack.Task.ID = previous.TaskID
	}
	record := RunRecord{
		ID:                 newRunID(),
		TaskID:             previous.TaskID,
		Provider:           previous.Provider,
		ProviderSessionRef: sessionRef,
		StartedAt:          start,
		Status:             RunStatusRunning,
		Context:            ctxPack,
	}

	ctx, cancel := context.WithTimeout(ctx, runtime.Timeout)
	defer cancel()

	cmd, err := buildResumeCommand(ctx, runtime, args)
	if err != nil {
		return RunRecord{}, err
	}
	cmd.Stdin = strings.NewReader(feedback)

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

func resumeArgs(provider, sessionRef string, extra []string) ([]string, error) {
	switch normalizeProvider(provider) {
	case "codex":
		args := []string{"exec", "--full-auto", "resume", sessionRef}
		return append(args, extra...), nil
	case "claude":
		args := []string{"--permission-mode", "bypassPermissions", "--resume", sessionRef}
		return append(args, extra...), nil
	default:
		return nil, fmt.Errorf("resume with feedback unsupported for provider %q", provider)
	}
}

func buildResumeCommand(ctx context.Context, runtime agent.Runtime, args []string) (*exec.Cmd, error) {
	if runtime.UseShell {
		command := appendShellArgs(runtime.Command, args)
		return exec.CommandContext(ctx, "sh", "-c", command), nil
	}
	return exec.CommandContext(ctx, runtime.Command, args...), nil
}

func appendShellArgs(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, shellQuote(arg))
	}
	return command + " " + strings.Join(quoted, " ")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$&;|<>*?()[]{}!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
