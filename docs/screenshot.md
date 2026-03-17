# fac screenshot

Nimmt einen Screenshot der laufenden App oder des gesamten Simulator-Screens.

## Usage

```bash
fac screenshot [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `-o`, `--output` | string | `screenshot.png` | Pfad für die Output-Datei |
| `--device` | bool | `false` | Device-Level Screenshot (ganzer Simulator-Frame inkl. Status Bar) statt App-Level |
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

1. Client sendet `GET /sessions/{id}/screenshot` an den Server
2. Server nimmt Screenshot (App-Level oder Device-Level)
3. PNG-Daten werden als Response-Body zurückgegeben
4. Client schreibt PNG in die Output-Datei

## Zwei Screenshot-Modi

### App-Level (Default)
- Über den Dart VM Service: `_flutter.screenshot`
- Zeigt nur den gerenderten Flutter-Frame
- Keine Status Bar, keine System-UI
- Basis: Der Screenshot den die Flutter-Engine rendert

### Device-Level (`--device`)
- Über `xcrun simctl io <udid> screenshot --type=png -` (stdout)
- Zeigt den kompletten Simulator-Screen inklusive Status Bar, Navigation Bar, System-Dialoge
- Nützlich um System-Prompts (Permissions, Alerts) zu sehen

## API-Mapping

```
GET /sessions/{id}/screenshot?device=false
Authorization: Bearer <token>

Response:
Content-Type: image/png
Body: <raw PNG bytes>
```

Query Parameter:
| Parameter | Default | Beschreibung |
|-----------|---------|--------------|
| `device` | `false` | `true` für Device-Level Screenshot |

## Technischer Kontext

### VM Service Screenshot

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "ext.flutter.debugScreenshot",
  "params": {
    "isolateId": "isolates/123456"
  }
}
```

Alternativ über `_flutter.screenshot` Extension. Die Response enthält base64-encoded PNG.

### simctl Screenshot

```bash
xcrun simctl io <udid> screenshot --type=png /dev/stdout
```

Gibt PNG-Bytes auf stdout aus. Server captured diese und sendet sie als Response.

### Screenshot-Größe

- iPhone 16 Pro Simulator: ~1179 x 2556 Pixel (3x Retina)
- Typische Dateigröße: 200-800 KB je nach Inhalt
- Für den AI-Agenten ist das die richtige Auflösung — er muss Details erkennen können

## CLI Output

```bash
$ fac screenshot -o screen.png
Screenshot saved to screen.png (456 KB, 1179x2556)

$ fac screenshot --device -o full.png
Device screenshot saved to full.png (512 KB, 1179x2556)
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app" (für App-Level). Device-Level funktioniert trotzdem |
| VM Service disconnected | Fallback auf Device-Level Screenshot |
| Simulator nicht gebootet | Error "Simulator not running" |
| Output-Pfad nicht schreibbar | Error mit Pfad-Info |
