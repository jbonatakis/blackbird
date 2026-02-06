# Task Review Checkpoint (Stop After Each Task)
status: complete

## Purpose

Add an optional **task review checkpoint** that stops plan execution after each task completes and requires the user to decide what happens next. This enables incremental review and change requests without running the whole plan.

The default behavior remains unchanged: when the setting is disabled, Blackbird runs through all ready tasks until the plan finishes.

This feature must work for both **Codex** and **Claude Code** sessions and must reuse shared execution/controller code so CLI and TUI behavior stays in sync.

## Goals

- Add a config setting that, when enabled, **pauses plan execution after each task completes** and prompts for a decision.
- Provide three decisions:
  - **Approve** (continue to next task or quit).
  - **Request changes** (resume the same agent session with user feedback).
  - **Reject changes** (mark run as rejected/failed; do not continue automatically).
- Make the “request changes” UI behave like the plan generate modal, including **multi-line input** and **`@` file references**.
- Provide a **clear, prominent prompt** when a decision is required, with a **summary of changes** (files edited, key snippets).
- Keep CLI and TUI code paths **DRY** by routing through a shared execution controller.

Non-goals:

- Automatic code rollback on reject (user can revert manually).
- Changing agent provider semantics beyond existing pause/resume capabilities.

## User Experience

### CLI

When executing with the setting enabled:

- After each task completes, execution stops and prints a prominent prompt:
  - Task ID + title
  - Run status (done/failed)
  - Summary of changes (file list, diff stat, optional snippets)
  - Options:
    - Approve and continue
    - Approve and quit
    - Request changes
    - Reject changes
  These should eb provided as a list of options that can be keyed to using the up and down arrows or j/k. Enter to select.

- `Request changes` opens an interactive input (multi-line) similar to plan prompt editing, with `@` file reference support.
- After decision:
  - **Approve + continue** resumes plan execution for the next ready task.
  - **Approve + quit** exits with state preserved.
  - **Request changes** resumes the same agent session for that task (see Resume section) and re-enters review when the task completes again.
  - **Reject** marks the run as rejected/failed and stops execution (no automatic next task).

### TUI

When the setting is enabled and a task completes:

- Show a **prominent banner** and a **modal** that requires action.
- The modal includes:
  - Task metadata
  - Summary of changes (file list + snippets)
  - Action buttons: Approve (continue), Approve (quit), Request changes, Reject
- `Request changes` uses the same **multi-line modal** and `@` file reference picker used by plan generation/refinement.
- Until a decision is made, the run is in a `waiting_user`-style state and the dashboard shows an “Action Required” indicator.

## Configuration

Add a config key with default `false`:

```json
{
  "schemaVersion": 1,
  "execution": {
    "stopAfterEachTask": false
  }
}
```

- **Precedence**: project config overrides global config, overrides defaults (same as existing config system).
- When `true`, plan execution enters the review checkpoint after each completed task.
- When `false`, execution continues normally with no prompts.

## Execution Semantics

### When the checkpoint triggers

- After a task run transitions to a **terminal state** (`succeeded`, `failed`, `canceled`, or `skipped`), if it was part of a plan execution queue and `stopAfterEachTask` is enabled.
- The system must **not start** the next task until a decision is made.

### Decision outcomes

- **Approve + continue**:
  - Mark decision as approved.
  - Continue plan execution normally (next ready task).
- **Approve + quit**:
  - Mark decision as approved.
  - Stop execution loop and return to CLI/TUI idle state.
- **Request changes**:
  - Resume the same agent session for the task, using the native provider resume feature.
  - The feedback text is injected into the resumed context (see Resume section).
  - On completion, the checkpoint triggers again.
- **Reject**:
  - Mark the run as rejected/failed (new terminal reason).
  - Stop execution loop; no automatic next task.

## Resume With Feedback

When the user requests changes:

- Blackbird uses the **existing resume path** (`blackbird resume <taskId>` or TUI resume action) but supplies additional feedback as the follow-up prompt.
- The run record must store:
  - provider (`codex` / `claude`)
  - provider session ref
  - repo root
  - last completion status
- The follow-up message to the agent includes:
  - The user’s change request
  - Optional contextual snippets from the summary (e.g., files involved)

If native resume is unavailable (missing session ref or provider refuses):

Claude:
```
--resume, -r	Resume a specific session by ID or name, or show an interactive picker to choose a session	claude --resume auth-refactor
```

Codex example:
```
codex exec resume 7f9f9a2e-1b3c-4c7a-9b0e-.... "Implement the plan"
```

- The system must surface a clear error and recommend **restart** (per Pause/Resume spec). No silent fallback.

## Summary of Changes (for review)

At task completion, compute a review summary and attach it to the run record so both CLI and TUI can render it:

- **File list**: changed files (tracked and untracked), preferably via `git status --porcelain` or equivalent repo state capture.
- **Diff stat**: `git diff --stat` (bounded to the task).
- **Snippets**: optional small excerpts from modified files or diffs (size-limited).

Constraints:

- Keep summary bounded (e.g. max N files and max M lines of snippets).
- Do not block completion if summary fails; log and continue with a minimal summary.

## Data Model / State

Add a “decision gate” record associated with a run:

- `decision_required` (bool)
- `decision_state`: `pending | approved_continue | approved_quit | changes_requested | rejected`
- `decision_requested_at`
- `decision_resolved_at`
- `decision_feedback` (for change requests)
- `review_summary` (files, diffstat, snippets)

The run’s lifecycle remains terminal on completion; the decision gate is a **post-run state** used by the plan executor to pause and await user input.

## DRY Execution Architecture

- The existing execution controller should own the checkpoint flow so both CLI and TUI use the same logic.
- UI layers should only:
  - Render the decision prompt
  - Collect input (including `@` references)
  - Pass the decision back to the controller

## Tests

Add tests to cover:

- Config loading: `execution.stopAfterEachTask` merges with global/project precedence and defaults to `false`.
- Execution loop: when enabled, **next task does not start** until a decision is recorded.
- Decision outcomes:
  - Approve + continue resumes plan execution.
  - Approve + quit stops execution.
  - Reject stops execution and marks run as rejected/failed.
  - Request changes invokes resume with the stored session ref and feedback.
- Summary generation: run completion attempts to capture file list/diff stat and persists review summary.
- TUI modal: `Request changes` supports multi-line input and `@` file picker (reuse plan modal behavior).

## Success Criteria

- With `stopAfterEachTask=false`, execution behavior is unchanged.
- With `stopAfterEachTask=true`, execution always pauses after each task and demands a decision.
- The decision prompt is prominent and includes a clear summary of changes.
- Request changes resumes the same session for both Codex and Claude, with feedback injected.
- CLI and TUI share the same execution/controller logic for checkpoint decisions.
- Tests cover the execution gate, config, and resume flow.
- All relevant documentation updates
