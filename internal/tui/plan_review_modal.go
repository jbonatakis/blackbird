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
	"github.com/jbonatakis/blackbird/internal/planquality"
)

// PlanReviewMode represents the current mode in the review flow
type PlanReviewMode int

const (
	ReviewModeChooseAction PlanReviewMode = iota
	ReviewModeRevisionPrompt
)

const planReviewKeyFindingsLimit = 3

const (
	planReviewActionAccept = iota
	planReviewActionRevise
	planReviewActionReject
)

// PlanReviewQualitySummary tracks quality-gate outcomes shown during review.
type PlanReviewQualitySummary struct {
	InitialBlockingCount int
	InitialWarningCount  int
	BlockingCount        int
	WarningCount         int
	KeyFindings          []string
	AutoRefinePassesRun  int
}

// PlanReviewForm represents the state of the plan review modal
type PlanReviewForm struct {
	mode             PlanReviewMode
	plan             plan.WorkGraph
	qualitySummary   PlanReviewQualitySummary
	selectedAction   int // 0=Accept/Accept anyway, 1=Revise, 2=Reject
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

	form := PlanReviewForm{
		mode:             ReviewModeChooseAction,
		plan:             generatedPlan,
		qualitySummary:   defaultPlanReviewQualitySummary(generatedPlan),
		selectedAction:   planReviewActionAccept,
		revisionTextarea: revisionTA,
		width:            80,
		height:           30,
		revisionCount:    revisionCount,
	}
	form.resetDefaultSelection()
	return form
}

// SetSize updates the form dimensions
func (f *PlanReviewForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	// Adjust textarea width (full screen uses most of width)
	fieldWidth := width - 10
	if fieldWidth > 120 {
		fieldWidth = 120
	}
	if fieldWidth < 40 {
		fieldWidth = 40
	}
	f.revisionTextarea.SetWidth(fieldWidth)
	// Use more lines for revision textarea when full-screen
	taHeight := 4
	if height > 20 {
		taHeight = height / 4
		if taHeight > 12 {
			taHeight = 12
		}
	}
	f.revisionTextarea.SetHeight(taHeight)
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
				if f.selectedAction < planReviewActionReject {
					f.selectedAction++
				}
				return f, nil
			case "1":
				f.selectedAction = planReviewActionAccept
				return f, nil
			case "2":
				// Only allow revise if not exceeded limit
				if f.revisionCount < agent.MaxPlanGenerateRevisions {
					f.selectedAction = planReviewActionRevise
				}
				return f, nil
			case "3":
				f.selectedAction = planReviewActionReject
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

// GetAction returns the selected action index (0=Accept/Accept anyway, 1=Revise, 2=Reject).
func (f PlanReviewForm) GetAction() int {
	return f.selectedAction
}

// GetRevisionRequest returns the revision request text
func (f PlanReviewForm) GetRevisionRequest() string {
	return strings.TrimSpace(f.revisionTextarea.Value())
}

// CanRevise returns true if another revision is allowed
func (f PlanReviewForm) CanRevise() bool {
	return f.revisionCount < agent.MaxPlanGenerateRevisions
}

func (f PlanReviewForm) HasBlockingFindings() bool {
	return f.qualitySummary.BlockingCount > 0
}

func (f PlanReviewForm) acceptLabel() string {
	if f.HasBlockingFindings() {
		return "1. Accept anyway"
	}
	return "1. Accept"
}

func (f *PlanReviewForm) resetDefaultSelection() {
	if f.HasBlockingFindings() {
		if f.CanRevise() {
			f.selectedAction = planReviewActionRevise
			return
		}
		f.selectedAction = planReviewActionReject
		return
	}
	f.selectedAction = planReviewActionAccept
}

func (f *PlanReviewForm) SetQualitySummary(summary PlanReviewQualitySummary) {
	summary.KeyFindings = append([]string(nil), summary.KeyFindings...)
	if summary.InitialBlockingCount < 0 {
		summary.InitialBlockingCount = 0
	}
	if summary.InitialWarningCount < 0 {
		summary.InitialWarningCount = 0
	}
	if summary.BlockingCount < 0 {
		summary.BlockingCount = 0
	}
	if summary.WarningCount < 0 {
		summary.WarningCount = 0
	}
	if summary.AutoRefinePassesRun < 0 {
		summary.AutoRefinePassesRun = 0
	}
	f.qualitySummary = summary
	f.resetDefaultSelection()
}

func defaultPlanReviewQualitySummary(generatedPlan plan.WorkGraph) PlanReviewQualitySummary {
	findings := planquality.Lint(generatedPlan)
	return planReviewQualitySummaryFromFindings(findings, findings, 0)
}

func buildPlanReviewQualitySummary(result planquality.QualityGateResult) PlanReviewQualitySummary {
	return planReviewQualitySummaryFromFindings(result.InitialFindings, result.FinalFindings, result.AutoRefinePassesRun)
}

func planReviewQualitySummaryFromFindings(initialFindings []planquality.PlanQualityFinding, finalFindings []planquality.PlanQualityFinding, autoRefinePassesRun int) PlanReviewQualitySummary {
	initialSummary := planquality.Summarize(initialFindings)
	finalSummary := planquality.Summarize(finalFindings)
	summary := PlanReviewQualitySummary{
		InitialBlockingCount: initialSummary.Blocking,
		InitialWarningCount:  initialSummary.Warning,
		BlockingCount:        finalSummary.Blocking,
		WarningCount:         finalSummary.Warning,
		KeyFindings:          buildPlanReviewKeyFindings(finalFindings, planReviewKeyFindingsLimit),
		AutoRefinePassesRun:  autoRefinePassesRun,
	}
	return summary
}

func buildPlanReviewKeyFindings(findings []planquality.PlanQualityFinding, limit int) []string {
	if limit <= 0 || len(findings) == 0 {
		return nil
	}
	ordered := planquality.Summarize(findings)
	out := make([]string, 0, limit)
	for _, task := range ordered.Tasks {
		for _, field := range task.Fields {
			for _, finding := range field.Findings {
				label := fmt.Sprintf("%s.%s [%s] %s", task.TaskID, field.Field, finding.Severity, strings.TrimSpace(finding.Message))
				out = append(out, strings.TrimSpace(label))
				if len(out) >= limit {
					return out
				}
			}
		}
	}
	return out
}

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
			case planReviewActionAccept:
				if m.planReviewForm.HasBlockingFindings() {
					return acceptPlanAnyway(m)
				}
				return acceptPlan(m)
			case planReviewActionRevise:
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
			case planReviewActionReject:
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

func acceptPlanAnyway(m Model) (Model, tea.Cmd) {
	if m.planReviewForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	m.plan = m.planReviewForm.plan
	m.ensureSelectionVisible()
	m.actionMode = ActionModeNone
	m.planReviewForm = nil
	m.pendingPlanRequest = PendingPlanRequest{}
	m.actionOutput = &ActionOutput{
		Message: "WARNING: blocking findings were overridden; saving plan anyway",
		IsError: false,
	}

	return m, SavePlanCmdWithAction(
		m.plan,
		"save plan override",
		"WARNING: blocking findings were overridden; saved plan anyway",
	)
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

	modalWidth := m.windowWidth - 4
	if modalWidth < 50 {
		modalWidth = 50
	}
	modalHeight := m.windowHeight - 3
	if modalHeight < 10 {
		modalHeight = 10
	}
	contentWidth := modalWidth - 8
	if contentWidth < 24 {
		contentWidth = 24
	}
	topLevelPreviewLimit := 5
	if contentWidth < 40 {
		topLevelPreviewLimit = 3
	}

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

		// Quality summary
		lines = append(lines, labelStyle.Render("Quality summary:"))
		lines = append(lines, textStyle.Render(fmt.Sprintf("  Initial: blocking=%d warning=%d", form.qualitySummary.InitialBlockingCount, form.qualitySummary.InitialWarningCount)))
		lines = append(lines, textStyle.Render(fmt.Sprintf("  Final: blocking=%d warning=%d", form.qualitySummary.BlockingCount, form.qualitySummary.WarningCount)))
		if form.qualitySummary.AutoRefinePassesRun > 0 {
			passLabel := "passes"
			if form.qualitySummary.AutoRefinePassesRun == 1 {
				passLabel = "pass"
			}
			outcome := "blocking findings remain"
			if form.qualitySummary.BlockingCount == 0 {
				outcome = "no blocking findings remain"
			}
			lines = append(lines, textStyle.Render(fmt.Sprintf("  Auto-refine: %d %s run, %s", form.qualitySummary.AutoRefinePassesRun, passLabel, outcome)))
		}
		if form.HasBlockingFindings() {
			lines = append(lines, textStyle.Render("  Blocking findings remain: explicit override required to accept"))
		}
		if len(form.qualitySummary.KeyFindings) == 0 {
			lines = append(lines, textStyle.Render("  Key findings: none"))
		} else {
			lines = append(lines, labelStyle.Render("  Key findings:"))
			maxFindingWidth := contentWidth - 6
			if maxFindingWidth < 10 {
				maxFindingWidth = 10
			}
			for _, finding := range form.qualitySummary.KeyFindings {
				lines = append(lines, textStyle.Render("    - "+truncateField(finding, maxFindingWidth)))
			}
		}
		lines = append(lines, "")

		// Top-level features preview
		roots := plan.BuildTaskTree(form.plan).Roots
		if len(roots) > 0 {
			lines = append(lines, labelStyle.Render("Top-level features:"))
			for i, rootID := range roots {
				if i >= topLevelPreviewLimit {
					lines = append(lines, labelStyle.Render(fmt.Sprintf("  ... and %d more", len(roots)-topLevelPreviewLimit)))
					break
				}
				item, ok := form.plan.Items[rootID]
				if ok {
					lines = append(lines, textStyle.Render(fmt.Sprintf("  • %s", truncateField(item.Title, contentWidth-6))))
				}
			}
			lines = append(lines, "")
		}

		// Action choices
		lines = append(lines, labelStyle.Render("Choose an action:"))
		lines = append(lines, "")

		// Accept
		acceptLabel := form.acceptLabel()
		if form.selectedAction == planReviewActionAccept {
			lines = append(lines, selectedStyle.Render(acceptLabel))
		} else {
			lines = append(lines, unselectedStyle.Render(acceptLabel))
		}

		// Revise
		reviseLabel := "2. Revise"
		if !form.CanRevise() {
			lines = append(lines, disabledStyle.Render(reviseLabel+" (limit reached)"))
		} else if form.selectedAction == planReviewActionRevise {
			lines = append(lines, selectedStyle.Render(reviseLabel))
		} else {
			lines = append(lines, unselectedStyle.Render(reviseLabel))
		}

		// Reject
		rejectLabel := "3. Reject"
		if form.selectedAction == planReviewActionReject {
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

		helpText := helpStyle.Render("[ctrl+s or ctrl+enter]submit • Enter: new line • [esc]back")
		lines = append(lines, helpText)
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight)

	modal := modalStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
	return modal
}

// SavePlanCmd saves the plan to disk
func SavePlanCmd(g plan.WorkGraph) tea.Cmd {
	return SavePlanCmdWithAction(g, "save plan", "")
}

func SavePlanCmdWithAction(g plan.WorkGraph, action string, message string) tea.Cmd {
	return func() tea.Msg {
		path := plan.PlanPath()
		if err := plan.SaveAtomic(path, g); err != nil {
			return PlanActionComplete{
				Action:  action,
				Success: false,
				Err:     err,
			}
		}
		output := fmt.Sprintf("Saved plan: %s", path)
		if strings.TrimSpace(message) != "" {
			output = fmt.Sprintf("%s: %s", message, path)
		}
		return PlanActionComplete{
			Action:  action,
			Success: true,
			Output:  output,
		}
	}
}
