package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ListRuns returns all run records for a task, sorted by StartedAt.
func ListRuns(baseDir, taskID string) ([]RunRecord, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("baseDir required")
	}
	if taskID == "" {
		return nil, fmt.Errorf("task id required")
	}

	dir := filepath.Join(baseDir, runsDirName, taskID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunRecord{}, nil
		}
		return nil, fmt.Errorf("read run directory: %w", err)
	}

	records := make([]RunRecord, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read run record: %w", err)
		}
		var record RunRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("decode run record: %w", err)
		}
		records = append(records, record)
	}

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].StartedAt.Before(records[j].StartedAt)
	})

	return records, nil
}

// LoadRun loads a specific run record by task ID and run ID.
func LoadRun(baseDir, taskID, runID string) (RunRecord, error) {
	if baseDir == "" {
		return RunRecord{}, fmt.Errorf("baseDir required")
	}
	if taskID == "" {
		return RunRecord{}, fmt.Errorf("task id required")
	}
	if runID == "" {
		return RunRecord{}, fmt.Errorf("run id required")
	}

	path := filepath.Join(baseDir, runsDirName, taskID, runID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return RunRecord{}, fmt.Errorf("read run record: %w", err)
	}

	var record RunRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return RunRecord{}, fmt.Errorf("decode run record: %w", err)
	}

	return record, nil
}

// GetLatestRun returns the most recent run for a task, or nil if none exist.
func GetLatestRun(baseDir, taskID string) (*RunRecord, error) {
	records, err := ListRuns(baseDir, taskID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	latest := records[len(records)-1]
	return &latest, nil
}
