GO ?= go
GOCOVERAGEPLUS ?= gocoverageplus
ARGS ?=

.PHONY: run test coverage

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

test:
	$(GO) test ./...

coverage:
	mkdir -p dist/coverage
	$(GO) test ./... -covermode=atomic -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
