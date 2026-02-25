---
id: TASK-3
title: KIND cluster setup
status: Done
assignee: []
created_date: '2026-02-25 13:59'
updated_date: '2026-02-25 14:36'
labels: []
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create a local Kubernetes development environment using KIND (Kubernetes IN Docker) so DRAFT-3 has a real cluster to develop and test against. This task is infrastructure only — no Go code changes.

Deliverables: a KIND cluster configuration, Kubernetes manifests to deploy both `strixlog` and `randomlog` as pods, and helper scripts to stand up and tear down the cluster.

Both apps are deployed using their existing Docker images, built locally and loaded into KIND via `kind load docker-image` (no registry needed). The `randomlog` deployment writes structured JSON logs to stdout — these are the logs that DRAFT-3's `KubernetesSource` will stream. The `strixlog` deployment includes a ServiceAccount with a Role granting read access to pods and pod logs in the same namespace.

The `strixlog` deployment manifest sets `STRIXLOG_ENV=kubernetes` so the auto-detection logic from DRAFT-3 is bypassed during K8s testing — the source is explicit.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 KIND cluster config at k8s/kind-config.yaml creates a single-node cluster named strixlog-dev
- [x] #2 randomlog manifest at k8s/manifests/randomlog-deployment.yaml deploys 1 replica with imagePullPolicy: Never and a liveness probe on /health
- [x] #3 strixlog manifest at k8s/manifests/strixlog-deployment.yaml deploys 1 replica with imagePullPolicy: Never, serviceAccountName: strixlog, and env STRIXLOG_ENV=kubernetes
- [x] #4 RBAC manifest at k8s/manifests/strixlog-rbac.yaml defines ServiceAccount strixlog, a Role granting get/list/watch on pods and get on pods/log, and a RoleBinding in the default namespace
- [x] #5 Setup script k8s/setup.sh builds images, creates KIND cluster, loads images, applies manifests, waits for pods ready
- [x] #6 Teardown script k8s/teardown.sh deletes the KIND cluster
- [ ] #7 After running k8s/setup.sh both pods are Running and kubectl logs deploy/randomlog shows structured JSON log lines
- [x] #8 README updated with KIND prerequisites (kind, kubectl, docker) and usage of setup/teardown scripts
<!-- AC:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
## Assessment: single task — infrastructure only, no Go code

## File structure

```
k8s/
  kind-config.yaml
  manifests/
    randomlog-deployment.yaml
    strixlog-deployment.yaml
    strixlog-rbac.yaml
  setup.sh
  teardown.sh
README.md  (updated)
```

## Implementation steps

### 1. KIND cluster config (`k8s/kind-config.yaml`)
Single control-plane node. Explicit `kindest/node` image pinned to match client-go version in DRAFT-3:
```yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: strixlog-dev
nodes:
  - role: control-plane
```

### 2. RBAC (`k8s/manifests/strixlog-rbac.yaml`)
ServiceAccount + Role (namespace-scoped, not ClusterRole) + RoleBinding:
- Role rules: `apiGroups: [""]`, `resources: ["pods", "pods/log"]`, `verbs: ["get", "list", "watch"]`
- All in `default` namespace

### 3. randomlog manifest (`k8s/manifests/randomlog-deployment.yaml`)
- 1 replica, `image: randomlog:latest`, `imagePullPolicy: Never`
- Port 8081, liveness probe on `GET /health`
- No service needed (no external access required)

### 4. strixlog manifest (`k8s/manifests/strixlog-deployment.yaml`)
- 1 replica, `image: strixlog:latest`, `imagePullPolicy: Never`
- Port 8080, liveness probe on `GET /health`
- `serviceAccountName: strixlog`
- `env: [{name: STRIXLOG_ENV, value: kubernetes}]`

### 5. Setup script (`k8s/setup.sh`)
```bash
#!/usr/bin/env bash
set -euo pipefail
command -v kind >/dev/null || { echo "kind not found"; exit 1; }
command -v kubectl >/dev/null || { echo "kubectl not found"; exit 1; }
docker compose build
kind get clusters | grep -q strixlog-dev || kind create cluster --config k8s/kind-config.yaml
kind load docker-image strixlog:latest randomlog:latest --name strixlog-dev
kubectl apply -f k8s/manifests/
kubectl wait --for=condition=Ready pod -l app=randomlog --timeout=60s
kubectl wait --for=condition=Ready pod -l app=strixlog --timeout=60s
echo "Ready. View logs: kubectl logs -f deploy/randomlog"
```

### 6. Teardown script (`k8s/teardown.sh`)
```bash
#!/usr/bin/env bash
kind delete cluster --name strixlog-dev
```

### 7. README update
Add "Kubernetes (KIND)" section: prerequisites, `k8s/setup.sh` / `k8s/teardown.sh`, how to verify, `kubectl logs -f deploy/randomlog`.
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
## Risks
- `imagePullPolicy: Never` is critical — without it K8s will attempt a Docker Hub pull and fail on locally-loaded images
- `setup.sh` checks `kind get clusters` before creating to be idempotent
- RBAC is namespace-scoped (Role, not ClusterRole) — least privilege; strixlog only reads pods in the default namespace
- The `kindest/node` image version should be pinned to match the `client-go` version used in DRAFT-3 to avoid client/server skew
- `STRIXLOG_ENV=kubernetes` is set explicitly in the strixlog deployment so DRAFT-3 auto-detection is bypassed during testing and forced into K8s mode
<!-- SECTION:NOTES:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Added KIND cluster setup with all infrastructure files. k8s/kind-config.yaml creates a single-node cluster named strixlog-dev. Manifests deploy randomlog (1 replica, imagePullPolicy: Never, liveness probe) and strixlog (1 replica, imagePullPolicy: Never, serviceAccountName: strixlog, STRIXLOG_ENV=kubernetes, HOSTNAME from pod metadata). RBAC manifest defines ServiceAccount, namespace-scoped Role (get/list/watch pods, get pods/log), and RoleBinding. setup.sh is idempotent (checks for existing cluster), builds images, loads via kind load docker-image, applies manifests, and waits for pods ready. teardown.sh deletes the cluster. README updated with KIND prerequisites and usage. AC-7 (manual verification) requires a running Docker + KIND installation.
<!-- SECTION:FINAL_SUMMARY:END -->
