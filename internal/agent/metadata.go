package agent

import "strings"

// ApplyRuntimeProvider sets the request provider from the runtime when unset.
func ApplyRuntimeProvider(meta RequestMetadata, runtime Runtime) RequestMetadata {
	if strings.TrimSpace(meta.Provider) != "" {
		return meta
	}
	if strings.TrimSpace(runtime.Provider) != "" {
		meta.Provider = runtime.Provider
	}
	return meta
}
