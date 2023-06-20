# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

export PATH := $(abspath bin/):${PATH}

# Dependency versions
GOLANGCI_VERSION = 1.53.3
LICENSEI_VERSION = 0.8.0
GORELEASER_VERSION = 1.18.2

.PHONY: up
up: ## Start development environment
	docker compose up -d

.PHONY: stop
stop: ## Stop development environment
	docker compose stop

.PHONY: down
down: ## Destroy development environment
	docker compose down -v

.PHONY: build
build: ## Build binary
	@mkdir -p build
	go build -race -o build/vault-env .

.PHONY: artifacts
artifacts: container-image binary-snapshot
artifacts: ## Build artifacts

.PHONY: container-image
container-image: ## Build container image
	docker build .

.PHONY: binary-snapshot
binary-snapshot: ## Build binary snapshot
	goreleaser release --rm-dist --skip-publish --snapshot

.PHONY: check
check: test lint ## Run checks (tests and linters)

.PHONY: test
test: ## Run tests
	go test -race -v ./...

.PHONY: lint
lint: lint-go lint-docker lint-yaml
lint: ## Run linters

.PHONY: lint-go
lint-go:
	golangci-lint run $(if ${CI},--out-format github-actions,)

.PHONY: lint-docker
lint-docker:
	hadolint Dockerfile

.PHONY: lint-yaml
lint-yaml:
	yamllint $(if ${CI},-f github,) --no-warnings .

.PHONY: fmt
fmt: ## Format code
	golangci-lint run --fix

.PHONY: license-check
license-check: ## Run license check
	licensei check
	licensei header

deps: bin/golangci-lint bin/licensei bin/goreleaser
deps: ## Install dependencies

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

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}'
