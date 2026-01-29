# Plan Generation Modal - Bugs and Fixes

**Analysis Date:** 2026-01-28

---

## Summary

During comprehensive testing of the plan generation modal feature, several minor issues were identified. This document catalogs each issue, its impact, and recommended fixes.

---

## Bug #1: Window Resize Not Propagated to Modal Forms

### Status: ⚠️ MINOR

### Description
When a window resize event occurs while a modal is open, the model's `windowWidth` and `windowHeight` are updated, but the modal form's `SetSize()` method is not called to adjust the form's internal dimensions.

### Impact
- Low severity
- Modal may not optimally use available space after resize
- Visual elements may not be properly centered
- Functionally works but suboptimal UX

### Location
`internal/tui/model.go:100-106`

### Current Code
```go
case tea.WindowSizeMsg:
    m.windowWidth = typed.Width
    m.windowHeight = typed.Height
    return m, nil
```

### Recommended Fix
```go
case tea.WindowSizeMsg:
    m.windowWidth = typed.Width
    m.windowHeight = typed.Height

    // Update active modal forms with new dimensions
    if m.planGenerateForm != nil {
        m.planGenerateForm.SetSize(typed.Width, typed.Height)
    }
    if m.agentQuestionForm != nil {
        m.agentQuestionForm.SetSize(typed.Width, typed.Height)
    }
    if m.planReviewForm != nil {
        m.planReviewForm.SetSize(typed.Width, typed.Height)
    }

    return m, nil
```

### Test Case
Add test to verify SetSize is called on resize:
```go
func TestModel_WindowResizeUpdatesModalForms(t *testing.T) {
    m := NewModel(plan.NewEmptyWorkGraph())
    form := NewPlanGenerateForm()
    form.SetSize(80, 24)
    m.planGenerateForm = &form
    m.actionMode = ActionModeGeneratePlan

    // Resize window
    msg := tea.WindowSizeMsg{Width: 120, Height: 40}
    m, _ = m.Update(msg)

    // Verify form dimensions updated
    if m.planGenerateForm.width != 120 || m.planGenerateForm.height != 40 {
        t.Error("Expected form dimensions to be updated")
    }
}
```

### Priority
Low - works in practice due to re-rendering

---

## Bug #2: Incomplete Test Coverage for focusPrev()

### Status: 📝 TEST GAP

### Description
The `focusPrev()` function in `plan_generate_modal.go` has only 50% test coverage. The Submit → Granularity backward transition is not explicitly tested.

### Impact
- No functional impact (code works correctly)
- Testing gap could hide future regressions

### Location
`internal/tui/plan_generate_modal.go:145-165`

### Current Test Coverage
```
focusPrev: 50.0%
```

### Recommended Fix
Add explicit test for backward navigation from Submit:

```go
func TestPlanGenerateForm_FocusPrevFromSubmit(t *testing.T) {
    form := NewPlanGenerateForm()

    // Navigate to Submit
    form.focusedField = FieldSubmit

    // Move backward
    form = form.focusPrev()

    if form.focusedField != FieldGranularity {
        t.Errorf("Expected FieldGranularity, got %v", form.focusedField)
    }

    // Verify Granularity is focused
    if !form.granularity.Focused() {
        t.Error("Expected granularity textinput to be focused")
    }
}
```

### Priority
Low - cosmetic test improvement

---

## Bug #3: pendingPlanRequest Not Reset on New Generation

### Status: 🤔 BY DESIGN?

### Description
When opening the plan generation modal with 'g', the `pendingPlanRequest` from a previous session is not cleared. This could potentially cause confusion if the user:
1. Starts plan generation
2. Answers some questions
3. Cancels (ESC)
4. Reopens modal (g)
5. The old request data still exists

### Impact
- Low severity
- May cause unexpected behavior if implementation changes
- Could accumulate state across sessions

### Location
`internal/tui/model.go:363-378`

### Current Code
```go
case "g":
    if m.actionMode != ActionModeNone || m.actionInProgress {
        return m, nil
    }
    // Check if plan already exists with items
    if len(m.plan.Items) > 0 {
        m.actionMode = ActionModeConfirmOverwrite
        return m, nil
    }
    form := NewPlanGenerateForm()
    form.SetSize(m.windowWidth, m.windowHeight)
    m.planGenerateForm = &form
    m.actionMode = ActionModeGeneratePlan
    return m, nil
```

### Recommended Fix (Option 1: Reset Always)
```go
case "g":
    if m.actionMode != ActionModeNone || m.actionInProgress {
        return m, nil
    }

    // Reset pending request state for new generation
    m.pendingPlanRequest = PendingPlanRequest{}

    // Check if plan already exists with items
    if len(m.plan.Items) > 0 {
        m.actionMode = ActionModeConfirmOverwrite
        return m, nil
    }
    form := NewPlanGenerateForm()
    form.SetSize(m.windowWidth, m.windowHeight)
    m.planGenerateForm = &form
    m.actionMode = ActionModeGeneratePlan
    return m, nil
```

### Recommended Fix (Option 2: Document Behavior)
If the persistence is intentional (for resumption), add a comment:
```go
case "g":
    // Note: pendingPlanRequest is intentionally preserved
    // to allow resumption if generation was previously cancelled
    if m.actionMode != ActionModeNone || m.actionInProgress {
        return m, nil
    }
    // ...
```

### Priority
Low - document or fix for clarity

---

## Bug #4: Agent Question Modal Doesn't Auto-Select First Option

### Status: 📝 OBSERVATION

### Description
When a multiple-choice question is first displayed, `selectedOption` is initialized to `-1`, meaning no option is selected by default. Users must explicitly press a number key or arrow down before selecting.

### Impact
- Low severity
- Slightly inconvenient UX (requires extra keypress)
- Could be confusing if user presses Enter immediately

### Location
`internal/tui/agent_question_modal.go:48`

### Current Code
```go
return AgentQuestionForm{
    questions:      questions,
    currentIndex:   0,
    textInput:      ti,
    selectedOption: -1,  // No selection
    answers:        make([]agent.Answer, 0, len(questions)),
    width:          70,
    height:         25,
}
```

### Actual Behavior
The `moveToNextQuestion()` function (line 172) does auto-select the first option:
```go
if len(currentQ.Options) == 0 {
    f.textInput.Focus()
} else {
    f.textInput.Blur()
    f.selectedOption = 0  // Auto-select first option
}
```

However, this only happens after moving to the next question, not for the first question.

### Recommended Fix
Auto-select first option if the first question has options:
```go
func NewAgentQuestionForm(questions []agent.Question) AgentQuestionForm {
    if len(questions) == 0 {
        return AgentQuestionForm{}
    }

    ti := textinput.New()
    ti.Placeholder = "Enter your answer..."
    ti.CharLimit = 500
    ti.Width = 60

    // Determine initial state based on first question
    initialSelection := -1
    if len(questions[0].Options) == 0 {
        ti.Focus()
    } else {
        ti.Blur()
        initialSelection = 0  // Auto-select first option for multiple choice
    }

    return AgentQuestionForm{
        questions:      questions,
        currentIndex:   0,
        textInput:      ti,
        selectedOption: initialSelection,
        answers:        make([]agent.Answer, 0, len(questions)),
        width:          70,
        height:         25,
    }
}
```

### Test Case
```go
func TestAgentQuestionForm_FirstMultipleChoiceAutoSelected(t *testing.T) {
    questions := []agent.Question{
        {
            ID:      "q1",
            Prompt:  "Which framework?",
            Options: []string{"React", "Vue", "Angular"},
        },
    }

    form := NewAgentQuestionForm(questions)

    if form.selectedOption != 0 {
        t.Errorf("Expected first option to be auto-selected, got %d", form.selectedOption)
    }
}
```

### Priority
Low - UX improvement

---

## Bug #5: No Keyboard Shortcut to Submit Revision

### Status: 💡 FEATURE REQUEST

### Description
When in revision prompt mode, the user must press `Ctrl+S` to submit. There's no indication of this in the help text, and `Enter` alone inserts a newline (normal textarea behavior).

### Impact
- Low severity
- Discoverability issue
- Users might expect Enter to submit

### Location
`internal/tui/plan_review_modal.go:196-221`

### Current Behavior
- Enter: inserts newline in textarea
- Ctrl+S: submits revision
- ESC: cancels

### Recommended Fix (Option 1: Add Ctrl+Enter)
Allow `Ctrl+Enter` as an alternate submit shortcut:
```go
case "ctrl+enter", "ctrl+s":
    // Submit revision request
    revisionRequest := m.planReviewForm.GetRevisionRequest()
    // ...
```

Update help text:
```go
helpText := helpStyle.Render("[ctrl+enter or ctrl+s]submit [esc]back")
```

### Recommended Fix (Option 2: Better Documentation)
Ensure help text is clear:
```go
helpText := helpStyle.Render("[ctrl+s]submit revision [enter]newline [esc]back")
```

### Priority
Low - documentation/UX improvement

---

## Bug #6: Multiple Rapid 'g' Presses Could Queue Actions

### Status: ✅ ALREADY PREVENTED

### Description
If a user rapidly presses 'g' multiple times, could this cause multiple modals to open or actions to queue?

### Investigation
The code already prevents this:
```go
case "g":
    if m.actionMode != ActionModeNone || m.actionInProgress {
        return m, nil  // Ignored if modal open or action in progress
    }
```

### Status
✅ **No bug** - already properly handled

### Test Coverage
✅ Test exists: `TestModel_RapidKeyPressesWhileGenerating`

---

## Bug #7: Spinner Continues After Error

### Status: ✅ WORKING CORRECTLY

### Description
Does the spinner stop correctly when an error occurs during plan generation?

### Investigation
Code correctly stops spinner on error:
```go
case PlanGenerateInMemoryResult:
    m.actionInProgress = false  // Stops spinner
    m.actionName = ""
    if typed.Err != nil {
        m.actionOutput = &ActionOutput{
            Message: fmt.Sprintf("Plan generation failed: %v", typed.Err),
            IsError: true,
        }
    }
```

### Status
✅ **No bug** - working correctly

### Test Coverage
✅ Test exists: `TestModel_HandlePlanGenerationError`

---

## Summary Table

| Bug # | Severity | Status | Priority | Fix Effort |
|-------|----------|--------|----------|------------|
| 1 | Minor | New | Low | 10 min |
| 2 | Test Gap | New | Low | 5 min |
| 3 | Observation | New | Low | 2 min |
| 4 | UX Issue | New | Low | 10 min |
| 5 | Feature | New | Low | 15 min |
| 6 | N/A | Already Fixed | - | - |
| 7 | N/A | Already Fixed | - | - |

---

## Recommendations

### For Immediate Release
✅ No critical bugs - safe to deploy

### For Next Sprint
1. Fix Bug #1 (window resize) - 10 min
2. Fix Bug #2 (test coverage) - 5 min
3. Fix Bug #4 (auto-select) - 10 min
4. Document Bug #3 behavior - 2 min
5. Improve Bug #5 (keyboard shortcuts) - 15 min

**Total estimated effort:** ~45 minutes

---

## Verification Plan

After applying fixes:

1. **Run all tests**
   ```bash
   go test ./internal/tui/... -v
   ```

2. **Verify coverage improvements**
   ```bash
   go test ./internal/tui/... -coverprofile=coverage.out
   go tool cover -func=coverage.out | grep focusPrev
   ```

3. **Manual testing**
   - Open modal and resize window
   - Test multiple choice auto-selection
   - Test revision submission shortcuts

4. **Regression check**
   - Verify all original functionality still works
   - Verify no new issues introduced

---

## Change History

### 2026-01-28
- Initial bug analysis
- 5 minor issues identified
- 2 items verified as working correctly
- No critical bugs found
- Fixes recommended for next sprint
