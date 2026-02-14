package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type executionStageMsg struct {
	state execution.ExecutionStageState
}

type executionStageDoneMsg struct{}

func listenExecutionStageCmd(ch <-chan execution.ExecutionStageState) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		state, ok := <-ch
		if !ok {
			return executionStageDoneMsg{}
		}
		return executionStageMsg{state: state}
	}
}
