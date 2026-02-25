---
id: DRAFT-3
title: Kubernetes log streaming and runtime auto-detection
status: Draft
assignee: []
created_date: '2026-02-25 14:11'
updated_date: '2026-02-25 14:24'
labels: []
dependencies:
  - DRAFT-1
  - DRAFT-2
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Extend `strixlog` to stream logs from Kubernetes pods using `k8s.io/client-go`, and add runtime auto-detection so the app selects the correct backend at startup without code changes.

`k8s.io/client-go` is the approved dependency for K8s access — it handles API versioning, in-cluster auth, watch reconnection, and is the industry standard. The "minimal dependencies" principle still applies to other libraries, but K8s interaction specifically warrants `client-go`.

The `KubernetesSource` implements the same `LogSource` interface defined in DRAFT-1. It uses `client-go` to list and watch pods in the current namespace, streaming logs from each non-self pod via `clientset.CoreV1().Pods(ns).GetLogs(name, &PodLogOptions{Follow: true}).Stream(ctx)`. K8s log streams are plain text — no mux header parsing needed (unlike Docker).

The `DetectRuntime()` function replaces the hardcoded `DockerSource` instantiation in `main.go` from DRAFT-1:
1. `STRIXLOG_ENV=docker|kubernetes` — explicit override, used first
2. `/var/run/secrets/kubernetes.io/serviceaccount/token` exists → Kubernetes
3. `/var/run/docker.sock` exists and is a socket → Docker
4. Neither found → fail fast with a clear error message

Integration tests run against the KIND cluster from DRAFT-2, deploying both services and asserting strixlog captures `randomlog` pod output within 30 seconds.

**Depends on: DRAFT-1 (LogSource interface + LogEntry model) and DRAFT-2 (KIND cluster for testing).**
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 k8s.io/client-go added to strixlog/go.mod, version pinned to match KIND cluster K8s version from DRAFT-2
- [ ] #2 KubernetesSource in internal/source/kubernetes/ implements source.LogSource using client-go
- [ ] #3 KubernetesSource lists pods in the current namespace (read from /var/run/secrets/kubernetes.io/serviceaccount/namespace), watches for new/deleted pods, streams logs from each non-self pod
- [ ] #4 strixlog excludes its own pod by matching HOSTNAME env var against pod name
- [ ] #5 When a pod is deleted or enters a terminal phase (Succeeded/Failed) its log stream goroutine is cleaned up without crashing strixlog
- [ ] #6 DetectRuntime() in internal/runtime/detect.go implements three-step detection: STRIXLOG_ENV override -> K8s token file probe -> Docker socket probe -> fail fast
- [ ] #7 main.go updated to call DetectRuntime() instead of hardcoding DockerSource — no other wiring changes needed
- [ ] #8 Docker mode continues to work exactly as before (no regressions from DRAFT-1)
- [ ] #9 Unit tests: DetectRuntime() with all env/filesystem permutations; KubernetesSource with fake.NewSimpleClientset(); pod self-exclusion; goroutine cleanup on pod deletion
- [ ] #10 Integration test runs against KIND cluster from DRAFT-2: strixlog captures randomlog pod log lines within 30 seconds
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
## Package structure

```
strixlog/
  cmd/server/main.go                            -- updated: DetectRuntime() replaces hardcoded DockerSource
  internal/
    runtime/
      detect.go                                 -- DetectRuntime() -> (source.LogSource, error)
      detect_test.go                            -- table-driven: all 5 env/filesystem permutations
    source/
      source.go                                 -- unchanged (defined in DRAFT-1)
      kubernetes/
        source.go                               -- KubernetesSource implementing LogSource
        source_test.go                          -- fake.NewSimpleClientset() unit tests
        watcher.go                              -- pod watch loop: ADDED -> start stream, DELETED -> cancel
        watcher_test.go
        integration_test.go                    -- //go:build integration; uses KIND cluster
  go.mod                                        -- k8s.io/client-go added
```

## Implementation steps

### Phase 1 — Dependency (1 file)
1. `go.mod` — `go get k8s.io/client-go@v0.32.x` (pin to match KIND node image version). Run `go mod tidy`.

### Phase 2 — Runtime detection (2 files)
2. `internal/runtime/detect.go`:
```go
func DetectRuntime() (source.LogSource, error) {
    if env := os.Getenv("STRIXLOG_ENV"); env != "" {
        switch env {
        case "kubernetes": return kubernetes.New()
        case "docker":     return docker.New(defaultSocketPath)
        default:           return nil, fmt.Errorf("unknown STRIXLOG_ENV: %q", env)
        }
    }
    if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
        return kubernetes.New()
    }
    if fi, err := os.Stat("/var/run/docker.sock"); err == nil && fi.Mode()&os.ModeSocket != 0 {
        return docker.New(defaultSocketPath)
    }
    return nil, errors.New("no runtime detected: set STRIXLOG_ENV=docker or STRIXLOG_ENV=kubernetes")
}
```
3. `internal/runtime/detect_test.go` — 5 table-driven cases using temp files + env var manipulation

### Phase 3 — Kubernetes source (3 files)
4. `internal/source/kubernetes/source.go` — `KubernetesSource` struct:
   - `New()` calls `rest.InClusterConfig()` → creates `kubernetes.Clientset`
   - Reads namespace from `/var/run/secrets/kubernetes.io/serviceaccount/namespace`
   - `Start`: lists existing pods (excluding self via `HOSTNAME`), spawns log stream per pod, starts watcher
   - `Stop`: cancels root context
   - Log stream: `clientset.CoreV1().Pods(ns).GetLogs(name, &corev1.PodLogOptions{Follow: true}).Stream(ctx)` → `bufio.Scanner` → send `LogEntry` on channel

5. `internal/source/kubernetes/watcher.go` — watches pod events using `clientset.CoreV1().Pods(ns).Watch(ctx, metav1.ListOptions{})`:
   - `ADDED` event: spawn log stream goroutine with child context
   - `DELETED` / terminal phase: cancel goroutine context + remove from map
   - On watch disconnect: reconnect with last `resourceVersion`
   - Track active streams: `map[string]context.CancelFunc`

6. Tests:
   - `source_test.go` — `fake.NewSimpleClientset()` with pre-populated pod list; assert LogEntry arrives; assert self-pod excluded
   - `watcher_test.go` — fake clientset watch; inject ADDED/DELETED events; assert stream starts/stops
   - `integration_test.go` (`//go:build integration`) — use `KUBECONFIG` env var or `~/.kube/config`; create `KubernetesSource`; assert at least one `LogEntry` with Source matching "randomlog" pod name arrives within 30s

### Phase 4 — Wire main.go (1 file)
7. `cmd/server/main.go` — replace `docker.New(...)` with `runtime.DetectRuntime()`; log which runtime was selected; rest of wiring unchanged
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Architectural decisions

- **client-go approved:** Handles API versioning, in-cluster auth, watch reconnection. Worth the dependency for K8s. No `client-go` equivalent needed for Docker (raw HTTP is sufficient there).
- **In-cluster config:** `rest.InClusterConfig()` — one call handles service account token, CA cert, and API server host. No manual token file reading.
- **Pod discovery:** Raw `Watch` for initial implementation (simpler than Informers). Upgrade to `SharedInformerFactory` if watch reliability becomes a concern.
- **Log streaming:** `GetLogs(...).Stream(ctx)` returns `io.ReadCloser`; `bufio.Scanner` reads lines. No 8-byte mux header — K8s API returns plain text.
- **Namespace:** Read from `/var/run/secrets/kubernetes.io/serviceaccount/namespace`. Optional override via `STRIXLOG_NAMESPACE` env var.
- **Self-exclusion:** `HOSTNAME` env var in the pod = pod name (set by Kubernetes by default via `fieldRef: fieldPath: metadata.name`).
- **client-go version:** Must match KIND cluster K8s version from DRAFT-2. Pin to `v0.32.x` if KIND uses `kindest/node:v1.32.x`.
- **Unit tests:** `k8s.io/client-go/kubernetes/fake` provides `fake.NewSimpleClientset()` — standard pattern for testing without a real cluster.

## Open questions
1. Informer vs raw Watch for pod discovery? Start with raw Watch; migrate to Informer if reconnection proves unreliable.
2. Namespace scope: own namespace only (recommended) vs cluster-wide (requires ClusterRole)?
3. Label selector to filter pods? Proposal: stream all pods in namespace for now; add `strixlog.io/collect=true` label filter as a future enhancement.
<!-- SECTION:NOTES:END -->
