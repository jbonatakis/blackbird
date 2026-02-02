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

func intPtr(value int) *int {
	return &value
}
