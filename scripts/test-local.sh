#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

./scripts/dev-setup.sh

echo "[test-local] Running linters..."
if command -v golangci-lint >/dev/null 2>&1; then
  golangci-lint run
else
  echo "[test-local] golangci-lint not installed; skipping."
fi

echo "[test-local] Building project..."
make build

echo "[test-local] Running unit tests..."
make test

echo "[test-local] Success"

