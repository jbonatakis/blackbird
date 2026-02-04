package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbonatakis/blackbird/internal/config"
)

func TestNewStartupModelNoPlanFile(t *testing.T) {
	model := newStartupModel("")

	if model.viewMode != ViewModeHome {
		t.Fatalf("expected viewMode to be ViewModeHome, got %v", model.viewMode)
	}
	if model.planExists {
		t.Fatalf("expected planExists to be false when no plan file exists")
	}
	if len(model.plan.Items) != 0 {
		t.Fatalf("expected empty plan on startup, got %d items", len(model.plan.Items))
	}
}

func TestNewStartupModelLoadsConfigFromProjectRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	configDir := filepath.Join(root, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	path := filepath.Join(configDir, "config.json")
	data := []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12,"planDataRefreshIntervalSeconds":34}}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	model := newStartupModel(root)
	if model.projectRoot != root {
		t.Fatalf("expected projectRoot %q, got %q", root, model.projectRoot)
	}
	if model.config.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("expected run refresh interval 12, got %d", model.config.TUI.RunDataRefreshIntervalSeconds)
	}
	if model.config.TUI.PlanDataRefreshIntervalSeconds != 34 {
		t.Fatalf("expected plan refresh interval 34, got %d", model.config.TUI.PlanDataRefreshIntervalSeconds)
	}
	if model.config.SchemaVersion != config.SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", config.SchemaVersion, model.config.SchemaVersion)
	}
	if model.settings.ProjectRoot != root {
		t.Fatalf("expected settings projectRoot %q, got %q", root, model.settings.ProjectRoot)
	}
	projectValue, ok := model.settings.Resolution.Project.Values["tui.runDataRefreshIntervalSeconds"]
	if !ok || projectValue.Int == nil || *projectValue.Int != 12 {
		t.Fatalf("expected settings to load local run refresh interval 12, got %#v", projectValue)
	}
	appliedValue, ok := model.settings.Resolution.Applied["tui.runDataRefreshIntervalSeconds"]
	if !ok || appliedValue.Value.Int == nil || *appliedValue.Value.Int != 12 || appliedValue.Source != config.ConfigSourceLocal {
		t.Fatalf("expected applied run refresh interval 12 from local, got %#v", appliedValue)
	}
}

func TestNewStartupModelUsesDefaultConfigWhenMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", t.TempDir())

	model := newStartupModel(root)
	defaults := config.DefaultResolvedConfig()

	if model.config.TUI.RunDataRefreshIntervalSeconds != defaults.TUI.RunDataRefreshIntervalSeconds {
		t.Fatalf("expected run refresh interval %d, got %d", defaults.TUI.RunDataRefreshIntervalSeconds, model.config.TUI.RunDataRefreshIntervalSeconds)
	}
	if model.config.TUI.PlanDataRefreshIntervalSeconds != defaults.TUI.PlanDataRefreshIntervalSeconds {
		t.Fatalf("expected plan refresh interval %d, got %d", defaults.TUI.PlanDataRefreshIntervalSeconds, model.config.TUI.PlanDataRefreshIntervalSeconds)
	}
	if model.config.SchemaVersion != defaults.SchemaVersion {
		t.Fatalf("expected schema version %d, got %d", defaults.SchemaVersion, model.config.SchemaVersion)
	}
	appliedValue, ok := model.settings.Resolution.Applied["tui.runDataRefreshIntervalSeconds"]
	if !ok || appliedValue.Value.Int == nil || *appliedValue.Value.Int != defaults.TUI.RunDataRefreshIntervalSeconds {
		t.Fatalf("expected settings applied run refresh interval %d, got %#v", defaults.TUI.RunDataRefreshIntervalSeconds, appliedValue)
	}
}
