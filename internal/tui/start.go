package tui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func Start() error {
	model := newStartupModel("")
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
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
	model.settings = NewSettingsState(root, cfg)
	return model
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
