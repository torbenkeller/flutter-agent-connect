# fac status

Zeigt den aktuellen Status der Verbindung, der aktiven Session und der App.

## Usage

```bash
fac status [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |
| `--json` | bool | `false` | Ausgabe als JSON (für programmatische Nutzung) |

## Was der Command tut

1. Prüft Client-Config (`~/.fac/config.json`)
2. Pingt den Server (`GET /health`)
3. Holt Session-Status (`GET /sessions/{id}`)
4. Gibt alles formatiert aus

## API-Mapping

| Schritt | HTTP Request |
|---------|--------------|
| Server-Status | `GET /health` |
| Session-Status | `GET /sessions/{id}` |

**Health Response:**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "active_sessions": 2,
  "available_simulators": 10
}
```

**Session Response:**
```json
{
  "id": "a1b2c3d4-...",
  "platform": "ios",
  "state": "running",
  "device": {
    "udid": "D59AF284-...",
    "name": "iPhone 16 Pro",
    "runtime": "iOS 18.4",
    "state": "Booted"
  },
  "app": {
    "state": "running",
    "target": "lib/main.dart",
    "started_at": "2026-03-16T21:35:00Z",
    "last_reload_at": "2026-03-16T21:40:00Z",
    "reload_count": 5
  },
  "sync": {
    "last_sync_at": "2026-03-16T21:40:00Z",
    "files_synced": 42,
    "total_bytes": 156000
  },
  "debug_flags": {
    "paint": false,
    "repaint": false,
    "performance": false
  }
}
```

## CLI Output

```bash
$ fac status
Server:  http://192.168.1.50:8420 (connected, v0.1.0)
Session: a1b2c3d4 (ios, iPhone 16 Pro)
Device:  Booted
App:     Running (lib/main.dart)
         Started 5 min ago, 5 reloads
Sync:    42 files, last sync 30s ago
Debug:   paint=off, repaint=off, performance=off

# Wenn nicht verbunden:
$ fac status
Server:  Not connected
Run 'fac connect <url> --token <token>' to connect

# Wenn verbunden aber keine Session:
$ fac status
Server:  http://192.168.1.50:8420 (connected, v0.1.0)
Session: None
Run 'fac session create --platform ios' to create a session
```

## Kontext

Dieser Command ist besonders nützlich für:
- Den Agenten, um schnell den aktuellen Zustand zu prüfen
- Debugging wenn etwas nicht funktioniert
- Nach einer Unterbrechung: "Wo war ich?"

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Keine Config vorhanden | Hinweis auf `fac connect` |
| Server nicht erreichbar | "Server unreachable" + letzte bekannte Info |
| Session existiert nicht mehr | Hinweis, Config bereinigen |
