TITLE
Phase 1 Spec — Agent-Assisted Project Plan Builder (Tasks/Subtasks + Prompts + Auto-Dependencies) + “Ready Task” View

OBJECTIVE
Deliver a CLI that, with assistance from an AI agent runtime, produces a durable project plan:
- a hierarchical task/subtask tree,
- fleshed-out descriptions and canonical prompts for every node,
- a best-effort inferred dependency DAG across nodes,
- validation + explainability (“why blocked?”),
- and a usable “ready task” loop (pick/list/show, manual status updates).

Phase 1 includes agent integration specifically for PLAN CREATION/REFINEMENT and DEPENDENCY INFERENCE. Phase 1 does NOT include agent-driven code execution for implementing tasks, nor a full real-time dashboard.

NON-GOALS (OUT OF SCOPE)
- Agent execution to implement code changes for tasks (coding runs)
- Automated patch application / git branches / PR creation
- Automated tests/lint execution
- Real-time multi-worker dashboard and parallel task execution
- Background project snapshot summarization (“current state of app”) as a continuously updating daemon
- Rich alerting / audible ding / waiting_user workflow (beyond simple interactive Q&A during planning)

PRIMARY USER STORIES
US-1 Initialize Plan
As a user, I can initialize a new project plan in a directory so it persists on disk.

US-2 Agent-Assisted Plan Generation (from a project description)
As a user, I can provide a high-level project description and the agent generates:
- a structured feature/task tree,
- descriptions and acceptance criteria,
- a canonical prompt for each node,
- and a best-effort inferred dependency graph.

US-3 Interactive Refinement with Agent
As a user, I can iteratively refine the plan with the agent:
- add/remove tasks,
- rewrite prompts/descriptions,
- restructure hierarchy,
- and adjust dependencies.

US-4 Dependency Auto-Inference + Explanation
As a user, I can ask the agent to infer dependencies (or re-infer after edits) and receive:
- computed deps for each node,
- plus a human-readable rationale for edges (at least in summary).

US-5 Manual Override
As a user, I can manually edit tasks and dependencies at any time via CLI commands; agent output never locks me in.

US-6 Validate Plan
As a user, I can validate the plan (no cycles, deps exist) and see actionable errors.

US-7 Ready/Blocked Views
As a user, I can list “ready” tasks and see blocked tasks with clear reasons (unmet deps, etc.).

US-8 Pick Next Task (planning-to-execution handoff)
As a user, I can select a ready task and view its canonical prompt and details, then mark status (in_progress/done/blocked) manually.

USER EXPERIENCE PRINCIPLES
- The plan file is the source of truth and is human-readable.
- The agent produces a best-effort plan; the user can override anything.
- The CLI should be usable even if the agent is unavailable (manual mode).
- Agent interactions must be bounded and structured: outputs are machine-parseable and validated before being written.

DATA MODEL REQUIREMENTS

WorkGraph
- Represents:
  - Hierarchy: tree/forest (parent-child)
  - Dependencies: DAG edges across any nodes
- Stable IDs: must remain stable across edits and agent regenerations.

WorkItem (node) fields (minimum)
- id: string (stable, unique)
- title: string
- description: string (may be empty)
- acceptanceCriteria: string[] (optional but strongly preferred; can be empty)
- prompt: string (canonical agent prompt for this work item; may be empty but field must exist)
- parentId: string | null
- childIds: string[] (or derivable)
- deps: string[] (IDs of prerequisite nodes)
- status: one of:
  - todo
  - in_progress
  - blocked
  - done
  - skipped
- createdAt: timestamp
- updatedAt: timestamp
- notes: string (optional; for rationale, scoping notes)
- depRationale: Record<depId, string> (optional but recommended; brief reason why dependency exists)

Derived Concepts
- READY if:
  - status in {todo, blocked} (see note below)
  - AND all deps have status == done
- BLOCKED if:
  - any deps are not done (dependency-blocked)
  - OR status == blocked (manual blocked)
NOTE: If status==blocked but deps are now satisfied, the CLI must still be able to show “deps satisfied” and indicate whether the user must manually clear blocked to proceed (choose one behavior and document it).

Plan Persistence
- Store in a single file in project root (human-readable; JSON/YAML/etc.).
- Schema validation on load.
- Must preserve stable IDs.
- Must be safe against failed writes (atomic write or rollback semantics).

AGENT INTEGRATION REQUIREMENTS (PLANNING ONLY)

Conceptual Contract
The CLI integrates with an agent runtime that can:
- receive a structured request (project goal + constraints + existing plan),
- produce a structured response that describes plan mutations and/or a full plan proposal,
- ask clarifying questions when necessary.

Agent Capabilities in Phase 1
A) Generate a plan from scratch:
- Inputs: project description, optional constraints, optional desired granularity.
- Output: feature tree + tasks + prompts + best-effort deps.

B) Refine an existing plan:
- Inputs: current plan + user change request (e.g., “add auth, split backend/frontend, add tests tasks”).
- Output: proposed edits (add/update/delete/move nodes) and updated deps.

C) Infer/re-infer dependencies:
- Inputs: current plan (nodes with titles/descriptions/prompts) + optional “dependency style” hints (e.g., prefer fewer edges).
- Output:
  - deps for each node (or a patch),
  - plus per-edge rationale (at least brief).

Agent Runtime (Claude Code + Codex)
- Phase 1 agent integration targets Claude Code and Codex specifically.
- Provider selection:
  - `BLACKBIRD_AGENT_PROVIDER=claude|codex`
  - Optional override: `BLACKBIRD_AGENT_CMD` to supply a custom command.
- Output format:
  - Must include exactly one JSON object.
  - JSON may be the full stdout or inside a single fenced ```json block.
  - Any other output is allowed but will be ignored for parsing.
- Request metadata fields are passed to the runtime adapter:
  - `provider`, `model`, `maxTokens`, `temperature`, `responseFormat`

Agent Output MUST be Machine-Readable
The agent must output either:
1) A complete plan object conforming to schema, OR
2) A patch-style set of operations:
   - add
   - update
   - delete
   - move (parent change)
   - set_deps (replace)
   - add_dep / remove_dep
Each operation includes enough info to validate and apply deterministically.

Clarifying Questions During Planning
During plan generation/refinement, the agent may ask a small number of clarifying questions. Phase 1 only needs a basic interactive Q&A flow (stdin/stdout) to answer these and continue. This is not the full alerting/waiting_user system.

AGENT “BEST EFFORT” DEPENDENCY INFERENCE SPEC

Expected behavior
- Infer prerequisites based on:
  - logical build order (foundation before integration),
  - shared interfaces/contracts (define API before implementing clients),
  - environment/setup before dependent work,
  - cross-cutting concerns (auth, schemas, data models),
  - integration tasks depend on underlying modules.

Constraints
- Prefer a minimal, sufficient dependency set (avoid overly dense graphs).
- Dependencies should be between the smallest practical units (often leaf tasks).
- Must not create cycles; if inference suggests a cycle, the agent must:
  - choose an alternative ordering,
  - or flag the conflict explicitly for user input.

Rationale
- Provide at least brief reasons for dependencies, either:
  - per edge (preferred), or
  - per node summary (“depends on X because…”).

COMMANDS / INTERACTIONS

1) init
- Creates plan file if none exists.

2) plan generate
- Starts an interactive flow:
  - user provides project description (and optionally constraints)
  - agent returns a proposed plan (tree + deps + prompts)
  - CLI validates, shows summary (counts + top-level features), and writes to disk
- Must allow user to accept or request revisions (at least one iteration).

3) plan refine
- User provides a change request in natural language.
- Agent returns patch operations or full plan.
- CLI validates and applies.
- Show summary of changes:
  - nodes added/removed/modified/moved
  - deps added/removed

4) deps infer
- Agent analyzes existing plan and proposes dependency updates.
- CLI validates (no cycles) and applies if accepted.
- Show diff summary + rationale excerpt.

5) add / edit / delete / move (manual mode)
- Manual commands to modify nodes and deps.

6) list
- Default shows READY leaf tasks.
- Flags:
  - --all
  - --blocked
  - --tree (hierarchy)
  - --features (top-level only)
  - --status <status>

7) show <id>
- Full details including:
  - deps + their statuses
  - dependents (reverse deps)
  - readiness/blocked reason(s)
  - prompt

8) pick
- Interactive selection of READY tasks (leaf tasks by default; can include non-leaf if configured).
- After selection, show details and offer actions:
  - set-status in_progress
  - set-status done
  - set-status blocked
  - back/exit

9) validate
- Validates:
  - dependency edges refer to existing nodes
  - no cycles in deps
  - hierarchy has no cycles
  - required fields present

OUTPUT REQUIREMENTS
- Summaries must be readable and diff-oriented:
  - “Added 8 tasks, updated 3 prompts, added 12 deps…”
- list output must include:
  - id (short ok), title, status, readiness indicator
  - for blocked: unmet deps list (short)

ERROR HANDLING REQUIREMENTS
- If agent output is not parseable or fails schema validation:
  - do not write plan
  - display validation errors and allow retry
- If deps inference introduces a cycle:
  - reject changes and show the cycle path if possible
- If manual dep add introduces a cycle:
  - reject and do not change plan

ACCEPTANCE TESTS (BEHAVIORAL)
AT-1 Generate Plan
Given a short project description, the CLI invokes agent and persists a valid plan with:
- >= 1 top-level feature
- >= 5 leaf tasks
- prompts present for all nodes (can be empty, but field exists)
- at least some deps inferred (unless agent explicitly states none needed)

AT-2 Dependency Readiness
Given tasks A and B with B depending on A:
- list shows only A as ready (initially)
- marking A done makes B ready immediately

AT-3 Re-Infer Deps
After manual edits (add a new integration task), deps infer:
- proposes deps for the new task
- applies without cycles
- show explains why task is blocked until deps done

AT-4 Invalid Agent Output
If agent returns invalid schema:
- CLI rejects, prints actionable error, plan file unchanged

AT-5 Cycle Prevention
If agent proposes deps that create a cycle:
- CLI rejects and reports cycle; plan unchanged

DELIVERABLES
- CLI implementing commands above
- Plan file schema + validation
- Agent planning integration with structured outputs (full plan or patch ops)
- README documenting:
  - planning workflow (generate/refine/deps infer)
  - readiness rules
  - how to manually override plan

END OF SPEC
