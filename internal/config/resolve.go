package config

// ResolveConfig merges project/global configs with built-in defaults.
// Precedence per key: project > global > defaults, then clamp intervals to bounds.
func ResolveConfig(project RawConfig, global RawConfig) ResolvedConfig {
	defaults := DefaultResolvedConfig()

	runInterval := resolveInterval(
		valueFromRaw(project, func(tui RawTUI) *int { return tui.RunDataRefreshIntervalSeconds }),
		valueFromRaw(global, func(tui RawTUI) *int { return tui.RunDataRefreshIntervalSeconds }),
		defaults.TUI.RunDataRefreshIntervalSeconds,
	)
	planInterval := resolveInterval(
		valueFromRaw(project, func(tui RawTUI) *int { return tui.PlanDataRefreshIntervalSeconds }),
		valueFromRaw(global, func(tui RawTUI) *int { return tui.PlanDataRefreshIntervalSeconds }),
		defaults.TUI.PlanDataRefreshIntervalSeconds,
	)
	maxPlanAutoRefinePasses := resolvePlanAutoRefinePasses(
		valueFromRawPlanning(project, func(planning RawPlanning) *int { return planning.MaxPlanAutoRefinePasses }),
		valueFromRawPlanning(global, func(planning RawPlanning) *int { return planning.MaxPlanAutoRefinePasses }),
		defaults.Planning.MaxPlanAutoRefinePasses,
	)
	stopAfterEachTask := resolveBool(
		valueFromRawExecution(project, func(exec RawExecution) *bool { return exec.StopAfterEachTask }),
		valueFromRawExecution(global, func(exec RawExecution) *bool { return exec.StopAfterEachTask }),
		defaults.Execution.StopAfterEachTask,
	)
	parentReviewEnabled := resolveBool(
		valueFromRawExecution(project, func(exec RawExecution) *bool { return exec.ParentReviewEnabled }),
		valueFromRawExecution(global, func(exec RawExecution) *bool { return exec.ParentReviewEnabled }),
		defaults.Execution.ParentReviewEnabled,
	)

	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  runInterval,
			PlanDataRefreshIntervalSeconds: planInterval,
		},
		Planning: ResolvedPlanning{
			MaxPlanAutoRefinePasses: maxPlanAutoRefinePasses,
		},
		Execution: ResolvedExecution{
			StopAfterEachTask:   stopAfterEachTask,
			ParentReviewEnabled: parentReviewEnabled,
		},
	}
}

func valueFromRaw(cfg RawConfig, pick func(RawTUI) *int) *int {
	if cfg.TUI == nil {
		return nil
	}
	return pick(*cfg.TUI)
}

func valueFromRawExecution(cfg RawConfig, pick func(RawExecution) *bool) *bool {
	if cfg.Execution == nil {
		return nil
	}
	return pick(*cfg.Execution)
}

func valueFromRawPlanning(cfg RawConfig, pick func(RawPlanning) *int) *int {
	if cfg.Planning == nil {
		return nil
	}
	return pick(*cfg.Planning)
}

func resolveInterval(projectVal *int, globalVal *int, defaultVal int) int {
	if projectVal != nil {
		return clampInterval(*projectVal)
	}
	if globalVal != nil {
		return clampInterval(*globalVal)
	}
	return clampInterval(defaultVal)
}

func resolveBool(projectVal *bool, globalVal *bool, defaultVal bool) bool {
	if projectVal != nil {
		return *projectVal
	}
	if globalVal != nil {
		return *globalVal
	}
	return defaultVal
}

func resolvePlanAutoRefinePasses(projectVal *int, globalVal *int, defaultVal int) int {
	if projectVal != nil {
		return clampPlanAutoRefinePasses(*projectVal)
	}
	if globalVal != nil {
		return clampPlanAutoRefinePasses(*globalVal)
	}
	return clampPlanAutoRefinePasses(defaultVal)
}

func clampInterval(value int) int {
	if value < MinRefreshIntervalSeconds {
		return MinRefreshIntervalSeconds
	}
	if value > MaxRefreshIntervalSeconds {
		return MaxRefreshIntervalSeconds
	}
	return value
}

func clampPlanAutoRefinePasses(value int) int {
	if value < MinPlanAutoRefinePasses {
		return MinPlanAutoRefinePasses
	}
	if value > MaxPlanAutoRefinePasses {
		return MaxPlanAutoRefinePasses
	}
	return value
}
