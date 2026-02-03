package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/config"
	memproxy "github.com/jbonatakis/blackbird/internal/memory/proxy"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func Start() error {
	model := newStartupModel("")
	handle, err := startMemoryProxy(model)
	if err != nil {
		return err
	}
	if handle != nil {
		defer func() { _ = handle.Close() }()
	}
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, runErr := program.Run()
	return runErr
}

func newStartupModel(projectRoot string) Model {
	root := resolveProjectRoot(projectRoot)
	cfg, err := config.LoadConfig(root)
	if err != nil {
		cfg = config.DefaultResolvedConfig()
	}
	model := NewModel(plan.NewEmptyWorkGraph())
	model.planExists = false
	model.viewMode = ViewModeHome
	model.projectRoot = root
	model.config = cfg
	return model
}

func startMemoryProxy(model Model) (*memproxy.SupervisorHandle, error) {
	provider := ""
	if runtime, err := agent.NewRuntimeFromEnv(); err == nil {
		provider = runtime.Provider
	}
	return memproxy.StartSupervisor(memproxy.SupervisorOptions{
		ProviderID:  provider,
		ProjectRoot: model.projectRoot,
		Config:      &model.config,
	})
}

func resolveProjectRoot(projectRoot string) string {
	if projectRoot != "" {
		return projectRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func formatPlanErrors(path string, errs []plan.ValidationError) error {
	var b strings.Builder
	fmt.Fprintf(&b, "invalid plan: %s\n", path)
	for _, e := range errs {
		fmt.Fprintf(&b, "- %s: %s\n", e.Path, e.Message)
	}
	return errors.New(strings.TrimRight(b.String(), "\n"))
}
