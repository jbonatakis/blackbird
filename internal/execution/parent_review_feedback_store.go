package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const parentReviewFeedbackDirName = ".blackbird/parent-review-feedback"

// PendingParentReviewFeedback is feedback from a parent review run assigned to a child task.
type PendingParentReviewFeedback struct {
	ParentTaskID string    `json:"parentTaskId"`
	ReviewRunID  string    `json:"reviewRunId"`
	Feedback     string    `json:"feedback"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// UpsertPendingParentReviewFeedback writes pending parent-review feedback for a child task.
func UpsertPendingParentReviewFeedback(baseDir, childTaskID, parentTaskID, reviewRunID, feedback string) (PendingParentReviewFeedback, error) {
	return upsertPendingParentReviewFeedback(
		baseDir,
		childTaskID,
		parentTaskID,
		reviewRunID,
		feedback,
		time.Now().UTC(),
	)
}

func upsertPendingParentReviewFeedback(baseDir, childTaskID, parentTaskID, reviewRunID, feedback string, now time.Time) (PendingParentReviewFeedback, error) {
	parentTaskID = strings.TrimSpace(parentTaskID)
	if parentTaskID == "" {
		return PendingParentReviewFeedback{}, fmt.Errorf("parent task id required")
	}
	reviewRunID = strings.TrimSpace(reviewRunID)
	if reviewRunID == "" {
		return PendingParentReviewFeedback{}, fmt.Errorf("review run id required")
	}
	if strings.TrimSpace(feedback) == "" {
		return PendingParentReviewFeedback{}, fmt.Errorf("feedback required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	path, dir, err := parentReviewFeedbackPath(baseDir, childTaskID)
	if err != nil {
		return PendingParentReviewFeedback{}, err
	}

	existing, err := loadPendingParentReviewFeedbackPath(path)
	if err != nil {
		return PendingParentReviewFeedback{}, err
	}

	record := PendingParentReviewFeedback{
		ParentTaskID: parentTaskID,
		ReviewRunID:  reviewRunID,
		Feedback:     feedback,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if existing != nil && !existing.CreatedAt.IsZero() {
		record.CreatedAt = existing.CreatedAt
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return PendingParentReviewFeedback{}, fmt.Errorf("create pending parent review feedback directory: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return PendingParentReviewFeedback{}, fmt.Errorf("marshal pending parent review feedback: %w", err)
	}
	data = append(data, '\n')

	if err := atomicWriteFile(path, data, 0o644); err != nil {
		return PendingParentReviewFeedback{}, fmt.Errorf("write pending parent review feedback: %w", err)
	}

	return record, nil
}

// LoadPendingParentReviewFeedback loads pending parent-review feedback for a child task.
// It returns nil,nil when no feedback exists.
func LoadPendingParentReviewFeedback(baseDir, childTaskID string) (*PendingParentReviewFeedback, error) {
	path, _, err := parentReviewFeedbackPath(baseDir, childTaskID)
	if err != nil {
		return nil, err
	}
	return loadPendingParentReviewFeedbackPath(path)
}

func loadPendingParentReviewFeedbackPath(path string) (*PendingParentReviewFeedback, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read pending parent review feedback: %w", err)
	}

	var record PendingParentReviewFeedback
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("decode pending parent review feedback: %w", err)
	}

	return &record, nil
}

// ClearPendingParentReviewFeedback clears pending parent-review feedback for a child task.
func ClearPendingParentReviewFeedback(baseDir, childTaskID string) error {
	path, _, err := parentReviewFeedbackPath(baseDir, childTaskID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear pending parent review feedback: %w", err)
	}
	return nil
}

func parentReviewFeedbackPath(baseDir, childTaskID string) (string, string, error) {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		return "", "", fmt.Errorf("baseDir required")
	}
	childTaskID = strings.TrimSpace(childTaskID)
	if childTaskID == "" {
		return "", "", fmt.Errorf("child task id required")
	}
	if childTaskID == "." || childTaskID == ".." {
		return "", "", fmt.Errorf("child task id required")
	}
	if strings.ContainsAny(childTaskID, `/\`) {
		return "", "", fmt.Errorf("child task id must be a single path segment")
	}

	dir := filepath.Join(baseDir, parentReviewFeedbackDirName)
	path := filepath.Join(dir, childTaskID+".json")

	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return "", "", fmt.Errorf("resolve pending parent review feedback path: %w", err)
	}
	parentPrefix := ".." + string(os.PathSeparator)
	if rel == ".." || strings.HasPrefix(rel, parentPrefix) {
		return "", "", fmt.Errorf("child task id resolves outside pending parent review feedback directory")
	}

	return path, dir, nil
}
