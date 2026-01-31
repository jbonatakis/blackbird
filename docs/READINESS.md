# Readiness rules

- **Dependencies** are satisfied when all deps have status `done`.
- A **task is actionable** when status is `todo` and deps are satisfied.
- `blocked` is a manual override even if deps are satisfied.
