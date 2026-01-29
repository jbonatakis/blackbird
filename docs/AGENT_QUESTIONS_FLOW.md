# Agent Question Handling Flow

## High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Actions                              │
└─────────────────────────────────────────────────────────────────┘
                                │
                                ▼
                    ┌─────────────────────┐
                    │  Press 'g' to open  │
                    │  plan generation    │
                    │      modal          │
                    └──────────┬──────────┘
                                │
                                ▼
                    ┌─────────────────────┐
                    │  Fill description,  │
                    │  constraints, etc.  │
                    └──────────┬──────────┘
                                │
                                ▼
                    ┌─────────────────────┐
                    │  Submit form with   │
                    │     Enter key       │
                    └──────────┬──────────┘
                                │
                                ▼
┌───────────────────────────────────────────────────────────────────┐
│                    GeneratePlanInMemory                           │
│  • Call agent with description, constraints, granularity          │
│  • Store request in pendingPlanRequest                            │
│  • Set questionRound = 0                                          │
└───────────────────────┬───────────────────────────────────────────┘
                        │
                        ▼
            ┌───────────────────────┐
            │  Agent returns...     │
            └─────────┬─────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
        ▼             ▼             ▼
   ┌────────┐   ┌──────────┐   ┌──────┐
   │ Error  │   │Questions │   │ Plan │
   └────┬───┘   └────┬─────┘   └───┬──┘
        │            │              │
        ▼            ▼              ▼
 ┌──────────┐  ┌──────────┐  ┌──────────┐
 │Show error│  │   Open   │  │  Update  │
 │  modal   │  │ question │  │   plan   │
 └──────────┘  │  modal   │  │   and    │
               └────┬─────┘  │  show    │
                    │        │ success  │
                    │        └──────────┘
                    ▼
         ┌─────────────────────┐
         │  Show question 1/N  │
         │  with prompt and    │
         │  options (if any)   │
         └──────────┬──────────┘
                    │
                    ▼
         ┌─────────────────────┐
         │  User answers:      │
         │  • Type text, or    │
         │  • Select option    │
         │  • Press Enter      │
         └──────────┬──────────┘
                    │
                    ▼
         ┌─────────────────────┐
         │ More questions?     │
         └──────┬──────────────┘
                │
      ┌─────────┼─────────┐
      │                   │
     Yes                 No
      │                   │
      ▼                   ▼
 ┌─────────┐      ┌──────────────┐
 │ Next    │      │All questions │
 │question │      │answered      │
 └────┬────┘      └──────┬───────┘
      │                  │
      └──────────────────┘
                │
                ▼
  ┌──────────────────────────────┐
  │ ContinuePlanGenerationWith   │
  │         Answers              │
  │ • Re-invoke agent with same  │
  │   description/constraints    │
  │ • Include collected answers  │
  │ • Increment questionRound    │
  └────────────┬─────────────────┘
               │
               ▼
       ┌───────────────┐
       │ Check round   │
       │    limit      │
       └───────┬───────┘
               │
    ┌──────────┼──────────┐
    │                     │
 < limit            >= limit
    │                     │
    ▼                     ▼
┌────────┐        ┌──────────────┐
│ Agent  │        │Show "too many│
│returns │        │clarification │
│result  │        │rounds" error │
└───┬────┘        └──────────────┘
    │
    └──► Repeat from "Agent returns..."
```

## State Tracking

### Model State Fields

```go
type Model struct {
    // ... other fields ...

    // Question modal state
    agentQuestionForm  *AgentQuestionForm   // Current question form
    pendingPlanRequest PendingPlanRequest   // Original request parameters
}

type PendingPlanRequest struct {
    description   string      // Original project description
    constraints   []string    // Original constraints
    granularity   string      // Original granularity
    questionRound int         // Current question round (0-indexed)
}
```

### AgentQuestionForm State

```go
type AgentQuestionForm struct {
    questions      []agent.Question  // All questions to answer
    currentIndex   int               // Current question (0-indexed)
    textInput      textinput.Model   // For free-text answers
    selectedOption int               // For multiple choice (0-indexed, -1 = none)
    answers        []agent.Answer    // Collected answers so far
}
```

## Key Components

### 1. Question Types

#### Multiple Choice
```
Question 1 of 3

Which framework do you prefer?

1) React
2) Vue      ← Selected
3) Angular

↑/↓ or k/j: navigate • 1-9: quick select • Enter: confirm • ESC: cancel
```

#### Free Text
```
Question 2 of 3

What is your project name?

┌────────────────────────────────┐
│ MyAwesomeProject_              │
└────────────────────────────────┘

Enter: submit answer • ESC: cancel
```

### 2. Completion State
```
All questions answered

Press Enter to continue generating the plan.

Enter: continue • ESC: cancel
```

## Round Limit Protection

```go
const maxAgentQuestionRounds = 2

func ContinuePlanGenerationWithAnswers(..., questionRound int) tea.Cmd {
    if questionRound >= maxAgentQuestionRounds {
        return errorMsg("too many clarification rounds")
    }
    // ... continue with agent call
}
```

This prevents infinite loops where the agent keeps asking questions indefinitely.

## Integration Points

### 1. Plan Generation Modal → Agent Question Modal
When form is submitted in plan generation modal:
- Store `PendingPlanRequest` with description, constraints, granularity
- Set `questionRound = 0`
- Call `GeneratePlanInMemory`

### 2. Agent Response → Question Modal
When agent returns `PlanGenerateInMemoryResult`:
```go
if len(typed.Questions) > 0 {
    // Open question modal
    form := NewAgentQuestionForm(typed.Questions)
    m.agentQuestionForm = &form
    m.actionMode = ActionModeAgentQuestion
}
```

### 3. Question Modal → Continued Generation
When all questions answered:
```go
answers := m.agentQuestionForm.GetAnswers()
ContinuePlanGenerationWithAnswers(
    m.pendingPlanRequest.description,
    m.pendingPlanRequest.constraints,
    m.pendingPlanRequest.granularity,
    answers,
    m.pendingPlanRequest.questionRound + 1,
)
```

### 4. Success → Plan Update
When agent returns plan (no questions):
```go
if typed.Plan != nil {
    m.plan = *typed.Plan
    m.pendingPlanRequest = PendingPlanRequest{} // Clear
    showSuccess("Plan generated successfully")
}
```

## Error Handling

1. **Agent Error**: Show error modal with message
2. **Too Many Rounds**: Show error after exceeding `maxAgentQuestionRounds`
3. **Empty Answer**: Don't allow submission of empty text answers
4. **No Option Selected**: Don't allow submission without selecting an option
5. **Cancellation**: User can press ESC at any time to cancel and return to main view

## Testing Coverage

- ✅ Free text question answering
- ✅ Multiple choice question answering
- ✅ Multi-question sequences
- ✅ Option navigation (arrows, number keys)
- ✅ Empty answer validation
- ✅ Form completion state
- ✅ Answer collection and retrieval
