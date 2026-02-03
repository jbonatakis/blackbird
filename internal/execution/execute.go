package execution

import (
	"context"
	"fmt"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// ExecuteTask builds context and launches the agent for a single task.
func ExecuteTask(ctx context.Context, g plan.WorkGraph, task plan.WorkItem, runtime agent.Runtime) (RunRecord, error) {
	if task.ID == "" {
		return RunRecord{}, fmt.Errorf("task id required")
	}
	if _, ok := g.Items[task.ID]; !ok {
		return RunRecord{}, fmt.Errorf("unknown task id %q", task.ID)
	}

	runtime = applySelectedProvider(runtime)
	ctxPack, err := BuildContextWithOptions(g, task.ID, ContextBuildOptions{
		Provider: runtime.Provider,
	})
	if err != nil {
		return RunRecord{}, err
	}
	return LaunchAgent(ctx, runtime, ctxPack)
}
