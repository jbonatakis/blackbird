package config

const (
	SchemaVersion = 1

	DefaultRunDataRefreshIntervalSeconds  = 5
	DefaultPlanDataRefreshIntervalSeconds = 5
	DefaultStopAfterEachTask              = false

	MinRefreshIntervalSeconds = 1
	MaxRefreshIntervalSeconds = 300
)

type RawConfig struct {
	SchemaVersion *int          `json:"schemaVersion,omitempty"`
	TUI           *RawTUI       `json:"tui,omitempty"`
	Execution     *RawExecution `json:"execution,omitempty"`
}

type RawTUI struct {
	RunDataRefreshIntervalSeconds  *int `json:"runDataRefreshIntervalSeconds,omitempty"`
	PlanDataRefreshIntervalSeconds *int `json:"planDataRefreshIntervalSeconds,omitempty"`
}

type RawExecution struct {
	StopAfterEachTask *bool `json:"stopAfterEachTask,omitempty"`
}

type ResolvedConfig struct {
	SchemaVersion int               `json:"schemaVersion"`
	TUI           ResolvedTUI       `json:"tui"`
	Execution     ResolvedExecution `json:"execution"`
}

type ResolvedTUI struct {
	RunDataRefreshIntervalSeconds  int `json:"runDataRefreshIntervalSeconds"`
	PlanDataRefreshIntervalSeconds int `json:"planDataRefreshIntervalSeconds"`
}

type ResolvedExecution struct {
	StopAfterEachTask bool `json:"stopAfterEachTask"`
}

func DefaultResolvedConfig() ResolvedConfig {
	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  DefaultRunDataRefreshIntervalSeconds,
			PlanDataRefreshIntervalSeconds: DefaultPlanDataRefreshIntervalSeconds,
		},
		Execution: ResolvedExecution{
			StopAfterEachTask: DefaultStopAfterEachTask,
		},
	}
}
