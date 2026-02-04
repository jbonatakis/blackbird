# Core commands

## Plan and graph management

- `blackbird plan generate` — Generate a plan from a project description.
- `blackbird plan refine` — Apply agent-proposed edits to the current plan.
- `blackbird deps infer` — Propose dependency updates with rationale.
- `blackbird validate` — Check plan integrity and dependency consistency.
- `blackbird show <id>` — Print task details and readiness explanations.
- `blackbird set-status <id> <status>` — Update task status manually.

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
- `blackbird resume <taskID>` — Answer questions and continue a waiting task.
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

Limitations and errors:
- `blackbird resume` only handles `waiting_user` runs that contain agent questions; it does not resolve review checkpoints. Run `blackbird execute` again (or use the TUI) to handle pending review decisions.
- `Request changes` depends on provider resume support and a saved session reference; if the provider does not support resume or the run lacks a session ref, the decision will error and execution will stop.
- Review summaries are best-effort; if git status/diff commands fail or time out, the prompt shows an empty summary.
