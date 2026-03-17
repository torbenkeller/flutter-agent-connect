# fac inspect

Holt Inspektionsdaten der laufenden Flutter-App — Widget Tree, Render Tree oder Semantics Tree. Ersetzt die browserbasierten Flutter DevTools durch strukturierte CLI-Ausgaben.

## Subcommands

### fac inspect widgets

Gibt den Widget Tree als JSON aus.

```bash
fac inspect widgets [flags]
```

**Was es zeigt:** Die komplette Widget-Hierarchie der App — welche Widgets existieren, wie sie verschachtelt sind, ihre Properties.

**VM Service Call:** `ext.flutter.debugDumpApp`

**Beispiel-Output (gekürzt):**
```json
{
  "type": "WidgetTree",
  "root": {
    "widget": "MyApp",
    "children": [
      {
        "widget": "MaterialApp",
        "children": [
          {
            "widget": "Scaffold",
            "children": [
              {"widget": "AppBar", "children": [{"widget": "Text", "data": "My App"}]},
              {"widget": "Column", "children": [
                {"widget": "Text", "data": "Hello World"},
                {"widget": "ElevatedButton", "children": [{"widget": "Text", "data": "Login"}]}
              ]}
            ]
          }
        ]
      }
    ]
  }
}
```

### fac inspect render

Gibt den Render Tree als JSON aus. Enthält Layout-Informationen: Size, Constraints, Position, Overflow.

```bash
fac inspect render [flags]
```

**Was es zeigt:** Wie die Widgets tatsächlich gerendert werden — Größen, Positionen, Constraints, Overflow-Situationen.

**VM Service Call:** `ext.flutter.debugDumpRenderTree`

**Besonders nützlich für:**
- Layout-Debugging (Overflow-Errors)
- Verständnis der Constraints-Chain
- Prüfen ob Widgets die erwartete Größe haben

**Beispiel-Output (gekürzt):**
```json
{
  "type": "RenderTree",
  "root": {
    "renderer": "RenderView",
    "size": {"width": 393, "height": 852},
    "children": [
      {
        "renderer": "RenderFlex",
        "direction": "vertical",
        "size": {"width": 393, "height": 500},
        "constraints": {"minWidth": 0, "maxWidth": 393, "minHeight": 0, "maxHeight": 852},
        "children": [...]
      }
    ]
  }
}
```

### fac inspect semantics

Gibt den Semantics Tree als JSON aus. Enthält Accessibility-Informationen: Labels, Actions, Rects. **Dieser Tree wird auch für Widget-basiertes Tapping verwendet.**

```bash
fac inspect semantics [flags]
```

**Was es zeigt:** Wie die App für Accessibility (und FAC-Tapping) aussieht — welche Elemente interaktiv sind, wo sie liegen, welche Actions sie unterstützen.

**VM Service Call:** `ext.flutter.debugDumpSemanticsTreeInTraversalOrder`

**Beispiel-Output (gekürzt):**
```json
{
  "type": "SemanticsTree",
  "nodes": [
    {
      "id": 1,
      "label": "",
      "rect": {"left": 0, "top": 0, "right": 393, "bottom": 852},
      "flags": [],
      "actions": [],
      "children": [
        {
          "id": 5,
          "label": "Login",
          "rect": {"left": 150, "top": 380, "right": 243, "bottom": 420},
          "flags": ["isButton", "isFocusable"],
          "actions": ["tap"],
          "children": []
        },
        {
          "id": 6,
          "label": "Email",
          "rect": {"left": 20, "top": 200, "right": 373, "bottom": 256},
          "flags": ["isTextField", "isFocusable"],
          "actions": ["tap", "setText"],
          "children": []
        }
      ]
    }
  ]
}
```

## Gemeinsame Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |
| `--depth` | int | unbegrenzt | Maximale Tiefe des Trees (für übersichtlichere Ausgabe) |
| `--compact` | bool | `false` | Kompakte Ausgabe (weniger Details) |

## API-Mapping

| Command | HTTP Endpoint |
|---------|--------------|
| `fac inspect widgets` | `GET /sessions/{id}/inspect/widgets` |
| `fac inspect render` | `GET /sessions/{id}/inspect/render` |
| `fac inspect semantics` | `GET /sessions/{id}/inspect/semantics` |

Query Parameter:
| Parameter | Default | Beschreibung |
|-----------|---------|--------------|
| `depth` | -1 (unbegrenzt) | Maximale Tree-Tiefe |
| `compact` | `false` | Kompakte Ausgabe |

**Response:** JSON mit dem jeweiligen Tree.

## Technischer Kontext

### VM Service Extensions

Die Trees werden über Flutter Service Extensions abgerufen:

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "ext.flutter.debugDumpApp",
  "params": {"isolateId": "isolates/123456"}
}
```

Die Response ist ein String (formatierter Text-Dump). FAC muss diesen parsen und in strukturiertes JSON umwandeln. Alternativ gibt es `ext.flutter.inspector.getDetailsSubtree` das bereits strukturiertes JSON liefert.

### Semantics Tree für Tapping

Der Semantics Tree ist die Basis für `fac tap --label` / `fac tap --key`. Der `fac inspect semantics` Command gibt exakt die Daten zurück, die für das Tapping verwendet werden. Das hilft dem Agenten zu verstehen, welche Elemente verfügbar sind und welche Labels/Keys sie haben.

### Performance-Hinweis

Die Tree-Abfragen sind synchron und können bei komplexen Apps einige hundert Millisekunden dauern. Für den Agenten-Loop ist das akzeptabel. Die Trees sollten nicht in einer engen Schleife abgefragt werden.

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app" |
| VM Service disconnected | Error mit Reconnect-Hinweis |
| Semantics nicht aktiviert | Hinweis: "Semantics may not be fully available in release mode" |
