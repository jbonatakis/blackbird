package execution

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jbonatakis/blackbird/internal/plan"
)

const defaultParentReviewChildSummaryMaxBytes = 1200

type ParentReviewContextOptions struct {
	MaxChildSummaryBytes int
}

func BuildParentReviewContext(
	g plan.WorkGraph,
	baseDir string,
	parentTaskID string,
	opts ParentReviewContextOptions,
) (ContextPack, error) {
	if strings.TrimSpace(baseDir) == "" {
		return ContextPack{}, fmt.Errorf("baseDir required")
	}
	if parentTaskID == "" {
		return ContextPack{}, fmt.Errorf("task id required")
	}

	parent, ok := g.Items[parentTaskID]
	if !ok {
		return ContextPack{}, fmt.Errorf("unknown task id %q", parentTaskID)
	}
	if len(parent.ChildIDs) == 0 {
		return ContextPack{}, fmt.Errorf("parent task %q has no child ids", parentTaskID)
	}

	contexts, err := GetLatestCompletedChildRuns(g, baseDir, parentTaskID)
	if err != nil {
		return ContextPack{}, err
	}

	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].ChildID < contexts[j].ChildID
	})

	limits := parentReviewContextLimits{
		MaxChildSummaryBytes: opts.MaxChildSummaryBytes,
	}
	limits = applyDefaultParentReviewContextLimits(limits)

	children := make([]ParentReviewChildContext, 0, len(contexts))
	for _, childContext := range contexts {
		children = append(children, ParentReviewChildContext{
			ChildID:          childContext.ChildID,
			ChildTitle:       childContext.Child.Title,
			LatestRunID:      childContext.Run.ID,
			LatestRunSummary: parentReviewSummaryForChildRun(childContext.Run, limits.MaxChildSummaryBytes),
			ArtifactRefs:     parentReviewArtifactRefsForChildRun(childContext.Run),
		})
	}

	acceptanceCriteria := append([]string{}, parent.AcceptanceCriteria...)
	return ContextPack{
		SchemaVersion: ContextPackSchemaVersion,
		Task: TaskContext{
			ID:                 parent.ID,
			Title:              parent.Title,
			Description:        parent.Description,
			AcceptanceCriteria: acceptanceCriteria,
			Prompt:             parent.Prompt,
		},
		ParentReview: &ParentReviewContext{
			ParentTaskID:         parent.ID,
			ParentTaskTitle:      parent.Title,
			AcceptanceCriteria:   acceptanceCriteria,
			ReviewerInstructions: parentReviewerInstructions(),
			Children:             children,
		},
		SystemPrompt: parentReviewSystemPrompt(),
	}, nil
}

type parentReviewContextLimits struct {
	MaxChildSummaryBytes int
}

func applyDefaultParentReviewContextLimits(limits parentReviewContextLimits) parentReviewContextLimits {
	if limits.MaxChildSummaryBytes <= 0 {
		limits.MaxChildSummaryBytes = defaultParentReviewChildSummaryMaxBytes
	}
	return limits
}

func parentReviewSummaryForChildRun(run RunRecord, maxBytes int) string {
	if run.ReviewSummary == nil {
		return ""
	}
	return truncateString(strings.TrimSpace(run.ReviewSummary.DiffStat), maxBytes)
}

func parentReviewArtifactRefsForChildRun(run RunRecord) []string {
	refs := []string{
		filepath.ToSlash(filepath.Join(runsDirName, run.TaskID, run.ID+".json")),
	}
	if run.ReviewSummary == nil {
		return refs
	}

	fileRefs := make([]string, 0, len(run.ReviewSummary.Files))
	seen := make(map[string]struct{}, len(run.ReviewSummary.Files))
	for _, file := range run.ReviewSummary.Files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		if _, ok := seen[file]; ok {
			continue
		}
		seen[file] = struct{}{}
		fileRefs = append(fileRefs, file)
	}
	sort.Strings(fileRefs)
	return append(refs, fileRefs...)
}

func parentReviewSystemPrompt() string {
	return "You are running a parent-task review. Evaluate whether child-task outputs satisfy the parent acceptance criteria. Do not implement code changes."
}

func parentReviewerInstructions() string {
	return "Act as a reviewer only. Assess the parent acceptance criteria against child outputs, flag major correctness or security issues, and map failures to child task IDs with actionable feedback."
}
