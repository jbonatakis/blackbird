package tui

import (
	"bytes"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type PlanActionComplete struct {
	Action  string
	Success bool
	Output  string
	Err     error
}

type ExecuteActionComplete struct {
	Action  string
	Success bool
	Output  string
	Err     error
}

func PlanGenerateCmd() tea.Cmd {
	return runPlanAction("plan generate", []string{"plan", "generate"})
}

func PlanRefineCmd() tea.Cmd {
	return runPlanAction("plan refine", []string{"plan", "refine"})
}

func ExecuteCmd() tea.Cmd {
	return runExecuteAction("execute", []string{"execute"})
}

func ResumeCmd(taskID string) tea.Cmd {
	return runExecuteAction("resume", []string{"resume", taskID})
}

func SetStatusCmd(id string, status string) tea.Cmd {
	return runExecuteAction("set-status", []string{"set-status", id, status})
}

func runPlanAction(action string, args []string) tea.Cmd {
	return func() tea.Msg {
		output, err := runCommand(args)
		return PlanActionComplete{Action: action, Success: err == nil, Output: output, Err: err}
	}
}

func runExecuteAction(action string, args []string) tea.Cmd {
	return func() tea.Msg {
		output, err := runCommand(args)
		return ExecuteActionComplete{Action: action, Success: err == nil, Output: output, Err: err}
	}
}

func runCommand(args []string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		exe = os.Args[0]
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	runErr := cmd.Run()
	return buf.String(), runErr
}
