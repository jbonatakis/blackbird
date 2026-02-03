package execution

import (
	"time"

	"github.com/jbonatakis/blackbird/internal/memory/contextpack"
)

// RunTimeLookupFromExecution uses execution run records for timestamps.
func RunTimeLookupFromExecution(baseDir string) contextpack.RunTimeLookup {
	cache := make(map[string]time.Time)
	return func(taskID, runID string) (time.Time, bool) {
		if taskID == "" || runID == "" {
			return time.Time{}, false
		}
		key := taskID + ":" + runID
		if ts, ok := cache[key]; ok {
			return ts, true
		}
		record, err := LoadRun(baseDir, taskID, runID)
		if err != nil {
			return time.Time{}, false
		}
		cache[key] = record.StartedAt
		return record.StartedAt, true
	}
}
