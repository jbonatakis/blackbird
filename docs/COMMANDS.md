# Core commands

## Plan and graph management

- `blackbird plan generate` — Generate a plan from a project description.
- `blackbird plan refine` — Apply agent-proposed edits to the current plan.
- `blackbird deps infer` — Propose dependency updates with rationale.
- `blackbird validate` — Check plan integrity and dependency consistency.
- `blackbird show <id>` — Print task details and readiness explanations.
- `blackbird set-status <id> <status>` — Update task status manually.

## Plan quality gate (`blackbird plan generate`)

`blackbird plan generate` runs deterministic plan-quality lint before save.

- It prints `Quality summary (initial)` and `Quality summary (final)` counts (`blocking`, `warning`, `total`).
- If blocking findings exist and auto-refine is enabled, it runs bounded auto-refine passes and prints progress as `quality auto-refine pass X/Y`.
- Final blocking and warning findings are shown in deterministic order when present.
- If blocking findings remain after bounded auto-refine, save is not implicit and you must choose one action:
  - `revise` (manual revision request; quality gate reruns on the revised plan),
  - `accept_anyway` (save with an override warning),
  - `cancel` (abort without writing a plan).

Auto-refine pass count is controlled by `planning.maxPlanAutoRefinePasses` (default `1`, bounds `0`..`3`, `0` disables auto-refine).

## Manual graph edits

- `blackbird add --title "..." [--parent <parentId|root>]`
- `blackbird edit <id> --title "..." --description "..." --prompt "..."`
- `blackbird move <id> --parent <parentId|root> [--index <n>]`
- `blackbird delete <id> [--cascade-children] [--force]`
- `blackbird deps add <id> <depId>`
- `blackbird deps remove <id> <depId>`
- `blackbird deps set <id> [<depId> ...]`

## Execution

- `blackbird execute` — Run ready tasks in dependency order.
- `blackbird runs <taskID>` — List runs for a task (`--verbose` shows logs).
- `blackbird resume <taskID>` — Resume a task from either pending parent-review feedback or `waiting_user` questions.
- `blackbird retry <taskID>` — Reset failed tasks with failed runs back to `todo`.

Execute and resume share the execution runner in `internal/execution`. The CLI and TUI call the same runner API; the TUI runs execute/resume in-process (no subprocess) and cancels the shared context on quit so any in-flight run stops promptly.

**Review Checkpoints**
When `execution.stopAfterEachTask` is `true`, `blackbird execute` pauses after each task reaches a terminal state and shows a review prompt. The prompt includes task metadata, run status, and a review summary (changed files, diffstat, optional snippets).

Actions:
- Approve and continue: record approval and continue to the next ready task.
- Approve and quit: record approval and exit execution.
- Request changes: open a multi-line change request (blank line submits, `/cancel` returns to the menu, `@` opens the file picker) and resume the same agent session for that task.
- Reject changes: mark the task failed and stop execution.

If stdin is not a TTY, the review prompt falls back to line mode where you can type the option number or label.

**Parent Review Quality Gate**
After a child task succeeds, `blackbird execute` may run a parent review for newly-eligible ancestor parents. If the parent review fails with resume targets, execute pauses before running other ready tasks and prints a deterministic summary:

- `running parent review for <parentTaskId>`
- `parent review failed for <parentTaskId>`
- `resume tasks: <sorted child task ids>`
- `feedback: <normalized feedback excerpt>`
- `next step: blackbird resume <childTaskId>` (one line per resume target)

There is no auto-resume in this flow. Continue by running `blackbird resume <taskID>` for the target child task(s).

Limitations and errors:
- `blackbird resume` does not resolve execution review checkpoints. Run `blackbird execute` again (or use the TUI) to handle pending review decisions.
- If pending parent-review feedback exists for the task, `blackbird resume` uses that feedback path and skips waiting-question prompts.
- If no pending parent-review feedback exists, `blackbird resume` requires a `waiting_user` run with agent questions.
- Resume answers cannot be combined with feedback-based resume for the same task.
- `Request changes` depends on provider resume support and a saved session reference; if the provider does not support resume or the run lacks a session ref, the decision will error and execution will stop.
- Review summaries are best-effort; if git status/diff commands fail or time out, the prompt shows an empty summary.
