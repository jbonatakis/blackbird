package tui

import "github.com/jbonatakis/blackbird/internal/config"

type SettingsState struct {
	ProjectRoot string
	Options     []config.OptionMetadata
	Resolved    config.ResolvedConfig
	Resolution  config.SettingsResolution
	Selected    int
	Column      SettingsColumn
	Editing     bool
	EditValue   string
	SaveErr     error
	Err         error
}

func NewSettingsState(projectRoot string, resolved config.ResolvedConfig) SettingsState {
	resolution, err := config.ResolveSettings(projectRoot)
	if err != nil {
		resolution = settingsResolutionFromResolved(resolved)
	}

	return SettingsState{
		ProjectRoot: projectRoot,
		Options:     config.OptionRegistry(),
		Resolved:    resolved,
		Resolution:  resolution,
		Selected:    0,
		Column:      SettingsColumnLocal,
		Err:         err,
	}
}

func defaultSettingsState() SettingsState {
	resolved := config.DefaultResolvedConfig()
	return SettingsState{
		Options:    config.OptionRegistry(),
		Resolved:   resolved,
		Resolution: settingsResolutionFromResolved(resolved),
		Selected:   0,
		Column:     SettingsColumnLocal,
	}
}

func settingsResolutionFromResolved(resolved config.ResolvedConfig) config.SettingsResolution {
	resolvedValues := config.ResolvedOptionValues(resolved)
	applied := map[string]config.AppliedOption{}

	for _, option := range config.OptionRegistry() {
		value, ok := resolvedValues[option.KeyPath]
		if !ok {
			value = settingsDefaultOptionValue(option)
		}
		applied[option.KeyPath] = config.AppliedOption{
			Value:  value,
			Source: config.ConfigSourceDefault,
		}
	}

	return config.SettingsResolution{
		Project: config.SettingsLayer{Values: map[string]config.RawOptionValue{}},
		Global:  config.SettingsLayer{Values: map[string]config.RawOptionValue{}},
		Applied: applied,
	}
}

func settingsDefaultOptionValue(option config.OptionMetadata) config.RawOptionValue {
	if option.Type == config.OptionTypeInt {
		value := option.DefaultInt
		return config.RawOptionValue{Int: &value}
	}
	value := option.DefaultBool
	return config.RawOptionValue{Bool: &value}
}
