# Strixlog

A simple log aggregator and analyzer with a dashboard to review logs. A companion log generator app (`randomlog`) produces structured JSON logs to stdout, which `strixlog` consumes via the Docker log driver.

## Prerequisites

- [Go 1.26+](https://golang.org/dl/)
- [Docker](https://docs.docker.com/get-docker/) with Compose plugin

## Getting Started

### Run with Docker Compose

```bash
docker compose up --build
```

- `strixlog` available at <http://localhost:8080>
- `randomlog` available at <http://localhost:8081>

### Run locally

```bash
# strixlog
cd strixlog
go run ./cmd/server

# randomlog (separate terminal)
cd randomlog
go run ./cmd/server
```

### Run tests

```bash
cd strixlog && go test ./...
cd randomlog && go test ./...
```

## Folder Structure

```
.
├── strixlog/       # Main log aggregator and dashboard application (port 8080)
├── randomlog/      # Random log generator with REST API (port 8081)
├── docker-compose.yml
└── .devcontainer/  # VS Code dev container configuration
```

## API Reference

Both apps expose a health check endpoint:

```
GET /health
200 OK
{"status":"ok"}
```

## Log Format

`randomlog` emits structured JSON to stdout:

```json
{"timestamp":"2026-02-25T12:00:00Z","level":"INFO","message":"user login successful","source":"randomlog"}
```

## Development

Open this repository in VS Code and select **Reopen in Container** to use the pre-configured dev container with Go 1.26, Delve debugger, and Docker-in-Docker support.
