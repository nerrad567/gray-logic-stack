---
title: Workflow Learning Specification
version: 1.0.0
status: planned
implementation_status: not_started
target_version: v2.0
last_updated: 2026-01-19
depends_on:
  - architecture/supervisor.md
  - architecture/failure-memory.md
  - errors/catalog.md
---

# Workflow Learning Specification

This document specifies the Workflow Learning system — enabling Gray Logic to improve its playbooks and recovery procedures based on actual outcomes.

> [!NOTE]
> **Status: Planned for v2.0 (Year 4+)**
> 
> This is a future enhancement building on [Failure Memory](failure-memory.md).

---

## Overview

### The Problem

Static playbooks become stale:
- They were designed for anticipated scenarios
- Real-world failures often differ from predictions
- Manual playbook updates are slow and error-prone
- Success rates aren't tracked or optimized

### The Solution

A learning system that:
1. **Tracks** every playbook execution and outcome
2. **Analyzes** what steps succeeded vs. failed
3. **Identifies** patterns in successful recoveries
4. **Suggests** playbook improvements
5. **Evolves** playbooks over time (with human approval)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        WORKFLOW LEARNING LOOP                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   Playbook Executes                                                          │
│        │                                                                     │
│        ▼                                                                     │
│   Track Every Step:                                                          │
│     - Which step executed                                                    │
│     - Duration per step                                                      │
│     - Step outcome (success/failure/skip)                                    │
│     - What errors occurred                                                   │
│        │                                                                     │
│        ▼                                                                     │
│   After Completion:                                                          │
│     - Overall success or failure?                                            │
│     - Which step actually fixed it?                                          │
│     - Were any steps unnecessary?                                            │
│     - Did we escalate? Should we have?                                       │
│        │                                                                     │
│        ▼                                                                     │
│   Periodic Analysis (weekly):                                                │
│     - Calculate per-step success rates                                       │
│     - Identify bottleneck steps                                              │
│     - Detect step ordering opportunities                                     │
│     - Flag playbooks for review                                              │
│        │                                                                     │
│        ▼                                                                     │
│   Generate Recommendations:                                                  │
│     - "Step 2 succeeds 90% of the time; consider moving to Step 1"           │
│     - "Step 4 never succeeds after Step 3 fails; add skip condition"         │
│     - "New pattern detected: add as Step 0"                                  │
│        │                                                                     │
│        ▼                                                                     │
│   Human Review:                                                              │
│     - Present recommendations with evidence                                  │
│     - Human approves, modifies, or rejects                                   │
│     - Approved changes become new playbook version                           │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Concepts

### Execution Record

Every playbook execution is fully logged:

```yaml
execution_record:
  id: "exec-2026-01-19-14-25-001"
  playbook_id: "PLAY-MQTT-RECOVER-001"
  playbook_version: "1.0.0"
  trigger:
    failure_id: "fail-2026-01-19-14-23-001"
    error_id: "ERR-INFRA-MQTT-001"
  
  started_at: "2026-01-19T14:25:00Z"
  completed_at: "2026-01-19T14:25:45Z"
  duration_seconds: 45
  
  # Per-step tracking
  steps:
    - step: 1
      name: "Check broker status"
      action: "check_service"
      started_at: "2026-01-19T14:25:00Z"
      duration_seconds: 2
      outcome: "success"
      result: "service running"
      transitioned_to: "step_2"  # Where did we go next?
      
    - step: 2
      name: "Restart broker"
      action: "restart_service"
      started_at: "2026-01-19T14:25:02Z"
      duration_seconds: 15
      outcome: "failed"
      error: "Service already running, restart had no effect"
      transitioned_to: "step_3"
      
    - step: 3
      name: "Check network connectivity"
      action: "check_port"
      started_at: "2026-01-19T14:25:17Z"
      duration_seconds: 3
      outcome: "success"
      result: "port 1883 blocked"
      insight: "This step revealed the actual issue"
      transitioned_to: "manual_intervention"
      
  # Overall outcome
  outcome:
    status: "resolved"
    resolved_by: "manual_intervention"
    what_fixed_it: "Firewall rule removed"
    playbook_sufficient: false
    recommendation: "Add firewall check as Step 0"
    
  # Learning annotations
  learning:
    steps_that_helped: [3]
    steps_that_wasted_time: [2]
    missing_steps: ["check_firewall_rules"]
    suggested_improvements:
      - "Add firewall check before restart attempt"
      - "Skip restart if service already running"
```

### Playbook Metrics

Aggregate statistics for each playbook:

```yaml
playbook_metrics:
  playbook_id: "PLAY-MQTT-RECOVER-001"
  version: "1.0.0"
  
  # Overall stats
  total_executions: 47
  successful_resolutions: 38
  escalations: 9
  success_rate: 0.81
  
  # Time metrics
  avg_resolution_time_seconds: 35
  p95_resolution_time_seconds: 120
  
  # Per-step metrics
  steps:
    - step: 1
      name: "Check broker status"
      executions: 47
      successes: 42
      success_rate: 0.89
      avg_duration_seconds: 2
      
    - step: 2
      name: "Restart broker"
      executions: 45
      successes: 30
      success_rate: 0.67
      avg_duration_seconds: 18
      often_skipped_after: "step_1_success"  # Pattern detected
      
    - step: 3
      name: "Verify recovery"
      executions: 30
      successes: 28
      success_rate: 0.93
      avg_duration_seconds: 8
      
  # Flow analysis
  common_paths:
    - path: [1, 2, 3]
      frequency: 0.60
      success_rate: 0.85
      
    - path: [1, 2, "escalate"]
      frequency: 0.15
      success_rate: 0.0  # These required human intervention
      
    - path: [1, "done"]
      frequency: 0.10
      success_rate: 1.0  # Service was already healthy
      
  # Problem areas
  bottlenecks:
    - step: 2
      issue: "Restart often fails when service already running"
      recommendation: "Add condition: only restart if service stopped"
```

### Improvement Recommendation

Generated suggestions for playbook enhancement:

```yaml
improvement_recommendation:
  id: "rec-2026-01-20-001"
  playbook_id: "PLAY-MQTT-RECOVER-001"
  generated_at: "2026-01-20T02:00:00Z"
  
  recommendation_type: "add_step"
  priority: "high"
  confidence: 0.85
  
  # What we recommend
  recommendation:
    action: "Insert new step before Step 2"
    new_step:
      name: "Check firewall rules"
      action: "check_firewall_rules"
      target: "port 1883"
      on_success: "goto:step_2"
      on_failure: "Execute: remove_blocking_rule"
      
  # Evidence for recommendation
  evidence:
    based_on_executions: 12
    pattern:
      description: "In 8/12 recent MQTT failures, firewall was the root cause"
      first_detected: "2026-01-10"
      occurrences: 8
      resolution: "Firewall rule removal resolved the issue"
      
    supporting_failures:
      - "fail-2026-01-10-09-30-005"
      - "fail-2026-01-15-14-22-003"
      - "fail-2026-01-19-14-23-001"
      
  # Expected impact
  expected_impact:
    success_rate_improvement: "+15%"
    avg_resolution_time_reduction: "-20 seconds"
    escalation_rate_reduction: "-30%"
    
  # Approval workflow
  approval:
    status: "pending"
    requires_approval_from: "admin"
    auto_approve_if: null  # Always require human for playbook changes
```

---

## Analysis Engine

### Weekly Analysis Process

```yaml
analysis_schedule:
  frequency: "weekly"
  day: "sunday"
  time: "03:00"
  
analysis_steps:
  1_aggregate_metrics:
    - Calculate per-playbook success rates
    - Calculate per-step success rates
    - Identify execution path frequencies
    
  2_detect_patterns:
    - Find steps that often fail together
    - Find steps that could be reordered
    - Find steps that are often skipped
    
  3_cross_reference_failures:
    - Match failed playbooks with failure memory
    - Identify what actually fixed the issues
    - Detect gaps in playbook coverage
    
  4_generate_recommendations:
    - Create improvement recommendations
    - Calculate confidence scores
    - Estimate expected impact
    
  5_notify_admins:
    - Send summary to admin dashboard
    - Highlight high-priority recommendations
    - Include evidence links
```

### Pattern Detection Algorithms

```go
// internal/resilience/learning/analyzer.go

type Analyzer struct {
    executions  ExecutionStore
    failures    FailureMemory
    playbooks   PlaybookRegistry
}

func (a *Analyzer) DetectImprovementOpportunities(playbookID string) []Recommendation {
    var recommendations []Recommendation
    
    // Get recent executions
    execs := a.executions.GetByPlaybook(playbookID, lastDays: 30)
    
    // 1. Detect steps with low success rates
    for _, step := range a.calculateStepMetrics(execs) {
        if step.SuccessRate < 0.5 && step.Executions > 10 {
            recommendations = append(recommendations, Recommendation{
                Type: "modify_step",
                Step: step.Name,
                Issue: fmt.Sprintf("Step has %.0f%% success rate", step.SuccessRate*100),
                Suggestion: "Review step conditions or add alternative path",
            })
        }
    }
    
    // 2. Detect missing steps from failure memory
    failedExecs := filter(execs, func(e Execution) bool { return e.Outcome.Status == "escalated" })
    for _, exec := range failedExecs {
        failure := a.failures.Get(exec.Trigger.FailureID)
        if failure != nil && failure.Resolution.Successful {
            if !a.playbookContainsAction(playbookID, failure.Resolution.FixAction) {
                recommendations = append(recommendations, Recommendation{
                    Type: "add_step",
                    Action: failure.Resolution.FixAction,
                    Evidence: failure.ID,
                    Suggestion: "Add this action that resolved similar failures",
                })
            }
        }
    }
    
    // 3. Detect step reordering opportunities
    paths := a.analyzeExecutionPaths(execs)
    for _, path := range paths {
        if betterOrder := a.findBetterOrdering(path); betterOrder != nil {
            recommendations = append(recommendations, Recommendation{
                Type: "reorder_steps",
                CurrentOrder: path.Steps,
                ProposedOrder: betterOrder,
                Suggestion: "Reordering could improve success rate",
            })
        }
    }
    
    return recommendations
}
```

---

## Playbook Versioning

### Version Control

Playbooks are versioned like code:

```yaml
playbook:
  id: "PLAY-MQTT-RECOVER-001"
  
  versions:
    - version: "1.0.0"
      created_at: "2026-01-01"
      created_by: "initial_setup"
      status: "deprecated"
      
    - version: "1.1.0"
      created_at: "2026-01-20"
      created_by: "workflow_learning"
      approved_by: "admin"
      approval_reason: "Added firewall check based on 8 observed failures"
      source_recommendation: "rec-2026-01-20-001"
      status: "active"
      
      changes:
        - type: "add_step"
          step: 0
          description: "Check firewall rules before attempting restart"
          evidence: "8/12 MQTT failures in Jan 2026 were firewall-related"
```

### Rollback Capability

If a new version performs worse:

```yaml
rollback_trigger:
  condition: "version.success_rate < previous_version.success_rate - 0.10"
  min_executions: 10  # Need enough data to judge
  
rollback_action:
  - Revert to previous version
  - Mark new version as "failed"
  - Generate investigation report
  - Notify admin
```

---

## Human-in-the-Loop

### Approval Workflow

No playbook changes are automatic:

```yaml
approval_workflow:
  # All recommendations require human review
  require_approval: true
  
  # Who can approve
  approvers:
    - role: "admin"
    - role: "system_owner"
    
  # How long to wait
  timeout_days: 7
  
  # If not approved in time
  on_timeout: "archive_recommendation"
  
  # Approval interface
  presentation:
    show_recommendation: true
    show_evidence: true
    show_expected_impact: true
    show_affected_playbooks: true
    allow_modification: true  # Human can adjust before approving
```

### Admin Dashboard

```yaml
dashboard_elements:
  - section: "Pending Recommendations"
    items:
      - recommendation_id
      - playbook_affected
      - recommendation_type
      - confidence_score
      - expected_impact
      - evidence_summary
      - approve_button
      - reject_button
      - modify_button
      
  - section: "Playbook Performance"
    items:
      - playbook_id
      - current_version
      - success_rate_trend (30 day chart)
      - avg_resolution_time_trend
      - recent_escalations
      
  - section: "Learning Activity"
    items:
      - patterns_detected_this_week
      - improvements_applied_this_month
      - success_rate_improvements_achieved
```

---

## Integration Points

### With Supervisor

```go
// Supervisor notifies learning system of execution
func (s *Supervisor) executePlaybook(state *SubsystemState, playbook *Playbook) {
    execution := s.learning.StartExecution(playbook.ID, state.CurrentFailure)
    
    for _, step := range playbook.Steps {
        stepExec := execution.StartStep(step)
        
        result := s.runStep(step)
        
        stepExec.Complete(result)
    }
    
    execution.Complete(s.evaluateOutcome())
}
```

### With Failure Memory

```go
// Cross-reference with failure memory for insights
func (a *Analyzer) CrossReferenceWithFailures(exec *Execution) {
    if exec.Outcome.Status == "escalated" {
        // Find the failure that was eventually resolved
        failure := a.failures.Get(exec.Trigger.FailureID)
        
        if failure != nil && failure.Resolution.Successful {
            // Learn what actually fixed it
            a.suggestNewStep(exec.PlaybookID, failure.Resolution)
        }
    }
}
```

---

## Implementation Milestones

### M-LEARN.1: Execution Tracking

**Deliverables:**
- [ ] Define execution record schema
- [ ] Implement per-step tracking in Supervisor
- [ ] Store execution records

**Acceptance:**
- Every playbook execution is fully logged
- Per-step durations and outcomes captured

### M-LEARN.2: Metrics Aggregation

**Deliverables:**
- [ ] Implement playbook metrics calculator
- [ ] Implement step success rate tracking
- [ ] Create execution path analysis

**Acceptance:**
- Metrics update after each execution
- Path frequencies calculated accurately

### M-LEARN.3: Pattern Detection

**Deliverables:**
- [ ] Implement bottleneck detection
- [ ] Implement step reordering analysis
- [ ] Implement gap detection (missing steps)

**Acceptance:**
- Obvious improvements detected with high confidence
- False positive rate < 20%

### M-LEARN.4: Recommendation Generation

**Deliverables:**
- [ ] Implement recommendation generator
- [ ] Calculate confidence scores
- [ ] Generate evidence summaries

**Acceptance:**
- Recommendations are actionable and specific
- Evidence clearly supports recommendation

### M-LEARN.5: Approval Workflow

**Deliverables:**
- [ ] Implement approval queue
- [ ] Create admin notification
- [ ] Implement playbook versioning

**Acceptance:**
- Recommendations require human approval
- Approved changes create new playbook version
- Rollback works correctly

---

## See Also

- [Failure Memory](failure-memory.md) — Learning from individual failures
- [Supervisor](supervisor.md) — Executes playbooks
- [Error Catalog](../errors/catalog.md) — Static error definitions
- [Monitoring](../operations/monitoring.md) — Metrics and observability
