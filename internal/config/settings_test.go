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
	refinePasses := 2
	stop := false
	parentReview := true
	version := SchemaVersion
	cfg := RawConfig{
		SchemaVersion: &version,
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds: &run,
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: &refinePasses,
		},
		Execution: &RawExecution{
			StopAfterEachTask:   &stop,
			ParentReviewEnabled: &parentReview,
		},
	}

	values := RawOptionValues(cfg)
	if len(values) != 4 {
		t.Fatalf("values len = %d, want 4", len(values))
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

	parentReviewValue, ok := values[keyExecutionParentReviewEnabled]
	if !ok || parentReviewValue.Bool == nil || *parentReviewValue.Bool != true {
		t.Fatalf("parent review value = %#v, want true", parentReviewValue)
	}
	if parentReviewValue.Int != nil {
		t.Fatalf("parent review value int = %v, want nil", parentReviewValue.Int)
	}

	planningValue, ok := values[keyPlanningMaxPlanAutoRefinePasses]
	if !ok || planningValue.Int == nil || *planningValue.Int != 2 {
		t.Fatalf("planning value = %#v, want 2", planningValue)
	}
	if planningValue.Bool != nil {
		t.Fatalf("planning value bool = %v, want nil", planningValue.Bool)
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":20},"planning":{"maxPlanAutoRefinePasses":3},"execution":{"parentReviewEnabled":true}}`), 0o644); err != nil {
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

	planningValue, ok := global.Values[keyPlanningMaxPlanAutoRefinePasses]
	if !ok || planningValue.Int == nil || *planningValue.Int != 3 {
		t.Fatalf("global planning value = %#v, want 3", planningValue)
	}
	if _, ok := global.Values[keyExecutionStopAfterEachTask]; ok {
		t.Fatalf("expected global stop after each task to be unset")
	}
	parentReviewValue, ok := global.Values[keyExecutionParentReviewEnabled]
	if !ok || parentReviewValue.Bool == nil || *parentReviewValue.Bool != true {
		t.Fatalf("global parent review value = %#v, want true", parentReviewValue)
	}
}

func TestSaveConfigValuesWritesFile(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")

	run := 8
	maxRefinePasses := 2
	stop := false
	parentReview := true
	values := map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds: {Int: &run},
		keyPlanningMaxPlanAutoRefinePasses:  {Int: &maxRefinePasses},
		keyExecutionStopAfterEachTask:       {Bool: &stop},
		keyExecutionParentReviewEnabled:     {Bool: &parentReview},
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
	if cfg.Planning == nil || cfg.Planning.MaxPlanAutoRefinePasses == nil || *cfg.Planning.MaxPlanAutoRefinePasses != 2 {
		t.Fatalf("maxPlanAutoRefinePasses = %#v, want 2", cfg.Planning)
	}
	if cfg.Execution == nil || cfg.Execution.StopAfterEachTask == nil || *cfg.Execution.StopAfterEachTask != false {
		t.Fatalf("stop after each task = %#v, want false", cfg.Execution)
	}
	if cfg.Execution.ParentReviewEnabled == nil || *cfg.Execution.ParentReviewEnabled != true {
		t.Fatalf("parent review enabled = %#v, want true", cfg.Execution)
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
	maxRefinePasses := 1
	stop := false
	parentReview := false
	values := map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds: {Int: &run},
		keyPlanningMaxPlanAutoRefinePasses:  {Int: &maxRefinePasses},
		keyExecutionStopAfterEachTask:       {Bool: &stop},
		keyExecutionParentReviewEnabled:     {Bool: &parentReview},
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
	if cfg.Planning == nil || cfg.Planning.MaxPlanAutoRefinePasses == nil || *cfg.Planning.MaxPlanAutoRefinePasses != 1 {
		t.Fatalf("maxPlanAutoRefinePasses = %#v, want 1", cfg.Planning)
	}
	if cfg.Execution == nil || cfg.Execution.StopAfterEachTask == nil || *cfg.Execution.StopAfterEachTask != false {
		t.Fatalf("stop after each task = %#v, want false", cfg.Execution)
	}
	if cfg.Execution.ParentReviewEnabled == nil || *cfg.Execution.ParentReviewEnabled != false {
		t.Fatalf("parent review enabled = %#v, want false", cfg.Execution)
	}
}

func TestSaveConfigValuesRoundTripPreservesExistingKeys(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, ".blackbird", "config.json")

	run := 11
	plan := 6
	refinePasses := 3
	stop := true
	parentReview := false
	values := map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds:  {Int: &run},
		keyTuiPlanDataRefreshIntervalSeconds: {Int: &plan},
		keyPlanningMaxPlanAutoRefinePasses:   {Int: &refinePasses},
		keyExecutionStopAfterEachTask:        {Bool: &stop},
		keyExecutionParentReviewEnabled:      {Bool: &parentReview},
	}

	if err := SaveConfigValues(path, values); err != nil {
		t.Fatalf("save config values: %v", err)
	}

	cfg, present, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !present {
		t.Fatalf("expected config file to be present")
	}

	roundTripValues := RawOptionValues(cfg)
	if got := roundTripValues[keyTuiRunDataRefreshIntervalSeconds]; got.Int == nil || *got.Int != run {
		t.Fatalf("round-trip run value = %#v, want %d", got, run)
	}
	if got := roundTripValues[keyTuiPlanDataRefreshIntervalSeconds]; got.Int == nil || *got.Int != plan {
		t.Fatalf("round-trip plan value = %#v, want %d", got, plan)
	}
	if got := roundTripValues[keyPlanningMaxPlanAutoRefinePasses]; got.Int == nil || *got.Int != refinePasses {
		t.Fatalf("round-trip planning value = %#v, want %d", got, refinePasses)
	}
	if got := roundTripValues[keyExecutionStopAfterEachTask]; got.Bool == nil || *got.Bool != stop {
		t.Fatalf("round-trip stop value = %#v, want %v", got, stop)
	}
	if got := roundTripValues[keyExecutionParentReviewEnabled]; got.Bool == nil || *got.Bool != parentReview {
		t.Fatalf("round-trip parent review value = %#v, want %v", got, parentReview)
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
