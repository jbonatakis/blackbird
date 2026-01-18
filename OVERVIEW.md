# Product Specification: AI-Orchestrated CLI for Structured, Dependency-Aware Software Delivery

## 1. Product Summary

A terminal-native product that functions as the authoritative “master memory” and execution control plane for building software with AI coding agents (e.g., Claude Code, Codex, or other agent runtimes). The product externalizes project planning, state, decisions, and execution history into durable, inspectable artifacts, enabling short-lived, task-scoped agent sessions to reliably deliver work without needing long-lived conversational context.

The core workflow is:

1. define a structured feature/task graph where every node has an associated agent prompt,
2. compute what work is actionable based on dependencies and current status,
3. select and queue work from the terminal,
4. run agents against specific tasks with a consistent “context pack” (project snapshot + task prompt + relevant prior outputs),
5. continuously track and surface progress in a live CLI dashboard,
6. allow agents to request clarification/confirmation from the user with prominent alerts and inline responses,
7. continuously maintain a concise “current state of the app” summary for new agent runs.

The product is designed to make AI work reliable, repeatable, and coordinated across many agent invocations by treating memory and task structure as first-class artifacts.

---

## 2. Target Users

### Primary

* Solo developers and senior engineers building non-trivial systems who want:

  * reliable continuity across many agent runs
  * stronger control over what agents do
  * a structured plan that stays synchronized with code reality

### Secondary

* Small teams coordinating AI-assisted work through a shared repo-local plan, where task structure and “project memory” are versioned artifacts.

---

## 3. Problems Solved

1. **Loss of continuity across agent runs**

   * Agents forget past context, causing rework and regressions.

2. **Unreliable execution when context is oversized**

   * Overloaded prompts reduce quality and increase drift.

3. **Poor coordination between tasks**

   * Flat lists and ad-hoc prompting don’t enforce ordering, dependencies, or readiness.

4. **Weak visibility into what AI is doing**

   * Users lack a clear real-time view of status, progress, logs, and time in state.

5. **High friction when agents need human input**

   * Agents often need confirmation or clarification; existing tooling doesn’t integrate user responses cleanly.

6. **Parallelization without guardrails**

   * Running multiple tasks concurrently can cause collisions and inconsistent outcomes.

---

## 4. Core Product Concepts

### 4.1 Project “Master Memory”

The product maintains a durable, human-readable representation of project state that persists across sessions and can be used to seed new agent runs. This “master memory” is made of:

* **Structured work graph**: the set of tasks/features, their hierarchy, and dependencies.
* **Project snapshot**: a periodically refreshed summary of “current state of the app.”
* **Decision log**: a record of key decisions and rationale to prevent re-deciding.

These artifacts are the source of truth for intent and context, reducing reliance on any single agent’s context window.

### 4.2 Structured Work Graph (Feature Tree + Dependency DAG)

Work is represented as a hierarchical tree for human comprehension and an explicit dependency graph for execution correctness.

* **Tree**: features → subfeatures → tasks
* **DAG**: prerequisite relationships that determine readiness and build order

Each node in the graph is a first-class work item with metadata and a canonical agent prompt.

### 4.3 Stateless, Task-Scoped Agent Execution

Agents are treated as disposable workers:

* each run has a bounded scope (a single task node)
* receives a standardized context pack
* produces outputs that are recorded and linked to the task
* updates task status and project memory as appropriate

This reduces drift and makes the system resilient to agent restarts or failures.

### 4.4 Context Pack

A context pack is the curated set of information provided to an agent for a given task. It is designed to:

* be sufficient for task completion
* remain compact and consistent
* be inspectable for auditability

The context pack is composed of:

* the task’s canonical prompt
* the latest project snapshot
* relevant decision log entries
* outputs/artifacts from prerequisite tasks
* optionally task-scoped notes and constraints

The product surfaces context-pack size and composition (including token estimates where possible) to help users manage context window usage.

---

## 5. Product Capabilities

## 5.1 Work Definition and Management

### Work Items

Each work item (at every level—feature, subfeature, task) includes:

* **Identifier**: stable, unique ID
* **Title**: concise summary
* **Description**: context and acceptance criteria
* **Canonical prompt**: the instruction sent to the agent for that node
* **Hierarchy**: parent/children relationships
* **Dependencies**: prerequisite node IDs (graph edges)
* **Status**: current lifecycle state (see below)
* **Artifacts**: links/refs to outputs (diffs, branches, files, PRs, notes)
* **History**: timestamps and status transitions
* **Tags/metadata**: optional categorization, priority, ownership, estimates

### Status Model

The product supports clear statuses that reflect both planning and execution reality. At minimum:

* `todo`: defined but not yet actionable or started
* `ready`: all dependencies satisfied; actionable
* `queued`: selected for execution but not yet started
* `in_progress`: actively being worked on by an agent or user
* `waiting_user`: blocked on user clarification/confirmation
* `blocked`: cannot proceed due to unmet dependency or external constraint
* `done`: completed
* `failed`: execution ended unsuccessfully
* `skipped`: intentionally not done

The product must:

* compute readiness based on dependency completion
* explain why items are blocked
* optionally derive parent status from children (e.g., feature is “in progress” if any child is in progress)

### Dependency Awareness

The product:

* validates the dependency graph (e.g., rejects cycles)
* computes which tasks are actionable (“ready”) based on completion of prerequisites
* allows users to view dependency chains and block reasons
* supports selectively showing/hiding tasks based on dependency state (e.g., only show ready tasks)

---

## 5.2 Terminal Task Selection and Navigation

### Fast Selection

The product provides an interactive terminal selection interface that lets users:

* filter by readiness (default: show “ready”)
* toggle visibility of blocked/done items
* search by title/ID/tags
* quickly open a task to view details or run it

### Task Detail View

Users can view:

* full description and acceptance criteria
* canonical prompt
* dependencies and readiness explanation
* execution history and artifacts
* current context pack composition (snapshot version, included decision entries, prerequisite outputs, estimated token usage)

---

## 5.3 Queueing and Execution

### Task Queue

Users can build a queue of tasks to execute. The product supports:

* enqueue/dequeue/reorder
* queue views filtered by readiness
* execution state per queued item

### Execution Semantics

The product supports:

* executing a single selected task
* executing queued tasks in order, constrained by readiness
* optionally executing multiple independent tasks concurrently (when safe and permitted by dependency constraints)

Execution outcomes are recorded as task artifacts and in run history.

---

## 5.4 Agent Integration as a Pluggable Runtime (Conceptual)

The product can invoke one or more agent runtimes to execute tasks. Regardless of the underlying agent provider, the product treats agents uniformly:

A task run results in:

* a run record with lifecycle state
* a log/event stream
* produced artifacts (code changes, patch/diff, notes, generated docs)
* optional structured outputs (e.g., “created files”, “tests run”, “questions asked”)
* status updates on the associated task

The product does not require persistent agent sessions; instead it optimizes for consistent, repeatable task runs.

---

## 5.5 Real-Time CLI Dashboard

### Purpose

A live terminal dashboard provides immediate visibility into what is happening now and what is blocked, waiting, or completed.

### Dashboard Views

The dashboard includes:

1. **Active workers / runs**

   * which task each worker is processing
   * current run state
   * elapsed time in state
   * last activity timestamp
2. **Selected task/run details**

   * task metadata, dependencies, artifacts
   * recent status transitions
   * context pack summary

     * snapshot version identifier
     * included decision entries count
     * included prerequisite outputs count
     * estimated context size and, where available, actual usage
3. **Event/log stream**

   * streaming view of events (system/agent/git/tests-style categories conceptually)
   * ability to filter the stream and inspect recent history

### Run Lifecycle States (Dashboard-Oriented)

The dashboard surfaces run-specific states such as:

* `queued`
* `building_context`
* `running_agent`
* `waiting_user`
* `applying_changes`
* `verifying`
* `done`
* `failed`
* `canceled`

Each run state change is time-stamped and reflected in elapsed-time metrics.

---

## 5.6 Human-in-the-Loop Clarification & Confirmation

### Agent-to-User Questions

Agents can request:

* **clarification** (missing info)
* **confirmation** (permission to proceed)
* **decision** (choose among options)

These requests must:

* transition the run into a `waiting_user` state
* generate a prominent alert in the CLI
* be answered directly in the CLI
* resume execution using the user’s response
* be recorded permanently in run history (and optionally in the project decision log)

### Alerting

When user input is requested, the product provides:

* prominent visual alerting in the dashboard (highlight/badge/attention state)
* optional audible alert
* a clear “unread questions” indicator
* a queue of pending questions across runs

### Response Experience

Users can:

* answer inline in the dashboard
* choose from options when provided
* attach a note explaining rationale
* optionally mark the response as a durable project decision

All Q/A is associated with a run and task for traceability.

---

## 5.7 Continuous Project Snapshot (“Current State of the App”)

### Purpose

Maintain a compact, regularly refreshed representation of current application state that can be used as the first thing included in new agent contexts.

### Snapshot Content (What it captures)

At minimum:

* implemented features and current behavior
* current architecture overview (major modules and responsibilities)
* key interfaces/contracts and invariants
* known limitations and outstanding issues
* conventions (naming, patterns, guidelines that agents should follow)
* pointers to where key code lives

### Snapshot Requirements

* **Bounded**: stays within a target size and format so it is usable in an agent context window
* **Trustworthy**: updated frequently enough to remain accurate
* **Inspectable**: users can read it directly
* **Versioned**: each snapshot has an identifier (timestamp/hash) so task runs can reference exactly what they used

### Relationship to Task Runs

Each task run references:

* which snapshot version it used
* which decisions/notes were included
* optionally which prerequisite outputs were included

This supports reproducibility and debugging.

---

## 5.8 Decision Log

### Purpose

Prevent repeated re-litigation of foundational choices by capturing “what we decided and why.”

### Decision Entries

Each decision includes:

* decision statement
* rationale / tradeoffs
* scope (what it affects)
* timestamp and origin (user vs agent-assisted)
* optionally links to tasks/runs that produced it

The product enables promoting a clarification/confirmation answer into a durable decision entry.

---

## 6. End-to-End User Journeys

## 6.1 From Idea to Executable Plan

1. User defines a high-level goal.
2. The product holds a structured feature tree with tasks and subtasks.
3. Every node has a canonical prompt so execution is possible at any level.
4. Dependencies are defined so readiness can be computed.

Outcome: a durable, navigable work graph exists, and “ready tasks” are identifiable.

## 6.2 Selecting and Running Work

1. User opens the task picker and sees only “ready” tasks by default.
2. User selects a task and starts execution.
3. The product constructs a context pack (task prompt + project snapshot + relevant history).
4. A run begins and appears in the dashboard.

Outcome: user can see exactly what is being worked on and how long it has been running.

## 6.3 Agent Requires Input

1. During execution, the agent asks a clarification/confirmation question.
2. The dashboard prominently alerts the user and shows the question.
3. User answers inline; optionally marks it as a decision.
4. Execution resumes with that response included in context.

Outcome: the agent is unblocked quickly, and the interaction is recorded.

## 6.4 Queueing and Ongoing Progress

1. User enqueues multiple tasks.
2. The product executes tasks when they become ready.
3. The dashboard shows:

   * which tasks are running
   * which are queued but blocked
   * which completed and produced artifacts

Outcome: the user can run structured, dependency-aware work sessions with high visibility.

---

## 7. Product Outputs and Artifacts

The product produces durable artifacts that users can inspect and version:

* work graph definitions (features/tasks/prompts/deps/status)
* run records (what ran, when, final state)
* event/log history per run
* question/answer history per run
* project snapshot versions
* decision log entries
* links to produced artifacts (patches/diffs/docs)

These artifacts enable:

* reproducibility (“what context did this run use?”)
* debugging (“why did it fail?”)
* continuity (“what’s the current state?”)
* onboarding (“how does the system work?”)

---

## 8. Non-Functional Requirements (What the product must feel like)

### 8.1 Trust and Inspectability

* Users must be able to see:

  * what the agent was asked to do
  * what context it was given
  * what it changed/produced
  * why a task is blocked or waiting

### 8.2 Low Friction

* Fast selection and navigation in the terminal
* Minimal ceremony to run the next task
* Clear, immediate signaling when the user is needed

### 8.3 Resilience

* Runs, status, and memory persist across restarts
* The dashboard can reconnect and reconstruct the current state
* Failures leave clear traces rather than silent corruption

### 8.4 Boundedness and Drift Control

* Project snapshot and prompts must be bounded and structured so agent runs remain reliable.
* The system should emphasize stable “canonical prompts” and durable project memory over conversational accumulation.

---

## 9. Scope Boundaries

### In-scope

* structured work graph with prompts and dependencies
* readiness computation and filtered selection
* task queueing and execution tracking
* real-time dashboard
* clarification/confirmation question flow with alerting + inline responses
* continuous project snapshot and decision log

### Explicitly out of scope (for this spec)

* specific implementation details (tech stack, storage format, process model)
* specific agent provider features or APIs
* detailed merge strategies, CI integration, or repository governance
* advanced multi-user concurrency controls (beyond shared artifacts)

---

## 10. Success Criteria (Product-Level)

A user should be able to:

* maintain a durable, structured plan where every task is executable via an associated prompt
* see only actionable work by default, based on explicit dependencies
* run tasks with AI agents without re-explaining the project each time
* recover instantly from agent restarts because memory is externalized
* monitor active work in a live dashboard with clear run states and elapsed time
* respond to agent questions promptly via CLI alerts and inline answers
* onboard a new agent run with a reliable project snapshot that reduces drift and repeated questions
