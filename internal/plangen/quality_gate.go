package plangen

import (
	"context"
	"path/filepath"

	"github.com/jbonatakis/blackbird/internal/config"
	"github.com/jbonatakis/blackbird/internal/plan"
	"github.com/jbonatakis/blackbird/internal/planquality"
)

// QualityGateRefineFunc performs one plan_refine attempt for quality auto-refine.
type QualityGateRefineFunc func(ctx context.Context, changeRequest string, currentPlan plan.WorkGraph) (plan.WorkGraph, error)

// RunQualityGate applies shared deterministic plan-quality orchestration.
func RunQualityGate(ctx context.Context, initialPlan plan.WorkGraph, maxAutoRefinePasses int, refine QualityGateRefineFunc) (planquality.QualityGateResult, error) {
	var refineCallback planquality.AutoRefineFunc
	if refine != nil {
		refineCallback = func(input planquality.AutoRefineInput) (plan.WorkGraph, error) {
			if err := contextErr(ctx); err != nil {
				return plan.WorkGraph{}, err
			}
			return refine(ctx, input.ChangeRequest, input.Plan)
		}
	}
	return planquality.RunQualityGate(initialPlan, maxAutoRefinePasses, refineCallback)
}

// ResolveMaxAutoRefinePasses loads resolved config and returns planning.maxPlanAutoRefinePasses.
// On config load failure it falls back to defaults.
func ResolveMaxAutoRefinePasses(projectRoot string) int {
	cfg, err := config.LoadConfig(projectRoot)
	if err != nil {
		return config.DefaultResolvedConfig().Planning.MaxPlanAutoRefinePasses
	}
	return cfg.Planning.MaxPlanAutoRefinePasses
}

// ResolveMaxAutoRefinePassesFromPlanPath resolves max auto-refine passes from a plan file path.
func ResolveMaxAutoRefinePassesFromPlanPath(planPath string) int {
	return ResolveMaxAutoRefinePasses(filepath.Dir(planPath))
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
