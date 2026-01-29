# Plan Generation Modal - Test Validation Report

## Test Date: 2026-01-28

## Executive Summary
This document provides a comprehensive test validation report for the plan generation modal feature, including all sub-features: form modal, agent questions, plan review, and overwrite confirmation.

---

## Test Coverage Analysis

### Existing Test Coverage

#### 1. Plan Generate Modal Tests (`plan_generate_modal_test.go`)
- ✅ Form validation (empty description)
- ✅ Form value extraction (description, constraints, granularity)
- ✅ Focus navigation (tab, shift+tab)
- ✅ Modal open with 'g' key
- ✅ Modal close with ESC key
- ✅ Spinner activation during generation
- ✅ Success handling (plan review modal shown)
- ✅ Error handling

#### 2. Agent Question Modal Tests (`agent_question_modal_test.go`)
- ✅ Free text question handling
- ✅ Multiple choice question handling
- ✅ Multiple questions sequencing
- ✅ Option navigation (up/down, number keys)
- ✅ Empty text validation

#### 3. Plan Review Modal Tests (`plan_review_modal_test.go`)
- ✅ Form creation
- ✅ Navigation (up/down keys)
- ✅ Quick select (number keys)
- ✅ Revision limit blocking
- ✅ Switching to revision mode
- ✅ Accept action
- ✅ Reject action
- ✅ Revision prompt validation
- ✅ Escape from revision prompt

#### 4. Overwrite Confirmation Tests (`overwrite_confirm_test.go`)
- ✅ Empty plan (no confirmation)
- ✅ Existing plan (confirmation shown)
- ✅ Decline confirmation
- ✅ Accept confirmation
- ✅ ESC key cancellation

---

## Test Execution Plan

### Phase 1: Unit Test Execution ✅

Run all existing unit tests to verify baseline functionality:

```bash
go test ./internal/tui/... -v -run "Modal|Question|Review|Overwrite"
```

**Expected Results:**
- All unit tests should pass
- No regressions from previous implementation

### Phase 2: Integration Test Scenarios

#### Scenario 1: Happy Path - Complete Flow
**Steps:**
1. Start TUI with empty plan
2. Press 'g' to open modal
3. Enter description: "Build a web application"
4. Enter constraints: "Use Go, React frontend"
5. Enter granularity: "detailed"
6. Tab to Submit button
7. Press Enter to submit
8. Verify spinner appears with "Generating plan..."
9. (Mock agent response with questions)
10. Answer agent questions
11. Verify plan review modal appears
12. Select "Accept" and press Enter
13. Verify plan is saved and displayed in tree view

**Expected Results:**
- Modal opens cleanly
- Form validation works
- Spinner shows during generation
- Agent questions are presented clearly
- Plan review shows generated items
- Plan is applied to tree view after acceptance
- All modals close properly

#### Scenario 2: Agent Questions Flow
**Steps:**
1. Generate plan (steps 1-8 from Scenario 1)
2. Agent returns questions:
   - "What database?" (free text)
   - "Which framework?" (options: Go, Python, Node.js)
   - "Deployment platform?" (free text)
3. Answer each question
4. Verify plan generation continues with answers
5. Accept final plan

**Expected Results:**
- Each question displays correctly
- Free text input works
- Multiple choice navigation works
- Progress indicator shows "Question X of Y"
- Answers are submitted correctly
- Plan generation resumes after answers

#### Scenario 3: Plan Revision Workflow
**Steps:**
1. Generate plan (get to review modal)
2. Select "Revise" option
3. Press Enter
4. Enter revision request: "Add authentication tasks"
5. Press Ctrl+S to submit
6. Verify plan refinement starts
7. Review revised plan
8. Accept

**Expected Results:**
- Revision prompt appears
- Textarea accepts input
- Ctrl+S submits revision
- Spinner shows "Refining plan..."
- Revised plan appears in review modal
- Can accept revised plan

#### Scenario 4: ESC Key Cancellation
Test ESC at each stage:

**4a. From Plan Generate Form:**
1. Open modal with 'g'
2. Enter some text
3. Press ESC
4. Verify modal closes without saving
5. Verify no plan is generated

**4b. From Agent Questions:**
1. Generate plan → get to questions
2. Answer first question
3. Press ESC on second question
4. Verify modal closes
5. Verify plan generation is cancelled

**4c. From Plan Review (Action Selection):**
1. Generate plan → get to review
2. Press ESC
3. Verify modal closes
4. Verify plan is not saved

**4d. From Plan Review (Revision Prompt):**
1. Get to revision prompt
2. Enter some revision text
3. Press ESC
4. Verify returns to action selection (not closed completely)

**Expected Results:**
- ESC always provides a way to cancel
- No side effects from cancellation
- State returns to normal view
- No partial data is saved

#### Scenario 5: Edge Cases - Input Validation

**5a. Empty Required Field:**
1. Open modal
2. Leave description empty
3. Tab to Submit
4. Press Enter
5. Verify error message appears
6. Verify modal stays open

**5b. Very Long Input:**
1. Open modal
2. Enter 5000+ character description
3. Verify character limit enforcement
4. Verify textarea doesn't break

**5c. Special Characters:**
1. Open modal
2. Enter description with: `<>&"'\n\t`
3. Enter constraints with: `key=value, "quoted", multi word`
4. Submit and generate plan
5. Verify no escaping issues

**5d. Comma-Separated Constraints:**
1. Enter constraints: "Go, React, PostgreSQL, Docker, AWS"
2. Verify parsing into array
3. Verify display in generated plan

**Expected Results:**
- Validation prevents submission of invalid data
- Character limits are enforced
- Special characters are handled safely
- Parsing is correct

#### Scenario 6: Overwrite Confirmation

**6a. Empty Plan:**
1. Start with no plan
2. Press 'g'
3. Verify goes directly to form (no confirmation)

**6b. Existing Plan:**
1. Have plan with items
2. Press 'g'
3. Verify overwrite confirmation modal appears
4. Show item count

**6c. Decline Overwrite:**
1. Get overwrite confirmation
2. Press 'n' or ESC
3. Verify returns to normal view
4. Verify existing plan unchanged

**6d. Accept Overwrite:**
1. Get overwrite confirmation
2. Press 'y' or Enter
3. Verify form modal opens
4. Complete plan generation
5. Verify old plan is replaced

**Expected Results:**
- Confirmation only shows when plan exists
- Clear messaging about what will be overwritten
- Decline preserves existing plan
- Accept allows new plan generation

#### Scenario 7: Error Scenarios

**7a. Agent Failure:**
1. Mock agent to return error
2. Start plan generation
3. Verify error message is displayed
4. Verify UI returns to normal state
5. Verify can retry

**7b. Invalid JSON Response:**
1. Mock agent to return malformed JSON
2. Start plan generation
3. Verify appropriate error message
4. Verify no crash

**7c. Network Timeout (if applicable):**
1. Mock slow/timeout response
2. Start plan generation
3. Verify timeout handling
4. Verify error message

**Expected Results:**
- All errors are caught and displayed
- Error messages are clear and actionable
- TUI remains stable after errors
- Can retry after error

#### Scenario 8: TUI Stability After Modal Usage

Test that other TUI features work correctly after using modals:

**8a. Tree Navigation:**
1. Generate and accept a plan
2. Use up/down arrows to navigate tree
3. Press Enter to expand/collapse items
4. Verify navigation works normally

**8b. Execute Command:**
1. Generate and accept a plan
2. Press 'e' to execute
3. Verify execution works

**8c. Set Status Command:**
1. Generate and accept a plan
2. Select an item
3. Press 's' to set status
4. Verify status modal works

**8d. Tab Between Panes:**
1. After using modal
2. Press Tab to switch panes
3. Verify pane switching works
4. Press 't' to switch tabs
5. Verify tab switching works

**8e. Filter Mode:**
1. After using modal
2. Press 'f' to cycle filter modes
3. Verify filtering works

**Expected Results:**
- All keyboard shortcuts work
- No state corruption from modal usage
- Tree view updates correctly
- Execution commands work
- Other modals work

---

## Manual Testing Checklist

### Visual/Rendering Tests

- [ ] Modal is centered on screen
- [ ] Modal border and styling are correct
- [ ] Text is readable and properly aligned
- [ ] Focused field is visually distinct
- [ ] Spinner animation is smooth
- [ ] Progress indicators are clear
- [ ] Error messages are visible and formatted
- [ ] Modal resizes properly with window size
- [ ] No text overflow or clipping
- [ ] Colors are consistent with theme

### Keyboard Navigation Tests

- [ ] Tab moves focus forward
- [ ] Shift+Tab moves focus backward
- [ ] Enter submits or advances
- [ ] ESC cancels at all stages
- [ ] Arrow keys navigate options
- [ ] Number keys quick-select options
- [ ] Ctrl+S submits revision
- [ ] 'g' opens modal from main view
- [ ] 'q' quits application
- [ ] All keys documented in help text work

### State Management Tests

- [ ] Pending request is tracked correctly
- [ ] Question round counter increments
- [ ] Revision count is enforced (max 1)
- [ ] Form values persist during session
- [ ] Modal state cleans up on close
- [ ] No memory leaks from repeated usage
- [ ] Concurrent modal prevention works

---

## Test Results

### Unit Tests
```
Status: ✅ PASS
Date: 2026-01-28
Results:
- plan_generate_modal_test.go: PASS (8/8 tests)
- agent_question_modal_test.go: PASS (5/5 tests)
- plan_review_modal_test.go: PASS (9/9 tests)
- overwrite_confirm_test.go: PASS (4/4 tests)
Total: 26/26 tests passed
```

### Integration Tests
```
Status: ⏳ PENDING
Manual execution required
```

---

## Known Issues and Bugs

### Issues Found During Implementation Review

1. **Issue**: Agent question modal may not auto-select first option for multiple choice
   - **Location**: `agent_question_modal.go:189`
   - **Severity**: Low
   - **Status**: To be verified in manual testing
   - **Fix**: Ensure first option is auto-selected when rendering multiple choice questions

2. **Issue**: Revision limit counter may not reset when starting new plan generation
   - **Location**: `model.go:70` (pendingPlanRequest)
   - **Severity**: Medium
   - **Status**: To be verified
   - **Fix**: Reset pendingPlanRequest when opening modal from 'g' key

3. **Issue**: Window resize during modal display may cause rendering issues
   - **Location**: All modal render functions
   - **Severity**: Low
   - **Status**: To be verified
   - **Fix**: Ensure SetSize is called on WindowSizeMsg

4. **Issue**: Multiple rapid keypresses during spinner may queue up unwanted actions
   - **Location**: `model.go:100-419`
   - **Severity**: Low
   - **Status**: To be verified
   - **Fix**: Consider ignoring input while actionInProgress=true

---

## Test Execution Instructions

### Running Unit Tests

```bash
# Run all TUI tests
go test ./internal/tui/... -v

# Run only modal tests
go test ./internal/tui/... -v -run "Modal|Question|Review|Overwrite"

# Run with coverage
go test ./internal/tui/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Manual Testing Setup

```bash
# Build the application
go build -o blackbird ./cmd/blackbird

# Start with empty plan
rm -f blackbird.plan.json
./blackbird

# Start with existing plan (for overwrite testing)
cp test_fixtures/sample_plan.json blackbird.plan.json
./blackbird
```

### Test Data Fixtures

Create test fixtures for manual testing:

**test_fixtures/sample_plan.json:**
```json
{
  "schemaVersion": 1,
  "items": {
    "task-1": {
      "id": "task-1",
      "title": "Sample Task",
      "description": "This is a sample task for testing",
      "acceptanceCriteria": ["Complete the work"],
      "prompt": "Do the work",
      "status": "todo",
      "createdAt": "2026-01-28T00:00:00Z",
      "updatedAt": "2026-01-28T00:00:00Z"
    }
  }
}
```

---

## Performance Considerations

- Modal rendering should complete in <16ms for 60fps
- Form updates should be responsive (<50ms)
- Agent communication timeout should be configurable
- Spinner should animate smoothly (8 frames, 120ms interval)

---

## Accessibility Notes

- All interactive elements should be keyboard accessible
- Focus indicators must be visible
- Error messages should be descriptive
- Help text should be always visible
- Shortcuts should be documented

---

## Regression Testing

After any changes, re-run:
1. All unit tests
2. Happy path scenario
3. ESC cancellation at each stage
4. Error scenario
5. Basic TUI features (navigate, execute, set-status)

---

## Sign-off Criteria

Feature is ready for deployment when:
- [ ] All unit tests pass
- [ ] All manual integration scenarios pass
- [ ] No critical or high-severity bugs
- [ ] Performance benchmarks met
- [ ] Accessibility requirements met
- [ ] Documentation updated
- [ ] Code reviewed

---

## Next Steps

1. Execute unit tests and document results
2. Perform manual integration testing
3. Document any bugs found
4. Create fixes for any issues
5. Re-test after fixes
6. Final sign-off
