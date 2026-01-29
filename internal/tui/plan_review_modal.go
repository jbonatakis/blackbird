package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
)

// PlanReviewMode represents the current mode in the review flow
type PlanReviewMode int

const (
	ReviewModeChooseAction PlanReviewMode = iota
	ReviewModeRevisionPrompt
)

// PlanReviewForm represents the state of the plan review modal
type PlanReviewForm struct {
	mode             PlanReviewMode
	plan             plan.WorkGraph
	selectedAction   int // 0=Accept, 1=Revise, 2=Reject
	revisionTextarea textarea.Model
	width            int
	height           int
	revisionCount    int
}

// NewPlanReviewForm creates a new plan review form
func NewPlanReviewForm(generatedPlan plan.WorkGraph, revisionCount int) PlanReviewForm {
	// Initialize revision textarea
	revisionTA := textarea.New()
	revisionTA.Placeholder = "Enter revision request (describe desired changes)..."
	revisionTA.Focus()
	revisionTA.CharLimit = 2000
	revisionTA.SetWidth(60)
	revisionTA.SetHeight(4)

	return PlanReviewForm{
		mode:             ReviewModeChooseAction,
		plan:             generatedPlan,
		selectedAction:   0, // Default to Accept
		revisionTextarea: revisionTA,
		width:            80,
		height:           30,
		revisionCount:    revisionCount,
	}
}

// SetSize updates the form dimensions
func (f *PlanReviewForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	// Adjust textarea width
	fieldWidth := width - 10
	if fieldWidth > 80 {
		fieldWidth = 80
	}
	if fieldWidth < 40 {
		fieldWidth = 40
	}
	f.revisionTextarea.SetWidth(fieldWidth)
}

// Update handles form updates
func (f PlanReviewForm) Update(msg tea.Msg) (PlanReviewForm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if f.mode == ReviewModeChooseAction {
			switch msg.String() {
			case "up", "k":
				if f.selectedAction > 0 {
					f.selectedAction--
				}
				return f, nil
			case "down", "j":
				if f.selectedAction < 2 {
					f.selectedAction++
				}
				return f, nil
			case "1":
				f.selectedAction = 0
				return f, nil
			case "2":
				// Only allow revise if not exceeded limit
				if f.revisionCount < maxGenerateRevisions {
					f.selectedAction = 1
				}
				return f, nil
			case "3":
				f.selectedAction = 2
				return f, nil
			case "enter":
				// Handle action selection
				return f, nil
			}
		} else if f.mode == ReviewModeRevisionPrompt {
			switch msg.String() {
			case "enter":
				// Don't submit on enter if shift not held or textarea still has focus
				// Let textarea handle it
				f.revisionTextarea, cmd = f.revisionTextarea.Update(msg)
				return f, cmd
			case "ctrl+s":
				// Submit revision request
				return f, nil
			default:
				// Pass to textarea
				f.revisionTextarea, cmd = f.revisionTextarea.Update(msg)
				return f, cmd
			}
		}
	}

	return f, nil
}

// GetAction returns the currently selected action (0=Accept, 1=Revise, 2=Reject)
func (f PlanReviewForm) GetAction() int {
	return f.selectedAction
}

// GetRevisionRequest returns the revision request text
func (f PlanReviewForm) GetRevisionRequest() string {
	return strings.TrimSpace(f.revisionTextarea.Value())
}

// CanRevise returns true if another revision is allowed
func (f PlanReviewForm) CanRevise() bool {
	return f.revisionCount < maxGenerateRevisions
}

const maxGenerateRevisions = 1

// HandlePlanReviewKey handles key presses in plan-review mode
func HandlePlanReviewKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.planReviewForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	// Handle action selection mode
	if m.planReviewForm.mode == ReviewModeChooseAction {
		switch msg.String() {
		case "enter":
			action := m.planReviewForm.GetAction()
			switch action {
			case 0: // Accept
				// Save plan and return to main view
				return acceptPlan(m)
			case 1: // Revise
				if !m.planReviewForm.CanRevise() {
					// Show error - too many revisions
					m.actionOutput = &ActionOutput{
						Message: "Revision limit reached",
						IsError: true,
					}
					m.actionMode = ActionModeNone
					m.planReviewForm = nil
					return m, nil
				}
				// Switch to revision prompt mode
				updatedForm := *m.planReviewForm
				updatedForm.mode = ReviewModeRevisionPrompt
				updatedForm.revisionTextarea.Focus()
				m.planReviewForm = &updatedForm
				return m, nil
			case 2: // Reject
				// Discard plan and return to main view
				m.actionMode = ActionModeNone
				m.planReviewForm = nil
				m.actionOutput = &ActionOutput{
					Message: "Plan generation cancelled",
					IsError: false,
				}
				return m, nil
			}
		default:
			// Update form with navigation
			updatedForm, cmd := m.planReviewForm.Update(msg)
			m.planReviewForm = &updatedForm
			return m, cmd
		}
	}

	// Handle revision prompt mode
	if m.planReviewForm.mode == ReviewModeRevisionPrompt {
		switch msg.String() {
		case "ctrl+s", "ctrl+enter":
			// Submit revision request
			revisionRequest := m.planReviewForm.GetRevisionRequest()
			if strings.TrimSpace(revisionRequest) == "" {
				// Show validation error
				return m, nil
			}

			// Build refine request with current plan
			revisionCount := m.planReviewForm.revisionCount + 1
			currentPlan := m.planReviewForm.plan

			// Close modal and start refining
			m.actionMode = ActionModeNone
			m.planReviewForm = nil
			m.actionInProgress = true
			m.actionName = "Refining plan..."

			// Store revision count for next review
			m.pendingPlanRequest.questionRound = revisionCount

			return m, tea.Batch(
				RefinePlanInMemory(context.Background(), revisionRequest, currentPlan),
				spinnerTickCmd(),
			)
		case "esc":
			// Cancel revision, go back to action selection
			updatedForm := *m.planReviewForm
			updatedForm.mode = ReviewModeChooseAction
			updatedForm.revisionTextarea.Blur()
			m.planReviewForm = &updatedForm
			return m, nil
		default:
			// Update textarea
			updatedForm, cmd := m.planReviewForm.Update(msg)
			m.planReviewForm = &updatedForm
			return m, cmd
		}
	}

	return m, nil
}

// acceptPlan saves the plan and returns to main view
func acceptPlan(m Model) (Model, tea.Cmd) {
	if m.planReviewForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	// Update the model's plan with the reviewed plan
	m.plan = m.planReviewForm.plan
	m.ensureSelectionVisible()

	// Close modal
	m.actionMode = ActionModeNone
	m.planReviewForm = nil

	// Clear pending request
	m.pendingPlanRequest = PendingPlanRequest{}

	m.actionOutput = &ActionOutput{
		Message: "Plan accepted and saved",
		IsError: false,
	}

	// Save plan to disk
	return m, SavePlanCmd(m.plan)
}

// RenderPlanReviewModal renders the plan review modal
func RenderPlanReviewModal(m Model, form PlanReviewForm) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("69")).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	unselectedStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("15"))
	disabledStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("242"))

	var lines []string

	// Mode: Choose action
	if form.mode == ReviewModeChooseAction {
		title := titleStyle.Render("Review Generated Plan")
		lines = append(lines, title)
		lines = append(lines, "")

		// Plan summary
		itemCount := len(form.plan.Items)
		summary := textStyle.Render(fmt.Sprintf("Plan contains %d items", itemCount))
		lines = append(lines, summary)
		lines = append(lines, "")

		// Top-level features preview
		roots := rootIDs(form.plan)
		if len(roots) > 0 {
			lines = append(lines, labelStyle.Render("Top-level features:"))
			for i, rootID := range roots {
				if i >= 5 {
					lines = append(lines, labelStyle.Render(fmt.Sprintf("  ... and %d more", len(roots)-5)))
					break
				}
				item, ok := form.plan.Items[rootID]
				if ok {
					lines = append(lines, textStyle.Render(fmt.Sprintf("  • %s", item.Title)))
				}
			}
			lines = append(lines, "")
		}

		// Action choices
		lines = append(lines, labelStyle.Render("Choose an action:"))
		lines = append(lines, "")

		// Accept
		acceptLabel := "1. Accept"
		if form.selectedAction == 0 {
			lines = append(lines, selectedStyle.Render(acceptLabel))
		} else {
			lines = append(lines, unselectedStyle.Render(acceptLabel))
		}

		// Revise
		reviseLabel := "2. Revise"
		if !form.CanRevise() {
			lines = append(lines, disabledStyle.Render(reviseLabel+" (limit reached)"))
		} else if form.selectedAction == 1 {
			lines = append(lines, selectedStyle.Render(reviseLabel))
		} else {
			lines = append(lines, unselectedStyle.Render(reviseLabel))
		}

		// Reject
		rejectLabel := "3. Reject"
		if form.selectedAction == 2 {
			lines = append(lines, selectedStyle.Render(rejectLabel))
		} else {
			lines = append(lines, unselectedStyle.Render(rejectLabel))
		}

		lines = append(lines, "")
		helpText := helpStyle.Render("[↑/↓]navigate [1-3]select [enter]confirm [esc]cancel")
		lines = append(lines, helpText)
	}

	// Mode: Revision prompt
	if form.mode == ReviewModeRevisionPrompt {
		title := titleStyle.Render("Revise Plan")
		lines = append(lines, title)
		lines = append(lines, "")

		revisionLabel := labelStyle.Render("Describe the changes you want:")
		lines = append(lines, revisionLabel)
		lines = append(lines, form.revisionTextarea.View())
		lines = append(lines, "")

		// Validation
		if strings.TrimSpace(form.revisionTextarea.Value()) == "" {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			lines = append(lines, errorStyle.Render("⚠ Revision request cannot be empty"))
			lines = append(lines, "")
		}

		helpText := helpStyle.Render("[ctrl+s or ctrl+enter]submit [esc]back")
		lines = append(lines, helpText)
	}

	// Calculate modal width
	modalWidth := form.width
	if modalWidth > 90 {
		modalWidth = 90
	}
	if modalWidth < 60 {
		modalWidth = 60
	}
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	// Center the modal
	if m.windowHeight > 0 {
		topPadding := (m.windowHeight - lipgloss.Height(modal)) / 2
		if topPadding > 0 {
			padding := lipgloss.NewStyle().PaddingTop(topPadding).Render(modal)
			return padding
		}
	}

	return modal
}

// RefinePlanInMemory refines an existing plan with a change request
func RefinePlanInMemory(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) tea.Cmd {
	return func() tea.Msg {
		// Create agent runtime from environment
		runtime, err := agent.NewRuntimeFromEnv()
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Prepare request metadata with JSON schema
		requestMeta := agent.RequestMetadata{
			JSONSchema: defaultPlanJSONSchema(),
		}

		// Build the agent request
		req := agent.Request{
			SchemaVersion: agent.SchemaVersion,
			Type:          agent.RequestPlanRefine,
			SystemPrompt:  defaultPlanSystemPrompt(),
			ChangeRequest: strings.TrimSpace(changeRequest),
			Plan:          &currentPlan,
			Metadata:      requestMeta,
		}

		// Run the agent
		resp, _, err := runtime.Run(ctx, req)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		// Check if agent is asking questions
		if len(resp.Questions) > 0 {
			return PlanGenerateInMemoryResult{
				Success:   false,
				Questions: resp.Questions,
			}
		}

		// Convert response to plan
		resultPlan, err := responseToPlan(currentPlan, resp)
		if err != nil {
			return PlanGenerateInMemoryResult{Success: false, Err: err}
		}

		return PlanGenerateInMemoryResult{
			Success: true,
			Plan:    &resultPlan,
		}
	}
}

// SavePlanCmd saves the plan to disk
func SavePlanCmd(g plan.WorkGraph) tea.Cmd {
	return func() tea.Msg {
		path := planPath()
		if err := plan.SaveAtomic(path, g); err != nil {
			return PlanActionComplete{
				Action:  "save plan",
				Success: false,
				Err:     err,
			}
		}
		return PlanActionComplete{
			Action:  "save plan",
			Success: true,
			Output:  fmt.Sprintf("Saved plan: %s", path),
		}
	}
}
