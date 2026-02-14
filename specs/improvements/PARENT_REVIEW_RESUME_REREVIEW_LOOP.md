# Parent Review Re-Review Loop on Resume
status: proposed

## Purpose

When a parent review fails and a child task is resumed, Blackbird should re-run parent review after the resumed child completes successfully. This loop should continue until parent review passes or the user explicitly chooses to ignore a failing review and move on.

## Problem Statement

Current behavior runs parent review after child completion in execute/decision flows, but not after `blackbird resume` success. This creates a gap in quality-gate enforcement:

- parent review fails and writes pending parent-review feedback,
- user resumes a targeted child task,
- child resume succeeds,
- no new parent review runs,
- workflow continues without re-checking parent acceptance criteria.

This diverges from the expected quality-gate behavior in `specs/improvements/PARENT_REVIEW_QUALITY_GATE.md`.

## Goals

- Enforce parent review after successful resumed child completion when parent review is enabled.
- Preserve existing parent-review idempotence behavior (completion signature based).
- Surface review-failure pause semantics to both CLI and TUI resume flows.
- Support repeated fail -> resume -> re-review cycles without manual internal intervention.

## Non-Goals

- Automatic child re-resume after a failing parent review.
- Changing parent/leaf readiness rules.
- Changing parent-review response schema/validation rules.

## Proposed Behavior

### Resume-Triggered Parent Review Gate

- After `RunResume` completes with resumed run status `success`, and `ParentReviewEnabled=true`, invoke the same parent-review gate orchestration used by execute for the resumed child task.
- If no parent review candidate is eligible or idempotence says no run is needed, resume completes normally.
- If a parent review runs and passes, resume completes normally.
- If a parent review runs and fails with resume targets, resume stops in `parent_review_required` state and returns the review run context.

### Resume Result Contract

- Resume call sites need structured stop reasons (not only a resumed run record) so parent-review-required can be surfaced.
- Introduce a structured resume outcome contract that can represent:
  - completed,
  - waiting_user,
  - parent_review_required,
  - canceled,
  - error.
- The contract must carry the review run context when the stop reason is `parent_review_required`.

### CLI Behavior (`blackbird resume <taskId>`)

- Load resolved config and honor `execution.parentReviewEnabled`.
- On resume-triggered parent review failure, render the existing parent-review-required summary and next-step commands.
- On resume-triggered parent review pass (or no gate run), keep current completion output behavior.

### TUI Behavior (Parent Review Modal Resume Actions)

- Resume actions from parent-review modal must consume structured resume outcomes.
- If resume returns `parent_review_required`, enqueue/open parent-review modal again with the latest review run.
- This enables repeated review loops directly in modal flow.
- `Continue` or `Quit` behavior remains the explicit user escape hatch for ignoring a failing review and moving on.

### Bulk Resume in TUI (`Resume all failed`)

- Process targets sequentially.
- If any resumed task triggers `parent_review_required`, halt remaining targets in that bulk action and show the new failing review modal immediately.
- Remaining targets are not auto-resumed in that action once the gate blocks.

### Persistence and Idempotence Guarantees

- Keep existing order guarantees:
  - resume run persists before pending feedback clear,
  - review run persists before pending feedback upsert.
- Reuse completion-signature idempotence so repeated checks do not duplicate reviews for unchanged child completion state.

## Implementation Outline

1. Execution layer:
   - add resume flow orchestration that can invoke parent-review gate post-resume success.
   - add structured resume result/stop-reason model for caller consumption.
2. CLI layer:
   - update `runResume` to use resolved config and structured resume result handling.
3. TUI layer:
   - update resume action wrappers and `ExecuteActionComplete` handling for resume-triggered `parent_review_required`.
   - enforce bulk-resume short-circuit on first gate pause.
4. Tests:
   - execution regression coverage for fail -> resume -> re-review loop behavior.
   - CLI resume coverage for resume-triggered parent review fail/pass outputs.
   - TUI modal resume coverage for repeated modal reopening and bulk-resume short-circuit.

## Done Criteria

- A failing parent review followed by successful child resume triggers a new parent review automatically when enabled.
- If that review fails, resume flow pauses again with explicit parent-review-required outcome in CLI and TUI.
- TUI allows repeated resume/review cycles without leaving the modal workflow.
- Users can explicitly ignore failing review and continue manually via existing modal controls.
- Idempotence and pending-feedback persistence invariants remain intact.
