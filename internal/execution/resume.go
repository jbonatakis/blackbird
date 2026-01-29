package execution

import (
	"fmt"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// ResumeWithAnswer validates answers and builds continuation context.
func ResumeWithAnswer(run RunRecord, answers []agent.Answer) (ContextPack, error) {
	questions, err := ParseQuestions(run.Stdout)
	if err != nil {
		return ContextPack{}, err
	}
	if len(questions) == 0 {
		return ContextPack{}, fmt.Errorf("no questions found in run output")
	}

	answerByID := map[string]agent.Answer{}
	for _, ans := range answers {
		answerByID[ans.ID] = ans
	}

	for _, q := range questions {
		ans, ok := answerByID[q.ID]
		if !ok {
			return ContextPack{}, fmt.Errorf("missing answer for question %q", q.ID)
		}
		if len(q.Options) > 0 && !containsOption(q.Options, ans.Value) {
			return ContextPack{}, fmt.Errorf("answer %q not in options for question %q", ans.Value, q.ID)
		}
	}

	ctx := run.Context
	ctx.Questions = questions
	ctx.Answers = answers

	return ctx, nil
}

func containsOption(options []string, value string) bool {
	for _, opt := range options {
		if opt == value {
			return true
		}
	}
	return false
}
