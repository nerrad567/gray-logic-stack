# Protocol Topology: KNX Physical Layer + Direct DALI Intelligence

This document defines the Gray Logic architecture for integrating DALI lighting with KNX building automation.

## Core Principle: Separation of Concerns

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     PHYSICAL LAYER (Always Works)                           │
│                                                                             │
│   Wall Switch ──► KNX Bus ──► KNX Relay Actuator ──► 230V ──► DALI Driver  │
│                                                                             │
│   • No software dependency                                                  │
│   • Works if Gray Logic is down                                            │
│   • Works if DALI bus is down                                              │
│   • Driver defaults to 100% brightness on power                            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ DALI bus (parallel, independent)
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     INTELLIGENCE LAYER (Optional)                           │
│                                                                             │
│   DALI Driver ◄──► DALI Bus ◄──► DALI Gateway ◄──► Gray Logic Core         │
│                                   (Lunatone IoT)    (REST API)              │
│                                                                             │
│   • Dimming, colour temperature, scenes                                    │
│   • Full DALI-2 diagnostics (voltage, current, runtime, temperature)       │
│   • PHM — Predictive Health Monitoring                                     │
│   • If this layer fails: lights still work at ON/OFF via KNX              │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Insight:** KNX provides physical reliability. DALI provides intelligence. They operate independently — failure of one doesn't break the other.

---

## Architecture Diagram

```
                    ┌─────────────────────────────────────────┐
                    │            Gray Logic Core              │
                    │                                         │
                    │  ┌─────────────┐    ┌───────────────┐  │
                    │  │ KNX Bridge  │    │ DALI Bridge   │  │
                    │  │ (via knxd)  │    │ (REST API)    │  │
                    │  └──────┬──────┘    └───────┬───────┘  │
                    └─────────┼───────────────────┼──────────┘
                              │                   │
                         KNXnet/IP            REST/MQTT
                              │                   │
┌─────────────────────────────┼───────────────────┼──────────────────────────┐
│                    ELECTRICAL PANEL (DIN Rail)  │                          │
│                             │                   │                          │
│    ┌────────────────────────┼───────────────────┼────────────────────┐     │
│    │                        ▼                   ▼                    │     │
│    │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐     │     │
│    │  │ KNX IP      │  │ KNX Relay   │  │ Lunatone DALI-2     │     │     │
│    │  │ Interface   │  │ Actuator    │  │ IoT Gateway         │     │     │
│    │  │ (~€250)     │  │ (~€150-200) │  │ (~€400-450)         │     │     │
│    │  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘     │     │
│    │         │                │                    │                │     │
│    │    KNX TP Bus            │              DALI Bus               │     │
│    │         │                │                    │                │     │
│    └─────────┼────────────────┼────────────────────┼────────────────┘     │
│              │                │                    │                      │
│              ▼                ▼                    ▼                      │
│    ┌─────────────┐     ┌─────────────┐     ┌─────────────┐               │
│    │ KNX Wall    │     │   230V      │     │ DALI Driver │               │
│    │ Switch      │────►│   Mains     │────►│ (in light)  │──► Light     │
│    └─────────────┘     └─────────────┘     └─────────────┘               │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## How It Works

### Normal Operation (Everything Working)

| Action | Path | Result |
|--------|------|--------|
| Wall switch ON | KNX → Relay → 230V → Driver | Light ON at last DALI level |
| Wall switch OFF | KNX → Relay → No 230V | Light OFF |
| "Dim to 50%" | Gray Logic → DALI Gateway → Driver | Light at 50% |
| "Set 3000K warm" | Gray Logic → DALI Gateway → Driver | CCT changes |
| "Movie scene" | Gray Logic → Multiple DALI commands | Scene levels |
| PHM health check | Gray Logic → Query DALI diagnostics | Voltage/current/runtime logged |

### Degraded Mode: Gray Logic Down

| Action | Path | Result |
|--------|------|--------|
| Wall switch ON | KNX → Relay → 230V → Driver | **Light ON at 100%** (Power On Level) |
| Wall switch OFF | KNX → Relay → No 230V | **Light OFF** |
| Dimming | ❌ Not available | — |
| Scenes | ❌ Not available | — |
| PHM | ❌ Not available | — |

**Basic lighting works perfectly** — just without smart features.

### Degraded Mode: DALI Bus Down

| Action | Path | Result |
|--------|------|--------|
| Wall switch ON | KNX → Relay → 230V → Driver | **Light ON at 100%** (System Failure Level) |
| Wall switch OFF | KNX → Relay → No 230V | **Light OFF** |

Same result — KNX physical layer is independent of DALI bus health.

---

## DALI Driver Configuration (Critical)

For this architecture to work correctly, configure each DALI driver during commissioning:

| Parameter | Recommended Value | Purpose |
|-----------|-------------------|---------|
| **Power On Level** | 254 (100%) | Full brightness when 230V applied |
| **System Failure Level** | 254 (100%) or MASK (255) | Stay on / at last level if DALI bus fails |
| **Min Level** | Application-specific | Lowest dim level (prevents flicker) |
| **Max Level** | 254 | Full brightness |
| **Fade Time** | Application-specific | Smooth dimming transitions |

These values are stored in the driver's **non-volatile memory**. Once configured, the driver maintains these defaults regardless of DALI bus or Gray Logic state.

**Configuration can be done via:**
- Gray Logic Core (during commissioning)
- Tridonic DALI Cockpit
- Lunatone configuration tool

---

## Hardware Options

### DALI Gateway Selection

| Product | Price | Lines | Devices | Interface | Best For |
|---------|-------|-------|---------|-----------|----------|
| **Lunatone DALI-2 IoT** | ~€400-450 | 1 | 64 | REST API, WebSocket | Residential, small commercial |
| **Lunatone DALI-2 IoT4** | ~€1,100 | 4 | 256 | REST API, WebSocket | Large commercial |
| **Tridonic DALI USB** | ~€100-150 | 1 | 64 | USB | Budget residential |

**Recommendation:** Lunatone DALI-2 IoT Gateway for most installations. IP-based, full DALI-2 diagnostics, no Lua scripting required.

### Why NOT KNX-DALI Gateways?

Traditional KNX-DALI gateways (MDT, ABB, Theben) have a fundamental limitation:

| What DALI Driver Reports | What KNX Gateway Exposes |
|--------------------------|--------------------------|
| On/Off state | ✅ Yes |
| Brightness level | ✅ Yes |
| Lamp failure | ✅ Yes |
| **Internal voltage** | ❌ No |
| **Driver temperature** | ❌ No |
| **Operating hours** | ❌ No |
| **Power consumption** | ❌ No |
| **Colour temperature** | ⚠️ Limited |

The gateway acts as a **filter** — it can only expose data points that have been pre-mapped to KNX Group Addresses. Mapping 50 data points × 64 drivers = 3,200 Group Addresses, which is impractical.

**Direct DALI access via Lunatone IoT gives us everything.**

---

## Cost Comparison

### For 6 DALI Downlights (Typical Room)

| Approach | Hardware | Cost | Diagnostics |
|----------|----------|------|-------------|
| **KNX-DALI Gateway** | MDT/ABB gateway | ~€450-600 | ⚠️ Limited (lamp fault only) |
| **KNX Relay + Lunatone IoT** | Relay + Lunatone | ~€550-650 | ✅ Full DALI-2 |
| **KNX Relay + DALI USB** | Relay + Tridonic USB | ~€250-350 | ✅ Full (USB not IP) |

### For Whole House (40 DALI Drivers)

| Approach | Hardware | Cost | Notes |
|----------|----------|------|-------|
| **KNX-DALI Gateway (64ch)** | Single gateway | ~€500-700 | Limited diagnostics |
| **KNX Relays + Lunatone IoT** | 2× relay actuators + gateway | ~€700-900 | Full diagnostics, better PHM |

The cost difference is marginal, but the capability difference is significant.

---

## Gray Logic DALI Bridge Implementation

### API Integration with Lunatone

The Lunatone DALI-2 IoT Gateway exposes a REST API:

```
# Discovery
GET http://{gateway}/info

# Control
POST http://{gateway}/dali/device/{address}/level
Body: {"value": 128}  // 0-254

# Query diagnostics
GET http://{gateway}/dali/device/{address}/status
Response: {
  "address": 5,
  "level": 128,
  "lampFailure": false,
  "powerCycles": 1234,
  "operatingTime": 5678,
  "voltage": 230.5,
  "current": 0.15
}

# Group control
POST http://{gateway}/dali/group/{group}/level
Body: {"value": 200}
```

### Gray Logic Bridge Architecture

```go
// internal/bridges/dali/bridge.go

type DALIBridge struct {
    gateway    *LunatoneClient  // REST API client
    mqttClient mqtt.Client      // Internal message bus
    devices    map[int]*DALIDevice
}

// Publish state changes to internal MQTT
func (b *DALIBridge) pollDeviceStatus(ctx context.Context) {
    for addr, device := range b.devices {
        status, err := b.gateway.GetDeviceStatus(addr)
        if err != nil {
            continue
        }
        
        // Publish to internal bus
        b.mqttClient.Publish(fmt.Sprintf("graylogic/dali/%d/state", addr), status)
        
        // PHM: Log to VictoriaMetrics for trending
        b.influx.WritePoint("dali_diagnostics", map[string]interface{}{
            "voltage":       status.Voltage,
            "current":       status.Current,
            "operating_hrs": status.OperatingTime,
            "power_cycles":  status.PowerCycles,
        })
    }
}
```

---

## PHM: Predictive Health Monitoring for DALI

With direct DALI-2 access, Gray Logic can implement genuine predictive maintenance:

| Metric | Normal Range | Warning | Alert |
|--------|--------------|---------|-------|
| **Driver voltage** | 220-240V | <210V or >250V | <200V or >260V |
| **Operating hours** | — | >40,000 hrs | >50,000 hrs (LED lifespan) |
| **Power cycles** | — | >100,000 | >150,000 |
| **Lamp failure flag** | false | — | true |
| **Response time** | <100ms | >200ms | >500ms or timeout |

**Example PHM alert:**
> "Kitchen downlight #3 showing voltage sag (205V vs 230V expected). Driver may be failing. Recommend inspection within 30 days."

This is **impossible** with a standard KNX-DALI gateway — they don't expose voltage data.

---

## Wiring Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           DISTRIBUTION BOARD                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────────┐  ┌──────────────┐  │
│  │ KNX PSU │  │ KNX IP  │  │ KNX     │  │ Lunatone    │  │ DALI PSU     │  │
│  │ 640mA   │  │Interface│  │ Relay   │  │ DALI-2 IoT  │  │ (if needed)  │  │
│  │         │  │         │  │ 4ch/8ch │  │ Gateway     │  │              │  │
│  └────┬────┘  └────┬────┘  └────┬────┘  └──────┬──────┘  └──────┬───────┘  │
│       │            │            │              │                │          │
│  ─────┴────────────┴────────────┴──────────────┴────────────────┴────────  │
│              KNX TP Bus (green)           DALI Bus (purple)                │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                              │
                    │                              │
        ┌───────────┴───────────┐      ┌──────────┴──────────┐
        │                       │      │                     │
        ▼                       ▼      ▼                     ▼
   ┌─────────┐            ┌─────────────────┐          ┌─────────┐
   │ KNX     │            │   DALI Driver   │          │ KNX     │
   │ Wall    │            │   (in fitting)  │          │ Wall    │
   │ Switch  │            │                 │          │ Switch  │
   └─────────┘            │  ┌───────────┐  │          └─────────┘
                          │  │   LED     │  │
        230V from ───────►│  │   Module  │  │
        KNX Relay         │  └───────────┘  │
                          └─────────────────┘
```

**Key points:**
- KNX bus and DALI bus are **separate** physical wiring
- 230V mains to DALI driver is switched by **KNX relay actuator**
- DALI bus carries **only** control/diagnostic signals (not power)
- Lunatone gateway connects to network switch via **Ethernet**

---

## Migration Path

### From "Dumb" DALI Installation

Many existing DALI installations have drivers but no bus connected — just mains switching.

| Step | Action | Result |
|------|--------|--------|
| 1 | Audit existing DALI drivers | Know what you have |
| 2 | Install Lunatone IoT Gateway | DALI bus master |
| 3 | Wire DALI bus to all drivers | Enable communication |
| 4 | Configure driver failsafe levels | Power On = 100% |
| 5 | Connect gateway to Gray Logic | Full control + PHM |

Existing KNX relay switching continues to work throughout migration.

### From KNX-DALI Gateway

| Step | Action | Result |
|------|--------|--------|
| 1 | Add Lunatone IoT Gateway | Parallel DALI access |
| 2 | Keep KNX-DALI gateway for basic control | Or remove if desired |
| 3 | Configure Gray Logic DALI bridge | Full diagnostics |

---

## Summary

| Aspect | Our Approach |
|--------|--------------|
| **Physical reliability** | KNX relay actuators — wall switches always work |
| **DALI failsafe** | Power On Level = 100%, System Failure Level = 100% |
| **Intelligence layer** | Lunatone DALI-2 IoT Gateway with REST API |
| **Diagnostics** | Full DALI-2: voltage, current, runtime, temperature |
| **PHM capability** | Yes — trend analysis, predictive alerts |
| **Cost** | Comparable to KNX-DALI gateway approach |
| **Complexity** | Lower — no Lua scripting, no LogicMachine |

**The architecture ensures:**
1. Lights always work via wall switches (KNX physical layer)
2. Gray Logic provides intelligence without being a dependency
3. Full DALI-2 diagnostics enable genuine PHM
4. Simple, maintainable, cost-effective
