---
layout: default
title: Home
---

# TracePoint

TracePoint is a lightweight, cross-platform system administration dashboard built in Go with the [Gio](https://gioui.org) immediate-mode GUI toolkit. It monitors system health, network activity, and installed services through a dark-themed desktop interface.

## Pages

- [Features](features.md) — what the dashboard can do
- [Architecture](architecture.md) — how collectors, state, and actions fit together
- [Building](build.md) — how to build and run TracePoint
- [Deploying the docs](deploy.md) — how to publish these docs to GitHub Pages

## Quick start

```sh
go build ./...
go run .
```

Requires Go 1.22+ and a GPU-capable desktop. See [Building](build.md) for platform notes.
