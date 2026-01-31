package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jbonatakis/blackbird/internal/agent"
)

// AgentQuestionForm represents the state of the agent question modal
type AgentQuestionForm struct {
	questions      []agent.Question
	currentIndex   int
	textInput      textinput.Model
	selectedOption int // For questions with options (0-indexed, -1 means none selected)
	answers        []agent.Answer
	width          int
	height         int
}

// NewAgentQuestionForm creates a new agent question form
func NewAgentQuestionForm(questions []agent.Question) AgentQuestionForm {
	if len(questions) == 0 {
		return AgentQuestionForm{}
	}

	// Initialize text input for free-text questions
	ti := textinput.New()
	ti.Placeholder = "Enter your answer..."
	ti.CharLimit = 500
	ti.Width = 60

	// Determine initial state based on first question
	initialSelection := -1
	if len(questions[0].Options) == 0 {
		// Free-text question: focus text input
		ti.Focus()
	} else {
		// Multiple choice: blur text input and auto-select first option
		ti.Blur()
		initialSelection = 0
	}

	return AgentQuestionForm{
		questions:      questions,
		currentIndex:   0,
		textInput:      ti,
		selectedOption: initialSelection,
		answers:        make([]agent.Answer, 0, len(questions)),
		width:          70,
		height:         25,
	}
}

// SetSize updates the form dimensions
func (f *AgentQuestionForm) SetSize(width, height int) {
	f.width = width
	f.height = height

	// Adjust input width based on modal width
	inputWidth := width - 10
	if inputWidth > 80 {
		inputWidth = 80
	}
	if inputWidth < 40 {
		inputWidth = 40
	}

	f.textInput.Width = inputWidth
}

// CurrentQuestion returns the current question being answered
func (f AgentQuestionForm) CurrentQuestion() agent.Question {
	if f.currentIndex >= 0 && f.currentIndex < len(f.questions) {
		return f.questions[f.currentIndex]
	}
	return agent.Question{}
}

// IsComplete returns true if all questions have been answered
func (f AgentQuestionForm) IsComplete() bool {
	return f.currentIndex >= len(f.questions)
}

// GetAnswers returns all collected answers
func (f AgentQuestionForm) GetAnswers() []agent.Answer {
	return f.answers
}

// Update handles form updates
func (f AgentQuestionForm) Update(msg tea.Msg) (AgentQuestionForm, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		currentQ := f.CurrentQuestion()
		hasOptions := len(currentQ.Options) > 0

		switch msg.String() {
		case "enter":
			// Submit current answer
			if hasOptions {
				if f.selectedOption >= 0 && f.selectedOption < len(currentQ.Options) {
					// Submit selected option
					f.answers = append(f.answers, agent.Answer{
						ID:    currentQ.ID,
						Value: currentQ.Options[f.selectedOption],
					})
					f.moveToNextQuestion()
					return f, nil
				}
			} else {
				// Submit text input
				value := strings.TrimSpace(f.textInput.Value())
				if value != "" {
					f.answers = append(f.answers, agent.Answer{
						ID:    currentQ.ID,
						Value: value,
					})
					f.moveToNextQuestion()
					return f, nil
				}
			}
			return f, nil

		case "up", "k":
			if hasOptions && f.selectedOption > 0 {
				f.selectedOption--
			}
			return f, nil

		case "down", "j":
			if hasOptions && f.selectedOption < len(currentQ.Options)-1 {
				f.selectedOption++
			}
			return f, nil

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if hasOptions {
				idx, _ := strconv.Atoi(msg.String())
				idx-- // Convert to 0-indexed
				if idx >= 0 && idx < len(currentQ.Options) {
					f.selectedOption = idx
				}
			}
			return f, nil
		}

		// Update text input for free-text questions
		if !hasOptions {
			f.textInput, cmd = f.textInput.Update(msg)
		}
	}

	return f, cmd
}

// moveToNextQuestion advances to the next question or marks the form as complete
func (f *AgentQuestionForm) moveToNextQuestion() {
	f.currentIndex++
	f.selectedOption = -1
	f.textInput.SetValue("")

	if f.currentIndex < len(f.questions) {
		currentQ := f.questions[f.currentIndex]
		if len(currentQ.Options) == 0 {
			// Free-text question
			f.textInput.Focus()
		} else {
			// Multiple choice question
			f.textInput.Blur()
			// Auto-select first option
			f.selectedOption = 0
		}
	}
}

// HandleAgentQuestionKey handles key presses in agent-question mode
func HandleAgentQuestionKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.agentQuestionForm == nil {
		m.actionMode = ActionModeNone
		return m, nil
	}

	// Handle Enter on a complete form - submit answers
	if msg.String() == "enter" && m.agentQuestionForm.IsComplete() {
		answers := m.agentQuestionForm.GetAnswers()

		// Close modal and continue with the appropriate flow
		m.actionMode = ActionModeNone
		m.agentQuestionForm = nil
		if m.pendingResumeTask != "" {
			taskID := m.pendingResumeTask
			m.pendingResumeTask = ""
			m.actionInProgress = true
			m.actionName = "Resuming..."
			ctx, cancel := context.WithCancel(context.Background())
			m.actionCancel = cancel
			streamCh, stdout, stderr := m.startLiveOutput()
			return m, tea.Batch(
				ResumeCmdWithContextAndStream(ctx, taskID, answers, stdout, stderr, streamCh),
				listenLiveOutputCmd(streamCh),
				spinnerTickCmd(),
			)
		}

		m.actionInProgress = true
		m.actionName = "Generating plan..."

		// Update question round
		nextRound := m.pendingPlanRequest.questionRound + 1
		m.pendingPlanRequest.questionRound = nextRound

		// Continue generation with answers
		return m, tea.Batch(
			ContinuePlanGenerationWithAnswers(
				m.pendingPlanRequest.description,
				m.pendingPlanRequest.constraints,
				m.pendingPlanRequest.granularity,
				answers,
				nextRound,
			),
			spinnerTickCmd(),
		)
	}

	// Update the form with the key message
	updatedForm, cmd := m.agentQuestionForm.Update(msg)
	m.agentQuestionForm = &updatedForm
	return m, cmd
}

// RenderAgentQuestionModal renders the agent question modal
func RenderAgentQuestionModal(m Model, form AgentQuestionForm) string {
	if form.IsComplete() {
		return renderCompleteMessage(m)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	questionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true)
	optionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	selectedOptionStyle := optionStyle.Copy().
		Background(lipgloss.Color("69")).
		Foreground(lipgloss.Color("15")).
		Bold(true)
	helpStyle := labelStyle.Copy()

	currentQ := form.CurrentQuestion()
	hasOptions := len(currentQ.Options) > 0

	title := titleStyle.Render("Agent has questions")
	progress := labelStyle.Render(fmt.Sprintf("Question %d of %d", form.currentIndex+1, len(form.questions)))

	// Build form content
	var lines []string
	lines = append(lines, title)
	lines = append(lines, progress)
	lines = append(lines, "")

	// Display question prompt
	lines = append(lines, questionStyle.Render(currentQ.Prompt))
	lines = append(lines, "")

	if hasOptions {
		// Display options
		for i, opt := range currentQ.Options {
			optionLine := fmt.Sprintf("%d) %s", i+1, opt)
			if i == form.selectedOption {
				lines = append(lines, selectedOptionStyle.Render(optionLine))
			} else {
				lines = append(lines, optionStyle.Render(optionLine))
			}
		}
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("↑/↓ or k/j: navigate • 1-9: quick select • Enter: confirm • ESC: cancel"))
	} else {
		// Display text input
		lines = append(lines, form.textInput.View())
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("Enter: submit answer • ESC: cancel"))
	}

	// Calculate modal width
	modalWidth := form.width
	if modalWidth > 90 {
		modalWidth = 90
	}
	if modalWidth < 50 {
		modalWidth = 50
	}
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
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

// renderCompleteMessage shows a message when all questions are answered
func renderCompleteMessage(m Model) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("46"))
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	title := titleStyle.Render("All questions answered")
	message := textStyle.Render("Press Enter to continue generating the plan.")
	helpText := helpStyle.Render("Enter: continue • ESC: cancel")

	lines := []string{
		title,
		"",
		message,
		"",
		helpText,
	}

	modalWidth := 60
	if m.windowWidth > 0 && m.windowWidth < modalWidth+4 {
		modalWidth = m.windowWidth - 4
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("46")).
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
