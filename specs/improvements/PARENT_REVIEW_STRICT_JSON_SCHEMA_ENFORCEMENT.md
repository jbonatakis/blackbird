# Parent Review Strict JSON Schema Enforcement
status: proposed

## Purpose

Parent review outputs currently depend on prompt compliance plus post-parse validation. This is not sufficient for deterministic behavior. Parent review runs must be launched with an explicit review-specific JSON schema so provider-side structured-output constraints are applied at invocation time.

## Problem

Current parent review execution:

1. builds parent review context (`internal/execution/parent_review_context.go`)
2. launches agent via execution launcher (`internal/execution/launcher.go`)
3. parses stdout as parent review response (`internal/execution/parent_review_response.go`)

The launcher path does not currently pass a review-specific JSON schema to the provider. This allows non-JSON/prose responses to be emitted even when prompt instructions request JSON.

## Goals

1. Enforce structured output for parent review runs using an explicit JSON schema at launch time.
2. Keep post-parse validation as a second safety layer.
3. Make behavior deterministic and testable across supported providers.
4. Fail fast when a provider cannot enforce schema for parent-review runs (no prompt-only fallback).

## Non-Goals

1. Replacing `ParseParentReviewResponse` or topology validation logic.
2. Changing parent-review gate semantics (idempotence, pause-on-fail, resume flow).
3. Generalizing all execution flows to schema-constrained output in this change.

## Current State

- Parent review run uses `RunParentReview` -> `LaunchAgentWithStream`.
- `LaunchAgentWithStream` receives only runtime + context; no request metadata/schema.
- `internal/agent/runtime.go` already supports schema flags via `RequestMetadata.JSONSchema` but that path is used by planning flows, not execution launcher.

## Required Design

### 1. Add parent-review response JSON schema

Create a review-specific schema helper in execution (or agent, if shared reuse is preferred), e.g.:

- `internal/execution/parent_review_schema.go`

Schema requirements:

1. Top-level object.
2. Required fields:
   - `passed` boolean
   - `resumeTaskIds` array of strings
   - `feedbackForResume` string
3. Optional field:
   - `reviewResults` array of objects
4. `reviewResults` item:
   - required: `taskId`, `status`
   - optional: `feedback`
   - `status` enum: `passed|failed`
5. Child ID constraints:
   - `resumeTaskIds.items.enum` must be parent child IDs
   - `reviewResults[].taskId.enum` must be parent child IDs

Note:
- Conditional constraints (`if passed=true then resumeTaskIds empty`) remain enforced by `ValidateParentReviewResponse` even if provider schema support is limited.

### 2. Make execution launcher schema-aware

Extend launch API to accept structured-output options, e.g.:

- new launch options struct on execution launcher:
  - `ResponseFormat` (optional)
  - `JSONSchema` (optional)

Files:

- `internal/execution/launcher.go`
- `internal/execution/launcher_test.go`

Requirements:

1. When launch options include schema, provider-specific schema flags are appended to command args.
2. Keep existing behavior unchanged for calls that do not pass schema options.
3. Avoid duplicating provider-flag logic; extract shared builder used by both:
   - `internal/agent/runtime.go` and `internal/execution/launcher.go`
   - or move shared flag builder into a small helper package.

### 3. Enforce schema use in parent review runs

Update parent review orchestration to always pass schema:

- `internal/execution/parent_review_runner.go`

Behavior:

1. Build child-aware review schema before launch.
2. Pass schema through launcher options for every `RunParentReview` invocation.
3. If selected provider cannot enforce schema for this launch path, return explicit error and stop review run (do not fall back to prompt-only mode).

### 4. Provider policy for strict mode

Because this requirement is explicit ("always schema"), parent review runs must be rejected when schema enforcement cannot be applied.

Policy:

1. Supported provider paths must set schema flags.
2. Unsupported provider paths must fail with actionable error:
   - include provider name
   - include remediation hint (switch provider or add schema flag support)

### 5. Keep parser validation as defense-in-depth

No removal of:

- `ParseParentReviewResponse`
- `ValidateParentReviewResponse`

Even with schema-enabled launch, parser validation remains authoritative for:

1. parent/child topology checks
2. normalization rules
3. semantic constraints not guaranteed by provider schema implementation

## Implementation Plan

1. Add `ParentReviewResponseJSONSchema(childIDs []string) string`.
2. Introduce launch metadata/options in execution launcher.
3. Wire schema options from `RunParentReview`.
4. Add strict provider support checks in launch path used by review runs.
5. Keep existing prompt guidance, but do not rely on it for structure.

## Testing Requirements

### Unit tests

1. Schema builder tests:
   - includes required fields
   - includes child ID enums
   - deterministic output ordering
2. Launcher tests:
   - schema flag emitted when schema provided (per provider support)
   - no schema flag emitted when schema omitted
   - unsupported provider + schema returns clear error
3. Parent review runner tests:
   - run invokes launcher with non-empty review schema
   - unsupported provider path fails fast before parsing

### Regression tests

1. Parent review gate integration still pauses on failed review.
2. Passing reviews still complete execution and preserve run record behavior.
3. Existing parse/validation tests continue to pass unchanged.

## Acceptance Criteria

1. Parent review launches always include a review-specific JSON schema.
2. Parent review does not run in prompt-only mode when schema cannot be enforced.
3. Provider-specific schema enforcement behavior is covered by tests.
4. Existing parent review parse/validation and gate semantics are preserved.
5. `go test ./internal/execution ./internal/agent ./internal/tui ./internal/cli` passes.

## Notes

- This spec is additive to `specs/improvements/PARENT_REVIEW_QUALITY_GATE.md`.
- The quality-gate behavior remains unchanged; only output-contract enforcement at launch time is strengthened.
