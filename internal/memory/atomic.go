package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Best-effort: keep tmp in same directory for atomic rename.
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
			return fmt.Errorf("fsync memory directory: %w", err)
		}
	}

	return nil
}

// AtomicWriteFile writes a file atomically, syncing the directory on non-Windows platforms.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(path, data, perm)
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
