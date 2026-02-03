package config

const (
	SchemaVersion = 1

	DefaultRunDataRefreshIntervalSeconds  = 5
	DefaultPlanDataRefreshIntervalSeconds = 5

	MinRefreshIntervalSeconds = 1
	MaxRefreshIntervalSeconds = 300

	MemoryModeOff           = "off"
	MemoryModePassthrough   = "passthrough"
	MemoryModeDeterministic = "deterministic"
	MemoryModeLocal         = "local"
	MemoryModeProvider      = "provider"

	DefaultMemoryMode = MemoryModeDeterministic

	DefaultMemoryProxyListenAddr              = "127.0.0.1:8080"
	DefaultMemoryProxyUpstreamURLCodex        = "https://api.openai.com"
	DefaultMemoryProxyChatGPTUpstreamURLCodex = "https://chatgpt.com"
	DefaultMemoryProxyLossless                = true
	DefaultMemoryTraceRetentionDays           = 14
	DefaultMemoryTraceMaxSizeMB               = 512
	DefaultMemoryBudgetTotalTokens            = 1200
	DefaultMemoryBudgetDecisionsTokens        = 200
	DefaultMemoryBudgetConstraintsTokens      = 150
	DefaultMemoryBudgetImplementedTokens      = 300
	DefaultMemoryBudgetOpenThreadsTokens      = 150
	DefaultMemoryBudgetArtifactPointersTokens = 100

	MinMemoryTraceRetentionDays = 1
	MaxMemoryTraceRetentionDays = 3650
	MinMemoryTraceMaxSizeMB     = 1
	MaxMemoryTraceMaxSizeMB     = 102400
	MinMemoryBudgetTokens       = 0
	MaxMemoryBudgetTokens       = 20000
)

type RawConfig struct {
	SchemaVersion *int       `json:"schemaVersion,omitempty"`
	TUI           *RawTUI    `json:"tui,omitempty"`
	Memory        *RawMemory `json:"memory,omitempty"`
}

type RawTUI struct {
	RunDataRefreshIntervalSeconds  *int `json:"runDataRefreshIntervalSeconds,omitempty"`
	PlanDataRefreshIntervalSeconds *int `json:"planDataRefreshIntervalSeconds,omitempty"`
}

type RawMemory struct {
	Mode      *string             `json:"mode,omitempty"`
	Proxy     *RawMemoryProxy     `json:"proxy,omitempty"`
	Retention *RawMemoryRetention `json:"retention,omitempty"`
	Budgets   *RawMemoryBudgets   `json:"budgets,omitempty"`
}

type RawMemoryProxy struct {
	ListenAddr         *string `json:"listenAddr,omitempty"`
	UpstreamURL        *string `json:"upstreamURL,omitempty"`
	ChatGPTUpstreamURL *string `json:"chatGPTUpstreamURL,omitempty"`
	Lossless           *bool   `json:"lossless,omitempty"`
}

type RawMemoryRetention struct {
	TraceRetentionDays *int `json:"traceRetentionDays,omitempty"`
	TraceMaxSizeMB     *int `json:"traceMaxSizeMB,omitempty"`
}

type RawMemoryBudgets struct {
	TotalTokens            *int `json:"totalTokens,omitempty"`
	DecisionsTokens        *int `json:"decisionsTokens,omitempty"`
	ConstraintsTokens      *int `json:"constraintsTokens,omitempty"`
	ImplementedTokens      *int `json:"implementedTokens,omitempty"`
	OpenThreadsTokens      *int `json:"openThreadsTokens,omitempty"`
	ArtifactPointersTokens *int `json:"artifactPointersTokens,omitempty"`
}

type ResolvedConfig struct {
	SchemaVersion int            `json:"schemaVersion"`
	TUI           ResolvedTUI    `json:"tui"`
	Memory        ResolvedMemory `json:"memory"`
}

type ResolvedTUI struct {
	RunDataRefreshIntervalSeconds  int `json:"runDataRefreshIntervalSeconds"`
	PlanDataRefreshIntervalSeconds int `json:"planDataRefreshIntervalSeconds"`
}

type ResolvedMemory struct {
	Mode      string                  `json:"mode"`
	Proxy     ResolvedMemoryProxy     `json:"proxy"`
	Retention ResolvedMemoryRetention `json:"retention"`
	Budgets   ResolvedMemoryBudgets   `json:"budgets"`
}

type ResolvedMemoryProxy struct {
	ListenAddr         string `json:"listenAddr"`
	UpstreamURL        string `json:"upstreamURL"`
	ChatGPTUpstreamURL string `json:"chatGPTUpstreamURL"`
	Lossless           bool   `json:"lossless"`
}

type ResolvedMemoryRetention struct {
	TraceRetentionDays int `json:"traceRetentionDays"`
	TraceMaxSizeMB     int `json:"traceMaxSizeMB"`
}

type ResolvedMemoryBudgets struct {
	TotalTokens            int `json:"totalTokens"`
	DecisionsTokens        int `json:"decisionsTokens"`
	ConstraintsTokens      int `json:"constraintsTokens"`
	ImplementedTokens      int `json:"implementedTokens"`
	OpenThreadsTokens      int `json:"openThreadsTokens"`
	ArtifactPointersTokens int `json:"artifactPointersTokens"`
}

func DefaultResolvedConfig() ResolvedConfig {
	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  DefaultRunDataRefreshIntervalSeconds,
			PlanDataRefreshIntervalSeconds: DefaultPlanDataRefreshIntervalSeconds,
		},
		Memory: ResolvedMemory{
			Mode: DefaultMemoryMode,
			Proxy: ResolvedMemoryProxy{
				ListenAddr:         DefaultMemoryProxyListenAddr,
				UpstreamURL:        DefaultMemoryProxyUpstreamURLCodex,
				ChatGPTUpstreamURL: DefaultMemoryProxyChatGPTUpstreamURLCodex,
				Lossless:           DefaultMemoryProxyLossless,
			},
			Retention: ResolvedMemoryRetention{
				TraceRetentionDays: DefaultMemoryTraceRetentionDays,
				TraceMaxSizeMB:     DefaultMemoryTraceMaxSizeMB,
			},
			Budgets: ResolvedMemoryBudgets{
				TotalTokens:            DefaultMemoryBudgetTotalTokens,
				DecisionsTokens:        DefaultMemoryBudgetDecisionsTokens,
				ConstraintsTokens:      DefaultMemoryBudgetConstraintsTokens,
				ImplementedTokens:      DefaultMemoryBudgetImplementedTokens,
				OpenThreadsTokens:      DefaultMemoryBudgetOpenThreadsTokens,
				ArtifactPointersTokens: DefaultMemoryBudgetArtifactPointersTokens,
			},
		},
	}
}
