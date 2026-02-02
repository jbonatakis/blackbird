package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jbonatakis/blackbird/internal/config"
)

func TestRunDataRefreshCmdUsesConfigInterval(t *testing.T) {
	model := Model{config: config.DefaultResolvedConfig()}
	model.config.TUI.RunDataRefreshIntervalSeconds = 1

	msg := assertRefreshCmdDelay(t, model.RunDataRefreshCmd(), 900*time.Millisecond, 2*time.Second)
	if _, ok := msg.(runDataRefreshMsg); !ok {
		t.Fatalf("expected runDataRefreshMsg, got %T", msg)
	}
}

func TestPlanDataRefreshCmdUsesConfigInterval(t *testing.T) {
	model := Model{config: config.DefaultResolvedConfig()}
	model.config.TUI.PlanDataRefreshIntervalSeconds = 1

	msg := assertRefreshCmdDelay(t, model.PlanDataRefreshCmd(), 900*time.Millisecond, 2*time.Second)
	if _, ok := msg.(planDataRefreshMsg); !ok {
		t.Fatalf("expected planDataRefreshMsg, got %T", msg)
	}
}

func assertRefreshCmdDelay(t *testing.T, cmd tea.Cmd, minDelay, maxDelay time.Duration) tea.Msg {
	t.Helper()

	start := time.Now()
	done := make(chan tea.Msg, 1)
	go func() {
		done <- cmd()
	}()

	select {
	case msg := <-done:
		elapsed := time.Since(start)
		if elapsed < minDelay {
			t.Fatalf("refresh cmd fired too quickly: %s < %s", elapsed, minDelay)
		}
		if elapsed > maxDelay {
			t.Fatalf("refresh cmd fired too late: %s > %s", elapsed, maxDelay)
		}
		return msg
	case <-time.After(maxDelay + 250*time.Millisecond):
		t.Fatalf("refresh cmd did not fire within %s", maxDelay)
		return nil
	}
}
