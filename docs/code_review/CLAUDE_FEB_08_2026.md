Project Overview

  Blackbird is a control plane for durable, dependency-aware planning and execution with AI agents. It's written in Go 1.25 and functions as a terminal-native CLI tool that orchestrates AI coding agents (Claude
  Code, Codex) to execute structured software development tasks.

  ---
  Architecture & Core Components

  1. Project Structure

  blackbird/
  ├── cmd/blackbird/          # Main entry point
  ├── internal/
  │   ├── agent/             # Agent registry & integration
  │   ├── cli/               # Command-line interface handlers
  │   ├── config/            # Configuration management
  │   ├── execution/         # Task execution engine
  │   ├── plan/              # Work graph data structures
  │   ├── plangen/           # AI-powered plan generation
  │   ├── planquality/       # Plan validation & linting
  │   └── tui/               # Terminal UI (Bubble Tea)
  ├── docs/                  # Documentation
  ├── specs/                 # Feature specifications
  └── static/                # Assets (GIFs, etc.)

  2. Core Packages

  internal/plan - Work Graph Management

  - Core data structure: WorkGraph containing a map of WorkItems
  - Each WorkItem has:
    - ID, title, description, acceptance criteria
    - Prompt (canonical instruction for the agent)
    - Parent/child relationships (hierarchical tree)
    - Dependencies (DAG for execution ordering)
    - Status (todo, in_progress, done, blocked, etc.)
    - Timestamps and metadata
  - Stored as JSON in blackbird.plan.json at the project root
  - Validates dependency graphs (no cycles), computes readiness based on deps

  internal/execution - Task Execution Engine

  The core execution flow:

  1. Context Building (context.go):
    - Assembles a "context pack" for each task
    - Includes: task prompt, project snapshot, dependency outputs
    - Prefers .blackbird/snapshot.md, falls back to OVERVIEW.md or README.md
  2. Launcher (launcher.go):
    - Spawns headless agent processes
    - Claude: Uses --permission-mode bypassPermissions + --session-id for resume
    - Codex: Uses exec --full-auto
    - Passes context pack via stdin as JSON
    - Captures stdout/stderr, detects agent questions
    - Supports streaming output via BLACKBIRD_AGENT_STREAM=1
  3. Run Records (storage.go, types.go):
    - Persistent execution artifacts stored in .blackbird/runs/<taskID>/<runID>.json
    - Contains: start/end times, stdout/stderr, exit codes, status, context used
    - Enables auditing, debugging, and resume capability
  4. Lifecycle Management (lifecycle.go):
    - Status transitions: in_progress → done/failed/waiting_user
    - Atomic updates to plan file
    - Handles execution review checkpoints (gates)
  5. Resume & Questions (resume.go, questions.go):
    - Detects AskUserQuestion output from agents
    - Transitions runs to waiting_user state
    - blackbird resume <taskID> prompts for answers and continues execution

  internal/agent - Agent Integration

  - Registry (registry.go): Supports Claude and Codex
  - Runtime abstraction: command, args, timeout, provider
  - Auto-applies provider-specific flags for headless execution
  - Metadata extraction for plan defaults and question parsing

  internal/cli - Command Interface

  Main commands implemented in cli.go:
  - init - Create empty plan
  - validate - Check plan integrity
  - plan generate/refine - AI-powered planning
  - list, pick, show - Browse work items
  - add, edit, delete, move - Manual plan editing
  - deps add/remove/set/infer - Dependency management
  - execute - Run ready tasks
  - resume, retry - Continue/retry executions
  - runs - View execution history

  internal/tui - Terminal User Interface

  Built with Bubble Tea (Charm Bracelet framework):
  - ViewModes: Home, Task List, Task Detail, Execution View, Settings
  - Interactive components:
    - Task picker with filtering (ready/blocked/all)
    - Execution dashboard with live status
    - Agent question modal for inline responses
    - Plan review/approval flow
  - Actions: Queue tasks, approve/reject work, answer questions
  - Real-time updates during task execution

  internal/plangen - AI Plan Generation

  - Generates structured work graphs from user descriptions
  - Uses LLM APIs to break down features into tasks with prompts
  - Supports constraints, granularity control, and iterative refinement

  internal/planquality - Plan Validation

  - Linting rules for plan quality
  - Validates prompts, dependencies, acceptance criteria
  - Ensures tasks are executable before agent runs

  ---
  Build System

  Local Development

  Makefile (Makefile):
  make build    # Builds to ~/.local/bin/blackbird-<branch>
  make test     # go test ./...
  make format   # go fmt ./...
  make tag      # Version tagging (BUMP=patch|minor|major)

  The build target:
  - Uses git rev-parse --abbrev-ref HEAD to determine branch
  - For main: builds as blackbird-dev
  - For feature branches: builds as blackbird-<branch-name>
  - Installs to ~/.local/bin/ for local testing

  CI/CD Pipeline

  GitHub Actions (.github/workflows/):

  1. CI Workflow (ci.yml)

  Triggered on: PRs and pushes to main
  - Checkout code
  - Setup Go (version from go.mod)
  - Format check (gofmt)
  - Build all packages (go build ./...)
  - Run tests (go test ./...)

  2. Release Workflow (release.yml)

  Triggered on: GitHub releases
  jobs:
    release:
      # Cross-platform build matrix
      - goos: [linux, windows, darwin]
      - goarch: [amd64, arm64]
      - Builds binaries with ldflags for version injection
      - Uploads to GitHub Release Assets
      - Generates SHA256 checksums

    update-homebrew:
      # Automatically updates Homebrew tap
      - Downloads release checksums
      - Updates formula with new version + SHA256s
      - Commits to jbonatakis/homebrew-tap

  Build flags used:
  -ldflags="-s -w -X main.Version=$TAG"
  - -s -w: Strip debug symbols (smaller binaries)
  - -X main.Version=...: Inject version at compile time

  ---
  Distribution

  Homebrew (macOS/Linux)

  brew tap jbonatakis/tap
  brew install blackbird
  Formula auto-updated by release workflow.

  From Source

  go build -o blackbird ./cmd/blackbird
  Requires Go 1.25+.

  Pre-built Binaries

  Available on GitHub Releases for:
  - Linux (amd64, arm64)
  - macOS/Darwin (amd64, arm64)
  - Windows (amd64)

  ---
  Dependencies

  Key Go modules (go.mod):
  - Bubble Tea (charmbracelet/bubbletea) - TUI framework
  - Lipgloss (charmbracelet/lipgloss) - Terminal styling
  - Bubbles (charmbracelet/bubbles) - Reusable UI components
  - Standard library for most functionality

  ---
  Execution Model

  Core Workflow

  1. Plan Definition: User creates/generates a work graph with tasks and dependencies
  2. Readiness Computation: System identifies "ready" tasks (dependencies met, status=todo)
  3. Task Selection: User picks tasks via CLI or TUI
  4. Context Assembly: System builds context pack (task prompt + project snapshot + dep outputs)
  5. Agent Invocation: Spawns headless agent with context on stdin
  6. Monitoring: Captures output, detects questions, records results
  7. Status Updates: Marks task done/failed/waiting, unblocks dependents
  8. Resume/Continue: Handles agent questions and continues execution

  Stateless Agent Execution

  - Agents are disposable workers (no long-lived sessions)
  - Each run gets a bounded scope (single task)
  - Context pack provides all necessary information
  - Outputs recorded in durable run records
  - Enables reliable execution across restarts

  Agent Integration Points

  Agents interact via:
  - stdin: Receive JSON context pack
  - stdout/stderr: Emit logs, questions, results
  - working tree: Modify files directly (native CLI behavior)
  - Special markers: AskUserQuestion for human-in-the-loop

  ---
  Data Persistence

  Files & Storage

  - blackbird.plan.json: Work graph (version controlled)
  - .blackbird/runs/<taskID>/<runID>.json: Execution records
  - .blackbird/snapshot.md: Project state summary (optional)
  - .blackbird/agent-selection.json: User's agent choice
  - .blackbird/config.json: Configuration settings

  Atomic Updates

  - Plan file writes use atomic rename pattern
  - Prevents corruption during concurrent access
  - Validation enforced before writes

  ---
  Testing Strategy

  - Unit tests: All packages have *_test.go files
  - Test coverage: Tracked in coverage.out
  - CI validation: Tests must pass on every PR
  - Format enforcement: gofmt checks in CI

  ---
  Key Design Principles

  1. Durability: All state externalized to disk (survives restarts)
  2. Inspectability: Human-readable JSON, clear audit trails
  3. Dependency-awareness: Explicit DAG prevents ordering errors
  4. Bounded context: Task-scoped execution prevents drift
  5. Agent-agnostic: Plugin-style architecture for multiple agents
  6. Terminal-native: Built for developer workflows (CLI + TUI)

  This architecture makes Blackbird a reliable "master memory" and control plane for AI-assisted software development, treating short-lived agent sessions as workers executing against a durable, structured plan.

✻ Sautéed for 1m 47s

❯ /context
  ⎿  Context Usage
     ⛁ ⛀ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁   claude-sonnet-4-5-20250929 · 47k/200k tokens (24%)
     ⛀ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛁
     ⛁ ⛁ ⛁ ⛁ ⛁ ⛁ ⛶ ⛶ ⛶ ⛶   Estimated usage by category
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ System prompt: 3.3k tokens (1.6%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ System tools: 16.7k tokens (8.3%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ Skills: 61 tokens (0.0%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛁ Messages: 30.9k tokens (15.4%)
     ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶ ⛶   ⛶ Free space: 116k (58.0%)
     ⛶ ⛶ ⛶ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝   ⛝ Autocompact buffer: 33k tokens (16.5%)
     ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝ ⛝

     Skills · /skills

❯ Give a detailed review. Don't focus on nits, but on architectural design and conceptual robustness.

⏺ Architectural & Conceptual Review of Blackbird

  Core Strengths

  1. Externalized State as First Principle

  The decision to treat the plan file as the authoritative source of truth is architecturally sound. By externalizing all project state to durable JSON artifacts, Blackbird solves a fundamental problem with AI
  agents: context amnesia. This is the project's strongest conceptual foundation.

  Why this works:
  - Enables agent restarts without loss of continuity
  - Makes state inspectable and debuggable
  - Allows version control of the plan itself
  - Decouples agent lifetime from work lifetime

  2. Dependency-Aware Execution Model

  The explicit DAG for task dependencies + hierarchical tree for human comprehension is a smart dual representation. Many task systems pick one or the other; having both serves different needs well:
  - Tree: Natural for human navigation and planning
  - DAG: Correct for execution ordering

  The readiness computation (UnmetDeps) is straightforward and correct. The system won't execute tasks prematurely.

  3. Stateless Agent Invocation

  Treating agents as disposable workers rather than persistent sessions is architecturally mature:
  - Reduces coupling to specific agent implementations
  - Makes failure recovery simple (just retry)
  - Bounds context size per invocation
  - Prevents conversation drift

  The "context pack" abstraction is clean: each run gets exactly what it needs, nothing more.

  4. Human-in-the-Loop Integration

  The question/answer flow (AskUserQuestion detection, waiting_user state, resume command) acknowledges that AI agents aren't fully autonomous. This is pragmatic and realistic. The alerting + inline response
  mechanism via TUI is well-thought-out.

  ---
  Architectural Concerns & Design Trade-offs

  1. Fragile Agent Integration Layer

  Issue: The agent integration relies on stdout parsing to detect questions and stdin JSON for context. This is brittle:

  // launcher.go:92
  questions, qErr := ParseQuestions(record.Stdout)

  Problems:
  - Format coupling: If Claude Code changes its output format, Blackbird breaks
  - No protocol versioning: What if agents need to communicate structured data beyond questions?
  - Limited extensibility: Hard to add new interaction patterns (progress updates, artifacts, etc.)

  Better approach:
  Consider a lightweight agent protocol (e.g., JSON-RPC over stdio, or structured output markers):
  {"type": "question", "id": "q1", "text": "...", "options": [...]}
  {"type": "progress", "percent": 50}
  {"type": "artifact", "path": "...", "content": "..."}

  This would make the integration more robust and allow richer agent interactions without stdout parsing hacks.

  2. Context Pack Size Management is Underspecified

  Issue: The system mentions "bounded context" as a principle, but there's no enforcement mechanism:

  // context.go - no size limits visible
  func BuildContext(g plan.WorkGraph, taskID string, snapshotPath string) (ContextPack, error)

  Problems:
  - What happens when the project snapshot grows to 50k tokens?
  - How are dependency outputs summarized if a parent task produces large artifacts?
  - No token counting or budgeting visible in the code

  Risks:
  - Context pack could exceed agent limits (200k for Claude, but still finite)
  - Quality degrades with oversized context
  - User has no visibility into context size until execution fails

  Recommendation:
  - Add explicit token budgeting to ContextPack
  - Implement summarization strategies for large outputs
  - Surface context size in TUI before execution
  - Allow configurable limits per agent type

  3. Execution Review Gates Feel Bolted-On

  The "execution review checkpoints" mentioned in README.md (Phase 2) seem conceptually incomplete:

  Questions:
  - When exactly are review gates triggered? After every task? Configurable?
  - What's the approval interface? (Not visible in TUI code I saw)
  - How do you handle "reject and revise" vs. "reject and rollback"?
  - What if you reject 5 tasks deep into a dependency chain?

  Missing:
  - Clear rollback/undo semantics
  - Diff visualization for changes
  - Cascading rejection handling

  This feels like a critical feature that's mentioned but not fully integrated into the execution model.

  4. Concurrency Model is Unclear

  Issue: The OVERVIEW.md mentions:
  "optionally executing multiple independent tasks concurrently (when safe and permitted by dependency constraints)"

  But I don't see any concurrency control in the execution package:

  // execute.go - appears to be sequential
  func runExecute(args []string) error

  Architectural gaps:
  - No worker pool or parallel execution visible
  - No conflict detection (what if two tasks modify the same file?)
  - No lock files or mutex around plan updates
  - Race conditions possible if multiple blackbird instances run

  Recommendation:
  Either:
  1. Remove the claim from the spec (be honest: it's sequential)
  2. Implement proper concurrency with file locks, conflict detection, and configurable parallelism

  Overpromising on concurrency and not delivering is worse than being upfront about sequential execution.

  5. Project Snapshot Update Strategy is Vague

  Issue: The snapshot is described as "periodically refreshed" but:
  - No code for automatic snapshot updates
  - No logic for detecting when the snapshot is stale
  - No diff detection between snapshot and reality

  Problem:
  If the snapshot diverges from codebase reality, agents get stale context. This defeats the entire "master memory" concept.

  Questions:
  - Who updates the snapshot? User manually? Agent after each task?
  - How do you validate snapshot freshness?
  - What if someone pushes code changes outside Blackbird?

  Recommendation:
  - Add snapshot versioning + timestamps
  - Implement staleness detection (hash current codebase state)
  - Auto-regenerate snapshot after N tasks or on demand
  - Surface "snapshot is stale" warnings in TUI

  6. Agent Provider Abstraction is Leaky

  The buildLaunchArgs function has provider-specific logic:

  // launcher.go:172
  switch normalizeProvider(provider) {
  case "codex":
      return append([]string{"exec", "--full-auto"}, args...)
  case "claude":
      return append([]string{"--permission-mode", "bypassPermissions"}, args...)

  Problem:
  - This will grow unboundedly as more agents are added
  - Provider-specific flags hardcoded in execution layer
  - Violates Open/Closed Principle

  Better approach:
  type Provider interface {
      BuildLaunchArgs(baseArgs []string) []string
      SupportsResume() bool
      SessionIDFormat() string
  }

  Each provider becomes a pluggable module with its own config.

  ---
  Data Model Concerns

  1. Status State Machine is Informal

  The statuses are defined:
  StatusTodo | StatusQueued | StatusInProgress | StatusWaitingUser |
  StatusBlocked | StatusDone | StatusFailed | StatusSkipped

  But valid transitions aren't enforced:
  - Can a task go from done back to todo? (Probably not, but nothing prevents it)
  - What about failed → in_progress (retry)?
  - Is skipped terminal?

  Recommendation:
  Implement a state machine with explicit transitions:
  type Transition struct {
      From Status
      To   Status
      Valid bool
  }

  var ValidTransitions = map[Transition]bool{
      {StatusTodo, StatusInProgress}: true,
      {StatusInProgress, StatusDone}: true,
      // etc.
  }

  This prevents invalid state mutations and makes the lifecycle explicit.

  2. WorkGraph is a Flat Map, Not a True Graph

  type WorkGraph struct {
      Items map[string]WorkItem
  }

  Problem:
  - Dependency queries are O(n) scans
  - No indexes for common queries (ready tasks, blocked tasks)
  - Computing transitive dependencies is expensive

  For small plans (< 100 tasks): Fine.

  For large plans (1000+ tasks): Will be slow.

  Recommendation:
  Consider building derived indexes on load:
  type WorkGraphIndex struct {
      ByStatus map[Status][]string
      ByParent map[string][]string
      Dependents map[string][]string
      TransitiveDeps map[string][]string
  }

  Or switch to a graph database for large-scale plans.

  3. No Plan Versioning or History

  Issue: The plan file is mutated in-place. There's no:
  - Undo/redo support
  - History of plan changes
  - Blame tracking (who changed what when)

  Risk:
  - Accidental deletions are permanent
  - Hard to debug "why is this task blocked now?"
  - No audit trail for team collaboration

  Recommendation:
  - Add plan versioning (store snapshots in .blackbird/history/)
  - Implement blackbird plan log to show changes over time
  - Add blackbird plan diff <v1> <v2> for comparisons

  ---
  Reliability & Error Handling

  1. Atomic Plan Updates: Good

  The SaveAtomic pattern (write to temp file, rename) is correct. This prevents corruption.

  2. Run Record Storage: No Compaction

  Issue: Run records accumulate in .blackbird/runs/<taskID>/ forever.

  Problems:
  - Disk space grows unbounded
  - Listing runs gets slower over time
  - No cleanup strategy

  Recommendation:
  - Add blackbird runs gc --older-than 30d for cleanup
  - Limit storage to last N runs per task
  - Archive old runs to compressed format

  3. No Distributed Locking

  Issue: Multiple blackbird instances can run concurrently and corrupt the plan.

  Current state: No locks visible.

  Recommendation:
  - Use file locking (flock on Unix) on blackbird.plan.json
  - Detect concurrent modification and abort
  - Or: Require explicit blackbird server mode for team usage

  ---
  Scalability Questions

  1. How Large Can Plans Get?

  - 100 tasks? Probably fine.
  - 1000 tasks? Plan file size, memory usage, TUI rendering could be issues.
  - 10,000 tasks? Definitely need optimization.

  Recommendation:
  Test with large synthetic plans and set explicit limits. Document max supported plan size.

  2. How Does This Work for Teams?

  The spec mentions "small teams coordinating" but:
  - No merge conflict resolution for plan file
  - No access control (who can execute what?)
  - No audit log for multi-user actions

  This feels like a single-developer tool that's being positioned for teams without the necessary infrastructure.

  Recommendation:
  Be honest: this is v1.0 for solo developers. Team features are future work.

  ---
  Conceptual Gaps

  1. Lack of Rollback Semantics

  If a task completes but produces broken code:
  - How do you undo it?
  - Does Blackbird integrate with git for rollback?
  - Is there a "safe mode" to run in a branch?

  Missing: Clear rollback story.

  2. No Resource Management

  Agents consume:
  - CPU/memory (running locally)
  - API tokens (Claude/Codex)
  - Time

  Missing:
  - Rate limiting
  - Cost tracking
  - Resource quotas

  For production use, these are critical.

  3. Testing Story is Weak

  How do you test a plan without executing it?
  - Dry-run mode?
  - Validate prompts are well-formed?
  - Mock agent runs?

  Missing: Testing/simulation capabilities.

  ---
  What's Working Exceptionally Well

  1. Clear Separation of Concerns

  The package structure is excellent:
  - plan = pure data structures
  - execution = runtime logic
  - tui = presentation
  - agent = external integration

  No circular dependencies, clean boundaries.

  2. Documentation Quality

  The OVERVIEW.md, README.md, and package-level docs are unusually good for a young project. Clear intent, well-explained concepts.

  3. TUI as a First-Class Interface

  Building a real TUI (not just CLI) shows commitment to UX. The Bubble Tea framework choice is solid.

  4. Testing Discipline

  Every package has test files. This is not universal in Go projects.

  ---
  Final Assessment

  Conceptual Robustness: 7.5/10

  Strengths:
  - Core idea (externalized state + dependency DAG + stateless agents) is architecturally sound
  - Solves real problems with AI agent workflows
  - Design principles are well-articulated

  Weaknesses:
  - Agent integration is too fragile (stdout parsing)
  - Context size management is underspecified
  - Concurrency claims aren't backed by implementation
  - Missing rollback/undo semantics
  - Team collaboration features are aspirational, not real

  Recommendations by Priority

  P0 (Critical):
  1. Implement agent protocol instead of stdout parsing
  2. Add context size limits and token budgeting
  3. Document max supported plan size and performance characteristics

  P1 (Important):
  4. Formalize status state machine with valid transitions
  5. Add snapshot staleness detection and auto-refresh
  6. Implement file locking for concurrent safety

  P2 (Nice to have):
  7. Add plan versioning and history
  8. Implement run record garbage collection
  9. Build provider abstraction interface
  10. Add testing/dry-run capabilities

  Overall:

  Blackbird is a well-designed MVP with a strong conceptual foundation. The core insight (externalizing agent memory to durable artifacts) is valuable and differentiating. However, it's currently optimized for solo
   developers working on small-to-medium projects.

  To scale to larger plans and team usage, it needs:
  - More robust agent integration
  - Better resource management
  - Clearer boundaries on what's implemented vs. aspirational

  The architecture is solid, but some key features mentioned in the spec aren't fully realized in the code. Be more explicit about what's v1.0 vs. future work.