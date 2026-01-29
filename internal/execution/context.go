package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// BuildContext assembles the execution context for a task.
func BuildContext(g plan.WorkGraph, taskID string) (ContextPack, error) {
	if taskID == "" {
		return ContextPack{}, fmt.Errorf("task id required")
	}
	it, ok := g.Items[taskID]
	if !ok {
		return ContextPack{}, fmt.Errorf("unknown task id %q", taskID)
	}

	deps := make([]DependencyContext, 0, len(it.Deps))
	for _, depID := range it.Deps {
		dep, ok := g.Items[depID]
		if !ok {
			return ContextPack{}, fmt.Errorf("unknown dependency id %q", depID)
		}
		deps = append(deps, DependencyContext{
			ID:     dep.ID,
			Title:  dep.Title,
			Status: string(dep.Status),
		})
	}

	pack := ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		SystemPrompt:  executionSystemPrompt(),
		Task: TaskContext{
			ID:                 it.ID,
			Title:              it.Title,
			Description:        it.Description,
			AcceptanceCriteria: append([]string{}, it.AcceptanceCriteria...),
			Prompt:             it.Prompt,
		},
		Dependencies:    deps,
		ProjectSnapshot: loadProjectSnapshot(),
	}

	return pack, nil
}

func executionSystemPrompt() string {
	return "You are authorized to run non-destructive commands and edit files needed to complete the task. " +
		"Do not ask for confirmation. Avoid destructive operations (e.g., deleting unrelated files, wiping directories, " +
		"resetting git history, or modifying system files)."
}

func loadProjectSnapshot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	candidates := []string{
		filepath.Join(wd, ".blackbird", "snapshot.md"),
		filepath.Join(wd, "OVERVIEW.md"),
		filepath.Join(wd, "README.md"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			continue
		}
		return strings.TrimSpace(string(data))
	}

	return ""
}
