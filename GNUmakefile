default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

PROTO_DIR=internal/provider/generated

.PHONY: proto
proto:
	buf generate

REGEN_OUT=internal/provider/generated
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

.PHONY: clean-generated
clean-generated:
	rm -f $(REGEN_OUT)/*.gen.go $(REGEN_OUT)/*.proto || true
	rm -rf docs generated gen || true

.PHONY: docs
docs: generate gen-examples
	cd tools; go generate ./...
	go run ./cmd/docsectioner

.PHONY: gen-examples
gen-examples:
	go run ./cmd/examplergen

.PHONY: regen
regen: clean-generated
	$(COMPONENTGEN)
	buf generate
	$(MAKE) gen-examples
	$(MAKE) docs

.PHONY: fmt lint test testacc build install generate
