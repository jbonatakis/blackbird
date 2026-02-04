package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jbonatakis/blackbird/internal/tui"
)

const (
	filePickerMaxResults   = 200
	filePickerDisplayLimit = 10
)

type filePickerToken struct {
	start int
	end   int
	query string
}

func applyFilePickerToLine(line string) (string, bool, error) {
	cursor := 0
	for {
		token, ok := nextFilePickerToken(line, cursor)
		if !ok {
			return line, false, nil
		}

		if !shouldPromptFilePicker(token.query) {
			cursor = token.end
			continue
		}

		selected, canceled, err := promptFilePickerSelection(token.query)
		if err != nil {
			return "", false, err
		}
		if canceled {
			return "", true, nil
		}
		if selected == "" {
			cursor = token.end
			continue
		}

		replacement := "@" + strings.TrimPrefix(selected, "@")
		line = line[:token.start] + replacement + line[token.end:]
		cursor = token.start + len(replacement)
	}
}

func nextFilePickerToken(line string, start int) (filePickerToken, bool) {
	if start < 0 {
		start = 0
	}
	if start >= len(line) {
		return filePickerToken{}, false
	}

	for i := start; i < len(line); i++ {
		if line[i] != '@' {
			continue
		}
		j := i + 1
		for j < len(line) && !isFilePickerTokenDelimiter(line[j]) {
			j++
		}
		return filePickerToken{start: i, end: j, query: line[i+1 : j]}, true
	}
	return filePickerToken{}, false
}

func isFilePickerTokenDelimiter(b byte) bool {
	return b == ' ' || b == '\t'
}

func shouldPromptFilePicker(query string) bool {
	trimmed := strings.TrimSpace(strings.TrimPrefix(query, "@"))
	if trimmed == "" {
		return true
	}
	path := filepath.FromSlash(trimmed)
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	if info.IsDir() {
		return true
	}
	return false
}

func promptFilePickerSelection(query string) (string, bool, error) {
	current := strings.TrimSpace(strings.TrimPrefix(query, "@"))

	for {
		matches, err := tui.ListWorkspaceFiles(current, filePickerMaxResults)
		if err != nil {
			return "", false, err
		}

		label := "@"
		if current != "" {
			label = "@" + current
		}
		fmt.Fprintf(os.Stdout, "File picker for %s:\n", label)

		display := matches
		if len(display) > filePickerDisplayLimit {
			display = display[:filePickerDisplayLimit]
		}

		if len(display) == 0 {
			fmt.Fprintln(os.Stdout, "  (no matches)")
		} else {
			for i, match := range display {
				fmt.Fprintf(os.Stdout, "  %d) %s\n", i+1, match)
			}
			if len(matches) > len(display) {
				fmt.Fprintf(os.Stdout, "  ... %d more\n", len(matches)-len(display))
			}
		}

		line, err := promptLine("Select file (number, new query, blank to keep, /cancel)")
		if err != nil {
			return "", false, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return "", false, nil
		}
		if strings.EqualFold(trimmed, "/cancel") {
			return "", true, nil
		}

		if idx, err := strconv.Atoi(trimmed); err == nil {
			if idx >= 1 && idx <= len(display) {
				return display[idx-1], false, nil
			}
			fmt.Fprintln(os.Stdout, "invalid selection; try again")
			continue
		}

		current = strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
	}
}
