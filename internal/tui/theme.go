package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme is a named collection of semantic tokens used by TUI render helpers.
type Theme struct {
	ID     string
	Colors ColorTokens
}

// ColorTokens are semantic colors used by status and chrome style helpers.
type ColorTokens struct {
	TextPrimary   lipgloss.Color
	TextMuted     lipgloss.Color
	TextOnAccent  lipgloss.Color
	Accent        lipgloss.Color
	Info          lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Danger        lipgloss.Color
	Surface       lipgloss.Color
	SurfaceActive lipgloss.Color
	SurfaceMuted  lipgloss.Color
	Border        lipgloss.Color
	BorderActive  lipgloss.Color
}
