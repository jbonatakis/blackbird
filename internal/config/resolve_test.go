package config

import "testing"

func TestResolveConfigPrecedence(t *testing.T) {
	project := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(12),
			PlanDataRefreshIntervalSeconds: nil,
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: intPtr(0),
		},
		Execution: &RawExecution{
			StopAfterEachTask: boolPtr(true),
		},
	}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(20),
			PlanDataRefreshIntervalSeconds: intPtr(9),
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: intPtr(2),
		},
		Execution: &RawExecution{
			StopAfterEachTask: boolPtr(false),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != 12 {
		t.Fatalf("run interval = %d, want 12", resolved.TUI.RunDataRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != 9 {
		t.Fatalf("plan interval = %d, want 9", resolved.TUI.PlanDataRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != 0 {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want 0", resolved.Planning.MaxPlanAutoRefinePasses)
	}
	if resolved.Execution.StopAfterEachTask != true {
		t.Fatalf("stopAfterEachTask = %v, want true", resolved.Execution.StopAfterEachTask)
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
	if resolved.Planning.MaxPlanAutoRefinePasses != DefaultMaxPlanAutoRefinePasses {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want %d", resolved.Planning.MaxPlanAutoRefinePasses, DefaultMaxPlanAutoRefinePasses)
	}
	if resolved.Execution.StopAfterEachTask != DefaultStopAfterEachTask {
		t.Fatalf("stopAfterEachTask = %v, want %v", resolved.Execution.StopAfterEachTask, DefaultStopAfterEachTask)
	}
}

func TestResolveConfigStopAfterEachTaskDefaultsFalse(t *testing.T) {
	resolved := ResolveConfig(RawConfig{}, RawConfig{})
	if resolved.Execution.StopAfterEachTask {
		t.Fatalf("expected stopAfterEachTask default false, got true")
	}
}

func TestResolveConfigClampBounds(t *testing.T) {
	project := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(0),
			PlanDataRefreshIntervalSeconds: intPtr(999),
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: intPtr(99),
		},
	}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(8),
			PlanDataRefreshIntervalSeconds: intPtr(10),
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: intPtr(2),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != MinRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, MinRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != MaxRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != MaxPlanAutoRefinePasses {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want %d", resolved.Planning.MaxPlanAutoRefinePasses, MaxPlanAutoRefinePasses)
	}
}

func TestResolveConfigClampGlobalWhenProjectMissing(t *testing.T) {
	project := RawConfig{}
	global := RawConfig{
		TUI: &RawTUI{
			RunDataRefreshIntervalSeconds:  intPtr(-5),
			PlanDataRefreshIntervalSeconds: intPtr(5000),
		},
		Planning: &RawPlanning{
			MaxPlanAutoRefinePasses: intPtr(-1),
		},
	}

	resolved := ResolveConfig(project, global)
	if resolved.TUI.RunDataRefreshIntervalSeconds != MinRefreshIntervalSeconds {
		t.Fatalf("run interval = %d, want %d", resolved.TUI.RunDataRefreshIntervalSeconds, MinRefreshIntervalSeconds)
	}
	if resolved.TUI.PlanDataRefreshIntervalSeconds != MaxRefreshIntervalSeconds {
		t.Fatalf("plan interval = %d, want %d", resolved.TUI.PlanDataRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
	if resolved.Planning.MaxPlanAutoRefinePasses != MinPlanAutoRefinePasses {
		t.Fatalf("maxPlanAutoRefinePasses = %d, want %d", resolved.Planning.MaxPlanAutoRefinePasses, MinPlanAutoRefinePasses)
	}
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
