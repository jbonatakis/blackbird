package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRawOptionValues(t *testing.T) {
	run := 12
	stop := false
	version := SchemaVersion
	cfg := RawConfig{
		SchemaVersion: &version,
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds: &run,
		},
		Execution: &RawExecution{
			StopAfterEachTask: &stop,
		},
	}

	values := RawOptionValues(cfg)
	if len(values) != 2 {
		t.Fatalf("values len = %d, want 2", len(values))
	}

	runValue, ok := values[keyTuiRunDataRefreshIntervalSeconds]
	if !ok || runValue.Int == nil || *runValue.Int != 12 {
		t.Fatalf("run value = %#v, want 12", runValue)
	}
	if runValue.Bool != nil {
		t.Fatalf("run value bool = %v, want nil", runValue.Bool)
	}

	stopValue, ok := values[keyExecutionStopAfterEachTask]
	if !ok || stopValue.Bool == nil || *stopValue.Bool != false {
		t.Fatalf("stop value = %#v, want false", stopValue)
	}
	if stopValue.Int != nil {
		t.Fatalf("stop value int = %v, want nil", stopValue.Int)
	}

	if _, ok := values[keyTuiPlanDataRefreshIntervalSeconds]; ok {
		t.Fatalf("expected plan refresh to be unset")
	}
}

func TestLoadLayerOptionValues(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":20}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	project, global, err := LoadLayerOptionValues(projectDir)
	if err != nil {
		t.Fatalf("load layer values: %v", err)
	}
	if !project.Present {
		t.Fatalf("expected project config to be present")
	}
	if !global.Present {
		t.Fatalf("expected global config to be present")
	}

	stopValue, ok := project.Values[keyExecutionStopAfterEachTask]
	if !ok || stopValue.Bool == nil || *stopValue.Bool != true {
		t.Fatalf("project stop value = %#v, want true", stopValue)
	}
	if _, ok := project.Values[keyTuiRunDataRefreshIntervalSeconds]; ok {
		t.Fatalf("expected project run refresh to be unset")
	}

	runValue, ok := global.Values[keyTuiRunDataRefreshIntervalSeconds]
	if !ok || runValue.Int == nil || *runValue.Int != 20 {
		t.Fatalf("global run value = %#v, want 20", runValue)
	}
	if _, ok := global.Values[keyExecutionStopAfterEachTask]; ok {
		t.Fatalf("expected global stop after each task to be unset")
	}
}

func TestSaveConfigValuesWritesFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")

	run := 8
	stop := false
	values := map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds: {Int: &run},
		keyExecutionStopAfterEachTask:       {Bool: &stop},
	}

	if err := SaveConfigValues(path, values); err != nil {
		t.Fatalf("save config values: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(b), "\"schemaVersion\": 1") {
		t.Fatalf("expected schemaVersion in config, got %s", string(b))
	}
	if strings.Contains(string(b), "planDataRefreshIntervalSeconds") {
		t.Fatalf("expected plan refresh interval to be omitted, got %s", string(b))
	}

	var cfg RawConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.SchemaVersion == nil || *cfg.SchemaVersion != SchemaVersion {
		t.Fatalf("schemaVersion = %#v, want %d", cfg.SchemaVersion, SchemaVersion)
	}
	if cfg.TUI == nil || cfg.TUI.RunDataRefreshIntervalSeconds == nil || *cfg.TUI.RunDataRefreshIntervalSeconds != 8 {
		t.Fatalf("run interval = %#v, want 8", cfg.TUI)
	}
	if cfg.TUI.PlanDataRefreshIntervalSeconds != nil {
		t.Fatalf("plan interval = %v, want nil", cfg.TUI.PlanDataRefreshIntervalSeconds)
	}
	if cfg.Execution == nil || cfg.Execution.StopAfterEachTask == nil || *cfg.Execution.StopAfterEachTask != false {
		t.Fatalf("stop after each task = %#v, want false", cfg.Execution)
	}
}

func TestSaveConfigValuesUpdatesExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":15},"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	run := 9
	stop := false
	values := map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds: {Int: &run},
		keyExecutionStopAfterEachTask:       {Bool: &stop},
	}

	if err := SaveConfigValues(path, values); err != nil {
		t.Fatalf("save config values: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(b), "planDataRefreshIntervalSeconds") {
		t.Fatalf("expected plan refresh interval to be removed, got %s", string(b))
	}

	var cfg RawConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg.SchemaVersion == nil || *cfg.SchemaVersion != SchemaVersion {
		t.Fatalf("schemaVersion = %#v, want %d", cfg.SchemaVersion, SchemaVersion)
	}
	if cfg.TUI == nil || cfg.TUI.RunDataRefreshIntervalSeconds == nil || *cfg.TUI.RunDataRefreshIntervalSeconds != 9 {
		t.Fatalf("run interval = %#v, want 9", cfg.TUI)
	}
	if cfg.TUI.PlanDataRefreshIntervalSeconds != nil {
		t.Fatalf("plan interval = %v, want nil", cfg.TUI.PlanDataRefreshIntervalSeconds)
	}
	if cfg.Execution == nil || cfg.Execution.StopAfterEachTask == nil || *cfg.Execution.StopAfterEachTask != false {
		t.Fatalf("stop after each task = %#v, want false", cfg.Execution)
	}
}

func TestSaveConfigValuesRemovesFileWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := SaveConfigValues(path, map[string]RawOptionValue{}); err != nil {
		t.Fatalf("save config values: %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected config file to be removed, got %v", err)
	}
}

func TestSaveConfigValuesRejectsUnknownKey(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")
	value := 3
	if err := SaveConfigValues(path, map[string]RawOptionValue{
		"unknown.key": {Int: &value},
	}); err == nil {
		t.Fatalf("expected error for unknown key")
	}
}

func TestSaveConfigValuesRejectsWrongType(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")
	value := 1
	if err := SaveConfigValues(path, map[string]RawOptionValue{
		keyExecutionStopAfterEachTask: {Int: &value},
	}); err == nil {
		t.Fatalf("expected error for wrong value type")
	}
}
