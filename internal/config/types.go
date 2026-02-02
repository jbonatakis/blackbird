package config

const (
	SchemaVersion = 1

	DefaultRunDataRefreshIntervalSeconds  = 5
	DefaultPlanDataRefreshIntervalSeconds = 5

	MinRefreshIntervalSeconds = 1
	MaxRefreshIntervalSeconds = 300
)

type RawConfig struct {
	SchemaVersion *int    `json:"schemaVersion,omitempty"`
	TUI           *RawTUI `json:"tui,omitempty"`
}

type RawTUI struct {
	RunDataRefreshIntervalSeconds  *int `json:"runDataRefreshIntervalSeconds,omitempty"`
	PlanDataRefreshIntervalSeconds *int `json:"planDataRefreshIntervalSeconds,omitempty"`
}

type ResolvedConfig struct {
	SchemaVersion int         `json:"schemaVersion"`
	TUI           ResolvedTUI `json:"tui"`
}

type ResolvedTUI struct {
	RunDataRefreshIntervalSeconds  int `json:"runDataRefreshIntervalSeconds"`
	PlanDataRefreshIntervalSeconds int `json:"planDataRefreshIntervalSeconds"`
}

func DefaultResolvedConfig() ResolvedConfig {
	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  DefaultRunDataRefreshIntervalSeconds,
			PlanDataRefreshIntervalSeconds: DefaultPlanDataRefreshIntervalSeconds,
		},
	}
}
