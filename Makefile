# wasm-go Makefile
# Dual toolchain build targets for Higress WASM plugins:
# - TinyGo (backward compatible, default)
# - Go 1.24+ native Wasm (wasip1)

SHELL := /bin/bash

# Plugin configuration
PLUGIN_DIR ?= examples/request-block
PLUGIN_NAME ?= plugin.wasm
GO ?= go

# TinyGo settings
TINYGO ?= tinygo
TINYGO_TARGET ?= wasi

# Go native Wasm settings
GOOS_WASM ?= wasip1
GOARCH_WASM ?= wasm
LDFLAGS ?= -s -w

.PHONY: help
help: ## Show this help message
	@echo "wasm-go build targets:"
	@echo "  make build PLUGIN_DIR=<dir>     Build with TinyGo (default)"
	@echo "  make build-native PLUGIN_DIR=<dir>  Build with Go native wasip1"
	@echo "  make build-all PLUGIN_DIR=<dir>     Build with both toolchains"
	@echo "  make test                        Run unit tests"
	@echo "  make test-native                 Run tests with Go native Wasm (GOOS=wasip1)"
	@echo "  make lint                        Run golangci-lint"
	@echo "  make clean                       Remove build artifacts"
	@echo ""
	@echo "Examples:"
	@echo "  make build PLUGIN_DIR=examples/request-block"
	@echo "  make build-native PLUGIN_DIR=examples/http-call"
	@echo "  make build-all PLUGIN_DIR=examples/request-block"

# ==================== Build Targets ====================

.PHONY: build
build: ## Build plugin with TinyGo (backward compatible)
	@echo "[TinyGo] Building $(PLUGIN_DIR)..."
	cd $(PLUGIN_DIR) && \
		GOOS=$(GOOS_WASM) GOARCH=$(GOARCH_WASM) $(TINYGO) build \
			-target=$(TINYGO_TARGET) \
			-o $(PLUGIN_NAME) \
			main.go
	@echo "[TinyGo] Output: $(PLUGIN_DIR)/$(PLUGIN_NAME)"

.PHONY: build-native
build-native: ## Build plugin with Go 1.24+ native Wasm
	@echo "[Go Native] Building $(PLUGIN_DIR)..."
	cd $(PLUGIN_DIR) && \
		GOOS=$(GOOS_WASM) GOARCH=$(GOARCH_WASM) $(GO) build \
			-buildmode=c-shared \
			-ldflags="$(LDFLAGS)" \
			-o $(PLUGIN_NAME) \
			main.go
	@echo "[Go Native] Output: $(PLUGIN_DIR)/$(PLUGIN_NAME)"

.PHONY: build-all
build-all: ## Build plugin with both toolchains
	@echo "=== Building with TinyGo ==="
	@$(MAKE) build PLUGIN_DIR=$(PLUGIN_DIR) PLUGIN_NAME=plugin.tinygo.wasm
	@echo ""
	@echo "=== Building with Go Native ==="
	@$(MAKE) build-native PLUGIN_DIR=$(PLUGIN_DIR) PLUGIN_NAME=plugin.gonative.wasm
	@echo ""
	@echo "=== Size Comparison ==="
	@echo -n "TinyGo:    " && ls -lh $(PLUGIN_DIR)/plugin.tinygo.wasm 2>/dev/null | awk '{print $$5}' || echo "N/A"
	@echo -n "Go Native: " && ls -lh $(PLUGIN_DIR)/plugin.gonative.wasm 2>/dev/null | awk '{print $$5}' || echo "N/A"

# ==================== Test Targets ====================

.PHONY: test
test: ## Run unit tests (current toolchain)
	$(GO) test ./... -count=1

.PHONY: test-race
test-race: ## Run unit tests with race detector
	$(GO) test ./... -race -count=1

.PHONY: test-native
test-native: ## Run tests with GOOS=wasip1 to verify Go native Wasm compatibility
	GOOS=$(GOOS_WASM) GOARCH=$(GOARCH_WASM) $(GO) test \
		./internal/abi/... -count=1 -v

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	$(GO) test ./... -v -count=1

# ==================== Lint Targets ====================

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format Go code
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

# ==================== Utility Targets ====================

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@find . -name "*.wasm" -type f -delete 2>/dev/null || true
	@echo "Done."

.PHONY: deps
deps: ## Download and tidy dependencies
	$(GO) mod download
	$(GO) mod tidy
