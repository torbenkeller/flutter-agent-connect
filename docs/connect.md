# fac connect

Verbindet den Client (im DevContainer) mit einem FAC-Server. Speichert die Verbindungsdaten in `~/.fac/config.json` für alle nachfolgenden Commands.

## Usage

```bash
fac connect [server-url] [flags]
```

## Arguments

| Argument | Default | Beschreibung |
|----------|---------|--------------|
| `server-url` | `http://host.docker.internal:8420` | URL des FAC-Servers |

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--agent` | string | auto-generated | Agent-Name (identifiziert diesen Container/User). Auto-generated: `agent-<random>` |
| `--port` | int | `8420` | Server-Port (Shortcut statt volle URL) |

## Was der Command tut

1. Sendet `GET /health` an den Server um die Verbindung zu prüfen
2. Registriert den Agent beim Server (`POST /agents`)
3. Speichert Server-URL und Agent-ID in `~/.fac/config.json`
4. Gibt Bestätigung aus mit Server-Info (Version, verfügbare Devices)

## Client Config

Nach `fac connect` wird `~/.fac/config.json` im Container geschrieben:

```json
{
  "server_url": "http://host.docker.internal:8420",
  "agent_id": "my-agent",
  "active_session_id": ""
}
```

- `agent_id` wird bei jedem Request mitgeschickt (als Header `X-Agent-ID`)
- `active_session_id` wird von `fac session create` / `fac session use` gesetzt
- Alle Commands lesen Server-URL und Agent-ID aus dieser Config

## API-Mapping

| Schritt | HTTP Request |
|---------|--------------|
| Connectivity Check | `GET /health` |

## Beispiel

```bash
# Standard (Docker Desktop / OrbStack auf dem Mac)
$ fac connect --agent feature-login
Connected to FAC Server v0.1.0
Agent: feature-login
Available devices: 12 iOS simulators

# Auto-generated Agent-Name
$ fac connect
Connected to FAC Server v0.1.0
Agent: agent-7f3a (auto-generated)

# Danach: alle Commands nutzen die gespeicherte Verbindung
$ fac session create --platform ios --name ios-main
```

## Voraussetzung: Volume Mount

Der Container muss die Projektdateien per Volume Mount mit dem Mac teilen:

```bash
# Docker
docker run -v /Users/torben/myapp:/workspace ...

# Docker Compose
volumes:
  - /Users/torben/myapp:/workspace

# DevContainer (devcontainer.json)
# Standardmäßig wird das Workspace-Verzeichnis automatisch gemountet
```

Dadurch sind Dateiänderungen im Container sofort auf dem Mac sichtbar — kein Sync nötig.

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Server nicht erreichbar | Error mit Hinweis: "Is `fac serve` running?" |
| Server-Version inkompatibel | Warnung (nicht blockierend) |
| `~/.fac/` existiert nicht | Wird automatisch angelegt |
