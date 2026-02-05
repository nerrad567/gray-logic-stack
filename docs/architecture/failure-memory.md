---
title: Failure Memory Specification
version: 1.0.0
status: planned
implementation_status: not_started
target_version: v2.0
last_updated: 2026-01-19
depends_on:
  - architecture/supervisor.md
  - errors/catalog.md
  - resilience/offline.md
---

# Failure Memory Specification

This document specifies the Failure Memory system — enabling Gray Logic to learn from actual failures and avoid repeating the same mistakes.

> [!NOTE]
> **Status: Planned for v2.0 (Year 3+)**
> 
> This is a future enhancement building on the [Supervisor](supervisor.md) and [Error Catalog](../errors/catalog.md).

---

## Overview

### The Problem

Static playbooks have limitations:
- They only cover *anticipated* failure patterns
- They can't adapt to site-specific conditions
- They don't learn from what *actually* worked

### The Solution

A dynamic memory system that:
1. **Captures** every failure with full context
2. **Extracts** patterns and root causes
3. **Stores** successful resolutions
4. **Retrieves** relevant past failures when similar issues occur
5. **Applies** known fixes before trying generic playbooks

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FAILURE MEMORY LOOP                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Failure Occurs                                                             │
│        │                                                                     │
│        ▼                                                                     │
│   Extract Context:                                                           │
│     - Error ID, component, timestamp                                         │
│     - System state (what else was happening)                                 │
│     - Recent actions (what led to this)                                      │
│     - Environment (temp, load, time of day)                                  │
│        │                                                                     │
│        ▼                                                                     │
│   Search Failure Memory:                                                     │
│     - Vector similarity against past failures                                │
│     - Match on error pattern + context                                       │
│        │                                                                     │
│        ├── Similar past failure found:                                       │
│        │     - Retrieve what fixed it                                        │
│        │     - Apply known fix FIRST                                         │
│        │     - Skip generic playbook                                         │
│        │                                                                     │
│        └── No match found:                                                   │
│              - Execute generic playbook                                      │
│              - Monitor outcome                                               │
│              - Store result in memory                                        │
│        │                                                                     │
│        ▼                                                                     │
│   After Resolution:                                                          │
│     - Record: what was tried, did it work, what finally fixed it             │
│     - Update memory with successful pattern                                  │
│     - Decay memory entries that never matched                                │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Concepts

### Failure Record

Every failure is captured with full context:

```yaml
failure_record:
  # Identity
  id: "fail-2026-01-19-14-23-001"
  timestamp: "2026-01-19T14:23:45Z"
  
  # What failed
  error_id: "ERR-INFRA-MQTT-001"
  component: "mqtt"
  severity: "critical"
  error_message: "connection refused"
  
  # System context at time of failure
  context:
    system_health:
      database: "healthy"
      victoriametrics: "healthy"
      bridges:
        knx: "healthy"
        dali: "degraded"
    
    recent_events:
      - "scene 'Good Night' activated 2 minutes ago"
      - "bridge-dali reconnected 5 minutes ago"
      - "system backup completed 1 hour ago"
    
    resource_state:
      memory_percent: 45
      disk_percent: 62
      cpu_percent: 12
    
    environment:
      time_of_day: "afternoon"
      day_of_week: "sunday"
      uptime_hours: 168
  
  # What was tried
  recovery_attempts:
    - attempt: 1
      action: "restart_service"
      target: "mosquitto"
      outcome: "failed"
      duration_seconds: 15
      error: "service already running"
      
    - attempt: 2
      action: "check_network"
      target: "mqtt_port"
      outcome: "success"
      finding: "port blocked by firewall rule"
      
    - attempt: 3
      action: "remove_firewall_rule"
      target: "ufw rule 23"
      outcome: "success"
      
    - attempt: 4
      action: "verify_connection"
      target: "mqtt"
      outcome: "success"
  
  # Resolution
  resolution:
    successful: true
    what_fixed_it: "Firewall rule was blocking MQTT port"
    root_cause: "UFW rule added during security hardening blocked localhost"
    fix_action: "remove_firewall_rule"
    fix_target: "ufw rule blocking 1883"
    
  # Learning
  pattern:
    signature: ["MQTT connection refused", "service running", "port unreachable"]
    context_markers: ["after security update", "firewall change"]
    recommended_first_check: "firewall_rules"
```

### Pattern Extraction

After each resolution, extract a reusable pattern:

```yaml
learned_pattern:
  id: "pattern-mqtt-firewall-block"
  created_from: "fail-2026-01-19-14-23-001"
  
  # When to apply this pattern
  trigger:
    error_ids: ["ERR-INFRA-MQTT-001"]
    symptoms:
      - "connection refused"
      - "service status: running"
    context_hints:
      - "recent security changes"
      - "firewall modified"
  
  # What to check first
  recommended_actions:
    - order: 1
      action: "check_firewall_rules"
      target: "port 1883"
      expected: "port should be open"
      
    - order: 2
      action: "check_iptables"
      target: "MQTT traffic"
  
  # Success metrics
  metrics:
    times_matched: 3
    times_resolved: 2
    success_rate: 0.67
    last_matched: "2026-03-15T10:00:00Z"
```

---

## Memory Architecture

### Storage Structure

```
/var/lib/graylogic/memory/
├── failures/
│   ├── 2026-01/
│   │   ├── fail-2026-01-19-14-23-001.yaml
│   │   └── fail-2026-01-20-08-15-002.yaml
│   └── 2026-02/
│       └── ...
├── patterns/
│   ├── pattern-mqtt-firewall-block.yaml
│   ├── pattern-influx-disk-full.yaml
│   └── pattern-bridge-clock-skew.yaml
├── solutions/
│   ├── solution-firewall-rules.yaml
│   └── solution-disk-cleanup.yaml
└── index/
    ├── embeddings.db       # Vector embeddings for similarity search
    └── metadata.db         # SQLite index for fast lookup
```

### Embedding Strategy

For similarity search, failures are embedded based on:

```yaml
embedding_features:
  # Primary (high weight)
  - error_id
  - error_message_normalized
  - component
  
  # Secondary (medium weight)
  - symptoms_list
  - recent_events
  
  # Tertiary (low weight)
  - resource_state
  - time_of_day
  - uptime_category
```

### Retrieval Process

```go
// internal/resilience/memory/retriever.go

func (r *Retriever) FindSimilarFailures(current *FailureContext) ([]*PastFailure, error) {
    // 1. Generate embedding for current failure
    embedding := r.embedder.Embed(current)
    
    // 2. Search vector index for similar failures
    candidates := r.vectorIndex.Search(embedding, topK: 10, threshold: 0.75)
    
    // 3. Filter by recency (prefer recent matches)
    candidates = r.filterByRecency(candidates, maxAge: 90*24*time.Hour)
    
    // 4. Rank by success rate of associated patterns
    ranked := r.rankBySuccessRate(candidates)
    
    // 5. Return top matches
    return ranked[:min(3, len(ranked))], nil
}
```

---

## Integration with Supervisor

The Failure Memory enhances the Supervisor's decision-making:

```go
// internal/resilience/supervisor/supervisor.go

func (s *Supervisor) handleUnhealthy(state *SubsystemState, status health.HealthStatus) {
    // 1. Build failure context
    context := s.buildFailureContext(state, status)
    
    // 2. Search failure memory for similar past failures
    pastFailures, _ := s.memory.FindSimilarFailures(context)
    
    if len(pastFailures) > 0 && pastFailures[0].Resolution.Successful {
        // 3a. Apply learned fix first
        log.Info("applying learned fix from past failure",
            "past_failure_id", pastFailures[0].ID,
            "recommended_action", pastFailures[0].Resolution.FixAction,
        )
        
        result := s.applyLearnedFix(pastFailures[0])
        
        if result.Success {
            // Update pattern success metrics
            s.memory.RecordPatternSuccess(pastFailures[0].PatternID)
            return
        }
        
        // Learned fix didn't work this time
        s.memory.RecordPatternFailure(pastFailures[0].PatternID)
    }
    
    // 3b. Fall back to generic playbook
    playbook := s.playbooks.FindPlaybook(state.Name, status)
    s.executePlaybook(state, playbook, context)
}
```

---

## Memory Lifecycle

### Capture Phase

Every failure automatically captured:

```yaml
capture_triggers:
  - health_status: "unhealthy"
  - error_severity: ["critical", "high"]
  - playbook_executed: true
  
capture_excludes:
  - transient_errors: true    # Single occurrence, immediately resolved
  - test_mode: true           # Don't learn from test/dev failures
```

### Consolidation Phase

Nightly process to consolidate and compress:

```yaml
consolidation:
  schedule: "02:00 daily"
  
  actions:
    # Group similar failures
    - deduplicate:
        threshold: 0.95  # Nearly identical failures
        action: "merge into single record with count"
    
    # Extract patterns from repeated failures
    - pattern_extraction:
        min_occurrences: 3
        min_similarity: 0.80
        action: "create learned pattern"
    
    # Decay old memories
    - decay:
        age_threshold_days: 180
        success_rate_threshold: 0.3
        action: "archive to cold storage"
```

### Decay Phase

Memories that don't prove useful fade:

```yaml
decay_rules:
  # Patterns that never match
  - condition: "pattern.times_matched == 0 AND age > 90 days"
    action: "delete"
    
  # Patterns with low success rate
  - condition: "pattern.success_rate < 0.2 AND pattern.times_matched > 5"
    action: "archive and flag for review"
    
  # Old failure records (detail retention)
  - condition: "failure.age > 365 days"
    action: "compress to summary only"
```

---

## Audit Trail Enhancement

Per article requirement: "stores what was decided, why, who authorized it, what was the outcome"

```yaml
audit_entry:
  timestamp: "2026-01-19T14:25:00Z"
  
  # What was decided
  decision: "apply learned fix: check firewall rules"
  
  # Why it was decided
  reasoning:
    trigger: "ERR-INFRA-MQTT-001 detected"
    similar_past_failure: "fail-2026-01-10-09-30-005"
    similarity_score: 0.89
    past_resolution: "Firewall rule removed, MQTT recovered"
    confidence: "high (3 past successes with this pattern)"
    
  # Who/what authorized it
  authority: "supervisor_automatic"
  requires_human_approval: false
  
  # What was the outcome
  outcome:
    action_taken: "check_firewall_rules"
    result: "rule found blocking port 1883"
    follow_up_action: "remove_firewall_rule"
    final_status: "resolved"
    resolution_time_seconds: 45
```

---

## Transparent Reasoning

Per article requirement: "every suggestion includes 'why i'm recommending this' with citations to specific data points"

When the Supervisor takes action, it logs reasoning:

```json
{
  "level": "info",
  "msg": "applying learned recovery action",
  "component": "supervisor",
  "decision": {
    "action": "check_firewall_rules",
    "reasoning": {
      "current_failure": "MQTT connection refused while service running",
      "matched_pattern": "pattern-mqtt-firewall-block",
      "pattern_success_rate": 0.67,
      "similar_past_failures": [
        {
          "id": "fail-2026-01-10-09-30-005",
          "similarity": 0.89,
          "what_fixed_it": "Remove UFW rule blocking 1883"
        }
      ],
      "recommendation_source": "failure_memory",
      "confidence": "high"
    }
  }
}
```

---

## Implementation Milestones

### M-MEM.1: Failure Capture

**Deliverables:**
- [ ] Define failure record schema
- [ ] Implement failure context builder
- [ ] Hook into Supervisor for automatic capture
- [ ] Store failures in structured YAML files

**Acceptance:**
- Every Supervisor-handled failure is recorded
- Context includes system state and recent events

### M-MEM.2: Pattern Extraction

**Deliverables:**
- [ ] Implement pattern extraction algorithm
- [ ] Define pattern matching rules
- [ ] Create consolidation nightly job

**Acceptance:**
- Repeated similar failures generate patterns
- Patterns include recommended first actions

### M-MEM.3: Similarity Search

**Deliverables:**
- [ ] Choose embedding model (local, small)
- [ ] Implement vector index
- [ ] Implement retrieval with ranking

**Acceptance:**
- Similar failures retrieved in <100ms
- Ranking prioritizes successful resolutions

### M-MEM.4: Learning Integration

**Deliverables:**
- [ ] Integrate memory into Supervisor loop
- [ ] Implement learned fix application
- [ ] Track pattern success/failure metrics

**Acceptance:**
- Supervisor checks memory before playbooks
- Success rates update after each application

### M-MEM.5: Audit Trail

**Deliverables:**
- [ ] Implement reasoning capture
- [ ] Create audit log entries
- [ ] Enable querying "why did you do X?"

**Acceptance:**
- Every automated action has reasoning log
- Audit queryable by time range and action type

---

## See Also

- [Supervisor Specification](supervisor.md) — What triggers failure handling
- [Error Catalog](../errors/catalog.md) — Static error definitions
- [Workflow Learning](workflow-learning.md) — Improving playbooks over time
- [Offline Behavior](../resilience/offline.md) — Failure scenarios
