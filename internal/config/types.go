package config

const (
	SchemaVersion = 1

	ThemeIDBlackbird    = "blackbird"
	ThemeIDHighContrast = "high-contrast"
	DefaultTUITheme     = ThemeIDBlackbird

	DefaultRunDataRefreshIntervalSeconds  = 5
	DefaultPlanDataRefreshIntervalSeconds = 5
	DefaultMaxPlanAutoRefinePasses        = 1
	DefaultStopAfterEachTask              = false
	DefaultParentReviewEnabled            = false

	MinRefreshIntervalSeconds = 1
	MaxRefreshIntervalSeconds = 300
	MinPlanAutoRefinePasses   = 0
	MaxPlanAutoRefinePasses   = 3
)

type RawConfig struct {
	SchemaVersion *int          `json:"schemaVersion,omitempty"`
	TUI           *RawTUI       `json:"tui,omitempty"`
	Planning      *RawPlanning  `json:"planning,omitempty"`
	Execution     *RawExecution `json:"execution,omitempty"`
}

type RawTUI struct {
	RunDataRefreshIntervalSeconds  *int    `json:"runDataRefreshIntervalSeconds,omitempty"`
	PlanDataRefreshIntervalSeconds *int    `json:"planDataRefreshIntervalSeconds,omitempty"`
	Theme                          *string `json:"theme,omitempty"`
}

type RawExecution struct {
	StopAfterEachTask   *bool `json:"stopAfterEachTask,omitempty"`
	ParentReviewEnabled *bool `json:"parentReviewEnabled,omitempty"`
}

type RawPlanning struct {
	MaxPlanAutoRefinePasses *int `json:"maxPlanAutoRefinePasses,omitempty"`
}

type ResolvedConfig struct {
	SchemaVersion int               `json:"schemaVersion"`
	TUI           ResolvedTUI       `json:"tui"`
	Planning      ResolvedPlanning  `json:"planning"`
	Execution     ResolvedExecution `json:"execution"`
}

type ResolvedTUI struct {
	RunDataRefreshIntervalSeconds  int    `json:"runDataRefreshIntervalSeconds"`
	PlanDataRefreshIntervalSeconds int    `json:"planDataRefreshIntervalSeconds"`
	Theme                          string `json:"theme"`
}

type ResolvedExecution struct {
	StopAfterEachTask   bool `json:"stopAfterEachTask"`
	ParentReviewEnabled bool `json:"parentReviewEnabled"`
}

type ResolvedPlanning struct {
	MaxPlanAutoRefinePasses int `json:"maxPlanAutoRefinePasses"`
}

func DefaultResolvedConfig() ResolvedConfig {
	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  DefaultRunDataRefreshIntervalSeconds,
			PlanDataRefreshIntervalSeconds: DefaultPlanDataRefreshIntervalSeconds,
			Theme:                          DefaultTUITheme,
		},
		Planning: ResolvedPlanning{
			MaxPlanAutoRefinePasses: DefaultMaxPlanAutoRefinePasses,
		},
		Execution: ResolvedExecution{
			StopAfterEachTask:   DefaultStopAfterEachTask,
			ParentReviewEnabled: DefaultParentReviewEnabled,
		},
	}
}

// BuiltInThemeIDs returns the known theme IDs in deterministic alphabetical order.
func BuiltInThemeIDs() []string {
	return []string{
		ThemeIDBlackbird,
		ThemeIDHighContrast,
	}
}

func isBuiltInThemeID(themeID string) bool {
	switch themeID {
	case ThemeIDBlackbird, ThemeIDHighContrast:
		return true
	default:
		return false
	}
}
