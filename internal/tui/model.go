package tui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ActionMode int

const (
	ActionModeNone ActionMode = iota
	ActionModeSetStatus
	ActionModeGeneratePlan
	ActionModeConfirmOverwrite
	ActionModeAgentQuestion
	ActionModePlanReview
	ActionModePlanRefine
	ActionModeSelectAgent
	ActionModeReviewCheckpoint
)

type ActivePane int

const (
	PaneTree ActivePane = iota
	PaneDetail
)

type TabMode int

const (
	TabDetails TabMode = iota
	TabExecution
)

type ViewMode int

const (
	ViewModeHome ViewMode = iota
	ViewModeMain
)

type PendingPlanRequestKind int

const (
	PendingPlanGenerate PendingPlanRequestKind = iota
	PendingPlanRefine
)

// PendingPlanRequest tracks the original plan generation request for question rounds
type PendingPlanRequest struct {
	kind          PendingPlanRequestKind
	description   string
	constraints   []string
	granularity   string
	changeRequest string
	basePlan      plan.WorkGraph
	questionRound int
}

type Model struct {
	plan                    plan.WorkGraph
	selectedID              string
	pendingStatusID         string
	actionMode              ActionMode
	activePane              ActivePane
	tabMode                 TabMode
	viewMode                ViewMode
	planExists              bool
	planValidationErr       string
	windowWidth             int
	windowHeight            int
	actionInProgress        bool
	actionName              string
	actionCancel            context.CancelFunc
	spinnerIndex            int
	runData                 map[string]execution.RunRecord
	timerActive             bool
	liveStdout              string
	liveStderr              string
	liveOutputChan          chan liveOutputMsg
	expandedItems           map[string]bool
	filterMode              FilterMode
	detailOffset            int
	actionOutput            *ActionOutput
	planGenerateForm        *PlanGenerateForm
	planRefineForm          *PlanRefineForm
	agentQuestionForm       *AgentQuestionForm
	planReviewForm          *PlanReviewForm
	reviewCheckpointForm    *ReviewCheckpointForm
	pendingPlanRequest      PendingPlanRequest
	pendingResumeTask       string
	agentSelection          agent.AgentSelection
	agentSelectionErr       string
	agentSelectionHighlight int // index into agent.AgentRegistry when modal is open
	projectRoot             string
	config                  config.ResolvedConfig
}

func NewModel(g plan.WorkGraph) Model {
	m := Model{
		plan:          g,
		actionMode:    ActionModeNone,
		activePane:    PaneTree,
		tabMode:       TabDetails,
		viewMode:      ViewModeHome,
		planExists:    true,
		runData:       map[string]execution.RunRecord{},
		expandedItems: map[string]bool{},
		filterMode:    FilterModeAll,
		agentSelection: agent.AgentSelection{
			Agent:         agent.DefaultAgent(),
			ConfigPresent: false,
		},
		config: config.DefaultResolvedConfig(),
	}
	for id := range g.Items {
		m.selectedID = id
		break
	}
	return m
}

func (m Model) hasPlan() bool {
	return m.planExists
}

func (m Model) canExecute() bool {
	return m.planExists && len(execution.ReadyTasks(m.plan)) > 0
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.LoadRunData(), m.RunDataRefreshCmd(), m.LoadPlanData(), m.PlanDataRefreshCmd(), m.LoadAgentSelection()}
	if hasActiveRuns(m.runData) {
		cmds = append(cmds, StartTimerCmd())
	}
	return tea.Batch(cmds...)
}

type spinnerTickMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = typed.Width
		m.windowHeight = typed.Height

		// Update active modal forms with new dimensions
		if m.planGenerateForm != nil {
			m.planGenerateForm.SetSize(typed.Width, typed.Height)
		}
		if m.planRefineForm != nil {
			m.planRefineForm.SetSize(typed.Width, typed.Height)
		}
		if m.agentQuestionForm != nil {
			m.agentQuestionForm.SetSize(typed.Width, typed.Height)
		}
		if m.planReviewForm != nil {
			m.planReviewForm.SetSize(typed.Width, typed.Height)
		}
		if m.reviewCheckpointForm != nil {
			m.reviewCheckpointForm.SetSize(typed.Width, typed.Height)
		}

		return m, nil
	case spinnerTickMsg:
		if !m.actionInProgress {
			return m, nil
		}
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinnerFrames)
		return m, spinnerTickCmd()
	case PlanGenerateInMemoryResult:
		m.actionInProgress = false
		m.actionName = ""
		if typed.Err != nil {
			message := "Plan generation failed"
			if m.pendingPlanRequest.kind == PendingPlanRefine {
				message = "Plan refine failed"
			}
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("%s: %v", message, typed.Err),
				IsError: true,
			}
		} else if len(typed.Questions) > 0 {
			// Agent asked questions - show question modal
			form := NewAgentQuestionForm(typed.Questions)
			form.SetSize(m.windowWidth, m.windowHeight)
			m.agentQuestionForm = &form
			m.actionMode = ActionModeAgentQuestion
			return m, nil
		} else if typed.Plan != nil {
			if m.pendingPlanRequest.kind == PendingPlanRefine {
				m.plan = *typed.Plan
				m.ensureSelectionVisible()
				m.pendingPlanRequest = PendingPlanRequest{}
				return m, SavePlanCmdWithAction(m.plan, "plan refine", "Refined plan saved")
			}
			// Success - show plan review modal
			form := NewPlanReviewForm(*typed.Plan, m.pendingPlanRequest.questionRound)
			form.SetSize(m.windowWidth, m.windowHeight)
			m.planReviewForm = &form
			m.actionMode = ActionModePlanReview
			return m, nil
		}
		return m, m.LoadRunData()
	case PlanActionComplete:
		m.actionInProgress = false
		m.actionName = ""
		if typed.Err != nil {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Action failed: %v\n\n%s", typed.Err, typed.Output),
				IsError: true,
			}
		} else {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Action completed successfully\n\n%s", typed.Output),
				IsError: false,
			}
			if typed.Action == "save plan" {
				m.planExists = true
				m.viewMode = ViewModeMain
			}
		}
		return m, tea.Batch(m.LoadRunData(), m.LoadPlanData())
	case ExecuteActionComplete:
		m.actionInProgress = false
		m.actionName = ""
		m.actionCancel = nil
		if typed.Action == "execute" || typed.Action == "resume" {
			m.clearLiveOutput()
		}
		decisionRequired := typed.Result != nil && typed.Result.Reason == execution.ExecuteReasonDecisionRequired
		if typed.Err != nil {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Action failed: %v\n\n%s", typed.Err, typed.Output),
				IsError: true,
			}
		} else if decisionRequired {
			m.actionOutput = nil
			if typed.Result != nil && typed.Result.Run != nil {
				m = openReviewCheckpointModal(m, *typed.Result.Run)
			}
		} else {
			if typed.Action == "execute" || typed.Action == "resume" {
				message := typed.Output
				if message == "" {
					message = "Action completed successfully"
				}
				m.actionOutput = &ActionOutput{
					Message: message,
					IsError: false,
				}
			} else {
				m.actionOutput = &ActionOutput{
					Message: fmt.Sprintf("Action completed successfully\n\n%s", typed.Output),
					IsError: false,
				}
			}
		}
		if typed.Action == "execute" || typed.Action == "resume" || typed.Action == "set-status" {
			return m, tea.Batch(m.LoadRunData(), m.LoadPlanData())
		}
		return m, nil
	case DecisionActionComplete:
		m.actionInProgress = false
		m.actionName = ""
		m.actionCancel = nil
		if typed.Action == execution.DecisionStateChangesRequested {
			m.clearLiveOutput()
		}
		if typed.Err != nil {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Decision failed: %v", typed.Err),
				IsError: true,
			}
			return m, tea.Batch(m.LoadRunData(), m.LoadPlanData())
		}

		if typed.Result.Next != nil {
			next := typed.Result.Next
			if next.Err != nil {
				m.actionOutput = &ActionOutput{
					Message: fmt.Sprintf("Action failed: %v", next.Err),
					IsError: true,
				}
			}
			switch next.Reason {
			case execution.ExecuteReasonDecisionRequired:
				if next.Run != nil {
					m = openReviewCheckpointModal(m, *next.Run)
				}
			case execution.ExecuteReasonWaitingUser:
				m.actionOutput = &ActionOutput{
					Message: summarizeExecuteResult(*next),
					IsError: false,
				}
			}
		}

		if typed.Action == execution.DecisionStateApprovedContinue {
			m.actionInProgress = true
			m.actionName = "Executing..."
			ctx, cancel := context.WithCancel(context.Background())
			m.actionCancel = cancel
			streamCh, stdout, stderr := m.startLiveOutput()
			return m, tea.Batch(
				ExecuteCmdWithContextAndStream(ctx, stdout, stderr, streamCh, m.config.Execution.StopAfterEachTask),
				listenLiveOutputCmd(streamCh),
				spinnerTickCmd(),
				m.LoadRunData(),
				m.LoadPlanData(),
			)
		}

		if typed.Action == execution.DecisionStateApprovedQuit {
			m.actionOutput = &ActionOutput{
				Message: "Decision recorded: approved and quit",
				IsError: false,
			}
		}
		if typed.Action == execution.DecisionStateRejected {
			m.actionOutput = &ActionOutput{
				Message: "Decision recorded: changes rejected",
				IsError: false,
			}
		}
		if typed.Action == execution.DecisionStateChangesRequested && typed.Result.Next == nil {
			m.actionOutput = &ActionOutput{
				Message: "Change request submitted",
				IsError: false,
			}
		}
		return m, tea.Batch(m.LoadRunData(), m.LoadPlanData())
	case PlanDataLoaded:
		m.plan = typed.Plan
		m.planExists = typed.PlanExists
		m.planValidationErr = typed.ValidationErr
		if typed.Err != nil {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Plan load failed: %v", typed.Err),
				IsError: true,
			}
		}
		m.ensureSelectionVisible()
		return m, nil
	case AgentSelectionLoaded:
		m.agentSelection = typed.Selection
		if typed.Err != nil {
			m.agentSelectionErr = typed.Err.Error()
		} else {
			m.agentSelectionErr = ""
		}
		return m, nil
	case AgentSelectionSaved:
		if typed.Err != nil {
			m.actionOutput = &ActionOutput{
				Message: fmt.Sprintf("Agent selection failed: %v", typed.Err),
				IsError: true,
			}
			return m, nil
		}
		m.agentSelection = typed.Selection
		m.agentSelectionErr = ""
		m.actionOutput = &ActionOutput{
			Message: fmt.Sprintf("Agent set to %s", typed.Selection.Agent.Label),
			IsError: false,
		}
		return m, nil
	case RunDataLoaded:
		if typed.Err != nil {
			return m, nil
		}
		m.runData = typed.Data
		if pending := pendingDecisionRunFromData(m.runData); pending != nil {
			if m.actionMode == ActionModeNone && m.reviewCheckpointForm == nil && !m.actionInProgress {
				m = openReviewCheckpointModal(m, *pending)
			}
		} else if m.actionMode == ActionModeReviewCheckpoint && m.reviewCheckpointForm != nil {
			m.actionMode = ActionModeNone
			m.reviewCheckpointForm = nil
		}
		if hasActiveRuns(m.runData) {
			if !m.timerActive {
				m.timerActive = true
				return m, TickCmd()
			}
			return m, nil
		}
		m.timerActive = false
		return m, nil
	case liveOutputMsg:
		switch typed.stream {
		case "stdout":
			m.liveStdout += typed.data
		case "stderr":
			m.liveStderr += typed.data
		}
		if m.liveOutputChan != nil {
			return m, listenLiveOutputCmd(m.liveOutputChan)
		}
		return m, nil
	case liveOutputDoneMsg:
		m.liveOutputChan = nil
		return m, nil
	case runDataRefreshMsg:
		return m, tea.Batch(m.LoadRunData(), m.RunDataRefreshCmd())
	case planDataRefreshMsg:
		return m, tea.Batch(m.LoadPlanData(), m.PlanDataRefreshCmd())
	case timerStartMsg:
		if hasActiveRuns(m.runData) && !m.timerActive {
			m.timerActive = true
			return m, TickCmd()
		}
		return m, nil
	case timerTickMsg:
		m.timerActive = false
		if hasActiveRuns(m.runData) {
			m.timerActive = true
			return m, TickCmd()
		}
		return m, nil
	case tea.KeyMsg:
		if m.actionMode == ActionModeSetStatus {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			default:
				return HandleSetStatusKey(m, typed.String())
			}
		}
		if m.actionMode == ActionModeConfirmOverwrite {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			default:
				return HandleConfirmOverwriteKey(m, typed.String())
			}
		}
		if m.actionMode == ActionModeGeneratePlan {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			case "esc":
				if m.planGenerateForm != nil && m.planGenerateForm.filePicker.Open {
					updatedForm, cmd := m.planGenerateForm.Update(typed)
					m.planGenerateForm = &updatedForm
					return m, cmd
				}
				return HandlePlanGenerateKey(m, typed)
			default:
				return HandlePlanGenerateKey(m, typed)
			}
		}
		if m.actionMode == ActionModePlanRefine {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			case "esc":
				if m.planRefineForm != nil && m.planRefineForm.filePicker.Open {
					updatedForm, cmd := m.planRefineForm.Update(typed)
					m.planRefineForm = &updatedForm
					return m, cmd
				}
				return HandlePlanRefineKey(m, typed)
			default:
				return HandlePlanRefineKey(m, typed)
			}
		}
		if m.actionMode == ActionModeAgentQuestion {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			case "esc":
				// Cancel modal
				m.actionMode = ActionModeNone
				m.agentQuestionForm = nil
				m.pendingPlanRequest = PendingPlanRequest{}
				m.pendingResumeTask = ""
				return m, nil
			default:
				return HandleAgentQuestionKey(m, typed)
			}
		}
		if m.actionMode == ActionModePlanReview {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			case "esc":
				// Cancel modal - discard plan
				m.actionMode = ActionModeNone
				m.planReviewForm = nil
				m.pendingPlanRequest = PendingPlanRequest{}
				m.actionOutput = &ActionOutput{
					Message: "Plan review cancelled",
					IsError: false,
				}
				return m, nil
			default:
				return HandlePlanReviewKey(m, typed)
			}
		}
		if m.actionMode == ActionModeReviewCheckpoint {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			default:
				return HandleReviewCheckpointKey(m, typed)
			}
		}
		if m.actionMode == ActionModeSelectAgent {
			switch typed.String() {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			default:
				return HandleAgentSelectionKey(m, typed.String())
			}
		}
		// Clear action output on any key press (after reading)
		if m.actionOutput != nil && !m.actionInProgress {
			m.actionOutput = nil
		}
		key := typed.String()
		switch key {
		case "h":
			if m.viewMode == ViewModeHome {
				if m.planExists {
					m.viewMode = ViewModeMain
				}
			} else {
				m.viewMode = ViewModeHome
			}
			return m, nil
		}
		if m.viewMode == ViewModeHome && m.actionMode == ActionModeNone {
			switch key {
			case "ctrl+c":
				m = cancelRunningAction(m)
				return m, tea.Quit
			case "g":
				return m.startPlanGenerate()
			case "v":
				if m.planExists {
					m.viewMode = ViewModeMain
				}
				return m, nil
			case "r":
				if m.actionMode != ActionModeNone || m.actionInProgress || !m.planExists {
					return m, nil
				}
				return m.startPlanRefine()
			case "e":
				if m.actionMode != ActionModeNone || m.actionInProgress || !m.canExecute() {
					return m, nil
				}
				m.actionInProgress = true
				m.actionName = "Executing..."
				ctx, cancel := context.WithCancel(context.Background())
				m.actionCancel = cancel
				streamCh, stdout, stderr := m.startLiveOutput()
				return m, tea.Batch(
					ExecuteCmdWithContextAndStream(ctx, stdout, stderr, streamCh, m.config.Execution.StopAfterEachTask),
					listenLiveOutputCmd(streamCh),
					spinnerTickCmd(),
				)
			case "c":
				if m.actionMode != ActionModeNone || m.actionInProgress {
					return m, nil
				}
				if agentIsFromEnv() {
					return m, nil // env overrides; change agent disabled
				}
				m.actionMode = ActionModeSelectAgent
				m.agentSelectionHighlight = agentSelectionHighlightIndex(m)
				return m, nil
			default:
				return m, nil
			}
		}
		switch key {
		case "ctrl+c":
			m = cancelRunningAction(m)
			return m, tea.Quit
		case "tab":
			if m.activePane == PaneTree {
				m.activePane = PaneDetail
			} else {
				m.activePane = PaneTree
			}
			return m, nil
		case "t":
			if m.actionMode != ActionModeNone {
				return m, nil
			}
			if m.actionInProgress && m.actionName != "Executing..." && m.actionName != "Resuming..." {
				return m, nil
			}
			if m.tabMode == TabDetails {
				m.tabMode = TabExecution
			} else {
				m.tabMode = TabDetails
			}
			m.detailOffset = 0
			return m, nil
		case "f":
			m.filterMode = nextFilterMode(m.filterMode)
			m.ensureSelectionVisible()
			m.detailOffset = 0
			return m, nil
		case "up", "k":
			if m.activePane != PaneTree {
				return m, nil
			}
			next := m.prevVisibleItem()
			if next != "" && next != m.selectedID {
				m.selectedID = next
			}
			return m, nil
		case "down", "j":
			if m.activePane != PaneTree {
				return m, nil
			}
			next := m.nextVisibleItem()
			if next != "" && next != m.selectedID {
				m.selectedID = next
			}
			return m, nil
		case "home":
			if m.activePane != PaneTree {
				return m, nil
			}
			visible := m.visibleItemIDs()
			if len(visible) > 0 && m.selectedID != visible[0] {
				m.selectedID = visible[0]
			}
			return m, nil
		case "end":
			if m.activePane != PaneTree {
				return m, nil
			}
			visible := m.visibleItemIDs()
			if len(visible) > 0 {
				last := visible[len(visible)-1]
				if m.selectedID != last {
					m.selectedID = last
				}
			}
			return m, nil
		case "enter", " ":
			if m.activePane != PaneTree || m.selectedID == "" {
				return m, nil
			}
			if m.isParent(m.selectedID) {
				m.toggleExpanded(m.selectedID)
				m.ensureSelectionVisible()
			}
			return m, nil
		case "pgup", "pageup":
			if m.activePane != PaneDetail {
				return m, nil
			}
			page := m.detailPageSize()
			if page > 0 {
				m.detailOffset -= page
				if m.detailOffset < 0 {
					m.detailOffset = 0
				}
			}
			return m, nil
		case "pgdown", "pagedown":
			if m.activePane != PaneDetail {
				return m, nil
			}
			page := m.detailPageSize()
			if page > 0 {
				m.detailOffset += page
			}
			return m, nil
		case "g":
			return m.startPlanGenerate()
		case "r":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			return m.startPlanRefine()
		case "e":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			if !m.canExecute() {
				return m, nil
			}
			m.actionInProgress = true
			m.actionName = "Executing..."
			ctx, cancel := context.WithCancel(context.Background())
			m.actionCancel = cancel
			streamCh, stdout, stderr := m.startLiveOutput()
			return m, tea.Batch(
				ExecuteCmdWithContextAndStream(ctx, stdout, stderr, streamCh, m.config.Execution.StopAfterEachTask),
				listenLiveOutputCmd(streamCh),
				spinnerTickCmd(),
			)
		case "s":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			if m.selectedID == "" {
				return m, nil
			}
			m.actionMode = ActionModeSetStatus
			m.pendingStatusID = m.selectedID
			return m, nil
		case "u":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			if !CanResume(m) {
				return m, nil
			}
			questions, err := execution.ParseQuestionsFromLatestWaitingRun(plan.PlanPath(), m.plan, m.selectedID)
			if err != nil {
				m.actionOutput = &ActionOutput{
					Message: fmt.Sprintf("Resume failed: %v", err),
					IsError: true,
				}
				return m, nil
			}
			if len(questions) == 0 {
				m.actionOutput = &ActionOutput{
					Message: fmt.Sprintf("No questions found in waiting run for %s", m.selectedID),
					IsError: false,
				}
				return m, nil
			}

			form := NewAgentQuestionForm(questions)
			form.SetSize(m.windowWidth, m.windowHeight)
			m.agentQuestionForm = &form
			m.actionMode = ActionModeAgentQuestion
			m.pendingResumeTask = m.selectedID
			return m, nil
		}
	}
	return m, nil
}

func (m Model) startPlanGenerate() (Model, tea.Cmd) {
	if m.actionMode != ActionModeNone || m.actionInProgress {
		return m, nil
	}

	// Reset pending request state for new generation
	m.pendingPlanRequest = PendingPlanRequest{kind: PendingPlanGenerate}

	// Check if plan already exists with items
	if len(m.plan.Items) > 0 {
		// Show confirmation modal
		m.actionMode = ActionModeConfirmOverwrite
		return m, nil
	}
	// Plan is empty, proceed directly to generation modal
	form := NewPlanGenerateForm()
	form.SetSize(m.windowWidth, m.windowHeight)
	m.planGenerateForm = &form
	m.actionMode = ActionModeGeneratePlan
	return m, nil
}

func (m Model) startPlanRefine() (Model, tea.Cmd) {
	if m.actionMode != ActionModeNone || m.actionInProgress || !m.planExists {
		return m, nil
	}

	form := NewPlanRefineForm()
	form.SetSize(m.windowWidth, m.windowHeight)
	m.planRefineForm = &form
	m.actionMode = ActionModePlanRefine
	return m, nil
}

func cancelRunningAction(m Model) Model {
	if m.actionCancel != nil {
		m.actionCancel()
		m.actionCancel = nil
	}
	return m
}

func (m Model) View() string {
	banner := ""
	bannerHeight := 0
	if m.viewMode == ViewModeMain {
		banner = RenderActionRequiredBanner(m)
		if banner != "" {
			bannerHeight = lipgloss.Height(banner)
		}
	}

	// Reserve space so total output is strictly less than windowHeight. Each pane
	// gets Height(availableHeight) and lipgloss adds top+bottom border (2 lines),
	// so pane height is availableHeight+2. We add a newline and the bar (2 lines).
	// Total = (availableHeight+2)+2 = availableHeight+4. Use windowHeight-5 so
	// total = windowHeight-1, avoiding exact-height redraw bugs and ensuring the
	// top border is visible.
	availableHeight := m.windowHeight - 5 - bannerHeight
	if availableHeight < 0 {
		availableHeight = 0
	}

	if availableHeight == 0 {
		return RenderBottomBar(m)
	}

	var content string
	if m.viewMode == ViewModeHome {
		content = RenderHomeView(m)
	} else {
		content = m.renderMainView(availableHeight)
		if banner != "" {
			content = banner + "\n" + content
		}
	}

	// Overlay action output if present
	if m.actionOutput != nil && !m.actionInProgress {
		outputView := RenderActionOutput(m.actionOutput, m.windowWidth)
		// Simple overlay at the top
		content = outputView + "\n" + content
	}

	modalModel := m
	if bannerHeight > 0 {
		modalModel.windowHeight = m.windowHeight - bannerHeight
		if modalModel.windowHeight < 0 {
			modalModel.windowHeight = 0
		}
	}

	// Overlay set-status modal if active
	if m.actionMode == ActionModeSetStatus {
		modal := RenderSetStatusModal(modalModel)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay confirm overwrite modal if active
	if m.actionMode == ActionModeConfirmOverwrite {
		modal := RenderConfirmOverwriteModal(modalModel)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay plan generate modal if active
	if m.actionMode == ActionModeGeneratePlan && m.planGenerateForm != nil {
		modal := RenderPlanGenerateModal(modalModel, *m.planGenerateForm)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay plan refine modal if active
	if m.actionMode == ActionModePlanRefine && m.planRefineForm != nil {
		modal := RenderPlanRefineModal(modalModel, *m.planRefineForm)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay agent question modal if active
	if m.actionMode == ActionModeAgentQuestion && m.agentQuestionForm != nil {
		modal := RenderAgentQuestionModal(modalModel, *m.agentQuestionForm)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay plan review modal if active
	if m.actionMode == ActionModePlanReview && m.planReviewForm != nil {
		modal := RenderPlanReviewModal(modalModel, *m.planReviewForm)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay review checkpoint modal if active
	if m.actionMode == ActionModeReviewCheckpoint && m.reviewCheckpointForm != nil {
		modal := RenderReviewCheckpointModal(modalModel, *m.reviewCheckpointForm)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	// Overlay agent selection modal if active
	if m.actionMode == ActionModeSelectAgent {
		modal := RenderAgentSelectionModal(modalModel)
		if modal != "" {
			content = modal
			if banner != "" {
				content = banner + "\n" + content
			}
		}
	}

	if m.windowHeight > 1 {
		return content + "\n" + RenderBottomBar(m)
	}
	return content
}

func spinnerTickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m *Model) startLiveOutput() (chan liveOutputMsg, io.Writer, io.Writer) {
	m.liveStdout = ""
	m.liveStderr = ""
	streamCh := make(chan liveOutputMsg, 256)
	m.liveOutputChan = streamCh
	return streamCh,
		liveOutputWriter{ch: streamCh, stream: "stdout"},
		liveOutputWriter{ch: streamCh, stream: "stderr"}
}

func (m *Model) clearLiveOutput() {
	m.liveStdout = ""
	m.liveStderr = ""
	m.liveOutputChan = nil
}

func (m Model) nextVisibleItem() string {
	visible := m.visibleItemIDs()
	if len(visible) == 0 {
		return ""
	}
	if m.selectedID == "" {
		return visible[0]
	}
	for i, id := range visible {
		if id == m.selectedID {
			if i+1 < len(visible) {
				return visible[i+1]
			}
			return id
		}
	}
	return visible[0]
}

func (m Model) prevVisibleItem() string {
	visible := m.visibleItemIDs()
	if len(visible) == 0 {
		return ""
	}
	if m.selectedID == "" {
		return visible[0]
	}
	for i, id := range visible {
		if id == m.selectedID {
			if i-1 >= 0 {
				return visible[i-1]
			}
			return id
		}
	}
	return visible[0]
}

func (m Model) isParent(id string) bool {
	tree := plan.BuildTaskTree(m.plan)
	return len(tree.Children[id]) > 0
}

func (m Model) visibleItemIDs() []string {
	tree := plan.BuildTaskTree(m.plan)
	visited := map[string]bool{}
	out := make([]string, 0)
	for _, id := range tree.Roots {
		items, _ := m.visibleBranch(tree, id, visited)
		out = append(out, items...)
	}
	return out
}

func (m Model) visibleBranch(tree plan.TaskTree, id string, visited map[string]bool) ([]string, bool) {
	if visited[id] {
		return nil, false
	}
	visited[id] = true
	it, ok := m.plan.Items[id]
	if !ok {
		return nil, false
	}
	children := append([]string{}, tree.Children[it.ID]...)

	depsOK := len(plan.UnmetDeps(m.plan, it)) == 0
	label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
	matchesSelf := filterMatch(m.filterMode, label)

	isExpanded := isExpanded(m, it.ID)
	var childLines []string
	var childMatched bool
	for _, childID := range children {
		lines, matched := m.visibleBranch(tree, childID, visited)
		if matched {
			childMatched = true
		}
		if isExpanded {
			childLines = append(childLines, lines...)
		}
	}

	shouldRender := matchesSelf || childMatched
	if !shouldRender {
		return nil, false
	}
	lines := []string{it.ID}
	if isExpanded {
		lines = append(lines, childLines...)
	}
	return lines, true
}

func (m *Model) toggleExpanded(id string) {
	if m.expandedItems == nil {
		m.expandedItems = map[string]bool{}
	}
	if isExpanded(*m, id) {
		m.expandedItems[id] = false
		return
	}
	m.expandedItems[id] = true
}

func (m *Model) ensureSelectionVisible() {
	visible := m.visibleItemIDs()
	if len(visible) == 0 {
		m.selectedID = ""
		return
	}
	if m.selectedID == "" {
		m.selectedID = visible[0]
		return
	}
	for _, id := range visible {
		if id == m.selectedID {
			return
		}
	}
	m.selectedID = visible[0]
}

func nextFilterMode(current FilterMode) FilterMode {
	switch current {
	case FilterModeAll:
		return FilterModeReady
	case FilterModeReady:
		return FilterModeBlocked
	default:
		return FilterModeAll
	}
}

// detailPageSize returns the number of lines per page in the detail viewport,
// matching the pane content height (availableHeight = windowHeight-5).
func (m Model) detailPageSize() int {
	height := m.windowHeight - 5
	if height < 0 {
		return 0
	}
	return height
}

func (m Model) renderMainView(availableHeight int) string {
	if m.windowWidth <= 0 {
		tree := RenderTreeView(m)
		detail := RenderDetailView(m)
		content := tree + "\n\n" + detail
		return content
	}

	leftWidth, rightWidth := splitPaneWidths(m.windowWidth)
	treeModel := m
	treeModel.windowWidth = leftWidth
	treeModel.windowHeight = availableHeight
	detailModel := m
	detailModel.windowWidth = rightWidth
	detailModel.windowHeight = availableHeight

	treeContent := RenderTreeView(treeModel)

	var detailContent string
	var rightPaneTitle string
	if m.tabMode == TabExecution {
		detailContent = RenderExecutionView(detailModel)
		rightPaneTitle = "Execution"
	} else {
		detailContent = RenderDetailView(detailModel)
		rightPaneTitle = "Details"
	}

	treeBox := renderPane(treeContent, leftWidth, availableHeight, "Plan", m.activePane == PaneTree)
	detailBox := renderPane(detailContent, rightWidth, availableHeight, rightPaneTitle, m.activePane == PaneDetail)
	content := lipgloss.JoinHorizontal(lipgloss.Top, treeBox, detailBox)
	return content
}

func splitPaneWidths(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	// Each pane's rendered width is content width + 2 (left and right border).
	// So we need left + right + 4 = total, i.e. left + right = total - 4.
	minLeft := 24
	minRight := 30
	available := total - 4
	if available < 0 {
		available = 0
	}
	left := available / 3
	if left < minLeft {
		left = minLeft
	}
	if available-left < minRight {
		left = available - minRight
		if left < minLeft {
			left = available / 2
		}
	}
	right := available - left
	if right < 0 {
		right = 0
	}
	return left, right
}

func renderPane(content string, width int, height int, title string, active bool) string {
	borderColor := lipgloss.Color("240")
	titleColor := lipgloss.Color("240")
	if active {
		borderColor = lipgloss.Color("69")
		titleColor = lipgloss.Color("69")
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width).
		Height(height).
		Padding(0, 1)

	rendered := style.Render(content)

	if title != "" {
		// Rebuild the top border line so its display width matches the pane. The
		// original top line contains ANSI escape codes; replacing runes corrupts them.
		// Use the first content line's width as the target so we match lipgloss exactly.
		lines := strings.Split(rendered, "\n")
		if len(lines) > 1 {
			targetWidth := lipgloss.Width(lines[1])
			borderStyle := lipgloss.NewStyle().Foreground(borderColor)
			titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)
			titleWidth := lipgloss.Width(title)
			// Top line: "╭ " (2) + " "+title+" " (4+titleWidth) + "─"*n + "╮" (1)
			nMiddle := targetWidth - 7 - titleWidth
			if nMiddle < 0 {
				nMiddle = 0
			}
			topLine := borderStyle.Render("╭ ") +
				titleStyle.Render(" "+title+" ") +
				borderStyle.Render(strings.Repeat("─", nMiddle)+"╮")
			// Pad with more middle dashes if short (rune-width or rounding)
			if w := lipgloss.Width(topLine); w < targetWidth {
				nMiddle += targetWidth - w
				topLine = borderStyle.Render("╭ ") +
					titleStyle.Render(" "+title+" ") +
					borderStyle.Render(strings.Repeat("─", nMiddle)+"╮")
			}
			lines[0] = topLine
			rendered = strings.Join(lines, "\n")
		}
	}

	return rendered
}

func formatQuestions(questions []agent.Question) string {
	var parts []string
	for i, q := range questions {
		parts = append(parts, fmt.Sprintf("%d. %s", i+1, q.Prompt))
	}
	return strings.Join(parts, "\n")
}
