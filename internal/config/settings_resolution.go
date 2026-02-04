package config

import "path/filepath"

type ConfigSource string

const (
	ConfigSourceLocal   ConfigSource = "local"
	ConfigSourceGlobal  ConfigSource = "global"
	ConfigSourceDefault ConfigSource = "default"
)

type LayerWarningKind string

const (
	LayerWarningInvalidJSON       LayerWarningKind = "invalid_json"
	LayerWarningUnsupportedSchema LayerWarningKind = "unsupported_schema"
)

type OptionWarningKind string

const (
	OptionWarningOutOfRange OptionWarningKind = "out_of_range"
)

type LayerWarning struct {
	Source ConfigSource
	Kind   LayerWarningKind
}

type OptionWarning struct {
	Source     ConfigSource
	KeyPath    string
	Kind       OptionWarningKind
	ClampedInt *int
}

type AppliedOption struct {
	Value  RawOptionValue
	Source ConfigSource
}

type SettingsLayer struct {
	Available bool
	Path      string
	Present   bool
	Values    map[string]RawOptionValue
}

type SettingsResolution struct {
	Project        SettingsLayer
	Global         SettingsLayer
	Applied        map[string]AppliedOption
	OptionWarnings []OptionWarning
	LayerWarnings  []LayerWarning
}

// ResolveSettings loads local/global config values and computes applied values with warnings.
func ResolveSettings(projectRoot string) (SettingsResolution, error) {
	projectLayer := SettingsLayer{
		Available: projectRoot != "",
		Path:      projectConfigPath(projectRoot),
		Values:    map[string]RawOptionValue{},
	}
	globalLayer := SettingsLayer{
		Available: false,
		Path:      "",
		Values:    map[string]RawOptionValue{},
	}

	var layerWarnings []LayerWarning
	var projectRaw RawConfig
	var globalRaw RawConfig

	if projectLayer.Available {
		cfg, present, warningKind, err := loadConfigFileDetailed(projectLayer.Path)
		if err != nil {
			return SettingsResolution{}, err
		}
		if warningKind != nil {
			layerWarnings = append(layerWarnings, LayerWarning{
				Source: ConfigSourceLocal,
				Kind:   *warningKind,
			})
		} else if present {
			projectLayer.Present = true
			projectLayer.Values = RawOptionValues(cfg)
			projectRaw = cfg
		}
	}

	globalPath, globalAvailable := globalConfigPath()
	globalLayer.Available = globalAvailable
	globalLayer.Path = globalPath
	if globalAvailable {
		cfg, present, warningKind, err := loadConfigFileDetailed(globalPath)
		if err != nil {
			return SettingsResolution{}, err
		}
		if warningKind != nil {
			layerWarnings = append(layerWarnings, LayerWarning{
				Source: ConfigSourceGlobal,
				Kind:   *warningKind,
			})
		} else if present {
			globalLayer.Present = true
			globalLayer.Values = RawOptionValues(cfg)
			globalRaw = cfg
		}
	}

	resolved := ResolveConfig(projectRaw, globalRaw)
	resolvedValues := ResolvedOptionValues(resolved)

	applied := map[string]AppliedOption{}
	for _, option := range OptionRegistry() {
		key := option.KeyPath
		value, ok := resolvedValues[key]
		if !ok {
			value = defaultOptionValue(option)
		}
		source := ConfigSourceDefault
		if _, ok := projectLayer.Values[key]; ok {
			source = ConfigSourceLocal
		} else if _, ok := globalLayer.Values[key]; ok {
			source = ConfigSourceGlobal
		}
		applied[key] = AppliedOption{
			Value:  value,
			Source: source,
		}
	}

	optionWarnings := append(
		collectOutOfRangeWarnings(ConfigSourceLocal, projectLayer.Values),
		collectOutOfRangeWarnings(ConfigSourceGlobal, globalLayer.Values)...,
	)

	return SettingsResolution{
		Project:        projectLayer,
		Global:         globalLayer,
		Applied:        applied,
		OptionWarnings: optionWarnings,
		LayerWarnings:  layerWarnings,
	}, nil
}

func ResolvedOptionValues(cfg ResolvedConfig) map[string]RawOptionValue {
	return map[string]RawOptionValue{
		keyTuiRunDataRefreshIntervalSeconds: {
			Int: copyInt(cfg.TUI.RunDataRefreshIntervalSeconds),
		},
		keyTuiPlanDataRefreshIntervalSeconds: {
			Int: copyInt(cfg.TUI.PlanDataRefreshIntervalSeconds),
		},
		keyExecutionStopAfterEachTask: {
			Bool: copyBool(cfg.Execution.StopAfterEachTask),
		},
	}
}

func defaultOptionValue(option OptionMetadata) RawOptionValue {
	if option.Type == OptionTypeInt {
		value := option.DefaultInt
		return RawOptionValue{Int: &value}
	}
	value := option.DefaultBool
	return RawOptionValue{Bool: &value}
}

func collectOutOfRangeWarnings(source ConfigSource, values map[string]RawOptionValue) []OptionWarning {
	warnings := []OptionWarning{}
	for key, value := range values {
		if value.Int == nil {
			continue
		}
		clamped := clampIntForKey(key, *value.Int)
		if clamped != *value.Int {
			warnings = append(warnings, OptionWarning{
				Source:     source,
				KeyPath:    key,
				Kind:       OptionWarningOutOfRange,
				ClampedInt: copyInt(clamped),
			})
		}
	}
	return warnings
}

func clampIntForKey(key string, value int) int {
	switch key {
	case keyTuiRunDataRefreshIntervalSeconds, keyTuiPlanDataRefreshIntervalSeconds:
		return clampInterval(value)
	default:
		return value
	}
}

func projectConfigPath(projectRoot string) string {
	if projectRoot == "" {
		return ""
	}
	return filepath.Join(projectRoot, ".blackbird", "config.json")
}

func globalConfigPath() (string, bool) {
	home, err := userHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".blackbird", "config.json"), true
}

func warningPtr(kind LayerWarningKind) *LayerWarningKind {
	return &kind
}
