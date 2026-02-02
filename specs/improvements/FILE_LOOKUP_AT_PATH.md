status: complete
---
name: file-lookup-at-path
overview: Add "@path" file lookup in the plan generate and plan refine text boxes so that typing "@" opens a picker that filters existing files by path prefix, similar to Cursor's @ file mention.
todos:
  - id: file-picker-core
    content: Implement file picker state, file listing by prefix, and picker UI (list + key handling)
    status: pending
  - id: intercept-at-plan-generate
    content: Intercept "@" in plan generate modal textareas; route keys to picker; insert chosen path
    status: pending
  - id: intercept-at-plan-refine
    content: Intercept "@" in plan refine modal textarea; route keys to picker; insert chosen path
    status: pending
  - id: tests-docs
    content: Add tests for file listing and picker behavior; document in docs if needed
    status: pending
---

# File Lookup (@path) in Plan and Refine Text Boxes

## Goals

- **@-triggered file picker**: In the TUI plan generate and plan refine modals, when the user types `@` in a text area, show a picker that lists files under the project root, filtered by what they type after `@` (e.g. `@src/` shows paths starting with `src/`).
- **Cursor-like UX**: Behavior similar to Cursor's @ file mention—type `@path/to/` and see a list of matching paths; arrow keys to select, Enter to insert the chosen path (e.g. `@internal/tui/model.go`), Escape to cancel.
- **No new dependencies**: Use existing Bubble Tea components (`bubbles/textarea`, optionally `bubbles/list`) and standard library for file walking. Prefer low-dependency, clarity over cleverness.

## Scope

**In scope**

- **Plan generate modal**: Enable @ file lookup in the **Project Description** and **Constraints** text areas (multi-line). Optionally in **Granularity** (single-line) if straightforward.
- **Plan refine modal**: Enable @ file lookup in the **Change request** text area.
- **Workspace root**: Files are listed relative to the current working directory (same as execution context; e.g. `os.Getwd()`). No separate "project root" config for this feature initially.
- **Insert format**: On selection, insert the chosen path including the `@` prefix, e.g. `@internal/tui/plan_generate_modal.go`. Exact format (with or without leading `@` in the inserted string) is an implementation detail as long as the result is clearly a file reference.

**Out of scope (initial spec)**

- CLI plan generate/refine prompts: no @ file picker in terminal prompts; TUI only.
- Other TUI text inputs (e.g. agent question modal, plan review revision prompt): can be added later using the same mechanism.
- Respecting `.gitignore` when listing files: initial version can list all files under the root (or a simple exclude list); `.gitignore` support can be a follow-up.
- Symbol/line lookup inside files (e.g. `@file.go:42` or `@file.go::functionName`): only path completion in this spec.

## User Experience

### Trigger

- User focuses a text area in the plan generate or plan refine modal and types `@`.
- The `@` is inserted into the text and a **file picker** opens (e.g. a list below or adjacent to the text area).
- Initially the list shows a default set (e.g. top-level files/dirs, or all files up to a limit) or files matching the empty prefix.

### Filtering

- As the user continues typing after `@`, the list filters to paths that **start with** the typed prefix (e.g. `@src/` → paths like `src/foo.go`, `src/bar/baz.go`).
- The typed characters appear in the text area so the user sees e.g. `@src/` (Cursor-like).
- Matching is case-sensitive or case-insensitive; spec leaves this to implementation (case-sensitive is simpler and matches many path completions).

### Picker interaction

- **Arrow Up / Down**: Move selection in the list (wrap or clamp at ends).
- **Enter**: Insert the selected path into the text area, replacing the range from `@` through the current filter text (so the final value contains e.g. `@internal/tui/model.go`), and close the picker.
- **Escape**: Close the picker without inserting. Optionally remove the `@` and the filter text from the text area so the field is unchanged from before the @; or leave `@query` in place (implementation choice; recommend removing for a clean cancel).

### When picker is open

- Keys that would normally go to the text area (letters, digits, path separators, backspace) are **routed to the picker**: they update the filter query and the list; the same characters are also reflected in the text area so the user sees what they typed.
- Tab / Shift+Tab: can either (a) still move focus between form fields, closing the picker, or (b) be consumed by the picker (e.g. Tab = next list item). Recommend (a) for consistency with existing form navigation.

## Technical Approach

### 1. File listing

- **Root**: Use `os.Getwd()` as the workspace root (same as execution/context usage).
- **Walk**: Use `filepath.Walk` or `os.ReadDir` (with recursion) to list files and directories. Limit depth or count if needed to avoid huge lists (e.g. cap at 500 entries or 3 levels initially).
- **Prefix filter**: After `@`, the "query" is the string the user typed. Filter walked paths so that each path is **relative to root** and has a prefix matching the query (e.g. query `src/` matches `src/foo.go`). Directories can be included so the user can narrow (e.g. `src/` then choose `src/internal/` or a file under `src/`).
- **Format for display**: Show paths relative to workspace root (e.g. `internal/tui/model.go`). No need to show absolute paths in the picker.

### 2. Picker state

- State can live on the form (e.g. `PlanGenerateForm`, `PlanRefineForm`) or in a small shared struct used by both modals. Suggested fields:
  - **Open**: bool (picker visible).
  - **Query**: string (filter text after `@`).
  - **Matches**: []string (paths matching the query).
  - **Selected index**: int (index into `Matches`).
  - **@ start position**: needed to replace the range when inserting (see below). Store either a character offset into the text area value or (line, col). Depends on what the textarea exposes; if only full value is available, track the value length at `@` time and length of query to compute the span to replace.

### 3. Key handling

- In the form’s `Update()` (and in `HandlePlanGenerateKey` / `HandlePlanRefineKey`), **before** delegating to the textarea:
  - If picker is **closed** and the key is `@`: insert `@` into the textarea, set picker open, set query to `""`, run file list with empty prefix, set selected index 0.
  - If picker is **open**:
    - **Escape**: close picker; optionally set text area value to value before `@` (cancel).
    - **Enter**: replace the span from `@` to end of query in the text area with `@` + selected path; close picker.
    - **Up/Down**: update selected index; do not pass to textarea.
    - **Backspace**: if query is non-empty, remove last rune from query and refilter; also remove that character from the text area. If query is empty, optionally close picker or no-op.
    - **Printable character** (e.g. letter, digit, `/`): append to query, refilter list, update text area to show `@` + query.
    - **Tab / Shift+Tab**: recommend closing picker and passing through for form field navigation.
- When picker is open, do **not** pass the key to the textarea for normal editing; the text area’s content is updated only by our logic (insert @, append query runes, replace span on select, or revert on cancel).

### 4. Inserting the chosen path

- When the user presses Enter with a selection:
  - **Text to insert**: e.g. `@internal/tui/model.go` (the selected path with `@` prefix).
  - **Range to replace**: from the position where `@` was inserted to the current end of the filter text in the text area. So if the value is `...some text @src/` and the user selects `src/foo.go`, replace `@src/` with `@src/foo.go`.
- Implementation: get current value, compute `valueBeforeAt + "@" + selectedPath + valueAfterQuery`, then set the text area value. If the textarea exposes cursor position (or a single index), restore cursor after the inserted path. Bubble Tea textarea has `SetValue` and `InsertString`; use whichever allows correct replacement and cursor placement.

### 5. Picker UI

- Render a list of matches below (or beside) the active text area when picker is open. Use `bubbles/list` or a simple custom list (e.g. a vertical list of strings with a highlighted row).
- Limit visible rows (e.g. 8–10) with scrolling if there are many matches.
- Style to fit the existing modal (e.g. border, same palette as plan generate/refine modals).

## Key code touchpoints

- **New helper (optional)**: A small `filepicker` or `at_file_picker` helper in `internal/tui` for: file list by prefix, picker state (query, matches, selected index), and key handling for open picker. Forms can hold this state and call into the helper for "filter matches" and "handle key."
- **Plan generate modal**: `internal/tui/plan_generate_modal.go` — in `PlanGenerateForm.Update` and `HandlePlanGenerateKey`, add branch for "picker open" and "key is @"; for the description and constraints textareas, run the picker flow and insert chosen path into the focused textarea.
- **Plan refine modal**: `internal/tui/plan_refine_modal.go` — same idea for the change-request textarea.
- **Workspace root**: Use `os.Getwd()` (or a shared helper that returns cwd for TUI); no new config in this spec.

## Edge cases

- **Empty match list**: If the query matches no files, show an empty list and still allow Up/Down (no-op), Enter (no-op or insert current query as literal text), Escape (close). Do not crash.
- **Very large repos**: Cap the number of files walked or shown (e.g. 500 paths); when over limit, show a message like "Too many matches; narrow your search" or truncate the list.
- **Root not available**: If `os.Getwd()` fails, do not open the picker on `@`; optionally show a brief message or no-op.
- **Picker already open**: If the user types `@` again while picker is open, treat it as a printable character (append `@` to query and refilter).
- **Focus change**: If the user moves focus to another field (e.g. Tab) while picker is open, close the picker and apply the same Tab behavior as today (no insert).

## Future extensions (out of scope)

- Respect `.gitignore` (or a project-specific ignore list) when listing files.
- Optional project root config (e.g. from global/project config) instead of always `os.Getwd()`.
- Extend to other TUI text inputs (agent question modal, plan review revision prompt).
- Symbol or line completion inside files (`@file:line` or `@file::symbol`).
