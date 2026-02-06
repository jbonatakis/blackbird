package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jbonatakis/blackbird/internal/agent"
	"github.com/jbonatakis/blackbird/internal/plan"
	"github.com/jbonatakis/blackbird/internal/planquality"
)

func TestRunPlanGeneratePrintsQualitySummaryNoFindings(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 1)
	scriptPath := writeMockAgentScript(t, tempDir)
	responsePath := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(validPlanNoFindings("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", responsePath)

	setPromptReader(strings.NewReader("yes\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=0, warning=0, total=0",
		"Quality summary (final): blocking=0, warning=0, total=0",
		"saved plan: " + plan.PlanPath(),
	})
	if strings.Contains(output, "quality auto-refine pass") {
		t.Fatalf("expected no auto-refine progress line, got output:\n%s", output)
	}
}

func TestRunPlanGeneratePrintsAutoRefineProgressAndWarnings(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 1)
	scriptPath := writeMockAgentScript(t, tempDir)
	response1Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})
	response2Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-2.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		Plan:          ptrPlanGraph(validPlanWithOneWarning("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", response1Path)
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_REFINE", response2Path)

	setPromptReader(strings.NewReader("yes\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=3, warning=0, total=3",
		"quality auto-refine pass 1/1",
		"Quality summary (final): blocking=0, warning=1, total=1",
		"Warning findings (non-blocking):",
		"- task-1.acceptanceCriteria [warning] leaf_acceptance_criteria_low_count: Leaf task has fewer than the recommended number of acceptance criteria.",
		"saved plan: " + plan.PlanPath(),
	})
	if strings.Contains(output, "Blocking findings:") {
		t.Fatalf("expected no blocking findings in final output, got:\n%s", output)
	}
}

func TestRunPlanGenerateBlockingRemainsAfterAutoRefineRequiresExplicitDecision(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 1)
	scriptPath := writeMockAgentScript(t, tempDir)
	response1Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})
	response2Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-2.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", response1Path)
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_REFINE", response2Path)

	setPromptReader(strings.NewReader("accept_anyway\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=3, warning=0, total=3",
		"quality auto-refine pass 1/1",
		"Quality summary (final): blocking=3, warning=0, total=3",
		"Blocking findings:",
		"Blocking findings remain. Choose action [revise/accept_anyway/cancel]:",
		"WARNING: blocking findings were overridden; saving plan anyway",
		"saved plan: " + plan.PlanPath(),
	})
	if strings.Count(output, "quality auto-refine pass ") != 1 {
		t.Fatalf("expected exactly one auto-refine pass line, got output:\n%s", output)
	}

	saved, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load saved plan: %v", err)
	}
	if !planquality.HasBlocking(planquality.Lint(saved)) {
		t.Fatalf("expected saved plan to retain blocking findings after bounded auto-refine and override")
	}
}

func TestRunPlanGenerateBlockingDecisionReviseRerunsQualityGateBeforeSave(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 0)
	scriptPath := writeMockAgentScript(t, tempDir)
	response1Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})
	response2Path := writeAgentResponseFile(t, filepath.Join(tempDir, "response-2.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanRefine,
		Plan:          ptrPlanGraph(validPlanNoFindings("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", response1Path)
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_REFINE", response2Path)

	setPromptReader(strings.NewReader("revise\nmake leaf tasks explicit and executable\nyes\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=3, warning=0, total=3",
		"Quality summary (final): blocking=3, warning=0, total=3",
		"Blocking findings remain. Choose action [revise/accept_anyway/cancel]:",
		"Revision request:",
		"Quality summary (initial): blocking=0, warning=0, total=0",
		"Quality summary (final): blocking=0, warning=0, total=0",
		"saved plan: " + plan.PlanPath(),
	})

	saved, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load saved plan: %v", err)
	}
	if planquality.HasBlocking(planquality.Lint(saved)) {
		t.Fatalf("expected revised plan to be non-blocking")
	}
}

func TestRunPlanGenerateBlockingDecisionAcceptAnywaySavesWithWarning(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 0)
	scriptPath := writeMockAgentScript(t, tempDir)
	responsePath := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", responsePath)

	setPromptReader(strings.NewReader("accept_anyway\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=3, warning=0, total=3",
		"Quality summary (final): blocking=3, warning=0, total=3",
		"Blocking findings remain. Choose action [revise/accept_anyway/cancel]:",
		"WARNING: blocking findings were overridden; saving plan anyway",
		"saved plan: " + plan.PlanPath(),
	})

	saved, err := plan.Load(plan.PlanPath())
	if err != nil {
		t.Fatalf("load saved plan: %v", err)
	}
	if !planquality.HasBlocking(planquality.Lint(saved)) {
		t.Fatalf("expected saved plan to retain blocking findings after override")
	}
}

func TestRunPlanGenerateBlockingDecisionCancelDoesNotWritePlan(t *testing.T) {
	tempDir := t.TempDir()
	chdirForTest(t, tempDir)

	writePlanningConfig(t, tempDir, 0)
	scriptPath := writeMockAgentScript(t, tempDir)
	responsePath := writeAgentResponseFile(t, filepath.Join(tempDir, "response-1.json"), agent.Response{
		SchemaVersion: agent.SchemaVersion,
		Type:          agent.RequestPlanGenerate,
		Plan:          ptrPlanGraph(blockingPlan("task-1")),
	})

	t.Setenv("BLACKBIRD_AGENT_PROVIDER", "")
	t.Setenv("BLACKBIRD_AGENT_CMD", scriptPath)
	t.Setenv("BLACKBIRD_AGENT_STREAM", "")
	t.Setenv("BLACKBIRD_AGENT_DEBUG", "")
	t.Setenv("MOCK_AGENT_RESPONSE_PLAN_GENERATE", responsePath)

	setPromptReader(strings.NewReader("cancel\n"))
	t.Cleanup(func() { setPromptReader(os.Stdin) })

	output, err := captureStdout(func() error {
		return runPlanGenerate([]string{
			"--description", "Generate a deterministic quality-gate test plan",
			"--constraint", "keep output stable",
		})
	})
	if err != nil {
		t.Fatalf("runPlanGenerate() error = %v\noutput:\n%s", err, output)
	}

	assertOutputContainsInOrder(t, output, []string{
		"Quality summary (initial): blocking=3, warning=0, total=3",
		"Quality summary (final): blocking=3, warning=0, total=3",
		"Blocking findings remain. Choose action [revise/accept_anyway/cancel]:",
		"aborted; plan unchanged",
	})
	if strings.Contains(output, "saved plan: ") {
		t.Fatalf("expected cancel path to avoid saving, got output:\n%s", output)
	}
	if _, err := os.Stat(plan.PlanPath()); !os.IsNotExist(err) {
		if err == nil {
			t.Fatalf("expected no plan file to be written")
		}
		t.Fatalf("stat plan path: %v", err)
	}
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
}

func writePlanningConfig(t *testing.T, root string, maxAutoRefinePasses int) {
	t.Helper()

	configDir := filepath.Join(root, ".blackbird")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	cfg := fmt.Sprintf(`{"schemaVersion":1,"planning":{"maxPlanAutoRefinePasses":%d}}`, maxAutoRefinePasses)
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func writeMockAgentScript(t *testing.T, root string) string {
	t.Helper()

	scriptPath := filepath.Join(root, "mock-agent.sh")
	script := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		`request="$(cat)"`,
		`response_path=""`,
		`case "${request}" in`,
		`  *'"type":"plan_generate"'*) response_path="${MOCK_AGENT_RESPONSE_PLAN_GENERATE:-}" ;;`,
		`  *'"type":"plan_refine"'*) response_path="${MOCK_AGENT_RESPONSE_PLAN_REFINE:-}" ;;`,
		"esac",
		`if [ -z "${response_path}" ]; then`,
		`  echo "missing response for request type" >&2`,
		"  exit 1",
		"fi",
		`cat "${response_path}"`,
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock agent script: %v", err)
	}
	return scriptPath
}

func writeAgentResponseFile(t *testing.T, path string, resp agent.Response) string {
	t.Helper()

	data, err := agent.EncodeResponse(resp)
	if err != nil {
		t.Fatalf("encode response: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write response file: %v", err)
	}
	return path
}

func assertOutputContainsInOrder(t *testing.T, output string, snippets []string) {
	t.Helper()

	searchFrom := 0
	for _, snippet := range snippets {
		idx := strings.Index(output[searchFrom:], snippet)
		if idx < 0 {
			t.Fatalf("expected output to contain %q after index %d\nfull output:\n%s", snippet, searchFrom, output)
		}
		searchFrom += idx + len(snippet)
	}
}

func ptrPlanGraph(g plan.WorkGraph) *plan.WorkGraph {
	out := plan.Clone(g)
	return &out
}

func validPlanNoFindings(id string) plan.WorkGraph {
	return qualityPlan(id,
		"Implement deterministic CLI quality summary rendering for generated plans and keep line output stable for tests.",
		[]string{
			"`blackbird plan generate` prints initial and final quality summaries.",
			"`go test ./internal/cli/...` passes.",
		},
		"Implement quality summary rendering and verify with `go test ./internal/cli/...`.",
	)
}

func validPlanWithOneWarning(id string) plan.WorkGraph {
	return qualityPlan(id,
		"Implement deterministic CLI quality summary rendering for generated plans and keep line output stable for tests.",
		[]string{
			"`go test ./internal/cli/...` passes.",
		},
		"Implement quality summary rendering and verify with `go test ./internal/cli/...`.",
	)
}

func blockingPlan(id string) plan.WorkGraph {
	return qualityPlan(id, "TODO", []string{}, "")
}

func qualityPlan(id string, description string, acceptanceCriteria []string, prompt string) plan.WorkGraph {
	now := time.Date(2026, 2, 6, 12, 0, 0, 0, time.UTC)
	criteriaCopy := make([]string, len(acceptanceCriteria))
	copy(criteriaCopy, acceptanceCriteria)
	return plan.WorkGraph{
		SchemaVersion: plan.SchemaVersion,
		Items: map[string]plan.WorkItem{
			id: {
				ID:                 id,
				Title:              "Task " + id,
				Description:        description,
				AcceptanceCriteria: criteriaCopy,
				Prompt:             prompt,
				ParentID:           nil,
				ChildIDs:           []string{},
				Deps:               []string{},
				Status:             plan.StatusTodo,
				CreatedAt:          now,
				UpdatedAt:          now,
			},
		},
	}
}
