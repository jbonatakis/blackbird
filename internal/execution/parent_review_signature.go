package execution

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// ChildCompletion captures the completion timestamp associated with a child task.
type ChildCompletion struct {
	ChildID     string
	CompletedAt time.Time
}

// ParentReviewCompletionSignature builds a deterministic signature for a parent's
// completed-child state. Input ordering does not affect output.
func ParentReviewCompletionSignature(parentTaskID string, completions []ChildCompletion) (string, error) {
	if parentTaskID == "" {
		return "", fmt.Errorf("task id required")
	}

	normalized, err := normalizeChildCompletions(completions)
	if err != nil {
		return "", err
	}

	payload := parentReviewCompletionSignaturePayload{
		Version:      1,
		ParentTaskID: parentTaskID,
		Children:     normalized,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal child completion signature payload: %w", err)
	}

	sum := sha256.Sum256(raw)
	return "parent-review:v1:" + hex.EncodeToString(sum[:]), nil
}

// ParentReviewCompletionSignatureFromMap is a map-input convenience wrapper for
// ParentReviewCompletionSignature.
func ParentReviewCompletionSignatureFromMap(parentTaskID string, completions map[string]time.Time) (string, error) {
	items := make([]ChildCompletion, 0, len(completions))
	for childID, completedAt := range completions {
		items = append(items, ChildCompletion{ChildID: childID, CompletedAt: completedAt})
	}
	return ParentReviewCompletionSignature(parentTaskID, items)
}

// ShouldRunParentReviewForSignature reports whether a parent review should run
// based on the latest persisted review signature.
func ShouldRunParentReviewForSignature(baseDir, parentTaskID, completionSignature string) (bool, error) {
	if baseDir == "" {
		return false, fmt.Errorf("baseDir required")
	}
	if parentTaskID == "" {
		return false, fmt.Errorf("task id required")
	}
	if completionSignature == "" {
		return false, fmt.Errorf("completion signature required")
	}

	latest, err := GetLatestReviewRun(baseDir, parentTaskID)
	if err != nil {
		return false, err
	}
	if latest == nil {
		return true, nil
	}
	if latest.ParentReviewCompletionSignature == "" {
		return true, nil
	}
	return latest.ParentReviewCompletionSignature != completionSignature, nil
}

type childCompletionSignatureEntry struct {
	ChildID           string `json:"childId"`
	CompletedUnixNano int64  `json:"completedUnixNano"`
}

type parentReviewCompletionSignaturePayload struct {
	Version      int                             `json:"version"`
	ParentTaskID string                          `json:"parentTaskId"`
	Children     []childCompletionSignatureEntry `json:"children"`
}

func normalizeChildCompletions(completions []ChildCompletion) ([]childCompletionSignatureEntry, error) {
	normalized := make([]childCompletionSignatureEntry, 0, len(completions))
	for _, completion := range completions {
		if completion.ChildID == "" {
			return nil, fmt.Errorf("child id required")
		}
		normalized = append(normalized, childCompletionSignatureEntry{
			ChildID:           completion.ChildID,
			CompletedUnixNano: completion.CompletedAt.UTC().UnixNano(),
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].ChildID != normalized[j].ChildID {
			return normalized[i].ChildID < normalized[j].ChildID
		}
		return normalized[i].CompletedUnixNano < normalized[j].CompletedUnixNano
	})

	for i := 1; i < len(normalized); i++ {
		if normalized[i-1].ChildID == normalized[i].ChildID {
			return nil, fmt.Errorf("duplicate child id %q", normalized[i].ChildID)
		}
	}

	return normalized, nil
}
