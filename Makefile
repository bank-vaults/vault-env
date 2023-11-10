# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

export PATH := $(abspath bin/):${PATH}

##@ General

# Targets commented with ## will be visible in "make help" info.
# Comments marked with ##@ will be used as categories for a group of targets.

.PHONY: help
default: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: up
up: ## Start development environment
	docker compose up -d

.PHONY: down
down: ## Destroy development environment
	docker compose down -v

##@ Build

.PHONY: build
build: ## Build binary
	@mkdir -p build
	go build -race -o build/vault-env .

.PHONY: container-image
container-image: ## Build container image
	docker build .

.PHONY: binary-snapshot
binary-snapshot: ## Build binary snapshot
	$(GORELEASER_BIN) release --rm-dist --skip-publish --snapshot

.PHONY: artifacts
artifacts: container-image binary-snapshot
artifacts: ## Build artifacts

##@ Checks

.PHONY: check
check: test lint ## Run lint checks and tests

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: lint
lint: lint-go lint-docker lint-yaml
lint: ## Run linters

.PHONY: lint-go
lint-go:
	$(GOLANGCI_LINT_BIN) run $(if ${CI},--out-format github-actions,)

.PHONY: lint-docker
lint-docker:
	$(HADOLINT_BIN) Dockerfile

.PHONY: lint-yaml
lint-yaml:
	$(YAMLLINT_BIN) $(if ${CI},-f github,) --no-warnings .

.PHONY: license-check
license-check: ## Run license check
	$(LICENSEI_BIN) check
	$(LICENSEI_BIN) header

.PHONY: fmt
fmt: ## Format code
	$(GOLANGCI_LINT_BIN) run --fix

##@ Dependencies

deps: bin/golangci-lint bin/licensei bin/goreleaser
deps: ## Install dependencies

# Dependency versions
GOLANGCI_VERSION = 1.53.3
LICENSEI_VERSION = 0.8.0
GORELEASER_VERSION = 1.18.2

# Dependency binaries
GOLANGCI_LINT_BIN := golangci-lint
LICENSEI_BIN := licensei
GORELEASER_BIN := kind

# If we have "bin" dir, use those binaries instead
ifneq ($(wildcard ./bin/.),)
	GOLANGCI_LINT_BIN := bin/$(GOLANGCI_LINT_BIN)
	LICENSEI_BIN := bin/$(LICENSEI_BIN)
	GORELEASER_BIN := bin/$(GORELEASER_BIN)
endif

bin/golangci-lint:
	@mkdir -p bin
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | bash -s -- v${GOLANGCI_VERSION}

bin/licensei:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s -- v${LICENSEI_VERSION}

bin/goreleaser:
	@mkdir -p bin
	@mkdir -p tmpgoreleaser
	curl -sfL https://goreleaser.com/static/run | VERSION=v${GORELEASER_VERSION} TMPDIR=${PWD}/tmpgoreleaser bash -s -- --version
	mv tmpgoreleaser/goreleaser bin/
	@rm -rf tmpgoreleaser
