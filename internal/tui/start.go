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
	path := planPath()

	g, err := plan.Load(path)
	if err != nil {
		if errors.Is(err, plan.ErrPlanNotFound) {
			return fmt.Errorf("plan file not found: %s (run `blackbird init`)", path)
		}
		return err
	}

	if errs := plan.Validate(g); len(errs) > 0 {
		return formatPlanErrors(path, errs)
	}

	model := NewModel(g)
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err = program.Run()
	return err
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
