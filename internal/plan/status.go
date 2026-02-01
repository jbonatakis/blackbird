package plan

// ParseStatus validates and parses a status string.
func ParseStatus(s string) (Status, bool) {
	switch Status(s) {
	case StatusTodo, StatusQueued, StatusInProgress, StatusWaitingUser, StatusBlocked, StatusDone, StatusFailed, StatusSkipped:
		return Status(s), true
	default:
		return "", false
	}
}
