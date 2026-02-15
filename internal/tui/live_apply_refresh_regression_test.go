package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
	"github.com/muesli/termenv"
)

const (
	regressionKeyTheme          = "tui.theme"
	regressionKeyRunDataRefresh = "tui.runDataRefreshIntervalSeconds"
)

func TestSettingsThemeChangeAppliesToRenderedOutputInSession(t *testing.T) {
	setANSI256ColorProfile(t)

	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	model := Model{
		projectRoot: projectRoot,
		config:      config.DefaultResolvedConfig(),
		theme:       resolveActiveTheme(config.DefaultResolvedConfig()),
		runData: map[string]execution.RunRecord{
			"run-1": testRunningRunRecord(),
		},
	}
	model.settings = NewSettingsState(projectRoot, model.config)

	themeRow := settingsOptionIndex(model.settings.Options, regressionKeyTheme)
	if themeRow < 0 {
		t.Fatalf("missing theme option %q", regressionKeyTheme)
	}
	model.settings.Selected = themeRow
	model.settings.Column = SettingsColumnLocal

	beforeStatus := renderRunStatus(model.theme, execution.RunStatusRunning)
	before := RenderExecutionView(model)
	if !strings.Contains(before, beforeStatus) {
		t.Fatalf("expected execution view to include running status with current theme")
	}

	originalDebounce := settingsThemeDebounceCmd
	settingsThemeDebounceCmd = func(_ time.Duration, token uint64) tea.Cmd {
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
	if updated.theme.ID != ThemeIDBlackbird {
		t.Fatalf("expected theme to remain %q before debounce flush, got %q", ThemeIDBlackbird, updated.theme.ID)
	}

	updatedModel, _ := updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected active theme %q after debounce flush, got %q", ThemeIDHighContrast, updated.theme.ID)
	}

	afterStatus := renderRunStatus(updated.theme, execution.RunStatusRunning)
	if beforeStatus == afterStatus {
		t.Fatalf("expected rendered running status style to change across themes")
	}
	after := RenderExecutionView(updated)
	if !strings.Contains(after, afterStatus) {
		t.Fatalf("expected execution view to include running status with updated theme")
	}
	if strings.Contains(after, beforeStatus) {
		t.Fatalf("expected execution view to stop using pre-change theme style")
	}
}

func TestConfigRefreshAppliesThemeWhileReviewCheckpointModalOpen(t *testing.T) {
	setANSI256ColorProfile(t)

	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	blackbird := config.ThemeIDBlackbird
	saveConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		regressionKeyTheme: {String: &blackbird},
	})
	initialConfig, err := config.LoadConfig(projectRoot)
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}

	form := NewReviewCheckpointForm(testRunningDecisionRunRecord(), plan.NewEmptyWorkGraph())
	model := Model{
		projectRoot:          projectRoot,
		config:               initialConfig,
		theme:                resolveActiveTheme(initialConfig),
		settings:             NewSettingsState(projectRoot, initialConfig),
		windowWidth:          100,
		windowHeight:         40,
		actionMode:           ActionModeReviewCheckpoint,
		reviewCheckpointForm: &form,
	}

	beforeStatus := renderRunStatus(model.theme, execution.RunStatusRunning)
	before := model.View()
	if !strings.Contains(before, beforeStatus) {
		t.Fatalf("expected modal render to include running status with current theme")
	}

	highContrast := config.ThemeIDHighContrast
	saveConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		regressionKeyTheme: {String: &highContrast},
	})

	updated := applyConfigRefreshCycle(t, model)
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected refreshed theme %q, got %q", ThemeIDHighContrast, updated.theme.ID)
	}
	if updated.actionMode != ActionModeReviewCheckpoint || updated.reviewCheckpointForm == nil {
		t.Fatalf("expected review checkpoint modal to remain open after refresh")
	}

	afterStatus := renderRunStatus(updated.theme, execution.RunStatusRunning)
	if beforeStatus == afterStatus {
		t.Fatalf("expected running status style to change after theme refresh")
	}
	after := updated.View()
	if !strings.Contains(after, afterStatus) {
		t.Fatalf("expected modal render to include running status with refreshed theme")
	}
	if strings.Contains(after, beforeStatus) {
		t.Fatalf("expected modal render to stop using pre-refresh theme style")
	}
}

func TestConfigRefreshCyclePreservesSettingsCursorAndNonThemeEditBuffer(t *testing.T) {
	projectRoot := t.TempDir()
	home := t.TempDir()
	restoreHome := config.SetUserHomeDirForTest(func() (string, error) {
		return home, nil
	})
	defer restoreHome()

	initialRunInterval := 12
	blackbird := config.ThemeIDBlackbird
	saveConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		regressionKeyRunDataRefresh: {Int: &initialRunInterval},
		regressionKeyTheme:          {String: &blackbird},
	})
	initialConfig, err := config.LoadConfig(projectRoot)
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}

	model := Model{
		viewMode:    ViewModeSettings,
		projectRoot: projectRoot,
		config:      initialConfig,
		theme:       resolveActiveTheme(initialConfig),
	}
	model.settings = NewSettingsState(projectRoot, initialConfig)

	row := settingsOptionIndex(model.settings.Options, regressionKeyRunDataRefresh)
	if row < 0 {
		t.Fatalf("missing option %q", regressionKeyRunDataRefresh)
	}
	model.settings.Selected = row
	model.settings.Column = SettingsColumnLocal
	model.settings.Editing = true
	model.settings.EditValue = "299"

	updatedRunInterval := 25
	highContrast := config.ThemeIDHighContrast
	saveConfigValues(t, filepath.Join(projectRoot, ".blackbird", "config.json"), map[string]config.RawOptionValue{
		regressionKeyRunDataRefresh: {Int: &updatedRunInterval},
		regressionKeyTheme:          {String: &highContrast},
	})

	updated := applyConfigRefreshCycle(t, model)

	if updated.settings.Selected != row {
		t.Fatalf("settings selected row = %d, want %d", updated.settings.Selected, row)
	}
	if updated.settings.Column != SettingsColumnLocal {
		t.Fatalf("settings selected column = %d, want %d", updated.settings.Column, SettingsColumnLocal)
	}
	if !updated.settings.Editing {
		t.Fatalf("expected text edit mode to remain active")
	}
	if updated.settings.EditValue != "299" {
		t.Fatalf("expected edit buffer to be preserved, got %q", updated.settings.EditValue)
	}

	local := updated.settings.Resolution.Project.Values[regressionKeyRunDataRefresh]
	if local.Int == nil || *local.Int != updatedRunInterval {
		t.Fatalf("expected refreshed local run interval %d, got %#v", updatedRunInterval, local.Int)
	}
	applied := updated.settings.Resolution.Applied[regressionKeyRunDataRefresh]
	if applied.Source != config.ConfigSourceLocal {
		t.Fatalf("applied source = %s, want %s", applied.Source, config.ConfigSourceLocal)
	}
	if applied.Value.Int == nil || *applied.Value.Int != updatedRunInterval {
		t.Fatalf("expected applied run interval %d, got %#v", updatedRunInterval, applied.Value.Int)
	}
	if updated.config.TUI.RunDataRefreshIntervalSeconds != updatedRunInterval {
		t.Fatalf("expected refreshed model run interval %d, got %d", updatedRunInterval, updated.config.TUI.RunDataRefreshIntervalSeconds)
	}
	if updated.theme.ID != ThemeIDHighContrast {
		t.Fatalf("expected refreshed theme %q, got %q", ThemeIDHighContrast, updated.theme.ID)
	}
}

func applyConfigRefreshCycle(t *testing.T, model Model) Model {
	t.Helper()

	updatedModel, cmd := model.Update(configRefreshMsg{})
	updated := updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("expected refresh command batch")
	}

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected tea.BatchMsg from refresh command, got %T", msg)
	}
	if len(batch) < 1 || batch[0] == nil {
		t.Fatalf("expected first batch command to load config")
	}

	loadMsg := batch[0]()
	loaded, ok := loadMsg.(ConfigStateLoaded)
	if !ok {
		t.Fatalf("expected ConfigStateLoaded, got %T", loadMsg)
	}
	if loaded.Err != nil {
		t.Fatalf("load config during refresh: %v", loaded.Err)
	}

	updatedModel, _ = updated.Update(loaded)
	return updatedModel.(Model)
}

func setANSI256ColorProfile(t *testing.T) {
	t.Helper()
	original := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() {
		lipgloss.SetColorProfile(original)
	})
}

func settingsOptionIndex(options []config.OptionMetadata, key string) int {
	for idx, option := range options {
		if option.KeyPath == key {
			return idx
		}
	}
	return -1
}

func saveConfigValues(t *testing.T, path string, values map[string]config.RawOptionValue) {
	t.Helper()
	if err := config.SaveConfigValues(path, values); err != nil {
		t.Fatalf("save config values (%s): %v", path, err)
	}
}

func testRunningRunRecord() execution.RunRecord {
	started := time.Date(2026, 2, 15, 11, 0, 0, 0, time.UTC)
	return execution.RunRecord{
		ID:        "run-1",
		TaskID:    "task-1",
		StartedAt: started,
		Status:    execution.RunStatusRunning,
	}
}

func testRunningDecisionRunRecord() execution.RunRecord {
	started := time.Date(2026, 2, 15, 11, 0, 0, 0, time.UTC)
	return execution.RunRecord{
		ID:               "run-review-1",
		TaskID:           "task-1",
		StartedAt:        started,
		Status:           execution.RunStatusRunning,
		DecisionRequired: true,
		DecisionState:    execution.DecisionStatePending,
		Context: execution.ContextPack{
			Task: execution.TaskContext{
				ID:    "task-1",
				Title: "Theme refresh task",
			},
		},
	}
}
