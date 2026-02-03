package provider

import "strings"

var adapters = []Adapter{
	CodexAdapter{},
	ClaudeAdapter{},
}

func Select(provider string) Adapter {
	normalized := strings.ToLower(strings.TrimSpace(provider))
	if normalized == "" {
		return nil
	}
	for _, adapter := range adapters {
		if strings.EqualFold(adapter.ProviderID(), normalized) {
			return adapter
		}
	}
	return nil
}
