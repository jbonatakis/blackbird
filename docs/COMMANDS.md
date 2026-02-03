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

## Memory (Codex only)

Memory commands operate on durable artifacts captured via the Codex memory proxy.

- `blackbird mem search [--session <id>] [--task <id>] [--run <id>] [--type outcome,decision,constraint,open_thread,transcript] [--limit N] [--offset N] [--snippet-max N] [--snippet-tokens N] <query>` — Search artifacts and print a summary table.
- `blackbird mem get <artifact_id>` — Print a single artifact as JSON.
- `blackbird mem context --task <id> [--session <id>] [--goal "..."] [--budget N]` — Render a memory context pack for a task using stored artifacts and configured budgets.
