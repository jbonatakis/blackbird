package tui

import (
	"os"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// agentIsFromEnv returns true when BLACKBIRD_AGENT_PROVIDER is set, so the
// effective agent is from environment and the TUI should show that and disable
// the change-agent action.
func agentIsFromEnv() bool {
	return strings.TrimSpace(os.Getenv(agent.EnvProvider)) != ""
}

// agentLabel returns the agent name shown in the TUI. Uses the same precedence
// as execution: env (BLACKBIRD_AGENT_PROVIDER) overrides agent.json, then
// saved selection, then default.
func agentLabel(m Model) string {
	if p := strings.ToLower(strings.TrimSpace(os.Getenv(agent.EnvProvider))); p != "" {
		if info, ok := agent.LookupAgent(p); ok {
			return info.Label
		}
	}
	if m.agentSelection.Agent.Label != "" {
		return m.agentSelection.Agent.Label
	}
	return agent.DefaultAgent().Label
}
