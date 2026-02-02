package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

type PlanDataLoaded struct {
	Plan          plan.WorkGraph
	PlanExists    bool
	ValidationErr string
	Err           error
}

type planDataRefreshMsg struct{}

func (m Model) LoadPlanData() tea.Cmd {
	return func() tea.Msg {
		path := plan.PlanPath()
		if m.projectRoot != "" {
			path = filepath.Join(m.projectRoot, plan.DefaultPlanFilename)
		}
		g, err := plan.Load(path)
		if err != nil {
			if errors.Is(err, plan.ErrPlanNotFound) || os.IsNotExist(err) {
				return PlanDataLoaded{Plan: plan.NewEmptyWorkGraph(), PlanExists: false, ValidationErr: "", Err: nil}
			}
			return PlanDataLoaded{Plan: plan.NewEmptyWorkGraph(), PlanExists: false, ValidationErr: "", Err: err}
		}
		if errs := plan.Validate(g); len(errs) != 0 {
			return PlanDataLoaded{
				Plan:          plan.NewEmptyWorkGraph(),
				PlanExists:    true,
				ValidationErr: summarizePlanValidation(errs),
				Err:           nil,
			}
		}

		return PlanDataLoaded{Plan: g, PlanExists: true, ValidationErr: "", Err: nil}
	}
}

func (m Model) PlanDataRefreshCmd() tea.Cmd {
	interval := time.Duration(m.config.TUI.PlanDataRefreshIntervalSeconds) * time.Second
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return planDataRefreshMsg{}
	})
}

func summarizePlanValidation(errs []plan.ValidationError) string {
	if len(errs) == 0 {
		return ""
	}
	first := errs[0]
	if first.Path != "" {
		return fmt.Sprintf("%s: %s", first.Path, first.Message)
	}
	return first.Message
}
