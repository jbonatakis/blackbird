package execution

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/memory"
	"github.com/jbonatakis/blackbird/internal/memory/artifact"
	"github.com/jbonatakis/blackbird/internal/memory/contextpack"
	memprovider "github.com/jbonatakis/blackbird/internal/memory/provider"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ContextBuildOptions struct {
	BaseDir  string
	Provider string
	Memory   *config.ResolvedMemory
	Now      time.Time
}

// BuildContext assembles the execution context for a task.
func BuildContext(g plan.WorkGraph, taskID string) (ContextPack, error) {
	return BuildContextWithOptions(g, taskID, ContextBuildOptions{})
}

// BuildContextWithOptions assembles the execution context with optional memory context pack support.
func BuildContextWithOptions(g plan.WorkGraph, taskID string, opts ContextBuildOptions) (ContextPack, error) {
	baseDir := resolveBaseDir(opts.BaseDir)
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
		ProjectSnapshot: loadProjectSnapshot(baseDir),
	}

	if err := attachMemoryContextPack(&pack, baseDir, opts); err != nil {
		return ContextPack{}, err
	}

	return pack, nil
}

func executionSystemPrompt() string {
	return "You are authorized to run non-destructive commands and edit files needed to complete the task. " +
		"Do not ask for confirmation. Avoid destructive operations (e.g., deleting unrelated files, wiping directories, " +
		"resetting git history, or modifying system files)."
}

func loadProjectSnapshot(baseDir string) string {
	baseDir = resolveBaseDir(baseDir)
	if baseDir == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(baseDir, ".blackbird", "snapshot.md"),
		filepath.Join(baseDir, "OVERVIEW.md"),
		filepath.Join(baseDir, "README.md"),
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

func attachMemoryContextPack(pack *ContextPack, baseDir string, opts ContextBuildOptions) error {
	providerID := strings.TrimSpace(opts.Provider)
	if providerID == "" {
		return nil
	}

	adapter := memprovider.Select(providerID)
	if adapter == nil {
		return nil
	}

	memoryConfig := resolveMemoryConfig(baseDir, opts.Memory)
	if !adapter.Enabled(memoryConfig) {
		return nil
	}

	session, _, err := memory.LoadOrCreateSession(memory.SessionPath(baseDir), pack.SessionGoal)
	if err != nil {
		return err
	}

	pack.SessionID = session.SessionID
	if pack.SessionGoal == "" {
		pack.SessionGoal = session.Goal
	}

	store, _, err := artifact.LoadStoreForProject(baseDir)
	if err != nil {
		return err
	}

	memPack := contextpack.Build(contextpack.BuildOptions{
		SessionID:     pack.SessionID,
		SessionGoal:   pack.SessionGoal,
		Artifacts:     store.Artifacts,
		Budget:        budgetFromMemory(memoryConfig),
		RunTimeLookup: RunTimeLookupFromExecution(baseDir),
		Now:           opts.Now,
	})
	pack.Memory = &memPack
	return nil
}

func resolveBaseDir(baseDir string) string {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir != "" {
		return baseDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func resolveMemoryConfig(baseDir string, override *config.ResolvedMemory) config.ResolvedMemory {
	if override != nil {
		return *override
	}
	resolved, err := config.LoadConfig(baseDir)
	if err != nil {
		return config.DefaultResolvedConfig().Memory
	}
	return resolved.Memory
}

func budgetFromMemory(memoryConfig config.ResolvedMemory) contextpack.Budget {
	budgets := memoryConfig.Budgets
	return contextpack.Budget{
		TotalTokens:            budgets.TotalTokens,
		DecisionsTokens:        budgets.DecisionsTokens,
		ConstraintsTokens:      budgets.ConstraintsTokens,
		ImplementedTokens:      budgets.ImplementedTokens,
		OpenThreadsTokens:      budgets.OpenThreadsTokens,
		ArtifactPointersTokens: budgets.ArtifactPointersTokens,
	}
}
