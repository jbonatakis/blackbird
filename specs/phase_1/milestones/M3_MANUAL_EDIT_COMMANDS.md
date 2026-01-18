TITLE
Phase 1 — Milestone M3: Manual Edit Commands (CRUD + Hierarchy + Dep Editing)

OBJECTIVE
Make the CLI fully usable without an agent by enabling manual creation and maintenance of the work graph: nodes, hierarchy, and dependencies.

USER STORIES / ACCEPTANCE TESTS COVERED
- US-5 Manual Override
- US-1 Initialize Plan (complements)
- US-6 Validate Plan (manual edits must preserve validity)

SCOPE
- Node CRUD:
  - add (create new WorkItem with stable ID)
  - edit (update title/description/acceptanceCriteria/prompt/notes)
  - delete (remove node; define semantics for children: reject unless `--cascade` or reparent)
  - move (change parent; update parent/child links)
- Dependency editing:
  - deps add/remove/set
  - must reject changes that introduce a dependency cycle
- Ensure timestamps update on any mutation (`updatedAt`)
- Ensure ID stability and uniqueness (IDs are never auto-regenerated)

NON-GOALS (M3)
- Agent-driven edits (M5/M6).
- Interactive selection/picker (M4).

CLI SURFACE (M3)
- `blackbird add` (interactive or flags; create node)
- `blackbird edit <id>` (interactive or flags)
- `blackbird delete <id>` (with clear safety semantics)
- `blackbird move <id> --parent <parentId|root> --index <n?>`
- `blackbird deps add <id> <depId>`
- `blackbird deps remove <id> <depId>`
- `blackbird deps set <id> <depId...>`

VALIDATION & SAFETY REQUIREMENTS (M3)
- Any mutation must be validated before commit to disk.
- Reject dep edits that create cycles; show a cycle path.
- Reject moves that create hierarchy cycles.
- Delete behavior must be explicit:
  - default: refuse delete if node has children or dependents
  - optional flags: `--cascade-children`, `--force` (still safe and explicit)

DELIVERABLES
- Manual CRUD and dependency commands
- Clear error messaging for invalid edits
- Updated validation to cover new invariants and reference integrity

DONE CRITERIA
- A user can build a non-trivial plan entirely via CLI commands.
- Invalid dep addition that would create a cycle is rejected without modifying the plan file.

