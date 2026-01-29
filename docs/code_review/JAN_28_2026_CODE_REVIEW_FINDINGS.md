## Code Review Findings

### High
- Agent validation and TUI JSON schema reject valid plan statuses (`queued`, `waiting_user`, `failed`). This can break plan refine/patch flows after execution updates task statuses, because agent responses containing those statuses are considered invalid even though the plan allows them.

### Medium
- `LaunchAgent` sets `waiting_user` when questions are present even if the agent command failed (non-zero exit or parse errors). This can mask failures and leave tasks stuck without surfacing the underlying error.
- `ParseQuestions` does not enforce non-empty question IDs, but resume logic uses IDs as map keys. Empty or duplicate IDs can cause missing answers or overwrites.

### Low
- `BLACKBIRD_AGENT_STREAM` parsing is inconsistent: `LaunchAgent` only enables streaming on `"1"` while the agent runtime also accepts `"true"/"yes"/"y"`. This can surprise users depending on the execution path.

### Testing Gaps
- No tests cover agent response validation against the full plan status set (including `queued`, `waiting_user`, `failed`).
- No tests cover `LaunchAgent` when the command fails but question JSON is present.
- No tests cover `ParseQuestions` rejecting empty or duplicate question IDs.
