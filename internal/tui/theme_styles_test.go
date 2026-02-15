package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
)

func TestReadinessLabelColorMappingsForBuiltInThemes(t *testing.T) {
	labels := []struct {
		label  string
		expect func(Theme) lipgloss.Color
	}{
		{label: "READY", expect: func(theme Theme) lipgloss.Color { return theme.Colors.Info }},
		{label: "DONE", expect: func(theme Theme) lipgloss.Color { return theme.Colors.Success }},
		{label: "IN_PROGRESS", expect: func(theme Theme) lipgloss.Color { return theme.Colors.Warning }},
		{label: "BLOCKED", expect: func(theme Theme) lipgloss.Color { return theme.Colors.Danger }},
		{label: "FAILED", expect: func(theme Theme) lipgloss.Color { return theme.Colors.Danger }},
		{label: "QUEUED", expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
		{label: "WAITING_USER", expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
		{label: "SKIPPED", expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
		{label: "UNKNOWN", expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
	}

	for _, themeID := range []string{ThemeIDBlackbird, ThemeIDHighContrast} {
		theme := ResolveTheme(themeID)
		for _, tc := range labels {
			got := readinessLabelColor(theme, tc.label)
			want := tc.expect(theme)
			if got != want {
				t.Fatalf("%s readiness %q color = %q, want %q", themeID, tc.label, got, want)
			}
			if fg := readinessLabelStyleForTheme(theme, tc.label).GetForeground(); fg != want {
				t.Fatalf("%s readiness %q foreground = %q, want %q", themeID, tc.label, fg, want)
			}
		}
	}
}

func TestRunStatusColorMappingsForBuiltInThemes(t *testing.T) {
	statuses := []struct {
		status execution.RunStatus
		expect func(Theme) lipgloss.Color
	}{
		{status: execution.RunStatusRunning, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Info }},
		{status: execution.RunStatusSuccess, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Success }},
		{status: execution.RunStatusFailed, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Danger }},
		{status: execution.RunStatusWaitingUser, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Warning }},
		{status: execution.RunStatus("unknown"), expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
	}

	for _, themeID := range []string{ThemeIDBlackbird, ThemeIDHighContrast} {
		theme := ResolveTheme(themeID)
		for _, tc := range statuses {
			got := runStatusColor(theme, tc.status)
			want := tc.expect(theme)
			if got != want {
				t.Fatalf("%s run status %q color = %q, want %q", themeID, tc.status, got, want)
			}
			if fg := runStatusStyleForTheme(theme, tc.status).GetForeground(); fg != want {
				t.Fatalf("%s run status %q foreground = %q, want %q", themeID, tc.status, fg, want)
			}
		}
	}
}

func TestReadinessLabelColorMappingsFallbackToBlackbirdWhenThemeUnset(t *testing.T) {
	blackbird := ResolveTheme(ThemeIDBlackbird)
	unsetTheme := Theme{}

	if got, want := readinessLabelColor(unsetTheme, "READY"), blackbird.Colors.Info; got != want {
		t.Fatalf("unset readiness READY color = %q, want %q", got, want)
	}
	if got, want := readinessLabelStyleForTheme(unsetTheme, "READY").GetForeground(), blackbird.Colors.Info; got != want {
		t.Fatalf("unset readiness READY style fg = %q, want %q", got, want)
	}
	if got, want := readinessLabelColor(unsetTheme, "UNKNOWN"), blackbird.Colors.TextMuted; got != want {
		t.Fatalf("unset readiness UNKNOWN color = %q, want %q", got, want)
	}
}

func TestRunStatusColorMappingsFallbackToBlackbirdWhenThemeUnset(t *testing.T) {
	blackbird := ResolveTheme(ThemeIDBlackbird)
	unsetTheme := Theme{}

	if got, want := runStatusColor(unsetTheme, execution.RunStatusRunning), blackbird.Colors.Info; got != want {
		t.Fatalf("unset run status running color = %q, want %q", got, want)
	}
	if got, want := runStatusStyleForTheme(unsetTheme, execution.RunStatusRunning).GetForeground(), blackbird.Colors.Info; got != want {
		t.Fatalf("unset run status running style fg = %q, want %q", got, want)
	}
	if got, want := runStatusColor(unsetTheme, execution.RunStatus("unknown")), blackbird.Colors.TextMuted; got != want {
		t.Fatalf("unset run status unknown color = %q, want %q", got, want)
	}
}

func TestModalEmphasisColorMappingsForBuiltInThemes(t *testing.T) {
	emphases := []struct {
		name     string
		emphasis ModalEmphasis
		expect   func(Theme) lipgloss.Color
	}{
		{name: "accent", emphasis: ModalEmphasisAccent, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Accent }},
		{name: "info", emphasis: ModalEmphasisInfo, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Info }},
		{name: "success", emphasis: ModalEmphasisSuccess, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Success }},
		{name: "warning", emphasis: ModalEmphasisWarning, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Warning }},
		{name: "danger", emphasis: ModalEmphasisDanger, expect: func(theme Theme) lipgloss.Color { return theme.Colors.Danger }},
		{name: "muted", emphasis: ModalEmphasisMuted, expect: func(theme Theme) lipgloss.Color { return theme.Colors.TextMuted }},
	}

	for _, themeID := range []string{ThemeIDBlackbird, ThemeIDHighContrast} {
		theme := ResolveTheme(themeID)
		for _, tc := range emphases {
			want := tc.expect(theme)
			if got := modalEmphasisColor(theme, tc.emphasis); got != want {
				t.Fatalf("%s modal emphasis %s color = %q, want %q", themeID, tc.name, got, want)
			}
			if fg := modalEmphasisStyleForTheme(theme, tc.emphasis).GetForeground(); fg != want {
				t.Fatalf("%s modal emphasis %s style fg = %q, want %q", themeID, tc.name, fg, want)
			}
			if border := modalBorderStyleForTheme(theme, tc.emphasis).GetBorderTopForeground(); border != want {
				t.Fatalf("%s modal emphasis %s border fg = %q, want %q", themeID, tc.name, border, want)
			}
		}
	}
}

func TestButtonVariantStyleMappingsForBuiltInThemes(t *testing.T) {
	for _, themeID := range []string{ThemeIDBlackbird, ThemeIDHighContrast} {
		theme := ResolveTheme(themeID)

		primary := buttonVariantStyleForTheme(theme, ButtonVariantPrimary)
		if fg, want := primary.GetForeground(), theme.Colors.TextOnAccent; fg != want {
			t.Fatalf("%s primary fg = %q, want %q", themeID, fg, want)
		}
		if bg, want := primary.GetBackground(), theme.Colors.Accent; bg != want {
			t.Fatalf("%s primary bg = %q, want %q", themeID, bg, want)
		}
		if !primary.GetBold() {
			t.Fatalf("%s primary should be bold", themeID)
		}

		secondary := buttonVariantStyleForTheme(theme, ButtonVariantSecondary)
		if fg, want := secondary.GetForeground(), theme.Colors.TextPrimary; fg != want {
			t.Fatalf("%s secondary fg = %q, want %q", themeID, fg, want)
		}
		if bg, want := secondary.GetBackground(), theme.Colors.Surface; bg != want {
			t.Fatalf("%s secondary bg = %q, want %q", themeID, bg, want)
		}

		disabled := buttonVariantStyleForTheme(theme, ButtonVariantDisabled)
		if fg, want := disabled.GetForeground(), theme.Colors.TextMuted; fg != want {
			t.Fatalf("%s disabled fg = %q, want %q", themeID, fg, want)
		}
		if bg, want := disabled.GetBackground(), theme.Colors.SurfaceMuted; bg != want {
			t.Fatalf("%s disabled bg = %q, want %q", themeID, bg, want)
		}

		success := buttonVariantStyleForTheme(theme, ButtonVariantSuccess)
		if fg, want := success.GetForeground(), theme.Colors.TextOnAccent; fg != want {
			t.Fatalf("%s success fg = %q, want %q", themeID, fg, want)
		}
		if bg, want := success.GetBackground(), theme.Colors.Success; bg != want {
			t.Fatalf("%s success bg = %q, want %q", themeID, bg, want)
		}
		if !success.GetBold() {
			t.Fatalf("%s success should be bold", themeID)
		}

		danger := buttonVariantStyleForTheme(theme, ButtonVariantDanger)
		if fg, want := danger.GetForeground(), theme.Colors.TextOnAccent; fg != want {
			t.Fatalf("%s danger fg = %q, want %q", themeID, fg, want)
		}
		if bg, want := danger.GetBackground(), theme.Colors.Danger; bg != want {
			t.Fatalf("%s danger bg = %q, want %q", themeID, bg, want)
		}
		if !danger.GetBold() {
			t.Fatalf("%s danger should be bold", themeID)
		}
	}
}
