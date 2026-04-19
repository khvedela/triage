# kubediag — Kubernetes diagnostic CLI
# https://github.com/khvedela/kubediag

# -----------------------------------------------------------------------------
# Variables
# -----------------------------------------------------------------------------
MODULE        := github.com/khvedela/kubediag
BIN_DIR       := bin
BINARY        := $(BIN_DIR)/kubediag
PLUGIN_BINARY := $(BIN_DIR)/kubectl-kubediag

VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X $(MODULE)/cmd.version=$(VERSION) \
  -X $(MODULE)/cmd.commit=$(COMMIT) \
  -X $(MODULE)/cmd.buildDate=$(BUILD_DATE)

GO        ?= go
GOFLAGS   ?=
TESTFLAGS ?= -race -count=1

# -----------------------------------------------------------------------------
# Targets
# -----------------------------------------------------------------------------
.PHONY: all
all: lint test build

.PHONY: build
build: ## Build the kubediag binary
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) ./

.PHONY: build-plugin
build-plugin: build ## Build kubectl-kubediag (symlink to same binary)
	@ln -sf kubediag $(PLUGIN_BINARY)

.PHONY: install
install: ## go install into $GOPATH/bin
	$(GO) install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./

.PHONY: test
test: ## Run unit tests
	$(GO) test $(TESTFLAGS) ./...

.PHONY: test-short
test-short: ## Run fast unit tests (no race)
	$(GO) test -count=1 -short ./...

.PHONY: test-e2e
test-e2e: ## Run envtest-based e2e tests (downloads kube-apiserver binary on first run)
	$(GO) test -tags=e2e -timeout=10m ./test/e2e/...

.PHONY: cover
cover: ## Generate coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: lint
lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	$(GO) fmt ./...
	@which goimports > /dev/null && goimports -w -local $(MODULE) . || true

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	$(GO) mod tidy

.PHONY: docgen
docgen: build ## Regenerate docs/rules.md from rule registry
	$(GO) run ./hack/docgen > docs/rules.md

.PHONY: snapshot
snapshot: ## Dry-run release with goreleaser
	goreleaser release --snapshot --clean

.PHONY: site-install
site-install: ## Install website dependencies
	cd website && npm install

.PHONY: site-dev
site-dev: ## Run the Docusaurus dev server
	cd website && npm run start

.PHONY: site-build
site-build: ## Build the static docs site
	cd website && npm run build

.PHONY: clean
clean: ## Remove build artifacts
	@rm -rf $(BIN_DIR) dist coverage.out coverage.html

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
