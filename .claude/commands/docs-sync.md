---
description: Check documentation is up to date with code
---

# Documentation Sync Check

Verifies documentation matches the current codebase state.

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
- `code/core/IMPLEMENTATION.md`
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
grep -rn "^func [A-Z]" --include="*.go" | while read line; do
  file=$(echo "$line" | cut -d: -f1)
  linenum=$(echo "$line" | cut -d: -f2)
  prevline=$((linenum - 1))
  prev=$(sed -n "${prevline}p" "$file")
  if [[ ! "$prev" =~ ^// ]]; then
    echo "Missing doc: $line"
  fi
done
```

### 4. IMPLEMENTATION.md Task Accuracy
Claude should verify:
- M1.1 tasks marked complete actually exist in code
- M1.2 progress percentage matches completed tasks
- "Resume Here" section is current

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
for pkg in config database mqtt influxdb logging; do
  code_dir="code/core/internal/infrastructure/$pkg"
  doc_file="code/core/docs/technical/packages/$pkg.md"
  
  if [ -d "$code_dir" ] && [ -f "$doc_file" ]; then
    code_date=$(git log -1 --format="%ai" -- "$code_dir" 2>/dev/null | cut -d' ' -f1)
    doc_date=$(git log -1 --format="%ai" -- "$doc_file" 2>/dev/null | cut -d' ' -f1)
    
    if [[ "$code_date" > "$doc_date" ]]; then
      echo "🔴 STALE: $pkg docs ($doc_date) older than code ($code_date)"
    else
      echo "✅ $pkg docs up to date"
    fi
  elif [ -d "$code_dir" ]; then
    echo "🔴 MISSING: $doc_file"
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
# Find code files changed since IMPLEMENTATION.md was last updated
impl_commit=$(git log -1 --format="%H" -- code/core/IMPLEMENTATION.md)
echo "Changes since IMPLEMENTATION.md update:"
git diff --name-only $impl_commit..HEAD -- code/core/internal/
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

# 3. Undocumented exports (sample)
echo -e "\n=== Undocumented Exports (first 10) ==="
grep -rn "^func [A-Z]" --include="*.go" internal/ | head -10
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

## When to Run

- After completing any milestone
- Before creating a PR
- When resuming work after a break
- Weekly during active development

## Related Commands

- `/code-audit` — Full 7-phase quality check (includes docs in Phase 7)
- `/pre-commit` — Quick lint/test before commits
