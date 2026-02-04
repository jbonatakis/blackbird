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
	stopAfterEachTask := resolveBool(
		valueFromRawExecution(project, func(exec RawExecution) *bool { return exec.StopAfterEachTask }),
		valueFromRawExecution(global, func(exec RawExecution) *bool { return exec.StopAfterEachTask }),
		defaults.Execution.StopAfterEachTask,
	)

	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  runInterval,
			PlanDataRefreshIntervalSeconds: planInterval,
		},
		Execution: ResolvedExecution{
			StopAfterEachTask: stopAfterEachTask,
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

func clampInterval(value int) int {
	if value < MinRefreshIntervalSeconds {
		return MinRefreshIntervalSeconds
	}
	if value > MaxRefreshIntervalSeconds {
		return MaxRefreshIntervalSeconds
	}
	return value
}
