TITLE
Phase 1 — Milestone M4: `pick` Interactive Ready-Task Loop

OBJECTIVE
Provide an interactive terminal loop to select a READY task, view details, and perform manual status transitions (planning-to-execution handoff in Phase 1).

USER STORIES / ACCEPTANCE TESTS COVERED
- US-8 Pick Next Task (planning-to-execution handoff)

SCOPE
- Implement `blackbird pick`:
  - enumerates READY tasks (leaf by default)
  - offers a simple selection UI (numbered list + stdin)
  - shows the selected task details (reuse `show`)
  - offers actions:
    - set-status in_progress
    - set-status done
    - set-status blocked
    - back / exit
- Keep UX usable even in non-TTY (fallback to prompt/print).

NON-GOALS (M4)
- Full-screen TUI dashboard, streaming logs, multi-worker UI (out of Phase 1).
- Automatic execution of prompts (out of Phase 1).

CLI SURFACE (M4)
- `blackbird pick`
  - Optional flags (if needed):
    - `--include-non-leaf`
    - `--all` / `--blocked` (mirrors `list`)

BEHAVIORAL REQUIREMENTS (M4)
- Default selection set matches `list` default (READY leaf tasks).
- If there are zero READY tasks, explain why (e.g., “0 ready; 12 blocked on deps”) and suggest `list --blocked` or `show`.

DELIVERABLES
- `pick` command wired into the readiness logic from M2
- Clear, minimal interactive UX (no heavy deps required)

DONE CRITERIA
- User can pick a ready task and mark it done; subsequent tasks become ready immediately (via readiness logic).

