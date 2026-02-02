package tui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type RunDataLoaded struct {
	Data map[string]execution.RunRecord
	Err  error
}

type runDataRefreshMsg struct{}

func (m Model) LoadRunData() tea.Cmd {
	return func() tea.Msg {
		baseDir := m.projectRoot
		if baseDir == "" {
			var err error
			baseDir, err = os.Getwd()
			if err != nil {
				return RunDataLoaded{Data: map[string]execution.RunRecord{}, Err: err}
			}
		}

		data := make(map[string]execution.RunRecord)
		for id := range m.plan.Items {
			latest, err := execution.GetLatestRun(baseDir, id)
			if err != nil {
				return RunDataLoaded{Data: data, Err: err}
			}
			if latest != nil {
				data[id] = *latest
			}
		}

		return RunDataLoaded{Data: data, Err: nil}
	}
}

func (m Model) RunDataRefreshCmd() tea.Cmd {
	interval := time.Duration(m.config.TUI.RunDataRefreshIntervalSeconds) * time.Second
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return runDataRefreshMsg{}
	})
}
