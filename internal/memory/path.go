package memory

import (
	"os"
	"path/filepath"
)

const (
	memoryDirName       = ".blackbird/memory"
	sessionFileName     = "session.json"
	traceDirName        = "trace"
	traceWALFileName    = "trace.wal"
	canonicalDirName    = "canonical"
	canonicalLogDefault = "canonical.json"
	artifactsDBFileName = "artifacts.db"
	indexDBFileName     = "index.db"
)

// MemoryRoot returns the memory root directory for a project.
func MemoryRoot(projectRoot string) string {
	if projectRoot != "" {
		return filepath.Join(projectRoot, memoryDirName)
	}
	wd, err := os.Getwd()
	if err != nil {
		return memoryDirName
	}
	return filepath.Join(wd, memoryDirName)
}

// SessionPath returns the session metadata path.
func SessionPath(projectRoot string) string {
	return filepath.Join(MemoryRoot(projectRoot), sessionFileName)
}

// TraceWALDir returns the directory that stores trace WAL files.
func TraceWALDir(projectRoot string) string {
	return filepath.Join(MemoryRoot(projectRoot), traceDirName)
}

// TraceWALPath returns the WAL path for a session. If sessionID is empty, a default filename is used.
func TraceWALPath(projectRoot string, sessionID string) string {
	fileName := traceWALFileName
	if sessionID != "" {
		fileName = sessionID + ".wal"
	}
	return filepath.Join(TraceWALDir(projectRoot), fileName)
}

// CanonicalLogDir returns the directory that stores canonical logs.
func CanonicalLogDir(projectRoot string) string {
	return filepath.Join(MemoryRoot(projectRoot), canonicalDirName)
}

// CanonicalLogPath returns the canonical log path for a run. If runID is empty, a default filename is used.
func CanonicalLogPath(projectRoot string, runID string) string {
	fileName := canonicalLogDefault
	if runID != "" {
		fileName = runID + ".json"
	}
	return filepath.Join(CanonicalLogDir(projectRoot), fileName)
}

// ArtifactsDBPath returns the artifacts database path.
func ArtifactsDBPath(projectRoot string) string {
	return filepath.Join(MemoryRoot(projectRoot), artifactsDBFileName)
}

// IndexDBPath returns the index database path.
func IndexDBPath(projectRoot string) string {
	return filepath.Join(MemoryRoot(projectRoot), indexDBFileName)
}
