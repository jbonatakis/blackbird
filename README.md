# blackbird

Go-first CLI for maintaining a durable, validated project work plan (Phase 1).

## Quickstart

- Initialize a plan file in the current directory:

  - `blackbird init`

- List ready work (leaf tasks whose deps are done):

  - `blackbird list`

- Show full details for a task:

  - `blackbird show <id>`

- Manually update a task status:

  - `blackbird set-status <id> <status>`

- Manually edit the work graph (M3):
  - `blackbird add --title "..." [--parent <parentId|root>]`
  - `blackbird edit <id> --title "..." --description "..." --prompt "..."`
  - `blackbird move <id> --parent <parentId|root> [--index <n>]`
  - `blackbird delete <id> [--cascade-children] [--force]`
  - `blackbird deps add <id> <depId>`
  - `blackbird deps remove <id> <depId>`
  - `blackbird deps set <id> [<depId> ...]`

- Validate the current plan file:

  - `blackbird validate`

The plan file lives at repo root as `blackbird.plan.json`.

## Readiness rules (M2)

- A task's deps are **satisfied** when **all deps have status `done`**.
- A task is **actionable ("READY" in `list`)** when:
  - status is `todo`
  - and deps are satisfied
- If a task is `blocked` but deps are satisfied, it remains **manually blocked** until you clear it (e.g. `set-status <id> todo`).

## Plan file schema (M1)

File format is JSON (no YAML dependency). Unknown fields are rejected on load.

Root object:

- `schemaVersion` (int)
- `items` (object/map of `id` → `WorkItem`)

`WorkItem` (minimum fields):

- `id` (string; must match the map key)
- `title` (string)
- `description` (string; may be empty)
- `acceptanceCriteria` ([]string; may be empty but must exist)
- `prompt` (string; may be empty but must exist)
- `parentId` (string|null)
- `childIds` ([]string; may be empty but must exist)
- `deps` ([]string; may be empty but must exist)
- `status` (one of: `todo`, `in_progress`, `blocked`, `done`, `skipped`)
- `createdAt` (RFC3339 timestamp)
- `updatedAt` (RFC3339 timestamp)
- `notes` (string; optional)
- `depRationale` (object/map of `depId` → string; optional)

## Validation (M1)

`blackbird validate` checks:

- required fields present
- IDs are non-empty and consistent (`items[id].id == id`)
- `deps` / `parentId` / `childIds` references exist
- parent/child relationships are consistent
- hierarchy contains no cycles

In M2, dependency cycles are also rejected (deps must form a DAG).

