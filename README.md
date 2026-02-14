# Powerwall TV for Linux

GTK-based Linux desktop app for Powerwall TV.

Ported from [Powerwall-TV for tvOS](https://github.com/sighmon/Powerwall-TV) by [OpenClaw](https://github.com/openclaw/openclaw) using `gpt-5.3-codex`.

## Features

- Home dashboard with dynamic background images
- Local Gateway and Fleet API modes
- Dedicated Graphs view:
  - Top chart cycles through Solar / Battery / Home / Grid (Up/Down)
  - Bottom chart shows battery percentage
  - `Esc` or **Back** returns to Home

## Requirements

- Go (with CGO enabled)
- GTK3 development libraries (gotk3)
- Linux desktop session (X11/Wayland)

## Run

```bash
make dev
```

or:

```bash
go run ./cmd/powerwall-tv
```

## Build

```bash
make build
```

Binary output:

- `bin/powerwall-tv`

## Vendoring and gotk3 patch

This repo tracks `vendor/` in git.

To refresh vendored dependencies and apply the gotk3 compatibility patch:

```bash
make vendor
```

This runs:

1. `go mod vendor`
2. `./scripts/patch-gotk3.sh`

## Flatpak

```bash
make flatpak
```

Quick iterative build:

```bash
make flatpak-fast
```

Run installed flatpak:

```bash
make flatpak-run
```

## Notes

- Uses GTK3 via gotk3
- Secrets/tokens stored via go-keyring
