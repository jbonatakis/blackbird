package tui

import "testing"

func TestNewStartupModelNoPlanFile(t *testing.T) {
	model := newStartupModel()

	if model.viewMode != ViewModeHome {
		t.Fatalf("expected viewMode to be ViewModeHome, got %v", model.viewMode)
	}
	if model.planExists {
		t.Fatalf("expected planExists to be false when no plan file exists")
	}
	if len(model.plan.Items) != 0 {
		t.Fatalf("expected empty plan on startup, got %d items", len(model.plan.Items))
	}
}
