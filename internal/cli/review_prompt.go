package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	term "github.com/charmbracelet/x/term"
	"github.com/jbonatakis/blackbird/internal/execution"
)

var isTerminal = term.IsTerminal

var errReviewPromptCanceled = errors.New("review prompt canceled")

type reviewDecisionOption struct {
	Label            string
	Action           execution.DecisionState
	RequiresFeedback bool
}

func defaultReviewDecisionOptions() []reviewDecisionOption {
	return []reviewDecisionOption{
		{Label: "Approve and continue", Action: execution.DecisionStateApprovedContinue},
		{Label: "Approve and quit", Action: execution.DecisionStateApprovedQuit},
		{Label: "Request changes", Action: execution.DecisionStateChangesRequested, RequiresFeedback: true},
		{Label: "Reject changes", Action: execution.DecisionStateRejected},
	}
}

func printReviewPrompt(w io.Writer, taskID, title string, status execution.RunStatus, summary *execution.ReviewSummary) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Task review checkpoint")
	if strings.TrimSpace(title) != "" {
		fmt.Fprintf(w, "Task: %s - %s\n", taskID, title)
	} else {
		fmt.Fprintf(w, "Task: %s\n", taskID)
	}
	fmt.Fprintf(w, "Run status: %s\n", formatRunStatus(status))
	printReviewSummary(w, summary)
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Select an action (↑/↓ or j/k, enter to select):")
}

func formatRunStatus(status execution.RunStatus) string {
	switch status {
	case execution.RunStatusSuccess:
		return "success"
	case execution.RunStatusFailed:
		return "failed"
	case execution.RunStatusWaitingUser:
		return "waiting_user"
	default:
		if status == "" {
			return "unknown"
		}
		return string(status)
	}
}

func printReviewSummary(w io.Writer, summary *execution.ReviewSummary) {
	fmt.Fprintln(w, "Review summary:")
	if summary == nil {
		fmt.Fprintln(w, "No review summary available.")
		return
	}

	if len(summary.Files) == 0 {
		fmt.Fprintln(w, "Files: (none)")
	} else {
		fmt.Fprintln(w, "Files:")
		for _, file := range summary.Files {
			fmt.Fprintf(w, "- %s\n", file)
		}
	}

	diffStat := strings.TrimSpace(summary.DiffStat)
	if diffStat == "" {
		fmt.Fprintln(w, "Diffstat: (none)")
	} else {
		fmt.Fprintln(w, "Diffstat:")
		for _, line := range indentLines(diffStat, "  ") {
			fmt.Fprintln(w, line)
		}
	}

	if len(summary.Snippets) == 0 {
		fmt.Fprintln(w, "Snippets: (none)")
	} else {
		fmt.Fprintln(w, "Snippets:")
		for _, snippet := range summary.Snippets {
			if strings.TrimSpace(snippet.File) != "" {
				fmt.Fprintf(w, "File: %s\n", snippet.File)
			}
			if strings.TrimSpace(snippet.Snippet) == "" {
				continue
			}
			for _, line := range indentLines(snippet.Snippet, "  ") {
				fmt.Fprintln(w, line)
			}
		}
	}
}

func promptReviewDecision(options []reviewDecisionOption) (reviewDecisionOption, error) {
	if len(options) == 0 {
		return reviewDecisionOption{}, errors.New("no review options available")
	}
	if !isTerminal(os.Stdin.Fd()) {
		return promptReviewDecisionLine(options)
	}
	return promptReviewDecisionTTY(options)
}

func promptReviewDecisionLine(options []reviewDecisionOption) (reviewDecisionOption, error) {
	for {
		for i, option := range options {
			fmt.Fprintf(os.Stdout, "%d) %s\n", i+1, option.Label)
		}
		line, err := promptLine("Select option")
		if err != nil {
			return reviewDecisionOption{}, err
		}
		choice := strings.TrimSpace(line)
		if choice == "" {
			fmt.Fprintln(os.Stdout, "invalid selection; try again")
			continue
		}
		if idx, err := strconv.Atoi(choice); err == nil {
			if idx >= 1 && idx <= len(options) {
				return options[idx-1], nil
			}
		}
		for _, option := range options {
			if strings.EqualFold(choice, option.Label) {
				return option, nil
			}
		}
		fmt.Fprintln(os.Stdout, "invalid selection; try again")
	}
}

func promptReviewDecisionTTY(options []reviewDecisionOption) (reviewDecisionOption, error) {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return promptReviewDecisionLine(options)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	reader := bufio.NewReader(os.Stdin)
	selected := 0
	renderReviewDecisionOptions(os.Stdout, options, selected, false)
	lineCount := len(options)

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return reviewDecisionOption{}, err
		}
		switch b {
		case '\r', '\n':
			return options[selected], nil
		case 'j':
			if selected < len(options)-1 {
				selected++
			}
		case 'k':
			if selected > 0 {
				selected--
			}
		case 0x03:
			return reviewDecisionOption{}, errors.New("interrupted")
		case 0x1b:
			next, err := reader.ReadByte()
			if err != nil {
				continue
			}
			if next != '[' {
				continue
			}
			arrow, err := reader.ReadByte()
			if err != nil {
				continue
			}
			switch arrow {
			case 'A':
				if selected > 0 {
					selected--
				}
			case 'B':
				if selected < len(options)-1 {
					selected++
				}
			}
		default:
			continue
		}

		if lineCount > 0 {
			fmt.Fprintf(os.Stdout, "\x1b[%dA", lineCount)
		}
		renderReviewDecisionOptions(os.Stdout, options, selected, true)
	}
}

func renderReviewDecisionOptions(w io.Writer, options []reviewDecisionOption, selected int, clear bool) {
	for i, option := range options {
		if clear {
			fmt.Fprint(w, "\r\x1b[2K")
		}
		marker := " "
		if i == selected {
			marker = ">"
		}
		fmt.Fprintf(w, "%s %s\n", marker, option.Label)
	}
}

func promptReviewFeedback() (string, error) {
	for {
		fmt.Fprintln(os.Stdout, "Change request (end with a blank line, /cancel to go back, use @ to pick files):")
		var lines []string
		for {
			line, err := promptLine("")
			if err != nil {
				return "", err
			}
			trimmed := strings.TrimSpace(line)
			if strings.EqualFold(trimmed, "/cancel") {
				return "", errReviewPromptCanceled
			}
			if trimmed == "" {
				break
			}
			updated, canceled, err := applyFilePickerToLine(line)
			if err != nil {
				return "", err
			}
			if canceled {
				return "", errReviewPromptCanceled
			}
			lines = append(lines, updated)
		}
		feedback := strings.TrimSpace(strings.Join(lines, "\n"))
		if feedback == "" {
			fmt.Fprintln(os.Stdout, "change request cannot be empty")
			continue
		}
		return feedback, nil
	}
}
