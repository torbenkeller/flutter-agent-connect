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

## CLI Command-Struktur

```
FAC-Infrastruktur:
  fac serve                      Server starten
  fac connect [--agent]          Mit Server verbinden
  fac session create/list/use/destroy   Sessions verwalten
  fac status                     Verbindungs- und Session-Status

Flutter-App (auf Simulator-Maschine):
  fac flutter run                App starten (flutter run --machine)
  fac flutter stop               App stoppen
  fac flutter hot-reload         Hot Reload (Code injizieren, State bleibt)
  fac flutter hot-restart        Hot Restart (App neu starten, State reset)
  fac flutter clean              build/ + .dart_tool/ löschen
  fac flutter pub-get            Dependencies installieren
  fac flutter version            Flutter-Version auf der Maschine
  fac flutter test               [Phase 2]
  fac flutter drive              [Phase 2]
  fac flutter widget-preview     [perspektivisch]

Device-Interaktion:
  fac device screenshot          Screenshot vom Simulator
  fac device tap                 Tap (Widget-basiert oder Pixel)
  fac device swipe               Swipe-Geste
  fac device type                Text eingeben

DevTools (Inspektion & Debugging):
  fac devtools widgets           Widget Tree
  fac devtools render            Render Tree (Layout, Constraints)
  fac devtools semantics         Semantics Tree (Labels, Rects, Actions)
  fac devtools performance       Performance Overlay togglen
  fac devtools paint             Debug Paint togglen
  fac devtools repaint           Repaint Rainbow togglen
  fac devtools logs              App-Logs
  fac devtools network           [Phase 2]
```

## Kern-Konzepte

### Agent

Ein Agent ist ein Namespace — typischerweise ein DevContainer bzw. AI-Agent. Agents sehen nur ihre eigenen Sessions und Simulatoren. Mehrere Agents können gleichzeitig auf dem gleichen Mac arbeiten.

### Session

Eine Session kapselt:
- Einen eigens erstellten Simulator (Name: `fac-<agent>-<session>`)
- Einen `flutter run --machine` Prozess
- Eine WebSocket-Verbindung zum Dart VM Service

Mehrere Sessions pro Agent möglich (z.B. iOS + Android parallel).

### flutter run --machine

Integrationspunkt zu Flutter. Dieser Prozess:
- Gibt strukturierte JSON-Events auf stdout (App-Start, VM Service URI, Errors)
- Akzeptiert JSON-RPC Commands auf stdin (Hot Reload, Hot Restart)

### Dart VM Service Protocol

Nach App-Start exponiert die Flutter-App einen WebSocket-Endpoint. FAC verbindet sich dorthin für:
- Hot Reload (`reloadSources`)
- Screenshots (`_flutter.screenshot`)
- Widget/Render/Semantics Tree
- Debug-Flags (Paint, Repaint, Performance)

### File Sharing via Volume Mount

Container und Mac teilen Projektdateien über Docker Volume Mount. Änderungen im Container sind sofort auf dem Mac sichtbar — kein Sync nötig.

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

## Spätere Erweiterungen (nicht Phase 1)

- **rsync/SSH** für Cloud-Deployments (Container und Mac auf verschiedenen Maschinen)
- **SSH-Tunnel** für sichere Remote-Verbindung
- **Authentifizierung** für öffentlich erreichbare Server
- **Android Emulator** Support
- **Web/macOS** Platform Support
