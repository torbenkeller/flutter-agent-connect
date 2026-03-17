# fac debug

Toggled Debug-Flags der laufenden Flutter-App. Equivalent zu den Toggles in Flutter DevTools.

## Subcommands

### fac debug paint

Toggled Debug Paint. Zeigt visuelle Hilfslinien für Layout-Debugging:
- Blaue Linien für Padding
- Gelbe Pfeile für Alignment
- Rote Balken für Overflow

```bash
fac debug paint
```

**VM Service Call:** `ext.flutter.debugPaint`

### fac debug repaint

Toggled Repaint Rainbow. Zeigt welche Bereiche des Screens bei jedem Frame neu gezeichnet werden — durch wechselnde Farb-Overlays.

```bash
fac debug repaint
```

**VM Service Call:** `ext.flutter.repaintRainbow`

### fac debug performance

Toggled Performance Overlay. Zeigt zwei Graphen oben im Screen:
- GPU-Thread Timing (oben)
- UI-Thread Timing (unten)
- Rote Balken = Frame hat zu lange gedauert (Jank)

```bash
fac debug performance
```

**VM Service Call:** `ext.flutter.showPerformanceOverlay`

## Gemeinsame Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |

## Toggle-Verhalten

Alle Debug-Commands sind Toggles:
- Erster Aufruf: **aktiviert** das Flag
- Zweiter Aufruf: **deaktiviert** das Flag

Der aktuelle State wird in der Response zurückgegeben.

## API-Mapping

| Command | HTTP Endpoint |
|---------|--------------|
| `fac debug paint` | `POST /sessions/{id}/debug/paint` |
| `fac debug repaint` | `POST /sessions/{id}/debug/repaint` |
| `fac debug performance` | `POST /sessions/{id}/debug/performance` |

**Response:**
```json
{
  "flag": "debugPaint",
  "enabled": true
}
```

## Technischer Kontext

### VM Service Extension Calls

Alle Debug-Flags werden über Flutter Service Extensions gesetzt:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "ext.flutter.debugPaint",
  "params": {
    "isolateId": "isolates/123456",
    "enabled": true
  }
}
```

Der Server trackt den aktuellen State der Flags pro Session, da die VM Service Extensions keine "get current value" API haben — nur toggles.

### Wann welches Flag nützlich ist

| Flag | Nützlich für |
|------|-------------|
| `paint` | Layout-Probleme, Overflow-Debugging, Padding/Margin-Verständnis |
| `repaint` | Performance-Optimierung — welche Widgets werden zu oft neu gezeichnet? |
| `performance` | Frame-Rate-Probleme, Jank-Erkennung, Scroll-Performance |

### Hinweis für Screenshots

Wenn Debug-Flags aktiviert sind, erscheinen die visuellen Overlays auch in Screenshots von `fac screenshot`. Das kann für den Agenten nützlich sein (er sieht die Debug-Info im Bild) oder störend. Der Agent sollte die Flags nach dem Debugging wieder deaktivieren.

## CLI Output

```bash
$ fac debug paint
Debug paint: enabled

$ fac debug paint
Debug paint: disabled

$ fac debug performance
Performance overlay: enabled
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app" |
| VM Service disconnected | Error |
