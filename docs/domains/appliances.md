# Appliance Domain Specification

This document defines the integration strategy for "White Goods" (Washing Machines, Dryers, Dishwashers, Fridges, Ovens) within the Gray Logic Stack.

**Philosophy**: We prioritize **availability** and **autonomy**. A smart dishwasher that requires a cloud server to start is a liability, not an asset.

---

## 1. Integration Strategy

We classify appliance integration into three accepted tiers. **Cloud-only APIs are strictly prohibited** for architectural dependencies.

### Tier 1: Native Local (Matter)
The gold standard for modern appliances.
*   **Protocol**: Matter over Thread or Matter over Wi-Fi.
*   **Capabilites**: Full status feedback (Time remaining, Cycle phase), Remote Start, Error codes.
*   **Requirement**: Device must function 100% when Internet Gateway is blocked at the firewall.

### Tier 2: Industrial / SG Ready (Dry Contacts)
The gold standard for reliability and " Dumb" smarts.
*   **Protocol**: Physical GPIO / KNX Binary Output → Appliance Input.
*   **Standard**: **SG Ready** (Smart Grid Ready) or manufacturer-specific "External Start" contacts.
*   **Behavior**:
    1.  User loads appliance and presses "Start".
    2.  Appliance pauses and waits for "Go" signal (contact closure).
    3.  Core closes contact when energy is cheap/solar is high.
*   **Pros**: 100% reliable, zero software updates to break.
*   **Cons**: Limited feedback (Binary: Running/Not Running).

### Tier 3A: Monitoring Only (Digital "Dumb")
For modern "dumb" appliances with digital push-buttons (Momentary Switches).
*   **Limitation**: Cutting power resets the computer. Restoration puts it in "Standby", not "Run".
*   **Role**: **Notification Only**.
    *   **Logic**: Smart plug monitors power. When power drops to <3W for 5 mins → Send "Dishwasher Finished" notification.
    *   **Automation**: **None**. You cannot remote start these safely.

### Tier 3B: Power Latching (Mechanical "Dumb")
For older appliances or industrial units with physical latching knobs/switches.
*   **Method**: High-current contactor on the supply.
*   **Behavior**:
    1.  User sets dial to "Cotton 40°".
    2.  User leaves machine "On".
    3.  System cuts power immediately.
    4.  System restores power at 2 AM → Machine resumes where it left off.
*   **Safety Warning**: Never use this on devices that need a cool-down fan (e.g., Induction hobs, confident digital ovens).

### Tier 4: Mechanical Bot (Retrofit)
The "Last Resort" for digital dumb appliances.
*   **Hardware**: SwitchBot or similar finger-bot.
*   **Integration**: Bluetooth/Matter bridge.
*   **Behavior**: Physical button press simulation.
*   **Reliability**: Low. (Adhesive failure, battery death). **Not recommended for critical paths.**

---

### Tier 5: Cloud API (Supplemental)
For appliances where no local option exists, but the user accepts the trade-offs.
*   **Examples**: Samsung SmartThings, LG ThinQ, Home Connect (Bosch/Siemens).
*   **Status**: **Supported but not Guaranteed**.
*   **Role**: Good for energy monitoring and remote start signal.
*   **Risks**:
    *   **Internet Dependency**: Will fail during internet outages.
    *   **API Deprecation**: Manufacturers often change APIs without warning.
    *   **Latency**: Control loops are slower (1-5s).
*   **Policy**: We allow these for convenience, but they must not be relied upon for critical safety or "Core" infrastructure promises.

---

## 3. Discouraged Methods

We **strongly advise against**:
1.  **Unofficial/Reverse-Engineered APIs**: Unstable local hacks that break on firmware updates.
    *   *Reason*: High maintenance burden. Use only as a last resort.

---

## 3. Specifications for New Deployments

When specifying appliances for a Gray Logic home, strictly adhere to this list:

| Category | Recommended Spec | Integration Method | Notes |
| :--- | :--- | :--- | :--- |
| **Dishwasher** | Miele (Pro / SG Ready) | Miele Gateway / SG Ready | Look for "FlexControl" or Solar inputs. |
| **Washing Machine** | Miele / Asko | SG Ready Contact | Often hidden in service menus or rear terminal blocks. |
| **Heat Pump Wrapper** | SG Ready Certified | Dry Contact (x2) | State 1: Normal, State 2: Overdrive (Solar Dump). |
| **Generic** | Matter Certified | IPv6 Local | Verify "Local Operation" on box. |

---

## 4. Automation Logic

### The "Price-Aware Start"

```yaml
automation:
  - id: "dishwasher-optimization"
    trigger:
      - type: "state_change"
        entity: "input.dishwasher_ready"
        to: "on"
    action:
      - service: "scheduler.schedule_device"
        data:
          device_id: "switch.dishwasher_trigger"
          strategy: "lowest_cost"
          window:
            start: "now"
            end: "07:00"
          duration_minutes: 180  # Eco cycle length
```

### The "Solar Dump"

```yaml
automation:
  - id: "solar-excess-trigger"
    trigger:
      - type: "numeric_state"
        entity: "sensor.grid_export_power"
        above: 2000 # 2kW excess
        for: "10m"
    condition:
      - condition: "state"
        entity: "input.washing_machine_ready"
        state: "on"
    action:
      - service: "switch.turn_on"
        entity: "switch.washing_machine_start"
```

---

## 5. Commissioning Test

1.  **Setup**: Load appliance, enable remote start/SG ready mode.
2.  **Sever WAN**: Unplug internet router.
3.  **Trigger**: Manually trigger "Start Now" from Gray Logic Core dashboard.
4.  **Verify**: Appliance must start immediately.
