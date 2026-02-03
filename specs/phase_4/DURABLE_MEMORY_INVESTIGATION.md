Got it — below is a **clean, copy/pastable handoff document** written as an **investigation write-up**, not a spec.
It assumes the reader is another engineer picking this up later and explains *what exists, what was learned, and what should change*, without prescribing an exact implementation.

I’ve removed meta commentary and “recommendation framing” language and written it as a factual handoff.

---

# Codex Reverse Proxy — Investigation Handoff

This document summarizes an exploratory investigation into building a reverse proxy in front of Codex / ChatGPT-backed APIs, with the goal of enabling **durable, session-scoped memory capture** for Blackbird.

The proxy was initially built to understand routing, authentication modes, streaming behavior, and what information can be extracted from Codex traffic. During the investigation, it became clear that the proxy can serve as a **flight recorder** for agent activity, but should not be responsible for higher-level memory semantics.

This document describes:

* how the proxy works today,
* what behaviors were validated,
* what assumptions did not hold,
* and what structural changes are needed before extending this further.

This is **not** a 1:1 build spec.

---

## 1. Purpose of the Proxy

The reverse proxy serves two roles:

1. **Codex / ChatGPT transport adapter**

   * Route requests to the correct upstream depending on authentication method.
   * Rewrite paths to match upstream expectations.
   * Preserve streaming semantics for SSE responses.

2. **Provider interaction flight recorder**

   * Capture what the agent sends to and receives from the provider.
   * Enable downstream reconstruction of conversations, tool calls, and outcomes.
   * Act as the source of truth for memory derivation.

The proxy must remain **non-invasive**:

* It does not modify model output.
* It does not inject memory or context mid-stream.
* It does not make decisions about task-level meaning.

---

## 2. High-level Behavior

* The proxy listens locally (default `:8080`).
* Requests are forwarded to one of two upstreams:

  * **API upstream** (default `https://api.openai.com`)
  * **ChatGPT backend upstream** (default `https://chatgpt.com`)
* Routing is path-based, with additional checks for ChatGPT-authenticated requests.
* Paths are rewritten to either `/v1/*` (API) or `/backend-api/*` (ChatGPT backend).
* All HTTP methods are forwarded.
* Streaming responses are preserved.
* The proxy emits structured logs for requests and responses.

Primary file: `cmd/reverse-proxy/main.go`
Config: `internal/config/config.go`

---

## 3. Authentication Modes and Routing

Codex can operate in two distinct modes, which require different upstream routing.

### API key mode

* Requests include `Authorization: Bearer ...`
* `/responses` must be sent to `https://api.openai.com/v1/responses`

### ChatGPT login mode

* Requests include ChatGPT session headers:

  * `Chatgpt-Account-Id`
  * or `Session_id`
* `/responses` must be sent to `https://chatgpt.com/backend-api/codex/responses`

### Routing decision

A request is treated as ChatGPT-authenticated if either header is present.
This affects only routing; the proxy does not validate tokens.

---

## 4. Path Rewrite Rules

### API upstream (`api.openai.com`)

Incoming paths are normalized to `/v1/*`:

* `/responses` → `/v1/responses`
* `/responses/...` → `/v1/responses/...`
* `/chat/completions` → `/v1/chat/completions`
* `/completions` → `/v1/completions`
* `/models` → `/v1/models`
* `/embeddings` → `/v1/embeddings`
* `/v1/...` → unchanged

### ChatGPT backend (`chatgpt.com`)

Incoming paths are normalized to `/backend-api/*`:

* `/api/codex/...` → `/backend-api/api/codex/...`
* `/wham/...` → `/backend-api/wham/...`
* `/backend-api/...` → unchanged
* `/responses` (ChatGPT login) → `/backend-api/codex/responses`

---

## 5. Request and Response Handling

* `Host` is rewritten to match the upstream.
* `X-Forwarded-For` is removed to avoid Cloudflare issues.
* Request URL scheme and host are rewritten.
* Streaming responses are passed through without buffering.

---

## 6. Logging and Capture (Current State)

The proxy emits structured JSON logs.

### Request logs

Each request emits a structured record containing:

* method
* path
* status
* duration
* correlation ID
* sanitized headers

Sensitive headers are redacted:

* `Authorization`
* `Proxy-Authorization`
* `X-API-Key`
* `X-OpenAI-API-Key`

### Request body logging (optional)

* Can decode `gzip` or `zstd`.
* Body logging is capped (default 32KB).
* Intended for inspection, not durability.

### Response body logging (optional)

* Can capture up to a configured max (default 256KB).
* Used primarily to tune SSE parsing.

---

## 7. Conversation Extraction (Derived Output)

The proxy currently performs **best-effort conversation extraction** and emits JSONL records.

### Inputs

* Parses `body.input[]`.
* Extracts only `type="message"` with `role="user"`.
* Filters out injected/system prompts.
* Logs only user messages after the last assistant message to avoid replaying history.

### Tool calls

* Extracts `function_call` and `function_call_output` items.
* Logs tool name, call ID, and truncated previews.

### Outputs

* Parses SSE streams even when `Content-Type` is empty.
* Extracts text from:

  * `response.output_text.delta`
  * or `response.output_text.done`
* Emits assistant output as `conversation_output`.

### Metadata

* Extracts token usage from `response.completed` / `response.failed`.
* Extracts response metadata (model, status, IDs, cache keys).

These records are intentionally **lossy** and truncated.

---

## 8. Identity and Scoping (Key Finding)

The proxy currently relies on:

* ChatGPT `Session_id`
* proxy-generated correlation IDs

This is **not sufficient** for Blackbird’s memory model.

### Conclusion

ChatGPT session identity must not be treated as the authoritative session boundary.

Blackbird needs its own identifiers:

* `blackbird_session_id` — one application / project thread
* `task_id` — unit of work
* `run_id` — execution attempt

### Required change

Blackbird should inject these IDs as headers on every provider request:

```
X-Blackbird-Session-Id
X-Blackbird-Task-Id
X-Blackbird-Run-Id
```

The proxy should treat them as opaque values and include them in all logs.
ChatGPT `Session_id` remains useful but secondary.

---

## 9. Capture vs Derivation (Important Distinction)

The investigation surfaced a critical separation of concerns.

### What the proxy should do

* Capture request/response metadata.
* Capture streamed bytes.
* Preserve ordering and timing.
* Remain fast and non-invasive.

### What the proxy should not do

* Decide what constitutes a “decision,” “constraint,” or “task outcome.”
* Be the only place where meaning exists.

### Updated model

There should be **two logical streams**:

1. **Trace / WAL (authoritative)**

   * Append-only.
   * Full-fidelity capture.
   * Used for replay and reprocessing.

2. **Conversation stream (derived)**

   * Parsed messages, tool calls, usage.
   * Truncated and filtered.
   * Optimized for immediate retrieval.

The current proxy output fits the second category and should be treated as such.

---

## 10. Truncation Policy

Current truncation limits are acceptable **only for derived logs**.

For durable memory:

* Full payloads must be captured somewhere (WAL or blob store).
* Derived records should include pointers back to raw trace data.

---

## 11. SQLite Store (Current Usage)

SQLite works well as a **real-time working store** for headless Codex workflows.

Current tables support:

* sessions
* turns
* messages
* tool calls / outputs
* usage and metadata

This enables immediate reuse of context across runs without batch jobs.

---

## 12. What’s Missing for Session Context

Transcript-level storage alone is not enough to build session-scoped context downloads.

An additional **artifact layer** is needed downstream:

* decisions
* constraints
* task outcomes
* open threads / TODOs
* “don’t redo this” warnings

These artifacts should:

* be generated outside the proxy,
* reference transcript/trace data for provenance,
* be deterministic initially.

---

## 13. Pre-flight Context Injection (Clarification)

“Pre-flight memory injection” does **not** mean modifying provider responses.

Instead:

* Blackbird assembles a context packet before starting a task.
* This packet summarizes prior session state.
* It is injected into the agent’s initial system prompt.
* The agent then runs normally through the proxy.

The proxy remains passive.

---

## 14. Updated Mental Model

| Layer             | Responsibility                                   |
| ----------------- | ------------------------------------------------ |
| Blackbird harness | Defines session/task/run, builds context packets |
| Reverse proxy     | Transport adapter + flight recorder              |
| Trace/WAL         | Source of truth                                  |
| Conversation logs | Derived, best-effort transcript                  |
| Artifact layer    | Decisions, constraints, outcomes                 |
| Retrieval         | Search + fetch by ID                             |
| Agent             | Executes task                                    |

---

## 15. Summary of Key Takeaways

* Codex / ChatGPT traffic can be reliably intercepted and streamed.
* Authentication mode determines routing but should not affect memory semantics.
* The proxy is viable as a **flight recorder**, not a memory engine.
* ChatGPT `Session_id` is not a substitute for Blackbird session identity.
* Conversation extraction is useful but inherently lossy.
* Durable memory requires a replayable trace and a downstream artifact layer.
* Session context should be injected **before** agent execution, not during.

This investigation validated the proxy approach and clarified the architectural boundaries needed to turn it into a durable memory foundation for Blackbird.
