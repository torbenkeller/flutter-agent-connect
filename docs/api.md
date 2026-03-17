# FAC REST API Reference

Alle Endpoints unter `http://<host>:<port>/`.
Phase 1: Keine Authentifizierung (Server läuft nur auf localhost).
Session-Endpoints erfordern Header `X-Agent-ID`.

## Infra

### GET /health
```json
{"status": "ok", "version": "0.1.0"}
```

## Agents

### POST /agents
```json
// Request
{"id": "my-agent"}
// Response 201
{"id": "my-agent", "created_at": "2026-03-17T10:00:00Z"}
```

## Sessions

Header: `X-Agent-ID: <agent-id>` (bei allen Session-Endpoints)

### POST /sessions
```json
// Request
{"platform": "ios", "device_type": "iPhone 16 Pro", "name": "ios-main", "work_dir": "/path/to/app"}
// Response 201
{"id": "abc-123", "agent_id": "my-agent", "name": "ios-main", "platform": "ios", "state": "created", "device": {"udid": "...", "name": "fac-my-agent-ios-main", "state": "Booted"}}
```

### GET /sessions
Listet Sessions des Agents. Response: `{"sessions": [...]}`

### GET /sessions/{id}
Session-Details.

### DELETE /sessions/{id}
Session zerstören (stoppt App, löscht Simulator).

## Flutter

### POST /sessions/{id}/flutter/run
```json
// Request
{"target": "lib/main.dart"}
// Response 200
{"app_id": "abc-123", "state": "running", "vm_service_uri": "ws://127.0.0.1:52981/..."}
```

### POST /sessions/{id}/flutter/stop
Response: `{"message": "App stopped"}`

### POST /sessions/{id}/flutter/hot-reload
Response: `{"success": true}`

### POST /sessions/{id}/flutter/hot-restart
Response: `{"success": true}`

### POST /sessions/{id}/flutter/clean
Response: `{"success": true}`

### POST /sessions/{id}/flutter/pub-get
Response: `{"success": true}`

### GET /flutter/version
```json
{"flutter": "3.32.0", "dart": "3.8.0", "channel": "stable"}
```

## Device

### GET /sessions/{id}/device/screenshot
Response: `Content-Type: image/png` mit PNG-Bytes.

### POST /sessions/{id}/device/tap
```json
// Widget-basiert
{"label": "Login", "index": 0}
{"key": "submitButton"}
// Pixel
{"x": 195, "y": 400}
// Response
{"success": true, "tapped_at": {"x": 195, "y": 400}}
```

### POST /sessions/{id}/device/swipe
```json
{"direction": "down", "duration_ms": 300}
```

### POST /sessions/{id}/device/type
```json
{"text": "user@example.com", "clear": true, "enter": false}
```

## DevTools

### GET /sessions/{id}/devtools/widgets
Widget Tree als JSON.

### GET /sessions/{id}/devtools/render
Render Tree als JSON.

### GET /sessions/{id}/devtools/semantics
Semantics Tree als JSON.

### POST /sessions/{id}/devtools/performance
Toggle Performance Overlay. Response: `{"flag": "performance", "enabled": true}`

### POST /sessions/{id}/devtools/paint
Toggle Debug Paint. Response: `{"flag": "paint", "enabled": true}`

### POST /sessions/{id}/devtools/repaint
Toggle Repaint Rainbow. Response: `{"flag": "repaint", "enabled": true}`

### GET /sessions/{id}/devtools/logs
```json
// Query: ?errors=false&lines=50
{"logs": [{"timestamp": "...", "level": "info", "message": "flutter: Hello"}]}
```

## Error Responses

```json
{"error": "not_found", "message": "Session not found"}
{"error": "conflict", "message": "App already running"}
{"error": "validation_error", "message": "Missing required field: platform"}
{"error": "internal_error", "message": "flutter run crashed"}
```
