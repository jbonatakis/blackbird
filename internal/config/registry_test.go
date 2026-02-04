package config

import "testing"

func TestOptionRegistryIncludesKnownOptions(t *testing.T) {
	defaults := DefaultResolvedConfig()
	options := OptionRegistry()
	if len(options) != 3 {
		t.Fatalf("options count = %d, want 3", len(options))
	}

	byKey := map[string]OptionMetadata{}
	for _, option := range options {
		if option.KeyPath == "" {
			t.Fatalf("option has empty key path")
		}
		if _, exists := byKey[option.KeyPath]; exists {
			t.Fatalf("duplicate option key: %s", option.KeyPath)
		}
		byKey[option.KeyPath] = option
	}

	run := requireOption(t, byKey, "tui.runDataRefreshIntervalSeconds")
	if run.DisplayName != "TUI Run Refresh (seconds)" {
		t.Fatalf("run display name = %q, want %q", run.DisplayName, "TUI Run Refresh (seconds)")
	}
	if run.Type != OptionTypeInt {
		t.Fatalf("run type = %q, want %q", run.Type, OptionTypeInt)
	}
	if run.DefaultInt != defaults.TUI.RunDataRefreshIntervalSeconds {
		t.Fatalf("run default = %d, want %d", run.DefaultInt, defaults.TUI.RunDataRefreshIntervalSeconds)
	}
	if run.Bounds == nil {
		t.Fatalf("run bounds = nil, want bounds")
	}
	if run.Bounds.Min != MinRefreshIntervalSeconds || run.Bounds.Max != MaxRefreshIntervalSeconds {
		t.Fatalf("run bounds = %d-%d, want %d-%d", run.Bounds.Min, run.Bounds.Max, MinRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
	if run.Description != "Run data refresh interval in seconds" {
		t.Fatalf("run description = %q, want %q", run.Description, "Run data refresh interval in seconds")
	}

	plan := requireOption(t, byKey, "tui.planDataRefreshIntervalSeconds")
	if plan.DisplayName != "TUI Plan Refresh (seconds)" {
		t.Fatalf("plan display name = %q, want %q", plan.DisplayName, "TUI Plan Refresh (seconds)")
	}
	if plan.Type != OptionTypeInt {
		t.Fatalf("plan type = %q, want %q", plan.Type, OptionTypeInt)
	}
	if plan.DefaultInt != defaults.TUI.PlanDataRefreshIntervalSeconds {
		t.Fatalf("plan default = %d, want %d", plan.DefaultInt, defaults.TUI.PlanDataRefreshIntervalSeconds)
	}
	if plan.Bounds == nil {
		t.Fatalf("plan bounds = nil, want bounds")
	}
	if plan.Bounds.Min != MinRefreshIntervalSeconds || plan.Bounds.Max != MaxRefreshIntervalSeconds {
		t.Fatalf("plan bounds = %d-%d, want %d-%d", plan.Bounds.Min, plan.Bounds.Max, MinRefreshIntervalSeconds, MaxRefreshIntervalSeconds)
	}
	if plan.Description != "Plan data refresh interval in seconds" {
		t.Fatalf("plan description = %q, want %q", plan.Description, "Plan data refresh interval in seconds")
	}

	stop := requireOption(t, byKey, "execution.stopAfterEachTask")
	if stop.DisplayName != "Execution Stop After Each Task" {
		t.Fatalf("stop display name = %q, want %q", stop.DisplayName, "Execution Stop After Each Task")
	}
	if stop.Type != OptionTypeBool {
		t.Fatalf("stop type = %q, want %q", stop.Type, OptionTypeBool)
	}
	if stop.DefaultBool != defaults.Execution.StopAfterEachTask {
		t.Fatalf("stop default = %v, want %v", stop.DefaultBool, defaults.Execution.StopAfterEachTask)
	}
	if stop.Bounds != nil {
		t.Fatalf("stop bounds = %v, want nil", stop.Bounds)
	}
	if stop.Description != "Pause execution for review after each task" {
		t.Fatalf("stop description = %q, want %q", stop.Description, "Pause execution for review after each task")
	}
}

func requireOption(t *testing.T, options map[string]OptionMetadata, key string) OptionMetadata {
	t.Helper()
	option, ok := options[key]
	if !ok {
		t.Fatalf("missing option: %s", key)
	}
	return option
}
