package tui

import (
	"fmt"
	"strconv"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
)

// HandleSettingsKey handles key presses in the Settings view.
func HandleSettingsKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	state := m.settings
	key := msg.String()

	if key == "ctrl+c" {
		m = cancelRunningAction(m)
		return m, tea.Quit
	}

	if state.Editing {
		return handleSettingsEditKey(m, msg)
	}

	switch key {
	case "esc":
		m.viewMode = ViewModeHome
		return m, nil
	case "up", "k":
		if len(state.Options) == 0 {
			return m, nil
		}
		state.Selected--
		if state.Selected < 0 {
			state.Selected = 0
		}
		m.settings = state
		return m, nil
	case "down", "j":
		if len(state.Options) == 0 {
			return m, nil
		}
		state.Selected++
		if state.Selected >= len(state.Options) {
			state.Selected = len(state.Options) - 1
		}
		m.settings = state
		return m, nil
	case "left":
		col := int(selectedSettingsColumn(state)) - 1
		if col < 0 {
			col = 0
		}
		state.Column = SettingsColumn(col)
		m.settings = state
		return m, nil
	case "right":
		col := int(selectedSettingsColumn(state)) + 1
		if col >= settingsColumnCount() {
			col = settingsColumnCount() - 1
		}
		state.Column = SettingsColumn(col)
		m.settings = state
		return m, nil
	case "enter":
		return handleSettingsActivate(m, true)
	case " ":
		return handleSettingsActivate(m, false)
	case "delete", "backspace":
		return clearSettingsValue(m)
	default:
		return m, nil
	}
}

func handleSettingsEditKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	state := m.settings
	key := msg.String()

	switch key {
	case "esc":
		state.Editing = false
		state.EditValue = ""
		state.SaveErr = nil
		m.settings = state
		return m, nil
	case "enter":
		return commitSettingsEdit(m)
	case "delete":
		return clearSettingsValue(m)
	case "backspace":
		if state.EditValue == "" {
			return clearSettingsValue(m)
		}
		state.EditValue = dropLastRune(state.EditValue)
		state.SaveErr = nil
		m.settings = state
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			if !allDigits(msg.Runes) {
				state.SaveErr = fmt.Errorf("digits only")
				m.settings = state
				return m, nil
			}
			state.EditValue += string(msg.Runes)
			state.SaveErr = nil
			m.settings = state
			return m, nil
		}
		return m, nil
	}
}

func handleSettingsActivate(m Model, enter bool) (Model, tea.Cmd) {
	state := m.settings
	option, ok := selectedOption(state)
	if !ok {
		return m, nil
	}
	column := selectedSettingsColumn(state)
	if !settingsColumnEditable(state, column) {
		return m, nil
	}
	switch option.Type {
	case config.OptionTypeBool:
		return toggleSettingsBool(m, option, column)
	case config.OptionTypeInt:
		if !enter {
			return m, nil
		}
		state.Editing = true
		state.EditValue = ""
		state.SaveErr = nil
		raw := rawValueForColumn(state, option, column)
		if raw.Int != nil {
			state.EditValue = strconv.Itoa(*raw.Int)
		}
		m.settings = state
		return m, nil
	default:
		return m, nil
	}
}

func toggleSettingsBool(m Model, option config.OptionMetadata, column SettingsColumn) (Model, tea.Cmd) {
	state := m.settings
	if option.Type != config.OptionTypeBool {
		return m, nil
	}
	raw := rawValueForColumn(state, option, column)
	var current bool
	if raw.Bool != nil {
		current = *raw.Bool
	} else {
		applied := appliedOptionFor(state, option)
		if applied.Value.Bool != nil {
			current = *applied.Value.Bool
		}
	}
	next := !current
	value := config.RawOptionValue{Bool: &next}
	state.SaveErr = nil
	m.settings = state
	updated, err := saveSettingsValue(m, option, column, &value)
	if err != nil {
		state = updated.settings
		state.SaveErr = err
		updated.settings = state
		return updated, nil
	}
	return updated, nil
}

func commitSettingsEdit(m Model) (Model, tea.Cmd) {
	state := m.settings
	option, ok := selectedOption(state)
	if !ok {
		return m, nil
	}
	column := selectedSettingsColumn(state)
	if !settingsColumnEditable(state, column) || option.Type != config.OptionTypeInt {
		return m, nil
	}

	if state.EditValue == "" {
		return clearSettingsValue(m)
	}

	value, err := strconv.Atoi(state.EditValue)
	if err != nil {
		state.SaveErr = fmt.Errorf("invalid number")
		m.settings = state
		return m, nil
	}
	if option.Bounds != nil {
		if value < option.Bounds.Min || value > option.Bounds.Max {
			state.SaveErr = fmt.Errorf("value must be between %d and %d", option.Bounds.Min, option.Bounds.Max)
			m.settings = state
			return m, nil
		}
	}

	raw := config.RawOptionValue{Int: &value}
	state.SaveErr = nil
	m.settings = state
	updated, err := saveSettingsValue(m, option, column, &raw)
	if err != nil {
		state = updated.settings
		state.SaveErr = err
		state.Editing = false
		state.EditValue = ""
		updated.settings = state
		return updated, nil
	}
	updated.settings.Editing = false
	updated.settings.EditValue = ""
	updated.settings.SaveErr = nil
	return updated, nil
}

func clearSettingsValue(m Model) (Model, tea.Cmd) {
	state := m.settings
	option, ok := selectedOption(state)
	if !ok {
		return m, nil
	}
	column := selectedSettingsColumn(state)
	if !settingsColumnEditable(state, column) {
		return m, nil
	}

	state.SaveErr = nil
	state.Editing = false
	state.EditValue = ""
	m.settings = state
	updated, err := saveSettingsValue(m, option, column, nil)
	if err != nil {
		state = updated.settings
		state.SaveErr = err
		updated.settings = state
		return updated, nil
	}
	return updated, nil
}

func saveSettingsValue(m Model, option config.OptionMetadata, column SettingsColumn, value *config.RawOptionValue) (Model, error) {
	state := m.settings
	var path string
	var current map[string]config.RawOptionValue

	switch column {
	case SettingsColumnLocal:
		path = state.Resolution.Project.Path
		current = state.Resolution.Project.Values
	case SettingsColumnGlobal:
		path = state.Resolution.Global.Path
		current = state.Resolution.Global.Values
	default:
		return m, fmt.Errorf("unsupported settings column")
	}

	values := copyRawOptionValues(current)
	if value == nil {
		delete(values, option.KeyPath)
	} else {
		values[option.KeyPath] = *value
	}

	if err := config.SaveConfigValues(path, values); err != nil {
		return m, err
	}

	resolved, err := config.LoadConfig(state.ProjectRoot)
	if err != nil {
		return m, err
	}
	resolution, err := config.ResolveSettings(state.ProjectRoot)
	if err != nil {
		return m, err
	}

	state.Resolved = resolved
	state.Resolution = resolution
	state.Err = nil
	state.SaveErr = nil
	m.settings = state
	m.config = resolved
	return m, nil
}

func copyRawOptionValues(values map[string]config.RawOptionValue) map[string]config.RawOptionValue {
	copied := make(map[string]config.RawOptionValue, len(values))
	for key, value := range values {
		copied[key] = value
	}
	return copied
}

func allDigits(runes []rune) bool {
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
