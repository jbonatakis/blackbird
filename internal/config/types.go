package config

const (
	SchemaVersion = 1

	DefaultRunDataRefreshIntervalSeconds  = 5
	DefaultPlanDataRefreshIntervalSeconds = 5
	DefaultMaxPlanAutoRefinePasses        = 1
	DefaultStopAfterEachTask              = false

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
	RunDataRefreshIntervalSeconds  *int `json:"runDataRefreshIntervalSeconds,omitempty"`
	PlanDataRefreshIntervalSeconds *int `json:"planDataRefreshIntervalSeconds,omitempty"`
}

type RawExecution struct {
	StopAfterEachTask *bool `json:"stopAfterEachTask,omitempty"`
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
	RunDataRefreshIntervalSeconds  int `json:"runDataRefreshIntervalSeconds"`
	PlanDataRefreshIntervalSeconds int `json:"planDataRefreshIntervalSeconds"`
}

type ResolvedExecution struct {
	StopAfterEachTask bool `json:"stopAfterEachTask"`
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
		},
		Planning: ResolvedPlanning{
			MaxPlanAutoRefinePasses: DefaultMaxPlanAutoRefinePasses,
		},
		Execution: ResolvedExecution{
			StopAfterEachTask: DefaultStopAfterEachTask,
		},
	}
}
