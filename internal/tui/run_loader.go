package tui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type RunDataLoaded struct {
	Data                  map[string]execution.RunRecord
	PendingParentFeedback map[string]execution.PendingParentReviewFeedback
	Err                   error
}

type runDataRefreshMsg struct{}

func (m Model) LoadRunData() tea.Cmd {
	return func() tea.Msg {
		baseDir := m.projectRoot
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return RunDataLoaded{
					Data:                  map[string]execution.RunRecord{},
					PendingParentFeedback: map[string]execution.PendingParentReviewFeedback{},
					Err:                   err,
				}
			}
		}

		data := make(map[string]execution.RunRecord)
		pendingFeedback := make(map[string]execution.PendingParentReviewFeedback)
		for id := range m.plan.Items {
			latest, err := execution.GetLatestRun(baseDir, id)
			if err != nil {
				return RunDataLoaded{
					Data:                  data,
					PendingParentFeedback: pendingFeedback,
					Err:                   err,
				}
			}
			if latest != nil {
				data[id] = *latest
			}

			pending, err := execution.LoadPendingParentReviewFeedback(baseDir, id)
			if err != nil {
				return RunDataLoaded{
					Data:                  data,
					PendingParentFeedback: pendingFeedback,
					Err:                   err,
				}
			}
			if pending != nil {
				pendingFeedback[id] = *pending
			}
		}

		return RunDataLoaded{
			Data:                  data,
			PendingParentFeedback: pendingFeedback,
			Err:                   nil,
		}
	}
}

func (m Model) RunDataRefreshCmd() tea.Cmd {
	interval := time.Duration(m.config.TUI.RunDataRefreshIntervalSeconds) * time.Second
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return runDataRefreshMsg{}
	})
}
