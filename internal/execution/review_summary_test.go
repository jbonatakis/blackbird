package execution

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestGenerateReviewSummaryBounds(t *testing.T) {
	statusOutput := strings.Join([]string{
		" M file1.go",
		" M file2.go",
		" M file3.go",
		" M file4.go",
		"?? newfile.txt",
		"R  old.go -> renamed.go",
	}, "\n")
	diffOutput := "file1.go | 10 ++++++++++\nfile2.go | 3 +++\nfile3.go | 2 ++\n"
	snippetOutput := strings.Join([]string{
		"diff --git a/file1.go b/file1.go",
		"@@ -1,2 +1,2 @@",
		"-old line",
		"+new line",
		" another line",
	}, "\n")

	runner := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		if name != "git" {
			return nil, fmt.Errorf("unexpected command: %s", name)
		}
		key := strings.Join(args, " ")
		switch {
		case key == "status --porcelain":
			return []byte(statusOutput), nil
		case key == "diff --stat HEAD":
			return []byte(diffOutput), nil
		case strings.HasPrefix(key, "diff -U2 HEAD -- "):
			return []byte(snippetOutput), nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", key)
		}
	}

	limits := reviewSummaryLimits{
		MaxFiles:         3,
		MaxDiffStatBytes: 12,
		MaxSnippets:      2,
		MaxSnippetLines:  2,
		MaxSnippetBytes:  30,
	}

	summary, err := generateReviewSummary(context.Background(), "/tmp", runner, limits)
	if err != nil {
		t.Fatalf("generateReviewSummary: %v", err)
	}

	if len(summary.Files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(summary.Files))
	}
	if summary.Files[0] != "file1.go" || summary.Files[1] != "file2.go" || summary.Files[2] != "file3.go" {
		t.Fatalf("unexpected files: %v", summary.Files)
	}

	if len(summary.DiffStat) > limits.MaxDiffStatBytes {
		t.Fatalf("diffstat not truncated: %d bytes", len(summary.DiffStat))
	}

	if len(summary.Snippets) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(summary.Snippets))
	}
	for _, snippet := range summary.Snippets {
		lines := strings.Split(snippet.Snippet, "\n")
		if len(lines) > limits.MaxSnippetLines {
			t.Fatalf("snippet lines not truncated: %d", len(lines))
		}
		if len(snippet.Snippet) > limits.MaxSnippetBytes {
			t.Fatalf("snippet bytes not truncated: %d", len(snippet.Snippet))
		}
	}
}

func TestCaptureReviewSummaryWithSuccess(t *testing.T) {
	statusOutput := strings.Join([]string{
		" M file1.go",
		"?? file2.go",
	}, "\n")
	diffOutput := "file1.go | 1 +\nfile2.go | 2 ++\n"
	snippetOutput := strings.Join([]string{
		"diff --git a/file1.go b/file1.go",
		"@@ -1 +1 @@",
		"-old",
		"+new",
	}, "\n")

	runner := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		if name != "git" {
			return nil, fmt.Errorf("unexpected command: %s", name)
		}
		key := strings.Join(args, " ")
		switch key {
		case "status --porcelain":
			return []byte(statusOutput), nil
		case "diff --stat HEAD":
			return []byte(diffOutput), nil
		case "diff -U2 HEAD -- file1.go":
			return []byte(snippetOutput), nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", key)
		}
	}

	limits := reviewSummaryLimits{
		MaxFiles:         2,
		MaxDiffStatBytes: 200,
		MaxSnippets:      1,
		MaxSnippetLines:  4,
		MaxSnippetBytes:  200,
	}

	summary := captureReviewSummaryWith(context.Background(), "/tmp", runner, limits)
	if len(summary.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(summary.Files))
	}
	if summary.DiffStat == "" {
		t.Fatalf("expected diffstat to be captured")
	}
	if len(summary.Snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(summary.Snippets))
	}
	if summary.Snippets[0].File != "file1.go" {
		t.Fatalf("expected snippet for file1.go, got %s", summary.Snippets[0].File)
	}
}

func TestCaptureReviewSummaryFallback(t *testing.T) {
	runner := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("status failed")
	}

	summary := captureReviewSummaryWith(context.Background(), "/tmp", runner, reviewSummaryLimits{})
	if len(summary.Files) != 0 || summary.DiffStat != "" || len(summary.Snippets) != 0 {
		t.Fatalf("expected empty summary on failure, got %#v", summary)
	}

	partialRunner := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		if key == "status --porcelain" {
			return nil, fmt.Errorf("status failed")
		}
		if key == "diff --stat HEAD" {
			return []byte("file1.go | 1 +"), nil
		}
		return nil, fmt.Errorf("unexpected command: %s", key)
	}

	summary, err := generateReviewSummary(context.Background(), "/tmp", partialRunner, reviewSummaryLimits{MaxDiffStatBytes: 100})
	if err != nil {
		t.Fatalf("expected partial summary, got error: %v", err)
	}
	if summary.DiffStat == "" {
		t.Fatalf("expected diffstat from partial summary")
	}
}
