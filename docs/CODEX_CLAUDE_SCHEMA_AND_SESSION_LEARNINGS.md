# Claude/Codex Output and Schema Learnings (Blackbird)

## Scope

This captures the practical behavior observed while testing `blackbird`-related CLI usage for structured outputs with Claude and Codex.

## Current Blackbird runtime behavior

- `internal/agent/runtime.go` currently:
  - passes `--json-schema` for Claude when `meta.JSONSchema` is set,
  - does **not** pass `--output-schema` for Codex yet.
- `blackbird` currently does not emit explicit `--output-format json` for Claude.
- `RequestMetadata` already supports `JSONSchema` and `ResponseFormat`, but Codex/Claude wiring is currently in `buildFlagArgs` + provider args only.

## Claude behavior observed

- Successful pattern for strict JSON:
  - use `--json-schema <schema>`
  - add `--output-format json` where available/required.
- In tested sessions, using only `--json-schema` could still yield non-strict text if `--output-format` is not set.
- If output must be machine-consumable, prefer:
  - `--permission-mode bypassPermissions` (Blackbird non-interactive baseline),
  - `--json-schema <schema>` plus `--output-format json` if the installed CLI requires it.
- System prompt guidance is still important for stability, even with schema flags.
- When using `--json`, Claude returns a response envelope; the actual schema result is under `structured_output` (not the top-level root object).
- Example includes fields like:
  - `type`, `subtype`, `duration_ms`, `duration_api_ms`, `session_id`, `usage`, and `structured_output`.
- Parsing for resume + structured payload should read:
  - provider session ID: top-level `session_id`
  - generated structured data: top-level `structured_output`

## Codex behavior observed

- Codex supports `--output-schema`.
- `--json` mode is stream/event output and emits NDJSON events (not a single final object).
- Without `--json`, output is a single JSON body on stdout, which is easier to parse as the final result.
- Practical behavior for one-shot structured call:
  - `--output-schema /tmp/blackbird-plan-schema.json`
  - avoid `--json` for final single-object parsing.
- Schema shape must satisfy Codex strictness.

## Schema incompatibility discovered (Claude default schema → Codex strictness)

Blackbird’s current schema string in `internal/agent/plan_defaults.go` is not directly safe for Codex without adjustment.

- Codex rejected the old shape because it requires strict object constraints.
- Most practical requirement observed: set `additionalProperties: false` on object schemas.
- Nested object schemas also need strictness where possible for robust enforcement.
- Therefore, shared one-shot schema should be validated for both providers before hard adoption.

## Quick Codex schema test we used

- `/tmp/blackbird-plan-schema.json` content:

  `{"type":"object","additionalProperties":false,"properties":{"x":{"type":"string","minLength":1}},"required":["x"]}`

- Non-stream call:

  `printf ... | codex exec --full-auto --skip-git-repo-check --output-schema /tmp/blackbird-plan-schema.json`

  -> returns a single JSON body on stdout.

- Stream mode:

  `... | codex exec --full-auto --skip-git-repo-check --output-schema ... --json`

  -> NDJSON events; parse `item.completed` for `agent_message.text` payload.

## Session-id / resume concerns

- Codex does not expose a pre-seeded session identifier argument equivalent to Claude’s `--session-id`.
- Resume token can be extracted from runtime logs:
  - stderr line: `session id: <uuid>`
  - or event stream: `{"type":"thread.started","thread_id":"..."}`
- For robust resume support in Blackbird, parse these artifacts and persist a normalized session id.
- This is the pragmatic approach until Codex exposes an explicit resume/session option compatible with pre-specified IDs.

## Suggested Blackbird implementation direction

1. Add Codex `--output-schema` support in `buildFlagArgs` using existing schema source.
2. Keep existing Claude `--json-schema` behavior.
3. Consider optional/conditional `--output-format json` for Claude based on CLI version expectations.
4. Standardize on non-stream parsing path for Codex unless stream analytics are needed.
5. Persist resumable session ids from stderr/event stream parsing for both providers in a single normalized field.
