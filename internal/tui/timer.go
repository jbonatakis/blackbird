package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type timerTickMsg struct{}

type timerStartMsg struct{}

func StartTimerCmd() tea.Cmd {
	return func() tea.Msg {
		return timerStartMsg{}
	}
}

func TickCmd() tea.Cmd {
	return func() tea.Msg {
		<-time.After(1 * time.Second)
		return timerTickMsg{}
	}
}

func hasActiveRuns(runData map[string]execution.RunRecord) bool {
	for _, record := range runData {
		if record.Status == execution.RunStatusRunning || record.Status == execution.RunStatusWaitingUser {
			return true
		}
	}
	return false
}
