
################################################################################
# ===================================================================== info = #
################################################################################

# kuda — MUD client for Aardwolf

################################################################################
# ============================================================ configuration = #
################################################################################

BINARY_NAME=kuda
BIN_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build run test lint tidy fmt clean release help

# set default target
.DEFAULT_GOAL := help

# single shell invocation
.ONESHELL:

# set fast fail
.SHELLFLAGS := -eu -o pipefail -c

################################################################################
# ===================================================================== main = #
################################################################################

##@ Build

all: build                                              ## default to building the project

build: tidy fmt                                         ## build the Kuda binary
	@printf "==> Building $(BINARY_NAME) $(VERSION)...\n"
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./main.go

run: build                                              ## build and run Kuda
	@./$(BIN_DIR)/$(BINARY_NAME)

install: build                                          ## install the binary to $GOPATH/bin
	@go install $(LDFLAGS) ./...

##@ Development

tidy:                                                   ## tidy up go modules
	@printf "==> Tidying modules...\n"
	@go mod tidy

fmt:                                                    ## format go code
	@printf "==> Formatting code...\n"
	@go fmt ./...

##@ Testing

test: lint                                              ## run project tests (includes lint)
	@printf "==> Running tests...\n"
	@go test -v -race ./...

lint:                                                   ## run go vet and golangci-lint
	@printf "==> Running linters...\n"
	@go vet ./...
	@if command -v golangci-lint >/dev/null; then \
		golangci-lint run ./...; \
	else \
		printf "WARNING: golangci-lint not found, skipping...\n"; \
	fi

##@ Release

release:                                                ## build a local snapshot release (requires goreleaser)
	@printf "==> Building snapshot release...\n"
	@goreleaser release --snapshot --clean

##@ Cleanup

clean:                                                  ## remove build artifacts
	@printf "==> Cleaning up...\n"
	@rm -rf $(BIN_DIR)/

##@ Helpers

help:                                                   ## display this help
	@awk 'BEGIN { FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"; } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2; } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); } \
		END { printf ""; }' $(MAKEFILE_LIST)
