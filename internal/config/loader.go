package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var userHomeDir = os.UserHomeDir

func LoadGlobalConfig() (RawConfig, bool, error) {
	home, err := userHomeDir()
	if err != nil || home == "" {
		return RawConfig{}, false, nil
	}

	path := filepath.Join(home, ".blackbird", "config.json")
	return loadConfigFile(path)
}

func LoadProjectConfig(projectRoot string) (RawConfig, bool, error) {
	if projectRoot == "" {
		return RawConfig{}, false, nil
	}

	path := filepath.Join(projectRoot, ".blackbird", "config.json")
	return loadConfigFile(path)
}

// LoadConfig reads global and project configs and returns the resolved config.
// Precedence per key: project > global > defaults.
func LoadConfig(projectRoot string) (ResolvedConfig, error) {
	globalCfg, _, err := LoadGlobalConfig()
	if err != nil {
		return ResolvedConfig{}, err
	}
	projectCfg, _, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return ResolvedConfig{}, err
	}
	return ResolveConfig(projectCfg, globalCfg), nil
}

func loadConfigFile(path string) (RawConfig, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RawConfig{}, false, nil
		}
		return RawConfig{}, false, fmt.Errorf("read config %s: %w", path, err)
	}

	dec := json.NewDecoder(bytes.NewReader(b))

	var cfg RawConfig
	if err := dec.Decode(&cfg); err != nil {
		return RawConfig{}, false, nil
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return RawConfig{}, false, nil
	}
	if !isSupportedSchemaVersion(cfg.SchemaVersion) {
		return RawConfig{}, false, nil
	}

	return cfg, true, nil
}

func isSupportedSchemaVersion(version *int) bool {
	if version == nil {
		return true
	}
	return *version == SchemaVersion
}
