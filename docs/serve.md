# fac serve

Startet den FAC-Server auf dem Mac. Der Server verwaltet Simulatoren, Flutter-Prozesse und exponiert die REST API.

## Usage

```bash
fac serve [flags]
```

## Flags

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--port` | int | `8420` | HTTP-Server Port |
| `--host` | string | `127.0.0.1` | Bind-Adresse (localhost only) |
| `--flutter-sdk` | string | auto-detect | Pfad zum Flutter SDK. Default: sucht `flutter` im PATH |

## Was der Server tut

1. **Startup:**
   - Prüft ob `flutter` und `xcrun simctl` verfügbar sind
   - Listet verfügbare Simulatoren/Emulatoren
   - Startet HTTP-Server auf dem konfigurierten Port
   - Schreibt Config nach `~/.fac/config.json`

2. **Laufzeit:**
   - Akzeptiert REST API Requests
   - Verwaltet Sessions (Simulator-Lifecycle, Flutter-Prozesse)
   - Pro Session: eigener Simulator, eigener `flutter run` Prozess

3. **Shutdown (SIGINT/SIGTERM):**
   - Stoppt alle laufenden Flutter-Prozesse graceful
   - Fährt alle Session-Simulatoren herunter

## REST API Endpoints

Siehe [api.md](api.md) für die vollständige API-Dokumentation.

## Server-seitige Abhängigkeiten

- **macOS** (für iOS Simulatoren)
- **Xcode** installiert (für `xcrun simctl`)
- **Flutter SDK** installiert und im PATH

## Interner Ablauf

```
fac serve --port 8420
  │
  ├─ Verify Prerequisites (flutter, xcrun simctl)
  ├─ Initialize DevicePool (list available simulators)
  ├─ Initialize SessionManager
  ├─ Setup HTTP Routes
  ├─ Write Config (~/.fac/config.json)
  └─ Start HTTP Server on 127.0.0.1:8420
       │
       ├─ POST /sessions → SessionManager.Create()
       ├─ POST /sessions/{id}/app/start → SessionManager.StartApp()
       │    └─ flutter run --machine -d <udid>
       │         └─ Parse stdout → extract VM Service URI
       │              └─ Connect WebSocket to VM Service
       ├─ POST /sessions/{id}/app/reload → SessionManager.HotReload()
       │    └─ VM Service reloadSources
       ├─ GET /sessions/{id}/screenshot → SessionManager.Screenshot()
       │    └─ VM Service _flutter.screenshot OR xcrun simctl io screenshot
       └─ ... (weitere Endpoints)
```

## Beispiel

```bash
$ fac serve
FAC Server starting on http://127.0.0.1:8420
Found 12 available iOS simulators
Ready.

# In einem DevContainer:
$ fac connect
Connected to FAC Server v0.1.0
```

## Error Cases

| Fehler | Verhalten |
|--------|-----------|
| Port bereits belegt | Exit mit Fehlermeldung |
| Flutter SDK nicht gefunden | Exit mit Installationshinweis |
| Xcode nicht installiert | Exit mit Hinweis auf `xcode-select --install` |
| Kein Simulator verfügbar | Server startet trotzdem, warnt aber |
