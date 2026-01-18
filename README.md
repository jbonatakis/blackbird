# blackbird

Go-first CLI for maintaining a durable, validated project work plan (Phase 1).

## Quickstart

- Initialize a plan file in the current directory:

  - `blackbird init`

- Validate the current plan file:

  - `blackbird validate`

The plan file lives at repo root as `blackbird.plan.json`.

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

In M1, dependency edges are validated for existence only (dependency cycle detection comes later).

