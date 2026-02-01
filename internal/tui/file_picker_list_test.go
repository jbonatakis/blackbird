package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListWorkspaceFilesFiltersAndSkipsNoise(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "internal", "tui", "model.go"))
	writeTestFile(t, filepath.Join(dir, "internal", "tui", "file_picker.go"))
	writeTestFile(t, filepath.Join(dir, "docs", "spec.md"))
	writeTestFile(t, filepath.Join(dir, ".git", "config"))
	writeTestFile(t, filepath.Join(dir, ".blackbird", "agent.json"))

	restore := chdirTemp(t, dir)
	defer restore()

	matches, err := listWorkspaceFiles("internal/", 500)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsString(matches, "internal/tui/model.go") {
		t.Fatalf("expected internal file, got %v", matches)
	}
	if containsString(matches, "docs/spec.md") {
		t.Fatalf("did not expect docs file when filtering by prefix, got %v", matches)
	}
	if containsString(matches, ".git/config") {
		t.Fatalf("did not expect .git entry in matches, got %v", matches)
	}
	if containsString(matches, ".blackbird/agent.json") {
		t.Fatalf("did not expect .blackbird entry in matches, got %v", matches)
	}
	for _, match := range matches {
		if strings.Contains(match, "\\") {
			t.Fatalf("expected forward slashes in %q", match)
		}
		if strings.HasPrefix(match, "/") {
			t.Fatalf("expected relative path without leading slash, got %q", match)
		}
	}
}

func TestListWorkspaceFilesRespectsLimit(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "a.txt"))
	writeTestFile(t, filepath.Join(dir, "b.txt"))
	writeTestFile(t, filepath.Join(dir, "c.txt"))

	restore := chdirTemp(t, dir)
	defer restore()

	matches, err := listWorkspaceFiles("", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d (%v)", len(matches), matches)
	}
}

func TestFilterFilePickerMatchesOrdersAndLimits(t *testing.T) {
	files := []string{
		"b/notes.txt",
		"a/zeta.txt",
		"a/alpha.txt",
	}

	matches := filterFilePickerMatches("a/", files, 2)
	expected := []string{"a/alpha.txt", "a/zeta.txt"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d (%v)", len(expected), len(matches), matches)
	}
	for i, match := range matches {
		if match != expected[i] {
			t.Fatalf("expected match %d to be %q, got %q", i, expected[i], match)
		}
	}
}

func TestFilterFilePickerMatchesEmptyQuery(t *testing.T) {
	files := []string{
		"b.go",
		"a.go",
		"c.go",
	}

	matches := filterFilePickerMatches("", files, 2)
	expected := []string{"a.go", "b.go"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d (%v)", len(expected), len(matches), matches)
	}
	for i, match := range matches {
		if match != expected[i] {
			t.Fatalf("expected match %d to be %q, got %q", i, expected[i], match)
		}
	}
}

func TestFilterFilePickerMatchesNormalizesSlashes(t *testing.T) {
	files := []string{
		"internal\\tui\\model.go",
		"internal/tui/file_picker.go",
	}

	matches := filterFilePickerMatches("internal/tui/", files, 10)
	expected := []string{"internal/tui/file_picker.go", "internal/tui/model.go"}
	if len(matches) != len(expected) {
		t.Fatalf("expected %d matches, got %d (%v)", len(expected), len(matches), matches)
	}
	for i, match := range matches {
		if match != expected[i] {
			t.Fatalf("expected match %d to be %q, got %q", i, expected[i], match)
		}
	}
}

func writeTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}

func chdirTemp(t *testing.T, dir string) func() {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	return func() {
		_ = os.Chdir(wd)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
