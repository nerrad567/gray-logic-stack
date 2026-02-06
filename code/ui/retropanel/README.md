# Retro Panel — Dockable LVGL Control Panel

A 1960s/70s-themed dockable wall panel for Gray Logic. Snaps onto a PoE wall
dock for permanent installation, lifts off for portable use on battery + WiFi.
$83/room complete vs $2,500+ for equivalent Crestron hardware.

## Product Concept

```
     WALL DOCK (dumb)                    PANEL (smart)
    ┌──────────────────┐               ┌──────────────────────┐
    │                  │   magnetic    │                      │
    │  Cat5e ──> PoE   │   snap-on    │  ESP32-S3 + WiFi     │
    │  splitter        │◄════════════►│  480x320 IPS touch   │
    │                  │  6 pogo pins  │  LiFePO4 battery     │
    │  5V + ETH data   │  (power+data) │  LD2410 presence     │
    │  out via pogo    │               │  W5500 (when docked) │
    │                  │               │  USB-C (portable)    │
    └──────┬───────────┘               └──────────────────────┘
           │                                    │
       mounts in                           lift off and
       standard                            carry to any room
       wall box
```

**Docked on wall**: Charges via PoE, uses wired Ethernet, screen always on,
fixed to one room.

**Portable (lifted off)**: Runs on battery, switches to WiFi, room picker
appears, charge via USB-C or Qi pad.

---

## Operating Modes

| Mode | Display | Network | Power | Trigger |
|------|---------|---------|-------|---------|
| **Docked — Active** | On (full brightness) | Ethernet | PoE (charging battery) | Default when docked |
| **Docked — Idle** | Backlight dimmed | Ethernet | PoE (charging battery) | No presence for 60s |
| **Portable — Active** | On | WiFi | Battery | Touch or LD2410 presence |
| **Portable — Idle** | Backlight off | WiFi (modem sleep) | Battery | No presence for 30s |
| **Portable — Sleep** | Off | Off (deep sleep) | Battery (~70uA) | Battery <15%, wake on touch |

### Battery Life Targets

**Design principle: maximise battery life at every level.** The panel should
last a full day of portable use without charging, and weeks in sleep mode.

| Usage Pattern | Estimated Life | Notes |
|---------------|---------------|-------|
| Active (screen on, WiFi) | ~4 hours | 5000mAh LiFePO4 at ~3.2W |
| Mixed (30% active, 70% idle) | ~8 hours | Typical portable day |
| Idle (backlight off, WiFi modem sleep) | ~40 hours | On a shelf, WiFi connected |
| Deep sleep (wake on touch) | ~8 months | Stored in drawer |

### Battery Life Optimisation Strategies

Every milliamp matters. These are built into the firmware:

1. **Backlight is the #1 drain (~1.5W)** — off when no presence detected
2. **WiFi modem sleep** — between DTIM beacons, radio powers down (~20mA vs ~150mA)
3. **CPU frequency scaling** — 80MHz when idle, 240MHz only during renders
4. **LVGL refresh rate** — 10fps for static content, 30fps only during interaction
5. **W5500 powered down** when undocked (saves ~60mA)
6. **Dark amber theme** — lower backlight brightness still readable (already done)
7. **Peripheral power gating** — LD2410 can be duty-cycled in portable mode
8. **Deep sleep as last resort** — only when battery critically low (<15%)

### Network Handover (Dock ↔ Portable)

```
Undocking:
1. Panel detects VCC pogo pin voltage drop → "undocked" event
2. W5500 Ethernet powered down (saves ~60mA)
3. WiFi STA starts connecting (pre-configured credentials)
4. WiFi connects (~1-2 seconds)
5. MQTT client reconnects via WiFi
6. UI shows room picker (no longer locked to dock's room)

Re-docking:
1. Panel detects VCC pogo pin voltage → "docked" event
2. W5500 Ethernet powered up, link established
3. Network switches to Ethernet (lower latency)
4. WiFi optionally powers down (faster battery charging)
5. UI auto-loads the dock's configured room
```

---

## Platform 1: SDL Simulator (Development)

Use this for all UI development. No hardware needed.

### Requirements

- Linux (tested on Ubuntu/Mint 24.04) or macOS
- GCC, CMake 3.16+, SDL2
- Optional: libcurl, libmosquitto, libcjson (for networking to live Core)

### Setup

```bash
# Install build dependencies
sudo apt install build-essential cmake libsdl2-dev

# Optional: networking libraries (enables live connection to Core)
sudo apt install libcurl4-openssl-dev libmosquitto-dev libcjson-dev

# Clone and build
cd code/ui/retropanel
make setup        # fetches LVGL v9.2.2 (~30s first time)
make sim-build    # builds SDL simulator binary
make sim-run      # builds and runs
```

### Configuration (SDL)

The simulator reads config from **environment variables**:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `GRAYLOGIC_URL` | No | `http://localhost:8090` | Core server URL |
| `GRAYLOGIC_TOKEN` | For live mode | — | Panel authentication token |
| `GRAYLOGIC_ROOM` | For live mode | — | Room ID to display |
| `GRAYLOGIC_MQTT_HOST` | No | `localhost` | MQTT broker host |
| `GRAYLOGIC_MQTT_PORT` | No | `1883` | MQTT broker port |

**Demo mode** (no env vars): Hardcoded "Living Room" with sample devices.
**Live mode** (token + room set):

```bash
cd code/core && make dev-services && make dev-run   # Start Core
GRAYLOGIC_TOKEN=your-panel-token GRAYLOGIC_ROOM=room-living-1 make sim-run
```

---

## Platform 2: ESP32-S3 (Production Hardware)

### The Dock

The wall dock is a simple breakout board in a standard wall plate:

```
   Back of wall plate              Front of wall plate
   (behind drywall)                (flush with wall)

   ┌─────────────────┐            ┌─────────────────┐
   │ Cat5e punch-down │            │  ○ ○ ○ ○ ○ ○   │ ← 6 pogo pins
   │ PoE splitter     │            │  (magnetic      │   (recessed)
   │ (48V → 5V)       │            │   alignment)    │
   │ ETH magnetics    │            │                 │
   │ Room ID EEPROM   │            │                 │
   └─────────────────┘            └─────────────────┘
                                   Fits standard 2-gang wall box
```

**6 pogo pins:**

| Pin | Signal | Purpose |
|-----|--------|---------|
| 1 | VCC (5V) | Power from PoE splitter |
| 2 | GND | Ground |
| 3 | ETH TX+ | Ethernet transmit |
| 4 | ETH TX- | Ethernet transmit |
| 5 | ETH RX+ | Ethernet receive |
| 6 | ETH RX- | Ethernet receive |

The panel detects docking via VCC presence. W5500 Ethernet on the panel
connects to pins 3-6 through on-panel magnetics.

### The Panel

The handheld unit contains all the intelligence:

| Component | Purpose |
|-----------|---------|
| ESP32-S3-WROOM-1 (16MB/8MB) | CPU, WiFi, Bluetooth |
| 3.5" 480x320 IPS + GT911 touch | Display and input |
| W5500 SPI Ethernet | Wired network (when docked) |
| LiFePO4 battery (5000mAh) | Portable power, 20-year safe chemistry |
| AXP2101 PMIC | Multi-input charging (PoE/USB-C/Qi), fuel gauge |
| LD2410C mmWave radar | Presence detection for backlight control |
| USB-C port | Portable charging + firmware flashing |
| Qi receiver coil | Wireless charging on pad (portable mode) |
| 6-pin magnetic pogo connector | Dock interface |

### Charging Options

| Method | When | How |
|--------|------|-----|
| **PoE via dock** | Wall-mounted | Automatic — snap on, charges continuously |
| **USB-C** | Desk, bedside, commissioning | Any phone charger |
| **Qi wireless pad** | Kitchen counter, nightstand | Panel lies flat on Qi pad |

Note: Qi through drywall doesn't work (needs <8mm gap). Qi on a pad works
perfectly because the panel sits directly on it.

### Build and Flash

```bash
# 1. Install ESP-IDF v5.2+ (one-time, ~20 minutes)
mkdir -p ~/esp && cd ~/esp
git clone -b v5.2.2 --recursive https://github.com/espressif/esp-idf.git
cd esp-idf && ./install.sh esp32s3
source ~/esp/esp-idf/export.sh

# 2. Cross-compile on your PC
cd code/ui/retropanel/esp_project
idf.py set-target esp32s3
idf.py build                    # ~2 min first build, ~10s incremental

# 3. Flash via USB
idf.py -p /dev/ttyUSB0 flash

# 4. Monitor serial (optional, debugging)
idf.py -p /dev/ttyUSB0 monitor
```

### First Boot Configuration

The panel's own touchscreen shows a setup wizard:
1. Scan and select WiFi network, enter password
2. Enter Gray Logic Core server URL
3. Enter panel authentication token
4. Select default room (or leave as "roaming")

Config stored in NVS flash — survives reboots and power cycles.
Triple-tap screen corner to re-enter setup wizard.

---

## Prototype Parts List

### Phase A: Panel Prototype (proof of concept)

Order these first. Goal: get the retro UI running on real hardware with
battery and WiFi.

| # | Component | Specific Part | Qty | Price | Source | Notes |
|---|-----------|---------------|-----|-------|--------|-------|
| 1 | Display board | Elecrow CrowPanel Advance 3.5" HMI (ESP32-S3-WROOM-1 N16R8) | 1 | ~$25 | elecrow.com | 480x320 IPS, cap touch, 16MB flash, 8MB PSRAM, mic+speaker, modular wireless slot |
| 2 | Display board alt | Waveshare ESP32-S3-Touch-LCD-3.5 | 1 | ~$26 | waveshare.com / Amazon | 480x320 IPS, cap touch (FT6336), 16MB/8MB, IMU + RTC + audio codec onboard |
| 3 | mmWave sensor | HLK-LD2410C (24GHz, BLE config) | 2 | ~$3 ea | AliExpress | UART + GPIO, BLE for sensitivity tuning, ~79mA draw. Get the C variant (not LD2410/LD2410B) |
| 4 | LiFePO4 cells | JGNE 26650 3.2V 4000mAh | 2 | ~$4 ea | 18650batterystore.com | 26.5x65mm, 2000+ cycles. 2.7x capacity of 18650 for 8mm wider. Reputable brand |
| 5 | Battery holder | Single 26650 holder with leads | 2 | ~$1 ea | AliExpress | For prototype wiring |
| 6 | Charge module | TP5000 3.6V/4.2V selectable (LiFePO4 mode) | 3 | ~$2 ea | AliExpress / Amazon | **NOT TP4056** — TP4056 is fixed 4.2V in silicon, cannot be modified. Set TP5000 solder jumper to 3.6V |
| 7 | USB-C breakout | USB-C female breakout board | 2 | ~$2 ea | AliExpress | For charging port on panel |
| 8 | Dupont wires | Male-female jumper wires 40pc | 1 | ~$3 | AliExpress | For connecting modules |
| 9 | Breadboard | Half-size solderless | 2 | ~$3 ea | AliExpress | For prototyping circuits |
| 10 | LoRa module (optional) | SX1262 868MHz for Elecrow wireless slot | 1 | ~$6 | AliExpress | Enables Meshtastic mesh networking. See `docs/resilience/mesh-comms.md` |
| | | | | **~$86** | | |

**Board choice notes:** Both the Elecrow and Waveshare are ESP32-S3 with
8MB PSRAM and IPS capacitive touch — either works. Buy one of each to
evaluate. The Sunton CYD (ESP32-3248S035) is NOT suitable — it uses a
regular ESP32 (not S3), has a TN panel (not IPS), and no PSRAM.

**LiFePO4 charging: TP5000, not TP4056.** The TP4056 IC has a fixed 4.2V
charge termination voltage hardcoded in silicon — there is no resistor
mod that changes it. The TP5000 has a solder pad to select 3.6V (LiFePO4)
or 4.2V (Li-ion). Verify the jumper is set to 3.6V before connecting a
battery. TP5000 boards do NOT include battery protection — add a separate
BMS or use CN3058E boards (dedicated LiFePO4 charger with protection).

**26650 vs 18650:** LiFePO4 18650 cells top out at ~1500mAh. The 26650
form factor (8mm wider) gives 4000mAh — 2.7x the runtime for minimal
size increase. Well worth it for a wall panel.

### Phase B: Dock Prototype (PoE + pogo pins)

Order once Phase A panel is working. Goal: magnetic dock with PoE charging
and Ethernet pass-through.

| # | Component | Specific Part | Qty | Price | Source | Notes |
|---|-----------|---------------|-----|-------|--------|-------|
| 10 | Ethernet module | W5500 SPI mini module | 2 | ~$3 ea | AliExpress | For panel Ethernet |
| 11 | PoE splitter | 802.3af to 5V/2.4A | 2 | ~$8 ea | AliExpress/Amazon | IEEE compliant, barrel or screw |
| 12 | Pogo pins (panel) | 6-pin spring-loaded pogo | 2 | ~$3 ea | AliExpress | Search "magnetic pogo connector" |
| 13 | Pogo pads (dock) | 6-pin flat contact pads | 2 | ~$3 ea | AliExpress | Matching counterpart |
| 14 | Magnets | 10x3mm neodymium disc | 10 | ~$5 | AliExpress | For alignment, 4 per dock/panel |
| 15 | ETH magnetics | HanRun HR911105A or similar | 2 | ~$2 ea | AliExpress | RJ45 with built-in magnetics |
| 16 | Wall box | Standard 2-gang back box (UK/EU) | 2 | ~$3 ea | Local electrical | For mounting dock |
| 17 | Prototype PCB | 5x7cm double-sided perfboard | 5 | ~$3 | AliExpress | For dock circuit |
| 18 | PoE switch | 4-port 802.3af PoE switch | 1 | ~$30 | Amazon | For testing, any brand |
| | | | | **~$75** | | |

### Phase C: Portable Charging (Qi + polish)

Order once dock is working. Goal: Qi charging for portable use, refined
power management.

| # | Component | Specific Part | Qty | Price | Source | Notes |
|---|-----------|---------------|-----|-------|--------|-------|
| 19 | Qi receiver | 5V/1A Qi receiver module | 2 | ~$5 ea | AliExpress | Thin coil + circuit board |
| 20 | Qi transmitter | 5W/10W Qi charging pad | 1 | ~$10 | Amazon | For testing, any brand |
| 21 | AXP2101 module | AXP2101 PMIC breakout | 2 | ~$5 ea | AliExpress | Multi-input, fuel gauge |
| 22 | Larger battery | EEMB LP7568130F 3.2V LiFePO4 5000mAh pouch | 2 | ~$10-15 ea | eemb.com (contact sales) | 7.8x68.5x131mm flat. Hard to source small qty — 26650 cylindrical as fallback |
| | | | | **~$55** | | |

**LiFePO4 pouch cell sourcing:** Pouch cells in 3-5Ah LiFePO4 are
industrial products with limited consumer availability. EEMB is a
reputable manufacturer (UL1642 listed) — contact sales@eemb.com for
small-quantity pricing. If pouch cells prove difficult to source, two
JGNE 26650 cells in parallel (8000mAh total) work well for prototyping.

### Total Prototype Budget

| Phase | Cost | Timeline |
|-------|------|----------|
| A: Panel prototype | ~$80 | Order now, 2-3 weeks delivery |
| B: Dock prototype | ~$75 | Order when A works |
| C: Qi + power polish | ~$55 | Order when B works |
| **Total** | **~$210** | For 2 complete prototypes |

### Tools Required

You likely already have most of these:

| Tool | Needed For | Price if missing |
|------|-----------|-----------------|
| Soldering iron (temperature controlled) | Wiring modules | ~$30 |
| Multimeter | Voltage/current testing | ~$15 |
| Hot glue gun | Mounting magnets, securing wires | ~$10 |
| 3D printer (optional) | Enclosure prototypes | Borrow/library |
| USB-C cables (2+) | Flashing + charging | ~$5 |
| Cat5e patch cables (2+) | Ethernet testing | ~$5 |

---

## Architecture

```
code/ui/retropanel/
├── CMakeLists.txt              # Top-level build (LVGL + app)
├── Makefile                    # Developer workflow (sim-build, sim-run)
├── README.md                   # This file
│
├── sdl_simulator/              # SDL2 desktop target
│   ├── CMakeLists.txt          # Links SDL2 + networking libs
│   ├── main.c                  # SDL entry point + main loop
│   └── lv_conf.h               # LVGL config (32bpp, SDL backend)
│
├── esp_project/                # ESP32-S3 target (Phase 4+)
│   ├── main/main.c             # FreeRTOS entry + WiFi + display init
│   ├── main/lv_conf.h          # LVGL config (16bpp, PSRAM)
│   └── partitions.csv          # Flash layout
│
├── components/lvgl/            # LVGL v9.2.2 (auto-fetched by make setup)
│
├── src/                        # Shared source (both platforms)
│   ├── app.h / app.c           # Boot sequence, tick loop, dock detection
│   ├── theme/                  # Retro visual theme
│   │   ├── retro_colors.h      # Amber/cream/olive palette
│   │   ├── retro_theme.h/.c    # LVGL theme + font accessors
│   │   └── *.c                 # Generated font files (Nixie, mono)
│   ├── widgets/                # Custom retro widgets
│   │   ├── vu_meter.h/.c       # Arc gauge with tick marks
│   │   ├── nixie_display.h/.c  # Glowing numeric readout
│   │   ├── bakelite_btn.h/.c   # Vintage toggle button
│   │   ├── scene_bar.h/.c      # Scene activation row
│   │   ├── blind_slider.h/.c   # Position slider
│   │   └── scanline_overlay.h/.c # CRT effect
│   ├── screens/
│   │   ├── scr_room_view.h/.c  # Main room control screen
│   │   ├── scr_room_selector.h/.c # Room picker (portable mode)
│   │   └── scr_settings.h/.c   # First-boot wizard + settings
│   ├── data/
│   │   ├── data_model.h/.c     # C structs + demo data
│   │   └── data_store.h/.c     # In-memory state store
│   ├── net/
│   │   ├── panel_config.h/.c   # Configuration loading
│   │   ├── rest_client.h/.c    # HTTP GET/PUT/POST
│   │   ├── mqtt_client.h/.c    # MQTT subscribe + ring buffer
│   │   └── command.h/.c        # Device commands + scene activate
│   └── hal/
│       ├── hal_power.h         # Battery, charging, dock detection
│       ├── hal_presence.h      # LD2410 presence sensor
│       ├── hal_backlight.h     # Display backlight control
│       └── hal_network.h       # WiFi/Ethernet handover
│
├── assets/fonts/               # Source TTF files
└── tools/font_convert.sh       # LVGL font generator wrapper
```

### Platform abstraction

| Concern | SDL Simulator | ESP32-S3 |
|---------|---------------|----------|
| Display | SDL2 window | SPI/parallel to LCD |
| Touch | SDL2 mouse | I2C to GT911/FT5x06 |
| Network | Host OS WiFi | WiFi + W5500 Ethernet (handover) |
| HTTP | libcurl | esp_http_client |
| MQTT | libmosquitto | esp_mqtt |
| Config | Environment variables | NVS flash + setup wizard |
| Memory | System malloc | PSRAM allocator |
| Threading | pthreads | FreeRTOS tasks |
| Power | N/A (always on) | AXP2101 PMIC, LiFePO4 battery |
| Presence | N/A (always active) | LD2410C mmWave radar |
| Backlight | N/A | PWM dimming, presence-controlled |
| Dock detect | N/A | VCC pogo pin voltage sense |

---

## Roadmap

### Software Development (SDL Simulator)

| Phase | Status | Description |
|-------|--------|-------------|
| 1. Visual Theme | Done | Retro UI: Nixie fonts, VU meters, bakelite buttons, scanlines |
| 2. Data Layer | Done | REST boot loading, MQTT live updates, cJSON parsing |
| 3. Interactive Controls | Done | Touch events send commands via REST, optimistic UI |
| 4. Room Selector Screen | Planned | Grid of rooms for portable mode |
| 5. Settings Screen | Planned | First-boot wizard: WiFi, server, token, room |
| 6. Power UI | Planned | Battery indicator, charging icon, WiFi/Ethernet icon |
| 7. Polish & Effects | Planned | VU sweep animation, nixie glow pulse, screen warm-up |
| 8. Meshtastic Integration | Planned | LoRa mesh node, secure command channel, repeater mode, fallback routing |

### Hardware Prototyping

| Phase | Status | Description |
|-------|--------|-------------|
| A. Panel Prototype | Order parts | ESP32-S3 board + battery + LD2410, port firmware |
| B. Dock Prototype | After A | PoE wall plate with pogo pins, Ethernet pass-through |
| C. Qi + Power | After B | Wireless charging, AXP2101 multi-input, power optimisation |
| D. Enclosure | After C | 3D printed case: hand-holdable, magnetic dock alignment |

### Manufacturing Preparation (once prototype proven)

| Phase | Description |
|-------|-------------|
| M1. Custom PCB design | Single-board panel: ESP32-S3 + display + battery + all peripherals |
| M2. Dock PCB design | Minimal board: PoE splitter + magnetics + pogo connector |
| M3. Enclosure design | Injection mould or CNC aluminium, wall plate fitment |
| M4. Certification | CE/FCC testing for WiFi + Ethernet emissions |
| M5. Small batch (10 units) | First production run for own home + beta testers |
| M6. Production tooling | If demand warrants, optimise for volume |

---

## Design Decisions Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Battery chemistry | LiFePO4 | 2,000-7,000 cycles, safe in sealed enclosure, 20-year viable |
| Primary charging | PoE via dock | One cable, professional, always charging |
| Portable charging | USB-C + Qi pad | USB-C is universal; Qi for nightstand/kitchen use |
| Wireless through wall | Rejected | Qi max gap 2-8mm; drywall is 10-15mm with potential metal |
| Presence sensor | LD2410C mmWave | Detects stationary people (PIR can't), $5, UART + GPIO |
| Sleep strategy | Backlight off (not deep sleep) | WiFi must stay connected for MQTT real-time updates |
| Deep sleep | Emergency only (<15% battery) | Loses WiFi — only for battery preservation |
| Network handover | Ethernet when docked, WiFi when portable | esp_netif handles both transparently |
| PMIC | AXP2101 | Multi-input (PoE/USB-C/Qi), fuel gauge, 15 power rails |
| Prototype charger | TP5000 (solder jumper to 3.6V) | TP4056 is fixed 4.2V in silicon — cannot charge LiFePO4. TP5000 has selectable voltage |
| Dock-to-panel data | 6 pogo pins (power + Ethernet) | Minimal, reliable, magnetic alignment |
| Room context | Dock stores room ID; portable shows picker | Natural UX for both modes |
| Dev board | Elecrow CrowPanel / Waveshare (ESP32-S3) | Sunton CYD rejected: regular ESP32, TN panel, no PSRAM. Both alternatives have S3 + 8MB PSRAM + IPS |
| Battery format | 26650 LiFePO4 (4000mAh) over 18650 (1500mAh) | 2.7x capacity for 8mm wider diameter. Worth it for panel that needs all-day runtime |
| LoRa / Meshtastic | Optional — Elecrow has modular slot for SX1262 | Panel becomes a mesh node + repeater. Secure control when WiFi is down. See `docs/resilience/mesh-comms.md` |
