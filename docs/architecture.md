---
layout: default
title: Architecture
---

# Architecture

TracePoint separates data collection, state, and presentation.

## Packages

```
main.go                                App entry point; wires everything together
internal/state                          Shared, mutex-protected snapshot + history
internal/collectors/                    Background samplers
  system/                               CPU, memory, disk, processes, OS info
  network/                              Interface traffic + ping/scan/lookup tools
  internet/                             Connectivity checks
  services/                             Service control (per-OS backends)
  docker/                               Docker engine + container API
  printers/                             Printer status
  apps/                                 Package manager info + search
  logs/                                 File tailing
internal/actions/                       Long-running action manager (ops panel)
internal/ui/                            Gio widgets, theme, and pages
```

## Data flow

1. Collectors run in goroutines started in `main.go`.
2. Each collector writes snapshots through `state.Store` (`Update`/`Read` guards a shared struct).
3. The network collector also pushes a history series (`PushHist`) for the overview sparklines.
4. The UI reads state on every frame and renders with Gio. Buttons that trigger work (checks, installs, service control) dispatch through `actions.Manager`, which records progress and logs in the operations panel.

## State model

`internal/state/state.go` defines `Store`, the `System`, `Net`, `Internet`, `Docker`, `Services`, `Printers`, `Packages` snapshots, and the history buffer. The UI never talks to OS APIs directly — it only reads state and invokes actions.
