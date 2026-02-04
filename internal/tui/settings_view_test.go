package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/jbonatakis/blackbird/internal/config"
)

func TestRenderSettingsTableLayout(t *testing.T) {
	opt1 := config.OptionMetadata{
		KeyPath:     "opt.one",
		DisplayName: "Option One",
		Type:        config.OptionTypeInt,
		DefaultInt:  5,
		Bounds:      &config.IntBounds{Min: 1, Max: 10},
		Description: "First option",
	}
	opt2 := config.OptionMetadata{
		KeyPath:     "opt.two",
		DisplayName: "Option Two",
		Type:        config.OptionTypeBool,
		DefaultBool: false,
		Description: "Second option",
	}

	state := SettingsState{
		Options:  []config.OptionMetadata{opt1, opt2},
		Selected: 0,
		Resolution: config.SettingsResolution{
			Project: config.SettingsLayer{
				Values: map[string]config.RawOptionValue{
					"opt.one": {Int: intPtr(7)},
				},
			},
			Global: config.SettingsLayer{
				Values: map[string]config.RawOptionValue{
					"opt.two": {Bool: boolPtr(true)},
				},
			},
			Applied: map[string]config.AppliedOption{
				"opt.one": {Value: config.RawOptionValue{Int: intPtr(7)}, Source: config.ConfigSourceLocal},
				"opt.two": {Value: config.RawOptionValue{Bool: boolPtr(true)}, Source: config.ConfigSourceGlobal},
			},
		},
	}

	out := renderSettingsTable(state)
	stripped := stripANSI(out)
	lines := strings.Split(stripped, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 table lines, got %d", len(lines))
	}

	assertContains(t, lines[0], "Option")
	assertContains(t, lines[0], "Local")
	assertContains(t, lines[0], "Global")
	assertContains(t, lines[0], "Default")
	assertContains(t, lines[0], "Applied")

	assertContains(t, stripped, "7 (local)")
	assertContains(t, stripped, "true (global)")

	row2 := findLineContaining(lines, "Option Two")
	expectedRow2 := "Option Two" + "  " + "  -  " + "  " + "true  " + "  " + "false  " + "  " + "true (global)"
	if row2 != expectedRow2 {
		t.Fatalf("expected row2 %q, got %q", expectedRow2, row2)
	}

	highlighted := settingsHighlightStyle().Render("7    ")
	if !strings.Contains(out, highlighted) {
		t.Fatalf("expected highlighted local cell in output, got %q", out)
	}
}

func TestRenderSettingsViewFooter(t *testing.T) {
	options := []config.OptionMetadata{
		{
			KeyPath:     "opt.one",
			DisplayName: "Run Refresh",
			Type:        config.OptionTypeInt,
			DefaultInt:  5,
			Bounds:      &config.IntBounds{Min: 1, Max: 10},
			Description: "Run refresh in seconds",
		},
		{
			KeyPath:     "opt.two",
			DisplayName: "Stop After",
			Type:        config.OptionTypeBool,
			DefaultBool: false,
			Description: "Pause execution",
		},
	}

	state := SettingsState{
		Options:  options,
		Selected: 1,
		Resolution: config.SettingsResolution{
			Project: config.SettingsLayer{
				Available: true,
				Path:      "/project/.blackbird/config.json",
				Values:    map[string]config.RawOptionValue{},
			},
			Global: config.SettingsLayer{
				Available: false,
				Path:      "",
				Values:    map[string]config.RawOptionValue{},
			},
			LayerWarnings: []config.LayerWarning{
				{Source: config.ConfigSourceGlobal, Kind: config.LayerWarningInvalidJSON},
			},
			OptionWarnings: []config.OptionWarning{
				{
					Source:     config.ConfigSourceLocal,
					KeyPath:    "opt.one",
					Kind:       config.OptionWarningOutOfRange,
					ClampedInt: intPtr(10),
				},
			},
		},
		Err: errors.New("boom"),
	}

	model := Model{
		settings:     state,
		viewMode:     ViewModeSettings,
		windowWidth:  0,
		windowHeight: 0,
	}
	out := RenderSettingsView(model)

	assertContains(t, out, "Local > Global > Default")
	assertContains(t, out, "Local: /project/.blackbird/config.json")
	assertContains(t, out, "Global: N/A")

	assertContains(t, out, "Selected:")
	assertContains(t, out, "Stop After")
	assertContains(t, out, "Pause execution")
	assertContains(t, out, "type: bool")

	assertContains(t, out, "Settings load warning: boom")
	assertContains(t, out, "global config warning: invalid_json")
	assertContains(t, out, "local Run Refresh warning: out_of_range")
	assertContains(t, out, "clamped to 10")
}

func findLineContaining(lines []string, needle string) string {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(value string) string {
	return ansiRegexp.ReplaceAllString(value, "")
}
