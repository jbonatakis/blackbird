package tui

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
)

func TestBuiltInThemeIDsDeterministicOrder(t *testing.T) {
	want := []string{ThemeIDBlackbird, ThemeIDHighContrast}
	got := BuiltInThemeIDs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuiltInThemeIDs() = %#v, want %#v", got, want)
	}
}

func TestResolveThemeKnownAndFallback(t *testing.T) {
	for _, id := range BuiltInThemeIDs() {
		got := ResolveTheme(id)
		if got.ID != id {
			t.Fatalf("ResolveTheme(%q).ID = %q, want %q", id, got.ID, id)
		}
	}

	for _, unknownID := range []string{"does-not-exist", "", "BLACKBIRD"} {
		fallback := ResolveTheme(unknownID)
		blackbird := ResolveTheme(ThemeIDBlackbird)
		if fallback != blackbird {
			t.Fatalf("ResolveTheme(%q) = %#v, want blackbird %#v", unknownID, fallback, blackbird)
		}
	}
}

func TestBlackbirdTokenBaselinesForStatusAndChrome(t *testing.T) {
	theme := ResolveTheme(ThemeIDBlackbird)

	assertThemeTokenSet(t, "blackbird", theme)

	assertToken(t, "blackbird.ready", readinessLabelColor(theme, "READY"), theme.Colors.Info)
	assertToken(t, "blackbird.inProgress", readinessLabelColor(theme, "IN_PROGRESS"), theme.Colors.Warning)
	assertToken(t, "blackbird.blocked", readinessLabelColor(theme, "BLOCKED"), theme.Colors.Danger)
	assertToken(t, "blackbird.unknownReadiness", readinessLabelColor(theme, "SOMETHING_ELSE"), theme.Colors.TextMuted)
	assertToken(t, "blackbird.failed", runStatusColor(theme, execution.RunStatusFailed), theme.Colors.Danger)
	assertToken(t, "blackbird.running", runStatusColor(theme, execution.RunStatusRunning), theme.Colors.Info)
	assertToken(t, "blackbird.defaultRunStatus", runStatusColor(theme, execution.RunStatus("queued")), theme.Colors.TextMuted)

	border, title := paneChromeColorsForTheme(theme, false)
	assertToken(t, "blackbird.chrome.border", border, theme.Colors.Border)
	assertToken(t, "blackbird.chrome.title", title, theme.Colors.TextMuted)

	activeBorder, activeTitle := paneChromeColorsForTheme(theme, true)
	assertToken(t, "blackbird.chrome.activeBorder", activeBorder, theme.Colors.BorderActive)
	assertToken(t, "blackbird.chrome.activeTitle", activeTitle, theme.Colors.Accent)
}

func TestHighContrastTokenBaselinesForStatusAndChrome(t *testing.T) {
	theme := ResolveTheme(ThemeIDHighContrast)

	assertThemeTokenSet(t, "high-contrast", theme)

	assertToken(t, "high-contrast.ready", readinessLabelColor(theme, "READY"), theme.Colors.Info)
	assertToken(t, "high-contrast.waiting", runStatusColor(theme, execution.RunStatusWaitingUser), theme.Colors.Warning)
	assertToken(t, "high-contrast.done", readinessLabelColor(theme, "DONE"), theme.Colors.Success)
	assertToken(t, "high-contrast.defaultReadiness", readinessLabelColor(theme, "UNKNOWN"), theme.Colors.TextMuted)
	assertToken(t, "high-contrast.defaultRunStatus", runStatusColor(theme, execution.RunStatus("queued")), theme.Colors.TextMuted)

	border, title := paneChromeColorsForTheme(theme, false)
	assertToken(t, "high-contrast.chrome.border", border, theme.Colors.Border)
	assertToken(t, "high-contrast.chrome.title", title, theme.Colors.TextMuted)

	activeBorder, activeTitle := paneChromeColorsForTheme(theme, true)
	assertToken(t, "high-contrast.chrome.activeBorder", activeBorder, theme.Colors.BorderActive)
	assertToken(t, "high-contrast.chrome.activeTitle", activeTitle, theme.Colors.Accent)
}

func TestBuiltInThemesProvideDistinctSemanticPalettes(t *testing.T) {
	blackbird := ResolveTheme(ThemeIDBlackbird)
	highContrast := ResolveTheme(ThemeIDHighContrast)

	if blackbird.Colors.Accent == highContrast.Colors.Accent {
		t.Fatalf("blackbird and high-contrast accent colors should differ to make theme switching visible")
	}
	if blackbird.Colors.TextMuted == highContrast.Colors.TextMuted {
		t.Fatalf("blackbird and high-contrast muted text colors should differ")
	}
	if blackbird.Colors.Surface == highContrast.Colors.Surface {
		t.Fatalf("blackbird and high-contrast surface colors should differ")
	}
}

func assertToken(t *testing.T, name string, got lipgloss.Color, want lipgloss.Color) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}

func assertThemeTokenSet(t *testing.T, themeName string, theme Theme) {
	t.Helper()
	checks := []struct {
		name  string
		value lipgloss.Color
	}{
		{name: "TextPrimary", value: theme.Colors.TextPrimary},
		{name: "TextMuted", value: theme.Colors.TextMuted},
		{name: "TextOnAccent", value: theme.Colors.TextOnAccent},
		{name: "Accent", value: theme.Colors.Accent},
		{name: "Info", value: theme.Colors.Info},
		{name: "Success", value: theme.Colors.Success},
		{name: "Warning", value: theme.Colors.Warning},
		{name: "Danger", value: theme.Colors.Danger},
		{name: "Surface", value: theme.Colors.Surface},
		{name: "SurfaceMuted", value: theme.Colors.SurfaceMuted},
		{name: "Border", value: theme.Colors.Border},
		{name: "BorderActive", value: theme.Colors.BorderActive},
	}
	for _, tc := range checks {
		if tc.value == "" {
			t.Fatalf("%s.%s should be set", themeName, tc.name)
		}
	}
}
