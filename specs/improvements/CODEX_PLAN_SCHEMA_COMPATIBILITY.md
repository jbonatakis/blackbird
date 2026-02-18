# Codex Plan Schema Compatibility and Planning Enforcement
status: proposed

## Purpose

Ensure Blackbird planning calls (`plan generate`, `plan refine`, `deps infer`, and TUI equivalents) use a JSON schema that is accepted by Codex while remaining compatible with Claude, and ensure Codex planning runs actually enforce that schema at invocation time.

## Problem

Current behavior is partially wired:

1. Planning metadata sets `JSONSchema` in CLI/TUI flows.
2. Runtime only maps `JSONSchema` to Claude `--json-schema`.
3. Runtime does not map `JSONSchema` to Codex `--output-schema`.
4. The default plan schema is not strict enough for observed Codex schema validation behavior.

Result: planning runs on Codex can fall back to prompt-only JSON compliance instead of provider-enforced schema output.

## Goals

1. Make the default planning JSON schema strict enough for Codex and still valid for Claude.
2. Ensure Codex planning invocations always enforce schema via `--output-schema`.
3. Keep planning behavior deterministic and compatible across CLI and TUI.
4. Preserve existing post-parse validation in `internal/agent` as defense in depth.

## Non-Goals

1. Streaming/NDJSON parsing support for planning responses.
2. Resume/session-id persistence enhancements.
3. Changing plan quality-gate semantics.
4. Redesigning the plan data model (`WorkGraph`, `WorkItem`) in this change.

## Scope

Planning-only flows:

1. `internal/cli/agent_flows.go`
2. `internal/tui/action_wrappers.go`
3. `internal/plangen/*` request paths
4. `internal/agent/runtime.go` provider flag wiring
5. `internal/agent/plan_defaults.go` schema definition

## Requirements

### 1. Shared strict schema for Codex + Claude

Update `agent.DefaultPlanJSONSchema()` so fixed-shape objects are strict:

1. Add `additionalProperties: false` to fixed-shape objects:
   - top-level response object
   - `definitions.workGraph` (except intentional map fields)
   - `definitions.workItem`
   - `definitions.patchOp`
   - `definitions.question`
2. Keep dictionary/map fields explicitly modeled where needed:
   - `workGraph.items` remains an object map of `workItem` values.
   - `depRationale` remains a string map.
3. Keep schema Draft-07 and provider-neutral (no provider-specific keywords).
4. Keep existing response contract: exactly one of `plan`, `patch`, or `questions`.

Compatibility policy:

1. Schema changes must continue to decode into current `agent.Response` and `plan.WorkGraph` types.
2. Schema changes must not require provider-specific schema forks for planning.

### 2. Codex schema enforcement at runtime

In runtime provider flag construction:

1. Claude path:
   - keep `--json-schema` behavior when `meta.JSONSchema` is present.
2. Codex path:
   - when `meta.JSONSchema` is present, materialize schema to a temp file and pass `--output-schema <tempfile>`.
   - avoid Codex `--json` stream mode in planning path; continue single-object stdout parsing.
3. Ensure temp schema file lifecycle is safe:
   - created per run attempt
   - cleaned up after command completes
   - path quoting safe in shell mode
4. Fail fast with actionable error if schema file creation/write fails.

### 3. Mandatory schema use for planning on Codex

Planning entrypoints must always attach default plan schema unless explicitly overridden by future design:

1. CLI:
   - `plan generate` initial request
   - quality auto-refine passes
   - manual revision passes
   - `plan refine`
   - `deps infer`
2. TUI:
   - `GeneratePlanInMemory`
   - `GeneratePlanInMemoryWithAnswers`
   - `RefinePlanInMemory`
   - `RefinePlanInMemoryWithAnswers`
   - quality auto-refine callback path

Implementation rule:

1. Use one shared helper for planning metadata assembly (schema + provider assignment) to avoid drift between CLI/TUI flows.
2. Codex planning runs without schema enforcement are considered invalid behavior.

### 4. Parsing and validation contract

No behavior change to response decoding pipeline:

1. Continue extracting one JSON object from stdout.
2. Continue decoding via `DecodeResponse`.
3. Continue semantic validation via `ValidateResponse`.

If provider output format does not match this contract, return explicit runtime error rather than silently accepting partial output.

## Implementation Plan

1. Harden `DefaultPlanJSONSchema` strictness in `internal/agent/plan_defaults.go`.
2. Extend provider schema flag handling in `internal/agent/runtime.go` to support Codex `--output-schema`.
3. Add/route a shared planning metadata helper used by CLI and TUI planning flows.
4. Wire all planning request sites to shared helper.
5. Keep existing extraction/validation pipeline unchanged.

## Testing Requirements

### Unit tests

1. `internal/agent/plan_defaults_test.go`:
   - assert strict-object markers (`additionalProperties: false`) are present on fixed-shape objects.
   - assert map-field exceptions remain present.
2. `internal/agent/runtime_test.go`:
   - Claude + schema -> emits `--json-schema`.
   - Codex + schema -> emits `--output-schema` with a readable schema file path.
   - Codex without schema -> does not emit `--output-schema`.
   - temp schema files are cleaned up.

### Planning flow tests

1. CLI tests verify planning requests include non-empty schema metadata under Codex provider.
2. TUI action-wrapper tests verify generated/refine requests include non-empty schema metadata under Codex provider.
3. Quality-gate auto-refine path tests verify schema is retained across passes.

## Acceptance Criteria

1. Codex planning runs are launched with `--output-schema <file>` whenever planning metadata includes schema.
2. Claude planning runs remain schema-enforced via `--json-schema`.
3. Default plan schema is strict enough to satisfy observed Codex requirements while remaining Claude-compatible.
4. CLI and TUI planning flows consistently attach planning schema metadata.
5. All related unit and flow tests pass.

## References

1. `docs/notes/CODEX_CLAUDE_SCHEMA_AND_SESSION_LEARNINGS.md`
2. `internal/agent/runtime.go`
3. `internal/agent/plan_defaults.go`
4. `internal/cli/agent_flows.go`
5. `internal/tui/action_wrappers.go`
