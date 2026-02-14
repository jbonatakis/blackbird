# Parent as Code Reviewer / Quality Gate
status: complete

## Purpose

When all children of a parent task are done, the parent runs as a reviewer (not an implementer) and evaluates parent acceptance criteria against child outputs. If review fails, Blackbird pauses normal execute flow, records parent feedback for targeted children, and requires an explicit user resume action per child (or all children in TUI).

This behavior is separate from normal ready-task execution: parent tasks are never in the ready leaf queue, and parent "execution" is a distinct review run (`run_type: "review"`).

## Shipped Behavior

### Trigger and idempotence

- Parent-review candidates are discovered from ancestors of the completed child task.
- A parent is eligible only when it has non-empty `childIds` and all listed children are `done`.
- Review idempotence is signature-based. The gate computes a deterministic completion signature from parent ID + each child `updatedAt` timestamp and only runs a review when that signature differs from the latest persisted review run.
- No parent review is triggered on plan load alone.

### Parent review run semantics

- Review runs are persisted as `RunRecord{Type: "review"}` for the parent task ID.
- Review context includes parent acceptance criteria plus child run summaries/artifact refs and reviewer-only instructions.
- Review response is strict JSON with:
  - `passed` (required boolean),
  - `resumeTaskIds` (required and non-empty when `passed=false`; must be unique child IDs of that parent),
  - `feedbackForResume` (required non-empty when `passed=false`; must be empty when `passed=true`).
- Response normalization trims whitespace and sorts `resumeTaskIds` for deterministic persistence and UX.

### Pause-on-fail semantics

- If review passes, execute continues to the next ready leaf task.
- If review fails with one or more `resumeTaskIds`, execute stops with `ExecuteReasonParentReviewRequired` (`"parent_review_required"`).
- This pause is explicit and blocks remaining ready leaf tasks until the user resumes children manually.
- CLI output includes normalized resume task IDs and per-task `blackbird resume <taskId>` next steps.
- TUI opens a dedicated parent-review failure modal with resume actions.

### Pending feedback persistence

- On failing review outcomes, Blackbird upserts pending feedback records per child under `.blackbird/parent-review-feedback/<childTaskId>.json`.
- Each record links `parentTaskId`, `reviewRunId`, and `feedback`.
- Review run persistence occurs before writing pending-feedback links.

### Manual child resume flow

- `blackbird resume <taskId>` and TUI resume actions consume pending parent-review feedback when present.
- Feedback source precedence is deterministic:
  - explicit resume feedback input,
  - pending parent-review feedback,
  - no feedback (waiting-user question flow).
- When pending parent feedback is used:
  - waiting-user question prompts are skipped,
  - resume requires provider/session continuity from the latest run,
  - resumed context gets a `parentReviewFeedback` section with `{parentTaskId, reviewRunId, feedback}`,
  - a new resumed run record is persisted,
  - pending feedback is cleared only after successful run persistence.
- Auto-resume is not implemented. Resume remains user-initiated.

### Relation to ready-task execution

- `ReadyTasks` selects only leaf tasks (`childIds` empty), `status=todo`, and satisfied deps.
- Parent tasks are excluded from normal execute dispatch.
- Parent review runs are a separate post-child-completion gate.

## Non-Goals

- Automatic child resume after review failure.
- Parent status derivation policy changes beyond existing status updates.
- Running parent review when any child is not done.

## Definitions

- **Parent task**: a plan item with non-empty `childIds`.
- **Leaf task**: a plan item with empty `childIds`; only leaf tasks are ready for normal implementation execution.
- **Review run**: a run record with `run_type: "review"` tied to a parent task.
- **Pending parent-review feedback**: per-child persisted linkage written after a failed review response.

## Done Criteria

- Parent review triggers after child completion only for eligible ancestor parents and runs at most once per completion signature.
- Review runs persist `passed/resumeTaskIds/feedback/completion signature` fields on parent run records.
- Failing reviews with resume targets pause execute (`parent_review_required`) and require explicit user resume actions.
- Resume for flagged children consumes pending parent feedback, injects `parentReviewFeedback` into resume context, and clears pending feedback only after resumed run persistence.
- Parent tasks remain excluded from normal ready-task execution; parent review remains a separate run flow.
