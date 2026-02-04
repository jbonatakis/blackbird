package tui

import (
	"os"
	"path/filepath"
	"testing"

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
		viewMode:   ViewModeSettings,
		projectRoot: projectRoot,
		config:     config.DefaultResolvedConfig(),
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

func TestSettingsIntEditValidationAndAutosave(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		viewMode:   ViewModeSettings,
		projectRoot: projectRoot,
		config:     config.DefaultResolvedConfig(),
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
		viewMode:   ViewModeSettings,
		projectRoot: projectRoot,
		config:     config.DefaultResolvedConfig(),
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
