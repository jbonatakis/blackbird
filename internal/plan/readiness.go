package plan

// ReadinessLabel returns a display label for readiness derived from status and deps.
func ReadinessLabel(status Status, depsOK bool, manualBlocked bool) string {
	if status == StatusDone {
		return "DONE"
	}
	if status == StatusSkipped {
		return "SKIPPED"
	}
	if status == StatusInProgress {
		return "IN_PROGRESS"
	}
	if status == StatusQueued {
		return "QUEUED"
	}
	if status == StatusWaitingUser {
		return "WAITING_USER"
	}
	if status == StatusFailed {
		return "FAILED"
	}
	if !depsOK {
		return "BLOCKED"
	}
	if manualBlocked {
		return "BLOCKED"
	}
	if status == StatusTodo {
		return "READY"
	}
	return ""
}
