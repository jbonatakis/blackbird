package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
)

func TestSettingsNavigationMovesRowAndColumn(t *testing.T) {
	state := SettingsState{
		Options:  config.OptionRegistry(),
		Selected: 0,
		Column:   SettingsColumnLocal,
		Resolution: config.SettingsResolution{
			Project: config.SettingsLayer{Available: true, Path: "/tmp/local", Values: map[string]config.RawOptionValue{}},
			Global:  config.SettingsLayer{Available: true, Path: "/tmp/global", Values: map[string]config.RawOptionValue{}},
		},
	}

	model := Model{
		viewMode: ViewModeSettings,
		settings: state,
	}

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeyDown})
	if updated.settings.Selected != 1 {
		t.Fatalf("expected row selection 1, got %d", updated.settings.Selected)
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRight})
	if updated.settings.Column != SettingsColumnGlobal {
		t.Fatalf("expected column global, got %d", updated.settings.Column)
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyLeft})
	if updated.settings.Column != SettingsColumnLocal {
		t.Fatalf("expected column local, got %d", updated.settings.Column)
	}
}

func TestSettingsBoolToggleAndClearAutosave(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "execution.stopAfterEachTask")
	if idx < 0 {
		t.Fatalf("missing bool option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnLocal

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeySpace})
	cfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config: %v", err)
	}
	if !present || cfg.Execution == nil || cfg.Execution.StopAfterEachTask == nil || !*cfg.Execution.StopAfterEachTask {
		t.Fatalf("expected bool value true in local config after toggle")
	}
	applied := updated.settings.Resolution.Applied["execution.stopAfterEachTask"]
	if applied.Source != config.ConfigSourceLocal {
		t.Fatalf("expected applied source local, got %s", applied.Source)
	}
	if updated.config.Execution.StopAfterEachTask != true {
		t.Fatalf("expected model config updated to true")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyDelete})
	_, present, err = config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config after clear: %v", err)
	}
	if present {
		t.Fatalf("expected config file removed after clearing last value")
	}
	applied = updated.settings.Resolution.Applied["execution.stopAfterEachTask"]
	if applied.Source != config.ConfigSourceDefault {
		t.Fatalf("expected applied source default after clear, got %s", applied.Source)
	}
}

func TestSettingsThemeCycleDebouncesSaveAndAppliesActiveThemeAfterIdle(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
		theme:       resolveActiveTheme(config.DefaultResolvedConfig()),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "tui.theme")
	if idx < 0 {
		t.Fatalf("missing theme option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnLocal

	var scheduled []time.Duration
	originalDebounce := settingsThemeDebounceCmd
	settingsThemeDebounceCmd = func(delay time.Duration, token uint64) tea.Cmd {
		scheduled = append(scheduled, delay)
		return func() tea.Msg {
			return settingsThemeDebounceMsg{Token: token}
		}
	}
	t.Cleanup(func() {
		settingsThemeDebounceCmd = originalDebounce
	})

	updated, cmd := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeySpace})
	if cmd == nil {
		t.Fatalf("expected debounce command")
	}
	if len(scheduled) != 1 || scheduled[0] != settingsThemeDebounceDelay {
		t.Fatalf("expected debounce scheduled once at %s, got %#v", settingsThemeDebounceDelay, scheduled)
	}
	if updated.config.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected config theme to remain %q before debounce flush, got %q", config.ThemeIDBlackbird, updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme to remain %q before debounce flush, got %q", ThemeIDBlackbird, updated.theme.ID)
	}
	raw := rawValueForColumn(updated.settings, updated.settings.Options[idx], SettingsColumnLocal)
	if raw.String == nil || *raw.String != config.ThemeIDHighContrast {
		t.Fatalf("expected pending local theme %q, got %+v", config.ThemeIDHighContrast, raw)
	}

	cfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config before debounce flush: %v", err)
	}
	if present {
		t.Fatalf("expected no config write before debounce flush, got %#v", cfg)
	}

	updatedModel, _ := updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.config.TUI.Theme != config.ThemeIDHighContrast {
		t.Fatalf("expected config theme %q after debounce flush, got %q", config.ThemeIDHighContrast, updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q after debounce flush, got %q", ThemeIDHighContrast, updated.theme.ID)
	}

	cfg, present, err = config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config after debounce flush: %v", err)
	}
	if !present || cfg.TUI == nil || cfg.TUI.Theme == nil || *cfg.TUI.Theme != config.ThemeIDHighContrast {
		t.Fatalf("expected local theme value %q in config after debounce flush, got %#v", config.ThemeIDHighContrast, cfg.TUI)
	}
}

func TestSettingsThemeCycleRapidInputDebouncesToSingleWrite(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
		theme:       resolveActiveTheme(config.DefaultResolvedConfig()),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "tui.theme")
	if idx < 0 {
		t.Fatalf("missing theme option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnLocal

	var scheduled []time.Duration
	originalDebounce := settingsThemeDebounceCmd
	settingsThemeDebounceCmd = func(delay time.Duration, token uint64) tea.Cmd {
		scheduled = append(scheduled, delay)
		return func() tea.Msg {
			return settingsThemeDebounceMsg{Token: token}
		}
	}
	t.Cleanup(func() {
		settingsThemeDebounceCmd = originalDebounce
	})

	writeCount := 0
	originalSave := saveConfigValuesForSettings
	saveConfigValuesForSettings = func(path string, values map[string]config.RawOptionValue) error {
		writeCount++
		return config.SaveConfigValues(path, values)
	}
	t.Cleanup(func() {
		saveConfigValuesForSettings = originalSave
	})

	updated, cmdFirst := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeySpace})
	if cmdFirst == nil {
		t.Fatalf("expected first debounce command")
	}
	updated, cmdSecond := HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeySpace})
	if cmdSecond == nil {
		t.Fatalf("expected second debounce command")
	}
	if len(scheduled) != 2 || scheduled[0] != settingsThemeDebounceDelay || scheduled[1] != settingsThemeDebounceDelay {
		t.Fatalf("expected two debounce schedules at %s, got %#v", settingsThemeDebounceDelay, scheduled)
	}
	if writeCount != 0 {
		t.Fatalf("expected no writes before debounce flush, got %d", writeCount)
	}
	if updated.config.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected config theme unchanged before flush, got %q", updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme unchanged before flush, got %q", updated.theme.ID)
	}
	raw := rawValueForColumn(updated.settings, updated.settings.Options[idx], SettingsColumnLocal)
	if raw.String == nil || *raw.String != config.ThemeIDBlackbird {
		t.Fatalf("expected pending theme to cycle back to %q, got %+v", config.ThemeIDBlackbird, raw)
	}
	if _, present, err := config.LoadProjectConfig(projectRoot); err != nil {
		t.Fatalf("load project config before flush: %v", err)
	} else if present {
		t.Fatalf("expected no persisted theme before flush")
	}

	updatedModel, _ := updated.Update(cmdFirst())
	updated = updatedModel.(Model)
	if writeCount != 0 {
		t.Fatalf("expected stale debounce message to be ignored, got %d writes", writeCount)
	}
	if _, present, err := config.LoadProjectConfig(projectRoot); err != nil {
		t.Fatalf("load project config after stale flush: %v", err)
	} else if present {
		t.Fatalf("expected no persisted theme after stale flush")
	}

	updatedModel, _ = updated.Update(cmdSecond())
	updated = updatedModel.(Model)
	if writeCount != 1 {
		t.Fatalf("expected exactly one write after final debounce flush, got %d", writeCount)
	}
	if updated.config.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected config theme %q after final flush, got %q", config.ThemeIDBlackbird, updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme %q after final flush, got %q", ThemeIDBlackbird, updated.theme.ID)
	}

	cfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config after final flush: %v", err)
	}
	if !present || cfg.TUI == nil || cfg.TUI.Theme == nil || *cfg.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected persisted local theme %q after final flush, got %#v", config.ThemeIDBlackbird, cfg.TUI)
	}
}

func TestCategoricalHelpersCycleAlphabetically(t *testing.T) {
	option := config.OptionMetadata{
		Type:          config.OptionTypeCategorical,
		AllowedValues: []string{"zulu", "alpha", "mango"},
	}

	allowed := sortedCategoricalAllowedValues(option)
	if len(allowed) != 3 {
		t.Fatalf("expected 3 allowed values, got %d", len(allowed))
	}
	if allowed[0] != "alpha" || allowed[1] != "mango" || allowed[2] != "zulu" {
		t.Fatalf("expected alphabetical sort, got %#v", allowed)
	}

	next, ok := nextCategoricalValue("alpha", allowed)
	if !ok || next != "mango" {
		t.Fatalf("expected next value mango, got %q (ok=%v)", next, ok)
	}
	next, ok = nextCategoricalValue("zulu", allowed)
	if !ok || next != "alpha" {
		t.Fatalf("expected wrap-around to alpha, got %q (ok=%v)", next, ok)
	}
	next, ok = nextCategoricalValue("unknown", allowed)
	if !ok || next != "alpha" {
		t.Fatalf("expected unknown current to start at alpha, got %q (ok=%v)", next, ok)
	}
}

func TestSettingsCategoricalEnterSpaceCycleAndClearFallbackAcrossLayers(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	const key = "tui.theme"

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
		theme:       resolveActiveTheme(config.DefaultResolvedConfig()),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, key)
	if idx < 0 {
		t.Fatalf("missing categorical option %q", key)
	}
	model.settings.Selected = idx

	model.settings.Column = SettingsColumnGlobal
	updated, cmd := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.settings.Editing {
		t.Fatalf("categorical enter should cycle; edit mode must remain off")
	}
	if cmd == nil {
		t.Fatalf("expected debounce command for categorical theme enter-cycle")
	}
	updatedModel, _ := updated.Update(cmd())
	updated = updatedModel.(Model)
	globalCfg, present, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load global config: %v", err)
	}
	if !present || globalCfg.TUI == nil || globalCfg.TUI.Theme == nil || *globalCfg.TUI.Theme != config.ThemeIDHighContrast {
		t.Fatalf("expected global theme %q after enter-cycle", config.ThemeIDHighContrast)
	}
	applied := updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.String == nil || *applied.Value.String != config.ThemeIDHighContrast {
		t.Fatalf("expected applied global high-contrast after enter-cycle, got %+v", applied)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q after global cycle, got %q", ThemeIDHighContrast, updated.theme.ID)
	}

	updated.settings.Column = SettingsColumnLocal
	updated, cmd = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeySpace})
	if updated.settings.Editing {
		t.Fatalf("categorical space should cycle; edit mode must remain off")
	}
	if cmd == nil {
		t.Fatalf("expected debounce command for categorical theme space-cycle")
	}
	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	localCfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load local config: %v", err)
	}
	if !present || localCfg.TUI == nil || localCfg.TUI.Theme == nil || *localCfg.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected local theme %q after space-cycle", config.ThemeIDBlackbird)
	}
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceLocal || applied.Value.String == nil || *applied.Value.String != config.ThemeIDBlackbird {
		t.Fatalf("expected applied local blackbird after local cycle, got %+v", applied)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme %q after local cycle, got %q", ThemeIDBlackbird, updated.theme.ID)
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyBackspace})
	if _, present, err := config.LoadProjectConfig(projectRoot); err != nil {
		t.Fatalf("load local config after clear: %v", err)
	} else if present {
		t.Fatalf("expected local config removed after clearing categorical value")
	}
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.String == nil || *applied.Value.String != config.ThemeIDHighContrast {
		t.Fatalf("expected applied fallback to global high-contrast after local clear, got %+v", applied)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q after local clear fallback, got %q", ThemeIDHighContrast, updated.theme.ID)
	}

	updated.settings.Column = SettingsColumnGlobal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyDelete})
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceDefault || applied.Value.String == nil || *applied.Value.String != config.ThemeIDBlackbird {
		t.Fatalf("expected applied fallback to default blackbird after global clear, got %+v", applied)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme %q after global clear fallback, got %q", ThemeIDBlackbird, updated.theme.ID)
	}
	if _, present, err := config.LoadGlobalConfig(); err != nil {
		t.Fatalf("load global config after clear: %v", err)
	} else if present {
		t.Fatalf("expected global config removed after clearing categorical value")
	}
}

func TestSaveSettingsThemeUnknownValueFallsBackToBlackbird(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
		theme:       resolveActiveTheme(config.DefaultResolvedConfig()),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "tui.theme")
	if idx < 0 {
		t.Fatalf("missing theme option")
	}
	option := model.settings.Options[idx]

	invalid := "unknown-theme-id"
	updated, err := saveSettingsValue(model, option, SettingsColumnLocal, &config.RawOptionValue{String: &invalid})
	if err != nil {
		t.Fatalf("save settings value: %v", err)
	}

	if updated.config.TUI.Theme != config.ThemeIDBlackbird {
		t.Fatalf("expected resolved config theme fallback %q, got %q", config.ThemeIDBlackbird, updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected active theme fallback %q, got %q", ThemeIDBlackbird, updated.theme.ID)
	}

	applied := updated.settings.Resolution.Applied["tui.theme"]
	if applied.Source != config.ConfigSourceDefault {
		t.Fatalf("expected applied source default for invalid theme, got %s", applied.Source)
	}
	if applied.Value.String == nil || *applied.Value.String != config.ThemeIDBlackbird {
		t.Fatalf("expected applied fallback theme %q, got %#v", config.ThemeIDBlackbird, applied.Value.String)
	}
}

func TestSettingsParentReviewTogglePersistenceAcrossLayers(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	const key = "execution.parentReviewEnabled"

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, key)
	if idx < 0 {
		t.Fatalf("missing parent-review bool option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnGlobal

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeySpace})
	globalCfg, present, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load global config: %v", err)
	}
	if !present || globalCfg.Execution == nil || globalCfg.Execution.ParentReviewEnabled == nil || !*globalCfg.Execution.ParentReviewEnabled {
		t.Fatalf("expected global parent-review value true after toggle")
	}
	applied := updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.Bool == nil || !*applied.Value.Bool {
		t.Fatalf("expected applied value true from global source, got %+v", applied)
	}
	if !updated.config.Execution.ParentReviewEnabled {
		t.Fatalf("expected model config parent-review value true after global toggle")
	}

	reopened := NewSettingsState(projectRoot, updated.config)
	applied = reopened.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.Bool == nil || !*applied.Value.Bool {
		t.Fatalf("expected reopened settings to apply global parent-review true, got %+v", applied)
	}

	updated.settings = reopened
	updated.settings.Selected = idx
	updated.settings.Column = SettingsColumnLocal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeySpace})

	localCfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config: %v", err)
	}
	if !present || localCfg.Execution == nil || localCfg.Execution.ParentReviewEnabled == nil || *localCfg.Execution.ParentReviewEnabled {
		t.Fatalf("expected local parent-review value false after local toggle")
	}
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceLocal || applied.Value.Bool == nil || *applied.Value.Bool {
		t.Fatalf("expected applied value false from local source, got %+v", applied)
	}
	if updated.config.Execution.ParentReviewEnabled {
		t.Fatalf("expected model config parent-review value false after local toggle")
	}

	reopened = NewSettingsState(projectRoot, updated.config)
	applied = reopened.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceLocal || applied.Value.Bool == nil || *applied.Value.Bool {
		t.Fatalf("expected reopened settings to apply local parent-review false, got %+v", applied)
	}

	updated.settings = reopened
	updated.settings.Selected = idx
	updated.settings.Column = SettingsColumnLocal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyDelete})
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.Bool == nil || !*applied.Value.Bool {
		t.Fatalf("expected applied value to fall back to global true after local clear, got %+v", applied)
	}

	updated.settings.Column = SettingsColumnGlobal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyDelete})
	applied = updated.settings.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceDefault || applied.Value.Bool == nil || *applied.Value.Bool {
		t.Fatalf("expected applied value to fall back to default false after global clear, got %+v", applied)
	}

	_, present, err = config.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load global config after clear: %v", err)
	}
	if present {
		t.Fatalf("expected global config file removed after clearing last value")
	}

	reopened = NewSettingsState(projectRoot, updated.config)
	applied = reopened.Resolution.Applied[key]
	if applied.Source != config.ConfigSourceDefault || applied.Value.Bool == nil || *applied.Value.Bool {
		t.Fatalf("expected reopened settings to apply default parent-review false, got %+v", applied)
	}
}

func TestSettingsIntEditValidationAndAutosave(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "tui.runDataRefreshIntervalSeconds")
	if idx < 0 {
		t.Fatalf("missing int option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnLocal

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.settings.Editing {
		t.Fatalf("expected edit mode to start")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if updated.settings.EditValue != "" {
		t.Fatalf("expected non-digit input to be ignored")
	}
	if updated.settings.SaveErr == nil {
		t.Fatalf("expected error on non-digit input")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4', '0', '0'}})
	if updated.settings.EditValue != "400" {
		t.Fatalf("expected edit value 400, got %q", updated.settings.EditValue)
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.settings.Editing {
		t.Fatalf("expected edit mode to remain on out-of-range commit")
	}
	if updated.settings.SaveErr == nil {
		t.Fatalf("expected error on out-of-range commit")
	}
	_, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config: %v", err)
	}
	if present {
		t.Fatalf("expected no config write on out-of-range value")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyBackspace})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyBackspace})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyBackspace})
	if updated.settings.EditValue != "" {
		t.Fatalf("expected edit value cleared after backspace")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1', '0'}})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.settings.Editing {
		t.Fatalf("expected edit mode to end after commit")
	}
	cfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config after save: %v", err)
	}
	if !present || cfg.TUI == nil || cfg.TUI.RunDataRefreshIntervalSeconds == nil || *cfg.TUI.RunDataRefreshIntervalSeconds != 10 {
		t.Fatalf("expected config value 10 after commit")
	}
	if updated.config.TUI.RunDataRefreshIntervalSeconds != 10 {
		t.Fatalf("expected model config updated to 10")
	}
	applied := updated.settings.Resolution.Applied["tui.runDataRefreshIntervalSeconds"]
	if applied.Source != config.ConfigSourceLocal || applied.Value.Int == nil || *applied.Value.Int != 10 {
		t.Fatalf("expected applied value 10 from local source, got %+v", applied)
	}
}

func TestSettingsPlanningIntEditAcrossGlobalAndLocalLayers(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, "planning.maxPlanAutoRefinePasses")
	if idx < 0 {
		t.Fatalf("missing planning int option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnGlobal

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.settings.Editing {
		t.Fatalf("expected global edit mode to start")
	}
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.settings.Editing {
		t.Fatalf("expected global edit mode to end after commit")
	}

	globalCfg, present, err := config.LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load global config: %v", err)
	}
	if !present || globalCfg.Planning == nil || globalCfg.Planning.MaxPlanAutoRefinePasses == nil || *globalCfg.Planning.MaxPlanAutoRefinePasses != 2 {
		t.Fatalf("expected global planning value 2, got %#v", globalCfg.Planning)
	}
	applied := updated.settings.Resolution.Applied["planning.maxPlanAutoRefinePasses"]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.Int == nil || *applied.Value.Int != 2 {
		t.Fatalf("expected applied value 2 from global source, got %+v", applied)
	}
	if updated.config.Planning.MaxPlanAutoRefinePasses != 2 {
		t.Fatalf("expected model planning value 2 after global save")
	}

	updated.settings.Column = SettingsColumnLocal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if !updated.settings.Editing {
		t.Fatalf("expected local edit mode to start")
	}
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.settings.Editing {
		t.Fatalf("expected local edit mode to end after commit")
	}

	localCfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load local config: %v", err)
	}
	if !present || localCfg.Planning == nil || localCfg.Planning.MaxPlanAutoRefinePasses == nil || *localCfg.Planning.MaxPlanAutoRefinePasses != 3 {
		t.Fatalf("expected local planning value 3, got %#v", localCfg.Planning)
	}
	applied = updated.settings.Resolution.Applied["planning.maxPlanAutoRefinePasses"]
	if applied.Source != config.ConfigSourceLocal || applied.Value.Int == nil || *applied.Value.Int != 3 {
		t.Fatalf("expected applied value 3 from local source, got %+v", applied)
	}
	if updated.config.Planning.MaxPlanAutoRefinePasses != 3 {
		t.Fatalf("expected model planning value 3 after local save")
	}

	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyDelete})
	_, present, err = config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load local config after clear: %v", err)
	}
	if present {
		t.Fatalf("expected local config removed after clearing planning value")
	}
	applied = updated.settings.Resolution.Applied["planning.maxPlanAutoRefinePasses"]
	if applied.Source != config.ConfigSourceGlobal || applied.Value.Int == nil || *applied.Value.Int != 2 {
		t.Fatalf("expected applied value to fall back to global 2, got %+v", applied)
	}
}

func TestSettingsSaveFailureKeepsPriorValue(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	key := "tui.runDataRefreshIntervalSeconds"
	initial := 5
	configPath := filepath.Join(projectRoot, ".blackbird", "config.json")
	if err := config.SaveConfigValues(configPath, map[string]config.RawOptionValue{
		key: {Int: &initial},
	}); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	idx := optionIndex(model.settings.Options, key)
	if idx < 0 {
		t.Fatalf("missing int option")
	}
	model.settings.Selected = idx
	model.settings.Column = SettingsColumnLocal

	badRoot := filepath.Join(projectRoot, "notadir")
	if err := os.WriteFile(badRoot, []byte("x"), 0o644); err != nil {
		t.Fatalf("write bad root: %v", err)
	}
	model.settings.Resolution.Project.Path = filepath.Join(badRoot, "config.json")

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeyEnter})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'7'}})
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})

	if updated.settings.SaveErr == nil {
		t.Fatalf("expected save error")
	}
	raw := updated.settings.Resolution.Project.Values[key]
	if raw.Int == nil || *raw.Int != initial {
		t.Fatalf("expected prior value preserved in state")
	}

	cfg, present, err := config.LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("load project config after failed save: %v", err)
	}
	if !present || cfg.TUI == nil || cfg.TUI.RunDataRefreshIntervalSeconds == nil || *cfg.TUI.RunDataRefreshIntervalSeconds != initial {
		t.Fatalf("expected config value unchanged after failed save")
	}
}

func TestSettingsGlobalUnavailableDisablesColumn(t *testing.T) {
	projectRoot := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return "", nil
	})
	defer restoreHome()

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.settings = NewSettingsState(projectRoot, model.config)
	if model.settings.Resolution.Global.Available {
		t.Fatalf("expected global config unavailable")
	}
	if model.settings.Resolution.Global.Path != "" {
		t.Fatalf("expected empty global config path, got %q", model.settings.Resolution.Global.Path)
	}

	boolIdx := optionIndex(model.settings.Options, "execution.stopAfterEachTask")
	if boolIdx < 0 {
		t.Fatalf("missing bool option")
	}
	model.settings.Selected = boolIdx
	model.settings.Column = SettingsColumnGlobal

	updated, _ := HandleSettingsKey(model, tea.KeyMsg{Type: tea.KeySpace})
	if updated.settings.Editing {
		t.Fatalf("expected edit mode to remain off when global is unavailable")
	}
	if updated.settings.SaveErr != nil {
		t.Fatalf("expected no save error, got %v", updated.settings.SaveErr)
	}
	applied := updated.settings.Resolution.Applied["execution.stopAfterEachTask"]
	if applied.Source != config.ConfigSourceDefault {
		t.Fatalf("expected applied source default, got %s", applied.Source)
	}
	if _, present, err := config.LoadProjectConfig(projectRoot); err != nil {
		t.Fatalf("load project config: %v", err)
	} else if present {
		t.Fatalf("expected no project config write when global is unavailable")
	}

	intIdx := optionIndex(updated.settings.Options, "tui.runDataRefreshIntervalSeconds")
	if intIdx < 0 {
		t.Fatalf("missing int option")
	}
	updated.settings.Selected = intIdx
	updated.settings.Column = SettingsColumnGlobal
	updated, _ = HandleSettingsKey(updated, tea.KeyMsg{Type: tea.KeyEnter})
	if updated.settings.Editing {
		t.Fatalf("expected edit mode to remain off when global is unavailable")
	}
}

func optionIndex(options []config.OptionMetadata, key string) int {
	for idx, option := range options {
		if option.KeyPath == key {
			return idx
		}
	}
	return -1
}
