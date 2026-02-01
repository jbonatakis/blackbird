package agent

import (
	"errors"
	"time"

	"github.com/jbonatakis/blackbird/internal/plan"
)

// ResponseToPlan converts an agent response into a plan.WorkGraph.
func ResponseToPlan(base plan.WorkGraph, resp Response, now time.Time) (plan.WorkGraph, error) {
	if resp.Plan != nil {
		return plan.NormalizeWorkGraphTimestamps(*resp.Plan, now), nil
	}
	if len(resp.Patch) == 0 {
		return plan.WorkGraph{}, errors.New("agent response contained no plan or patch")
	}
	next := plan.Clone(base)
	if err := ApplyPatch(&next, resp.Patch, now); err != nil {
		return plan.WorkGraph{}, err
	}
	return next, nil
}
