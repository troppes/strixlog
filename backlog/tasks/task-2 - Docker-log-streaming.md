---
id: TASK-2
title: Docker log streaming
status: Draft
assignee: []
created_date: '2026-02-25 13:59'
updated_date: '2026-02-25 14:23'
labels: []
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add the core log-reading capability to `strixlog` for Docker environments. This task introduces three things that all subsequent tasks build on:

1. **`LogSource` interface** — the abstraction all future runtimes implement (`internal/source/source.go`)
2. **`LogEntry` model** — the normalised internal representation of a log line (`internal/model/logentry.go`)
3. **`DockerSource`** — concrete implementation that reads container logs via the Docker Engine API over `/var/run/docker.sock`

The Docker socket is accessed using raw HTTP over a Unix socket (Go stdlib `net` + `net/http`). No third-party Docker SDK — only two endpoints are needed: `GET /containers/json` to list containers and `GET /containers/{id}/logs?follow=true` to stream logs. Container discovery uses the Docker `/events` endpoint (filtered to start/stop events) so new containers are picked up without polling.

`main.go` is updated to instantiate `DockerSource` directly — hardcoded, no auto-detection yet (that is DRAFT-3). `docker-compose.yml` is updated to mount the Docker socket into the `strixlog` container.

Integration tests start the real `randomlog` container and verify strixlog receives and prints its structured JSON log lines end-to-end.

**No Kubernetes code, no auto-detection — that is DRAFT-3's scope.**
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 LogSource interface defined in internal/source/source.go with Start(ctx context.Context) error, Stop() error, and Logs() <-chan model.LogEntry
- [ ] #2 LogEntry struct defined in internal/model/logentry.go with fields: Timestamp, Source (container name), Line (raw log text)
- [ ] #3 DockerSource in internal/source/docker/ connects to the Docker Engine API via /var/run/docker.sock using raw HTTP (no Docker SDK)
- [ ] #4 strixlog streams logs from all running containers on the same Docker host, excluding itself (matched by HOSTNAME env var against container ID)
- [ ] #5 New containers started after strixlog is already running are discovered via the Docker /events endpoint and streamed automatically
- [ ] #6 When a container stops its log stream goroutine is cleaned up without crashing strixlog
- [ ] #7 Collected log lines are printed to strixlog stdout in normalised format: [<timestamp>] [<container-name>] <raw line>
- [ ] #8 Existing /health endpoint continues to work unchanged
- [ ] #9 docker-compose.yml updated with Docker socket volume mount (/var/run/docker.sock:/var/run/docker.sock:ro) and HOSTNAME env var on the strixlog service
- [ ] #10 Integration test uses the real randomlog container via docker-compose and verifies strixlog captures at least one structured JSON log line from it
- [ ] #11 Unit tests cover: Docker API response parsing, 8-byte mux header parsing, container self-exclusion logic, graceful cleanup on container stop
- [ ] #12 No third-party dependencies added — Docker Engine API accessed via stdlib net/http over Unix socket
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
## Package structure

```
strixlog/
  cmd/server/main.go                       -- updated: creates DockerSource, starts streaming + health server, signal handling
  internal/
    model/
      logentry.go                          -- LogEntry struct + String() formatter
      logentry_test.go
    source/
      source.go                            -- LogSource interface
      docker/
        client.go                          -- raw HTTP over unix socket: listContainers, streamLogs, watchEvents
        client_test.go                     -- httptest mock of Docker API endpoints
        mux.go                             -- 8-byte Docker stream frame header parser (stdout/stderr demux)
        mux_test.go                        -- table-driven tests with known byte sequences
        filter.go                          -- isSelf(containerID, hostname) bool + container name extraction
        filter_test.go
        source.go                          -- DockerSource implementing LogSource: supervisor + per-container streamers
        source_test.go
        integration_test.go               -- //go:build integration; uses real randomlog container
    printer/
      printer.go                           -- PrintLogs(ctx, <-chan LogEntry) writes to stdout
      printer_test.go
docker-compose.yml                         -- add socket mount + HOSTNAME to strixlog service
```

## Implementation steps

### Phase 1 — Model and interface (2 files)
1. `internal/model/logentry.go` — `LogEntry{Timestamp, Source, Line}` + `String()` returning `[<ts>] [<source>] <line>`
2. `internal/source/source.go` — `LogSource` interface: `Start(ctx) error`, `Stop() error`, `Logs() <-chan model.LogEntry`

### Phase 2 — Docker source (5 files)
3. `internal/source/docker/mux.go` — parse 8-byte Docker frame header (byte 0 = stream type, bytes 4–7 = big-endian uint32 payload size); return `(streamType, payload, error)`
4. `internal/source/docker/client.go` — thin HTTP client: `net.Dial("unix", socketPath)` transport; functions: `listContainers() ([]Container, error)`, `streamLogs(ctx, id) (io.ReadCloser, error)`, `watchEvents(ctx) (<-chan ContainerEvent, error)`. Pin Docker API version to `v1.45`.
5. `internal/source/docker/filter.go` — `isSelf(containerID, hostname string) bool` + `containerName(c Container) string`
6. `internal/source/docker/source.go` — `DockerSource` struct; `Start`: lists running containers, excludes self, spawns per-container streamer goroutine, subscribes to `/events` for new containers; `Stop`: cancels root context; tracks active streams in `map[string]context.CancelFunc`
7. `internal/printer/printer.go` — `PrintLogs(ctx, ch)` reads from channel, writes each `entry.String()` to stdout

### Phase 3 — Wire into main.go (1 file)
8. `cmd/server/main.go` — `signal.NotifyContext` for SIGTERM/SIGINT; create `DockerSource` (hardcoded); call `Start`; pass `Logs()` to `PrintLogs` in goroutine; wait for signal; call `Stop`

### Phase 4 — Docker Compose update (1 file)
9. `docker-compose.yml` — add to `strixlog` service:
   ```yaml
   volumes:
     - /var/run/docker.sock:/var/run/docker.sock:ro
   environment:
     - HOSTNAME=strixlog
   ```

### Phase 5 — Tests (4 files)
10. Unit tests: `mux_test.go` (table-driven with raw byte fixtures), `client_test.go` (httptest mock of Docker endpoints), `filter_test.go`, `logentry_test.go`
11. `integration_test.go` (`//go:build integration`):
    - `docker compose up -d randomlog` (with socket mount already in docker-compose.yml)
    - Create `DockerSource`, call `Start`
    - Within 10s assert at least one `LogEntry` arrives with `Source` containing "randomlog"
    - Assert `Line` is valid JSON matching `{"timestamp","level","message","source"}` schema
    - Teardown: `docker compose down`
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Architectural decisions

- **Docker API:** Raw HTTP over unix socket. Only 2 log endpoints + `/events`. No Docker SDK.
- **Frame parsing:** Docker multiplexed stream uses 8-byte header per frame. Must detect TTY mode (`Config.Tty` in container inspect) — TTY containers send raw text with no header.
- **Discovery:** `/events?filters={"event":["start","die"]}` for event-driven container discovery. No polling.
- **Self-exclusion:** `HOSTNAME` env var in the container equals the container ID prefix. Set `HOSTNAME=strixlog` explicitly in docker-compose.yml for reliable matching.
- **Goroutine lifecycle:** Each container stream goroutine holds a `context.WithCancel` derived from the supervisor context. Tracked in `map[string]context.CancelFunc`. On `Stop()` or container die event, cancel + delete from map.
- **Output:** Raw log line printed as-is. Parsing of the JSON structure from randomlog is NOT done in this task — that is a future enrichment task.
- **Integration test:** Must use real `randomlog` container via Docker, not a mock. Build-tagged `//go:build integration` so `go test ./...` does not fail without Docker.

## Open questions
1. Output format: print raw line or re-serialise as enriched JSON? Current plan: raw line for simplicity.
2. Capture stdout only or also stderr? Current plan: both (Docker API returns both by default), identical prefix.
3. Docker API minimum version: pinned to `v1.45`.
<!-- SECTION:NOTES:END -->
