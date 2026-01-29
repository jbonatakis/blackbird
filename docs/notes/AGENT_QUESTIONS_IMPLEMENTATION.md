# Agent Clarifying Questions - TUI Implementation

## Overview

This implementation adds support for handling agent clarifying questions during plan generation in the TUI. When an agent returns questions instead of a plan, the TUI now displays them in an interactive modal dialog that allows users to answer and continue the generation process.

## Features Implemented

### 1. Agent Question Modal (`agent_question_modal.go`)

A new modal component that:
- Displays agent questions one at a time
- Supports two question types:
  - **Multiple choice**: Shows numbered options, user can navigate with arrow keys or press 1-9 to quick-select
  - **Free text**: Shows a text input field for the user to type their answer
- Shows progress indicator (Question X of Y)
- Validates answers before allowing submission
- Collects all answers before continuing plan generation

### 2. Question Navigation

**Multiple Choice Questions:**
- `↑/↓` or `k/j`: Navigate between options
- `1-9`: Quick select option by number
- `Enter`: Confirm selected option

**Free Text Questions:**
- Type answer directly
- `Enter`: Submit answer (only if non-empty)

**General:**
- `ESC`: Cancel and return to main view
- Questions are answered sequentially, one at a time

### 3. Integration with Plan Generation Flow

The plan generation flow now works as follows:

1. User opens plan generation modal (press `g`)
2. User fills in description, constraints, granularity
3. User submits form
4. TUI calls agent to generate plan
5. **If agent returns questions:**
   - TUI stores the original request parameters
   - TUI opens agent question modal
   - User answers questions sequentially
   - TUI re-invokes agent with answers
   - Process repeats up to `maxAgentQuestionRounds` (default: 2)
6. **If agent returns plan:**
   - TUI updates the plan and shows success message

### 4. State Management

New state fields added to `Model`:
- `agentQuestionForm`: Tracks the current question form state
- `pendingPlanRequest`: Stores original request parameters for question rounds
  - `description`, `constraints`, `granularity`
  - `questionRound`: Tracks how many question rounds have occurred

New action mode:
- `ActionModeAgentQuestion`: Active when question modal is displayed

### 5. Commands and Messages

New command:
- `ContinuePlanGenerationWithAnswers`: Continues plan generation with user's answers

Updated message handler:
- `PlanGenerateInMemoryResult`: Now detects questions and opens question modal

### 6. Bottom Bar Updates

The bottom bar now shows context-appropriate help text when in question mode:
- Multiple choice: `[↑/↓]navigate [1-9]select [enter]confirm [esc]cancel [q]uit`
- Free text: `[enter]submit [esc]cancel [q]uit`

## Testing

Comprehensive test coverage in `agent_question_modal_test.go`:
- Free text question handling
- Multiple choice question handling
- Multi-question sequences
- Option navigation
- Empty answer validation
- Form completion state

All tests pass successfully.

## Usage Example

1. Press `g` to open plan generation modal
2. Enter project description: "A web application for task management"
3. Press Tab to move through fields, Enter to submit
4. Agent asks: "Which database would you prefer?"
   - Options: 1) PostgreSQL  2) MySQL  3) SQLite
5. Press `2` or navigate with arrows and press Enter
6. Agent asks: "Should we include user authentication?"
7. Type "Yes, using JWT tokens" and press Enter
8. Agent generates plan with your answers incorporated
9. Plan is displayed in the TUI

## File Changes

### New Files:
- `internal/tui/agent_question_modal.go`: Question modal implementation
- `internal/tui/agent_question_modal_test.go`: Comprehensive tests

### Modified Files:
- `internal/tui/model.go`: Added question form state and action mode
- `internal/tui/plan_generate_modal.go`: Store pending request parameters
- `internal/tui/action_wrappers.go`: Added continue command with answers
- `internal/tui/bottom_bar.go`: Added question mode help text

## Design Decisions

1. **Sequential Questions**: Questions are presented one at a time rather than all at once. This matches the CLI behavior and provides a cleaner, more focused UX.

2. **Question Round Limit**: Enforces a maximum of 2 question rounds (configurable via `maxAgentQuestionRounds`) to prevent infinite loops if the agent keeps asking questions.

3. **Auto-Select First Option**: For multiple choice questions, the first option is auto-selected when moving to a new question, allowing quick confirmation with just Enter.

4. **Stored Request Context**: The original plan generation parameters (description, constraints, granularity) are stored in `pendingPlanRequest` so they can be reused when re-invoking the agent with answers.

5. **Validation**: Empty text answers are not accepted - user must provide a non-empty value before the form allows advancement.

## Acceptance Criteria Met

✅ When agent returns questions, display them in a new modal dialog
✅ Modal shows question text and options (if provided)
✅ User can input answers (text or select from options)
✅ Answers are sent back to agent to continue generation
✅ Multiple question rounds are supported (up to maxAgentQuestionRounds)

## Future Enhancements

Possible improvements for future iterations:
- Allow editing previous answers before final submission
- Show a summary of all Q&A before continuing
- Support for more complex question types (multi-select, date pickers, etc.)
- Save Q&A history for reference
- Allow configuring maxAgentQuestionRounds via environment variable
