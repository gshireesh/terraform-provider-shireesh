#!/usr/bin/env bash
# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

set -euo pipefail

# Bootstrap local development environment for terraform-provider-shireesh.
# Installs buf, golangci-lint (matching CI version), and runs initial generation.

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "[dev-setup] Ensuring Go modules are downloaded..."
go mod download

if ! command -v buf >/dev/null 2>&1; then
  echo "[dev-setup] Installing buf..."
  if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    brew install bufbuild/buf/buf
  else
    VERSION="v1.45.0"
    ARCH="$(uname -m)"
    case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac
    OS="$(uname -s)"
    if [[ "$OS" == "Linux" ]]; then OSLOW=linux; else OSLOW=darwin; fi
    TMP="$(mktemp -d)"
    curl -sSL "https://github.com/bufbuild/buf/releases/download/${VERSION}/buf-${OSLOW}-${ARCH}" -o "$TMP/buf"
    chmod +x "$TMP/buf"
    sudo mv "$TMP/buf" /usr/local/bin/buf
    rm -rf "$TMP"
  fi
else
  echo "[dev-setup] buf already installed"
fi

GOLANGCI_WANT="2.6.1"
GOLANGCI_CUR="$(golangci-lint version 2>/dev/null | sed -n 's/.*version \([^ ]*\).*/\1/p' || true)"
if [[ "$GOLANGCI_CUR" != "$GOLANGCI_WANT" ]]; then
  echo "[dev-setup] Installing golangci-lint $GOLANGCI_WANT (current='$GOLANGCI_CUR')..."
  ARCH="$(uname -m)"
  case "$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac
  OS="$(uname -s)"
  if [[ "$OS" == "Linux" ]]; then OSLOW=linux; else OSLOW=darwin; fi
  TMP="$(mktemp -d)"
  URL="https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_WANT}/golangci-lint-${GOLANGCI_WANT}-${OSLOW}-${ARCH}.tar.gz"
  curl -sSL "$URL" -o "$TMP/gcl.tar.gz"
  tar -xzf "$TMP/gcl.tar.gz" -C "$TMP"
  BIN="$(find "$TMP" -type f -name golangci-lint -perm +111 | head -n1)"
  install -m 0755 "$BIN" "$(go env GOPATH)/bin/golangci-lint"
  rm -rf "$TMP"
else
  echo "[dev-setup] golangci-lint $GOLANGCI_WANT already installed"
fi

echo "[dev-setup] Running make generate to scaffold code/docs..."
make generate

echo "[dev-setup] Done. You can now run: make build && make test"
