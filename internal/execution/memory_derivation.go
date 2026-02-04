package execution

import (
	"strings"

	"github.com/jbonatakis/blackbird/internal/memory/derive"
	"github.com/jbonatakis/blackbird/internal/memory/index"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
)

func deriveMemoryFromWAL(baseDir, providerID, runID string) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return nil
	}

	adapter := memprovider.Select(providerID)
	if adapter == nil {
		return nil
	}

	memoryConfig := resolveMemoryConfig(baseDir, nil)
	if !adapter.Enabled(memoryConfig) {
		return nil
	}

	lookup := RunTimeLookupFromExecution(baseDir)
	return derive.FromWAL(derive.Options{
		ProjectRoot:   baseDir,
		RunID:         runID,
		RunTimeLookup: index.RunTimeLookup(lookup),
	})
}
