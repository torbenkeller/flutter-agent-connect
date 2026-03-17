# fac tap

Führt einen Tap (Touch) auf dem Simulator-Screen aus. Unterstützt Widget-basiertes Tapping über den Semantics Tree sowie Pixel-Koordinaten als Fallback.

## Usage

```bash
# Widget-basiert (bevorzugt)
fac tap --label "Login"
fac tap --key "submitButton"

# Pixel-Koordinaten (Fallback)
fac tap <x> <y>
```

## Arguments

| Argument | Beschreibung |
|----------|--------------|
| `x y` | Pixel-Koordinaten für den Tap (nur wenn kein --label/--key) |

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--label` | string | - | Semantics-Label des Elements (z.B. Button-Text, Accessibility-Label) |
| `--key` | string | - | Widget-Key (ValueKey/Key das im Flutter-Code gesetzt wurde) |
| `--index` | int | `0` | Bei mehreren Matches: welches Element (0-basiert) |
| `--session` | string | aktive Session | Session-ID |

## Priorität der Tap-Modi

1. `--label` → sucht im Semantics Tree nach Label
2. `--key` → sucht im Semantics Tree nach Key
3. `x y` Argumente → direkte Pixel-Koordinaten

## Was der Command tut

### Widget-basiert (--label / --key)

1. Server holt den Semantics Tree via VM Service (`ext.flutter.debugDumpSemanticsTreeInTraversalOrder`)
2. Durchsucht den Tree nach dem Element mit passendem Label oder Key
3. Berechnet die Mitte des Bounding Rects des Elements
4. Führt Tap an diesen Koordinaten aus (via `osascript` auf iOS Simulator)
5. Gibt zurück: gefundenes Element, Koordinaten, Erfolg

### Pixel-basiert (x y)

1. Server führt Tap direkt an den gegebenen Koordinaten aus
2. Kein Semantics-Lookup nötig

## API-Mapping

```
POST /sessions/{id}/tap
Content-Type: application/json
Authorization: Bearer <token>
```

**Request Body (Widget-basiert):**
```json
{
  "label": "Login",
  "index": 0
}
```

oder:
```json
{
  "key": "submitButton"
}
```

**Request Body (Pixel-basiert):**
```json
{
  "x": 195,
  "y": 400
}
```

**Response:**
```json
{
  "success": true,
  "tapped_at": {"x": 195, "y": 400},
  "element": {
    "label": "Login",
    "rect": {"left": 150, "top": 380, "right": 240, "bottom": 420},
    "actions": ["tap"]
  }
}
```

## Technischer Kontext

### Semantics Tree Lookup

Der Semantics Tree kommt als verschachtelte Struktur vom VM Service. Jeder Knoten hat:
- `id`: Semantics-ID
- `label`: Text-Label (z.B. Button-Text, Accessibility-Label)
- `rect`: Bounding Box `{left, top, right, bottom}` in logischen Pixeln
- `actions`: Liste von Aktionen die das Element unterstützt (`tap`, `longPress`, `scrollUp`, etc.)
- `flags`: Semantics-Flags (`isButton`, `isTextField`, `isHeader`, etc.)
- `children`: Kind-Knoten

Die Suche nach `--label` matcht gegen das `label` Feld (Case-insensitive, Substring-Match).
Die Suche nach `--key` matcht gegen den `identifier` oder `value` des Knotens.

### Tap-Ausführung auf iOS Simulator

Für iOS Simulatoren gibt es mehrere Wege einen Tap auszuführen:

**Option 1: AppleScript (MVP)**
```bash
osascript -e '
  tell application "Simulator"
    activate
  end tell
  tell application "System Events"
    click at {x, y} of window 1 of application "Simulator"
  end tell
'
```

**Option 2: Facebook idb (robuster, spätere Phase)**
```bash
idb ui tap <x> <y> --udid <udid>
```

**Option 3: simctl (falls verfügbar)**
Neuere Versionen von `xcrun simctl` könnten Touch-Events unterstützen.

### Koordinaten-Systeme

- **Logische Pixel**: Die Koordinaten im Semantics Tree sind in logischen Pixeln (device-independent). Ein iPhone 16 Pro hat 393 x 852 logische Pixel.
- **Physische Pixel**: Der Simulator-Screen hat 1179 x 2556 physische Pixel (3x Scale Factor).
- **Simulator-Window**: Die Simulator.app zeigt den Screen in einem macOS-Window, möglicherweise skaliert.

FAC muss die Koordinaten korrekt zwischen diesen Systemen umrechnen. Der Scale Factor kommt aus `xcrun simctl list -j` (unter `devicetypes`).

## CLI Output

```bash
$ fac tap --label "Login"
Tapped "Login" at (195, 400)

$ fac tap --label "Login" --index 1
Tapped "Login" (2nd match) at (195, 600)

$ fac tap --key "submitButton"
Tapped element [submitButton] at (200, 450)

$ fac tap 100 300
Tapped at (100, 300)

# Wenn Element nicht gefunden:
$ fac tap --label "NonExistent"
Error: No element found with label "NonExistent"
Available elements: "Login", "Register", "Forgot Password"
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Element nicht gefunden (--label/--key) | Error mit Liste der verfügbaren Labels/Keys |
| Mehrere Matches ohne --index | Nimmt das erste Match, gibt Warnung aus |
| App läuft nicht | Error "No running app" |
| Koordinaten außerhalb Screen | Error mit Screen-Dimensionen |
| Simulator-Window nicht im Vordergrund | AppleScript bringt es in den Vordergrund |
