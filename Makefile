
################################################################################
# ===================================================================== info = #
################################################################################

# kuda — MUD client for Aardwolf

################################################################################
# ============================================================ configuration = #
################################################################################

.PHONY: all build test clean help

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

build:                                                  ## build the Kuda binary
	@printf "==> Building Kuda...\n"
	@go build -o bin/kuda ./main.go

run: build                                              ## build and run Kuda
	@./bin/kuda

##@ Testing

test:                                                   ## run project tests
	@go test ./...

##@ Cleanup

clean:                                                  ## remove build artifacts
	@rm -rf bin/

##@ Helpers

help:                                                   ## display this help
	@awk 'BEGIN { FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"; } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2; } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5); } \
		END { printf ""; }' $(MAKEFILE_LIST)
