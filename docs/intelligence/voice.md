---
title: Voice Pipeline Specification
version: 1.0.0
status: active
last_updated: 2026-01-12
depends_on:
  - architecture/system-overview.md
  - architecture/core-internals.md
  - automation/automation.md
  - protocols/mqtt.md
---

# Voice Pipeline Specification

This document specifies Gray Logic's voice control system — local speech-to-text, natural language understanding, and text-to-speech for hands-free building control.

---

## Overview

### What is the Voice Pipeline?

The Voice Pipeline enables natural language control of Gray Logic systems:

1. **Wake word detection** — "Hey Gray" activates listening
2. **Speech-to-text (STT)** — Converts audio to text (Whisper)
3. **Natural language understanding (NLU)** — Extracts intent from text
4. **Command execution** — Executes the intended action
5. **Text-to-speech (TTS)** — Provides spoken feedback (Piper)

### Design Principles

1. **100% local processing** — No cloud dependency, no data leaves the site
2. **Privacy-first** — Audio never stored, processed in real-time only
3. **Graceful degradation** — System works perfectly with voice disabled
4. **Room-aware** — Commands interpreted in context of the room they're spoken in
5. **Natural language** — Users speak naturally, not command syntax
6. **Fast response** — <2 seconds from wake word to action confirmation

### Use Cases

| Use Case | Example |
|----------|---------|
| **Device control** | "Turn on the living room lights" |
| **Scene activation** | "Cinema mode" |
| **Mode changes** | "Set mode to away" |
| **Status queries** | "What's the temperature in here?" |
| **Announcements** | "Dinner is ready" (TTS broadcast) |
| **Doorbell integration** | "Someone is at the front door" (TTS) |
| **Fire alarm** | "Fire alarm activated. Please evacuate." (TTS) |

---

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    ROOM MICROPHONE ARRAY                     │
│              (USB/analog, per room or zone)                  │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             │ Audio Stream
                             │
┌────────────────────────────▼─────────────────────────────────┐
│                      VOICE BRIDGE                            │
│  ┌──────────────┬──────────────┬──────────────┐             │
│  │ Wake Word    │   Whisper    │    Piper     │             │
│  │  Detection   │     (STT)    │     (TTS)    │             │
│  └──────────────┴──────────────┴──────────────┘             │
└────────────────────────────┬─────────────────────────────────┘
                             │
                             │ MQTT: graylogic/voice/transcript
                             │ MQTT: graylogic/voice/intent
                             │
┌────────────────────────────▼─────────────────────────────────┐
│                    GRAY LOGIC CORE                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              NLU ENGINE (Local LLM)                     │ │
│  │  - Intent extraction                                    │ │
│  │  - Entity recognition (room, device, scene)             │ │
│  │  - Context resolution                                   │ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              COMMAND PROCESSOR                           │ │
│  │  - Validates intent                                     │ │
│  │  - Resolves targets                                     │ │
│  │  - Executes commands                                    │ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │              TTS COORDINATOR                             │ │
│  │  - Routes TTS requests to Voice Bridge                  │ │
│  │  - Manages announcement priorities                       │ │
│  └─────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

### Components

| Component | Responsibility | Technology |
|-----------|----------------|------------|
| **Voice Bridge** | Wake word detection, STT, TTS | Separate process (Python/Go) |
| **Wake Word Detector** | "Hey Gray" detection | Porcupine (Picovoice) or Vosk |
| **STT Engine** | Speech-to-text | Whisper (OpenAI, local) |
| **NLU Engine** | Intent extraction | Local LLM (Llama/Phi) or rule-based |
| **TTS Engine** | Text-to-speech | Piper (rhasspy) |
| **Command Processor** | Execute intents | Gray Logic Core |
| **TTS Coordinator** | Manage announcements | Gray Logic Core |

---

## Data Flow

### Voice Command Flow

```
1. User: "Hey Gray, turn on the living room lights"
   │
   ▼
2. Microphone captures audio
   │
   ▼
3. Wake Word Detector recognizes "Hey Gray"
   │
   ▼
4. Voice Bridge starts recording (3-5 seconds)
   │
   ▼
5. Whisper transcribes: "turn on the living room lights"
   │
   ▼
6. Voice Bridge publishes to MQTT:
   graylogic/voice/transcript
   {
     "room_id": "room-living",
     "transcript": "turn on the living room lights",
     "confidence": 0.95,
     "timestamp": "2026-01-12T10:30:00Z"
   }
   │
   ▼
7. NLU Engine extracts intent:
   {
     "intent": "device_control",
     "action": "turn_on",
     "targets": [
       {"device_id": "light-living-main"},
       {"device_id": "light-living-accent"}
     ],
     "room_id": "room-living",
     "confidence": 0.92
   }
   │
   ▼
8. Command Processor executes:
   - Publishes commands to MQTT
   - Devices turn on
   │
   ▼
9. TTS generates response:
   "Turning on the living room lights"
   │
   ▼
10. Piper speaks response in room
```

### TTS Announcement Flow

```
1. Event triggers announcement:
   - Doorbell pressed
   - Fire alarm activated
   - Automation: "Dinner is ready"
   │
   ▼
2. TTS Coordinator receives request:
   {
     "text": "Someone is at the front door",
     "priority": "high",
     "rooms": ["room-living", "room-kitchen"],
     "source": "doorbell"
   }
   │
   ▼
3. TTS Coordinator checks priorities:
   - Fire alarm > Doorbell > General announcement
   - Interrupts lower-priority TTS
   │
   ▼
4. TTS Coordinator publishes to MQTT:
   graylogic/voice/tts
   {
     "text": "Someone is at the front door",
     "rooms": ["room-living", "room-kitchen"],
     "voice": "en_GB-alba-medium"
   }
   │
   ▼
5. Voice Bridge generates audio:
   - Piper synthesizes speech
   - Audio stream sent to audio zones
   │
   ▼
6. Audio plays in specified rooms
```

---

## Wake Word Detection

### Configuration

```yaml
wake_word:
  enabled: true
  phrase: "hey gray"              # Default, customizable
  sensitivity: 0.7                 # 0.0-1.0, higher = more sensitive
  engine: "porcupine"              # porcupine | vosk
  
  # Porcupine (Picovoice) - recommended
  porcupine:
    access_key: "${PORCUPINE_KEY}" # Free tier available
    model_path: "/usr/share/porcupine/hey_gray.ppn"
    
  # Vosk (alternative, open source)
  vosk:
    model_path: "/usr/share/vosk/model-small"
```

### Wake Word Behavior

- **Continuous listening** — Always listening for wake word
- **Low CPU usage** — Wake word detection is lightweight
- **Room-aware** — Each microphone array has independent detection
- **Timeout** — If no speech after wake word, timeout after 5 seconds
- **Visual feedback** — LED on microphone or wall panel indicates listening

### Custom Wake Words

Users can customize wake word per room or globally:

```yaml
wake_words:
  global: "hey gray"
  rooms:
    - room_id: "room-kitchen"
      phrase: "hey kitchen"
    - room_id: "room-bedroom"
      phrase: "good morning"  # Only activates in morning hours
```

---

## Speech-to-Text (STT)

### Whisper Configuration

```yaml
stt:
  engine: "whisper"
  model: "base"                    # tiny | base | small | medium | large
  language: "en"                   # Auto-detect if null
  device: "cpu"                    # cpu | cuda | metal
  beam_size: 5
  best_of: 5
  temperature: 0.0
  compression_ratio_threshold: 2.4
  logprob_threshold: -1.0
  no_speech_threshold: 0.6
  
  # Model files
  model_path: "/usr/share/graylogic/whisper"
  
  # Performance
  chunk_length: 30                # Seconds per chunk
  max_audio_length: 30             # Max seconds to process
```

### Model Selection

| Model | Size | Speed | Accuracy | Use Case |
|-------|------|-------|----------|----------|
| **tiny** | 39 MB | Fastest | Good | Low-resource systems |
| **base** | 74 MB | Fast | Better | **Recommended default** |
| **small** | 244 MB | Medium | Better | Higher accuracy needed |
| **medium** | 769 MB | Slow | Best | High-end systems |
| **large** | 1550 MB | Slowest | Best | Maximum accuracy |

### STT Processing

1. **Audio capture** — 3-5 seconds after wake word
2. **Preprocessing** — Noise reduction, normalization
3. **Whisper transcription** — Converts audio to text
4. **Confidence scoring** — Returns confidence level
5. **Publish transcript** — MQTT message to Core

### Error Handling

```yaml
stt_errors:
  no_speech_detected:
    timeout: 5                     # Seconds
    response: "I didn't hear anything"
    
  low_confidence:
    threshold: 0.5
    response: "I'm not sure I understood. Could you repeat that?"
    
  too_long:
    max_length: 30
    response: "That was too long. Please try a shorter command."
```

---

## Natural Language Understanding (NLU)

### NLU Architecture

Two approaches supported:

1. **Local LLM** (Year 4+) — Full natural language understanding
2. **Rule-based** (Year 1-3) — Pattern matching with intent templates

### Rule-Based NLU (Initial Implementation)

```yaml
nlu:
  engine: "rule_based"            # rule_based | llm
  
  # Intent patterns
  intents:
    - name: "device_control"
      patterns:
        - "turn on {device}"
        - "turn off {device}"
        - "set {device} to {value}"
        - "{device} on"
        - "{device} off"
        - "dim {device} to {percent} percent"
        - "brighten {device}"
        
    - name: "scene_activate"
      patterns:
        - "{scene} mode"
        - "activate {scene}"
        - "set {scene}"
        - "{scene}"
        
    - name: "mode_change"
      patterns:
        - "set mode to {mode}"
        - "go to {mode} mode"
        - "activate {mode} mode"
        - "{mode} mode"
        
    - name: "status_query"
      patterns:
        - "what's the {parameter} in {room}"
        - "how {parameter} is it in {room}"
        - "what's the {parameter}"
        - "temperature in {room}"
        
    - name: "announcement"
      patterns:
        - "announce {text}"
        - "tell everyone {text}"
        - "say {text}"
```

### Entity Extraction

```yaml
entities:
  device:
    synonyms:
      - light, lights, lamp, lamps
      - fan, fans
      - blind, blinds, shade, shades
      - heater, heating
      - aircon, air conditioning, ac
      
  room:
    synonyms:
      - living room, lounge, living
      - kitchen
      - bedroom, bed
      - bathroom, bath
      - hallway, hall
      
  scene:
    synonyms:
      - cinema, movie, theater
      - reading, read
      - dinner, dining
      - good night, night, sleep
      
  mode:
    synonyms:
      - home
      - away
      - night
      - holiday
```

### LLM-Based NLU (Year 4+)

```yaml
nlu:
  engine: "llm"
  
  llm:
    model: "llama-3.2-3b"          # Local model
    provider: "ollama"              # ollama | llama.cpp
    context_window: 4096
    temperature: 0.1                # Low for deterministic intents
    
  # Prompt template
  prompt: |
    You are a home automation assistant. Extract the intent from this command.
    
    Available intents:
    - device_control: Control devices (lights, fans, blinds, etc.)
    - scene_activate: Activate a scene
    - mode_change: Change system mode
    - status_query: Query status (temperature, etc.)
    - announcement: Make an announcement
    
    Command: "{transcript}"
    Room: {room_name}
    
    Respond in JSON:
    {
      "intent": "device_control",
      "action": "turn_on",
      "targets": ["light-living-main"],
      "confidence": 0.95
    }
```

### Intent Resolution

```yaml
intent_resolution:
  # Ambiguous device resolution
  ambiguous_device:
    strategy: "ask_clarification"
    response: "Which light? The main light or the accent light?"
    
  # Room context
  room_context:
    use_current_room: true          # Default to room where spoken
    fallback_to_all_rooms: false
    
  # Scene name matching
  scene_matching:
    fuzzy: true                     # Allow "cinema" to match "Cinema Mode"
    threshold: 0.8
```

---

## Intent Types

### Device Control

```yaml
intent: device_control
examples:
  - "turn on the living room lights"
  - "turn off the kitchen fan"
  - "set the bedroom lights to 50 percent"
  - "dim the hallway lights"
  - "open the blinds"
  - "close the shades in the living room"
  - "set temperature to 22 degrees"

structure:
  action: turn_on | turn_off | set_level | dim | brighten | open | close | set_temperature
  targets: [device_id, ...]
  parameters:
    level: 0-100                    # For dimming/setting
    temperature: float              # For climate control
```

### Scene Activation

```yaml
intent: scene_activate
examples:
  - "cinema mode"
  - "reading mode"
  - "good night"
  - "dinner mode"
  - "activate cinema mode"

structure:
  scene_id: uuid
  scene_name: string
```

### Mode Change

```yaml
intent: mode_change
examples:
  - "set mode to away"
  - "go to night mode"
  - "activate holiday mode"
  - "home mode"

structure:
  mode: home | away | night | holiday
```

### Status Query

```yaml
intent: status_query
examples:
  - "what's the temperature in here"
  - "how warm is the living room"
  - "what's the temperature"
  - "is the heating on"
  - "what lights are on"

structure:
  parameter: temperature | humidity | heating_status | lighting_status
  room_id: uuid | null              # null = current room
  response_type: spoken | display   # TTS or show on wall panel
```

### Announcement

```yaml
intent: announcement
examples:
  - "announce dinner is ready"
  - "tell everyone the meeting starts in 5 minutes"
  - "say the pool is ready"

structure:
  text: string
  rooms: [room_id, ...]             # null = all rooms
  priority: low | normal | high
```

---

## Text-to-Speech (TTS)

### Piper Configuration

```yaml
tts:
  engine: "piper"
  
  piper:
    model: "en_GB-alba-medium"      # Voice model
    model_path: "/usr/share/piper/voices"
    sample_rate: 22050
    speed: 1.0                       # 0.5-2.0
    volume: 1.0                      # 0.0-1.0
    
  # Voice selection
  voices:
    default: "en_GB-alba-medium"
    per_room:
      - room_id: "room-living"
        voice: "en_GB-alba-medium"
      - room_id: "room-kitchen"
        voice: "en_US-lessac-medium"
        
  # Caching
  cache:
    enabled: true
    directory: "/var/cache/graylogic/tts"
    max_size_mb: 500
```

### Available Voices

| Voice | Language | Gender | Quality |
|-------|----------|--------|---------|
| `en_GB-alba-medium` | British English | Female | High |
| `en_US-lessac-medium` | US English | Female | High |
| `en_US-joe-medium` | US English | Male | High |
| `en_GB-northern_english_male-medium` | British English | Male | High |

### TTS Priorities

```yaml
tts_priorities:
  fire_alarm: 100                   # Highest
  security_alert: 90
  doorbell: 80
  announcement: 50
  command_response: 30              # Lowest
```

### TTS Routing

```yaml
tts_routing:
  # Route to audio zones
  audio_zones:
    enabled: true
    fallback_to_voice_bridge: false
    
  # Route to specific rooms
  per_room:
    enabled: true
    
  # Broadcast to all rooms
  broadcast:
    enabled: true
    exclude_rooms: ["room-bedroom"]  # Don't wake sleeping people
```

---

## Voice Bridge

### Architecture

The Voice Bridge is a separate process that handles:

1. **Wake word detection** — Continuous listening
2. **Audio capture** — Records after wake word
3. **STT processing** — Whisper transcription
4. **TTS synthesis** — Piper audio generation
5. **Audio routing** — Sends TTS to audio zones

### MQTT Topics

```yaml
mqtt_topics:
  # Voice Bridge → Core
  transcript: "graylogic/voice/transcript"
  tts_request: "graylogic/voice/tts/request"
  
  # Core → Voice Bridge
  tts_audio: "graylogic/voice/tts/audio"
  wake_word_config: "graylogic/voice/config/wake_word"
  
  # Status
  bridge_status: "graylogic/voice/bridge/status"
```

### Voice Bridge Status

```yaml
bridge_status:
  status: running | stopped | error
  wake_word_detected: true | false
  stt_processing: true | false
  tts_processing: true | false
  active_rooms: ["room-living", "room-kitchen"]
  last_activity: "2026-01-12T10:30:00Z"
```

### Configuration

```yaml
# /etc/graylogic/voice-bridge.yaml
voice_bridge:
  # MQTT connection
  mqtt:
    broker: "localhost:1883"
    client_id: "voice-bridge"
    
  # Microphones
  microphones:
    - id: "mic-living"
      room_id: "room-living"
      device: "/dev/audio0"
      channels: 1
      sample_rate: 16000
      
    - id: "mic-kitchen"
      room_id: "room-kitchen"
      device: "/dev/audio1"
      channels: 1
      sample_rate: 16000
      
  # Wake word
  wake_word:
    enabled: true
    phrase: "hey gray"
    
  # STT
  stt:
    engine: "whisper"
    model: "base"
    
  # TTS
  tts:
    engine: "piper"
    voice: "en_GB-alba-medium"
```

---

## Integration with Domains

### Lighting

```yaml
voice_lighting:
  intents:
    - "turn on the {room} lights"
    - "turn off the lights"
    - "dim the lights to {percent} percent"
    - "brighten the lights"
    
  device_resolution:
    - "lights" → all lights in room
    - "main light" → primary light fixture
    - "accent lights" → accent lighting
```

### Climate

```yaml
voice_climate:
  intents:
    - "set temperature to {degrees}"
    - "make it warmer"
    - "make it cooler"
    - "turn on the heating"
    - "turn off the air conditioning"
    - "what's the temperature"
    
  room_context:
    use_current_room: true
    fallback_to_zone: true
```

### Blinds

```yaml
voice_blinds:
  intents:
    - "open the blinds"
    - "close the blinds"
    - "set the blinds to {percent} percent"
    - "lower the shades"
    - "raise the blinds"
    
  device_resolution:
    - "blinds" → all blinds in room
    - "window" → blinds for that window
```

### Audio

```yaml
voice_audio:
  intents:
    - "play {source} in {room}"
    - "turn up the volume"
    - "turn down the volume"
    - "mute"
    - "unmute"
    
  # TTS integration
  tts_routing:
    use_audio_zones: true
    interrupt_music: true           # For high-priority announcements
```

### Security

```yaml
voice_security:
  intents:
    - "arm the alarm"
    - "disarm the alarm"
    - "set alarm to {mode}"         # home | away | night
    
  # Security requires PIN for voice commands
  security:
    require_pin: true
    pin_prompt: "Please enter your PIN"
    max_attempts: 3
```

---

## Privacy & Security

### Privacy Principles

1. **No audio storage** — Audio processed in real-time, never recorded
2. **No cloud processing** — All STT/NLU/TTS happens on-site
3. **No transcript logging** — Transcripts not stored (except for debugging with opt-in)
4. **Room-level privacy** — Voice commands only affect the room they're spoken in (unless explicitly broadcast)

### Security

```yaml
voice_security:
  # Authentication
  require_authentication: false      # Voice is local, no auth needed
  
  # PIN for sensitive commands
  sensitive_commands:
    - "arm the alarm"
    - "disarm the alarm"
    - "unlock the door"
    require_pin: true
    
  # Rate limiting
  rate_limit:
    commands_per_minute: 20
    wake_word_per_minute: 60
    
  # Voice profile (future)
  voice_profiles:
    enabled: false                   # Year 4+ feature
    recognition: false               # Not implemented yet
```

### Data Retention

```yaml
data_retention:
  transcripts: 0                    # Never stored
  audio_recordings: 0                # Never stored
  intent_logs: 0                    # Never stored (privacy)
  
  # Debug mode (opt-in only)
  debug_mode:
    enabled: false
    retention_hours: 24
    require_explicit_opt_in: true
```

---

## Error Handling

### Common Errors

```yaml
error_responses:
  no_wake_word:
    response: null                  # Silent, no response
    
  no_speech_after_wake:
    timeout: 5
    response: "I didn't hear anything"
    
  low_confidence:
    threshold: 0.5
    response: "I'm not sure I understood. Could you repeat that?"
    
  ambiguous_intent:
    response: "Which {device_type}? {options}"
    example: "Which light? The main light or the accent light?"
    
  device_not_found:
    response: "I couldn't find {device_name} in {room_name}"
    
  scene_not_found:
    response: "I don't know a scene called {scene_name}"
    
  permission_denied:
    response: "I can't do that. {reason}"
    example: "I can't do that. Security commands require a PIN."
    
  command_failed:
    response: "Sorry, I couldn't {action} the {device}. {error}"
    
  system_error:
    response: "Sorry, there was an error. Please try again."
```

### Fallback Behavior

```yaml
fallbacks:
  # If NLU fails, try simple pattern matching
  nlu_fallback:
    enabled: true
    use_keyword_matching: true
    
  # If device not found, suggest alternatives
  device_suggestions:
    enabled: true
    fuzzy_matching: true
    threshold: 0.7
    
  # If voice disabled, commands fail gracefully
  voice_disabled:
    response: null                  # Silent failure, no error
    log_only: true
```

---

## Performance Requirements

### Latency Targets

| Stage | Target | Maximum |
|-------|--------|---------|
| Wake word detection | <100ms | 200ms |
| STT processing | <1s | 2s |
| NLU processing | <500ms | 1s |
| Command execution | <500ms | 1s |
| TTS generation | <500ms | 1s |
| **Total (wake → action)** | **<2s** | **<4s** |

### Resource Usage

```yaml
resource_limits:
  # CPU
  wake_word_cpu: 5%                 # Per microphone
  stt_cpu: 50%                       # During processing
  nlu_cpu: 30%                       # During processing
  tts_cpu: 20%                       # During synthesis
  
  # Memory
  whisper_model_memory: 500MB        # base model
  nlu_model_memory: 2GB              # llama-3.2-3b
  tts_cache_memory: 100MB
  
  # Disk
  model_storage: 5GB                 # All models
  tts_cache: 500MB
```

---

## Configuration

### Global Voice Settings

```yaml
# /etc/graylogic/voice.yaml
voice:
  enabled: true
  
  # Wake word
  wake_word:
    phrase: "hey gray"
    sensitivity: 0.7
    
  # STT
  stt:
    engine: "whisper"
    model: "base"
    language: "en"
    
  # NLU
  nlu:
    engine: "rule_based"             # rule_based | llm
    llm_model: "llama-3.2-3b"
    
  # TTS
  tts:
    engine: "piper"
    voice: "en_GB-alba-medium"
    
  # Privacy
  privacy:
    store_transcripts: false
    store_audio: false
    debug_mode: false
```

### Room-Specific Configuration

```yaml
# Per-room voice settings
voice_rooms:
  - room_id: "room-living"
    wake_word: "hey gray"
    tts_voice: "en_GB-alba-medium"
    enabled: true
    
  - room_id: "room-bedroom"
    wake_word: "hey gray"
    tts_voice: "en_GB-alba-medium"
    enabled: false                   # Disabled at night
    schedule:
      enabled: false
      times:
        - start: "22:00"
          end: "07:00"
```

---

## API Endpoints

### Submit Voice Command

```http
POST /api/v1/voice/command
```

**Request:**
```json
{
  "transcript": "turn on the living room lights",
  "context": {
    "room_id": "room-living",
    "user_id": "usr-001"
  }
}
```

**Response (200):**
```json
{
  "success": true,
  "intent": {
    "type": "device_control",
    "action": "turn_on",
    "targets": [
      { "device_id": "light-living-main" },
      { "device_id": "light-living-accent" }
    ]
  },
  "response_text": "Turning on the living room lights",
  "actions_executed": 2
}
```

**Response (Clarification Needed):**
```json
{
  "success": false,
  "intent": {
    "type": "ambiguous"
  },
  "response_text": "Which light? The main light or the accent light?",
  "options": [
    { "label": "Main Light", "device_id": "light-living-main" },
    { "label": "Accent Light", "device_id": "light-living-accent" }
  ]
}
```

### Request TTS Announcement

```http
POST /api/v1/voice/announce
```

**Request:**
```json
{
  "text": "Dinner is ready",
  "rooms": ["room-living", "room-kitchen"],
  "priority": "normal"
}
```

**Response (200):**
```json
{
  "success": true,
  "announcement_id": "ann-12345",
  "status": "queued"
}
```

### Voice Status

```http
GET /api/v1/voice/status
```

**Response (200):**
```json
{
  "enabled": true,
  "wake_word": "hey gray",
  "active_rooms": ["room-living", "room-kitchen"],
  "bridge_status": "running",
  "last_command": "2026-01-12T10:30:00Z"
}
```

---

## Hardware Requirements

### Microphones

| Type | Use Case | Connection | Notes |
|------|----------|------------|-------|
| **USB microphone array** | Per-room | USB | Recommended, good quality |
| **Analog microphone + ADC** | Custom install | Analog input | Requires ADC hardware |
| **IP microphone** | Remote rooms | Network | ONVIF audio stream |
| **Smart speaker** | Integration | API | Sonos, etc. (limited) |

### Recommended Hardware

```yaml
recommended_hardware:
  # Microphones
  usb_mics:
    - "ReSpeaker 4-Mic Array"
    - "Seeed Studio 6-Mic Array"
    - "Matrix Creator (8 mics)"
    
  # Processing
  minimum_cpu: "4 cores"
  recommended_cpu: "8 cores"
  minimum_ram: "4GB"
  recommended_ram: "8GB"
  
  # GPU (optional, for faster STT)
  gpu:
    cuda: true                       # NVIDIA GPU
    metal: true                      # Apple Silicon
```

---

## Commissioning

### Setup Workflow

1. **Install Voice Bridge** — Deploy voice bridge process
2. **Connect microphones** — Physical installation per room
3. **Configure wake word** — Set default or custom phrase
4. **Test STT** — Verify Whisper transcription works
5. **Test NLU** — Verify intent extraction
6. **Test TTS** — Verify Piper synthesis
7. **Train users** — Show available commands
8. **Fine-tune** — Adjust sensitivity, voice models

### Testing Checklist

```yaml
commissioning_tests:
  wake_word:
    - [ ] Wake word detected reliably
    - [ ] False positives < 1 per hour
    - [ ] Response time < 200ms
    
  stt:
    - [ ] Transcription accuracy > 90%
    - [ ] Handles accents/dialects
    - [ ] Works in noisy environments
    
  nlu:
    - [ ] Intent extraction > 85% accuracy
    - [ ] Device resolution correct
    - [ ] Room context applied correctly
    
  tts:
    - [ ] Audio quality acceptable
    - [ ] Latency < 1s
    - [ ] Routing to correct rooms
    
  integration:
    - [ ] Commands execute correctly
    - [ ] Scenes activate via voice
    - [ ] Modes change via voice
    - [ ] TTS announcements work
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| **Wake word not detected** | Sensitivity too low | Increase sensitivity |
| **False wake words** | Sensitivity too high | Decrease sensitivity |
| **Poor STT accuracy** | Wrong model, noise | Use larger model, improve mic placement |
| **Slow response** | CPU overload | Use smaller models, add GPU |
| **TTS not playing** | Audio routing issue | Check audio zone configuration |
| **Commands not executing** | NLU failure | Check intent patterns, enable debug |

### Debug Mode

```yaml
debug:
  enabled: true
  log_level: "debug"
  
  # Logging
  log_transcripts: true
  log_intents: true
  log_audio_samples: false           # Privacy: don't log audio
  
  # Metrics
  track_latency: true
  track_accuracy: true
```

---

## Roadmap

### Year 1-2: Foundation

- [x] Wake word detection (Porcupine/Vosk)
- [x] STT with Whisper
- [x] Rule-based NLU
- [x] TTS with Piper
- [x] Basic device control
- [x] Scene activation
- [x] Room-aware commands

### Year 3: Enhancement

- [ ] Multi-language support
- [ ] Custom wake words per room
- [ ] Voice profiles (user recognition)
- [ ] Improved NLU accuracy
- [ ] Contextual follow-up questions

### Year 4: Intelligence

- [ ] LLM-based NLU (local)
- [ ] Natural conversation
- [ ] Complex queries ("why is it cold?")
- [ ] Proactive suggestions
- [ ] Learning from corrections

---

## Related Documents

- [System Overview](../architecture/system-overview.md) — Overall architecture
- [Core Internals](../architecture/core-internals.md) — NLU Engine implementation
- [Automation](../automation/automation.md) — Voice-triggered automation
- [API Specification](../interfaces/api.md) — Voice API endpoints
- [Audio Domain](../domains/audio.md) — TTS routing to audio zones
- [Principles](../overview/principles.md) — Privacy and offline-first rules
