# FAC REST API Reference

Alle Endpoints sind unter `http://<host>:<port>/` erreichbar.
Phase 1: Keine Authentifizierung (Server läuft nur auf localhost).

## Infra

### GET /health
Server-Status. Kein Auth erforderlich.

**Response 200:**
```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "active_sessions": 2,
  "available_simulators": 10
}
```

### GET /devices
Listet alle verfügbaren Simulatoren/Emulatoren.

**Response 200:**
```json
{
  "devices": [
    {
      "udid": "D59AF284-...",
      "name": "iPhone 16 Pro",
      "platform": "ios",
      "runtime": "com.apple.CoreSimulator.SimRuntime.iOS-18-4",
      "state": "Shutdown",
      "available": true
    }
  ]
}
```

## Agents

### POST /agents
Registriert einen neuen Agent.

**Request:**
```json
{
  "id": "my-agent"
}
```

**Response 201:**
```json
{
  "id": "my-agent",
  "created_at": "2026-03-17T10:00:00Z"
}
```

## Sessions

Alle Session-Endpoints erfordern den Header `X-Agent-ID: <agent-id>`.
Sessions sind immer an einen Agent gebunden.

### POST /sessions
Erstellt eine neue Session für den Agent.

**Request:**
```json
{
  "platform": "ios",
  "device_type": "iPhone 16 Pro",
  "runtime_version": "18.4",
  "name": "ios-main"
}
```

**Response 201:**
```json
{
  "id": "abc-123",
  "agent_id": "my-agent",
  "name": "ios-main",
  "platform": "ios",
  "state": "created",
  "device": {
    "udid": "D59AF284-...",
    "name": "iPhone 16 Pro",
    "runtime": "com.apple.CoreSimulator.SimRuntime.iOS-18-4",
    "state": "Booted"
  },
  "created_at": "2026-03-17T10:30:00Z"
}
```

### GET /sessions
Listet Sessions des Agents (gefiltert nach `X-Agent-ID`).

**Response 200:**
```json
{
  "sessions": [
    {
      "id": "abc-123",
      "name": "ios-main",
      "platform": "ios",
      "state": "running",
      "device": {"name": "iPhone 16 Pro"}
    },
    {
      "id": "def-456",
      "name": "android",
      "platform": "android",
      "state": "running",
      "device": {"name": "Pixel 8"}
    }
  ]
}
```

### GET /sessions/{id}
Detaillierter Session-Status. Nur eigene Sessions sichtbar.

**Response 200:** Vollständiges Session-Objekt (siehe [status.md](status.md))

### DELETE /sessions/{id}
Zerstört eine Session. Nur eigene Sessions löschbar.

**Response 200:**
```json
{"message": "Session destroyed"}
```

## App Lifecycle

File-Sync passiert über Volume Mounts — nicht über die HTTP API. Die API steuert nur den App-Lifecycle.

### POST /sessions/{id}/app/start
App starten. Dateien müssen bereits im Working Directory liegen (Volume Mount).

**Request:**
```json
{
  "target": "lib/main.dart",
  "flavor": "dev",
  "dart_defines": ["API_URL=http://localhost:3000"]
}
```

**Response 200:**
```json
{
  "app_id": "abc-123",
  "state": "running",
  "vm_service_uri": "ws://127.0.0.1:52981/...",
  "started_at": "2026-03-16T21:35:00Z"
}
```

### POST /sessions/{id}/app/reload
Hot Reload. Dateien sind durch Volume Mount bereits aktuell.

**Response 200:**
```json
{
  "success": true,
  "reload_duration_ms": 287
}
```

### POST /sessions/{id}/app/restart
Hot Restart.

**Response 200:**
```json
{
  "success": true,
  "restart_duration_ms": 1850
}
```

### POST /sessions/{id}/app/stop
App stoppen.

**Response 200:**
```json
{"message": "App stopped"}
```

## Screenshots & UI

### GET /sessions/{id}/screenshot
Screenshot als PNG.

**Query Parameter:**
| Parameter | Default | Beschreibung |
|-----------|---------|--------------|
| `device` | `false` | `true` für Device-Level Screenshot |

**Response 200:**
```
Content-Type: image/png
Body: <PNG bytes>
```

### POST /sessions/{id}/tap
Tap-Event. Widget-basiert oder Pixel.

**Request (Widget-basiert):**
```json
{"label": "Login", "index": 0}
```
oder:
```json
{"key": "submitButton"}
```

**Request (Pixel):**
```json
{"x": 195, "y": 400}
```

**Response 200:**
```json
{
  "success": true,
  "tapped_at": {"x": 195, "y": 400},
  "element": {
    "label": "Login",
    "rect": {"left": 150, "top": 380, "right": 240, "bottom": 420}
  }
}
```

### POST /sessions/{id}/swipe
Swipe-Geste.

**Request:**
```json
{
  "from_x": 200,
  "from_y": 600,
  "to_x": 200,
  "to_y": 200,
  "duration_ms": 300
}
```

**Response 200:**
```json
{
  "success": true,
  "from": {"x": 200, "y": 600},
  "to": {"x": 200, "y": 200}
}
```

### POST /sessions/{id}/type
Text eingeben.

**Request:**
```json
{
  "text": "user@example.com",
  "clear": true,
  "enter": false
}
```

**Response 200:**
```json
{
  "success": true,
  "text_entered": "user@example.com"
}
```

## Inspection

### GET /sessions/{id}/inspect/widgets
Widget tree als JSON. Siehe [inspect.md](inspect.md).

### GET /sessions/{id}/inspect/render
Render tree als JSON. Siehe [inspect.md](inspect.md).

### GET /sessions/{id}/inspect/semantics
Semantics tree als JSON. Siehe [inspect.md](inspect.md).

**Query Parameter (alle inspect-Endpoints):**
| Parameter | Default | Beschreibung |
|-----------|---------|--------------|
| `depth` | `-1` | Max Tree-Tiefe (-1 = unbegrenzt) |
| `compact` | `false` | Kompakte Ausgabe |

## Debugging

### POST /sessions/{id}/debug/paint
Toggle Debug Paint.

**Response 200:**
```json
{"flag": "debugPaint", "enabled": true}
```

### POST /sessions/{id}/debug/repaint
Toggle Repaint Rainbow.

**Response 200:**
```json
{"flag": "repaintRainbow", "enabled": true}
```

### POST /sessions/{id}/debug/performance
Toggle Performance Overlay.

**Response 200:**
```json
{"flag": "performanceOverlay", "enabled": true}
```

## Logs

### GET /sessions/{id}/logs
App-Logs abrufen.

**Query Parameter:**
| Parameter | Default | Beschreibung |
|-----------|---------|--------------|
| `errors` | `false` | Nur Errors |
| `lines` | `50` | Anzahl Zeilen |

**Response 200:**
```json
{
  "logs": [
    {"timestamp": "2026-03-16T21:35:01Z", "level": "info", "message": "flutter: Hello World"}
  ],
  "total_lines": 42
}
```

## Error Responses

Alle Fehler folgen dem gleichen Format:

**404 Not Found:**
```json
{"error": "not_found", "message": "Session not found"}
```

**409 Conflict:**
```json
{"error": "conflict", "message": "App already running"}
```

**422 Unprocessable Entity:**
```json
{"error": "validation_error", "message": "Missing required field: platform"}
```

**500 Internal Server Error:**
```json
{"error": "internal_error", "message": "flutter run crashed", "details": "..."}
```
