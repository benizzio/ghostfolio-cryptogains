GO ?= go
GOCOVERAGEPLUS ?= $(GO) run github.com/Fabianexe/gocoverageplus@v1.2.0
ARGS ?=
PRODUCTION_PACKAGES = $(shell $(GO) run ./tools/coverpkg -go $(GO) ./cmd/... ./internal/...)

.PHONY: run run-dev test coverage

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

run-dev:
	$(GO) run ./cmd/ghostfolio-cryptogains --dev-mode $(ARGS)

test:
	$(GO) test ./...

coverage:
	mkdir -p dist/coverage
	$(GO) test ./cmd/... ./internal/... ./tests/contract ./tests/integration -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
