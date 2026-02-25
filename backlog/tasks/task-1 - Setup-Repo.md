---
id: TASK-1
title: Setup Repo
status: Done
assignee: []
created_date: '2026-02-25 12:59'
updated_date: '2026-02-25 13:33'
labels: []
milestone: m-0
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Bootstrap the repository with two Go 1.26 applications in a monorepo layout:

- **`strixlog/`** — Main log aggregator and dashboard application. Expose a single `GET /health` endpoint returning `{"status":"ok"}` (default port **8080**). Will access `randomlog` container logs via Docker stdout (no direct HTTP communication between services).
- **`randomlog/`** — Log generator application with a REST API. Expose a single `GET /health` endpoint returning `{"status":"ok"}` (default port **8081**). Writes logs to stdout.

Each application must have:
- Its own `go.mod` with module path `github.com/troppes/strixlog/<appname>` (separate Go modules, no shared `go.work` — already in `.gitignore`)
- A `main.go` entrypoint under `cmd/server/`
- A multi-stage `Dockerfile` (golang:1.26-alpine builder → alpine/distroless runtime)
- An integration test that starts the HTTP server and asserts `/health` returns 200 with `{"status":"ok"}`

Use only Go stdlib (`net/http`) — no HTTP frameworks. Ports are configurable via `PORT` env var with sensible defaults.

Log entry format (stdout): structured JSON `{"timestamp":"…","level":"…","message":"…","source":"…"}`

Repository-level files:
- `docker-compose.yml` — builds and runs both apps on a shared bridge network (`strixnet`), with healthcheck directives and `randomlog` depending on `strixlog` being healthy
- `.devcontainer/devcontainer.json` — Go devcontainer with Docker-in-Docker, delve; forward ports 8080 and 8081 (primary development workflow)
- `.github/workflows/ci.yml` — triggers on push and PRs; steps: `go vet ./...`, `go test -race -coverprofile`, `go build ./cmd/server`
- Updated `README.md` — prerequisites, getting started, project structure, API reference
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Random Log generator app is created
- [x] #2 Strixlog app is created
- [x] #3 Docker-Compose file is added
- [x] #4 Both apps can be run
- [x] #5 Readme is updated
- [x] #6 Github Actions is created
- [x] #7 DevContainer configuration is added
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
## Implementation Phases

### Phase 1 — strixlog app
1. `go mod init github.com/troppes/strixlog/strixlog` inside `strixlog/`
2. `internal/server/server.go` — `NewServer(port string) *Server` + `Start()`
3. `internal/server/health.go` — `handleHealth` returning `{"status":"ok"}`
4. `cmd/server/main.go` — reads `PORT` env var (default 8080), calls `NewServer().Start()`
5. `internal/server/server_test.go` — `httptest.NewServer`, GET /health, assert 200 + body
6. `Dockerfile` — multi-stage (golang:1.26-alpine builder → alpine/distroless runtime)

### Phase 2 — randomlog app
7–12. Mirror Phase 1 with module path `github.com/troppes/strixlog/randomlog` and default port 8081; `main.go` writes structured JSON log entries to stdout

### Phase 3 — Docker Compose
13. `docker-compose.yml` — two services on `strixnet` bridge network, ports 8080/8081, healthchecks, `randomlog` depends_on `strixlog` healthy

### Phase 4 — DevContainer
14. `.devcontainer/devcontainer.json` — `mcr.microsoft.com/devcontainers/go:1.26`, Docker-in-Docker feature, delve, `golang.go` + docker VS Code extensions, post-create `go mod download`, forward ports 8080/8081

### Phase 5 — GitHub Actions CI
15. `.github/workflows/ci.yml` — trigger on push + PRs; per-app matrix with `working-directory`; steps: `go vet ./...` → `go test -race -coverprofile=coverage.out ./...` → `go build ./cmd/server`

### Phase 6 — Docs & Cleanup
16. `README.md` — prerequisites (Go 1.26, Docker), getting started, project structure, API reference
17. `.gitignore` — verify binary output paths are covered
18. Update backlog task to Done before final commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Resolved Decisions

| # | Question | Decision |
|---|----------|----------|
| Go version | — | **1.26** |
| Module paths | — | `github.com/troppes/strixlog/strixlog` and `github.com/troppes/strixlog/randomlog` |
| Port numbers | — | `strixlog` → 8080, `randomlog` → 8081 |
| Inter-service log delivery | — | `strixlog` reads `randomlog` container **stdout** (no HTTP between services) |
| Log entry schema | — | `{"timestamp":"…","level":"…","message":"…","source":"…"}` |
| CI integration test (compose up) | — | **No** — not for this task |
| Linting | — | **`go vet`** only |
| DevContainer as primary workflow | — | **Yes** — include Docker-in-Docker, delve |
| `/health` vs `/ready` | — | `/health` only |
| Graceful shutdown | — | Bare `ListenAndServe` sufficient for bootstrap |
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Bootstrapped the monorepo with two Go 1.26 applications.\n\n- `strixlog/` and `randomlog/` each have separate `go.mod`, `cmd/server/main.go`, `internal/server` package with health handler, integration test, and multi-stage Dockerfile.\n- `docker-compose.yml` runs both on the `strixnet` bridge network with healthchecks and dependency ordering.\n- `.devcontainer/devcontainer.json` uses Go 1.26 image with Docker-in-Docker, Delve, and VS Code extensions.\n- `.github/workflows/ci.yml` runs vet, test, and build for each app via a matrix strategy.\n- `README.md` updated with prerequisites, run instructions, folder structure, API reference, and log format.\n- All tests pass locally.
<!-- SECTION:FINAL_SUMMARY:END -->
