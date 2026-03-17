# fac flutter

Steuert die Flutter-App auf der Simulator-Maschine. Die Commands orientieren sich an der Flutter CLI.

## Commands

### fac flutter run

Startet die Flutter-App auf dem Simulator der aktiven Session.

```bash
fac flutter run [flags]
```

| Flag | Typ | Default | Beschreibung |
|------|-----|---------|--------------|
| `--target` | string | `lib/main.dart` | Entry Point der App |
| `--flavor` | string | - | Build-Flavor (z.B. `dev`, `staging`, `prod`) |
| `--dart-define` | string[] | - | Compile-Time Variablen |
| `--session` | string | aktive Session | Session ID oder Name |

**Was passiert:**
1. Server startet `flutter run --machine -d <udid> --target <target>` im Working Directory
2. Parst JSON-Events von stdout (app.start, app.debugPort, app.started)
3. Verbindet sich zum Dart VM Service via WebSocket
4. Gibt zurück wenn die App läuft

**API-Mapping:** `POST /sessions/{id}/flutter/run`

**CLI Output:**
```
Building app... (this may take a moment)
App running on fac-demo-ios-test (iPhone 16 Pro)
```

### fac flutter stop

Stoppt die laufende Flutter-App. Simulator bleibt gebootet, Session existiert weiter.

```bash
fac flutter stop [--session <id>]
```

**API-Mapping:** `POST /sessions/{id}/flutter/stop`

### fac flutter hot-reload

Hot Reload — injiziert geänderten Dart-Code. App-State bleibt erhalten.

```bash
fac flutter hot-reload [--session <id>]
```

**Was passiert:** Server sendet `app.restart` mit `fullRestart: false` via stdin an den `flutter run` Prozess.

**Wann nutzen:** Normale Code-Änderungen (Widgets, Styling, Logik).

**API-Mapping:** `POST /sessions/{id}/flutter/hot-reload`

**CLI Output:**
```
Hot reload successful (287ms)
```

### fac flutter hot-restart

Hot Restart — App startet komplett in der Dart VM neu. State geht verloren.

```bash
fac flutter hot-restart [--session <id>]
```

**Was passiert:** Server sendet `app.restart` mit `fullRestart: true` via stdin.

**Wann nutzen:** Enum-Änderungen, `main()` geändert, globale Initializer geändert.

**API-Mapping:** `POST /sessions/{id}/flutter/hot-restart`

**CLI Output:**
```
Hot restart successful (1.8s)
```

### fac flutter clean

Löscht Build-Artefakte und Caches auf der Simulator-Maschine.

```bash
fac flutter clean [--session <id>]
```

**Was passiert:** Server führt `flutter clean` im Working Directory aus. Löscht `build/` und `.dart_tool/`.

**Wann nutzen:** Build-Fehler die durch Caches verursacht werden, nach großen Dependency-Änderungen.

**API-Mapping:** `POST /sessions/{id}/flutter/clean`

### fac flutter pub-get

Installiert Dependencies auf der Simulator-Maschine.

```bash
fac flutter pub-get [--session <id>]
```

**Was passiert:** Server führt `flutter pub get` im Working Directory aus.

**Wann nutzen:** Nach Änderungen an `pubspec.yaml`.

**API-Mapping:** `POST /sessions/{id}/flutter/pub-get`

### fac flutter version

Zeigt die Flutter-Version auf der Simulator-Maschine.

```bash
fac flutter version
```

**API-Mapping:** `GET /flutter/version`

**CLI Output:**
```
Flutter 3.32.0 (channel stable)
Dart 3.8.0
```

## Übersicht: Wann welcher Command?

| Situation | Command |
|-----------|---------|
| App zum ersten Mal starten | `fac flutter run` |
| Kleine Code-Änderung (Widget, Styling) | `fac flutter hot-reload` |
| Enum/main()/Initializer geändert | `fac flutter hot-restart` |
| Native Code oder Dependencies geändert | `fac flutter stop` → `fac flutter run` |
| Build-Cache kaputt | `fac flutter clean` → `fac flutter run` |
| pubspec.yaml geändert | `fac flutter pub-get` → `fac flutter hot-restart` |

## Error Cases

| Fehler | Command | Verhalten |
|--------|---------|-----------|
| Kein Working Directory | `run` | Error "No work directory. Pass --work-dir when creating session" |
| App läuft bereits | `run` | Error "App already running. Use hot-reload or stop first" |
| App läuft nicht | `hot-reload/hot-restart/stop` | Error "No running app. Use 'fac flutter run' first" |
| Hot Reload fehlgeschlagen | `hot-reload` | Error + Hinweis auf `fac flutter hot-restart` |
| Build-Fehler | `run` | Error mit Compiler-Output |
