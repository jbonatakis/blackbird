package config

import "testing"

func TestResolveConfigPrecedence(t *testing.T) {
	project := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(12),
			PlanDataRefreshIntervalSeconds: nil,
		},
	}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(20),
			PlanDataRefreshIntervalSeconds: intPtr(9),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("run interval = %d, want 12", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 9 {
		t.Fatalf("plan interval = %d, want 9", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
}

func TestResolveConfigDefaults(t *testing.T) {
	resolved := ResolveConfig(RawConfig{}, RawConfig{})
	if resolved.TUI.RunDataRefreshIntervalSeconds != DefaultRunDataRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, DefaultRunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != DefaultPlanDataRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, DefaultPlanDataRefreshIntervalSeconds)
	}
	if resolved.Memory.Mode != DefaultMemoryMode {
		t.Fatalf("memory mode = %s, want %s", resolved.Memory.Mode, DefaultMemoryMode)
	}
	if resolved.Memory.Proxy.UpstreamURL != DefaultMemoryProxyUpstreamURLCodex {
		t.Fatalf("memory proxy upstream = %s, want %s", resolved.Memory.Proxy.UpstreamURL, DefaultMemoryProxyUpstreamURLCodex)
	}
	if resolved.Memory.Retention.TraceRetentionDays != DefaultMemoryTraceRetentionDays {
		t.Fatalf("memory retention days = %d, want %d", resolved.Memory.Retention.TraceRetentionDays, DefaultMemoryTraceRetentionDays)
	}
	if resolved.Memory.Budgets.TotalTokens != DefaultMemoryBudgetTotalTokens {
		t.Fatalf("memory budget total = %d, want %d", resolved.Memory.Budgets.TotalTokens, DefaultMemoryBudgetTotalTokens)
	}
}

func TestResolveConfigClampBounds(t *testing.T) {
	project := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(0),
			PlanDataRefreshIntervalSeconds: intPtr(999),
		},
	}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(8),
			PlanDataRefreshIntervalSeconds: intPtr(10),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != MinRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, MinRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != MaxRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
}

func TestResolveConfigClampGlobalWhenProjectMissing(t *testing.T) {
	project := RawConfig{}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(-5),
			PlanDataRefreshIntervalSeconds: intPtr(5000),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != MinRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, MinRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != MaxRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
}

func TestResolveConfigMemoryPrecedence(t *testing.T) {
	project := RawConfig{
		Memory: &RawMemory{
			Mode: stringPtr("local"),
			Proxy: &RawMemoryProxy{
				UpstreamURL: stringPtr("https://project.example"),
			},
			Retention: &RawMemoryRetention{
				TraceRetentionDays: intPtr(30),
			},
			Budgets: &RawMemoryBudgets{
				TotalTokens: intPtr(900),
			},
		},
	}
	global := RawConfig{
		Memory: &RawMemory{
			Mode: stringPtr("provider"),
			Proxy: &RawMemoryProxy{
				UpstreamURL: stringPtr("https://global.example"),
			},
			Retention: &RawMemoryRetention{
				TraceRetentionDays: intPtr(7),
			},
			Budgets: &RawMemoryBudgets{
				TotalTokens: intPtr(1400),
			},
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.Memory.Mode != "local" {
		t.Fatalf("memory mode = %s, want local", resolved.Memory.Mode)
	}
	if resolved.Memory.Proxy.UpstreamURL != "https://project.example" {
		t.Fatalf("memory proxy upstream = %s, want project value", resolved.Memory.Proxy.UpstreamURL)
	}
	if resolved.Memory.Retention.TraceRetentionDays != 30 {
		t.Fatalf("memory retention days = %d, want 30", resolved.Memory.Retention.TraceRetentionDays)
	}
	if resolved.Memory.Budgets.TotalTokens != 900 {
		t.Fatalf("memory budget total = %d, want 900", resolved.Memory.Budgets.TotalTokens)
	}
}

func TestResolveConfigMemoryValidation(t *testing.T) {
	project := RawConfig{
		Memory: &RawMemory{
			Mode: stringPtr("unsupported"),
			Proxy: &RawMemoryProxy{
				UpstreamURL: stringPtr("   "),
				Lossless:    boolPtr(false),
			},
			Retention: &RawMemoryRetention{
				TraceRetentionDays: intPtr(-3),
				TraceMaxSizeMB:     intPtr(999999),
			},
			Budgets: &RawMemoryBudgets{
				TotalTokens:            intPtr(-5),
				ArtifactPointersTokens: intPtr(999999),
			},
		},
	}

	resolved := ResolveConfig(project, RawConfig{})
	if resolved.Memory.Mode != DefaultMemoryMode {
		t.Fatalf("memory mode = %s, want %s", resolved.Memory.Mode, DefaultMemoryMode)
	}
	if resolved.Memory.Proxy.UpstreamURL != DefaultMemoryProxyUpstreamURLCodex {
		t.Fatalf("memory proxy upstream = %s, want %s", resolved.Memory.Proxy.UpstreamURL, DefaultMemoryProxyUpstreamURLCodex)
	}
	if resolved.Memory.Proxy.Lossless != false {
		t.Fatalf("memory proxy lossless = %v, want false", resolved.Memory.Proxy.Lossless)
	}
	if resolved.Memory.Retention.TraceRetentionDays != MinMemoryTraceRetentionDays {
		t.Fatalf("memory retention days = %d, want %d", resolved.Memory.Retention.TraceRetentionDays, MinMemoryTraceRetentionDays)
	}
	if resolved.Memory.Retention.TraceMaxSizeMB != MaxMemoryTraceMaxSizeMB {
		t.Fatalf("memory trace max size = %d, want %d", resolved.Memory.Retention.TraceMaxSizeMB, MaxMemoryTraceMaxSizeMB)
	}
	if resolved.Memory.Budgets.TotalTokens != MinMemoryBudgetTokens {
		t.Fatalf("memory budget total = %d, want %d", resolved.Memory.Budgets.TotalTokens, MinMemoryBudgetTokens)
	}
	if resolved.Memory.Budgets.ArtifactPointersTokens != MaxMemoryBudgetTokens {
		t.Fatalf("memory budget artifact pointers = %d, want %d", resolved.Memory.Budgets.ArtifactPointersTokens, MaxMemoryBudgetTokens)
	}
}

func intPtr(value int) *int {
	return &value
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
