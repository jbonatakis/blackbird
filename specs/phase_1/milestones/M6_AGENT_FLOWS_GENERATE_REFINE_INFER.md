TITLE
Phase 1 — Milestone M6: Agent-Backed Flows (Generate / Refine / Deps Infer) + Summaries

OBJECTIVE
Ship the end-to-end planning workflow using the agent adapter: generate a plan, iteratively refine it, infer dependencies, and persist validated results with readable summaries.

USER STORIES / ACCEPTANCE TESTS COVERED
- US-2 Agent-Assisted Plan Generation
- US-3 Interactive Refinement with Agent
- US-4 Dependency Auto-Inference + Explanation
- US-6 Validate Plan (agent outputs must pass validation before write)
- AT-1 Generate Plan
- AT-3 Re-Infer Deps

SCOPE
- Implement interactive command flows:
  - `blackbird plan generate`
    - collect project description (+ optional constraints)
    - invoke agent
    - validate response
    - show summary (counts + top-level features)
    - allow at least one revision iteration (accept / revise)
    - write plan on accept
  - `blackbird plan refine`
    - read natural-language change request
    - invoke agent (patch or full plan)
    - validate + apply
    - show diff summary (added/removed/updated/moved + deps delta)
  - `blackbird deps infer`
    - invoke agent for dependency inference + rationales
    - validate no cycles
    - apply on accept
    - show diff summary + rationale excerpt
- Implement minimal clarifying Q&A loop:
  - if agent returns questions, prompt user and re-invoke with answers (bounded)

NON-GOALS (M6)
- Real-time dashboard, waiting_user alerts, multi-worker runs (out of Phase 1).
- Automated patch application to code / git workflows (out of Phase 1).

OUTPUT REQUIREMENTS (M6)
- Summaries must be diff-oriented and readable:
  - “Added N tasks, updated M prompts, added K deps…”
- For deps infer, include rationale excerpt:
  - per-edge preferred; otherwise per-node summaries.

ERROR HANDLING REQUIREMENTS (M6)
- If agent output is invalid schema: reject; plan unchanged (AT-4 from M5).
- If agent proposes a dep cycle: reject and show cycle path; plan unchanged (AT-5).

DELIVERABLES
- `plan generate`, `plan refine`, `deps infer` commands
- diff summary rendering (node + dep changes)
- documented readiness rules + manual override guidance (README or spec addendum)

DONE CRITERIA
- AT-1 passes: generate persists a valid plan with required minimums.
- AT-3 passes: after manual edits, deps infer proposes deps for new integration task and applies without cycles.

