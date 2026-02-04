package config

type OptionType string

const (
	OptionTypeBool OptionType = "bool"
	OptionTypeInt  OptionType = "int"
)

type IntBounds struct {
	Min int
	Max int
}

type OptionMetadata struct {
	KeyPath     string
	DisplayName string
	Type        OptionType
	DefaultInt  int
	DefaultBool bool
	Bounds      *IntBounds
	Description string
}

// OptionRegistry returns the known config options in display order.
func OptionRegistry() []OptionMetadata {
	defaults := DefaultResolvedConfig()

	return []OptionMetadata{
		newIntOption(
			"tui.runDataRefreshIntervalSeconds",
			"TUI Run Refresh (seconds)",
			defaults.TUI.RunDataRefreshIntervalSeconds,
			MinRefreshIntervalSeconds,
			MaxRefreshIntervalSeconds,
			"Run data refresh interval in seconds",
		),
		newIntOption(
			"tui.planDataRefreshIntervalSeconds",
			"TUI Plan Refresh (seconds)",
			defaults.TUI.PlanDataRefreshIntervalSeconds,
			MinRefreshIntervalSeconds,
			MaxRefreshIntervalSeconds,
			"Plan data refresh interval in seconds",
		),
		newBoolOption(
			"execution.stopAfterEachTask",
			"Execution Stop After Each Task",
			defaults.Execution.StopAfterEachTask,
			"Pause execution for review after each task",
		),
	}
}

func newIntOption(keyPath string, displayName string, defaultValue int, min int, max int, description string) OptionMetadata {
	return OptionMetadata{
		KeyPath:     keyPath,
		DisplayName: displayName,
		Type:        OptionTypeInt,
		DefaultInt:  defaultValue,
		Bounds: &IntBounds{
			Min: min,
			Max: max,
		},
		Description: description,
	}
}

func newBoolOption(keyPath string, displayName string, defaultValue bool, description string) OptionMetadata {
	return OptionMetadata{
		KeyPath:     keyPath,
		DisplayName: displayName,
		Type:        OptionTypeBool,
		DefaultBool: defaultValue,
		Bounds:      nil,
		Description: description,
	}
}
