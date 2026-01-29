package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
)

var timeNow = time.Now

func RenderExecutionView(model Model) string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	active := findActiveRun(model.runData)

	var b strings.Builder
	writeSectionHeader(&b, headerStyle, "Active Run")
	if active == nil {
		b.WriteString(mutedStyle.Render("No active runs."))
		b.WriteString("\n\n")
	} else {
		writeLabeledLine(&b, labelStyle, "Task", active.TaskID)
		status := renderRunStatus(active.Status)
		b.WriteString(labelStyle.Render("Status: "))
		b.WriteString(status)
		b.WriteString("\n")
		writeLabeledLine(&b, labelStyle, "Elapsed", formatElapsed(active.StartedAt, active.CompletedAt))
		if active.CompletedAt != nil {
			exitCode := "-"
			if active.ExitCode != nil {
				exitCode = fmt.Sprintf("%d", *active.ExitCode)
			}
			writeLabeledLine(&b, labelStyle, "Exit code", exitCode)
		}
		b.WriteString("\n")
	}

	writeSectionHeader(&b, headerStyle, "Log Output")
	if active == nil {
		b.WriteString(mutedStyle.Render("(no logs)"))
		b.WriteString("\n\n")
	} else {
		writeLogExcerpt(&b, "stdout", active.Stdout, mutedStyle)
		writeLogExcerpt(&b, "stderr", active.Stderr, mutedStyle)
		b.WriteString("\n")
	}

	writeSectionHeader(&b, headerStyle, "Task Summary")
	readyCount := len(execution.ReadyTasks(model.plan))
	blockedCount := blockedCount(model.plan)
	writeLabeledLine(&b, labelStyle, "Ready", fmt.Sprintf("%d", readyCount))
	writeLabeledLine(&b, labelStyle, "Blocked", fmt.Sprintf("%d", blockedCount))

	content := strings.TrimRight(b.String(), "\n")
	return applyViewport(model, content)
}

func findActiveRun(runData map[string]execution.RunRecord) *execution.RunRecord {
	if len(runData) == 0 {
		return nil
	}
	var selected *execution.RunRecord
	for _, record := range runData {
		if record.Status != execution.RunStatusRunning && record.Status != execution.RunStatusWaitingUser {
			continue
		}
		if selected == nil || record.StartedAt.After(selected.StartedAt) {
			copy := record
			selected = &copy
		}
	}
	return selected
}

func renderRunStatus(status execution.RunStatus) string {
	style := lipgloss.NewStyle()
	switch status {
	case execution.RunStatusSuccess:
		style = style.Foreground(lipgloss.Color("42"))
	case execution.RunStatusFailed:
		style = style.Foreground(lipgloss.Color("196"))
	case execution.RunStatusWaitingUser:
		style = style.Foreground(lipgloss.Color("214"))
	case execution.RunStatusRunning:
		style = style.Foreground(lipgloss.Color("39"))
	default:
		style = style.Foreground(lipgloss.Color("240"))
	}
	return style.Render(string(status))
}

func formatElapsed(startedAt time.Time, completedAt *time.Time) string {
	end := timeNow()
	if completedAt != nil {
		end = *completedAt
	}
	if end.Before(startedAt) {
		end = startedAt
	}
	duration := end.Sub(startedAt).Truncate(time.Second)
	totalSeconds := int(duration.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func writeLogExcerpt(b *strings.Builder, label string, content string, mutedStyle lipgloss.Style) {
	lines := tailLines(content, 20)
	b.WriteString(strings.ToUpper(label))
	b.WriteString(":\n")
	if len(lines) == 0 {
		b.WriteString(mutedStyle.Render("(empty)"))
		b.WriteString("\n")
		return
	}
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func tailLines(content string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimRight(normalized, "\n")
	if strings.TrimSpace(normalized) == "" {
		return nil
	}
	lines := strings.Split(normalized, "\n")
	if len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	return lines
}
