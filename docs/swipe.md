# fac swipe

Führt eine Swipe-Geste auf dem Simulator-Screen aus.

## Usage

```bash
# Richtungs-basiert
fac swipe --up
fac swipe --down
fac swipe --left
fac swipe --right

# Koordinaten-basiert
fac swipe --from 200,400 --to 200,100
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--up` | bool | - | Swipe nach oben (Scrollen nach unten) |
| `--down` | bool | - | Swipe nach unten (Scrollen nach oben) |
| `--left` | bool | - | Swipe nach links |
| `--right` | bool | - | Swipe nach rechts |
| `--from` | string | - | Start-Koordinaten als `x,y` |
| `--to` | string | - | End-Koordinaten als `x,y` |
| `--duration` | int | `300` | Dauer der Geste in Millisekunden |
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

### Richtungs-basiert
- Server berechnet Start- und Endpunkt basierend auf Bildschirmmitte und Richtung
- z.B. `--down`: von Mitte-oben nach Mitte-unten (simuliert "Seite nach oben scrollen")
- Swipe-Distanz: ca. 2/3 der Screen-Höhe/-Breite

### Koordinaten-basiert
- Direkter Swipe von `--from` nach `--to`
- Nützlich für präzise Gesten (z.B. Slider, Dismiss-Gesten)

## API-Mapping

```
POST /sessions/{id}/swipe
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "from_x": 200,
  "from_y": 600,
  "to_x": 200,
  "to_y": 200,
  "duration_ms": 300
}
```

**Response:**
```json
{
  "success": true,
  "from": {"x": 200, "y": 600},
  "to": {"x": 200, "y": 200},
  "duration_ms": 300
}
```

## Technischer Kontext

### iOS Simulator Swipe

**Via AppleScript (MVP):**
Simuliert Click-and-Drag im Simulator-Window.

**Via adb (Android, Phase 3):**
```bash
adb shell input swipe <x1> <y1> <x2> <y2> <duration_ms>
```

### Richtungs-Mapping

| Flag | Bedeutung | Von → Nach (bei 393x852 Screen) |
|------|-----------|--------------------------------|
| `--up` | Content nach oben scrollen | (196, 600) → (196, 200) |
| `--down` | Content nach unten scrollen | (196, 200) → (196, 600) |
| `--left` | Content nach links | (300, 426) → (90, 426) |
| `--right` | Content nach rechts | (90, 426) → (300, 426) |

## CLI Output

```bash
$ fac swipe --down
Swiped down (196,200) → (196,600)

$ fac swipe --from 200,400 --to 200,100 --duration 500
Swiped (200,400) → (200,100) in 500ms
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Keine Richtung und keine Koordinaten | Error "Specify --up/--down/--left/--right or --from/--to" |
| Koordinaten außerhalb Screen | Error mit Screen-Dimensionen |
| App/Simulator läuft nicht | Error |
