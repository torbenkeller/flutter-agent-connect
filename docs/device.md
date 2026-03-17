# fac device

Interaktion mit dem Simulator/Emulator — Screenshots, Touch-Events, Text-Eingabe.

## Commands

### fac device screenshot

Nimmt einen Screenshot des Simulator-Screens. Gibt nur den Dateipfad auf stdout aus (machine-parseable für AI-Agenten).

```bash
fac device screenshot [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `-o`, `--output` | string | Temp-Datei | Output-Dateipfad |
| `--session` | string | aktive Session | Session ID oder Name |

**CLI Output:** Nur der Dateipfad.
```
/tmp/fac-screenshot-abc123.png
```

**Agent-Workflow:**
```bash
SCREENSHOT=$(fac device screenshot)
# Agent liest das Bild via Read-Tool
```

**API-Mapping:** `GET /sessions/{id}/device/screenshot`
Response: `Content-Type: image/png` mit PNG-Bytes als Body.

### fac device tap

Tap auf ein Element — per Semantics-Label, Widget-Key, oder Pixel-Koordinaten.

```bash
# Widget-basiert (bevorzugt)
fac device tap --label "Login"
fac device tap --key "submitButton"

# Pixel-Koordinaten (Fallback)
fac device tap <x> <y>
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--label` | string | - | Semantics-Label des Elements |
| `--key` | string | - | Widget-Key |
| `--index` | int | `0` | Bei mehreren Matches: welches Element (0-basiert) |
| `--session` | string | aktive Session | Session ID oder Name |

**Widget-basierter Tap (empfohlen):**
1. Server holt Semantics Tree via VM Service
2. Sucht Element nach Label oder Key
3. Berechnet Mitte des Bounding Rects
4. Führt Tap an diesen Koordinaten aus

**CLI Output:**
```
Tapped "Login" at (195, 400)
```

**API-Mapping:** `POST /sessions/{id}/device/tap`
```json
{"label": "Login", "index": 0}
{"key": "submitButton"}
{"x": 195, "y": 400}
```

### fac device swipe

Swipe-Geste auf dem Simulator-Screen.

```bash
fac device swipe --down
fac device swipe --up
fac device swipe --left
fac device swipe --right
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--up/--down/--left/--right` | bool | - | Swipe-Richtung |
| `--duration` | int | `300` | Dauer in Millisekunden |
| `--session` | string | aktive Session | Session ID oder Name |

**API-Mapping:** `POST /sessions/{id}/device/swipe`
```json
{"direction": "down", "duration_ms": 300}
```

### fac device type

Text in das fokussierte Textfeld eingeben.

```bash
fac device type <text> [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--clear` | bool | `false` | Feld vorher leeren |
| `--enter` | bool | `false` | Enter drücken nach Eingabe |
| `--session` | string | aktive Session | Session ID oder Name |

**API-Mapping:** `POST /sessions/{id}/device/type`
```json
{"text": "user@example.com", "clear": true, "enter": false}
```

**Typischer Workflow:**
```bash
fac device tap --label "Email"
fac device type "user@example.com"
fac device tap --label "Password"
fac device type "secret123" --enter
```

## Technischer Kontext

### Screenshots
Device-Level Screenshots via `xcrun simctl io <udid> screenshot --type=png -`. Gibt den ganzen Simulator-Frame inkl. Status Bar zurück.

### Taps (iOS)
Phase 1: Über AppleScript an das Simulator.app Window. Später: Facebook idb für robusteres Tapping.

### Koordinaten-System
Semantics Tree gibt Koordinaten in logischen Pixeln (z.B. 393x852 für iPhone 16 Pro). FAC rechnet diese in die Simulator-Window-Koordinaten um.

## Error Cases

| Fehler | Command | Verhalten |
|--------|---------|-----------|
| Element nicht gefunden | `tap --label` | Error + Liste verfügbarer Labels |
| Simulator nicht gebootet | alle | Error "Simulator not running" |
| Kein Textfeld fokussiert | `type` | Warnung (Text wird trotzdem gesendet) |
