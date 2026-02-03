package artifact

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/jbonatakis/blackbird/internal/memory/canonical"
)

func detectOutcomeStatus(log canonical.Log) string {
	blocked := false
	failed := false

	for _, item := range log.Items {
		if item.Message != nil && item.Message.Role == "assistant" {
			text := strings.ToLower(item.Message.Content)
			if strings.Contains(text, "blocked") || strings.Contains(text, "waiting on") {
				blocked = true
			}
			if strings.Contains(text, "failed") || strings.Contains(text, "error") || strings.Contains(text, "unable") || strings.Contains(text, "could not") {
				failed = true
			}
		}
		if item.ToolResult != nil {
			if item.ToolResult.Error != "" {
				failed = true
			}
			text := strings.ToLower(item.ToolResult.Result)
			if strings.Contains(text, "blocked") {
				blocked = true
			}
			if strings.Contains(text, "error") || strings.Contains(text, "failed") || strings.Contains(text, "panic") {
				failed = true
			}
			if code := parseExitCode(item.ToolResult.Result); code != nil && *code != 0 {
				failed = true
			}
		}
	}

	if blocked {
		return "blocked"
	}
	if failed {
		return "fail"
	}
	return "success"
}

func summarizeOutcome(log canonical.Log) []string {
	ref, ok := lastAssistantMessage(log)
	if !ok || ref.item.Message == nil {
		return nil
	}
	text := ref.item.Message.Content
	lines := strings.Split(text, "\n")
	var bullets []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			bullets = append(bullets, strings.TrimSpace(trimmed[2:]))
		}
	}
	if len(bullets) > 0 {
		return limitStrings(bullets, 3)
	}

	for _, sentence := range splitSentences(text) {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		return []string{sentence}
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	return []string{trimmed}
}

func splitSentences(text string) []string {
	seps := []string{". ", "? ", "! "}
	for _, sep := range seps {
		if strings.Contains(text, sep) {
			return strings.Split(text, sep)
		}
	}
	return []string{text}
}

func limitStrings(values []string, max int) []string {
	if len(values) <= max {
		return values
	}
	return values[:max]
}

func extractFiles(log canonical.Log) []string {
	files := make(map[string]struct{})
	for _, item := range log.Items {
		if item.ToolResult != nil {
			collectFiles(item.ToolResult.Result, files)
			collectFiles(item.ToolResult.Error, files)
		}
		if item.Message != nil && item.Message.Role == "assistant" {
			collectFiles(item.Message.Content, files)
		}
	}
	if len(files) == 0 {
		return nil
	}
	out := make([]string, 0, len(files))
	for file := range files {
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

func collectFiles(text string, files map[string]struct{}) {
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "diff --git ") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				path := strings.TrimPrefix(parts[3], "b/")
				files[path] = struct{}{}
			}
			continue
		}
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				path := strings.TrimPrefix(parts[1], "a/")
				path = strings.TrimPrefix(path, "b/")
				if path != "/dev/null" {
					files[path] = struct{}{}
				}
			}
			continue
		}
		if len(line) > 2 && (line[0] == 'A' || line[0] == 'M' || line[0] == 'D' || line[0] == 'R' || line[0] == 'C') && line[1] == ' ' {
			path := strings.TrimSpace(line[2:])
			if path != "" {
				files[path] = struct{}{}
			}
		}
	}
}

func extractCommands(log canonical.Log) ([]CommandResult, []Provenance) {
	commands := []CommandResult{}
	commandRefs := map[string]*CommandResult{}
	provenance := []Provenance{}

	for idx, item := range log.Items {
		if item.ToolCall == nil {
			continue
		}
		call := item.ToolCall
		cmds := commandsFromToolCall(call)
		if len(cmds) == 0 {
			continue
		}
		for _, cmd := range cmds {
			command := CommandResult{Command: cmd}
			commands = append(commands, command)
			if call.ID != "" {
				commandRefs[call.ID] = &commands[len(commands)-1]
			}
			provenance = append(provenance, Provenance{
				ItemIndex: idx,
				ItemType:  string(item.Type),
			})
		}
	}

	for idx, item := range log.Items {
		if item.ToolResult == nil {
			continue
		}
		code := parseExitCode(item.ToolResult.Result)
		if code == nil {
			continue
		}
		if item.ToolResult.ID != "" {
			if cmd := commandRefs[item.ToolResult.ID]; cmd != nil {
				cmd.ExitCode = code
				provenance = append(provenance, Provenance{ItemIndex: idx, ItemType: string(item.Type)})
				continue
			}
		}
		commands = append(commands, CommandResult{Command: "", ExitCode: code})
		provenance = append(provenance, Provenance{ItemIndex: idx, ItemType: string(item.Type)})
	}

	return commands, provenance
}

func commandsFromToolCall(call *canonical.ToolCall) []string {
	if call == nil {
		return nil
	}
	name := strings.ToLower(call.Name)
	if name == "" {
		return nil
	}
	if !strings.Contains(name, "command") && !strings.Contains(name, "shell") && !strings.Contains(name, "exec") && !strings.Contains(name, "bash") {
		return nil
	}
	args := strings.TrimSpace(call.Arguments)
	if args == "" {
		return nil
	}

	var parsed map[string]any
	if json.Unmarshal([]byte(args), &parsed) == nil {
		if value, ok := parsed["command"]; ok {
			if cmd, ok := value.(string); ok && cmd != "" {
				return []string{cmd}
			}
		}
		if value, ok := parsed["cmd"]; ok {
			if cmd, ok := value.(string); ok && cmd != "" {
				return []string{cmd}
			}
		}
		if value, ok := parsed["commands"]; ok {
			if arr, ok := value.([]any); ok {
				var cmds []string
				for _, item := range arr {
					if cmd, ok := item.(string); ok && cmd != "" {
						cmds = append(cmds, cmd)
					}
				}
				if len(cmds) > 0 {
					return cmds
				}
			}
		}
	}

	return []string{args}
}

func parseExitCode(text string) *int {
	lower := strings.ToLower(text)
	markers := []string{"exit status", "exit code", "exitcode"}
	for _, marker := range markers {
		if idx := strings.Index(lower, marker); idx >= 0 {
			rest := lower[idx+len(marker):]
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				continue
			}
			val, err := strconv.Atoi(strings.Trim(fields[0], ":"))
			if err == nil {
				return &val
			}
		}
	}
	return nil
}

func extractErrors(log canonical.Log) ([]string, []Provenance) {
	seen := make(map[string]struct{})
	var errors []string
	var provenance []Provenance
	for idx, item := range log.Items {
		if item.ToolResult == nil {
			continue
		}
		lines := strings.Split(item.ToolResult.Result, "\n")
		lines = append(lines, strings.Split(item.ToolResult.Error, "\n")...)
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			lower := strings.ToLower(trimmed)
			if !strings.Contains(lower, "error") && !strings.Contains(lower, "failed") && !strings.Contains(lower, "panic") {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			errors = append(errors, trimmed)
			provenance = append(provenance, Provenance{ItemIndex: idx, ItemType: string(item.Type)})
		}
	}
	return errors, provenance
}
