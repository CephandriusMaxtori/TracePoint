---
layout: default
title: Features
---

# Features

TracePoint is organized into a sidebar of pages:

| Page | What it shows |
| --- | --- |
| Overview | CPU / memory gauges, uptime, system info, top processes, disks, package manager status |
| Network | Per-interface traffic rates and totals, ping, port scan, DNS lookup |
| Internet | Connectivity checks against well-known hosts with latency |
| Services | Start / stop / restart system services (Windows services, systemd, launchd, SysV) |
| Logs | Tail a log file with filtering, follow, and clear |
| Docker | Container list, status, show/hide, and per-container log tails |
| Printers | Installed printer status |
| Apps | Package manager overview and app search / install |

## Design notes

- **Dark theme** with a custom palette in `internal/ui/theme.go`.
- **Operations panel** (top-right) surfaces long-running actions with progress and logs.
- **Live updates**: collectors sample on short intervals and the window invalidates to repaint.
- **Cross-platform collectors** under `internal/collectors` with build-tagged backends (e.g. `services/backend_windows.go`).
