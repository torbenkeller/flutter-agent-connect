# fac reload

Führt einen Hot Reload der laufenden Flutter-App durch. Da Dateien per Volume Mount geteilt werden, sind Änderungen bereits auf dem Mac — es wird nur der Reload getriggert.

## Usage

```bash
fac reload [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

1. Client sendet `POST /sessions/{id}/app/reload` an den Server
2. Server ruft `reloadSources` auf dem Dart VM Service auf
3. Ergebnis wird zurückgegeben (Erfolg/Fehler)

Kein Sync-Schritt — Dateien sind durch Volume Mount schon auf dem Mac.

## Hot Reload vs. Hot Restart

- **Hot Reload** (`fac reload`): Nur geänderter Dart-Code wird injiziert. State bleibt erhalten. Schnell (~300ms). Funktioniert nicht bei Änderungen an `main()`, global initializers, enums, generic types.
- **Hot Restart** (`fac restart`): App wird komplett neu gestartet. State geht verloren. Langsamer (~2s). Funktioniert immer.

## API-Mapping

```
POST /sessions/{id}/app/reload
```

**Response:**
```json
{
  "success": true,
  "reload_duration_ms": 287
}
```

## Technischer Kontext

### VM Service reloadSources

Der Server sendet über die WebSocket-Verbindung zum Dart VM Service:
```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "reloadSources",
  "params": {
    "isolateId": "isolates/123456",
    "force": false
  }
}
```

### Wann Hot Reload fehlschlägt

Hot Reload kann fehlschlagen wenn die Änderungen inkompatibel sind. In dem Fall gibt der Server die Fehlermeldung weiter und schlägt `fac restart` vor.

## CLI Output

```bash
$ fac reload
Hot reload successful (287ms)

# Bei Fehler:
$ fac reload
Hot reload failed: Changed enum 'UserRole' requires hot restart
Hint: Run 'fac restart' instead
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app. Use 'fac app start' first" |
| Hot Reload fehlgeschlagen | Error mit Reason + Hinweis auf `fac restart` |
| VM Service disconnected | Versucht Reconnect, dann Error |
