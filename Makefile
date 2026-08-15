.PHONY: fmt fmt-check vet lint lint-install test coverage build clean ci preflight docs-sync docs-check

## Pinned golangci-lint version. MUST match .github/workflows/ci.yml so a
## green local run predicts a green CI run. If you bump this, bump the
## version in ci.yml in the same commit — a mismatch means the local
## preflight is lying about CI's verdict.
GOLANGCI_LINT_VERSION := v2.11.4
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

## Format all Go files with gofmt simplification
fmt:
	gofmt -s -w .

## Verify gofmt has nothing to change (fails if formatting is off).
## Uses `gofmt -l` which prints any file that needs formatting — a non-empty
## output means the tree is dirty. Portable across /bin/sh and bash.
fmt-check:
	@out=$$(gofmt -s -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt issues in:"; echo "$$out"; \
		echo "run 'make fmt' to fix"; \
		exit 1; \
	fi

## Run go vet across every package
vet:
	go vet ./...

## Install golangci-lint locally at the version pinned above. Idempotent —
## re-installs only if the version differs from what's already on $PATH.
lint-install:
	@if ! $(GOLANGCI_LINT) --version 2>/dev/null | grep -q "$(GOLANGCI_LINT_VERSION:v%=%)"; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

## Run golangci-lint with the same version CI uses
lint: lint-install
	$(GOLANGCI_LINT) run

## Run tests with the race detector (matches CI)
test:
	go test -race ./...

## Run tests with coverage report (matches CI)
coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

## Build every package (smoke check)
build:
	go build ./...

## Remove build artifacts
clean:
	rm -f coverage.out coverage.html storage_coverage.out

## Regenerate the README package tables from pkg/* doc comments
docs-sync:
	go run ./tools/docsgen

## Verify the README package tables match pkg/* doc comments (CI gate)
docs-check:
	go run ./tools/docsgen -check

## Run all checks (what CI runs)
ci: lint test build docs-check

## Preflight — the full set of checks that MUST pass locally before any
## task is considered complete. Intended to be run before every commit and
## before reporting a task done to the user. Runs the exact same linter
## version CI uses, so a green preflight predicts a green CI run.
##
## Order matters: fmt-check is first (cheapest, catches the most common
## slip), then vet, then lint (errcheck + staticcheck + revive + the
## rest), then race tests, then a build sanity check, then docs-check
## (README package tables must match pkg/* doc comments). Each step
## blocks the next — a formatting failure stops the run before tests
## even start.
preflight: fmt-check vet lint test build docs-check
	@echo ""
	@echo "  ✓ preflight green — safe to commit."
