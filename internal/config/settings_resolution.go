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
	OptionWarningOutOfRange   OptionWarningKind = "out_of_range"
	OptionWarningInvalidValue OptionWarningKind = "invalid_value"
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
	RawString  *string
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
	options := OptionRegistry()

	applied := map[string]AppliedOption{}
	for _, option := range options {
		key := option.KeyPath
		value, ok := resolvedValues[key]
		if !ok {
			value = defaultOptionValue(option)
		}
		source := appliedSourceForOption(option, projectLayer.Values, globalLayer.Values)
		applied[key] = AppliedOption{
			Value:  value,
			Source: source,
		}
	}

	optionsByKey := map[string]OptionMetadata{}
	for _, option := range options {
		optionsByKey[option.KeyPath] = option
	}
	optionWarnings := append(
		collectOptionWarnings(ConfigSourceLocal, projectLayer.Values, optionsByKey),
		collectOptionWarnings(ConfigSourceGlobal, globalLayer.Values, optionsByKey)...,
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
		keyTuiTheme: {
			String: copyString(cfg.TUI.Theme),
		},
		keyPlanningMaxPlanAutoRefinePasses: {
			Int: copyInt(cfg.Planning.MaxPlanAutoRefinePasses),
		},
		keyExecutionStopAfterEachTask: {
			Bool: copyBool(cfg.Execution.StopAfterEachTask),
		},
		keyExecutionParentReviewEnabled: {
			Bool: copyBool(cfg.Execution.ParentReviewEnabled),
		},
	}
}

func defaultOptionValue(option OptionMetadata) RawOptionValue {
	switch option.Type {
	case OptionTypeInt:
		value := option.DefaultInt
		return RawOptionValue{Int: &value}
	case OptionTypeBool:
		value := option.DefaultBool
		return RawOptionValue{Bool: &value}
	case OptionTypeCategorical:
		value := option.DefaultString
		return RawOptionValue{String: &value}
	default:
		return RawOptionValue{}
	}
}

func collectOptionWarnings(
	source ConfigSource,
	values map[string]RawOptionValue,
	optionsByKey map[string]OptionMetadata,
) []OptionWarning {
	warnings := []OptionWarning{}
	for key, value := range values {
		option, ok := optionsByKey[key]
		if !ok {
			continue
		}
		if value.Int == nil {
			// Non-int options still may emit warnings, e.g. invalid categorical values.
		} else {
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
		if option.Type == OptionTypeCategorical && value.String != nil && !categoricalValueAllowed(option, *value.String) {
			warnings = append(warnings, OptionWarning{
				Source:    source,
				KeyPath:   key,
				Kind:      OptionWarningInvalidValue,
				RawString: copyString(*value.String),
			})
		}
	}
	return warnings
}

func appliedSourceForOption(
	option OptionMetadata,
	projectValues map[string]RawOptionValue,
	globalValues map[string]RawOptionValue,
) ConfigSource {
	if raw, ok := projectValues[option.KeyPath]; ok && rawOptionValueApplies(option, raw) {
		return ConfigSourceLocal
	}
	if raw, ok := globalValues[option.KeyPath]; ok && rawOptionValueApplies(option, raw) {
		return ConfigSourceGlobal
	}
	return ConfigSourceDefault
}

func rawOptionValueApplies(option OptionMetadata, value RawOptionValue) bool {
	switch option.Type {
	case OptionTypeInt:
		return value.Int != nil
	case OptionTypeBool:
		return value.Bool != nil
	case OptionTypeCategorical:
		return value.String != nil && categoricalValueAllowed(option, *value.String)
	default:
		return false
	}
}

func categoricalValueAllowed(option OptionMetadata, value string) bool {
	if option.Type != OptionTypeCategorical {
		return false
	}
	for _, candidate := range option.AllowedValues {
		if candidate == value {
			return true
		}
	}
	return false
}

func clampIntForKey(key string, value int) int {
	switch key {
	case keyTuiRunDataRefreshIntervalSeconds, keyTuiPlanDataRefreshIntervalSeconds:
		return clampInterval(value)
	case keyPlanningMaxPlanAutoRefinePasses:
		return clampPlanAutoRefinePasses(value)
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
