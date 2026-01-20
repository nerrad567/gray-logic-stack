---
description: Run lint, tests, and principles check before committing
---

# Pre-Commit Check

Runs all quality gates before committing code.

## What It Does

1. **Lint** — Run golangci-lint for code quality and security issues
2. **Test** — Run all unit tests with race detection
3. **Principles** — Check for principle violations (cloud deps, safety code)

## Commands Executed

```bash
cd code/core

# 1. Lint check
golangci-lint run

# 2. Run tests with race detection
go test -race -v ./...

# 3. Check for cloud/external dependencies in new code
git diff --cached --name-only | xargs grep -l -E "(http\.Get|http\.Post|cloud)" || echo "✓ No cloud calls"

# 4. Check for life safety control attempts
git diff --cached --name-only | xargs grep -l -E "fire.*control|alarm.*set|estop.*trigger" || echo "✓ No safety control"
```

## When to Use

Run before every commit:
```bash
# After staging your changes
git add .
# Then run /pre-commit
# Then commit if all checks pass
git commit -m "feat(scope): description"
```

## Example Output

```
✓ golangci-lint: 0 issues
✓ go test: 15 tests passed
✓ No cloud calls detected
✓ No safety control attempts
All checks passed — safe to commit.
```

## What Fails the Check

- Any golangci-lint error (not warnings)
- Any test failure
- External HTTP calls in core code
- Attempts to control life safety systems
