package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultTimeout = 60 * 10 * time.Second
	DefaultRetries = 1
	EnvProvider    = "BLACKBIRD_AGENT_PROVIDER"
	EnvCommand     = "BLACKBIRD_AGENT_CMD"
	EnvStream      = "BLACKBIRD_AGENT_STREAM"
)

type Runtime struct {
	Provider   string
	Command    string
	Args       []string
	UseShell   bool
	Timeout    time.Duration
	MaxRetries int
}

type Diagnostics struct {
	Stdout string
	Stderr string
	JSON   string
}

type RuntimeError struct {
	Message string
	Cause   error
	Diag    Diagnostics
}

func (e RuntimeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e RuntimeError) Unwrap() error {
	return e.Cause
}

func NewRuntimeFromEnv() (Runtime, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv(EnvProvider)))
	override := strings.TrimSpace(os.Getenv(EnvCommand))

	if override != "" {
		return Runtime{
			Provider:   provider,
			Command:    override,
			UseShell:   true,
			Timeout:    DefaultTimeout,
			MaxRetries: DefaultRetries,
		}, nil
	}

	if provider == "" {
		provider = "claude"
	}

	cmd, ok := defaultCommand(provider)
	if !ok {
		return Runtime{}, fmt.Errorf("unsupported agent provider %q (set %s or %s)", provider, EnvProvider, EnvCommand)
	}

	return Runtime{
		Provider:   provider,
		Command:    cmd,
		Timeout:    DefaultTimeout,
		MaxRetries: DefaultRetries,
	}, nil
}

func (r Runtime) Run(ctx context.Context, req Request) (Response, Diagnostics, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r.Timeout == 0 {
		r.Timeout = DefaultTimeout
	}
	if r.MaxRetries < 0 {
		r.MaxRetries = 0
	}

	if req.Metadata.Provider != "" && r.Provider != "" && req.Metadata.Provider != r.Provider {
		return Response{}, Diagnostics{}, RuntimeError{
			Message: fmt.Sprintf("request provider %q does not match runtime provider %q", req.Metadata.Provider, r.Provider),
		}
	}

	payload, err := EncodeRequest(req)
	if err != nil {
		return Response{}, Diagnostics{}, RuntimeError{Message: "encode request", Cause: err}
	}

	var lastErr error
	var lastDiag Diagnostics
	attempts := r.MaxRetries + 1
	for i := 0; i < attempts; i++ {
		logRequestDebug(payload, i+1, attempts)
		resp, diag, err := r.runOnce(ctx, payload, req.Metadata)
		if err == nil {
			return resp, diag, nil
		}
		lastErr = err
		lastDiag = diag
	}

	return Response{}, lastDiag, lastErr
}

func (r Runtime) runOnce(ctx context.Context, payload []byte, meta RequestMetadata) (Response, Diagnostics, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	command, args := r.Command, append([]string{}, r.Args...)
	flagArgs := buildFlagArgs(r.Provider, meta)
	if r.UseShell {
		command = appendShellArgs(command, flagArgs)
		args = nil
	} else {
		args = applyProviderArgs(r.Provider, args)
		args = append(args, flagArgs...)
	}

	var cmd *exec.Cmd
	if r.UseShell {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, command, args...)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stdout = agentOutputWriter(&stdout, os.Stdout)
	cmd.Stderr = agentOutputWriter(&stderr, os.Stderr)

	if err := cmd.Run(); err != nil {
		diag := Diagnostics{Stdout: stdout.String(), Stderr: stderr.String()}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Response{}, diag, RuntimeError{Message: "agent command timed out", Cause: err, Diag: diag}
		}
		return Response{}, diag, RuntimeError{Message: "agent command failed", Cause: err, Diag: diag}
	}

	diag := Diagnostics{Stdout: stdout.String(), Stderr: stderr.String()}
	jsonStr, err := ExtractJSON(diag.Stdout)
	if err != nil {
		return Response{}, diag, RuntimeError{Message: extractionErrorMessage(err), Cause: err, Diag: diag}
	}
	diag.JSON = jsonStr

	resp, err := DecodeResponse([]byte(jsonStr))
	if err != nil {
		return Response{}, diag, RuntimeError{Message: "decode response", Cause: err, Diag: diag}
	}
	if errs := ValidateResponse(resp); len(errs) != 0 {
		return Response{}, diag, RuntimeError{Message: formatValidationErrors(errs), Diag: diag}
	}

	return resp, diag, nil
}

func defaultCommand(provider string) (string, bool) {
	switch provider {
	case "claude":
		return "claude", true
	case "codex":
		return "codex", true
	default:
		return "", false
	}
}

func buildFlagArgs(provider string, meta RequestMetadata) []string {
	var args []string
	if meta.Model != "" {
		args = append(args, "--model", meta.Model)
	}
	if meta.MaxTokens != nil {
		args = append(args, "--max-tokens", fmt.Sprintf("%d", *meta.MaxTokens))
	}
	if meta.Temperature != nil {
		args = append(args, "--temperature", fmt.Sprintf("%g", *meta.Temperature))
	}
	if meta.ResponseFormat != "" {
		args = append(args, "--response-format", meta.ResponseFormat)
	}
	if meta.JSONSchema != "" && strings.EqualFold(provider, "claude") {
		args = append(args, "--json-schema", meta.JSONSchema)
	}
	return args
}

func applyProviderArgs(provider string, args []string) []string {
	switch strings.ToLower(provider) {
	case "codex":
		// Match execution behavior for non-interactive runs.
		return append([]string{"exec", "--full-auto", "--skip-git-repo-check"}, args...)
	case "claude":
		// Claude Code permission mode to bypass prompts for edits and commands.
		return append([]string{"--permission-mode", "bypassPermissions"}, args...)
	default:
		return args
	}
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

func extractionErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNoJSONFound):
		return "no JSON object found in agent output (expected a single JSON object or a fenced ```json block)"
	case errors.Is(err, ErrMultipleJSONFound):
		return "multiple JSON objects found in agent output (expected exactly one JSON object or fenced ```json block)"
	default:
		return "unable to extract JSON from agent output"
	}
}

func formatValidationErrors(errs []ValidationError) string {
	lines := make([]string, 0, len(errs))
	for _, e := range errs {
		lines = append(lines, fmt.Sprintf("%s: %s", e.Path, e.Message))
	}
	return "schema validation failed:\n- " + strings.Join(lines, "\n- ")
}

func agentDebugEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("BLACKBIRD_AGENT_DEBUG"))) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func agentOutputWriter(buf *bytes.Buffer, stream *os.File) io.Writer {
	if !agentStreamEnabled() {
		return buf
	}
	return io.MultiWriter(stream, buf)
}

func agentStreamEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvStream))) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func logRequestDebug(payload []byte, attempt int, total int) {
	if !agentDebugEnabled() {
		return
	}
	label := "Agent request (debug)"
	if total > 1 {
		label = fmt.Sprintf("Agent request (debug) attempt %d/%d", attempt, total)
	}
	fmt.Fprintln(os.Stdout, label+":")

	var out bytes.Buffer
	if err := json.Indent(&out, payload, "", "  "); err != nil {
		fmt.Fprintln(os.Stdout, string(payload))
		return
	}
	fmt.Fprintln(os.Stdout, out.String())
}
