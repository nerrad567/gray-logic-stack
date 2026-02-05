---
description: Check documentation is up to date with code
---

# Documentation Sync Check

Verifies documentation matches the current codebase state.

> **IMPORTANT — Shell Execution:** All bash code blocks in this file use bash-specific syntax (`[[`, `=~`, `echo -e`, `$(())`). When executing these snippets, **always wrap them with `bash -c '...'`** because the tool's default shell is `/bin/sh` (dash on Debian), which does not support these features.

## What It Checks

### 1. Package Documentation
```bash
# Find packages missing doc.go
find code/core/internal -type d -exec sh -c '
  if ls "$1"/*.go >/dev/null 2>&1 && [ ! -f "$1/doc.go" ]; then
    echo "Missing doc.go: $1"
  fi
' _ {} \;
```

### 2. Version Consistency
Check these files have matching Go version:
- `code/core/go.mod`
- `code/core/docs/GETTING-STARTED.md`
- `.claude/commands/health-check.md`

```bash
# Current Go version
go version | awk '{print $3}'

# Check go.mod
grep "^go " code/core/go.mod
```

### 3. Exported Functions Without Comments
```bash
cd code/core
# Find exported functions missing doc comments
grep -rn "^func [A-Z]" --include="*.go" internal/ | grep -v "_test.go" | while IFS= read -r line; do
  file=$(echo "$line" | cut -d: -f1)
  linenum=$(echo "$line" | cut -d: -f2)
  prevline=$((linenum - 1))
  prev=$(sed -n "${prevline}p" "$file")
  if [[ ! "$prev" =~ ^// ]]; then
    echo "Missing doc: $line"
  fi
done
```

### 4. PROJECT-STATUS.md Task Accuracy
Claude should verify:
- Milestone tasks marked complete actually exist in code
- Progress checklists match completed tasks
- "RESUME HERE" section is current

### 5. Stale Code Examples
Check markdown files for code blocks that may be outdated:
```bash
# Find Go code blocks in docs
grep -l '```go' docs/**/*.md code/core/docs/*.md 2>/dev/null
```
For each file with Go examples, verify the examples compile or match current API.

### 6. Cross-Reference Validation
Check that file links in markdown actually exist:
```bash
# Find broken file links
grep -roh '\[.*\](file://[^)]*' docs/ | while read link; do
  path=$(echo "$link" | sed 's/.*file:\/\///' | cut -d')' -f1)
  if [ ! -f "$path" ]; then
    echo "Broken link: $path"
  fi
done
```

### 7. Docs Freshness Check (Code vs Docs)

Compare when code packages were last modified vs their documentation:

```bash
cd /home/graylogic-dev/gray-logic-stack

# For each package, compare code vs doc modification
for pkg in config database mqtt victoriametrics logging; do
  code_dir="code/core/internal/infrastructure/$pkg"
  doc_file="code/core/docs/technical/packages/$pkg.md"
  
  if [ -d "$code_dir" ] && [ -f "$doc_file" ]; then
    code_date=$(git log -1 --format="%ai" -- "$code_dir" 2>/dev/null | cut -d' ' -f1)
    doc_date=$(git log -1 --format="%ai" -- "$doc_file" 2>/dev/null | cut -d' ' -f1)
    
    if [[ "$code_date" > "$doc_date" ]]; then
      echo "STALE: $pkg docs ($doc_date) older than code ($code_date)"
    else
      echo "OK: $pkg docs up to date"
    fi
  elif [ -d "$code_dir" ]; then
    echo "MISSING: $doc_file"
  fi
done
```

**What triggers staleness:**
- Code file modified after its corresponding doc
- New files added without doc updates
- API changes not reflected in examples

### 8. Git-Based Change Tracking

Check what code changed since docs were last updated:

```bash
# Find code files changed since PROJECT-STATUS.md was last updated
status_commit=$(git log -1 --format="%H" -- PROJECT-STATUS.md)
echo "Changes since PROJECT-STATUS.md update:"
git diff --name-only $status_commit..HEAD -- code/core/internal/
```

```bash
# Files modified today that may need doc updates
echo "Code modified today:"
find code/core/internal -name "*.go" -mtime 0 -type f
```

### 9. API Signature Accuracy

When docs are flagged as stale, Claude should verify that code examples match actual function signatures:

```bash
# Check key function signatures in infrastructure packages
cd code/core
grep -n "^func.*Open\|^func.*Connect\|^func.*New" internal/infrastructure/**/*.go 2>/dev/null | grep -v "_test.go"
```

**Common drift patterns to check:**
- Functions now require `context.Context` as first parameter
- New optional methods (e.g., `SetLogger()`, `SetOnError()`)
- Changed error types or new sentinel errors
- New configuration fields or validation rules
- Shutdown order changes

**For each stale package doc:**
1. Compare function signatures in doc examples vs actual code
2. Check if new public methods are documented in thread-safety table
3. Verify error handling examples use current error types
4. Update "Known Limitations" if issues were fixed

## Quick Check (Automated)

```bash
cd /home/graylogic-dev/gray-logic-stack/code/core

# 1. Packages without doc.go
echo "=== Missing doc.go ==="
for d in $(find internal -type d); do
  if ls "$d"/*.go >/dev/null 2>&1 && [ ! -f "$d/doc.go" ]; then
    echo "  $d"
  fi
done

# 2. Go version check
echo -e "\n=== Go Version Check ==="
echo "Installed: $(go version | awk '{print $3}')"
echo "go.mod:    go $(grep '^go ' go.mod | awk '{print $2}')"

# 3. Undocumented exports
echo -e "\n=== Undocumented Exports ==="
grep -rn "^func [A-Z]" --include="*.go" internal/ | grep -v "_test.go" | while IFS= read -r line; do
  file=$(echo "$line" | cut -d: -f1)
  linenum=$(echo "$line" | cut -d: -f2)
  prevline=$((linenum - 1))
  prev=$(sed -n "${prevline}p" "$file")
  if [[ ! "$prev" =~ ^// ]]; then
    echo "  $line"
  fi
done
```

## What Claude Should Do

When `/docs-sync` is invoked:

1. **Run the Quick Check** commands above
2. **Report findings** in categories:
   - 🔴 Critical: Missing doc.go in production packages
   - 🟡 Warning: Undocumented exported functions
   - 🔵 Info: Stale version references
3. **Offer to fix** issues found:
   - Create missing doc.go files
   - Update version numbers
   - Add function doc comments

### 10. README.md Freshness

The root README.md is the project's public face. It must reflect current capabilities.

```bash
cd /home/graylogic-dev/gray-logic-stack

echo "=== README.md Freshness ==="
readme_commit=$(git log -1 --format="%H %ai" -- README.md)
echo "Last README update: $readme_commit"

echo -e "\nCode changes since README update:"
readme_hash=$(echo "$readme_commit" | cut -d' ' -f1)
git log --oneline "$readme_hash"..HEAD -- code/ 2>/dev/null | head -20

echo -e "\nMilestones mentioned in README:"
grep -i "complete\|in progress\|planned" README.md | head -20
```

**What to check:**
- Does the "What's Working" table list all completed milestones?
- Are component statuses accurate (✅ Complete vs 🔄 In Progress)?
- Do architecture diagrams match the current system?
- Are example commands/configs still valid?
- Is the tech stack table current?

**When stale, Claude should offer to update:**
- Component status table
- Architecture description
- Code examples and paths
- Any "next steps" or roadmap sections

### 11. Session Log Updates (CHANGELOG, PROJECT-STATUS)

After any significant coding work, verify that high-level project logs reflect what was done:

#### CHANGELOG.md
```bash
cd /home/graylogic-dev/gray-logic-stack

# What's changed since the last CHANGELOG entry?
last_changelog_commit=$(git log -1 --format="%H" -- CHANGELOG.md)
echo "=== Code changes since last CHANGELOG update ==="
git log --oneline "$last_changelog_commit"..HEAD -- code/ 2>/dev/null | head -20

# Are there uncommitted changes not in CHANGELOG?
echo -e "\n=== Uncommitted code files ==="
git diff --name-only -- code/
git diff --cached --name-only -- code/
```

**What to check:**
- New features, bug fixes, or structural changes need a CHANGELOG entry
- Entry format: `## X.Y.Z – Description (YYYY-MM-DD)`
- Group changes under: Added, Changed, Fixed, Removed

#### PROJECT-STATUS.md
```bash
cd /home/graylogic-dev/gray-logic-stack

# Compare claimed milestone status vs actual code
echo "=== PROJECT-STATUS.md current phase ==="
grep -A2 "Current Phase\|Current Milestone\|Active Work" PROJECT-STATUS.md | head -10

echo -e "\n=== Recent code commits ==="
git log --oneline -10 -- code/
```

**What to check:**
- Does the "Current Phase/Milestone" match what we just completed?
- Are completion percentages accurate?
- Is the "Next" section pointing at the right work?

#### Claude's Session Log Checklist

When checking session logs, Claude should answer:
1. ❓ Were any milestone tasks completed this session? → Update PROJECT-STATUS.md
2. ❓ Were features added, bugs fixed, or breaking changes made? → Update CHANGELOG.md
3. ❓ Were new files created or significant code written? → Update PROJECT-STATUS.md (Session Log)
4. ❓ Is "RESUME HERE" pointing at the correct next step? → Update PROJECT-STATUS.md

If ANY answer is "yes but not yet updated", Claude should offer to update the file.

## When to Run

- **After completing any milestone** (mandatory)
- **After any significant coding session** (strongly recommended)
- Before creating a PR
- When resuming work after a break
- Weekly during active development

## Auto-Trigger Behavior

Claude should **automatically offer to run `/docs-sync`** (specifically the Session Log checks) when:
- A milestone or sub-milestone is completed
- Multiple files have been created or significantly modified
- Tests pass after a feature implementation
- The user says they're "done" or "wrapping up"

Claude should ask: *"Session work looks complete — shall I run `/docs-sync` to update project logs?"*

## Related Commands

- `/code-audit` — Full 7-phase quality check (includes docs in Phase 7)
- `/pre-commit` — Quick lint/test before commits
