package derive

import (
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
	"github.com/jbonatakis/blackbird/internal/memory/canonical"
	"github.com/jbonatakis/blackbird/internal/memory/index"
)

type Options struct {
	ProjectRoot   string
	RunID         string
	TracePath     string
	RunTimeLookup index.RunTimeLookup
	Now           time.Time
}

// FromWAL replays the trace WAL and updates canonical logs, artifacts, and index.
func FromWAL(opts Options) error {
	projectRoot := strings.TrimSpace(opts.ProjectRoot)
	tracePath := strings.TrimSpace(opts.TracePath)
	if tracePath == "" {
		tracePath = memory.TraceWALPath(projectRoot, "")
	}

	logs, err := canonical.CanonicalizeWAL(tracePath)
	if err != nil {
		return err
	}
	if len(logs) == 0 {
		return nil
	}

	logsToProcess := logs
	runID := strings.TrimSpace(opts.RunID)
	if runID != "" {
		_, found, err := artifact.LoadStoreForProject(projectRoot)
		if err != nil {
			return err
		}
		if found {
			logsToProcess = filterLogsByRun(logs, runID)
		}
	}
	if len(logsToProcess) == 0 {
		return nil
	}

	if err := canonical.SaveLogs(projectRoot, logsToProcess); err != nil {
		return err
	}
	if _, err := artifact.UpdateStore(projectRoot, logsToProcess); err != nil {
		return err
	}

	rebuildOpts := index.RebuildOptions{
		Now:           opts.Now,
		RunTimeLookup: opts.RunTimeLookup,
	}
	if err := index.RebuildForProject(projectRoot, rebuildOpts); err != nil {
		return err
	}
	return nil
}

func filterLogsByRun(logs []canonical.Log, runID string) []canonical.Log {
	filtered := make([]canonical.Log, 0, len(logs))
	for _, log := range logs {
		if strings.TrimSpace(log.RunID) != runID {
			continue
		}
		filtered = append(filtered, log)
	}
	return filtered
}
