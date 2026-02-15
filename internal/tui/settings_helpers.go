package tui

import (
	"sort"

	"github.com/jbonatakis/blackbird/internal/config"
)

const settingsThemeOptionKey = "tui.theme"

type SettingsColumn int

const (
	SettingsColumnOption SettingsColumn = iota
	SettingsColumnLocal
	SettingsColumnGlobal
	SettingsColumnDefault
	SettingsColumnApplied
)

func settingsColumnCount() int {
	return int(SettingsColumnApplied) + 1
}

func selectedSettingsColumn(state SettingsState) SettingsColumn {
	col := state.Column
	if col < SettingsColumnOption {
		return SettingsColumnOption
	}
	if col > SettingsColumnApplied {
		return SettingsColumnApplied
	}
	return col
}

func settingsColumnEditable(state SettingsState, col SettingsColumn) bool {
	switch col {
	case SettingsColumnLocal:
		return state.Resolution.Project.Available && state.Resolution.Project.Path != ""
	case SettingsColumnGlobal:
		return state.Resolution.Global.Available && state.Resolution.Global.Path != ""
	default:
		return false
	}
}

func rawValueForColumn(state SettingsState, option config.OptionMetadata, col SettingsColumn) config.RawOptionValue {
	if option.KeyPath == settingsThemeOptionKey && state.PendingTheme != nil && state.PendingTheme.Column == col {
		value := state.PendingTheme.Value
		return config.RawOptionValue{String: &value}
	}

	switch col {
	case SettingsColumnLocal:
		return state.Resolution.Project.Values[option.KeyPath]
	case SettingsColumnGlobal:
		return state.Resolution.Global.Values[option.KeyPath]
	default:
		return config.RawOptionValue{}
	}
}

func sortedCategoricalAllowedValues(option config.OptionMetadata) []string {
	if option.Type != config.OptionTypeCategorical || len(option.AllowedValues) == 0 {
		return nil
	}
	values := append([]string(nil), option.AllowedValues...)
	sort.Strings(values)
	return values
}
