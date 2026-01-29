package execution

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestParseQuestionsExtractsAskUserQuestion(t *testing.T) {
	output := `log line
{"tool":"AskUserQuestion","id":"q1","prompt":"Pick one","options":["a","b"]}
more text`

	questions, err := ParseQuestions(output)
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
	q := questions[0]
	if q.ID != "q1" || q.Prompt != "Pick one" {
		t.Fatalf("unexpected question: %#v", q)
	}
	if len(q.Options) != 2 || q.Options[0] != "a" {
		t.Fatalf("unexpected options: %#v", q.Options)
	}
}

func TestParseQuestionsHandlesNameField(t *testing.T) {
	output := `{"name":"ask_user_question","id":"q2","question":"Continue?"}`

	questions, err := ParseQuestions(output)
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if len(questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(questions))
	}
	if questions[0].Prompt != "Continue?" {
		t.Fatalf("unexpected prompt: %#v", questions[0])
	}
}

func TestParseQuestionsNoneFound(t *testing.T) {
	questions, err := ParseQuestions("no json here")
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if questions != nil {
		t.Fatalf("expected nil questions, got %#v", questions)
	}
}

func TestParseQuestionsSkipsOtherJSON(t *testing.T) {
	output := `{"tool":"OtherTool","id":"x"}`
	questions, err := ParseQuestions(output)
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if questions != nil {
		t.Fatalf("expected nil questions, got %#v", questions)
	}
}

func TestParseQuestionsMissingPrompt(t *testing.T) {
	output := `{"tool":"AskUserQuestion","id":"q1"}`
	_, err := ParseQuestions(output)
	if err == nil {
		t.Fatalf("expected error for missing prompt")
	}
}

func TestParseQuestionsMultiple(t *testing.T) {
	output := `{"tool":"AskUserQuestion","id":"q1","prompt":"One"}
{"tool":"AskUserQuestion","id":"q2","prompt":"Two"}`

	questions, err := ParseQuestions(output)
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if len(questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(questions))
	}
	if questions[0].Prompt != "One" || questions[1].Prompt != "Two" {
		t.Fatalf("unexpected questions: %#v", questions)
	}
}

var _ = agent.Question{}
