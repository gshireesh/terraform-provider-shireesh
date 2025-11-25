default: generate fmt lint install

# help: ## Show help for all targets.
help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage: make [target]\n\nTargets:\n"} /^# [a-zA-Z_-]+:.*##/ { gsub(/^# /, "", $$1); printf "  %-24s %s\n", $$1, $$2 } /^##@/ { printf "\n%s\n", substr($$0,5)} ' $(MAKEFILE_LIST)

# build: ## Build all Go packages.
build:
	go build -v ./...

# install: ## Install the provider binary to your GOPATH/bin.
install: build
	go install -v ./...

# lint: ## Run golangci-lint across the repo.
lint:
	golangci-lint run

# fmt: ## Format Go code using gofmt.
fmt:
	gofmt -s -w -e .

# test: ## Run unit tests with coverage.
test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# testacc: ## Run acceptance tests (requires TF_ACC=1 and dependencies).
testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...


PROTO_DIR=internal/provider/generated

# proto: ## Generate code via buf for all configured APIs.
.PHONY: proto
proto:
	buf generate

REGEN_OUT=internal/provider/generated
COMPONENT_PROTO=api/shireesh.com/component/component.proto
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

# clean: ## Remove generated artifacts and build outputs.
.PHONY: clean
clean:
	rm -rf docs generated gen $(REGEN_OUT) api/shireesh.com || true

# proto-fmt: ## Format protobufs using buf format.
.PHONY: proto-fmt
proto-fmt:
	cd api; buf format -w $(COMPONENT_PROTO)

# docs: ## Generate docs (sections + site), depends on examples.
.PHONY: docs
docs: gen-examples
	cd tools; go generate ./...
	go run ./cmd/docsectioner

# gen-examples: ## Generate Terraform examples.
.PHONY: gen-examples
gen-examples:
	go run ./cmd/examplergen

# generate: ## Unified code generation pipeline (component, buf, fmt, tidy, examples, docs).
# regen: ## Backwards-compatible alias for `generate`.
.PHONY: generate regen
generate: clean
	$(COMPONENTGEN)
	cd api; buf generate
	$(MAKE) proto-fmt
	go mod tidy
	$(MAKE) gen-examples
	$(MAKE) docs

regen: generate

.PHONY: fmt lint test testacc build install generate regen clean docs gen-examples proto proto-fmt tools help

BINARY_NAME=terraform-provider-shireesh
VERSION=0.0.1-pre
PLUGIN_DIR=~/.terraform.d/plugins/registry.terraform.io/gshireesh/shireesh
OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH ?= $(shell uname -m)
ifeq ($(ARCH),x86_64)
    ARCH=amd64
else ifeq ($(ARCH),arm64)
    ARCH=arm64
else ifeq ($(ARCH),aarch64)
    ARCH=arm64
endif
TF_PLUGIN_DIR = $(OS)_$(ARCH)

# build-local: ## Build and install the provider binary into local Terraform plugin dir.
.PHONY: build-local
build-local:
	# Create the appropriate directory and copy the built binary
	mkdir -p $(PLUGIN_DIR)/$(VERSION)/$(TF_PLUGIN_DIR)
	# Build the Go project
	go build -o $(PLUGIN_DIR)/$(VERSION)/$(TF_PLUGIN_DIR)/$(BINARY_NAME)