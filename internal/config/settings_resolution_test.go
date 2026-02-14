package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSettingsPrecedenceAndSource(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return homeDir, nil
	})
	t.Cleanup(restore)

	globalPath := filepath.Join(homeDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12},"planning":{"maxPlanAutoRefinePasses":2},"execution":{"parentReviewEnabled":true}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":21},"planning":{"maxPlanAutoRefinePasses":3}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolution, err := ResolveSettings(projectDir)
	if err != nil {
		t.Fatalf("resolve settings: %v", err)
	}

	assertAppliedInt(t, resolution, keyTuiRunDataRefreshIntervalSeconds, 12, ConfigSourceGlobal)
	assertAppliedInt(t, resolution, keyTuiPlanDataRefreshIntervalSeconds, 21, ConfigSourceLocal)
	assertAppliedInt(t, resolution, keyPlanningMaxPlanAutoRefinePasses, 3, ConfigSourceLocal)
	assertAppliedBool(t, resolution, keyExecutionStopAfterEachTask, DefaultStopAfterEachTask, ConfigSourceDefault)
	assertAppliedBool(t, resolution, keyExecutionParentReviewEnabled, true, ConfigSourceGlobal)
}

func TestResolveSettingsParentReviewEnabledPrecedence(t *testing.T) {
	tests := []struct {
		name          string
		globalConfig  string
		projectConfig string
		wantValue     bool
		wantSource    ConfigSource
	}{
		{
			name:          "unset in both layers defaults false",
			globalConfig:  `{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`,
			projectConfig: `{"schemaVersion":1,"planning":{"maxPlanAutoRefinePasses":2}}`,
			wantValue:     false,
			wantSource:    ConfigSourceDefault,
		},
		{
			name:          "local true overrides global false",
			globalConfig:  `{"schemaVersion":1,"execution":{"parentReviewEnabled":false}}`,
			projectConfig: `{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`,
			wantValue:     true,
			wantSource:    ConfigSourceLocal,
		},
		{
			name:          "local false overrides global true",
			globalConfig:  `{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`,
			projectConfig: `{"schemaVersion":1,"execution":{"parentReviewEnabled":false}}`,
			wantValue:     false,
			wantSource:    ConfigSourceLocal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			homeDir := t.TempDir()
			projectDir := t.TempDir()
			restore := overrideUserHomeDir(func() (string, error) {
				return homeDir, nil
			})
			t.Cleanup(restore)

			globalPath := filepath.Join(homeDir, ".blackbird", "config.json")
			if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
				t.Fatalf("mkdir global: %v", err)
			}
			if err := os.WriteFile(globalPath, []byte(tt.globalConfig), 0o644); err != nil {
				t.Fatalf("write global config: %v", err)
			}

			projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
			if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
				t.Fatalf("mkdir project: %v", err)
			}
			if err := os.WriteFile(projectPath, []byte(tt.projectConfig), 0o644); err != nil {
				t.Fatalf("write project config: %v", err)
			}

			resolution, err := ResolveSettings(projectDir)
			if err != nil {
				t.Fatalf("resolve settings: %v", err)
			}

			assertAppliedBool(
				t,
				resolution,
				keyExecutionParentReviewEnabled,
				tt.wantValue,
				tt.wantSource,
			)
		})
	}
}

func TestResolveSettingsOutOfRangeWarnings(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return homeDir, nil
	})
	t.Cleanup(restore)

	globalPath := filepath.Join(homeDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":0},"planning":{"maxPlanAutoRefinePasses":5}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":400},"planning":{"maxPlanAutoRefinePasses":-1}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolution, err := ResolveSettings(projectDir)
	if err != nil {
		t.Fatalf("resolve settings: %v", err)
	}

	assertAppliedInt(t, resolution, keyTuiRunDataRefreshIntervalSeconds, MaxRefreshIntervalSeconds, ConfigSourceLocal)
	assertAppliedInt(t, resolution, keyTuiPlanDataRefreshIntervalSeconds, MinRefreshIntervalSeconds, ConfigSourceGlobal)
	assertAppliedInt(t, resolution, keyPlanningMaxPlanAutoRefinePasses, MinPlanAutoRefinePasses, ConfigSourceLocal)
	assertAppliedBool(t, resolution, keyExecutionParentReviewEnabled, DefaultParentReviewEnabled, ConfigSourceDefault)

	runWarning, ok := findOptionWarning(resolution.OptionWarnings, ConfigSourceLocal, keyTuiRunDataRefreshIntervalSeconds)
	if !ok {
		t.Fatalf("expected warning for local run interval")
	}
	if runWarning.ClampedInt == nil || *runWarning.ClampedInt != MaxRefreshIntervalSeconds {
		t.Fatalf("run warning clamped = %#v, want %d", runWarning.ClampedInt, MaxRefreshIntervalSeconds)
	}

	planWarning, ok := findOptionWarning(resolution.OptionWarnings, ConfigSourceGlobal, keyTuiPlanDataRefreshIntervalSeconds)
	if !ok {
		t.Fatalf("expected warning for global plan interval")
	}
	if planWarning.ClampedInt == nil || *planWarning.ClampedInt != MinRefreshIntervalSeconds {
		t.Fatalf("plan warning clamped = %#v, want %d", planWarning.ClampedInt, MinRefreshIntervalSeconds)
	}

	planningLocalWarning, ok := findOptionWarning(resolution.OptionWarnings, ConfigSourceLocal, keyPlanningMaxPlanAutoRefinePasses)
	if !ok {
		t.Fatalf("expected warning for local planning max auto-refine passes")
	}
	if planningLocalWarning.ClampedInt == nil || *planningLocalWarning.ClampedInt != MinPlanAutoRefinePasses {
		t.Fatalf("planning local warning clamped = %#v, want %d", planningLocalWarning.ClampedInt, MinPlanAutoRefinePasses)
	}

	planningGlobalWarning, ok := findOptionWarning(resolution.OptionWarnings, ConfigSourceGlobal, keyPlanningMaxPlanAutoRefinePasses)
	if !ok {
		t.Fatalf("expected warning for global planning max auto-refine passes")
	}
	if planningGlobalWarning.ClampedInt == nil || *planningGlobalWarning.ClampedInt != MaxPlanAutoRefinePasses {
		t.Fatalf("planning global warning clamped = %#v, want %d", planningGlobalWarning.ClampedInt, MaxPlanAutoRefinePasses)
	}
}

func TestResolveSettingsLayerWarnings(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return homeDir, nil
	})
	t.Cleanup(restore)

	globalPath := filepath.Join(homeDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":2,"tui":{"runDataRefreshIntervalSeconds":99}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolution, err := ResolveSettings(projectDir)
	if err != nil {
		t.Fatalf("resolve settings: %v", err)
	}

	if _, ok := findLayerWarning(resolution.LayerWarnings, ConfigSourceGlobal, LayerWarningInvalidJSON); !ok {
		t.Fatalf("expected global invalid JSON warning")
	}
	if _, ok := findLayerWarning(resolution.LayerWarnings, ConfigSourceLocal, LayerWarningUnsupportedSchema); !ok {
		t.Fatalf("expected local unsupported schema warning")
	}

	if len(resolution.Project.Values) != 0 {
		t.Fatalf("expected project values to be empty")
	}
	if len(resolution.Global.Values) != 0 {
		t.Fatalf("expected global values to be empty")
	}

	assertAppliedInt(t, resolution, keyTuiRunDataRefreshIntervalSeconds, DefaultRunDataRefreshIntervalSeconds, ConfigSourceDefault)
	assertAppliedInt(t, resolution, keyTuiPlanDataRefreshIntervalSeconds, DefaultPlanDataRefreshIntervalSeconds, ConfigSourceDefault)
	assertAppliedInt(t, resolution, keyPlanningMaxPlanAutoRefinePasses, DefaultMaxPlanAutoRefinePasses, ConfigSourceDefault)
	assertAppliedBool(t, resolution, keyExecutionStopAfterEachTask, DefaultStopAfterEachTask, ConfigSourceDefault)
	assertAppliedBool(t, resolution, keyExecutionParentReviewEnabled, DefaultParentReviewEnabled, ConfigSourceDefault)
}

func TestResolveSettingsGlobalUnavailable(t *testing.T) {
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return "", errors.New("no home")
	})
	t.Cleanup(restore)

	resolution, err := ResolveSettings(projectDir)
	if err != nil {
		t.Fatalf("resolve settings: %v", err)
	}

	if resolution.Global.Available {
		t.Fatalf("expected global to be unavailable")
	}
	if resolution.Global.Path != "" {
		t.Fatalf("expected global path to be empty, got %q", resolution.Global.Path)
	}
	if len(resolution.Global.Values) != 0 {
		t.Fatalf("expected global values to be empty")
	}
}

func assertAppliedInt(t *testing.T, resolution SettingsResolution, key string, want int, wantSource ConfigSource) {
	t.Helper()
	option, ok := resolution.Applied[key]
	if !ok {
		t.Fatalf("missing applied value for %s", key)
	}
	if option.Source != wantSource {
		t.Fatalf("applied source for %s = %s, want %s", key, option.Source, wantSource)
	}
	if option.Value.Int == nil || *option.Value.Int != want {
		t.Fatalf("applied int for %s = %#v, want %d", key, option.Value.Int, want)
	}
}

func assertAppliedBool(t *testing.T, resolution SettingsResolution, key string, want bool, wantSource ConfigSource) {
	t.Helper()
	option, ok := resolution.Applied[key]
	if !ok {
		t.Fatalf("missing applied value for %s", key)
	}
	if option.Source != wantSource {
		t.Fatalf("applied source for %s = %s, want %s", key, option.Source, wantSource)
	}
	if option.Value.Bool == nil || *option.Value.Bool != want {
		t.Fatalf("applied bool for %s = %#v, want %v", key, option.Value.Bool, want)
	}
}

func findOptionWarning(warnings []OptionWarning, source ConfigSource, key string) (OptionWarning, bool) {
	for _, warning := range warnings {
		if warning.Source == source && warning.KeyPath == key {
			return warning, true
		}
	}
	return OptionWarning{}, false
}

func findLayerWarning(warnings []LayerWarning, source ConfigSource, kind LayerWarningKind) (LayerWarning, bool) {
	for _, warning := range warnings {
		if warning.Source == source && warning.Kind == kind {
			return warning, true
		}
	}
	return LayerWarning{}, false
}
