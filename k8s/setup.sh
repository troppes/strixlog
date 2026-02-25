#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

command -v kind >/dev/null 2>&1 || { echo "Error: kind not found. Install from https://kind.sigs.k8s.io/"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "Error: kubectl not found. Install from https://kubernetes.io/docs/tasks/tools/"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Error: docker not found."; exit 1; }

echo "==> Building Docker images..."
docker compose -f "${REPO_ROOT}/docker-compose.yml" build

echo "==> Creating KIND cluster (strixlog-dev)..."
if kind get clusters 2>/dev/null | grep -q "^strixlog-dev$"; then
  echo "    Cluster already exists, skipping creation."
else
  kind create cluster --config "${SCRIPT_DIR}/kind-config.yaml"
fi

echo "==> Loading images into KIND..."
kind load docker-image strixlog:latest randomlog:latest --name strixlog-dev

echo "==> Applying manifests..."
kubectl apply -f "${SCRIPT_DIR}/manifests/"

echo "==> Waiting for pods to be ready..."
kubectl wait --for=condition=Ready pod -l app=randomlog --timeout=60s
kubectl wait --for=condition=Ready pod -l app=strixlog --timeout=60s

echo ""
echo "Cluster is ready."
echo "  View randomlog output : kubectl logs -f deploy/randomlog"
echo "  View strixlog output  : kubectl logs -f deploy/strixlog"
echo "  Access strixlog health: kubectl port-forward deploy/strixlog 8080:8080"
echo "  Tear down             : ${SCRIPT_DIR}/teardown.sh"
