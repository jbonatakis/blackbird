TITLE
Phase 1 — Milestone M5: Agent Contract (Schema) + External Runtime Adapter

OBJECTIVE
Define a strict, machine-parseable agent request/response contract and implement a pluggable runtime adapter for Claude Code and Codex that can be used for planning tasks.

USER STORIES / ACCEPTANCE TESTS COVERED
- US-2 Agent-Assisted Plan Generation (contract support)
- US-3 Interactive Refinement with Agent (contract support)
- US-4 Dependency Auto-Inference + Explanation (contract support)
- AT-4 Invalid Agent Output

SCOPE
- Define request/response schemas (JSON) for:
  - plan generation
  - plan refinement
  - dependency inference
- Response forms supported:
  - Full plan object (WorkGraph), OR
  - Patch operations (add/update/delete/move/set_deps/add_dep/remove_dep)
- Include structured clarifying questions support:
  - agent may return `questions[]` with `id`, `prompt`, `options?`
  - CLI can return `answers[]` on retry
- Implement external runtime adapter:
  - provider selection via `BLACKBIRD_AGENT_PROVIDER=claude|codex`
  - default provider commands documented (can be overridden)
  - optional override: `BLACKBIRD_AGENT_CMD` for custom command
  - CLI writes request JSON to stdin, reads response JSON from stdout
  - capture stderr for troubleshooting
  - timeouts + bounded retries
- JSON extraction:
  - allow output to contain non-JSON text, but require exactly one JSON object
  - accept either full-stdout JSON or a single fenced ```json block
  - reject multiple JSON objects or missing JSON
- Request metadata for provider-specific control:
  - `provider`, `model`, `maxTokens`, `temperature`, `responseFormat`
  - runtime adapter maps these into Claude Code/Codex CLI flags if supported

NON-GOALS (M5)
- Implement `plan generate/refine/deps infer` command flows (M6).
- Agent-driven code execution (out of Phase 1).

VALIDATION REQUIREMENTS (M5)
- Agent output must be parseable JSON and schema-valid:
  - if invalid: do not write plan; print actionable errors (AT-4)
- If agent response proposes a plan/patch that fails plan validation:
  - reject with details; plan file unchanged
- If JSON extraction fails (no JSON, multiple JSON objects, malformed JSON):
  - reject and show extraction guidance

DELIVERABLES
- `internal/agent/` types + parser/validator
- runtime adapter that can execute an external command and return a structured response
- doc snippet: how to configure `BLACKBIRD_AGENT_PROVIDER`/`BLACKBIRD_AGENT_CMD` and expected I/O

DONE CRITERIA
- When fed malformed JSON from the agent runtime, CLI rejects and leaves plan unchanged (AT-4).
- When fed non-JSON output or JSON inside a fenced block with surrounding text, CLI extracts/rejects correctly (AT-4 extension).
