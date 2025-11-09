default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

#generate:
#	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# Install/verify local tooling for generation and linting.
.PHONY: tools
tools:
	@set -e; \
	if ! command -v buf >/dev/null 2>&1; then \
		OS=$$(uname -s); \
		echo "buf not found; installing..."; \
		if [ "$$OS" = "Darwin" ] && command -v brew >/dev/null 2>&1; then \
			brew install bufbuild/buf/buf; \
		else \
			URI_BASE="https://github.com/bufbuild/buf/releases/download/v1.45.0"; \
			ARCH=$$(uname -m); \
			case "$$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac; \
			if [ "$$OS" = "Linux" ]; then OSLOW=linux; else OSLOW=darwin; fi; \
			tmp=$$(mktemp -d); \
			curl -sSL "$$URI_BASE/buf-$${OSLOW}-$${ARCH}" -o "$$tmp/buf"; \
			chmod +x "$$tmp/buf"; \
			sudo mv "$$tmp/buf" /usr/local/bin/buf; \
			rm -rf "$$tmp"; \
		fi; \
	fi; \
	GOLANGCI_WANT=2.6.1; \
	GOLANGCI_CUR=$$(golangci-lint version 2>/dev/null | sed -n 's/.*version \([^ ]*\).*/\1/p' || true); \
	if [ "$$GOLANGCI_CUR" != "$$GOLANGCI_WANT" ]; then \
		echo "Installing golangci-lint $$GOLANGCI_WANT (current='$$GOLANGCI_CUR')..."; \
		OS=$$(uname -s); ARCH=$$(uname -m); \
		case "$$ARCH" in x86_64) ARCH=amd64 ;; aarch64) ARCH=arm64 ;; esac; \
		if [ "$$OS" = "Linux" ]; then OSLOW=linux; else OSLOW=darwin; fi; \
		URL="https://github.com/golangci/golangci-lint/releases/download/v$${GOLANGCI_WANT}/golangci-lint-$${GOLANGCI_WANT}-$${OSLOW}-$${ARCH}.tar.gz"; \
		tmp=$$(mktemp -d); \
		curl -sSL "$$URL" -o "$$tmp/gcl.tar.gz"; \
		tar -xzf "$$tmp/gcl.tar.gz" -C "$$tmp"; \
		BIN=$$(find "$$tmp" -type f -name golangci-lint -perm +111 | head -n1); \
		install -m 0755 "$$BIN" "$$(go env GOPATH)/bin/golangci-lint"; \
		rm -rf "$$tmp"; \
	else \
		echo "golangci-lint $$GOLANGCI_WANT already installed"; \
	fi

PROTO_DIR=internal/provider/generated

.PHONY: proto
proto:
	buf generate

REGEN_OUT=internal/provider/generated
COMPONENT_PROTO=internal/provider/generated/components.proto
SERVICE_NAME?=GrpcTerraformService
PROTO_PKG?=component
GO_PKG_PREFIX?=github.com/gshireesh/terraform-provider-shireesh/internal/provider/generated
HTTP?=false

ifeq ($(HTTP),true)
HTTP_FLAG=-http=true
else
HTTP_FLAG=
endif

COMPONENTGEN=go run ./cmd/componentgen $(HTTP_FLAG) -service-name=$(SERVICE_NAME) -proto-package=$(PROTO_PKG) -go-package-prefix=$(GO_PKG_PREFIX)

.PHONY: clean
clean:
	rm -rf docs generated gen $(REGEN_OUT) || true

.PHONY: proto-fmt
proto-fmt:
	buf format -w $(COMPONENT_PROTO)

.PHONY: docs
docs: gen-examples
	cd tools; go generate ./...
	go run ./cmd/docsectioner

.PHONY: gen-examples
gen-examples:
	go run ./cmd/examplergen

# Unified generate pipeline (previously 'regen').
.PHONY: generate regen
generate: tools clean
	$(COMPONENTGEN)
	buf generate
	$(MAKE) proto-fmt
	go mod tidy
	$(MAKE) gen-examples
	$(MAKE) docs

# Backwards-compatible alias
regen: generate

.PHONY: fmt lint test testacc build install generate regen clean docs gen-examples proto proto-fmt tools
