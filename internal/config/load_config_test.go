package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMergesGlobalAndProject(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":20,"planDataRefreshIntervalSeconds":8},"planning":{"maxPlanAutoRefinePasses":2},"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("run interval = %d, want 12", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 8 {
		t.Fatalf("plan interval = %d, want 8", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != 2 {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want 2", resolved.Planning.MaxPlanAutoRefinePasses)
	}
	if resolved.Execution.StopAfterEachTask != true {
		t.Fatalf("stopAfterEachTask = %v, want true", resolved.Execution.StopAfterEachTask)
	}
	if resolved.Execution.ParentReviewEnabled != false {
		t.Fatalf("parentReviewEnabled = %v, want false", resolved.Execution.ParentReviewEnabled)
	}
}

func TestLoadConfigProjectOverridesGlobal(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":30,"planDataRefreshIntervalSeconds":25},"planning":{"maxPlanAutoRefinePasses":1},"execution":{"stopAfterEachTask":true,"parentReviewEnabled":true}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12,"planDataRefreshIntervalSeconds":9},"planning":{"maxPlanAutoRefinePasses":3},"execution":{"stopAfterEachTask":false,"parentReviewEnabled":false}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("run interval = %d, want 12", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 9 {
		t.Fatalf("plan interval = %d, want 9", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != 3 {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want 3", resolved.Planning.MaxPlanAutoRefinePasses)
	}
	if resolved.Execution.StopAfterEachTask != false {
		t.Fatalf("stopAfterEachTask = %v, want false", resolved.Execution.StopAfterEachTask)
	}
	if resolved.Execution.ParentReviewEnabled != false {
		t.Fatalf("parentReviewEnabled = %v, want false", resolved.Execution.ParentReviewEnabled)
	}
}

func TestLoadConfigFallsBackToGlobalWhenProjectMissing(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":18,"planDataRefreshIntervalSeconds":7},"planning":{"maxPlanAutoRefinePasses":0}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 18 {
		t.Fatalf("run interval = %d, want 18", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 7 {
		t.Fatalf("plan interval = %d, want 7", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != 0 {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want 0", resolved.Planning.MaxPlanAutoRefinePasses)
	}
}

func TestLoadConfigDefaultsWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return homeDir, nil
	})
	t.Cleanup(restore)

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != DefaultRunDataRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, DefaultRunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != DefaultPlanDataRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, DefaultPlanDataRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != DefaultMaxPlanAutoRefinePasses {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want %d", resolved.Planning.MaxPlanAutoRefinePasses, DefaultMaxPlanAutoRefinePasses)
	}
	if resolved.Execution.StopAfterEachTask != DefaultStopAfterEachTask {
		t.Fatalf("stopAfterEachTask = %v, want %v", resolved.Execution.StopAfterEachTask, DefaultStopAfterEachTask)
	}
	if resolved.Execution.ParentReviewEnabled != DefaultParentReviewEnabled {
		t.Fatalf("parentReviewEnabled = %v, want %v", resolved.Execution.ParentReviewEnabled, DefaultParentReviewEnabled)
	}
}

func TestLoadConfigParentReviewEnabledGlobalOnly(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"execution":{"parentReviewEnabled":true}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.Execution.ParentReviewEnabled != true {
		t.Fatalf("parentReviewEnabled = %v, want true", resolved.Execution.ParentReviewEnabled)
	}
}

func TestLoadConfigParentReviewEnabledMissingInBothLayersDefaultsFalse(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":true}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"execution":{"stopAfterEachTask":false}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.Execution.ParentReviewEnabled != false {
		t.Fatalf("parentReviewEnabled = %v, want false", resolved.Execution.ParentReviewEnabled)
	}
}

func TestLoadConfigUsesGlobalWhenProjectRootEmpty(t *testing.T) {
	homeDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return homeDir, nil
	})
	t.Cleanup(restore)

	globalPath := filepath.Join(homeDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
		t.Fatalf("mkdir global: %v", err)
	}
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":7}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	resolved, err := LoadConfig("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 7 {
		t.Fatalf("run interval = %d, want 7", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsGlobalWhenHomeDirErrors(t *testing.T) {
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return "", errors.New("no home")
	})
	t.Cleanup(restore)

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":11}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 11 {
		t.Fatalf("run interval = %d, want 11", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsGlobalWhenHomeDirEmpty(t *testing.T) {
	projectDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return "", nil
	})
	t.Cleanup(restore)

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":6}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 6 {
		t.Fatalf("plan interval = %d, want 6", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsInvalidProjectConfig(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":21}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 21 {
		t.Fatalf("run interval = %d, want 21", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsUnsupportedProjectSchema(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":13}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":99,"tui":{"planDataRefreshIntervalSeconds":5}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 13 {
		t.Fatalf("plan interval = %d, want 13", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsInvalidGlobalConfig(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":1`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":14}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.RunDataRefreshIntervalSeconds != 14 {
		t.Fatalf("run interval = %d, want 14", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
}

func TestLoadConfigSkipsUnsupportedGlobalSchema(t *testing.T) {
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
	if err := os.WriteFile(globalPath, []byte(`{"schemaVersion":0,"tui":{"runDataRefreshIntervalSeconds":3}}`), 0o644); err != nil {
		t.Fatalf("write global config: %v", err)
	}

	projectPath := filepath.Join(projectDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatalf("mkdir project: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":4}}`), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	resolved, err := LoadConfig(projectDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 4 {
		t.Fatalf("plan interval = %d, want 4", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
}
