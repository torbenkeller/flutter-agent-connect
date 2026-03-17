# fac restart

Führt einen Hot Restart der laufenden Flutter-App durch. Im Gegensatz zu `fac reload` wird die App komplett neu gestartet — State geht verloren, aber alle Arten von Code-Änderungen werden angewendet.

## Usage

```bash
fac restart [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |

## Was der Command tut

1. Client sendet `POST /sessions/{id}/app/restart` an den Server
2. Server sendet `app.restart` mit `fullRestart: true` an den `flutter run` Prozess via stdin
3. Nach dem Restart bekommt die App eine neue VM Service URI → Server reconnected automatisch
4. Ergebnis wird zurückgegeben

## API-Mapping

```
POST /sessions/{id}/app/restart
```

**Response:**
```json
{
  "success": true,
  "files_synced": 3,
  "restart_duration_ms": 1850,
  "message": "Restart successful"
}
```

## Technischer Kontext

### flutter run stdin Command

```json
[{"id":1,"method":"app.restart","params":{"appId":"abc-123","fullRestart":true,"pause":false}}]
```

Response auf stdout:
```json
[{"id":1,"result":{"code":0,"message":"Success"}}]
```

### VM Service URI ändert sich

Nach einem Hot Restart bekommt die Dart VM einen neuen Port. Der `flutter run` Prozess gibt ein neues `app.debugPort` Event auf stdout aus. Der Server muss:
1. Alte WebSocket-Verbindung schließen
2. Auf neues `app.debugPort` Event warten
3. Neue WebSocket-Verbindung aufbauen

## CLI Output

```bash
$ fac restart
Syncing 5 files (3.2 KB)...
Hot restart successful (1.8s)
```

## Wann Restart statt Reload?

| Änderung | Reload reicht? | Restart nötig? |
|----------|---------------|----------------|
| Widget build() Methode geändert | Ja | - |
| Neues Widget hinzugefügt | Ja | - |
| Styling/Layout geändert | Ja | - |
| Enum geändert | - | Ja |
| main() geändert | - | Ja |
| Globale Variablen-Initializer | - | Ja |
| Native Code (iOS/Android) geändert | - | Nein (Full Rebuild nötig: `fac app stop` → `fac app start`) |

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| App läuft nicht | Error "No running app. Use 'fac app start' first" |
| Restart fehlgeschlagen | Error mit Details aus flutter run |
| VM Service Reconnect Timeout | Error "Could not reconnect to VM Service after restart" |
