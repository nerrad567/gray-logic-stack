# Ralph Experiment Setup

This document explains how to set up the "amped" fork for autonomous AI development with Ralph.

## Fork Creation (Manual Steps)

1. **Create fork on GitHub**
   ```bash
   # Authenticate GitHub CLI
   gh auth login
   
   # Create the fork
   gh repo fork nerrad567/gray-logic-stack --fork-name amped --clone=false
   ```

2. **Clone the fork locally**
   ```bash
   cd ~
   git clone git@github.com:nerrad567/amped.git
   cd amped
   ```

3. **Install Ralph from source**
   ```bash
   # Clone Ralph skills
   git clone https://github.com/snarktank/ralph.git /tmp/ralph
   
   # Copy scripts
   cp -r /tmp/ralph/scripts/ralph ./scripts/
   cp -r /tmp/ralph/skills ./skills
   
   # Make executable
   chmod +x scripts/ralph/ralph.sh
   ```

4. **Install prerequisites**
   ```bash
   # Install Amp CLI
   # See: https://ampcode.com
   
   # Install jq
   sudo apt install jq  # or: brew install jq
   ```

## Running Ralph

```bash
# Start dev services first
docker compose -f docker-compose.dev.yml up -d

# Run Ralph with 10 iterations max
./scripts/ralph/ralph.sh 10
```

## Files Already Prepared

| File | Purpose |
|------|---------|
| `AGENTS.md` | Context for Amp (like CLAUDE.md) |
| `prd.json` | User stories for M1.1 completion |
| `progress.txt` | Learnings persist across iterations |
| `tasks/prd-m1.1-complete.md` | Human-readable PRD |

## M1.1 Stories in prd.json

1. **mqtt-race-fix** (P0) — Fix data race in MQTT callback
2. **main-wiring** (P1) — Wire main.go with all infrastructure  
3. **health-endpoint** (P2) — Add HTTP /health endpoint
4. **device-registry-schema** (P2) — Add device tables to database
5. **documentation-update** (P3) — Update docs

## Monitoring Progress

Ralph updates these files each iteration:
- `prd.json` — Stories marked `"passes": true` when complete
- `progress.txt` — Learnings appended
- `AGENTS.md` — Patterns/gotchas discovered

Watch for:
```bash
# Check which stories are complete
cat prd.json | jq '.stories[] | select(.passes == true) | .title'

# Read learnings
tail -50 progress.txt
```

## Comparison Methodology

Track these metrics for both branches:

| Metric | Main (Manual) | Amped (Ralph) |
|--------|---------------|---------------|
| Time to complete M1.1 | TBD | TBD |
| Commits | TBD | TBD |
| Lines of code | TBD | TBD |
| Test coverage | TBD | TBD |
| Principle violations | TBD | TBD |
| Manual interventions | N/A | TBD |

After M1.1 completes on both branches, compare the diffs.
