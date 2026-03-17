# FAC Architecture Overview

## Was ist FAC?

FAC (Flutter Agent Connect) ist eine Bridge zwischen AI-Agenten in DevContainern und Flutter-Simulatoren/Emulatoren auf einem Mac. Ein einziges Go-Binary, zwei Modi:

- **`fac serve`** — HTTP-Server auf dem Mac, verwaltet Simulatoren und Flutter-Prozesse
- **`fac connect`** + Client-Commands — im DevContainer, steuert den Server per REST API

## Architektur-Diagramm (Phase 1 — Lokaler Mac)

```
┌──────────────────────────────────────────────────────────────────┐
│  Mac                                                             │
│                                                                  │
│  ┌────────────────────────┐       ┌────────────────────────────┐ │
│  │  DevContainer          │       │  FAC Server (fac serve)    │ │
│  │                        │       │                            │ │
│  │  AI Agent              │ HTTP  │  Session Manager           │ │
│  │  fac CLI ─────────────────────►│  ├─ iOS Simulator          │ │
│  │                        │       │  ├─ flutter run --machine  │ │
│  │  /workspace ◄──────────────┐   │  ├─ Dart VM Service (WS)  │ │
│  │  (Volume Mount)        │   │   │  └─ Work Dir ─────────┐   │ │
│  └────────────────────────┘   │   └───────────────────────┼───┘ │
│                               │                           │     │
│                               └───────────────────────────┘     │
│                            Gleiche Dateien (Volume Mount)        │
│                                                                  │
│  Xcode, Flutter SDK, Simulators                                  │
└──────────────────────────────────────────────────────────────────┘
```

Alles läuft auf einem Mac. Der DevContainer und der FAC-Server teilen sich die Projektdateien über einen Docker Volume Mount. Kein Netzwerk-Sync, keine Authentifizierung — alles lokal.

## Kern-Konzepte

### Session

Eine Session ist die zentrale Einheit. Sie kapselt:
- Einen zugewiesenen Simulator/Emulator (mit UDID)
- Ein Working Directory auf dem Mac (= das gemountete Volume)
- Einen `flutter run --machine` Prozess
- Eine WebSocket-Verbindung zum Dart VM Service
- Den aktuellen State (created → building → running → stopped → destroyed)

Mehrere Sessions können parallel laufen (begrenzt durch Mac-Hardware: ~4-6 auf 16GB RAM).

### flutter run --machine

Das ist der Integrationspunkt zu Flutter. Dieser Prozess:
- Gibt strukturierte JSON-Events auf stdout (App-Start, VM Service URI, Errors)
- Akzeptiert JSON-RPC Commands auf stdin (Hot Reload, Hot Restart)
- Wird pro Session genau einmal gestartet

### Dart VM Service Protocol

Nach App-Start exponiert die Flutter-App einen WebSocket-Endpoint (die VM Service URI). FAC verbindet sich dorthin für:
- Hot Reload (`reloadSources`)
- Screenshots (`_flutter.screenshot`)
- Widget Tree (`ext.flutter.debugDumpApp`)
- Render Tree (`ext.flutter.debugDumpRenderTree`)
- Semantics Tree (`ext.flutter.debugDumpSemanticsTreeInTraversalOrder`)
- Debug-Flags (`ext.flutter.debugPaint`, `ext.flutter.repaintRainbow`, etc.)

### File Sharing via Volume Mount

Container und Mac teilen die Projektdateien über einen Docker Volume Mount. Änderungen im Container sind sofort auf dem Mac sichtbar — kein Sync-Schritt nötig. `fac reload` triggert einfach nur den Hot Reload, die Dateien sind schon da.

```bash
docker run -v /Users/torben/myapp:/workspace ...
```

### Widget-basiertes Tapping

Statt nur Pixel-Koordinaten zu unterstützen, kann FAC auch nach Semantics-Label oder Widget-Key tappen:
1. FAC holt den Semantics Tree via VM Service
2. Sucht das Element nach Label/Key
3. Berechnet die Mitte des Bounding Rects des Elements
4. Führt den Tap an diesen Koordinaten aus

## Technologie-Stack

| Komponente | Technologie |
|------------|-------------|
| Sprache | Go |
| CLI Framework | cobra |
| HTTP Server | net/http (stdlib) |
| WebSocket | gorilla/websocket |
| JSON | encoding/json (stdlib) |
| Logging | zerolog |
| IDs | google/uuid |
| File Sharing | Docker Volume Mount |

## Verzeichnisse auf dem Mac

```
~/.fac/
├── config.json          # Server-Config (Port)
└── sessions/
    └── <session-id>/    # Metadata pro Session
        └── state.json   # Session-State, Device-Info, Flutter-Process-Info
```

Die Projektdateien selbst liegen im gemounteten Volume (z.B. `/Users/torben/myapp`), nicht in `~/.fac/`.

## Verzeichnisse im Container

```
~/.fac/
└── config.json          # Client-Config (Server URL, aktive Session ID)
```

## Spätere Erweiterungen (nicht Phase 1)

- **rsync/SSH** für Cloud-Deployments (Container und Mac auf verschiedenen Maschinen)
- **SSH-Tunnel** für sichere Remote-Verbindung
- **Authentifizierung** (Bearer Token oder SSH-basiert) für öffentlich erreichbare Server
