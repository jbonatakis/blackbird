package execution

import (
	"testing"

	"github.com/jbonatakis/blackbird/internal/agent"
)

func TestResumeWithAnswer(t *testing.T) {
	run := RunRecord{
		TaskID: "task-1",
		Stdout: `{"tool":"AskUserQuestion","id":"q1","prompt":"Pick","options":["a","b"]}`,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	ctx, err := ResumeWithAnswer(run, []agent.Answer{{ID: "q1", Value: "a"}})
	if err != nil {
		t.Fatalf("ResumeWithAnswer: %v", err)
	}
	if len(ctx.Questions) != 1 || len(ctx.Answers) != 1 {
		t.Fatalf("expected questions/answers on context")
	}
	if ctx.Answers[0].Value != "a" {
		t.Fatalf("unexpected answer: %#v", ctx.Answers[0])
	}
}

func TestResumeWithAnswerMissing(t *testing.T) {
	run := RunRecord{
		TaskID: "task-1",
		Stdout: `{"tool":"AskUserQuestion","id":"q1","prompt":"Pick"}`,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	_, err := ResumeWithAnswer(run, []agent.Answer{})
	if err == nil {
		t.Fatalf("expected error for missing answer")
	}
}

func TestResumeWithAnswerInvalidOption(t *testing.T) {
	run := RunRecord{
		TaskID: "task-1",
		Stdout: `{"tool":"AskUserQuestion","id":"q1","prompt":"Pick","options":["a"]}`,
		Context: ContextPack{
			SchemaVersion: ContextPackSchemaVersion,
			Task:          TaskContext{ID: "task-1", Title: "Task"},
		},
	}

	_, err := ResumeWithAnswer(run, []agent.Answer{{ID: "q1", Value: "b"}})
	if err == nil {
		t.Fatalf("expected error for invalid option")
	}
}
