# fac sync

**Phase 1: Nicht implementiert.** File-Sync passiert über Docker Volume Mounts — Dateien sind automatisch auf beiden Seiten sichtbar.

## Phase 1: Volume Mount

Container und Mac teilen die Projektdateien über einen Docker Volume Mount:

```bash
docker run -v /Users/torben/myapp:/workspace ...
```

Änderungen im Container sind sofort auf dem Mac sichtbar. Kein Sync-Schritt nötig. `fac reload` triggert nur den Hot Reload — die Dateien sind schon da.

## Später: rsync für Cloud-Deployments

Wenn Container und Mac auf verschiedenen Maschinen laufen (z.B. Container in der Cloud, Mac bei MacStadium), kann kein Volume Mount verwendet werden. Dann wird `fac sync` implementiert:

```bash
fac sync [path]    # rsync über SSH zum Mac
```

Siehe Plan für Details zur rsync-Integration.
