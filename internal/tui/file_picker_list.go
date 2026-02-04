package tui

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const filePickerDefaultMaxResults = 500

var filePickerSkipDirs = map[string]struct{}{
	".blackbird": {},
	".git":       {},
}

var errFilePickerLimit = errors.New("file picker limit reached")

func listWorkspaceFiles(query string, limit int) ([]string, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return listWorkspaceFilesFromRoot(root, query, limit)
}

func listWorkspaceFilesFromRoot(root string, query string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = filePickerDefaultMaxResults
	}
	normalizedQuery := normalizeFilePickerPath(query)
	matches := make([]string, 0, minInt(limit, 64))

	walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if _, skip := filePickerSkipDirs[entry.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = normalizeFilePickerPath(rel)
		if normalizedQuery != "" && !strings.HasPrefix(rel, normalizedQuery) {
			return nil
		}

		matches = append(matches, rel)
		if len(matches) >= limit {
			return errFilePickerLimit
		}
		return nil
	})
	if walkErr != nil {
		if errors.Is(walkErr, errFilePickerLimit) {
			return filterFilePickerMatches(query, matches, limit), nil
		}
		return nil, walkErr
	}

	return filterFilePickerMatches(query, matches, limit), nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func filterFilePickerMatches(query string, files []string, limit int) []string {
	if limit <= 0 {
		limit = filePickerDefaultMaxResults
	}

	normalizedQuery := normalizeFilePickerPath(query)
	matches := make([]string, 0, minInt(limit, len(files)))

	for _, file := range files {
		normalizedFile := normalizeFilePickerPath(file)
		if normalizedQuery != "" && !strings.HasPrefix(normalizedFile, normalizedQuery) {
			continue
		}
		matches = append(matches, normalizedFile)
	}

	sort.Strings(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches
}

func normalizeFilePickerPath(value string) string {
	return strings.ReplaceAll(filepath.ToSlash(value), "\\", "/")
}

// ListWorkspaceFiles exposes workspace file matching for non-TUI callers.
func ListWorkspaceFiles(query string, limit int) ([]string, error) {
	return listWorkspaceFiles(query, limit)
}
