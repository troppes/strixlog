#!/usr/bin/env bash
set -euo pipefail

echo "==> Deleting KIND cluster (strixlog-dev)..."
kind delete cluster --name strixlog-dev
echo "Done."
