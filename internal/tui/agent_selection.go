package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
)

type AgentSelectionLoaded struct {
	Selection agent.AgentSelection
	Err       error
}

type AgentSelectionSaved struct {
	Selection agent.AgentSelection
	Err       error
}

func (m Model) LoadAgentSelection() tea.Cmd {
	return func() tea.Msg {
		selection, err := agent.LoadAgentSelection(agent.AgentSelectionPath())
		return AgentSelectionLoaded{Selection: selection, Err: err}
	}
}

func SaveAgentSelectionCmd(selectedAgent string) tea.Cmd {
	return func() tea.Msg {
		err := agent.SaveAgentSelection(agent.AgentSelectionPath(), selectedAgent)
		if err != nil {
			return AgentSelectionSaved{Err: err}
		}
		selection, loadErr := agent.LoadAgentSelection(agent.AgentSelectionPath())
		if loadErr != nil {
			return AgentSelectionSaved{Selection: selection, Err: loadErr}
		}
		return AgentSelectionSaved{Selection: selection, Err: nil}
	}
}
