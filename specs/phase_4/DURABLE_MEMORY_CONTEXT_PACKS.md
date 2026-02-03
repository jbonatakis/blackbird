Blackbird Durable Memory Backbone

Status: incomplete

Product specification (final)

1) Overview

Blackbird Durable Memory Backbone (DMMB) is a local-first, provider-agnostic system that captures what agents do during a session and exposes that history back to agents as fast, structured retrieval and “session context downloads” for each new task. It minimizes impact on agent execution by using a low-latency reverse proxy as a flight recorder and a separate, replayable derivation pipeline to build memory artifacts and indexes.

Key principles:
	•	Capture is truth: the immutable record is the intercepted provider traffic (+ optional Blackbird tool instrumentation), not agent-authored logs.
	•	Derivation is replayable: summaries, facts, and indexes are generated from the capture log and can be regenerated as logic improves.
	•	Session-first: each task gets a compact context download summarizing prior tasks in that session.
	•	Zero-key baseline: core memory works without any LLM provider key (lexical search + deterministic extractors).
	•	Optional enrichment: local models (LM Studio/Ollama) or the user’s already-selected provider can enhance memory, without affecting the core.

2) Problem statement

Agents working across multiple tasks in a session repeatedly re-discover:
	•	baseline project information,
	•	what decisions were made earlier,
	•	what files were changed,
	•	what constraints exist,
	•	what failed previously,
	•	what remains open.

Relying on agents to maintain logs is unreliable and pollutes prompts. Provider CLIs also often do heavy startup context gathering (repo scans, tool calls) that is expensive and non-deterministic.

Blackbird needs a durable memory backbone that:
	•	captures task outcomes and reasoning trails with provenance,
	•	supports fast retrieval and deterministic context packs,
	•	adds minimal latency to agent execution,
	•	does not require new API keys to function.

3) Goals and non-goals

Goals
	1.	Durable capture of provider interaction per run/task/session with minimal latency.
	2.	Session context pack injected pre-flight for every task.
	3.	Fast retrieval with bounded outputs.
	4.	Zero-provider-key mode: usable memory without embeddings/summarization from a hosted LLM.
	5.	Multi-surface integration: MCP for Claude-like ecosystems, tool schema for OpenAI-like ecosystems, CLI fallback.
	6.	Provenance: every memory item links back to source evidence (trace offsets, request IDs, tool outputs).
	7.	Upgradeable: regenerate derived artifacts from the same capture log.
        8. Compatible: works seamlessly regardless of user selected agent

Non-goals (initial)
	•	Perfect capture of local tool execution if it never appears in provider traffic (addressed via optional Blackbird instrumentation).
	•	“Rewrite provider responses” or modify assistant text mid-stream (explicitly avoided).
	•	Distributed/multi-tenant SaaS requirements.

4) Key concepts and scope

Session

A “session” is a coherent sequence of tasks contributing to the same overarching goal state. A session will often, but not always, represent an application to build, feature to add, or refactor to complete. It has:
	•	session_id
	•	session_goal (north star)
	•	ordered tasks and runs

Task and run
	•	task_id: an atomic unit of work within a session. It represents one leaf node on the blackbird generated plan tree
	•	run_id: one execution attempt of a task

Memory truth model
	•	Truth: trace log (WAL) + optional local tool event log
	•	Derived: artifacts (decisions, constraints, outcomes, open threads) + indexes

5) User stories

For Blackbird (orchestrator)
	•	As Blackbird, I can generate a compact context pack for a task using everything that happened earlier in the session, bounded by a token budget.
	•	As Blackbird, I can store a complete flight recorder of provider traffic for audit/debug/replay.
	•	As Blackbird, I can rebuild memory artifacts after improving extractors.

For agents
	•	As an agent, I receive a “session context pack” at task start that tells me high-level project details, sessions goals, what’s already been done and what constraints exist.
	•	As an agent, I can search memory and fetch full details by ID without receiving a transcript dump.

For developers
	•	As a developer, I can inspect what the agent saw/said and trace any memory item back to raw evidence.
	•	As a developer, I can run the system without configuring any hosted LLM keys.

6) System architecture

High-level components
	1.	Provider Proxy (Flight Recorder)
	•	Reverse proxy agent CLIs point to via base URL override or HTTP proxy.
	•	Streams requests/responses to upstream without buffering.
	•	Writes append-only trace events.
	2.	Trace Store (WAL)
	•	Durable append-only storage for raw events.
	•	Rotatable and compressible.
	3.	Canonicalizer
	•	Reconstructs provider streams into canonical “messages/toolcalls/results” timeline.
	4.	Artifact Builder
	•	Deterministic extractors produce task/session memory artifacts.
	•	Optional enrichment worker (local or provider API) can improve artifacts.
	5.	Indexes
	•	Baseline: lexical index (BM25) over artifacts and transcript segments.
	6.	Retrieval Engine
	•	Hybrid ranking (lexical + optional vector + scope + recency).
	7.	Context Pack Builder
	•	Composes the session context pack for the next task.
	8.	Integration Surfaces
	•	MCP server
	•	Tool schema adapter
	•	CLI commands
	•	Harness-only API for building context packs

Explicit design constraint: no response injection

The proxy does not modify provider responses. “Pre-flight injection” means Blackbird adds context to the agent’s initial prompt before the run begins.

7) Capture layer specification

7.1 Provider Proxy requirements
	•	Must support streaming responses (SSE/chunked) without buffering.
	•	Must be low latency; proxy overhead should be dominated by network and disk flush strategy.
	•	Must capture:
	•	request/response headers (sanitized)
	•	request/response bodies as streamed
	•	timings, status codes, error conditions
	•	Must propagate identifiers:
	•	read session_id, task_id, run_id if provided
	•	generate request_id always
	•	Must support backpressure policy:
	•	default lossless with bounded in-memory queue and fast local WAL
	•	configurable degrade mode: drop body chunks only under extreme pressure (still keep metadata)

7.2 Implementation language
	•	Proxy should be implemented in Go or Rust.
	•	Selection criteria:
	•	correctness and streaming behavior under load
	•	operational simplicity (local dev)
	•	low allocation and stable throughput

7.3 Redaction and privacy
	•	Always redact:
	•	Authorization, X-Api-Key, similar headers
	•	Support pluggable redaction rules for:
	•	known secret regexes
	•	user-defined patterns
	•	Store redaction policy version with each run.

7.4 Trace event format

Store append-only events (binary or framed JSON). Minimum event types:
	•	request.start (method, path, headers, ids)
	•	request.body.chunk (opaque bytes, ordering)
	•	response.start (status, headers)
	•	response.body.chunk (opaque bytes)
	•	response.end (durations, totals)
	•	error

8) Canonical model

Canonicalization normalizes different provider payloads into a stable timeline.

Canonical objects
	•	Message
	•	role: system/user/assistant
	•	content: text + structured parts (if available)
	•	ToolCall (if present in provider protocol or local tool log)
	•	name, args, call_id
	•	ToolResult
	•	call_id, result, error
	•	Metadata
	•	model name, temperature, max tokens, etc.
	•	Provenance
	•	source request_ids and event offsets

Canonicalization behavior
	•	Reassemble assistant message text from streaming deltas.
	•	Preserve an optional mapping from message spans → trace offsets for provenance.
	•	Normalize provider-specific field names without losing raw originals.

9) Memory artifacts and derivation

9.1 Artifact types (baseline, deterministic)

Each artifact is stored with:
	•	artifact_id
	•	session_id, task_id, run_id
	•	type
	•	content (structured + human-readable)
	•	provenance pointers
	•	builder_version

Core artifact types:
	1.	Task Outcome Record
	•	status: success/fail/blocked
	•	summary bullets (deterministic assembly)
	•	files touched (from diffs/tool logs if available)
	•	commands run + exit codes (if captured)
	•	notable errors (first lines)
	2.	Decisions
	•	decision statement
	•	rationale snippet (evidence-backed)
	•	supersedes/precedence links when changed
	3.	Constraints / invariants
	•	“must/must not” requirements
	•	scope: session-wide vs task-local
	4.	Open threads
	•	TODOs, blockers, unanswered questions
	5.	Transcript segments
	•	minimal chunking for citation and fallback retrieval

9.2 Deterministic extraction methods
	•	Pattern-based sentence detection for decisions/constraints/open threads.
	•	Structured parsing of:
	•	tool outputs (tests, build logs)
	•	diffstat and file lists
	•	Deduplication and consolidation rules:
	•	same decision repeated → single canonical decision with updated timestamps
	•	conflicting decisions → chain with “supersedes” relationship

9.3 Evidence bundle for enrichment

The memory worker sees only what Blackbird selects:
	•	task description
	•	assistant final answer or selected assistant turns
	•	tool outputs (if available)
	•	file-change summary (paths + diffstat + optionally selected hunks)
	•	key errors

No repo scanning, no tool calls, no external browsing.

10) Retrieval engine

10.1 Baseline retrieval
	•	Use a lexical index (BM25 via SQLite FTS5 or equivalent).
	•	Index artifact text plus key structured fields (file paths, symbols, error strings).
	•	Rank with boosts:
	•	scope match (same session/task)
	•	recency (last N tasks)
	•	artifact type priority (decision/constraint > outcome > transcript)

10.2 Retrieval API shape (conceptual)

Return bounded “cards”:
	•	id
	•	type (decision/constraint/outcome/open_thread/transcript)
	•	title (optional)
	•	snippet (short)
	•	score
	•	provenance summary (task/run, file paths)

And provide:
	•	get(id) to fetch full content
	•	related(id) to expand adjacency

11) Session context pack

11.1 Definition

A session context download is a structured, bounded packet describing:
	•	session goal
	•	key decisions and constraints so far
	•	what prior tasks implemented
	•	open threads
	•	relevant artifacts and pointers (ids) for expansion

This packet is injected pre-flight into the next task’s prompt by Blackbird.

11.2 Generation algorithm (high-level)

Inputs:
	•	current task description
	•	session artifacts
	•	token budget

Steps:
	1.	Scope: filter artifacts by session_id
	2.	Relevance scoring:
	•	keyword matching
	•	type weights
	•	recency
	•	task affinity (files/symbols mentioned)
	3.	Deduplicate and consolidate:
	•	keep latest active decisions/constraints
	•	summarize repeated outcomes as a single line per task
	4.	Budget allocation by section:
	•	decisions: N items / max tokens
	•	constraints: N items / max tokens
	•	implemented: last K tasks with 1–3 bullets each
	•	open threads: top M
	•	artifacts list: IDs only
	5.	Serialize in stable order

11.3 Token budgeting

Configurable hard limits (example defaults):
	•	total: 800–1500 tokens
	•	decisions: 200
	•	constraints: 150
	•	implemented: 300
	•	open threads: 150
	•	artifact pointers: 100

11.4 Agent usage contract

Blackbird includes instruction text:
	•	Use the context pack as authoritative session state.
	•	Only call memory tools if missing details.
	•	Prefer get(id) on referenced artifacts over broad searches.
	•	Limit tool calls (e.g., one search, up to two get).

12) Integration surfaces

12.1 MCP server

Expose:
	•	memory.search
	•	memory.get
	•	memory.context_for_task (optional, returns a fresh pack)
	•	memory.related (optional)

Use cases:
	•	Claude-style agents that can consume MCP tools.

12.2 Tool schema adapter

Expose the same semantics for tool-calling ecosystems:
	•	memory_search
	•	memory_get
	•	etc.

12.3 CLI fallback

Provide:
	•	blackbird mem search "query" --session <id> [--task <id>]
	•	blackbird mem get <artifact_id>
	•	blackbird mem context --task <id> --budget <n>

Rationale:
	•	Works even if an agent cannot call tools but can run commands.

12.4 Harness-only API

Blackbird uses internal calls to:
	•	build context packs
	•	write task outcome records
	•	checkpoint session summaries

13) Storage and indexing

Baseline storage (local-first)
	•	Store raw trace events (WAL) in a format optimized for append and replay.
	•	Store derived artifacts and indexes in a local database (SQLite is acceptable for MVP).

Must support:
	•	incremental updates (append new artifacts after each task)
	•	retention/rotation for traces
	•	“pin run/session” to prevent deletion

14) Operational requirements

Performance targets
	•	Proxy adds negligible latency relative to upstream (primarily disk write overhead).
	•	Retrieval should be fast locally (sub-100ms typical for scoped queries).
	•	Context pack generation should be fast (sub-second typical), with caching by:
	•	(session_id, task_id, builder_version, budget, session_state_hash)

Reliability
	•	If enrichment fails, baseline memory still works.
	•	If indexing lags, at minimum artifacts remain queryable.
	•	Trace capture should be resilient to upstream errors and client disconnects.

Observability
	•	Metrics:
	•	proxy overhead p50/p95
	•	queue depth / backpressure events
	•	trace write throughput
	•	index update lag
	•	retrieval latency p50/p95
	•	Debug:
	•	artifact → provenance → raw trace navigation
	•	Token usage from provider responses

15) Security and privacy
	•	Redact secrets in capture layer.
	•	Optional encryption at rest for trace store and artifacts.
	•	Default to local-only access for MCP/tool servers (127.0.0.1).
	•	Provide a “privacy mode” to disable capture of request/response bodies (metadata only) at the cost of memory quality.

16) Configuration

Key knobs:
	•	memory.mode = deterministic | local | provider
	•	proxy.upstream_url
	•	proxy.lossless = true|false
	•	trace.retention_days, trace.max_size, rotation
	•	context.budget_tokens, per-section caps
	•	retrieval.k_default, retrieval.max_snippet_tokens
	•	enrichment.local_endpoint (optional)
	•	enrichment.provider_model (optional; uses existing configured provider)
	•	redaction.ruleset_version
	•	“Passthrough mode” that collects token information and other relevant metrics from provider calls, but does not run the core memory workflows

17) Phased delivery plan

Phase 1 (MVP): deterministic backbone
	•	Go/Rust streaming proxy flight recorder
	•	Trace store + canonicalizer for initial provider
	•	Deterministic artifacts: outcomes/decisions/constraints/open threads
	•	Lexical index + retrieval API
	•	Context download generation + pre-flight injection
	•	CLI retrieval

Success criteria:
	•	Each task starts with a useful session context download.
	•	Agents stop repeating “what happened before?” within a session.

Phase 2: tool integrations
	•	MCP server
	•	Tool schema adapter
	•	Agent prompt contract tuned for minimal tool calls

Success criteria:
	•	Agents can expand details via get(id) with low token overhead.

Phase 3: local tool instrumentation (if needed)
	•	Blackbird logs tool execution events (file reads/writes, diffs, commands)
	•	Merge into canonical timeline
	•	Strong “files changed” and “why” memory

Success criteria:
	•	Memory can answer “what changed” accurately, not just “what was said.”

18) Risks and mitigations
	1.	Proxy overhead under heavy streaming
	•	Mitigation: lossless WAL tuned for sequential writes; bounded queue; avoid base64; separate binary blob store.
	2.	Incomplete tool visibility
	•	Mitigation: optional Blackbird instrumentation for tool events.
	3.	Memory drift / contradictions
	•	Mitigation: decision supersession model; prioritize latest active constraints; provenance required.
	4.	Agent ignores memory tools / pack
	•	Mitigation: strict prompt contract + pre-flight context pack; limit tool budget; provide “do not redo” warnings.

⸻

Deliverable definition (what “done” means)

A Blackbird session can run multiple tasks where each task begins with a compact, provenance-backed context pack derived from prior tasks, and agents can retrieve memory via MCP/tools/CLI—without requiring any new hosted LLM API key for the baseline system.

