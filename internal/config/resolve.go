package config

import "strings"

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

	mode := resolveMemoryMode(
		valueFromMemory(project, func(memory RawMemory) *string { return memory.Mode }),
		valueFromMemory(global, func(memory RawMemory) *string { return memory.Mode }),
		defaults.Memory.Mode,
	)
	listenAddr := resolveString(
		valueFromMemoryProxy(project, func(proxy RawMemoryProxy) *string { return proxy.ListenAddr }),
		valueFromMemoryProxy(global, func(proxy RawMemoryProxy) *string { return proxy.ListenAddr }),
		defaults.Memory.Proxy.ListenAddr,
	)
	upstreamURL := resolveString(
		valueFromMemoryProxy(project, func(proxy RawMemoryProxy) *string { return proxy.UpstreamURL }),
		valueFromMemoryProxy(global, func(proxy RawMemoryProxy) *string { return proxy.UpstreamURL }),
		defaults.Memory.Proxy.UpstreamURL,
	)
	chatGPTUpstreamURL := resolveString(
		valueFromMemoryProxy(project, func(proxy RawMemoryProxy) *string { return proxy.ChatGPTUpstreamURL }),
		valueFromMemoryProxy(global, func(proxy RawMemoryProxy) *string { return proxy.ChatGPTUpstreamURL }),
		defaults.Memory.Proxy.ChatGPTUpstreamURL,
	)
	lossless := resolveBool(
		valueFromMemoryProxyBool(project, func(proxy RawMemoryProxy) *bool { return proxy.Lossless }),
		valueFromMemoryProxyBool(global, func(proxy RawMemoryProxy) *bool { return proxy.Lossless }),
		defaults.Memory.Proxy.Lossless,
	)
	traceRetentionDays := resolveIntWithBounds(
		valueFromMemoryRetention(project, func(retention RawMemoryRetention) *int { return retention.TraceRetentionDays }),
		valueFromMemoryRetention(global, func(retention RawMemoryRetention) *int { return retention.TraceRetentionDays }),
		defaults.Memory.Retention.TraceRetentionDays,
		MinMemoryTraceRetentionDays,
		MaxMemoryTraceRetentionDays,
	)
	traceMaxSizeMB := resolveIntWithBounds(
		valueFromMemoryRetention(project, func(retention RawMemoryRetention) *int { return retention.TraceMaxSizeMB }),
		valueFromMemoryRetention(global, func(retention RawMemoryRetention) *int { return retention.TraceMaxSizeMB }),
		defaults.Memory.Retention.TraceMaxSizeMB,
		MinMemoryTraceMaxSizeMB,
		MaxMemoryTraceMaxSizeMB,
	)
	totalTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.TotalTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.TotalTokens }),
		defaults.Memory.Budgets.TotalTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)
	decisionTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.DecisionsTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.DecisionsTokens }),
		defaults.Memory.Budgets.DecisionsTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)
	constraintTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.ConstraintsTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.ConstraintsTokens }),
		defaults.Memory.Budgets.ConstraintsTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)
	implementedTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.ImplementedTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.ImplementedTokens }),
		defaults.Memory.Budgets.ImplementedTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)
	openThreadsTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.OpenThreadsTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.OpenThreadsTokens }),
		defaults.Memory.Budgets.OpenThreadsTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)
	artifactPointersTokens := resolveIntWithBounds(
		valueFromMemoryBudgets(project, func(budgets RawMemoryBudgets) *int { return budgets.ArtifactPointersTokens }),
		valueFromMemoryBudgets(global, func(budgets RawMemoryBudgets) *int { return budgets.ArtifactPointersTokens }),
		defaults.Memory.Budgets.ArtifactPointersTokens,
		MinMemoryBudgetTokens,
		MaxMemoryBudgetTokens,
	)

	return ResolvedConfig{
		SchemaVersion: SchemaVersion,
		TUI: ResolvedTUI{
			RunDataRefreshIntervalSeconds:  runInterval,
			PlanDataRefreshIntervalSeconds: planInterval,
		},
		Memory: ResolvedMemory{
			Mode: mode,
			Proxy: ResolvedMemoryProxy{
				ListenAddr:         listenAddr,
				UpstreamURL:        upstreamURL,
				ChatGPTUpstreamURL: chatGPTUpstreamURL,
				Lossless:           lossless,
			},
			Retention: ResolvedMemoryRetention{
				TraceRetentionDays: traceRetentionDays,
				TraceMaxSizeMB:     traceMaxSizeMB,
			},
			Budgets: ResolvedMemoryBudgets{
				TotalTokens:            totalTokens,
				DecisionsTokens:        decisionTokens,
				ConstraintsTokens:      constraintTokens,
				ImplementedTokens:      implementedTokens,
				OpenThreadsTokens:      openThreadsTokens,
				ArtifactPointersTokens: artifactPointersTokens,
			},
		},
	}
}

func valueFromRaw(cfg RawConfig, pick func(RawTUI) *int) *int {
	if cfg.TUI == nil {
		return nil
	}
	return pick(*cfg.TUI)
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

func clampInterval(value int) int {
	if value < MinRefreshIntervalSeconds {
		return MinRefreshIntervalSeconds
	}
	if value > MaxRefreshIntervalSeconds {
		return MaxRefreshIntervalSeconds
	}
	return value
}

func valueFromMemory(cfg RawConfig, pick func(RawMemory) *string) *string {
	if cfg.Memory == nil {
		return nil
	}
	return pick(*cfg.Memory)
}

func valueFromMemoryProxy(cfg RawConfig, pick func(RawMemoryProxy) *string) *string {
	if cfg.Memory == nil || cfg.Memory.Proxy == nil {
		return nil
	}
	return pick(*cfg.Memory.Proxy)
}

func valueFromMemoryProxyBool(cfg RawConfig, pick func(RawMemoryProxy) *bool) *bool {
	if cfg.Memory == nil || cfg.Memory.Proxy == nil {
		return nil
	}
	return pick(*cfg.Memory.Proxy)
}

func valueFromMemoryRetention(cfg RawConfig, pick func(RawMemoryRetention) *int) *int {
	if cfg.Memory == nil || cfg.Memory.Retention == nil {
		return nil
	}
	return pick(*cfg.Memory.Retention)
}

func valueFromMemoryBudgets(cfg RawConfig, pick func(RawMemoryBudgets) *int) *int {
	if cfg.Memory == nil || cfg.Memory.Budgets == nil {
		return nil
	}
	return pick(*cfg.Memory.Budgets)
}

func resolveMemoryMode(projectVal *string, globalVal *string, defaultVal string) string {
	if mode, ok := normalizeMemoryMode(projectVal); ok {
		return mode
	}
	if mode, ok := normalizeMemoryMode(globalVal); ok {
		return mode
	}
	if mode, ok := normalizeMemoryMode(&defaultVal); ok {
		return mode
	}
	return defaultVal
}

func normalizeMemoryMode(value *string) (string, bool) {
	if value == nil {
		return "", false
	}
	mode := strings.ToLower(strings.TrimSpace(*value))
	switch mode {
	case MemoryModeOff, MemoryModePassthrough, MemoryModeDeterministic, MemoryModeLocal, MemoryModeProvider:
		return mode, true
	default:
		return "", false
	}
}

func resolveString(projectVal *string, globalVal *string, defaultVal string) string {
	if value := normalizeString(projectVal); value != "" {
		return value
	}
	if value := normalizeString(globalVal); value != "" {
		return value
	}
	if value := normalizeString(&defaultVal); value != "" {
		return value
	}
	return defaultVal
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

func resolveIntWithBounds(projectVal *int, globalVal *int, defaultVal int, min int, max int) int {
	if projectVal != nil {
		return clampInt(*projectVal, min, max)
	}
	if globalVal != nil {
		return clampInt(*globalVal, min, max)
	}
	return clampInt(defaultVal, min, max)
}

func clampInt(value int, min int, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func normalizeString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
