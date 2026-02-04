package execution

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type reviewSummaryLimits struct {
	MaxFiles         int
	MaxDiffStatBytes int
	MaxSnippets      int
	MaxSnippetLines  int
	MaxSnippetBytes  int
}

var defaultReviewSummaryLimits = reviewSummaryLimits{
	MaxFiles:         25,
	MaxDiffStatBytes: 4000,
	MaxSnippets:      3,
	MaxSnippetLines:  20,
	MaxSnippetBytes:  800,
}

const defaultReviewSummaryTimeout = 3 * time.Second

type reviewCommandRunner func(ctx context.Context, dir, name string, args ...string) ([]byte, error)

func maybeAttachReviewSummary(baseDir string, record *RunRecord) {
	if record == nil {
		return
	}
	if record.Status == RunStatusWaitingUser {
		return
	}
	if record.ReviewSummary != nil {
		return
	}

	summary := captureReviewSummary(baseDir)
	record.ReviewSummary = &summary
}

func captureReviewSummary(baseDir string) ReviewSummary {
	ctx, cancel := context.WithTimeout(context.Background(), defaultReviewSummaryTimeout)
	defer cancel()
	return captureReviewSummaryWith(ctx, baseDir, execReviewCommand, defaultReviewSummaryLimits)
}

func captureReviewSummaryWith(ctx context.Context, baseDir string, runner reviewCommandRunner, limits reviewSummaryLimits) ReviewSummary {
	summary, err := generateReviewSummary(ctx, baseDir, runner, limits)
	if err != nil {
		return ReviewSummary{}
	}
	return summary
}

func generateReviewSummary(ctx context.Context, baseDir string, runner reviewCommandRunner, limits reviewSummaryLimits) (ReviewSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(baseDir) == "" {
		return ReviewSummary{}, fmt.Errorf("baseDir required")
	}
	if runner == nil {
		return ReviewSummary{}, fmt.Errorf("command runner required")
	}

	limits = applyDefaultReviewSummaryLimits(limits)

	files, statusErr := gitStatusFiles(ctx, baseDir, runner)
	if limits.MaxFiles > 0 && len(files) > limits.MaxFiles {
		files = files[:limits.MaxFiles]
	}

	diffStat, diffErr := gitDiffStat(ctx, baseDir, runner, limits.MaxDiffStatBytes)

	snippets := buildSnippets(ctx, baseDir, runner, files, limits)

	if statusErr != nil && diffErr != nil {
		return ReviewSummary{}, errors.Join(statusErr, diffErr)
	}

	return ReviewSummary{
		Files:    files,
		DiffStat: diffStat,
		Snippets: snippets,
	}, nil
}

func applyDefaultReviewSummaryLimits(limits reviewSummaryLimits) reviewSummaryLimits {
	if limits.MaxFiles <= 0 {
		limits.MaxFiles = defaultReviewSummaryLimits.MaxFiles
	}
	if limits.MaxDiffStatBytes <= 0 {
		limits.MaxDiffStatBytes = defaultReviewSummaryLimits.MaxDiffStatBytes
	}
	if limits.MaxSnippets <= 0 {
		limits.MaxSnippets = defaultReviewSummaryLimits.MaxSnippets
	}
	if limits.MaxSnippetLines <= 0 {
		limits.MaxSnippetLines = defaultReviewSummaryLimits.MaxSnippetLines
	}
	if limits.MaxSnippetBytes <= 0 {
		limits.MaxSnippetBytes = defaultReviewSummaryLimits.MaxSnippetBytes
	}
	return limits
}

func execReviewCommand(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func gitStatusFiles(ctx context.Context, baseDir string, runner reviewCommandRunner) ([]string, error) {
	out, err := runner(ctx, baseDir, "git", "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseGitStatus(string(out)), nil
}

func gitDiffStat(ctx context.Context, baseDir string, runner reviewCommandRunner, maxBytes int) (string, error) {
	out, err := runner(ctx, baseDir, "git", "diff", "--stat", "HEAD")
	if err != nil {
		out, err = runner(ctx, baseDir, "git", "diff", "--stat")
		if err != nil {
			return "", err
		}
	}
	diffStat := strings.TrimSpace(string(out))
	return truncateString(diffStat, maxBytes), nil
}

func buildSnippets(ctx context.Context, baseDir string, runner reviewCommandRunner, files []string, limits reviewSummaryLimits) []ReviewSnippet {
	if limits.MaxSnippets <= 0 || limits.MaxSnippetLines <= 0 || limits.MaxSnippetBytes <= 0 {
		return nil
	}
	snippets := make([]ReviewSnippet, 0, limits.MaxSnippets)
	for _, file := range files {
		if len(snippets) >= limits.MaxSnippets {
			break
		}
		snippet, err := gitDiffSnippet(ctx, baseDir, runner, file, limits.MaxSnippetLines, limits.MaxSnippetBytes)
		if err != nil || snippet == "" {
			continue
		}
		snippets = append(snippets, ReviewSnippet{File: file, Snippet: snippet})
	}
	return snippets
}

func gitDiffSnippet(ctx context.Context, baseDir string, runner reviewCommandRunner, file string, maxLines, maxBytes int) (string, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", nil
	}

	out, err := runner(ctx, baseDir, "git", "diff", "-U2", "HEAD", "--", file)
	if err != nil {
		out, err = runner(ctx, baseDir, "git", "diff", "-U2", "--", file)
		if err != nil {
			return "", err
		}
	}
	return trimSnippet(string(out), maxLines, maxBytes), nil
}

func parseGitStatus(output string) []string {
	scanner := bufio.NewScanner(strings.NewReader(output))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	seen := make(map[string]struct{})
	var files []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}
		entry := strings.TrimSpace(line[3:])
		if entry == "" {
			continue
		}
		if arrow := strings.LastIndex(entry, "->"); arrow != -1 {
			entry = strings.TrimSpace(entry[arrow+2:])
		}
		if entry == "" {
			continue
		}
		if _, ok := seen[entry]; ok {
			continue
		}
		seen[entry] = struct{}{}
		files = append(files, entry)
	}
	return files
}

func trimSnippet(output string, maxLines, maxBytes int) string {
	snippet := strings.TrimSpace(output)
	if snippet == "" {
		return ""
	}

	if maxLines > 0 {
		lines := strings.Split(snippet, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
		}
		snippet = strings.Join(lines, "\n")
	}

	if maxBytes > 0 {
		snippet = truncateString(snippet, maxBytes)
	}

	return strings.TrimSpace(snippet)
}

func truncateString(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes]
}
