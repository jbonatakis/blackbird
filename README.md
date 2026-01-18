# blackbird

Go-first CLI for maintaining a durable, validated project work plan (Phase 1).

## Quickstart

- Initialize a plan file in the current directory:

  - `blackbird init`

- Generate an initial plan with the agent:

  - `blackbird plan generate`

- Refine an existing plan with the agent:

  - `blackbird plan refine`

- Infer or re-infer dependencies with the agent:

  - `blackbird deps infer`

- List ready work (leaf tasks whose deps are done):

  - `blackbird list`

- Show full details for a task:

  - `blackbird show <id>`

- Manually update a task status:

  - `blackbird set-status <id> <status>`

- Pick a ready task and update status interactively (M4):

  - `blackbird pick`
  - `blackbird pick --blocked`
  - `blackbird pick --all`
  - `blackbird pick --include-non-leaf`

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

## Agent-backed planning (M6)

- `blackbird plan generate` prompts for a project description, calls the agent, shows a summary, and lets you revise once before saving.
- `blackbird plan refine` sends a change request + current plan, applies validated edits, and prints a diff summary.
- `blackbird deps infer` proposes dependency updates, shows a diff + rationale excerpt, and applies on acceptance.
- Summaries include the provider/model used for the run.

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

## Agent runtime adapter (M5)

Phase 1 planning uses an external agent command that speaks a strict JSON
request/response contract.

Configuration:

- `BLACKBIRD_AGENT_PROVIDER=claude|codex` selects the default command
  (`claude` or `codex`).
- `BLACKBIRD_AGENT_CMD` overrides the command entirely (runs via `sh -c`).

I/O contract:

- The CLI writes a JSON request to stdin.
- The command must emit exactly one JSON object on stdout.
- Output may include extra non-JSON text, but JSON must either be the full
  stdout or inside a single fenced ```json block.
- Multiple JSON objects or missing JSON cause a hard failure.
- Requests include a default `systemPrompt` to require strict JSON output and
  validate plan/patch rules (plan generate/refine + deps infer).

Provider metadata (optional in the request):

- `provider`, `model`, `maxTokens`, `temperature`, `responseFormat`, `jsonSchema`
- The runtime adapter maps these to CLI flags when supported:
  `--model`, `--max-tokens`, `--temperature`, `--response-format`
- For Claude, `jsonSchema` is passed via `--json-schema`.
