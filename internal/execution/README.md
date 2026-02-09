# Execution Architecture

This package owns execution primitives used by both CLI and TUI.

Core responsibilities:
- **Task selection** (`ReadyTasks`): only leaf `todo` tasks with satisfied deps are executable.
- **Context building** (`BuildContext`, `BuildParentReviewContext`): assembles task/review context and snapshot.
- **Run records** (`RunRecord` + `SaveRun`/`ListRuns`/`LoadRun`/`GetLatestRun`): persisted under `.blackbird/runs/<taskID>/<runID>.json`.
- **Agent launch/resume** (`LaunchAgentWithStream`, `ResumeWithAnswer`, `ResumeWithFeedback`).
- **Plan lifecycle** (`UpdateTaskStatus`): status transition + atomic plan save.
- **Parent-review gate orchestration** (`RunParentReviewGate`, `RunParentReview`, pending feedback storage).

## Flow Overview

1. `blackbird execute` loads the plan and selects ready tasks via `ReadyTasks`.
2. Each task is marked `in_progress`, context is built, and the agent is launched.
3. `RunRecord` is written to disk and plan status is updated to `done`, `failed`, or `waiting_user`.
   If execution review checkpoints are enabled, the run record is marked with a pending decision
   and execution halts until a decision is resolved.
4. After each successful child task, execute runs parent-review gate checks for eligible ancestor parents.
5. `blackbird resume` resumes from either waiting-user answers or pending parent-review feedback.

## Parent Review Gate

- Trigger source: successful child completion inside `RunExecute`.
- Candidate discovery: ancestor parents of the completed child where all `childIds` are `done`.
- Idempotence: completion signature (`parent_review_completion_signature`) derived from parent ID + child completion timestamps; review only runs when signature changes.
- Pause behavior: if a review run returns `passed=false` with non-empty `resumeTaskIds`, execute returns stop reason `parent_review_required`.

`RunParentReview` persists a review run (`run_type: "review"`) and stores parsed review outcome fields on the run record:
- `parent_review_passed`
- `parent_review_resume_task_ids`
- `parent_review_feedback`
- `parent_review_completion_signature`

## Pending Parent Feedback Store

Failed review outcomes write per-child pending feedback records:
- Path: `.blackbird/parent-review-feedback/<childTaskID>.json`
- Type: `PendingParentReviewFeedback`
- Fields:
  - `parentTaskId`
  - `reviewRunId`
  - `feedback`
  - `createdAt`
  - `updatedAt`

Integration points:
- Write/upsert: `UpsertPendingParentReviewFeedback(...)`
- Read: `LoadPendingParentReviewFeedback(...)`
- Clear on consume: `ClearPendingParentReviewFeedback(...)`

## Resume Integration (Pending Feedback Consumption)

`RunResume` resolves feedback source with deterministic precedence:
1. explicit feedback argument
2. pending parent-review feedback
3. none (`waiting_user` question flow)

For pending parent-review feedback:
- answer prompts are skipped,
- latest run provider + session reference are validated for resume support,
- resumed context is merged with `context.parentReviewFeedback` (`parentTaskId`, `reviewRunId`, `feedback`),
- resumed run is persisted,
- pending feedback is cleared only after successful run persistence.

This keeps parent-review feedback durable across restarts and guarantees it is consumed exactly by an explicit resume action.

## Execution Output

Agents are expected to modify the working tree directly (native CLI behavior).
Blackbird records stdout/stderr and exit codes for auditing.

Execution context includes a system prompt that authorizes non-destructive
commands and file edits without confirmation.

## Auto-Approve Flags

When launching headless runs, Blackbird appends provider-specific auto-approve
flags (unless `BLACKBIRD_AGENT_CMD` is set, which bypasses this logic):

- Codex: `exec --full-auto`
- Claude: `--permission-mode bypassPermissions`

## Streaming Output

If `BLACKBIRD_AGENT_STREAM=1` is set, stdout/stderr are streamed live while still being captured
in the run record.
