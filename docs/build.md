---
layout: default
title: Building
---

# Building

## Requirements

- Go 1.22 or newer
- A desktop with GPU support (Gio requires OpenGL/Vulkan/Metal)

## Commands

```sh
go build ./...      # compile everything
go run .            # run the dashboard
```

## Platform notes

- **Windows**: uses `gopsutil` for system metrics and the Windows service API for service control.
- **Linux**: systemd / SysV service backends.
- **macOS**: launchd service backend.

Collectors that depend on the OS are isolated behind build-tagged files
(e.g. `internal/collectors/services/backend_windows.go`), so cross-compiling
is straightforward:

```sh
GOOS=linux GOARCH=amd64 go build -o tracepoint-linux .
```
