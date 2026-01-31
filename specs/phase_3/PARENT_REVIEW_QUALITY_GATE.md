# Parent as Code Reviewer / Quality Gate

## Purpose

When all children of a parent task are done, the parent acts as a **code reviewer / quality gate**: it runs once (as a reviewer, not as a doer) to evaluate its own acceptance criteria against the work produced by the children, and optionally check for major bugs or security issues. If the review finds deficiencies, the system identifies which child task(s) need to be reworked and **resumes** those children's runs (using stored session information) with the parent's feedback injected into the resumed session. This keeps work chunked (only leaf tasks do implementation) while adding a review layer that can trigger targeted rework via resume + context.

This feature depends on **pause/resume/restart** (session capture, resume semantics, run records with `provider_session_ref` and `repo_root`) and is **separate from** "ready for execution": parents are never in the ready list for normal execution; parent "execution" is a distinct **review run**.

## Core Requirements

### 1) Trigger: when to run parent review

* When **all children** of a parent task are **done**, the parent becomes a candidate for a **review run**.
* A review run is triggered **at most once per "all children just became done"** event (idempotent per parent per such event). No re-review on every plan load.
* Only tasks that **have children** (non-empty `childIds`) are eligible for review; leaf tasks are never "reviewed" in this sense.

### 2) Parent review run semantics

* A **review run** is a distinct run type (e.g. `review` or `quality_gate`), not "execute task."
* The run is for the **parent task ID**; the agent receives:
  * The parent's **acceptance criteria** (which encompass the children's scope).
  * Context from **completed child runs** (e.g. summaries, artifacts, or references to run outputs).
  * Optional: repo state, diffs, or other bounded context.
* The agent's role is **reviewer**: evaluate whether acceptance criteria are met, and optionally flag major bugs or security issues. It must **not** implement work; it only assesses and, if needed, produces structured output for resuming children.

### 3) Review response: which children to resume and what feedback

* The review run must produce a **structured response** that includes:
  * **Pass/fail**: whether the parent's acceptance criteria are satisfied.
  * If **fail**:
    * **Resume task IDs**: the list of child task(s) the review says need to be reworked (only those the review deems relevant).
    * **Feedback for resume**: text (and/or structured issues) to inject into the **resumed** session(s)—e.g. deficiencies in acceptance criteria, bugs, or security issues the parent identified. This is the context the resumed run receives so the agent can address it.
* The parent's acceptance criteria are the source of truth for "what must be true"; the review maps failures to specific children and produces feedback for those children.

### 4) Resume with feedback (no new run)

* For each child task the review says must be reworked:
  * **Resume** that child's run using the **existing** run record: same provider, `provider_session_ref`, `repo_root`, and resume semantics as in the Pause/Resume spec (e.g. `blackbird resume <taskId>` / TUI resume action).
  * **Inject** the parent's feedback (deficiencies, bugs, security issues) into the **context** sent to the agent when resuming. The resumed session receives the original task context **plus** the review feedback so the agent can fix the issues.
* No "mark todo and start a new run": we **resume** the same run/session and add feedback. If resume is impossible (e.g. no session ref, provider lost session), behavior falls back to existing resume failure handling (error and recommend restart); this spec does not change that.

### 5) Default: pause and show user

* **By default**, when a parent review **fails** and identifies children to resume:
  * The system **pauses** and **surfaces** the review result to the user (e.g. in TUI and/or CLI): which children need rework, and the feedback text.
  * The user **explicitly** chooses to resume (e.g. "Resume task X with review feedback" or "Resume all identified"). No automatic resume until the user (or future config) enables it.

### 6) Future config: auto-resume after review

* A **future** config system may expose an option (e.g. `parent_review.auto_resume` or similar) that, when enabled, skips "pause and show user" and **automatically** resumes each child task identified by the review with the parent's feedback injected. This spec does not implement the config system; it only defines the behavior so that when config exists, auto-resume can be added without changing the review or resume-with-feedback contract.

### 7) Relation to "ready for execution"

* Parent tasks remain **ineligible** for normal "ready for execution" (see READY_LEAF_ONLY). Only **leaf** tasks are dispatched for implementation. Parent "execution" is **only** the review run, triggered by "all children done," and produces at most "resume these children with this feedback."

## Non-Goals

* Implementing the config system or the `parent_review.auto_resume` option in this spec (only defining the behavior so it can be added later).
* Changing pause/resume/restart semantics or run record schema beyond what is needed to pass feedback into the resumed context.
* Deriving or auto-updating parent task **status** (e.g. parent becomes "done") when review passes—can be in scope later; this spec focuses on trigger, review run, response shape, and resume-with-feedback.
* Running parent review when **not** all children are done.

## Definitions

* **Parent task**: a plan item with non-empty `childIds` (a container for child work).
* **Leaf task**: a plan item with empty `childIds`; the only tasks eligible for normal "ready for execution" and implementation runs.
* **Review run**: a run of type `review` (or `quality_gate`) for a **parent** task ID, triggered when all its children are done. Input: parent acceptance criteria + child run context; output: pass/fail + optional `resumeTaskIds` + feedback for resume.
* **Resume with feedback**: resuming a child's existing run (using stored session ref and resume semantics) and adding the parent review's feedback to the context sent to the agent so the resumed session can address deficiencies, bugs, or security issues.

## User Experience

### CLI

* When a parent review is triggered (e.g. after the last child of that parent completes):
  * Run the review agent for the parent; display progress (e.g. "Running parent review for <parentId>").
* When review **passes**:
  * Indicate success; no further action required for that parent.
* When review **fails** and identifies children to resume:
  * **Default**: Print the review result: which children need rework and the feedback. Instruct user to resume explicitly, e.g. `blackbird resume <taskId>` (and ensure resume path injects the stored feedback into context). Do not auto-resume.
  * **Future (config)**: If auto-resume is enabled, after printing or logging the review result, automatically invoke resume for each identified child with feedback; user may still see summary output.

### TUI

* When a parent review runs:
  * Show that the parent is under review (e.g. "Reviewing <parentId>…").
* When review **fails** and identifies children to resume:
  * **Default**: Show the review result in a dedicated view or modal: list of children to resume and the feedback text. Provide explicit actions, e.g. "Resume <taskId> with feedback" or "Resume all identified." No automatic resume.
  * **Future (config)**: If auto-resume is enabled, after showing the result, automatically start resume for each identified child with feedback.

## Persistence and Integration

### Review run record

* Review runs should be stored (e.g. under `.blackbird/runs` or equivalent) with:
  * Run type: `review` (or `quality_gate`).
  * Parent task ID.
  * Outcome: pass / fail.
  * If fail: `resumeTaskIds`, feedback text (or ref to stored feedback), and linkage so that when the user (or auto-resume) resumes a child, the correct feedback is loaded and injected into the resume context.

### Resume context composition

* When resuming a child task **with** parent review feedback:
  * Load the child's run record (for session ref, provider, repo root).
  * Load the review feedback associated with "this child was identified by parent <parentId> review run <reviewRunId>."
  * Build the context pack for the resumed run = **normal task context** (prompt, etc.) **plus** a **"Parent review feedback"** section containing the feedback text (and optionally structured issues). The agent receives both so it can continue the session and address the feedback.

### Idempotence

* Trigger parent review only when transitioning to "all children done" (e.g. when the last child is set to `done`). Do not re-trigger on every execute loop or plan load. A simple approach: record that "parent P has been reviewed for this set of completed children" (e.g. by review run existence or a lightweight marker) and skip if already done.

## Agent Integration

### Review request/response schema

* **Request** to the review agent (conceptual):
  * Role: "You are a code reviewer. Do not implement; only evaluate."
  * Parent task: id, title, acceptance criteria.
  * Child run context: for each child, task id, summary or artifact refs (and optionally snippets) so the reviewer can assess against parent acceptance criteria.
  * Optional: instructions to also flag major bugs or security issues, and to map failures to specific child task IDs.
* **Response** schema (structured, e.g. JSON):
  * `passed`: boolean.
  * If not `passed`:
    * `resumeTaskIds`: array of child task IDs that need to be reworked.
    * `feedbackForResume`: string (or object with per-task feedback) to inject when resuming those tasks. Must be suitable for appending to or including in the resumed run's context.

### Resume path

* Existing resume path (CLI `blackbird resume <taskId>`, TUI resume action) must be extended so that when resuming a task that has **pending parent review feedback** (stored when the review run failed and identified this task), the context builder includes that feedback. No new resume entry point is required; the same "resume" action carries feedback when present.

## Deliverables (implementation scope)

* Trigger logic: when last child of a parent becomes `done`, enqueue or run parent review (idempotent).
* Review run type and execution path: run parent as reviewer with parent acceptance criteria + child run context; parse structured response (pass/fail, `resumeTaskIds`, `feedbackForResume`).
* Persistence: store review runs; store or link feedback so resume can load it.
* Resume-with-feedback: when resuming a task that has pending review feedback, include feedback in the context sent to the agent.
* Default UX: on review failure, pause and show user the result; user explicitly resumes with feedback (no auto-resume in initial implementation).
* Tests: (1) parent review triggers when all children done; (2) review failure produces correct `resumeTaskIds` and feedback; (3) resuming a task with pending feedback includes that feedback in context; (4) idempotence: no duplicate review for same "all children done" state.

## Done Criteria

* When all children of a parent are done, a single parent review run executes and evaluates the parent's acceptance criteria (and optionally bugs/security).
* If the review fails, it identifies which child task(s) to resume and supplies feedback; the system pauses and shows the user, who can explicitly resume those children; on resume, the session receives the parent's feedback in context.
* Parent tasks are never "ready for execution" for implementation; only leaf tasks are. Parent review is a separate, well-defined run type and flow.