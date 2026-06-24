# GNUmakefile for terraform-provider-kanidm
# Targets use tabs (GNU make requirement).

BINARY        := terraform-provider-kanidm
REGISTRY_HOST := registry.opentofu.org
ORG           := slop-incubator
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
OS_ARCH       := $(shell go env GOOS)_$(shell go env GOARCH)

# Installation path for local development testing.
PLUGIN_DIR := ~/.terraform.d/plugins/$(REGISTRY_HOST)/$(ORG)/kanidm/$(VERSION)/$(OS_ARCH)

.DEFAULT_GOAL := help

##@ Development

.PHONY: build
build: ## Build the provider binary
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

.PHONY: install
install: build ## Install the provider into the local plugin cache for development
	mkdir -p $(PLUGIN_DIR)
	cp $(BINARY) $(PLUGIN_DIR)/$(BINARY)_v$(VERSION)
	@echo "Installed to $(PLUGIN_DIR)"

.PHONY: generate
generate: ## Run go generate (tfplugindocs)
	go generate ./...

.PHONY: docs
docs: generate ## Regenerate provider documentation via tfplugindocs
	tfplugindocs generate --provider-name kanidm
	@echo "Docs written to docs/"

##@ Testing

.PHONY: test
test: ## Run unit tests (no live Kanidm required)
	go test -v -count=1 -race -timeout 120s ./...

.PHONY: test-acceptance
test-acceptance: ## Run acceptance tests against a live Kanidm instance
	@bash scripts/bootstrap-kanidm.sh
	TF_ACC=1 \
	KANIDM_URL=$${KANIDM_URL:-https://localhost:8443} \
	KANIDM_TOKEN=$${KANIDM_TOKEN} \
	go test -v -count=1 -race -timeout 600s ./...

##@ Quality

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format Go source files
	gofmt -s -w .
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

##@ Tooling

.PHONY: scaffold
scaffold: ## Scaffold a new resource: make scaffold RESOURCE=my_resource
ifndef RESOURCE
	$(error RESOURCE is required: make scaffold RESOURCE=my_resource)
endif
	go run ./tools/codegen \
		--spec tools/codegen/specs/$(RESOURCE).yaml \
		--out  internal/resources/$(RESOURCE)/

.PHONY: schema-diff
schema-diff: ## Check for OpenAPI schema drift: make schema-diff KANIDM_URL=https://idm.example.com
ifndef KANIDM_URL
	$(error KANIDM_URL is required: make schema-diff KANIDM_URL=https://idm.example.com)
endif
	go run ./tools/schema-sync \
		--url $(KANIDM_URL) \
		--baseline tools/schema-sync/baseline.json

##@ Release

.PHONY: release-dry-run
release-dry-run: ## Dry-run GoReleaser without publishing
	goreleaser release --snapshot --clean

##@ Cleanup

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist/

##@ Help

.PHONY: help
help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
	/^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } \
	/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
