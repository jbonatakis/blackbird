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
			StopAfterEachTask:   boolPtr(true),
			ParentReviewEnabled: boolPtr(false),
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
			StopAfterEachTask:   boolPtr(false),
			ParentReviewEnabled: boolPtr(true),
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
	if resolved.Execution.ParentReviewEnabled != false {
		t.Fatalf("parentReviewEnabled = %v, want false", resolved.Execution.ParentReviewEnabled)
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
	if resolved.Execution.ParentReviewEnabled != DefaultParentReviewEnabled {
		t.Fatalf("parentReviewEnabled = %v, want %v", resolved.Execution.ParentReviewEnabled, DefaultParentReviewEnabled)
	}
}

func TestResolveConfigStopAfterEachTaskDefaultsFalse(t *testing.T) {
	resolved := ResolveConfig(RawConfig{}, RawConfig{})
	if resolved.Execution.StopAfterEachTask {
		t.Fatalf("expected stopAfterEachTask default false, got true")
	}
}

func TestResolveConfigParentReviewEnabledPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		projectVal *bool
		globalVal  *bool
		want       bool
	}{
		{
			name:       "default false when unset in both layers",
			projectVal: nil,
			globalVal:  nil,
			want:       false,
		},
		{
			name:       "global true when project unset",
			projectVal: nil,
			globalVal:  boolPtr(true),
			want:       true,
		},
		{
			name:       "project true overrides global false",
			projectVal: boolPtr(true),
			globalVal:  boolPtr(false),
			want:       true,
		},
		{
			name:       "project false overrides global true",
			projectVal: boolPtr(false),
			globalVal:  boolPtr(true),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := RawConfig{}
			if tt.projectVal != nil {
				project.Execution = &RawExecution{
					ParentReviewEnabled: tt.projectVal,
				}
			}

			global := RawConfig{}
			if tt.globalVal != nil {
				global.Execution = &RawExecution{
					ParentReviewEnabled: tt.globalVal,
				}
			}

			resolved := ResolveConfig(project, global)
			if resolved.Execution.ParentReviewEnabled != tt.want {
				t.Fatalf(
					"parentReviewEnabled = %v, want %v (project=%#v global=%#v)",
					resolved.Execution.ParentReviewEnabled,
					tt.want,
					tt.projectVal,
					tt.globalVal,
				)
			}
		})
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
