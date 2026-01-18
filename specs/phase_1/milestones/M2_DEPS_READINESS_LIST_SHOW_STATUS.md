TITLE
Phase 1 — Milestone M2: Dependencies + Readiness + List/Show/Set-Status

OBJECTIVE
Implement dependency DAG validation, readiness computation, and the first “ready task” loop primitives: `list`, `show`, and `set-status`.

USER STORIES / ACCEPTANCE TESTS COVERED
- US-6 Validate Plan (deps cycle prevention)
- US-7 Ready/Blocked Views
- US-8 Pick Next Task (partial: status updates + visibility)
- AT-2 Dependency Readiness
- AT-5 Cycle Prevention (manual + agent-proposed cycles rejected once agent exists)

SCOPE
- Dependency graph utilities:
  - detect cycles and report a cycle path
  - compute reverse-deps for a node (dependents)
- Readiness logic (per `PHASE_1.md`):
  - depsSatisfied if all deps are status==done
  - READY if status in {todo, blocked} AND depsSatisfied
  - BLOCKED reasons must distinguish unmet deps vs manual blocked
- Implement CLI commands:
  - `list` (default: READY leaf tasks)
  - `show <id>` (details + deps + dependents + readiness explanation)
  - `set-status <id> <status>` (update status + timestamps)
- Extend `validate` to include dependency DAG cycle checking (no cycles).

NON-GOALS (M2)
- Manual edit commands (add/edit/delete/move) beyond status changes (M3).
- Interactive picking UX (`pick`) (M4).
- Agent integration (M5+).

CLI SURFACE (M2)
- `blackbird list`
  - Default: READY leaf tasks
  - Flags: `--all`, `--blocked`, `--tree`, `--features`, `--status <status>`
- `blackbird show <id>`
  - Includes:
    - deps + their statuses
    - dependents (reverse deps)
    - readiness/blocked reason(s)
    - prompt
- `blackbird set-status <id> <status>`

BEHAVIORAL REQUIREMENTS (M2)
- If `status==blocked` and deps are satisfied:
  - show “deps satisfied” and indicate that it remains manually blocked until user clears it
- `list --blocked` must show blocked tasks and a short unmet deps summary.

DELIVERABLES
- Dependency DAG + cycle reporting
- Readiness computation and “why blocked?” explainers
- Working `list/show/set-status` commands

DONE CRITERIA
- AT-2 passes: B depends on A; only A ready until A marked done.
- AT-5 passes for manual dep edits once M3 adds dep editing; for now, `validate` rejects cycles in existing plan files.

