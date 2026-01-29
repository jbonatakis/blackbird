package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/execution"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type ActionMode int

const (
	ActionModeNone ActionMode = iota
	ActionModeSetStatus
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

type Model struct {
	plan             plan.WorkGraph
	selectedID       string
	pendingStatusID  string
	actionMode       ActionMode
	activePane       ActivePane
	tabMode          TabMode
	windowWidth      int
	windowHeight     int
	actionInProgress bool
	actionName       string
	spinnerIndex     int
	runData          map[string]execution.RunRecord
	timerActive      bool
	expandedItems    map[string]bool
	filterMode       FilterMode
	detailOffset     int
	actionOutput     *ActionOutput
}

func NewModel(g plan.WorkGraph) Model {
	m := Model{
		plan:          g,
		actionMode:    ActionModeNone,
		activePane:    PaneTree,
		tabMode:       TabDetails,
		runData:       map[string]execution.RunRecord{},
		expandedItems: map[string]bool{},
		filterMode:    FilterModeAll,
	}
	for id := range g.Items {
		m.selectedID = id
		break
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.LoadRunData(), RunDataRefreshCmd()}
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
		return m, nil
	case spinnerTickMsg:
		if !m.actionInProgress {
			return m, nil
		}
		m.spinnerIndex = (m.spinnerIndex + 1) % len(spinnerFrames)
		return m, spinnerTickCmd()
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
		}
		return m, m.LoadRunData()
	case ExecuteActionComplete:
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
		}
		if typed.Action == "execute" || typed.Action == "resume" || typed.Action == "set-status" {
			return m, m.LoadRunData()
		}
		return m, nil
	case RunDataLoaded:
		if typed.Err != nil {
			return m, nil
		}
		m.runData = typed.Data
		if hasActiveRuns(m.runData) {
			if !m.timerActive {
				m.timerActive = true
				return m, TickCmd()
			}
			return m, nil
		}
		m.timerActive = false
		return m, nil
	case runDataRefreshMsg:
		return m, tea.Batch(m.LoadRunData(), RunDataRefreshCmd())
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
			case "ctrl+c", "q":
				return m, tea.Quit
			default:
				return HandleSetStatusKey(m, typed.String())
			}
		}
		// Clear action output on any key press (after reading)
		if m.actionOutput != nil && !m.actionInProgress {
			m.actionOutput = nil
		}
		switch typed.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			if m.activePane == PaneTree {
				m.activePane = PaneDetail
			} else {
				m.activePane = PaneTree
			}
			return m, nil
		case "t":
			if m.actionMode != ActionModeNone || m.actionInProgress {
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
				m.detailOffset = 0
			}
			return m, nil
		case "down", "j":
			if m.activePane != PaneTree {
				return m, nil
			}
			next := m.nextVisibleItem()
			if next != "" && next != m.selectedID {
				m.selectedID = next
				m.detailOffset = 0
			}
			return m, nil
		case "home":
			if m.activePane != PaneTree {
				return m, nil
			}
			visible := m.visibleItemIDs()
			if len(visible) > 0 && m.selectedID != visible[0] {
				m.selectedID = visible[0]
				m.detailOffset = 0
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
					m.detailOffset = 0
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
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			m.actionInProgress = true
			m.actionName = "Generating plan..."
			return m, tea.Batch(PlanGenerateCmd(), spinnerTickCmd())
		case "r":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			m.actionInProgress = true
			m.actionName = "Refining plan..."
			return m, tea.Batch(PlanRefineCmd(), spinnerTickCmd())
		case "e":
			if m.actionMode != ActionModeNone || m.actionInProgress {
				return m, nil
			}
			if len(execution.ReadyTasks(m.plan)) == 0 {
				return m, nil
			}
			m.actionInProgress = true
			m.actionName = "Executing..."
			return m, tea.Batch(ExecuteCmd(), spinnerTickCmd())
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
			m.actionInProgress = true
			m.actionName = "Resuming..."
			return m, tea.Batch(ResumeCmd(m.selectedID), spinnerTickCmd())
		}
	}
	return m, nil
}

func (m Model) View() string {
	availableHeight := m.windowHeight
	if availableHeight > 0 {
		availableHeight--
	}
	if availableHeight < 0 {
		availableHeight = 0
	}

	if availableHeight == 0 {
		return RenderBottomBar(m)
	}

	content := m.renderMainView(availableHeight)

	// Overlay action output if present
	if m.actionOutput != nil && !m.actionInProgress {
		outputView := RenderActionOutput(m.actionOutput, m.windowWidth)
		// Simple overlay at the top
		content = outputView + "\n" + content
	}

	// Overlay set-status modal if active
	if m.actionMode == ActionModeSetStatus {
		modal := RenderSetStatusModal(m)
		if modal != "" {
			content = modal
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
	it, ok := m.plan.Items[id]
	if !ok {
		return false
	}
	return len(it.ChildIDs) > 0
}

func (m Model) visibleItemIDs() []string {
	roots := rootIDs(m.plan)
	visited := map[string]bool{}
	out := make([]string, 0)
	for _, id := range roots {
		items, _ := m.visibleBranch(id, visited)
		out = append(out, items...)
	}
	return out
}

func (m Model) visibleBranch(id string, visited map[string]bool) ([]string, bool) {
	if visited[id] {
		return nil, false
	}
	visited[id] = true
	it, ok := m.plan.Items[id]
	if !ok {
		return nil, false
	}
	children := append([]string{}, it.ChildIDs...)
	sort.Strings(children)

	depsOK := len(plan.UnmetDeps(m.plan, it)) == 0
	label := plan.ReadinessLabel(it.Status, depsOK, it.Status == plan.StatusBlocked)
	matchesSelf := filterMatch(m.filterMode, label)

	isExpanded := isExpanded(m, it.ID)
	var childLines []string
	var childMatched bool
	for _, childID := range children {
		lines, matched := m.visibleBranch(childID, visited)
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

func (m Model) detailPageSize() int {
	height := m.windowHeight
	if height > 0 {
		height--
	}
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
	minLeft := 24
	minRight := 30
	gap := 1
	left := total / 3
	if left < minLeft {
		left = minLeft
	}
	if total-left-gap < minRight {
		left = total - minRight - gap
		if left < minLeft {
			left = total / 2
		}
	}
	right := total - left - gap
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
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(titleColor)
		titleLine := titleStyle.Render(" " + title + " ")
		// Insert title in the top border
		lines := strings.Split(rendered, "\n")
		if len(lines) > 0 {
			// Replace part of the top border with the title
			topLine := lines[0]
			if len(topLine) > len(title)+4 {
				runes := []rune(topLine)
				titleRunes := []rune(titleLine)
				// Insert title starting at position 2
				for i := 0; i < len(titleRunes) && i+2 < len(runes); i++ {
					runes[i+2] = titleRunes[i]
				}
				lines[0] = string(runes)
				rendered = strings.Join(lines, "\n")
			}
		}
	}

	return rendered
}
