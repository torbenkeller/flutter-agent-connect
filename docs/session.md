# fac session

Verwaltet Sessions. Jede Session = ein Simulator/Emulator + ein Flutter-Prozess. Sessions gehören immer zu einem Agent (gesetzt bei `fac connect`).

## Subcommands

### fac session create

Erstellt eine neue Session für den aktuellen Agent.

```bash
fac session create [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--platform` | string | `ios` | Plattform: `ios`, `android` (Phase 3), `web` (Phase 5), `macos` (Phase 5) |
| `--device` | string | auto | Device-Typ, z.B. `"iPhone 16 Pro"`. Default: nimmt ein passendes verfügbares Device |
| `--runtime` | string | latest | Runtime-Version, z.B. `"18.4"`. Default: neueste verfügbare |
| `--name` | string | auto | Optionaler Name für die Session (z.B. `"ios-main"`, `"android-test"`) |

**Was passiert:**
1. Server allokiert einen Simulator aus dem Device Pool
2. Simulator wird gebootet (falls nicht schon gestartet)
3. Session wird dem aktuellen Agent zugeordnet
4. Neue Session wird automatisch zur aktiven Session

**API-Mapping:** `POST /sessions`

**Request Body:**
```json
{
  "agent_id": "my-agent",
  "platform": "ios",
  "device_type": "iPhone 16 Pro",
  "runtime_version": "18.4",
  "name": "ios-main"
}
```

**Response:**
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

### fac session list

Listet alle Sessions des aktuellen Agents.

```bash
fac session list
```

**API-Mapping:** `GET /sessions?agent_id=my-agent`

**Output:**
```
  ID        Name          Platform  Device          State     Created
* abc-123   ios-main      ios       iPhone 16 Pro   running   2 min ago
  def-456   android-test  android   Pixel 8         running   30 sec ago

(* = active session)
```

### fac session use

Wechselt die aktive Session. Alle nachfolgenden Commands ohne `--session` Flag beziehen sich auf diese Session.

```bash
fac session use <session-id-or-name>
```

Man kann entweder die Session-ID oder den Session-Namen verwenden:

```bash
fac session use abc-123        # per ID
fac session use ios-main       # per Name
fac session use android-test   # wechselt zu Android
```

**Was passiert:**
1. Prüft ob die Session existiert und dem Agent gehört
2. Setzt `active_session_id` in `~/.fac/config.json`
3. Gibt Bestätigung mit Session-Info aus

**CLI Output:**
```bash
$ fac session use android-test
Active session: android-test (def-456)
Platform: android, Device: Pixel 8, State: running
```

### fac session destroy

Zerstört eine Session.

```bash
fac session destroy [session-id-or-name]
```

Ohne Argument wird die aktive Session zerstört.

**API-Mapping:** `DELETE /sessions/{id}`

**Was passiert:**
1. `flutter run` Prozess wird gestoppt
2. VM Service WebSocket wird geschlossen
3. Simulator wird heruntergefahren
4. Device wird im Pool freigegeben
5. Falls es die aktive Session war: `active_session_id` wird geleert

## Agent-Konzept

Ein Agent ist ein Namespace für Sessions — typischerweise ein DevContainer bzw. ein AI-Agent.

- Agent-ID wird bei `fac connect --agent <name>` gesetzt
- Alle Sessions gehören zu genau einem Agent
- `fac session list` zeigt nur Sessions des eigenen Agents
- Mehrere Agents können gleichzeitig auf dem gleichen Mac arbeiten
- Agents wissen nichts voneinander

```
Mac (FAC Server)
├── Agent "feature-login" (DevContainer 1)
│   ├── Session "ios-main" (iPhone 16 Pro) ← active
│   └── Session "android" (Pixel 8)
│
└── Agent "bugfix-42" (DevContainer 2)
    └── Session "ios" (iPhone 15)
```

## Session State Machine

```
created → building → running ⇄ reloading
                       ↓
                    stopped → destroyed
```

| State | Beschreibung |
|-------|-------------|
| `created` | Session existiert, Simulator gebootet, App noch nicht gestartet |
| `building` | `flutter run` kompiliert die App |
| `running` | App läuft, VM Service verbunden |
| `reloading` | Hot Reload läuft gerade |
| `stopped` | App gestoppt, Session existiert noch |
| `destroyed` | Alles aufgeräumt |

## Multi-Session Workflow

```bash
# iOS und Android parallel
fac session create --platform ios --name ios
fac session create --platform android --name android

# iOS testen
fac session use ios
fac flutter run
fac device screenshot -o ios-home.png

# Zu Android wechseln
fac session use android
fac flutter run
fac device screenshot -o android-home.png

# Oder explizit ohne Wechsel
fac device screenshot --session ios -o ios.png
fac device screenshot --session android -o android.png
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Kein Simulator verfügbar | Error mit Hinweis welche existieren |
| Simulator-Boot fehlgeschlagen | Error mit xcrun simctl Output |
| Session-Name existiert schon (für diesen Agent) | Error "Session name already in use" |
| `session use` mit fremder Session-ID | Error "Session not found" (sieht nur eigene) |
| Session existiert nicht (destroy) | Error "Session not found" |
