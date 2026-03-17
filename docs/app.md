# fac app

Steuert den Lifecycle der Flutter-App innerhalb einer Session.

## Subcommands

### fac app start

Startet die Flutter-App auf dem zugewiesenen Simulator. Synct automatisch vorher die Projektdateien.

```bash
fac app start [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--target` | string | `lib/main.dart` | Entry Point der App |
| `--flavor` | string | - | Build-Flavor (z.B. `dev`, `staging`, `prod`) |
| `--dart-define` | string[] | - | Dart Compile-Time Variablen (z.B. `API_URL=http://...`) |
| `--session` | string | aktive Session | Session-ID |
**Was passiert:**
1. Server startet `flutter run --machine -d <udid> --target <target>` im Working Directory (Volume Mount)
3. Server parst JSON-Events von stdout:
   - `app.start` → State wechselt zu `building`
   - `app.debugPort` → VM Service URI wird extrahiert, WebSocket-Verbindung wird aufgebaut
   - `app.started` → State wechselt zu `running`
4. Command kehrt zurück mit App-Info

**API-Mapping:** `POST /sessions/{id}/app/start`

**Request Body:**
```json
{
  "target": "lib/main.dart",
  "flavor": "dev",
  "dart_defines": ["API_URL=http://localhost:3000"]
}
```

**Response:**
```json
{
  "app_id": "abc-123",
  "state": "running",
  "vm_service_uri": "ws://127.0.0.1:52981/...",
  "started_at": "2026-03-16T21:35:00Z"
}
```

**CLI Output:**
```
Building app... (this may take a moment)
App started on iPhone 16 Pro (iOS 18.4)
```

### fac app stop

Stoppt die laufende Flutter-App. Der Simulator bleibt gebootet, die Session existiert weiter.

```bash
fac app stop [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--session` | string | aktive Session | Session-ID |

**Was passiert:**
1. Server sendet `app.stop` via stdin an den `flutter run` Prozess
2. Wartet auf graceful shutdown (Timeout: 10s)
3. Falls Timeout: SIGKILL
4. VM Service WebSocket wird geschlossen
5. Session-State wechselt zu `stopped`

**API-Mapping:** `POST /sessions/{id}/app/stop`

## Kontext

### flutter run --machine Protokoll

Der Server nutzt `flutter run --machine` das JSON-RPC über stdin/stdout spricht:

**stdout Events (Server → parst diese):**
```json
[{"event":"app.start","params":{"appId":"abc-123","deviceId":"D59AF284-...","supportsRestart":true,"launchMode":"run"}}]
[{"event":"app.debugPort","params":{"appId":"abc-123","port":52981,"wsUri":"ws://127.0.0.1:52981/..."}}]
[{"event":"app.started","params":{"appId":"abc-123"}}]
[{"event":"app.log","params":{"appId":"abc-123","log":"flutter: Hello World"}}]
[{"event":"app.stop","params":{"appId":"abc-123"}}]
```

**stdin Commands (Server → sendet diese):**
```json
[{"id":1,"method":"app.stop","params":{"appId":"abc-123"}}]
```

### Build-Zeit

Der erste `app start` einer Session dauert am längsten (Cold Build: 30-60s für iOS). Nachfolgende Starts nach `app stop` sind schneller, da Caches im Working Directory erhalten bleiben.

### flutter pub get

Wird automatisch vom Server ausgeführt wenn `pubspec.yaml` im Sync enthalten war. Das passiert vor `flutter run`.

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Kein Projekt im Working Directory | Error "No pubspec.yaml found. Is the volume mounted?" |
| pubspec.yaml fehlt | Error "No pubspec.yaml found in synced files" |
| Build-Fehler (Dart Compile Error) | Error mit Compiler-Output |
| flutter run crashed | Error mit letzten Logs, Session-State → `stopped` |
| App läuft bereits | Error "App already running. Use 'fac reload' or 'fac app stop' first" |
| Simulator nicht gebootet | Server bootet Simulator automatisch neu |
