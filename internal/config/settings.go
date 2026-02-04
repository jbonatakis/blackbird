package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	keyTuiRunDataRefreshIntervalSeconds  = "tui.runDataRefreshIntervalSeconds"
	keyTuiPlanDataRefreshIntervalSeconds = "tui.planDataRefreshIntervalSeconds"
	keyExecutionStopAfterEachTask        = "execution.stopAfterEachTask"
)

type RawOptionValue struct {
	Int  *int
	Bool *bool
}

type LayerOptionValues struct {
	Present bool
	Values  map[string]RawOptionValue
}

// LoadLayerOptionValues reads project and global configs and returns per-option raw values
// (project first, then global).
func LoadLayerOptionValues(projectRoot string) (LayerOptionValues, LayerOptionValues, error) {
	globalCfg, globalPresent, err := LoadGlobalConfig()
	if err != nil {
		return LayerOptionValues{}, LayerOptionValues{}, err
	}
	projectCfg, projectPresent, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return LayerOptionValues{}, LayerOptionValues{}, err
	}

	project := LayerOptionValues{
		Present: projectPresent,
		Values:  RawOptionValues(projectCfg),
	}
	global := LayerOptionValues{
		Present: globalPresent,
		Values:  RawOptionValues(globalCfg),
	}

	return project, global, nil
}

// RawOptionValues extracts known raw option values from a config layer.
func RawOptionValues(cfg RawConfig) map[string]RawOptionValue {
	values := map[string]RawOptionValue{}

	if cfg.TUI != nil {
		if cfg.TUI.RunDataRefreshIntervalSeconds != nil {
			values[keyTuiRunDataRefreshIntervalSeconds] = RawOptionValue{
				Int: copyInt(*cfg.TUI.RunDataRefreshIntervalSeconds),
			}
		}
		if cfg.TUI.PlanDataRefreshIntervalSeconds != nil {
			values[keyTuiPlanDataRefreshIntervalSeconds] = RawOptionValue{
				Int: copyInt(*cfg.TUI.PlanDataRefreshIntervalSeconds),
			}
		}
	}

	if cfg.Execution != nil {
		if cfg.Execution.StopAfterEachTask != nil {
			values[keyExecutionStopAfterEachTask] = RawOptionValue{
				Bool: copyBool(*cfg.Execution.StopAfterEachTask),
			}
		}
	}

	return values
}

// SaveConfigValues writes the provided raw option values to disk.
// The file includes schemaVersion and only set keys; empty layers remove the file.
func SaveConfigValues(path string, values map[string]RawOptionValue) error {
	if path == "" {
		return errors.New("config path is empty")
	}

	cfg, hasValues, err := buildRawConfig(values)
	if err != nil {
		return err
	}
	if !hasValues {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove config %s: %w", path, err)
		}
		return nil
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := atomicWriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func buildRawConfig(values map[string]RawOptionValue) (RawConfig, bool, error) {
	var cfg RawConfig
	var tui RawTUI
	var exec RawExecution
	var hasTUI bool
	var hasExec bool

	for key, value := range values {
		if value.Int != nil && value.Bool != nil {
			return RawConfig{}, false, fmt.Errorf("config key %q has both int and bool values", key)
		}
		if value.Int == nil && value.Bool == nil {
			continue
		}

		switch key {
		case keyTuiRunDataRefreshIntervalSeconds:
			if value.Int == nil {
				return RawConfig{}, false, fmt.Errorf("config key %q expects int value", key)
			}
			v := *value.Int
			tui.RunDataRefreshIntervalSeconds = &v
			hasTUI = true
		case keyTuiPlanDataRefreshIntervalSeconds:
			if value.Int == nil {
				return RawConfig{}, false, fmt.Errorf("config key %q expects int value", key)
			}
			v := *value.Int
			tui.PlanDataRefreshIntervalSeconds = &v
			hasTUI = true
		case keyExecutionStopAfterEachTask:
			if value.Bool == nil {
				return RawConfig{}, false, fmt.Errorf("config key %q expects bool value", key)
			}
			v := *value.Bool
			exec.StopAfterEachTask = &v
			hasExec = true
		default:
			return RawConfig{}, false, fmt.Errorf("unknown config key %q", key)
		}
	}

	if !hasTUI && !hasExec {
		return RawConfig{}, false, nil
	}

	if hasTUI {
		cfg.TUI = &tui
	}
	if hasExec {
		cfg.Execution = &exec
	}
	version := SchemaVersion
	cfg.SchemaVersion = &version

	return cfg, true, nil
}

func copyInt(v int) *int {
	return &v
}

func copyBool(v bool) *bool {
	return &v
}
