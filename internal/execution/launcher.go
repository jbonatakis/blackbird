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
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/memory"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
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

	start := time.Now().UTC()
	runID := newRunID()
	contextPack.RunID = runID
	updatedPack, proxyEnv, err := applyMemoryProxy(contextPack, runtime)
	if err != nil {
		return RunRecord{}, err
	}

	payload, err := json.Marshal(updatedPack)
	if err != nil {
		return RunRecord{}, fmt.Errorf("encode context pack: %w", err)
	}

	record := RunRecord{
		ID:        runID,
		TaskID:    updatedPack.Task.ID,
		Provider:  runtime.Provider,
		StartedAt: start,
		Status:    RunStatusRunning,
		Context:   updatedPack,
	}

	ctx, cancel := context.WithTimeout(ctx, runtime.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.UseShell {
		cmd = exec.CommandContext(ctx, "sh", "-c", runtime.Command)
	} else {
		args := append([]string{}, runtime.Args...)
		args = applyAutoApproveArgs(runtime.Provider, args)
		cmd = exec.CommandContext(ctx, runtime.Command, args...)
	}
	applyEnv(cmd, proxyEnv)
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

func applyAutoApproveArgs(provider string, args []string) []string {
	switch provider {
	case "codex":
		// Prefer headless auto-approve for execution runs.
		return append([]string{"exec", "--full-auto"}, args...)
	case "claude":
		// Claude Code permission mode to bypass prompts for edits and commands.
		return append([]string{"--permission-mode", "bypassPermissions"}, args...)
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

const (
	envOpenAIBaseURL        = "OPENAI_BASE_URL"
	envOpenAIAPIBase        = "OPENAI_API_BASE"
	envOpenAIDefaultHeaders = "OPENAI_DEFAULT_HEADERS"
)

func applyMemoryProxy(contextPack ContextPack, runtime agent.Runtime) (ContextPack, map[string]string, error) {
	adapter := memprovider.Select(runtime.Provider)
	if adapter == nil {
		return contextPack, nil, nil
	}

	baseDir := resolveBaseDir("")
	cfg, err := config.LoadConfig(baseDir)
	if err != nil {
		cfg = config.DefaultResolvedConfig()
	}
	if !adapter.Enabled(cfg.Memory) {
		return contextPack, nil, nil
	}

	sessionID := strings.TrimSpace(contextPack.SessionID)
	sessionGoal := strings.TrimSpace(contextPack.SessionGoal)
	if sessionID == "" {
		session, _, err := memory.LoadOrCreateSession(memory.SessionPath(baseDir), sessionGoal)
		if err != nil {
			return ContextPack{}, nil, err
		}
		sessionID = session.SessionID
		if sessionGoal == "" {
			sessionGoal = session.Goal
		}
		contextPack.SessionID = sessionID
		contextPack.SessionGoal = sessionGoal
	}
	if contextPack.Memory != nil {
		if contextPack.Memory.SessionID == "" {
			contextPack.Memory.SessionID = sessionID
		}
		if contextPack.Memory.SessionGoal == "" {
			contextPack.Memory.SessionGoal = sessionGoal
		}
	}

	headers := adapter.BaseHeaders(memprovider.RequestIDs{
		SessionID: sessionID,
		TaskID:    contextPack.Task.ID,
		RunID:     contextPack.RunID,
	})
	env := buildProxyEnv(cfg.Memory.Proxy, adapter, headers)
	return contextPack, env, nil
}

func buildProxyEnv(proxy config.ResolvedMemoryProxy, adapter memprovider.Adapter, headers http.Header) map[string]string {
	baseURL := proxyBaseURL(proxy.ListenAddr, adapter.BaseURLPrefix())
	if baseURL == "" && len(headers) == 0 {
		return nil
	}
	env := map[string]string{}
	if baseURL != "" {
		env[envOpenAIBaseURL] = baseURL
		env[envOpenAIAPIBase] = baseURL
	}
	if len(headers) != 0 {
		if encoded := encodeHeaders(headers); encoded != "" {
			env[envOpenAIDefaultHeaders] = encoded
		}
	}
	return env
}

func proxyBaseURL(listenAddr string, prefix string) string {
	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		return ""
	}
	if !strings.Contains(addr, "://") {
		addr = "http://" + addr
	}
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return strings.TrimRight(addr, "/")
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(addr, "/") + prefix
}

func encodeHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	values := make(map[string]string, len(headers))
	for key, vals := range headers {
		trimmed := make([]string, 0, len(vals))
		for _, val := range vals {
			if v := strings.TrimSpace(val); v != "" {
				trimmed = append(trimmed, v)
			}
		}
		if len(trimmed) == 0 {
			continue
		}
		values[key] = strings.Join(trimmed, ",")
	}
	if len(values) == 0 {
		return ""
	}
	data, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(data)
}

func applyEnv(cmd *exec.Cmd, updates map[string]string) {
	if len(updates) == 0 {
		return
	}
	env := cmd.Env
	if len(env) == 0 {
		env = os.Environ()
	}
	index := make(map[string]int, len(env))
	for i, entry := range env {
		if eq := strings.Index(entry, "="); eq != -1 {
			index[entry[:eq]] = i
		}
	}
	for key, value := range updates {
		if idx, ok := index[key]; ok {
			env[idx] = key + "=" + value
		} else {
			env = append(env, key+"="+value)
		}
	}
	cmd.Env = env
}
