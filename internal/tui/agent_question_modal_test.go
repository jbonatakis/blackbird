package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestAgentQuestionForm_FreeTextQuestion(t *testing.T) {
	questions := []agent.Question{
		{
			ID:     "q1",
			Prompt: "What is your preferred database?",
		},
	}

	form := NewAgentQuestionForm(questions)

	if form.IsComplete() {
		t.Error("form should not be complete initially")
	}

	currentQ := form.CurrentQuestion()
	if currentQ.ID != "q1" {
		t.Errorf("expected question q1, got %s", currentQ.ID)
	}

	// Simulate typing an answer
	form.textInput.SetValue("PostgreSQL")

	// Simulate pressing Enter
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	if !form.IsComplete() {
		t.Error("form should be complete after answering the only question")
	}

	answers := form.GetAnswers()
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}

	if answers[0].ID != "q1" {
		t.Errorf("expected answer ID q1, got %s", answers[0].ID)
	}

	if answers[0].Value != "PostgreSQL" {
		t.Errorf("expected answer value PostgreSQL, got %s", answers[0].Value)
	}
}

func TestAgentQuestionForm_MultipleChoiceQuestion(t *testing.T) {
	questions := []agent.Question{
		{
			ID:      "q1",
			Prompt:  "Which framework?",
			Options: []string{"React", "Vue", "Angular"},
		},
	}

	form := NewAgentQuestionForm(questions)

	if form.IsComplete() {
		t.Error("form should not be complete initially")
	}

	currentQ := form.CurrentQuestion()
	if currentQ.ID != "q1" {
		t.Errorf("expected question q1, got %s", currentQ.ID)
	}

	// Simulate selecting option 2 (Vue)
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	form = updatedForm

	if form.selectedOption != 1 { // 0-indexed
		t.Errorf("expected selectedOption 1, got %d", form.selectedOption)
	}

	// Simulate pressing Enter
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	if !form.IsComplete() {
		t.Error("form should be complete after answering the only question")
	}

	answers := form.GetAnswers()
	if len(answers) != 1 {
		t.Fatalf("expected 1 answer, got %d", len(answers))
	}

	if answers[0].ID != "q1" {
		t.Errorf("expected answer ID q1, got %s", answers[0].ID)
	}

	if answers[0].Value != "Vue" {
		t.Errorf("expected answer value Vue, got %s", answers[0].Value)
	}
}

func TestAgentQuestionForm_MultipleQuestions(t *testing.T) {
	questions := []agent.Question{
		{
			ID:     "q1",
			Prompt: "What is the project name?",
		},
		{
			ID:      "q2",
			Prompt:  "Which language?",
			Options: []string{"Go", "Python", "JavaScript"},
		},
		{
			ID:     "q3",
			Prompt: "Any special requirements?",
		},
	}

	form := NewAgentQuestionForm(questions)

	// Answer first question (free text)
	form.textInput.SetValue("MyApp")
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	if form.IsComplete() {
		t.Error("form should not be complete after answering 1/3 questions")
	}

	if form.currentIndex != 1 {
		t.Errorf("expected currentIndex 1, got %d", form.currentIndex)
	}

	// Answer second question (multiple choice)
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	form = updatedForm
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	if form.IsComplete() {
		t.Error("form should not be complete after answering 2/3 questions")
	}

	if form.currentIndex != 2 {
		t.Errorf("expected currentIndex 2, got %d", form.currentIndex)
	}

	// Answer third question (free text)
	form.textInput.SetValue("High performance")
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	if !form.IsComplete() {
		t.Error("form should be complete after answering all 3 questions")
	}

	answers := form.GetAnswers()
	if len(answers) != 3 {
		t.Fatalf("expected 3 answers, got %d", len(answers))
	}

	if answers[0].Value != "MyApp" {
		t.Errorf("expected first answer MyApp, got %s", answers[0].Value)
	}

	if answers[1].Value != "Go" {
		t.Errorf("expected second answer Go, got %s", answers[1].Value)
	}

	if answers[2].Value != "High performance" {
		t.Errorf("expected third answer 'High performance', got %s", answers[2].Value)
	}
}

func TestAgentQuestionForm_NavigateOptions(t *testing.T) {
	questions := []agent.Question{
		{
			ID:      "q1",
			Prompt:  "Choose color",
			Options: []string{"Red", "Green", "Blue"},
		},
	}

	form := NewAgentQuestionForm(questions)

	// First option should be auto-selected for multiple choice (changed behavior)
	if form.selectedOption != 0 {
		t.Errorf("expected initial selectedOption 0 (auto-selected), got %d", form.selectedOption)
	}

	// Navigate to option 2 using number key
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	form = updatedForm

	if form.selectedOption != 1 {
		t.Errorf("expected selectedOption 1 after pressing 2, got %d", form.selectedOption)
	}

	// Navigate up
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyUp})
	form = updatedForm

	if form.selectedOption != 0 {
		t.Errorf("expected selectedOption 0 after up, got %d", form.selectedOption)
	}

	// Navigate down twice
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyDown})
	form = updatedForm
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyDown})
	form = updatedForm

	if form.selectedOption != 2 {
		t.Errorf("expected selectedOption 2 after down twice, got %d", form.selectedOption)
	}

	// Try to go down beyond limit (should stay at 2)
	updatedForm, _ = form.Update(tea.KeyMsg{Type: tea.KeyDown})
	form = updatedForm

	if form.selectedOption != 2 {
		t.Errorf("expected selectedOption to stay at 2, got %d", form.selectedOption)
	}
}

func TestAgentQuestionForm_EmptyTextNotSubmitted(t *testing.T) {
	questions := []agent.Question{
		{
			ID:     "q1",
			Prompt: "Enter a value",
		},
	}

	form := NewAgentQuestionForm(questions)

	// Try to submit without entering text
	updatedForm, _ := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	form = updatedForm

	// Should not advance
	if form.IsComplete() {
		t.Error("form should not be complete when empty text is submitted")
	}

	if len(form.GetAnswers()) != 0 {
		t.Error("no answers should be recorded for empty submission")
	}
}

func TestAgentQuestionForm_FirstMultipleChoiceAutoSelected(t *testing.T) {
	questions := []agent.Question{
		{
			ID:      "q1",
			Prompt:  "Which framework?",
			Options: []string{"React", "Vue", "Angular"},
		},
	}

	form := NewAgentQuestionForm(questions)

	// First option should be auto-selected for multiple choice
	if form.selectedOption != 0 {
		t.Errorf("Expected first option to be auto-selected (0), got %d", form.selectedOption)
	}

	// Text input should be blurred for multiple choice
	if form.textInput.Focused() {
		t.Error("Expected text input to be blurred for multiple choice question")
	}
}

func TestAgentQuestionForm_FirstFreeTextNotAutoSelected(t *testing.T) {
	questions := []agent.Question{
		{
			ID:     "q1",
			Prompt: "Enter your name",
		},
	}

	form := NewAgentQuestionForm(questions)

	// No option should be selected for free text
	if form.selectedOption != -1 {
		t.Errorf("Expected no option selected (-1) for free text, got %d", form.selectedOption)
	}

	// Text input should be focused for free text
	if !form.textInput.Focused() {
		t.Error("Expected text input to be focused for free text question")
	}
}
