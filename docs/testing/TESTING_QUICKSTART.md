# Testing Quick Start Guide

Quick reference for running tests on the plan generation modal feature.

---

## TL;DR - Run Everything

```bash
# Run all TUI tests with coverage
go test ./internal/tui/... -coverprofile=coverage.out -v

# View coverage summary
go tool cover -func=coverage.out | tail -1

# Open coverage HTML report
go tool cover -html=coverage.out
```

**Expected Result:** All tests pass, ~51% overall coverage, ~73% on modal logic.

---

## Focused Test Runs

### Modal Tests Only (Fast - ~0.4s)
```bash
go test ./internal/tui/... -v -run "Modal|Question|Review|Overwrite"
```

### Edge Case Tests Only
```bash
go test ./internal/tui/... -v -run "Empty|Long|Special|Parse|Wrap|Rapid|Resize"
```

### Validation Tests Only
```bash
go test ./internal/tui/... -v -run "Validation"
```

### Specific Modal Tests
```bash
# Plan Generate Modal
go test ./internal/tui/... -v -run "PlanGenerate"

# Agent Questions Modal
go test ./internal/tui/... -v -run "AgentQuestion"

# Plan Review Modal
go test ./internal/tui/... -v -run "PlanReview"

# Overwrite Confirmation
go test ./internal/tui/... -v -run "Overwrite"
```

---

## Coverage Analysis

### Generate Coverage Report
```bash
go test ./internal/tui/... -coverprofile=coverage.out -covermode=atomic
```

### View Coverage by File
```bash
go tool cover -func=coverage.out | grep modal
```

### View Coverage HTML (Opens Browser)
```bash
go tool cover -html=coverage.out
```

### Coverage Expectations
- Overall TUI: ~51%
- Modal logic functions: 70-73%
- Rendering functions: 0% (expected)
- Critical paths: 100%

---

## Test File Locations

```
internal/tui/
├── plan_generate_modal_test.go          # Form, validation, navigation
├── plan_generate_modal_edgecases_test.go # Edge cases, special inputs
├── agent_question_modal_test.go         # Question handling, answers
├── plan_review_modal_test.go            # Review, accept, reject, revise
└── overwrite_confirm_test.go            # Overwrite confirmation
```

---

## Common Test Patterns

### Running a Single Test
```bash
go test ./internal/tui -v -run TestPlanGenerateForm_Validation
```

### Running Tests with Verbose Output
```bash
go test ./internal/tui/... -v
```

### Running Tests with Race Detection
```bash
go test ./internal/tui/... -race
```

### Running Tests Multiple Times (Catch Flaky Tests)
```bash
go test ./internal/tui/... -count=10
```

---

## Interpreting Results

### Success
```
PASS
ok  	github.com/jbonatakis/blackbird/internal/tui	0.396s
```

### Failure
```
--- FAIL: TestName (0.00s)
    file_test.go:123: error message here
FAIL
```

### Coverage Summary
```
ok  	package	0.396s	coverage: 51.1% of statements
```

---

## Manual Testing Checklist

After automated tests pass, perform quick manual validation:

### 1. Build the Application (1 min)
```bash
go build -o blackbird ./cmd/blackbird
```

### 2. Test Basic Flow (3 min)
```bash
# Start with clean plan
rm -f blackbird.plan.json
./blackbird

# In TUI:
# 1. Press 'g' → modal opens
# 2. Enter description: "Build a todo app"
# 3. Tab to Submit and press Enter
# 4. If agent asks questions, answer them
# 5. Review generated plan
# 6. Press '1' and Enter to accept
# 7. Verify plan shows in tree
# 8. Press 'q' to quit
```

### 3. Test ESC Cancellation (2 min)
```bash
./blackbird

# Test ESC at each stage:
# 1. Press 'g', type something, press ESC → returns to main view
# 2. Generate plan, when questions appear press ESC → cancelled
# 3. Generate plan, when review appears press ESC → cancelled
```

### 4. Test Overwrite Confirmation (2 min)
```bash
# Ensure plan exists from previous test
./blackbird

# 1. Press 'g' → confirmation modal appears
# 2. Press 'n' → returns to main view, plan unchanged
# 3. Press 'g' again → confirmation appears
# 4. Press 'y' → form opens
# 5. Press ESC → returns to main view
```

### 5. Test Error Handling (1 min)
```bash
./blackbird

# 1. Press 'g'
# 2. Tab to Submit WITHOUT entering description
# 3. Press Enter → error message shown, modal stays open
# 4. Enter description and submit → works
```

**Total Time: ~10 minutes**

---

## Troubleshooting

### Tests Fail to Compile
```bash
# Ensure dependencies are installed
go mod download
go mod tidy
```

### Coverage Report Not Generated
```bash
# Ensure coverage file is writable
rm -f coverage.out
go test ./internal/tui/... -coverprofile=coverage.out
```

### Can't Open Coverage HTML
```bash
# Manually open the file
open coverage.out  # macOS
xdg-open coverage.out  # Linux
start coverage.out  # Windows

# Or generate HTML file
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Tests Pass But Coverage is Low
This is expected for TUI code:
- Rendering functions are hard to test in unit tests
- Many functions are tested through integration
- Focus is on logic coverage (which is high ~73%)

### Test Timeout
```bash
# Increase timeout
go test ./internal/tui/... -timeout=5m
```

---

## CI/CD Integration

### GitHub Actions Example
```yaml
- name: Run TUI Tests
  run: go test ./internal/tui/... -coverprofile=coverage.out -v

- name: Upload Coverage
  uses: codecov/codecov-action@v3
  with:
    files: ./coverage.out
```

### Pre-Commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running TUI tests..."
go test ./internal/tui/... -v
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi
```

---

## Performance Benchmarking

### Run Benchmarks
```bash
go test ./internal/tui/... -bench=. -benchmem
```

### Profile CPU Usage
```bash
go test ./internal/tui/... -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Profile Memory Usage
```bash
go test ./internal/tui/... -memprofile=mem.prof
go tool pprof mem.prof
```

---

## Test Maintenance

### Adding New Tests

1. **Identify what to test:**
   - New functionality
   - Edge cases
   - Bug fixes

2. **Choose appropriate test file:**
   - Modal behavior → `*_modal_test.go`
   - Edge cases → `*_edgecases_test.go`
   - Integration → `model_test.go`

3. **Follow naming convention:**
   ```go
   func TestComponent_Behavior(t *testing.T) {
       // Test implementation
   }
   ```

4. **Run test in isolation:**
   ```bash
   go test ./internal/tui -v -run TestYourNewTest
   ```

### Updating Tests After Code Changes

1. Run full suite to find failures
2. Update test expectations
3. Ensure changes are intentional
4. Document behavior changes
5. Re-run full suite

---

## Quick Reference Card

| Task | Command |
|------|---------|
| Run all tests | `go test ./internal/tui/... -v` |
| Run modal tests | `go test ./internal/tui/... -v -run "Modal\|Question\|Review"` |
| Coverage report | `go test ./internal/tui/... -coverprofile=coverage.out` |
| View coverage | `go tool cover -html=coverage.out` |
| Run single test | `go test ./internal/tui -v -run TestName` |
| Race detection | `go test ./internal/tui/... -race` |
| Benchmarks | `go test ./internal/tui/... -bench=.` |

---

## Success Criteria

Tests should:
- ✅ Complete in < 1 second
- ✅ Pass 100% (39/39 tests)
- ✅ Coverage > 50% overall
- ✅ Coverage > 70% on modal logic
- ✅ No flaky tests
- ✅ No race conditions

---

## Related Documentation

- **TEST_VALIDATION_REPORT.md** - Comprehensive test plan
- **TEST_RESULTS.md** - Detailed results and findings
- **BUGS_AND_FIXES.md** - Issues and resolutions
- **TEST_COMPLETION_SUMMARY.md** - Executive summary

---

## Getting Help

### Test failures?
1. Read the error message carefully
2. Check if it's a known issue (see BUGS_AND_FIXES.md)
3. Run the specific test in isolation
4. Check recent code changes

### Coverage questions?
1. Focus on logic coverage, not rendering
2. Check coverage.html for visual gaps
3. Rendering functions are expected to be 0%

### Need to add tests?
1. Follow existing patterns in test files
2. Use descriptive test names
3. Test one thing per test
4. Include both positive and negative cases

---

**Last Updated:** 2026-01-28
