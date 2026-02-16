package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
)

const (
	testKeyRunDataRefresh = "tui.runDataRefreshIntervalSeconds"
	testKeyTheme          = "tui.theme"
	testKeyParentReview   = "execution.parentReviewEnabled"
	testKeyStopAfter      = "execution.stopAfterEachTask"
)

func TestLoadConfigStateReturnsResolvedConfigAndSettings(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	localRunInterval := 12
	globalStopAfter := true
	highContrast := config.ThemeIDHighContrast

	writeConfigValues(t, filepath.Join(home, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyTheme:     {String: &highContrast},
		testKeyStopAfter: {Bool: &globalStopAfter},
	})
	writeConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyRunDataRefresh: {Int: &localRunInterval},
	})

	model := Model{projectRoot: projectRoot}
	msg := model.LoadConfigState()()
	loaded, ok := msg.(ConfigStateLoaded)
	if !ok {
		t.Fatalf("expected ConfigStateLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadConfigState error: %v", loaded.Err)
	}
	if loaded.Config.TUI.RunDataRefreshIntervalSeconds != localRunInterval {
		t.Fatalf("resolved run interval = %d, want %d", loaded.Config.TUI.RunDataRefreshIntervalSeconds, localRunInterval)
	}
	if loaded.Config.TUI.Theme != config.ThemeIDHighContrast {
		t.Fatalf("resolved theme = %q, want %q", loaded.Config.TUI.Theme, config.ThemeIDHighContrast)
	}
	if !loaded.Config.Execution.StopAfterEachTask {
		t.Fatalf("resolved stop-after-each-task should be true from global config")
	}

	appliedRun := loaded.Resolution.Applied[testKeyRunDataRefresh]
	if appliedRun.Source != config.ConfigSourceLocal {
		t.Fatalf("run interval applied source = %s, want %s", appliedRun.Source, config.ConfigSourceLocal)
	}
	if appliedRun.Value.Int == nil || *appliedRun.Value.Int != localRunInterval {
		t.Fatalf("run interval applied value = %#v, want %d", appliedRun.Value.Int, localRunInterval)
	}

	appliedStopAfter := loaded.Resolution.Applied[testKeyStopAfter]
	if appliedStopAfter.Source != config.ConfigSourceGlobal {
		t.Fatalf("stop-after applied source = %s, want %s", appliedStopAfter.Source, config.ConfigSourceGlobal)
	}
	if appliedStopAfter.Value.Bool == nil || !*appliedStopAfter.Value.Bool {
		t.Fatalf("stop-after applied value = %#v, want true", appliedStopAfter.Value.Bool)
	}
}

func TestUpdateConfigRefreshMsgSchedulesReloadAndNextTick(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
	}
	model.config.TUI.RunDataRefreshIntervalSeconds = 1

	_, cmd := model.Update(configRefreshMsg{})
	if cmd == nil {
		t.Fatalf("expected config refresh update to return command batch")
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg, got %T", msg)
	}
	if len(batch) != 2 {
		t.Fatalf("expected two commands (load + reschedule), got %d", len(batch))
	}
	if batch[0] == nil {
		t.Fatalf("expected first batch command to be non-nil")
	}
	if _, ok := batch[0]().(ConfigStateLoaded); !ok {
		t.Fatalf("expected first batch command to load config state")
	}
}

func TestConfigRefreshApplyPreservesSettingsSelectionAndColumn(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	globalParentReview := true
	writeConfigValues(t, filepath.Join(home, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyParentReview: {Bool: &globalParentReview},
	})

	initialConfig, err := config.LoadConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadConfig initial: %v", err)
	}
	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      initialConfig,
		theme:       resolveActiveTheme(initialConfig),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	row := optionIndexByKey(model.settings.Options, testKeyParentReview)
	if row < 0 {
		t.Fatalf("missing option %q", testKeyParentReview)
	}
	model.settings.Selected = row
	model.settings.Column = SettingsColumnGlobal

	localParentReview := false
	highContrast := config.ThemeIDHighContrast
	writeConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyParentReview: {Bool: &localParentReview},
		testKeyTheme:        {String: &highContrast},
	})

	msg := model.LoadConfigState()()
	loaded, ok := msg.(ConfigStateLoaded)
	if !ok {
		t.Fatalf("expected ConfigStateLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadConfigState refresh: %v", loaded.Err)
	}

	updatedModel, _ := model.Update(msg)
	updated := updatedModel.(Model)

	if updated.settings.Selected != row {
		t.Fatalf("settings selected row = %d, want %d", updated.settings.Selected, row)
	}
	if updated.settings.Column != SettingsColumnGlobal {
		t.Fatalf("settings selected column = %d, want %d", updated.settings.Column, SettingsColumnGlobal)
	}
	if updated.config.Execution.ParentReviewEnabled {
		t.Fatalf("expected refreshed parent-review setting to be false from local config")
	}
	if updated.config.TUI.Theme != config.ThemeIDHighContrast {
		t.Fatalf("expected refreshed config theme %q, got %q", config.ThemeIDHighContrast, updated.config.TUI.Theme)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q, got %q", ThemeIDHighContrast, updated.theme.ID)
	}

	local := updated.settings.Resolution.Project.Values[testKeyParentReview]
	if local.Bool == nil || *local.Bool {
		t.Fatalf("expected refreshed local value false, got %#v", local.Bool)
	}
	global := updated.settings.Resolution.Global.Values[testKeyParentReview]
	if global.Bool == nil || !*global.Bool {
		t.Fatalf("expected refreshed global value true, got %#v", global.Bool)
	}
	applied := updated.settings.Resolution.Applied[testKeyParentReview]
	if applied.Source != config.ConfigSourceLocal {
		t.Fatalf("applied source = %s, want %s", applied.Source, config.ConfigSourceLocal)
	}
	if applied.Value.Bool == nil || *applied.Value.Bool {
		t.Fatalf("expected applied value false from local, got %#v", applied.Value.Bool)
	}
}

func TestConfigRefreshApplyPreservesNonThemeEditBuffer(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	initialRunInterval := 12
	blackbird := config.ThemeIDBlackbird
	writeConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyRunDataRefresh: {Int: &initialRunInterval},
		testKeyTheme:          {String: &blackbird},
	})

	initialConfig, err := config.LoadConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadConfig initial: %v", err)
	}
	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      initialConfig,
		theme:       resolveActiveTheme(initialConfig),
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	row := optionIndexByKey(model.settings.Options, testKeyRunDataRefresh)
	if row < 0 {
		t.Fatalf("missing option %q", testKeyRunDataRefresh)
	}
	model.settings.Selected = row
	model.settings.Column = SettingsColumnLocal
	model.settings.Editing = true
	model.settings.EditValue = "299"

	updatedRunInterval := 25
	highContrast := config.ThemeIDHighContrast
	writeConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		testKeyRunDataRefresh: {Int: &updatedRunInterval},
		testKeyTheme:          {String: &highContrast},
	})

	msg := model.LoadConfigState()()
	loaded, ok := msg.(ConfigStateLoaded)
	if !ok {
		t.Fatalf("expected ConfigStateLoaded, got %T", msg)
	}
	if loaded.Err != nil {
		t.Fatalf("LoadConfigState refresh: %v", loaded.Err)
	}

	updatedModel, _ := model.Update(msg)
	updated := updatedModel.(Model)

	if !updated.settings.Editing {
		t.Fatalf("expected text edit mode to remain active")
	}
	if updated.settings.EditValue != "299" {
		t.Fatalf("expected edit buffer to be preserved, got %q", updated.settings.EditValue)
	}
	if updated.settings.Selected != row {
		t.Fatalf("settings selected row = %d, want %d", updated.settings.Selected, row)
	}
	if updated.settings.Column != SettingsColumnLocal {
		t.Fatalf("settings selected column = %d, want %d", updated.settings.Column, SettingsColumnLocal)
	}

	local := updated.settings.Resolution.Project.Values[testKeyRunDataRefresh]
	if local.Int == nil || *local.Int != updatedRunInterval {
		t.Fatalf("expected refreshed local run interval %d, got %#v", updatedRunInterval, local.Int)
	}
	applied := updated.settings.Resolution.Applied[testKeyRunDataRefresh]
	if applied.Source != config.ConfigSourceLocal {
		t.Fatalf("applied source = %s, want %s", applied.Source, config.ConfigSourceLocal)
	}
	if applied.Value.Int == nil || *applied.Value.Int != updatedRunInterval {
		t.Fatalf("expected applied run interval %d, got %#v", updatedRunInterval, applied.Value.Int)
	}
	if updated.config.TUI.RunDataRefreshIntervalSeconds != updatedRunInterval {
		t.Fatalf("expected refreshed model config run interval %d, got %d", updatedRunInterval, updated.config.TUI.RunDataRefreshIntervalSeconds)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q, got %q", ThemeIDHighContrast, updated.theme.ID)
	}
}

func writeConfigValues(t *testing.T, path string, values map[string]config.RawOptionValue) {
	t.Helper()
	if err := config.SaveConfigValues(path, values); err != nil {
		t.Fatalf("SaveConfigValues(%s): %v", path, err)
	}
}

func optionIndexByKey(options []config.OptionMetadata, key string) int {
	for idx, option := range options {
		if option.KeyPath == key {
			return idx
		}
	}
	return -1
}
