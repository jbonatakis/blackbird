TITLE
Phase 1 — Milestone M1: Foundation (Go module + Plan IO + Atomic Writes + Validate)

OBJECTIVE
Establish a Go-first CLI skeleton and a durable, human-readable plan file with strict schema validation and safe, atomic persistence.

USER STORIES / ACCEPTANCE TESTS COVERED
- US-1 Initialize Plan
- US-6 Validate Plan
- Plan Persistence requirements

SCOPE
- Create initial Go module + `cmd/blackbird` entrypoint.
- Define `WorkGraph` / `WorkItem` types per `PHASE_1.md` (minimum required fields).
- Implement plan load/store from a single repo-root file (e.g. `blackbird.plan.json`).
- Implement atomic write semantics (temp file + fsync + rename).
- Implement `validate` command:
  - required fields present
  - unique IDs
  - references exist (deps, parent/children)
  - hierarchy has no cycles

NON-GOALS (M1)
- Dependency DAG cycle detection (for deps) beyond basic reference existence (that comes in M2).
- Readiness calculation, list/show/pick UX (M2+).
- Agent integration (M5+).

CLI SURFACE (M1)
- `blackbird init`
  - Creates plan file if none exists.
  - Writes an empty-but-valid plan root object with `schemaVersion`.
- `blackbird validate`
  - Validates current plan file and prints actionable errors.

DATA / SCHEMA DECISIONS (M1)
- File format: JSON (avoid YAML dependency).
- Root object includes:
  - `schemaVersion` (int)
  - `items` (map id → WorkItem)
- WorkItem fields per `PHASE_1.md`, with timestamps stored in RFC3339.

ERROR HANDLING REQUIREMENTS (M1)
- If plan file is missing:
  - `init` creates it
  - other commands fail with a clear message suggesting `init`
- If JSON parse fails or schema invalid:
  - do not write changes
  - print path + error list

DELIVERABLES
- `go.mod` and runnable `blackbird` binary
- `blackbird init` and `blackbird validate`
- Plan file schema documented (briefly) in README (or in the spec if README deferred)

DONE CRITERIA
- Running `blackbird init` creates a plan file.
- Running `blackbird validate` on a fresh plan succeeds.
- Corrupting JSON causes `validate` to fail without modifying the file.

