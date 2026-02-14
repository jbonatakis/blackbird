package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type parentReviewRunMsg struct {
	run execution.RunRecord
}

type parentReviewRunDoneMsg struct{}

func listenParentReviewRunCmd(ch <-chan execution.RunRecord) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		run, ok := <-ch
		if !ok {
			return parentReviewRunDoneMsg{}
		}
		return parentReviewRunMsg{run: run}
	}
}
