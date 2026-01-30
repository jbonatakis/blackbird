package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

func Start() error {
	model := newStartupModel()
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	return err
}

func newStartupModel() Model {
	model := NewModel(plan.NewEmptyWorkGraph())
	model.planExists = false
	model.viewMode = ViewModeHome
	return model
}

func planPath() string {
	wd, err := os.Getwd()
	if err != nil {
		return plan.DefaultPlanFilename
	}
	return filepath.Join(wd, plan.DefaultPlanFilename)
}

func formatPlanErrors(path string, errs []plan.ValidationError) error {
	var b strings.Builder
	fmt.Fprintf(&b, "invalid plan: %s\n", path)
	for _, e := range errs {
		fmt.Fprintf(&b, "- %s: %s\n", e.Path, e.Message)
	}
	return errors.New(strings.TrimRight(b.String(), "\n"))
}
