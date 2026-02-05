---
description: Performance benchmarking — establish baselines, detect regressions, profile hot paths
---

# Benchmark Agent — Performance Baselines & Regression Detection

You are a performance engineering specialist for the Gray Logic Stack.
Your goal is to establish, run, and analyse Go benchmarks for critical code paths.

**Context:** Gray Logic runs on embedded ARM hardware (SBCs) controlling physical building systems.
Latency matters — a 50ms delay in telegram processing means lights feel sluggish.
Memory matters — 2-4 GB RAM shared with VictoriaMetrics, Mosquitto, and the OS.

---

## What This Command Does

Depending on the argument, this command operates in one of three modes:

| Mode | Argument | Purpose |
|------|----------|---------|
| **Run** | `run` or no argument | Run all existing benchmarks, compare with saved baseline |
| **Baseline** | `baseline` | Run benchmarks and save results as the new baseline |
| **Create** | `create {package}` | Identify missing benchmarks in a package and write them |
| **Profile** | `profile {package}` | Run benchmarks with CPU/memory profiling |

---

## Mode: Run Benchmarks

```bash
cd code/core

# Run all benchmarks (10 iterations for stable results)
go test -bench=. -benchmem -count=5 -run=^$ ./... 2>&1 | tee /tmp/bench-current.txt

# Compare with baseline (if benchstat is available)
benchstat .bench-baseline.txt /tmp/bench-current.txt 2>/dev/null || \
  echo "No baseline found. Run '/bench baseline' to establish one."
```

### Interpreting Results

| Metric | What It Means | Alarm Threshold |
|--------|---------------|-----------------|
| `ns/op` | Time per operation | >2x baseline |
| `B/op` | Bytes allocated per operation | >2x baseline |
| `allocs/op` | Number of allocations per operation | >2x baseline |

Report any benchmark that has regressed >20% from baseline as a **warning**, >100% as **critical**.

---

## Mode: Save Baseline

```bash
cd code/core

# Run benchmarks and save as baseline
go test -bench=. -benchmem -count=5 -run=^$ ./... > .bench-baseline.txt 2>&1

echo "Baseline saved to .bench-baseline.txt"
echo "Commit this file to track performance over time."
```

---

## Mode: Create Benchmarks

When asked to create benchmarks for a package, identify the critical hot paths:

### Priority Targets (by package)

| Package | Hot Path | Why It Matters |
|---------|----------|----------------|
| `internal/device` | `Registry.GetDevice`, `Registry.SetDeviceState`, `RefreshCache` | Called on every KNX telegram |
| `internal/bridges/knx` | DPT encode/decode, telegram parsing | Real-time bus processing |
| `internal/infrastructure/tsdb` | `formatLineProtocol`, `addLine`, `Flush` | High-throughput telemetry |
| `internal/api` | WebSocket broadcast, JSON serialisation | Fan-out to all connected panels |
| `internal/auth` | `HashPassword`, `VerifyPassword`, JWT parse/sign | Login latency, per-request auth |
| `internal/device` | `ResolveGroup`, tag filtering | Called for scene execution |
| `internal/location` | Hierarchy building, zone lookups | Called on every hierarchy request |

### Benchmark Template

```go
func BenchmarkXxx(b *testing.B) {
    // Setup (not timed)
    sut := setupTestThing(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        sut.DoThing()
    }
}

// For benchmarks that need allocation tracking:
func BenchmarkXxx_Allocs(b *testing.B) {
    sut := setupTestThing(b)

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        sut.DoThing()
    }
}

// For concurrent benchmarks (simulating multiple panels/devices):
func BenchmarkXxx_Parallel(b *testing.B) {
    sut := setupTestThing(b)

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            sut.DoThing()
        }
    })
}
```

### What to Benchmark (Decision Framework)

| Question | If Yes → Benchmark |
|----------|-------------------|
| Called on every incoming KNX telegram? | Yes — latency critical |
| Called on every API request? | Yes — throughput critical |
| Involves cryptographic operations? | Yes — CPU bound |
| Allocates memory in a loop? | Yes — GC pressure |
| Called during scene execution? | Yes — real-time path |
| Fan-out to N clients? | Yes — scalability |

---

## Mode: Profile

```bash
cd code/core

# CPU profile
go test -bench=BenchmarkXxx -cpuprofile=cpu.prof -run=^$ ./{package}/...
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -bench=BenchmarkXxx -memprofile=mem.prof -run=^$ ./{package}/...
go tool pprof -http=:8080 mem.prof
```

---

## Output Format

```markdown
## Benchmark Report

### Environment
- Go version: {version}
- OS/Arch: {os/arch}
- Commit: {hash}
- Date: {date}

### Results

| Benchmark | ns/op | B/op | allocs/op | vs Baseline |
|-----------|-------|------|-----------|-------------|
| BenchmarkGetDevice | 125 | 48 | 1 | — (new) |
| BenchmarkSetState | 890 | 256 | 5 | +12% ⚠️ |

### Regressions
{List any benchmarks >20% slower than baseline}

### Recommendations
{Specific optimisation suggestions based on profile data}
```

---

## Integration with Other Commands

- `/bench baseline` should be run after each milestone is complete
- `/bench run` should be part of `/milestone-audit` Stage 6 (alongside coverage)
- `/optimise` uses benchmark data to identify and verify optimisations
- Baseline file (`.bench-baseline.txt`) should be committed to git

## Target

$ARGUMENTS
