# Plan Review and Revision Flow - TUI Implementation

## Overview

This implementation adds a plan review and revision flow to the TUI, matching the CLI's review loop functionality. After an agent successfully generates a plan, users can review it, accept it, request revisions, or reject it.

## Features Implemented

### 1. Plan Review Modal (`plan_review_modal.go`)

A new modal component that displays after plan generation with:
- **Plan Summary**: Shows item count and lists top-level features
- **Three Action Choices**:
  - **Accept**: Saves the plan and returns to normal TUI view
  - **Revise**: Prompts for revision request and re-runs agent (limited to 1 revision)
  - **Reject**: Discards the plan and returns to previous state

### 2. Two-Mode Review Flow

#### Mode 1: Choose Action (ReviewModeChooseAction)
- Displays plan summary with item count
- Shows up to 5 top-level features as preview
- Three options: Accept, Revise, Reject
- Navigation:
  - `↑/↓` or `k/j`: Navigate between options
  - `1-3`: Quick select option by number
  - `Enter`: Confirm selection
  - `ESC`: Cancel and return to main view

#### Mode 2: Revision Prompt (ReviewModeRevisionPrompt)
- Shows textarea for entering revision request
- Validates that request is non-empty
- Navigation:
  - Type revision request
  - `Ctrl+S`: Submit revision request
  - `ESC`: Go back to action selection

### 3. Integration with Plan Generation Flow

The updated plan generation flow:

1. User opens plan generation modal (press `g`)
2. User fills in description, constraints, granularity
3. User submits form
4. TUI calls agent to generate plan
5. **If agent returns questions:**
   - TUI opens agent question modal
   - User answers questions
   - Process continues
6. **If agent returns plan:**
   - TUI opens plan review modal (NEW)
   - User chooses Accept/Revise/Reject
7. **If Accept:**
   - Plan is saved to disk
   - Plan is loaded into TUI
   - Success message shown
8. **If Revise:**
   - Revision prompt is shown
   - User enters revision request
   - Agent is invoked with refine request
   - Process returns to step 6 (up to 1 revision)
9. **If Reject:**
   - Plan is discarded
   - Returns to normal view

### 4. State Management

New state additions to `Model`:
- `ActionModePlanReview`: Action mode when review modal is active
- `planReviewForm`: Stores the review form state

New form type:
- `PlanReviewForm`: Manages review modal state
  - `mode`: Current mode (choose action or revision prompt)
  - `plan`: The generated plan being reviewed
  - `selectedAction`: Currently selected action (0=Accept, 1=Revise, 2=Reject)
  - `revisionTextarea`: Textarea for revision request
  - `revisionCount`: Number of revisions performed (enforces limit)

### 5. Commands and Functions

New commands:
- `RefinePlanInMemory`: Refines existing plan with change request
- `SavePlanCmd`: Saves plan to disk asynchronously

New handlers:
- `HandlePlanReviewKey`: Handles keyboard input in review modal
- `acceptPlan`: Accepts and saves the plan
- `RenderPlanReviewModal`: Renders the review modal UI

### 6. Bottom Bar Updates

The bottom bar shows context-appropriate help text:
- **Choose action mode**: `[↑/↓]navigate [1-3]select [enter]confirm [esc]cancel [ctrl+c]quit`
- **Revision prompt mode**: `[ctrl+s]submit [esc]back [ctrl+c]quit`

### 7. Revision Limit

- Maximum of **1 revision** is allowed (matches CLI's `maxGenerateRevisions`)
- When limit is reached, the Revise option is disabled
- User can still Accept or Reject at any point

## Key Design Decisions

1. **Modal UI Pattern**: Following the existing pattern of agent questions modal, overwrite confirmation modal, and plan generation modal for consistency.

2. **Two-Step Revision Flow**: Separate the action selection from the revision prompt to provide clear UX and allow users to change their mind before submitting.

3. **Preview-Based Review**: Show plan summary and top-level features rather than full tree in the modal to keep it focused and readable.

4. **Deferred Plan Application**: The plan is NOT applied to the model until the user explicitly accepts it. This prevents premature changes and allows easy rejection.

5. **Revision Limit**: Enforced at the form level and in the key handler to prevent infinite revision loops.

6. **Integration with Questions Flow**: The review modal appears AFTER any agent questions are answered, maintaining the sequential flow.

## Testing

Comprehensive test coverage in `plan_review_modal_test.go`:
- Form creation and initialization
- Navigation (up/down, quick select)
- Revision limit enforcement
- Mode switching (action selection ↔ revision prompt)
- Accept action (plan saved and applied)
- Reject action (plan discarded)
- Revision prompt validation
- Escape handling (back from revision prompt)

All tests pass successfully.

## Usage Example

### Scenario: Generate and Accept Plan

1. Press `g` to open plan generation modal
2. Enter project description: "A task management web app"
3. Press Tab/Enter to submit
4. Agent generates plan
5. **Review modal appears:**
   - "Plan contains 15 items"
   - Top features: User Management, Task CRUD, Authentication, etc.
   - Options: 1. Accept | 2. Revise | 3. Reject
6. Press `1` or `Enter` to Accept
7. Plan is saved and loaded into TUI

### Scenario: Generate, Revise, and Accept

1. Press `g` to open plan generation modal
2. Enter project description: "A blog platform"
3. Press Tab/Enter to submit
4. Agent generates plan
5. **Review modal appears with plan summary**
6. Press `2` to select Revise
7. Press `Enter` to confirm
8. **Revision prompt appears**
9. Type: "Add a commenting system"
10. Press `Ctrl+S` to submit
11. Agent refines the plan
12. **Review modal appears again with updated plan**
13. Press `1` to Accept
14. Plan is saved and loaded into TUI

### Scenario: Generate and Reject

1. Press `g` to open plan generation modal
2. Enter project description: "An e-commerce site"
3. Press Tab/Enter to submit
4. Agent generates plan
5. **Review modal appears with plan summary**
6. Press `3` to select Reject
7. Press `Enter` to confirm
8. Plan is discarded, message shown: "Plan generation cancelled"
9. Returns to main TUI view

## File Changes

### New Files:
- `internal/tui/plan_review_modal.go`: Review modal implementation
- `internal/tui/plan_review_modal_test.go`: Comprehensive tests
- `docs/PLAN_REVIEW_FLOW.md`: This documentation

### Modified Files:
- `internal/tui/model.go`: Added review modal state and handlers
- `internal/tui/bottom_bar.go`: Added review mode help text
- `internal/tui/plan_generate_modal_test.go`: Updated test expectations

## Acceptance Criteria Met

✅ After agent returns plan, display summary and tree preview
✅ Show action choices: Accept, Revise, Reject
✅ Accept option saves plan and returns to normal TUI view
✅ Revise option prompts for revision request and re-runs agent (1 revision allowed)
✅ Reject option discards plan and returns to normal view
✅ Plan is displayed in a scrollable view or separate screen (modal with summary)

## Future Enhancements

Possible improvements:
- Show full plan tree in a scrollable view within the modal
- Allow multiple revisions with configurable limit
- Show diff between original and revised plans
- Allow editing plan summary/metadata before accepting
- Save rejected plans to history for reference
- Add keyboard shortcut to view full plan in detail pane before accepting
