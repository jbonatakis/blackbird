package execution

import "strings"

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func supportsResumeProvider(provider string) bool {
	switch normalizeProvider(provider) {
	case "codex", "claude":
		return true
	default:
		return false
	}
}

func defaultProviderCommand(provider string) (string, bool) {
	switch normalizeProvider(provider) {
	case "codex":
		return "codex", true
	case "claude":
		return "claude", true
	default:
		return "", false
	}
}
