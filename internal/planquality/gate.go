package planquality

import (
	"errors"
	"fmt"

	"github.com/jbonatakis/blackbird/internal/plan"
)

var ErrRefineCallbackRequired = errors.New("planquality: refine callback is required when blocking findings exist and auto-refine passes are enabled")

// AutoRefineInput contains deterministic input for one auto-refine callback invocation.
type AutoRefineInput struct {
	Pass          int
	MaxPasses     int
	Plan          plan.WorkGraph
	Findings      []PlanQualityFinding
	ChangeRequest string
}

// AutoRefineFunc executes one refine pass and returns a refined plan.
type AutoRefineFunc func(input AutoRefineInput) (plan.WorkGraph, error)

// QualityGateResult captures quality-gate outputs for downstream CLI/TUI rendering.
type QualityGateResult struct {
	FinalPlan           plan.WorkGraph
	InitialFindings     []PlanQualityFinding
	FinalFindings       []PlanQualityFinding
	AutoRefinePassesRun int
}

// RunQualityGate executes lint -> optional refine -> relint passes with a bounded loop.
func RunQualityGate(initialPlan plan.WorkGraph, maxAutoRefinePasses int, refine AutoRefineFunc) (QualityGateResult, error) {
	if maxAutoRefinePasses < 0 {
		maxAutoRefinePasses = 0
	}

	currentPlan := plan.Clone(initialPlan)
	initialFindings := Lint(currentPlan)
	finalFindings := append([]PlanQualityFinding(nil), initialFindings...)

	result := QualityGateResult{
		FinalPlan:           plan.Clone(currentPlan),
		InitialFindings:     append([]PlanQualityFinding(nil), initialFindings...),
		FinalFindings:       append([]PlanQualityFinding(nil), finalFindings...),
		AutoRefinePassesRun: 0,
	}

	for result.AutoRefinePassesRun < maxAutoRefinePasses && HasBlocking(finalFindings) {
		if refine == nil {
			return result, ErrRefineCallbackRequired
		}

		pass := result.AutoRefinePassesRun + 1
		nextPlan, err := refine(AutoRefineInput{
			Pass:          pass,
			MaxPasses:     maxAutoRefinePasses,
			Plan:          plan.Clone(currentPlan),
			Findings:      append([]PlanQualityFinding(nil), finalFindings...),
			ChangeRequest: BuildRefineRequest(finalFindings),
		})
		if err != nil {
			return result, err
		}

		if errs := plan.Validate(nextPlan); len(errs) != 0 {
			return result, fmt.Errorf("planquality: auto-refine pass %d returned invalid plan: %s", pass, summarizeValidationErrors(errs))
		}

		currentPlan = plan.Clone(nextPlan)
		finalFindings = Lint(currentPlan)
		result.FinalPlan = plan.Clone(currentPlan)
		result.FinalFindings = append([]PlanQualityFinding(nil), finalFindings...)
		result.AutoRefinePassesRun++
	}

	return result, nil
}

func summarizeValidationErrors(errs []plan.ValidationError) string {
	if len(errs) == 0 {
		return ""
	}
	if len(errs) == 1 {
		return errs[0].Error()
	}
	return fmt.Sprintf("%s (+%d more)", errs[0].Error(), len(errs)-1)
}
