package execution

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const runsDirName = ".blackbird/runs"

// SaveRun writes a run record to disk using an atomic write pattern.
func SaveRun(baseDir string, record RunRecord) error {
	if baseDir == "" {
		return fmt.Errorf("baseDir required")
	}
	if record.TaskID == "" {
		return fmt.Errorf("task id required")
	}
	if record.ID == "" {
		return fmt.Errorf("run id required")
	}

	path := filepath.Join(baseDir, runsDirName, record.TaskID, record.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create run directory: %w", err)
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run record: %w", err)
	}
	data = append(data, '\n')

	if err := atomicWriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write run record: %w", err)
	}

	return nil
}

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, tmpPattern(filepath.Base(path)))
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanupTmp := true

	defer func() {
		_ = tmp.Close()
		if cleanupTmp {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file into place: %w", err)
	}
	cleanupTmp = false

	if runtime.GOOS != "windows" {
		if err := fsyncDir(dir); err != nil {
			return fmt.Errorf("fsync run directory: %w", err)
		}
	}

	return nil
}

func fsyncDir(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func tmpPattern(base string) string {
	return fmt.Sprintf(".%s.*", base)
}
