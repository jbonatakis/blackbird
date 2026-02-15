package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
)

type ModalEmphasis int

const (
	ModalEmphasisAccent ModalEmphasis = iota
	ModalEmphasisInfo
	ModalEmphasisSuccess
	ModalEmphasisWarning
	ModalEmphasisDanger
	ModalEmphasisMuted
)

type ButtonVariant int

const (
	ButtonVariantPrimary ButtonVariant = iota
	ButtonVariantSecondary
	ButtonVariantDisabled
	ButtonVariantSuccess
	ButtonVariantDanger
)

func readinessLabelStyleForTheme(theme Theme, label string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(readinessLabelColor(theme, label))
}

func readinessLabelColor(theme Theme, label string) lipgloss.Color {
	theme = themeOrDefault(theme)
	switch label {
	case "READY":
		return theme.Colors.Info
	case "DONE":
		return theme.Colors.Success
	case "IN_PROGRESS":
		return theme.Colors.Warning
	case "BLOCKED", "FAILED":
		return theme.Colors.Danger
	case "QUEUED", "WAITING_USER", "SKIPPED":
		return theme.Colors.TextMuted
	default:
		return theme.Colors.TextMuted
	}
}

func runStatusStyleForTheme(theme Theme, status execution.RunStatus) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(runStatusColor(theme, status))
}

func runStatusColor(theme Theme, status execution.RunStatus) lipgloss.Color {
	theme = themeOrDefault(theme)
	switch status {
	case execution.RunStatusSuccess:
		return theme.Colors.Success
	case execution.RunStatusFailed:
		return theme.Colors.Danger
	case execution.RunStatusWaitingUser:
		return theme.Colors.Warning
	case execution.RunStatusRunning:
		return theme.Colors.Info
	default:
		return theme.Colors.TextMuted
	}
}

func accentTextStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(theme.Colors.Accent)
}

func sectionHeaderStyleForTheme(theme Theme) lipgloss.Style {
	return accentTextStyleForTheme(theme).Copy().Bold(true)
}

func mutedTextStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(theme.Colors.TextMuted)
}

func successTextStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(theme.Colors.Success)
}

func dangerTextStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(theme.Colors.Danger)
}

func errorBannerStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return dangerTextStyleForTheme(theme).Copy().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Colors.Danger).
		Padding(0, 1)
}

func paneBorderStyleForTheme(theme Theme, active bool) lipgloss.Style {
	border, _ := paneChromeColorsForTheme(theme, active)
	return lipgloss.NewStyle().Foreground(border)
}

func paneTitleStyleForTheme(theme Theme, active bool) lipgloss.Style {
	_, title := paneChromeColorsForTheme(theme, active)
	return lipgloss.NewStyle().Bold(true).Foreground(title)
}

func paneChromeColorsForTheme(theme Theme, active bool) (lipgloss.Color, lipgloss.Color) {
	theme = themeOrDefault(theme)
	if active {
		return theme.Colors.BorderActive, theme.Colors.Accent
	}
	return theme.Colors.Border, theme.Colors.TextMuted
}

func modalEmphasisStyleForTheme(theme Theme, emphasis ModalEmphasis) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(modalEmphasisColor(theme, emphasis))
}

func modalTitleStyleForTheme(theme Theme, emphasis ModalEmphasis) lipgloss.Style {
	return modalEmphasisStyleForTheme(theme, emphasis).Copy().Bold(true)
}

func modalBorderStyleForTheme(theme Theme, emphasis ModalEmphasis) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().BorderForeground(modalEmphasisColor(theme, emphasis))
}

func buttonVariantStyleForTheme(theme Theme, variant ButtonVariant) lipgloss.Style {
	theme = themeOrDefault(theme)
	switch variant {
	case ButtonVariantPrimary:
		return lipgloss.NewStyle().
			Background(theme.Colors.Accent).
			Foreground(theme.Colors.TextOnAccent).
			Bold(true)
	case ButtonVariantDisabled:
		return lipgloss.NewStyle().
			Background(theme.Colors.SurfaceMuted).
			Foreground(theme.Colors.TextMuted)
	case ButtonVariantSuccess:
		return lipgloss.NewStyle().
			Background(theme.Colors.Success).
			Foreground(theme.Colors.TextOnAccent).
			Bold(true)
	case ButtonVariantDanger:
		return lipgloss.NewStyle().
			Background(theme.Colors.Danger).
			Foreground(theme.Colors.TextOnAccent).
			Bold(true)
	default:
		return lipgloss.NewStyle().
			Background(theme.Colors.Surface).
			Foreground(theme.Colors.TextPrimary)
	}
}

func treeIndicatorStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Foreground(theme.Colors.Accent)
}

func reviewMarkerStyleForTheme(theme Theme) lipgloss.Style {
	theme = themeOrDefault(theme)
	return lipgloss.NewStyle().Bold(true).Foreground(theme.Colors.Warning)
}

func modalEmphasisColor(theme Theme, emphasis ModalEmphasis) lipgloss.Color {
	switch emphasis {
	case ModalEmphasisInfo:
		return theme.Colors.Info
	case ModalEmphasisSuccess:
		return theme.Colors.Success
	case ModalEmphasisWarning:
		return theme.Colors.Warning
	case ModalEmphasisDanger:
		return theme.Colors.Danger
	case ModalEmphasisMuted:
		return theme.Colors.TextMuted
	default:
		return theme.Colors.Accent
	}
}

func themeOrDefault(theme Theme) Theme {
	if theme.ID == "" {
		return ResolveTheme(ThemeIDBlackbird)
	}
	return theme
}
