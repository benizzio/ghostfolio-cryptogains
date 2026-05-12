GO ?= go
GOCOVERAGEPLUS ?= $(GO) run github.com/Fabianexe/gocoverageplus@v1.2.0
ARGS ?=
PRODUCTION_PACKAGES := $(shell $(GO) list ./cmd/... ./internal/... | paste -sd,)

.PHONY: run test coverage

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

test:
	$(GO) test ./...

coverage:
	mkdir -p dist/coverage
	$(GO) test ./cmd/... ./internal/... ./tests/contract ./tests/integration -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
