package config

type OptionType string

const (
	OptionTypeBool        OptionType = "bool"
	OptionTypeInt         OptionType = "int"
	OptionTypeCategorical OptionType = "categorical"
)

type IntBounds struct {
	Min int
	Max int
}

type OptionMetadata struct {
	KeyPath       string
	DisplayName   string
	Type          OptionType
	DefaultInt    int
	DefaultBool   bool
	DefaultString string
	AllowedValues []string
	Bounds        *IntBounds
	Description   string
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
		newCategoricalOption(
			"tui.theme",
			"TUI Theme",
			defaults.TUI.Theme,
			BuiltInThemeIDs(),
			"TUI color theme",
		),
		newIntOption(
			"planning.maxPlanAutoRefinePasses",
			"Planning Max Auto-Refine Passes",
			defaults.Planning.MaxPlanAutoRefinePasses,
			MinPlanAutoRefinePasses,
			MaxPlanAutoRefinePasses,
			"Maximum automatic refine passes when planning",
		),
		newBoolOption(
			"execution.stopAfterEachTask",
			"Execution Stop After Each Task",
			defaults.Execution.StopAfterEachTask,
			"Pause execution for review after each task",
		),
		newBoolOption(
			"execution.parentReviewEnabled",
			"Execution Parent Review Gate",
			defaults.Execution.ParentReviewEnabled,
			"Run parent-review checks after successful child tasks",
		),
	}
}

func newIntOption(keyPath string, displayName string, defaultValue int, min int, max int, description string) OptionMetadata {
	return OptionMetadata{
		KeyPath:       keyPath,
		DisplayName:   displayName,
		Type:          OptionTypeInt,
		DefaultInt:    defaultValue,
		AllowedValues: nil,
		Bounds: &IntBounds{
			Min: min,
			Max: max,
		},
		Description: description,
	}
}

func newBoolOption(keyPath string, displayName string, defaultValue bool, description string) OptionMetadata {
	return OptionMetadata{
		KeyPath:       keyPath,
		DisplayName:   displayName,
		Type:          OptionTypeBool,
		DefaultBool:   defaultValue,
		AllowedValues: nil,
		Bounds:        nil,
		Description:   description,
	}
}

func newCategoricalOption(
	keyPath string,
	displayName string,
	defaultValue string,
	allowedValues []string,
	description string,
) OptionMetadata {
	copied := append([]string(nil), allowedValues...)
	return OptionMetadata{
		KeyPath:       keyPath,
		DisplayName:   displayName,
		Type:          OptionTypeCategorical,
		DefaultString: defaultValue,
		AllowedValues: copied,
		Bounds:        nil,
		Description:   description,
	}
}
