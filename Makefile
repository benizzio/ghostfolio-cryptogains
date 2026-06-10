GO ?= go
GOCOVERAGEPLUS ?= $(GO) run github.com/Fabianexe/gocoverageplus@v1.2.0
ARGS ?=
PRODUCTION_PACKAGES = $(shell $(GO) run ./tools/coverpkg -go $(GO) ./cmd/... ./internal/...)
MODULE_PATH = $(shell $(GO) list -m)
ALL_PACKAGES = $(shell $(GO) list ./...)
NON_EMPIRICAL_PACKAGES = $(filter-out $(MODULE_PATH)/tests/empirical,$(ALL_PACKAGES))

.PHONY: run run-dev test test-empirical regenerate-empirical-fixtures coverage

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

run-dev:
	$(GO) run ./cmd/ghostfolio-cryptogains --dev-mode $(ARGS)

test: test-empirical
	$(GO) test $(NON_EMPIRICAL_PACKAGES)

test-empirical:
	$(GO) test ./tests/empirical -count=1 -v

regenerate-empirical-fixtures:
	$(GO) run ./tools/empiricaloracle --regenerate

coverage:
	mkdir -p dist/coverage
	$(GO) test ./cmd/... ./internal/... ./tests/contract ./tests/integration ./tests/unit -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
	$(GO) run ./tools/coveragegate -profile dist/coverage/coverage.out -cobertura dist/coverage/coverage.xml
