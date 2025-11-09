#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(git -C "$ROOT_DIR" rev-parse --show-toplevel 2>/dev/null || echo "$ROOT_DIR/..")"
HOOK_SRC="$REPO_ROOT/scripts/hooks/pre-commit"
HOOK_DEST="$REPO_ROOT/.git/hooks/pre-commit"

if [ ! -f "$HOOK_SRC" ]; then
  echo "[install-hooks] pre-commit hook source missing: $HOOK_SRC" >&2
  exit 1
fi

install -m 0755 "$HOOK_SRC" "$HOOK_DEST"

echo "[install-hooks] Installed pre-commit hook to $HOOK_DEST"

