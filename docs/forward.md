# fac forward

Leitet einen Port aus dem Container zum Mac weiter, sodass Simulatoren/Emulatoren das Backend erreichen können. Registriert optional eine Dart-Define-Variable, die automatisch bei `fac flutter run` injiziert wird.

## Usage

```bash
fac forward <container-port> [flags]
```

## Arguments

| Argument | Beschreibung |
|----------|--------------|
| `container-port` | Port auf dem das Backend im Container läuft (z.B. 8080) |

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `-e`, `--env` | string | - | Name der Dart-Define-Variable (z.B. `BACKEND_URL`) |
| `--session` | string | aktive Session | Session ID oder Name |

## Was der Command tut (Phase 1 — lokal)

1. Fragt Docker nach dem Host-Port: `docker port <container-id> <port>`
2. Speichert das Mapping: `BACKEND_URL → host-port`
3. Bei `fac flutter run` wird automatisch injiziert:
   - iOS: `--dart-define=BACKEND_URL=http://localhost:<host-port>`
   - Android: `--dart-define=BACKEND_URL=http://10.0.2.2:<host-port>`

## Voraussetzung

Der Container muss den Port exponieren. In `devcontainer.json`:
```json
{
  "forwardPorts": [8080]
}
```

Oder in `docker-compose.yml`:
```yaml
services:
  agent:
    ports:
      - "0:8080"   # Docker wählt freien Host-Port
```

## Beispiel

```bash
# 1. Backend im Container starten
cd backend && dart run bin/server.dart --port 8080

# 2. Port forwarden und Variable registrieren
fac forward 8080 -e BACKEND_URL
# → Forwarding :8080 → :9001 (BACKEND_URL)

# 3. Optional: zweiten Service forwarden
fac forward 6379 -e REDIS_URL
# → Forwarding :6379 → :9002 (REDIS_URL)

# 4. App starten — Dart Defines werden automatisch injiziert
fac flutter run
# → intern: flutter run --dart-define=BACKEND_URL=http://localhost:9001
#                        --dart-define=REDIS_URL=http://localhost:9002
```

## In der Flutter-App

```dart
const backendUrl = String.fromEnvironment(
  'BACKEND_URL',
  defaultValue: 'http://localhost:8080',
);
```

## Multi-Agent Isolation

Docker weist jedem Container automatisch verschiedene Host-Ports zu:

```
Container A: backend :8080 → Docker → Mac :9001
Container B: backend :8080 → Docker → Mac :9002

Agent A: fac forward 8080 -e BACKEND_URL → BACKEND_URL=http://localhost:9001
Agent B: fac forward 8080 -e BACKEND_URL → BACKEND_URL=http://localhost:9002
```

Keine Konflikte — jede Session hat ihre eigene URL.

## API-Mapping

```
POST /sessions/{id}/forward
```

**Request:**
```json
{
  "container_port": 8080,
  "env_name": "BACKEND_URL"
}
```

**Response:**
```json
{
  "container_port": 8080,
  "host_port": 9001,
  "env_name": "BACKEND_URL",
  "url_ios": "http://localhost:9001",
  "url_android": "http://10.0.2.2:9001"
}
```

## Forwarded Ports auflisten

```bash
fac forward list
# CONTAINER  HOST   ENV           URL
# :8080      :9001  BACKEND_URL   http://localhost:9001
# :6379      :9002  REDIS_URL     http://localhost:9002
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Port nicht exponiert in Docker | Error "Port 8080 not exposed. Add to forwardPorts in devcontainer.json" |
| Container nicht gefunden | Error "Could not detect container. Pass --host-port manually" |
| Kein Docker (Cloud-Modus) | Error "Docker not available. Use --host-port or configure cloud forwarding" |
