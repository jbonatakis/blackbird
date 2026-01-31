package plan

import "time"

// NormalizeWorkGraphTimestamps returns a copy of the plan with all item timestamps set to now.
func NormalizeWorkGraphTimestamps(g WorkGraph, now time.Time) WorkGraph {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	normalized := Clone(g)
	if normalized.Items == nil {
		return normalized
	}

	for id, it := range normalized.Items {
		it.CreatedAt = now
		it.UpdatedAt = now
		normalized.Items[id] = it
	}

	return normalized
}
