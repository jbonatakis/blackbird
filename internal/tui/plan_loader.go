package tui

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/plan"
)

const planDataRefreshInterval = 5 * time.Second

type PlanDataLoaded struct {
	Plan plan.WorkGraph
	Err  error
}

type planDataRefreshMsg struct{}

func (m Model) LoadPlanData() tea.Cmd {
	return func() tea.Msg {
		baseDir, err := os.Getwd()
		if err != nil {
			return PlanDataLoaded{Plan: plan.WorkGraph{}, Err: err}
		}

		path := filepath.Join(baseDir, plan.DefaultPlanFilename)
		g, err := plan.Load(path)
		if err != nil {
			return PlanDataLoaded{Plan: plan.WorkGraph{}, Err: err}
		}
		if errs := plan.Validate(g); len(errs) != 0 {
			return PlanDataLoaded{Plan: plan.WorkGraph{}, Err: formatPlanErrors(path, errs)}
		}

		return PlanDataLoaded{Plan: g, Err: nil}
	}
}

func PlanDataRefreshCmd() tea.Cmd {
	return tea.Tick(planDataRefreshInterval, func(time.Time) tea.Msg {
		return planDataRefreshMsg{}
	})
}
