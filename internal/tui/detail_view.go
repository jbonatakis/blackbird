package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func RenderDetailView(model Model) string {
	if model.selectedID == "" {
		return emptyDetailView("No item selected.")
	}
	it, ok := model.plan.Items[model.selectedID]
	if !ok {
		return emptyDetailView(fmt.Sprintf("Unknown item %q.", model.selectedID))
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var b strings.Builder
	writeSectionHeader(&b, headerStyle, "Item")
	writeLabeledLine(&b, labelStyle, "ID", it.ID)
	writeLabeledLine(&b, labelStyle, "Title", it.Title)
	writeLabeledLine(&b, labelStyle, "Status", string(it.Status))
	writeLabeledLine(&b, labelStyle, "Created", formatTimestamp(it.CreatedAt))
	writeLabeledLine(&b, labelStyle, "Updated", formatTimestamp(it.UpdatedAt))
	b.WriteString("\n")

	writeSectionHeader(&b, headerStyle, "Description")
	if strings.TrimSpace(it.Description) == "" {
		b.WriteString(mutedStyle.Render("(none)") + "\n\n")
	} else {
		b.WriteString(it.Description)
		b.WriteString("\n\n")
	}

	writeSectionHeader(&b, headerStyle, "Acceptance criteria")
	if len(it.AcceptanceCriteria) == 0 {
		b.WriteString(mutedStyle.Render("(none)") + "\n\n")
	} else {
		for _, ac := range it.AcceptanceCriteria {
			b.WriteString("- ")
			b.WriteString(ac)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	writeSectionHeader(&b, headerStyle, "Dependencies")
	if len(it.Deps) == 0 {
		b.WriteString(mutedStyle.Render("(none)") + "\n\n")
	} else {
		for _, depID := range it.Deps {
			dep, ok := model.plan.Items[depID]
			if !ok {
				b.WriteString(fmt.Sprintf("- %s [unknown]\n", depID))
				continue
			}
			b.WriteString(fmt.Sprintf("- %s [%s] %s\n", depID, dep.Status, dep.Title))
		}
		b.WriteString("\n")
	}

	dependents := plan.Dependents(model.plan, it.ID)
	writeSectionHeader(&b, headerStyle, "Dependents")
	if len(dependents) == 0 {
		b.WriteString(mutedStyle.Render("(none)") + "\n\n")
	} else {
		for _, depID := range dependents {
			dep, ok := model.plan.Items[depID]
			if !ok {
				b.WriteString(fmt.Sprintf("- %s [unknown]\n", depID))
				continue
			}
			b.WriteString(fmt.Sprintf("- %s [%s] %s\n", depID, dep.Status, dep.Title))
		}
		b.WriteString("\n")
	}

	unmet := plan.UnmetDeps(model.plan, it)
	depsOK := len(unmet) == 0
	actionable := it.Status == plan.StatusTodo && depsOK
	writeSectionHeader(&b, headerStyle, "Readiness")
	if depsOK {
		b.WriteString("- deps satisfied: yes\n")
	} else {
		b.WriteString(fmt.Sprintf("- deps satisfied: no (unmet: %s)\n", strings.Join(unmet, ", ")))
	}
	if it.Status == plan.StatusBlocked && depsOK {
		b.WriteString("- manually blocked: yes\n")
	}
	b.WriteString(fmt.Sprintf("- actionable now: %v\n\n", actionable))

	writeSectionHeader(&b, headerStyle, "Prompt")
	if strings.TrimSpace(it.Prompt) == "" {
		b.WriteString(mutedStyle.Render("(empty)") + "\n")
	} else {
		b.WriteString(it.Prompt)
		if !strings.HasSuffix(it.Prompt, "\n") {
			b.WriteString("\n")
		}
	}

	content := strings.TrimRight(b.String(), "\n")
	return applyViewport(model, content)
}

func emptyDetailView(message string) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	return style.Render(message)
}

func writeSectionHeader(b *strings.Builder, style lipgloss.Style, title string) {
	b.WriteString(style.Render(title))
	b.WriteString("\n")
}

func writeLabeledLine(b *strings.Builder, labelStyle lipgloss.Style, label string, value string) {
	b.WriteString(labelStyle.Render(label + ": "))
	b.WriteString(value)
	b.WriteString("\n")
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return "(unknown)"
	}
	return t.UTC().Format(time.RFC3339)
}

// applyViewport renders content in a viewport. model.windowHeight is the pane's
// content height (the Height passed to renderPane); the viewport fills that area.
func applyViewport(model Model, content string) string {
	height := model.windowHeight
	if height < 0 {
		height = 0
	}
	width := model.windowWidth
	if height <= 0 || width <= 0 {
		return content
	}
	view := viewport.New(width, height)
	view.SetContent(content)
	offset := model.detailOffset
	if offset < 0 {
		offset = 0
	}
	totalLines := len(strings.Split(content, "\n"))
	maxOffset := totalLines - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	view.YOffset = offset
	return view.View()
}
