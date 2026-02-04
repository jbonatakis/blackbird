package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/config"
)

func RenderSettingsView(m Model) string {
	state := m.settings
	titleStyle := settingsTitleStyle()
	mutedStyle := settingsMutedStyle()

	lines := []string{
		titleStyle.Render("Settings"),
		mutedStyle.Render("Local > Global > Default"),
	}

	localPath := "N/A"
	if state.Resolution.Project.Available && state.Resolution.Project.Path != "" {
		localPath = state.Resolution.Project.Path
	}
	globalPath := "N/A"
	if state.Resolution.Global.Available && state.Resolution.Global.Path != "" {
		globalPath = state.Resolution.Global.Path
	}

	lines = append(lines,
		fmt.Sprintf("Local: %s", localPath),
		fmt.Sprintf("Global: %s", globalPath),
		"",
		renderSettingsTable(state),
	)

	footer := renderSettingsFooter(state)
	if len(footer) > 0 {
		lines = append(lines, "")
		lines = append(lines, footer...)
	}

	content := strings.Join(lines, "\n")
	if m.windowWidth <= 0 || m.windowHeight <= 0 {
		return content
	}
	return lipgloss.Place(m.windowWidth, m.windowHeight, lipgloss.Left, lipgloss.Top, content)
}

func renderSettingsTable(state SettingsState) string {
	headerStyle := settingsHeaderStyle()
	rows := [][]tableCell{{
		{Text: "Option", Align: alignLeft, Style: headerStyle},
		{Text: "Local", Align: alignLeft, Style: headerStyle},
		{Text: "Global", Align: alignLeft, Style: headerStyle},
		{Text: "Default", Align: alignLeft, Style: headerStyle},
		{Text: "Applied", Align: alignLeft, Style: headerStyle},
	}}
	for idx, option := range state.Options {
		rows = append(rows, settingsRow(state, option, idx == selectedSettingsIndex(state)))
	}
	return renderTable(rows)
}

func settingsRow(state SettingsState, option config.OptionMetadata, selected bool) []tableCell {
	selectedColumn := selectedSettingsColumn(state)
	editing := state.Editing && selected && settingsColumnEditable(state, selectedColumn)

	localValue := formatRawOptionValue(state.Resolution.Project.Values[option.KeyPath])
	globalValue := formatRawOptionValue(state.Resolution.Global.Values[option.KeyPath])
	defaultValue := formatRawOptionValue(settingsDefaultOptionValue(option))
	applied := appliedOptionFor(state, option)
	appliedValue := formatAppliedValue(applied)

	optionCell := tableCell{Text: option.DisplayName, Align: alignLeft}
	if selected {
		optionCell.Style = settingsSelectedStyle()
	}

	localCell := valueCell(localValue)
	if editing && selectedColumn == SettingsColumnLocal {
		localCell = editValueCell(state.EditValue)
	}
	globalCell := valueCell(globalValue)
	if editing && selectedColumn == SettingsColumnGlobal {
		globalCell = editValueCell(state.EditValue)
	}
	defaultCell := valueCell(defaultValue)
	appliedCell := valueCell(appliedValue)

	switch applied.Source {
	case config.ConfigSourceLocal:
		if !(editing && selectedColumn == SettingsColumnLocal) {
			localCell.Style = settingsHighlightStyle()
		}
	case config.ConfigSourceGlobal:
		if !(editing && selectedColumn == SettingsColumnGlobal) {
			globalCell.Style = settingsHighlightStyle()
		}
	case config.ConfigSourceDefault:
		if !(editing && selectedColumn == SettingsColumnDefault) {
			defaultCell.Style = settingsHighlightStyle()
		}
	}

	return []tableCell{
		applySettingsCellStyle(optionCell, selected && selectedColumn == SettingsColumnOption, editing && selectedColumn == SettingsColumnOption),
		applySettingsCellStyle(localCell, selected && selectedColumn == SettingsColumnLocal, editing && selectedColumn == SettingsColumnLocal),
		applySettingsCellStyle(globalCell, selected && selectedColumn == SettingsColumnGlobal, editing && selectedColumn == SettingsColumnGlobal),
		applySettingsCellStyle(defaultCell, selected && selectedColumn == SettingsColumnDefault, editing && selectedColumn == SettingsColumnDefault),
		applySettingsCellStyle(appliedCell, selected && selectedColumn == SettingsColumnApplied, editing && selectedColumn == SettingsColumnApplied),
	}
}

func appliedOptionFor(state SettingsState, option config.OptionMetadata) config.AppliedOption {
	applied, ok := state.Resolution.Applied[option.KeyPath]
	if ok {
		return applied
	}
	return config.AppliedOption{Value: settingsDefaultOptionValue(option), Source: config.ConfigSourceDefault}
}

func formatAppliedValue(applied config.AppliedOption) string {
	value := formatRawOptionValue(applied.Value)
	if value == "-" {
		return value
	}
	return fmt.Sprintf("%s (%s)", value, applied.Source)
}

func formatRawOptionValue(value config.RawOptionValue) string {
	switch {
	case value.Int != nil:
		return fmt.Sprintf("%d", *value.Int)
	case value.Bool != nil:
		if *value.Bool {
			return "true"
		}
		return "false"
	default:
		return "-"
	}
}

func renderTable(rows [][]tableCell) string {
	if len(rows) == 0 {
		return ""
	}

	widths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			w := lipgloss.Width(cell.Text)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, renderTableRow(row, widths))
	}
	return strings.Join(lines, "\n")
}

func renderTableRow(row []tableCell, widths []int) string {
	parts := make([]string, len(row))
	for i, cell := range row {
		parts[i] = renderTableCell(cell, widths[i])
	}
	return strings.Join(parts, "  ")
}

func renderTableCell(cell tableCell, width int) string {
	aligned := alignText(cell.Text, width, cell.Align)
	return cell.Style.Render(aligned)
}

func alignText(value string, width int, align cellAlign) string {
	padding := width - lipgloss.Width(value)
	if padding <= 0 {
		return value
	}
	switch align {
	case alignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
	default:
		return value + strings.Repeat(" ", padding)
	}
}

func valueCell(value string) tableCell {
	align := alignLeft
	style := lipgloss.Style{}
	if value == "-" {
		align = alignCenter
		style = settingsMutedStyle()
	}
	return tableCell{Text: value, Align: align, Style: style}
}

func editValueCell(value string) tableCell {
	return tableCell{Text: value, Align: alignLeft, Style: settingsEditStyle()}
}

func renderSettingsFooter(state SettingsState) []string {
	lines := []string{}
	option, ok := selectedOption(state)
	if ok {
		labelStyle := settingsSelectedLabelStyle()
		desc := formatOptionDescription(option)
		lines = append(lines, fmt.Sprintf("%s %s — %s", labelStyle.Render("Selected:"), option.DisplayName, desc))
	}

	warnStyle := settingsWarnStyle()
	for _, warning := range settingsWarningLines(state) {
		lines = append(lines, warnStyle.Render(warning))
	}
	return lines
}

func settingsWarningLines(state SettingsState) []string {
	lines := []string{}
	if state.Err != nil {
		lines = append(lines, fmt.Sprintf("Settings load warning: %v", state.Err))
	}
	if state.SaveErr != nil {
		lines = append(lines, fmt.Sprintf("Settings error: %v", state.SaveErr))
	}
	lines = append(lines, layerWarningLines(state.Resolution.LayerWarnings)...)
	lines = append(lines, optionWarningLines(state.Resolution.OptionWarnings, state.Options)...)
	return lines
}

func layerWarningLines(warnings []config.LayerWarning) []string {
	lines := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		lines = append(lines, fmt.Sprintf("%s config warning: %s", warning.Source, warning.Kind))
	}
	return lines
}

func optionWarningLines(warnings []config.OptionWarning, options []config.OptionMetadata) []string {
	if len(warnings) == 0 {
		return nil
	}
	lookup := map[string]string{}
	for _, option := range options {
		lookup[option.KeyPath] = option.DisplayName
	}
	lines := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		name := lookup[warning.KeyPath]
		if name == "" {
			name = warning.KeyPath
		}
		line := fmt.Sprintf("%s %s warning: %s", warning.Source, name, warning.Kind)
		if warning.ClampedInt != nil {
			line = fmt.Sprintf("%s (clamped to %d)", line, *warning.ClampedInt)
		}
		lines = append(lines, line)
	}
	return lines
}

func formatOptionDescription(option config.OptionMetadata) string {
	desc := option.Description
	details := []string{fmt.Sprintf("type: %s", option.Type)}
	if option.Type == config.OptionTypeInt && option.Bounds != nil {
		details = append(details, fmt.Sprintf("bounds: %d-%d", option.Bounds.Min, option.Bounds.Max))
	}
	if len(details) == 0 {
		return desc
	}
	if desc == "" {
		return fmt.Sprintf("(%s)", strings.Join(details, ", "))
	}
	return fmt.Sprintf("%s (%s)", desc, strings.Join(details, ", "))
}

func selectedOption(state SettingsState) (config.OptionMetadata, bool) {
	if len(state.Options) == 0 {
		return config.OptionMetadata{}, false
	}
	idx := selectedSettingsIndex(state)
	if idx < 0 || idx >= len(state.Options) {
		idx = 0
	}
	return state.Options[idx], true
}

func selectedSettingsIndex(state SettingsState) int {
	if len(state.Options) == 0 {
		return 0
	}
	idx := state.Selected
	if idx < 0 {
		return 0
	}
	if idx >= len(state.Options) {
		return len(state.Options) - 1
	}
	return idx
}

type cellAlign int

const (
	alignLeft cellAlign = iota
	alignCenter
)

type tableCell struct {
	Text  string
	Align cellAlign
	Style lipgloss.Style
}

func settingsTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
}

func settingsHeaderStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func settingsMutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
}

func settingsWarnStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
}

func settingsHighlightStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Reverse(true)
}

func settingsSelectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func settingsSelectedLabelStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func settingsEditStyle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true)
}

func applySettingsCellStyle(cell tableCell, selected bool, editing bool) tableCell {
	if editing {
		cell.Style = cell.Style.Copy().Underline(true).Bold(true)
		return cell
	}
	if selected {
		cell.Style = cell.Style.Copy().Underline(true)
	}
	return cell
}
