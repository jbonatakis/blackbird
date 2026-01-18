TITLE
Phase 1 Implementation Plan — Agent-Assisted Project Plan Builder + Ready Task Loop

GOALS (PHASE 1)
- Durable plan artifact on disk containing a hierarchical tree + dependency DAG with stable IDs.
- Validation + explainability: reject invalid plans/agent outputs; show actionable errors and “why blocked?” reasons.
- Ready-task loop: list ready tasks by default, show details, pick a ready task, and allow manual status updates.
- Agent integration (planning only) for: plan generate, plan refine, deps infer with machine-readable outputs (full plan or patch ops).

ARCHITECTURE (GO + LOW DEPENDENCIES)

1) Repository layout
- cmd/blackbird/: CLI entrypoint
- internal/plan/: data model + load/store + validation + patch application
- internal/graph/: dependency graph utilities (cycle detection, topo sort, reverse deps)
- internal/cli/: command parsing/dispatch, I/O formatting, interactive prompts
- internal/agent/: request/response types + runtime adapter (external command hook)

2) Persistence + schema strategy
- Store in a single JSON file in project root, e.g. blackbird.plan.json.
- Atomic writes: write temp file + fsync + rename.
- Include schemaVersion in root for evolution.

Data model (minimum)
- WorkGraph
  - schemaVersion: int
  - items: map[id]WorkItem
  - (optional) createdAt/updatedAt
- WorkItem
  - id: string (stable)
  - title: string
  - description: string
  - acceptanceCriteria: []string
  - prompt: string
  - parentId: string | null
  - childIds: []string
  - deps: []string
  - status: todo | in_progress | blocked | done | skipped
  - createdAt / updatedAt: timestamp
  - notes: string (optional)
  - depRationale: map[depId]string (optional but recommended)

3) Core algorithms (shared by manual + agent flows)
- Validation:
  - required fields present, IDs unique
  - hierarchy has no cycles; parent/children references consistent
  - dependency edges reference existing nodes
  - dependency graph is acyclic; if a cycle is found, report a cycle path
- Readiness:
  - depsSatisfied(item) if all deps are status==done
  - ready(item) if status in {todo, blocked} AND depsSatisfied(item)
  - show blocked reasons: unmet deps vs manual blocked even if deps satisfied
- Reverse deps: compute dependents for show <id>

4) CLI commands (Phase 1)
A) Manual/local mode first (works without agent)
- init: create empty valid plan file if none exists
- validate: validate plan; print actionable errors
- add/edit/delete/move: CRUD nodes + hierarchy updates
- deps add/remove/set: update deps with cycle prevention
- set-status <id> <status>: update status + timestamps

B) Ready-task navigation loop
- list: default shows READY leaf tasks
  - flags: --all, --blocked, --tree, --features, --status <status>
- show <id>: deps + statuses, dependents, readiness/blocked reasons, prompt
- pick: interactive selection of READY tasks + actions to set status

5) Agent integration (planning only; structured)
- AgentResponse supports either:
  - full plan object (WorkGraph), OR
  - patch operations:
    - add_node, update_node_fields, delete_node, move_node
    - set_deps / add_dep / remove_dep
  - include per-edge rationale where possible (depRationale)

Runtime adapter
- Use an external command hook (e.g. env BLACKBIRD_AGENT_CMD).
- CLI writes request JSON to stdin and reads response JSON from stdout.
- Strict parsing + schema validation before writing plan file.
- Clarifying Q&A: if response includes questions, prompt user and re-invoke with answers (bounded loop).

6) Tests (behavioral + unit)
- Readiness behavior (AT-2)
- Cycle prevention and reporting (AT-5)
- Invalid agent output rejected without writing (AT-4)
- Patch application correctness + ID stability
- Atomic write happy path

MILESTONES (BUILD ORDER)
- M1: go module + plan structs + load/store + atomic write + validate
  - See: `specs/phase_1/milestones/M1_FOUNDATION_PLAN_IO_VALIDATE.md`
- M2: deps DAG + readiness + list/show/set-status
  - See: `specs/phase_1/milestones/M2_DEPS_READINESS_LIST_SHOW_STATUS.md`
- M3: manual edit commands + dep editing
  - See: `specs/phase_1/milestones/M3_MANUAL_EDIT_COMMANDS.md`
- M4: pick interactive loop
  - See: `specs/phase_1/milestones/M4_PICK_INTERACTIVE_LOOP.md`
- M5: agent schema + external command adapter
  - See: `specs/phase_1/milestones/M5_AGENT_SCHEMA_AND_RUNTIME_ADAPTER.md`
- M6: plan generate/refine/deps infer flows + summaries + rationale excerpts
  - See: `specs/phase_1/milestones/M6_AGENT_FLOWS_GENERATE_REFINE_INFER.md`

