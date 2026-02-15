package tui

import (
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
)

type ConfigStateLoaded struct {
	Config     config.ResolvedConfig
	Resolution config.SettingsResolution
	Err        error
}

type configRefreshMsg struct{}

func (m Model) LoadConfigState() tea.Cmd {
	return func() tea.Msg {
		projectRoot := m.projectRoot
		if projectRoot == "" {
			var err error
			projectRoot, err = os.Getwd()
			if err != nil {
				return ConfigStateLoaded{Err: err}
			}
		}

		resolved, err := config.LoadConfig(projectRoot)
		if err != nil {
			return ConfigStateLoaded{Err: err}
		}
		resolution, err := config.ResolveSettings(projectRoot)
		if err != nil {
			return ConfigStateLoaded{Err: err}
		}

		return ConfigStateLoaded{
			Config:     resolved,
			Resolution: resolution,
			Err:        nil,
		}
	}
}

func (m Model) ConfigRefreshCmd() tea.Cmd {
	interval := time.Duration(m.config.TUI.RunDataRefreshIntervalSeconds) * time.Second
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return configRefreshMsg{}
	})
}

func applyRefreshedSettingsState(
	current SettingsState,
	resolved config.ResolvedConfig,
	resolution config.SettingsResolution,
) SettingsState {
	current.Options = config.OptionRegistry()
	current.Resolved = resolved
	current.Resolution = resolution
	current.Err = nil
	return current
}
