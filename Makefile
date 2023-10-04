# See https://tech.davis-hansson.com/p/make/
SHELL := bash
.DELETE_ON_ERROR:
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := all
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-print-directory
BIN := .tmp/bin
COPYRIGHT_YEARS := 2023
LICENSE_IGNORE := -e testdata/
# Set to use a different compiler. For example, `GO=go1.18rc1 make test`.
GO ?= go
LATEST_VERSION = $(shell git describe --tags --abbrev=0)
CURRENT_VERSION = $(shell git describe --tags --always --dirty)
# If not on release tag, this is a dev build. Add suffix to version.
ifneq ($(CURRENT_VERSION), $(LATEST_VERSION))
	DEV_BUILD_VERSION_DIRECTIVE = buildVersionSuffix=-$(shell git describe --exclude '*' --always --dirty)
else
	DEV_BUILD_VERSION_DIRECTIVE = buildVersion=$(CURRENT_VERSION)
endif

.PHONY: help
help: ## Describe useful make targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-30s %s\n", $$1, $$2}'

.PHONY: all
all: ## Build, test, and lint (default)
	$(MAKE) test
	$(MAKE) lint

.PHONY: clean
clean: ## Delete intermediate build artifacts
	@# -X only removes untracked files, -d recurses into directories, -f actually removes files/dirs
	git clean -Xdf

.PHONY: test
test: build ## Run unit tests
	$(GO) test -vet=off -race -cover ./...

.PHONY: build
build: generate ## Build all packages
	$(GO) build ./...

.PHONY: generate
generate: $(BIN)/license-header ## Regenerate code and licenses
	cd internal && buf generate proto
	@# We want to operate on a list of modified and new files, excluding
	@# deleted and ignored files. git-ls-files can't do this alone. comm -23 takes
	@# two files and prints the union, dropping lines common to both (-3) and
	@# those only in the second file (-2). We make one git-ls-files call for
	@# the modified, cached, and new (--others) files, and a second for the
	@# deleted files.
	comm -23 \
		<(git ls-files --cached --modified --others --no-empty-directory --exclude-standard | sort -u | grep -v $(LICENSE_IGNORE) ) \
		<(git ls-files --deleted | sort -u) | \
		xargs $(BIN)/license-header \
			--license-type apache \
			--copyright-holder "Buf Technologies, Inc." \
			--year-range "$(COPYRIGHT_YEARS)"

.PHONY: lint
lint: $(BIN)/golangci-lint ## Lint
	$(GO) vet ./...
	$(BIN)/golangci-lint run

.PHONY: lintfix
lintfix: $(BIN)/golangci-lint ## Automatically fix some lint errors
	$(BIN)/golangci-lint run --fix

.PHONY: install
install: ## Install all binaries
	$(GO) install -ldflags '-X "github.com/bufbuild/knit-go/internal/app/knitgateway.$(DEV_BUILD_VERSION_DIRECTIVE)"' ./...

.PHONY: upgrade
upgrade: ## Upgrade dependencies
	go get -u -t ./...
	go mod tidy -v

.PHONY: checkgenerate
checkgenerate:
	@# Used in CI to verify that `make generate` doesn't produce a diff.
	test -z "$$(git status --porcelain | tee /dev/stderr)"

.PHONY: release
release:
	go install github.com/goreleaser/goreleaser@v1.16.2
	goreleaser release

.PHONY: checkrelease
checkrelease:
	go install github.com/goreleaser/goreleaser@v1.16.2
	# skips some validation and doesn't actually publish a release, just to test
	# that building a release works
	goreleaser release --clean --snapshot

$(BIN)/license-header: Makefile
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install \
		  github.com/bufbuild/buf/private/pkg/licenseheader/cmd/license-header@v1.12.0

$(BIN)/golangci-lint: Makefile
	@mkdir -p $(@D)
	GOBIN=$(abspath $(@D)) $(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.52.0
