# fac logs

Zeigt App-Logs der laufenden Flutter-App an.

## Usage

```bash
fac logs [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--errors` | bool | `false` | Nur Errors und Exceptions anzeigen |
| `--lines` | int | `50` | Anzahl der letzten Log-Zeilen (0 = alle verfügbaren) |
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

1. Client fragt `GET /sessions/{id}/logs` beim Server an
2. Server gibt die gesammelten Logs zurück (aus dem stdout/stderr des `flutter run` Prozesses)
3. Client gibt die Logs formatiert aus

## API-Mapping

```
GET /sessions/{id}/logs?errors=false&lines=50
Authorization: Bearer <token>
```

**Response:**
```json
{
  "logs": [
    {"timestamp": "2026-03-16T21:35:01Z", "level": "info", "message": "flutter: Hello World"},
    {"timestamp": "2026-03-16T21:35:02Z", "level": "info", "message": "flutter: Button pressed"},
    {"timestamp": "2026-03-16T21:35:03Z", "level": "error", "message": "flutter: ══╡ EXCEPTION CAUGHT BY WIDGETS LIBRARY ╞══..."}
  ],
  "total_lines": 42
}
```

## Technischer Kontext

### Log-Quellen

Logs kommen aus zwei Quellen:

1. **flutter run stdout:** Alles was `flutter run --machine` auf stdout ausgibt, inklusive:
   - `app.log` Events (die eigentlichen App-Logs)
   - Build-Nachrichten
   - Framework-Warnungen

2. **Dart VM Service Logging Stream:** Strukturierte Log-Events aus dem `Logging` Stream der VM Service:
   ```json
   {"jsonrpc":"2.0","method":"streamNotify","params":{"streamId":"Logging","event":{"kind":"Logging","logRecord":{"message":"Button pressed","level":800}}}}
   ```

### Log-Level Mapping

| VM Service Level | FAC Level |
|-----------------|-----------|
| < 500 | `debug` |
| 500-799 | `info` |
| 800-899 | `warning` |
| >= 900 | `error` |

### Log-Buffer

Der Server hält die letzten N Log-Zeilen pro Session im Memory (Default: 1000 Zeilen). Ältere Logs werden verworfen. Für Phase 2 (WebSocket Streaming) werden Logs in Echtzeit an verbundene Clients gestreamt.

### Error-Filter

`--errors` filtert auf:
- Log-Level >= 900 (error)
- Zeilen die "EXCEPTION" oder "Error" enthalten
- Flutter Framework Exceptions (die ══╡ EXCEPTION CAUGHT ╞══ Blöcke)

## CLI Output

```bash
$ fac logs
[21:35:01] flutter: Hello World
[21:35:02] flutter: Button pressed
[21:35:03] flutter: User logged in

$ fac logs --errors
[21:35:15] ERROR: ══╡ EXCEPTION CAUGHT BY WIDGETS LIBRARY ╞══
           The following assertion was thrown building Text("..."):
           A RenderFlex overflowed by 42 pixels on the right.

$ fac logs --lines 5
[21:35:10] flutter: navigating to /home
[21:35:11] flutter: loading user data
[21:35:12] flutter: user data loaded
[21:35:13] flutter: rendering home screen
[21:35:14] flutter: home screen ready
```

## Hinweis: Streaming (Phase 2)

In Phase 1 ist `fac logs` ein One-Shot-Command (holt Logs, zeigt sie an, beendet sich). In Phase 2 kommt WebSocket-basiertes Streaming dazu — dann kann `fac logs --follow` Logs in Echtzeit streamen (wie `tail -f`).

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Zeigt letzte Logs vor dem Stop (falls vorhanden) |
| Keine Logs vorhanden | Leere Ausgabe mit Hinweis |
| Session existiert nicht | Error |
