# fac devtools

Zugriff auf Flutter DevTools-Funktionalität über die CLI. Alles was die Flutter DevTools Web-App zeigt, als strukturierte JSON-Ausgabe.

## Commands

### fac devtools widgets

Widget Tree der laufenden App als JSON.

```bash
fac devtools widgets [--session <id>]
```

**Was es zeigt:** Die komplette Widget-Hierarchie — welche Widgets existieren, wie sie verschachtelt sind.

**VM Service Call:** `ext.flutter.debugDumpApp`

### fac devtools render

Render Tree als JSON. Enthält Layout-Informationen.

```bash
fac devtools render [--session <id>]
```

**Was es zeigt:** Wie Widgets gerendert werden — Größen, Positionen, Constraints, Overflow-Situationen.

**VM Service Call:** `ext.flutter.debugDumpRenderTree`

**Besonders nützlich für:** Layout-Debugging (Overflow-Errors), Constraints-Probleme.

### fac devtools semantics

Semantics Tree als JSON. **Dieser Tree wird auch für Widget-basiertes Tapping verwendet.**

```bash
fac devtools semantics [--session <id>]
```

**Was es zeigt:** Accessibility-Informationen — welche Elemente interaktiv sind, wo sie liegen, welche Actions sie unterstützen.

**VM Service Call:** `ext.flutter.debugDumpSemanticsTreeInTraversalOrder`

**Beispiel-Output:**
```json
{
  "nodes": [
    {
      "id": 5,
      "label": "Login",
      "rect": {"left": 150, "top": 380, "right": 243, "bottom": 420},
      "flags": ["isButton", "isFocusable"],
      "actions": ["tap"]
    },
    {
      "id": 6,
      "label": "Email",
      "rect": {"left": 20, "top": 200, "right": 373, "bottom": 256},
      "flags": ["isTextField"],
      "actions": ["tap", "setText"]
    }
  ]
}
```

### fac devtools performance

Toggle Performance Overlay. Zeigt Frame-Timing-Graphen.

```bash
fac devtools performance [--session <id>]
```

**VM Service Call:** `ext.flutter.showPerformanceOverlay`

**CLI Output:**
```
Performance overlay: enabled
```

### fac devtools paint

Toggle Debug Paint. Zeigt Layout-Hilfslinien (Padding, Alignment, Overflow).

```bash
fac devtools paint [--session <id>]
```

**VM Service Call:** `ext.flutter.debugPaint`

### fac devtools repaint

Toggle Repaint Rainbow. Zeigt welche Bereiche neu gezeichnet werden.

```bash
fac devtools repaint [--session <id>]
```

**VM Service Call:** `ext.flutter.repaintRainbow`

### fac devtools logs

App-Logs der laufenden Flutter-App.

```bash
fac devtools logs [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--errors` | bool | `false` | Nur Errors und Exceptions |
| `--lines` | int | `50` | Anzahl Zeilen |
| `--session` | string | aktive Session | Session ID oder Name |

**CLI Output:**
```
[21:35:01] flutter: Hello World
[21:35:02] flutter: Button pressed
[21:35:15] ERROR: RenderFlex overflowed by 42 pixels
```

**API-Mapping:** `GET /sessions/{id}/devtools/logs?errors=false&lines=50`

## Gemeinsame Flags

Alle devtools-Commands unterstützen:

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session ID oder Name |

## Toggle-Verhalten

`performance`, `paint` und `repaint` sind Toggles:
- Erster Aufruf: aktiviert
- Zweiter Aufruf: deaktiviert

Aktueller State wird in der Response zurückgegeben.

## Technischer Kontext

Alle DevTools-Commands nutzen den Dart VM Service Protocol (WebSocket JSON-RPC). Die Verbindung wird beim `fac flutter run` aufgebaut und bleibt bestehen.

Die Tree-Dumps (`widgets`, `render`, `semantics`) kommen als String vom VM Service. FAC parst diese in strukturiertes JSON.

## Perspektivisch: fac devtools network

Netzwerk-Requests der App anzeigen (HTTP-Calls, WebSocket-Verbindungen). Kommt in Phase 2.

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app. Use 'fac flutter run' first" |
| VM Service disconnected | Error mit Reconnect-Hinweis |
