package execution

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jbonatakis/blackbird/internal/agent"
)

// ParseQuestions scans agent output for AskUserQuestion tool invocations and returns parsed questions.
func ParseQuestions(agentOutput string) ([]agent.Question, error) {
	candidates := findJSONObjectCandidates(agentOutput)
	if len(candidates) == 0 {
		return nil, nil
	}

	var out []agent.Question
	for _, candidate := range candidates {
		var header struct {
			Tool string `json:"tool"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(candidate), &header); err != nil {
			continue
		}
		toolName := strings.TrimSpace(header.Tool)
		if toolName == "" {
			toolName = strings.TrimSpace(header.Name)
		}
		if !isAskUserTool(toolName) {
			continue
		}

		var payload struct {
			ID       string   `json:"id"`
			Prompt   string   `json:"prompt"`
			Question string   `json:"question"`
			Options  []string `json:"options"`
		}
		if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
			return nil, fmt.Errorf("decode question payload: %w", err)
		}
		prompt := strings.TrimSpace(payload.Prompt)
		if prompt == "" {
			prompt = strings.TrimSpace(payload.Question)
		}
		if prompt == "" {
			return nil, fmt.Errorf("question prompt required")
		}

		out = append(out, agent.Question{
			ID:      payload.ID,
			Prompt:  prompt,
			Options: payload.Options,
		})
	}

	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

func isAskUserTool(name string) bool {
	if name == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == "askuserquestion" || normalized == "ask_user_question"
}

func findJSONObjectCandidates(output string) []string {
	var objs []string
	var start int
	inString := false
	escape := false
	depth := 0

	for i, r := range output {
		if escape {
			escape = false
			continue
		}
		if r == '\\' && inString {
			escape = true
			continue
		}
		if r == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if r == '{' {
			if depth == 0 {
				start = i
			}
			depth++
			continue
		}
		if r == '}' {
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 {
				candidate := output[start : i+1]
				if json.Valid([]byte(candidate)) {
					objs = append(objs, candidate)
				}
			}
		}
	}

	return objs
}
