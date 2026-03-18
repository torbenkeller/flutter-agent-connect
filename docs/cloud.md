# FAC Cloud-Architektur (Zukunft)

Dieses Dokument beschreibt die geplante Architektur wenn DevContainer und Mac auf verschiedenen Maschinen laufen.

## Ausgangslage

Phase 1 (lokal) funktioniert weil Container und Mac auf derselben Maschine laufen:
- Volume Mounts für File-Sharing
- Docker Port-Mapping für Backend-Zugriff
- `host.docker.internal` für Kommunikation

Wenn Container in der Cloud laufen (AWS, GCP, etc.) und der Mac lokal steht oder bei MacStadium/Hetzner, fallen diese Mechanismen weg.

## Was sich ändert

| Feature | Phase 1 (lokal) | Cloud |
|---------|-----------------|-------|
| File-Sharing | Docker Volume Mount | rsync über SSH |
| HTTP API | `host.docker.internal:8420` | SSH-Tunnel → `localhost:8420` |
| Backend Port-Forwarding | Docker Port-Mapping | Reverse Proxy auf dem Mac |
| Authentifizierung | Keine (localhost only) | SSH-Key-basiert |
| Simulator-Zugriff | Direkt | Über FAC API |

## Cloud-Architektur

```
┌────────────────────────┐          SSH          ┌───────────────────────────┐
│  Cloud (Container)     │◄─────────────────────►│  Mac (MacStadium/lokal)   │
│                        │                        │                           │
│  AI Agent              │  SSH Tunnel :8420      │  FAC Server               │
│  fac CLI ──────────────────────────────────────►│  ├─ HTTP API              │
│                        │                        │  ├─ Session Manager       │
│  Backend :8080 ◄───────────────────────────┐    │  ├─ Simulatoren           │
│                        │  Reverse Proxy     │    │  │                       │
│  Projektdateien ───────────rsync──────────►│    │  ├─ Reverse Proxy ───────┘
│                        │                        │  │  :9001 → Container:8080│
│                        │                        │  │  :9002 → Container:8080│
└────────────────────────┘                        └───────────────────────────┘
```

## Erforderliche Änderungen

### 1. File-Sync: rsync über SSH

```bash
fac connect --ssh user@mac-host --ssh-key ~/.ssh/fac_key
fac sync    # rsync -avz --delete ./ user@mac:~/.fac/sessions/<id>/
```

`fac flutter run`, `fac flutter hot-reload` etc. rufen intern `fac sync` auf.

### 2. HTTP API: SSH-Tunnel

`fac connect` baut einen SSH-Tunnel auf:
```bash
ssh -L 8420:localhost:8420 user@mac-host
```

Alle HTTP-Requests gehen durch den Tunnel. Kein Bearer Token nötig — SSH authentifiziert.

### 3. Backend Port-Forwarding: Reverse Proxy

Im Cloud-Modus kann Docker Port-Mapping nicht genutzt werden. Stattdessen:

1. `fac forward 8080 -e BACKEND_URL` startet:
   - SSH Reverse-Tunnel: Mac:9001 → Container:8080
   - Oder: Go Reverse Proxy auf dem Mac der über SSH-Tunnel zum Container forwarded
2. Simulator erreicht das Backend über `localhost:9001`
3. Automatische Port-Zuweisung und Dart-Define-Injection wie in Phase 1

### 4. Authentifizierung: SSH-Keys

Ein SSH-Key pro Agent. Wird beim `fac connect` konfiguriert:
```bash
fac connect --ssh user@mac.cloud --ssh-key ~/.ssh/fac_key
```

## Migration lokal → Cloud

Der Wechsel soll minimal sein:

```bash
# Phase 1 (lokal)
fac connect
fac session create --platform ios --name my-app --work-dir /app

# Cloud — nur connect ändert sich
fac connect --ssh user@mac.cloud --ssh-key ~/.ssh/key
fac session create --platform ios --name my-app --work-dir /app
# Ab hier: gleiche Commands, FAC handhabt Sync/Tunnel/Proxy im Hintergrund
```

## Anbieter-Optionen

| Anbieter | Preis/Monat | Besonderheit |
|----------|-------------|-------------|
| MacStadium | ~110 USD | Orka-Virtualisierung, dediziert |
| Macly | ~100 USD | Tagesabrechnung möglich |
| Hetzner | ~75 EUR | Günstig, Verfügbarkeit schwankend |
| AWS EC2 Mac | ~470 USD | AWS-Integration, 24h Minimum |

## Offene Fragen

- Soll FAC den SSH-Tunnel selbst managen, oder wird Tailscale/WireGuard empfohlen?
- Brauchen wir Session-Persistenz (Simulator bleibt nach Container-Neustart)?
- Soll der Reverse Proxy auch WebSocket unterstützen (für VM Service DevTools)?
- Container-Orchestrierung: Kubernetes? Docker Swarm? Oder manuell?
