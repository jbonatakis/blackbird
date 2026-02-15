package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const settingsThemeDebounceDelay = 500 * time.Millisecond

type settingsThemeDebounceMsg struct {
	Token uint64
}

var settingsThemeDebounceCmd = func(delay time.Duration, token uint64) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return settingsThemeDebounceMsg{Token: token}
	})
}
