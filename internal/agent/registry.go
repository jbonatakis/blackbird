package agent

import "strings"

// AgentID provides a stable identifier for selecting agent runtimes.
type AgentID string

const (
	AgentClaude AgentID = "claude"
	AgentCodex  AgentID = "codex"
)

type AgentInfo struct {
	ID    AgentID
	Label string
}

var AgentRegistry = []AgentInfo{
	{ID: AgentClaude, Label: "Claude"},
	{ID: AgentCodex, Label: "Codex"},
}

func SupportedAgentIDs() []AgentID {
	ids := make([]AgentID, 0, len(AgentRegistry))
	for _, agent := range AgentRegistry {
		ids = append(ids, agent.ID)
	}
	return ids
}

func LookupAgent(id string) (AgentInfo, bool) {
	normalized := strings.ToLower(strings.TrimSpace(id))
	for _, agent := range AgentRegistry {
		if string(agent.ID) == normalized {
			return agent, true
		}
	}
	return AgentInfo{}, false
}

func DefaultAgent() AgentInfo {
	if agent, ok := LookupAgent(string(AgentClaude)); ok {
		return agent
	}
	if len(AgentRegistry) > 0 {
		return AgentRegistry[0]
	}
	return AgentInfo{ID: AgentClaude, Label: "Claude"}
}
