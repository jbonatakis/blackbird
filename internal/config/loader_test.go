package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGlobalConfigReadsFile(t *testing.T) {
	tempDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return tempDir, nil
	})
	t.Cleanup(restore)

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tui":{"runDataRefreshIntervalSeconds":12}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !present {
		t.Fatalf("expected config to be present")
	}
	if cfg.SchemaVersion == nil || *cfg.SchemaVersion != 1 {
		t.Fatalf("expected schemaVersion 1, got %#v", cfg.SchemaVersion)
	}
	if cfg.TUI == nil || cfg.TUI.RunDataRefreshIntervalSeconds == nil || *cfg.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("expected runDataRefreshIntervalSeconds 12, got %#v", cfg.TUI)
	}
	if cfg.TUI.PlanDataRefreshIntervalSeconds != nil {
		t.Fatalf("expected planDataRefreshIntervalSeconds to be nil, got %#v", cfg.TUI.PlanDataRefreshIntervalSeconds)
	}
}

func TestLoadGlobalConfigMissingFileSkips(t *testing.T) {
	tempDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return tempDir, nil
	})
	t.Cleanup(restore)

	cfg, present, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadGlobalConfigMissingHomeSkips(t *testing.T) {
	restore := overrideUserHomeDir(func() (string, error) {
		return "", errors.New("no home")
	})
	t.Cleanup(restore)

	cfg, present, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadGlobalConfigInvalidJSONSkips(t *testing.T) {
	tempDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return tempDir, nil
	})
	t.Cleanup(restore)

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadGlobalConfigUnsupportedSchemaSkips(t *testing.T) {
	tempDir := t.TempDir()
	restore := overrideUserHomeDir(func() (string, error) {
		return tempDir, nil
	})
	t.Cleanup(restore)

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":99,"tui":{"runDataRefreshIntervalSeconds":12}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadProjectConfigReadsFile(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tui":{"planDataRefreshIntervalSeconds":9}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !present {
		t.Fatalf("expected config to be present")
	}
	if cfg.SchemaVersion == nil || *cfg.SchemaVersion != 1 {
		t.Fatalf("expected schemaVersion 1, got %#v", cfg.SchemaVersion)
	}
	if cfg.TUI == nil || cfg.TUI.PlanDataRefreshIntervalSeconds == nil || *cfg.TUI.PlanDataRefreshIntervalSeconds != 9 {
		t.Fatalf("expected planDataRefreshIntervalSeconds 9, got %#v", cfg.TUI)
	}
	if cfg.TUI.RunDataRefreshIntervalSeconds != nil {
		t.Fatalf("expected runDataRefreshIntervalSeconds to be nil, got %#v", cfg.TUI.RunDataRefreshIntervalSeconds)
	}
}

func TestLoadProjectConfigMissingFileSkips(t *testing.T) {
	tempDir := t.TempDir()

	cfg, present, err := LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadProjectConfigInvalidJSONSkips(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"tui":`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadProjectConfigUnsupportedSchemaSkips(t *testing.T) {
	tempDir := t.TempDir()

	path := filepath.Join(tempDir, ".blackbird", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":0,"tui":{"planDataRefreshIntervalSeconds":9}}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, present, err := LoadProjectConfig(tempDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func TestLoadProjectConfigEmptyRootSkips(t *testing.T) {
	cfg, present, err := LoadProjectConfig("")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if present {
		t.Fatalf("expected config to be missing")
	}
	if cfg != (RawConfig{}) {
		t.Fatalf("expected empty config, got %#v", cfg)
	}
}

func overrideUserHomeDir(fn func() (string, error)) func() {
	orig := userHomeDir
	userHomeDir = fn
	return func() {
		userHomeDir = orig
	}
}
