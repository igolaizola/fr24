# 🛩️ fr24 (Go)

A Go client and CLI for retrieving data from Flightradar24 using JSON endpoints and gRPC‑web.
Ported from the original Python implementation (https://github.com/abc8747/fr24) with assistance from OpenAI's Codex.
This is a small, dependency‑light Go library and command‑line tool.

- JSON endpoints: flight list, airport list, playback, search (find)
- gRPC‑web endpoints: live feed, playback feed, nearest flights, live flight status, top flights, flight details, playback flight, follow flight (streaming)
- Outputs JSON to stdout; library includes helpers to flatten responses and write CSV

## Install

- Prebuilt binaries (recommended):

  - Download from GitHub Releases: https://github.com/igolaizola/fr24/releases
  - Assets follow: `fr24_<version>_<os>_<arch>.zip` (e.g., `fr24_v0.1.0_linux_amd64.zip`, `fr24_v0.1.0_macos_arm64.zip`).

- Go install (latest tag or specific):

  - CLI: `go install github.com/igolaizola/fr24/cmd/fr24@latest`
  - Library: `go get github.com/igolaizola/fr24`

- Build from source:
  - `go build ./cmd/fr24`
  - Or use the Makefile (see below) for cross‑compiles and versioned builds

Proto messages are vendored under `pkg/proto`; no protoc is required to use the library or CLI.

## Quick Start (CLI)

Basic invocation:

- `fr24 login` — authenticate using env/config (see Auth below)
- `fr24 dirs` — print cache base directory
- `fr24 flightlist -reg B-HPJ` — list flights by registration
- `fr24 flightlist -flight CX255` — list by flight number
- `fr24 airportlist -code HKG -mode arrivals` — arrivals/departures/ground
- `fr24 find -q A359` — search (airports/aircraft/operators/routes)
- `fr24 livefeed -south 42 -north 52 -west -8 -east 10` — live feed in bbox
- `fr24 playbackfeed -south 42 -north 52 -west -8 -east 10 -duration 7` — historical live feed window
- `fr24 nearest -lat 22.3 -lon 114.2` — nearest flights to a point
- `fr24 livestatus -id 12345` — live status for one flight id
- `fr24 topflights -limit 10` — most viewed flights
- `fr24 flightdetails -id 12345` — detailed info for a live flight
- `fr24 playbackflight -id 12345 -ts 1726480000` — details for a historic flight
- `fr24 followflight -id 12345` — stream updates for a flight (JSON frames)
- `fr24 followflight -id 12345` — stream updates for a flight (JSON frames)
  - Options: `-timeout 10` to stop after N seconds; `-once` to exit after the first frame

All commands write JSON to stdout. Errors go to stderr.

Show usage:

- `fr24` (no args) prints available commands

CLI is built with `ff/v3`.

- Top‑level help: `fr24 -h`
- Per‑command help: `fr24 <subcommand> -h`

## Makefile

Convenience targets for local builds and artifacts:

- `make build` — build current platform binary into `bin/`
- `make app-build` — cross‑compile for multiple platforms
  - Example: `PLATFORMS="linux/amd64 darwin/arm64 windows/amd64" make app-build`
- `make clean` — remove built artifacts
- `make zip` — build and package with README
- `make docker-build` — build a single‑arch Docker image
- `make docker-buildx` — build multi‑arch image via buildx

Builds include version metadata via `-ldflags` using the current tag/commit/date.

## Releases

This project uses GoReleaser; tagging the repo will produce release artifacts for common OS/arch combinations. See `.goreleaser.yml` for the full matrix and naming template.

## Authentication

The client supports three modes:

- Username/password login (preferred for gRPC)
- Subscription key (JSON endpoints, optional bearer token for gRPC)
- Anonymous device id (limited access; some endpoints may not work)

Environment variables:

- `fr24_username`, `fr24_password`
- `fr24_subscription_key`, `fr24_token` (token used as Bearer for gRPC‑web)

Optional config file (used if present):

- Path: `$XDG_CONFIG_HOME/fr24/fr24.conf` (e.g., `~/.config/fr24/fr24.conf`)
- INI format with a `[global]` section:

```
[global]
username = your-email@example.com
password = your-password
subscription_key = your-subscription-key
token = your-access-token
```

The CLI command `fr24 login` will read env/config and validate by performing a login if username/password are present.
If neither credentials nor keys are configured, it will run in anonymous mode and print `login anonymous`.

## Smoke Test

Run a best‑effort smoke test that exercises all commands with live data.

- Build: `make build`
- Execute: `scripts/smoke.sh bin/$(basename $(ls -1 bin/fr24-* | head -n1))`

What it does:

- Validates help for all subcommands
- Runs representative JSON calls: `find`, `airportlist`, `flightlist`
- Runs gRPC‑web calls: `livefeed`, `playbackfeed`, `nearest`, `topflights`, `livestatus`
- Derives a valid `playbackflight` request from `playbackfeed` output (uses returned `flightid` and `timestamp`)
- Uses `nearest` to pick a flight id for `livestatus`, `playbackflight`, and a short `followflight` stream (`-timeout`+`-once`)

## Library Usage

Minimal example:

```go
package main

import (
    "context"
    "encoding/json"
    "io"
    "os"

    fr "github.com/igolaizola/fr24/pkg/flightradar"
)

func main() {
    c := fr.New()
    _ = c.LoginFromEnvOrConfig() // optional; anonymous works for some endpoints

    // Live feed in a bounding box
    p := fr.LiveFeedParams{BoundingBox: fr.BoundingBox{South: 42, North: 52, West: -8, East: 10}}
    resp, err := c.GrpcLiveFeed(context.Background(), p)
    if err != nil { panic(err) }
    defer resp.Body.Close()

    raw, _ := io.ReadAll(resp.Body)
    msg, _ := fr.ParseLiveFeedGRPC(raw)
    out := make([]fr.LiveFeedFlightRecord, 0, len(msg.GetFlightsList()))
    for _, f := range msg.GetFlightsList() {
        out = append(out, fr.LiveFeedFlightToRecord(f))
    }
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    enc.Encode(out)
}
```

Higher‑level helpers are available under `pkg/flightradar/service.go`, e.g.:

- `NewServices(client).LiveFeed().Fetch(ctx, params).Records()` → `[]LiveFeedFlightRecord`
- `NewServices(client).FlightList().Fetch(ctx, params).Records()` → `[]FlightListRecord`

CSV helper:

- `flightradar.WriteCSV(io.Writer, []YourRecord)` writes slices to CSV using struct tags.

## Notes

- Output schemas for flattened records are defined in `pkg/flightradar/*.go` (e.g., `flatten.go`, `records.go`).
- Binaries rely on standard `net/http`; requests mimic the browser headers expected by Flightradar24.
- Protobuf types are under `pkg/proto`; gRPC‑web framing/parsing lives in `pkg/flightradar/grpcweb.go`.

## Disclaimer

- This project is for educational purposes only. Do not abuse it.
- Respect Flightradar24 terms of service and local laws when accessing data.
- Official Flightradar24 API: https://fr24api.flightradar24.com/

Flightradar24 mentions this on every API response:

> The contents and derived data are the property of Flightradar24 AB for use exclusively by its products and applications. Using, modifying or redistributing the data without prior written permission of Flightradar24 AB may be prohibited.
