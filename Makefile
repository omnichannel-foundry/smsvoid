# Makefile for SmsVoid (Go SMPP capture service).
# `make` (or `make help`) lists available targets.

# --- Preamble ----------------------------------------------------------------
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
MAKEFLAGS += --no-builtin-variables
.DEFAULT_GOAL := help

V ?= 0
ifeq ($(V),1)
  Q :=
else
  Q := @
endif

# --- Project -----------------------------------------------------------------
BINARY  := smsvoid
MODULE  := github.com/gnrfan/smsvoid
CMD     := ./cmd/$(BINARY)

OUTDIR  := bin
TARGET  := $(OUTDIR)/$(BINARY)

PREFIX  ?= $(HOME)/.local
BINDIR  ?= $(PREFIX)/bin
DESTDIR ?=

GO      ?= go
GOFLAGS ?=

# Extra arguments for `make run` / `make test`:
#   make run ARGS='-smpp-addr :2776 -http-user admin -http-pass admin'
#   make test ARGS='-run TestParseSubmit -v'
ARGS ?=

SOURCES := $(shell find . -name '*.go' -not -path './vendor/*' 2>/dev/null)

##@ Build

.PHONY: build
build: $(TARGET) ## Build the binary into bin/

$(TARGET): $(SOURCES)
	$(Q)mkdir -p $(@D)
	$(Q)$(GO) build $(GOFLAGS) -trimpath -o $@ $(CMD)
	$(Q)echo "built $@"

.PHONY: release
release: ## Build a stripped binary
	$(Q)mkdir -p $(OUTDIR)
	$(Q)$(GO) build $(GOFLAGS) -trimpath -ldflags '-s -w' -o $(TARGET) $(CMD)
	$(Q)echo "built $(TARGET) (stripped)"

.PHONY: run
run: build ## Build and run (make run ARGS='...')
	$(Q)$(TARGET) $(ARGS)

##@ Quality

.PHONY: test
test: ## Run tests
	$(Q)$(GO) test $(ARGS) ./...

.PHONY: test-race
test-race: ## Run tests with the race detector, uncached
	$(Q)$(GO) test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	$(Q)$(GO) test -coverprofile=coverage.out ./...
	$(Q)$(GO) tool cover -html=coverage.out

.PHONY: vet
vet: ## Run go vet
	$(Q)$(GO) vet ./...

.PHONY: fmt
fmt: ## Format code with gofmt
	$(Q)gofmt -s -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is unformatted
	$(Q)out=$$(gofmt -l .)
	if [ -n "$$out" ]; then
		echo "unformatted files:" >&2
		echo "$$out" >&2
		exit 1
	fi

.PHONY: tidy
tidy: ## Run go mod tidy
	$(Q)$(GO) mod tidy

.PHONY: tidy-check
tidy-check: ## Fail if go.mod/go.sum are not tidy
	$(Q)$(GO) mod tidy -diff

.PHONY: check
check: fmt-check vet test ## Run everything CI runs

##@ Install

.PHONY: install
install: build ## Install the binary into PREFIX/bin
	$(Q)install -D -m 0755 $(TARGET) $(DESTDIR)$(BINDIR)/$(BINARY)
	$(Q)echo "installed $(DESTDIR)$(BINDIR)/$(BINARY)"

.PHONY: uninstall
uninstall: ## Remove the installed binary
	$(Q)rm -f $(DESTDIR)$(BINDIR)/$(BINARY)

.PHONY: clean
clean: ## Remove build artifacts
	$(Q)rm -rf $(OUTDIR) coverage.out
	$(Q)rm -f $(BINARY)

##@ Help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_%\/-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
	@echo ""

.PHONY: require-%
require-%:
	@command -v $* >/dev/null 2>&1 || { \
		echo "error: '$*' is required but not installed" >&2; exit 1; }

.PHONY: print-%
print-%:
	@echo '$*=$($*)'
