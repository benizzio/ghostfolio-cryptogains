GO ?= go
GOCOVERAGEPLUS ?= $(GO) run github.com/Fabianexe/gocoverageplus@v1.2.0
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= $(GO) run golang.org/x/vuln/cmd/govulncheck@v1.5.0
GITLEAKS ?= $(GO) run github.com/zricethezav/gitleaks/v8@v8.30.1
QUALITY_BASE_REF ?= origin/main
ARGS ?=
PRODUCTION_PACKAGES = $(shell $(GO) run ./tools/coverpkg -go $(GO) ./cmd/... ./internal/...)
TEST_UNIT_PACKAGES = ./cmd/... ./internal/... ./tests/unit
TEST_CONTRACT_PACKAGES = ./tests/contract
TEST_INTEGRATION_PACKAGES = ./tests/integration
TEST_EMPIRICAL_PACKAGES = ./tests/empirical/...
TEST_TOOL_PACKAGES = ./tools/empiricaloracle
TEST_EXTERNAL_INTEGRATION_PACKAGES = ./tests/externalintegration
TEST_PERFORMANCE_PACKAGES = ./tests/performance
TEST_DETERMINISTIC_PACKAGES = $(TEST_UNIT_PACKAGES) $(TEST_CONTRACT_PACKAGES) $(TEST_INTEGRATION_PACKAGES) $(TEST_EMPIRICAL_PACKAGES)
CHANGED_SOURCE_FILES = $$({ git diff --name-only --diff-filter=ACMR $(QUALITY_BASE_REF)...HEAD -- '*.go' 'go.mod' 'go.sum'; git diff --name-only --diff-filter=ACMR -- '*.go' 'go.mod' 'go.sum'; git diff --cached --name-only --diff-filter=ACMR -- '*.go' 'go.mod' 'go.sum'; git ls-files --others --exclude-standard -- '*.go' 'go.mod' 'go.sum'; } | sort -u)
QUALITY_CHANGED_SOURCE_CACHE ?=
READ_CHANGED_SOURCE_FILES = $$(if [ -n "$(QUALITY_CHANGED_SOURCE_CACHE)" ] && [ -f "$(QUALITY_CHANGED_SOURCE_CACHE)" ]; then while IFS= read -r file; do printf '%s\n' "$$file"; done < "$(QUALITY_CHANGED_SOURCE_CACHE)"; else printf '%s\n' "$(CHANGED_SOURCE_FILES)"; fi)

.PHONY: run run-dev test test-unit test-contract test-integration test-empirical test-tools test-external-integration test-performance regenerate-empirical-fixtures coverage coverage-unit coverage-contract coverage-integration coverage-empirical coverage-tools coverage-external-integration coverage-performance quality quality-changed-source-files lint-changed vuln-changed secrets-changed

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

run-dev:
	$(GO) run ./cmd/ghostfolio-cryptogains --dev-mode $(ARGS)

test: test-unit test-contract test-integration test-empirical test-tools

test-unit:
	$(GO) test $(TEST_UNIT_PACKAGES) -count=1

test-contract:
	$(GO) test $(TEST_CONTRACT_PACKAGES) -count=1

test-integration:
	$(GO) test $(TEST_INTEGRATION_PACKAGES) -count=1

test-empirical:
	$(GO) test $(TEST_EMPIRICAL_PACKAGES) -count=1 -v

test-tools:
	$(GO) test $(TEST_TOOL_PACKAGES) -count=1

test-external-integration:
	GHOSTFOLIO_CRYPTOGAINS_RUN_EXTERNAL_INTEGRATION=1 $(GO) test $(TEST_EXTERNAL_INTEGRATION_PACKAGES) -count=1 -v

test-performance:
	$(GO) test -tags=performance $(TEST_PERFORMANCE_PACKAGES) -count=1 -v -parallel=1 -timeout=10m

regenerate-empirical-fixtures:
	$(GO) run ./tools/empiricaloracle --regenerate

coverage:
	mkdir -p dist/coverage
	rm -f dist/coverage/coverage.out dist/coverage/coverage.xml
	$(GO) test $(TEST_DETERMINISTIC_PACKAGES) -count=1 -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
	$(GO) run ./tools/coveragegate -profile dist/coverage/coverage.out -cobertura dist/coverage/coverage.xml
	$(MAKE) --no-print-directory coverage-tools

coverage-unit:
	mkdir -p dist/coverage
	$(GO) test $(TEST_UNIT_PACKAGES) -count=1 -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/unit.out

coverage-contract:
	mkdir -p dist/coverage
	$(GO) test $(TEST_CONTRACT_PACKAGES) -count=1 -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/contract.out

coverage-integration:
	mkdir -p dist/coverage
	$(GO) test $(TEST_INTEGRATION_PACKAGES) -count=1 -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/integration.out

coverage-empirical:
	mkdir -p dist/coverage
	$(GO) test $(TEST_EMPIRICAL_PACKAGES) -count=1 -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/empirical.out

coverage-tools:
	mkdir -p dist/coverage
	$(GO) test $(TEST_TOOL_PACKAGES) -count=1 -covermode=atomic -coverprofile=dist/coverage/tools.out

coverage-external-integration:
	mkdir -p dist/coverage
	GHOSTFOLIO_CRYPTOGAINS_RUN_EXTERNAL_INTEGRATION=1 $(GO) test $(TEST_EXTERNAL_INTEGRATION_PACKAGES) -count=1 -v -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/external-integration.out

coverage-performance:
	mkdir -p dist/coverage
	$(GO) test -tags=performance $(TEST_PERFORMANCE_PACKAGES) -count=1 -v -parallel=1 -timeout=15m -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/performance.out

quality:
	@tmp=$$(mktemp) || exit 1; \
	trap 'rm -f "$$tmp"' EXIT; \
	changed="$(CHANGED_SOURCE_FILES)"; \
	if [ -n "$$changed" ]; then \
		printf '%s\n' "$$changed" > "$$tmp"; \
	else \
		: > "$$tmp"; \
	fi; \
	quality_status=0; \
	$(MAKE) --no-print-directory QUALITY_CHANGED_SOURCE_CACHE="$$tmp" lint-changed || quality_status=1; \
	$(MAKE) --no-print-directory QUALITY_CHANGED_SOURCE_CACHE="$$tmp" vuln-changed || quality_status=1; \
	$(MAKE) --no-print-directory QUALITY_CHANGED_SOURCE_CACHE="$$tmp" secrets-changed || quality_status=1; \
	exit $$quality_status

quality-changed-source-files:
	@changed="$(READ_CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files."; \
	else \
		printf '%s\n' "$$changed"; \
	fi

lint-changed:
	@changed="$(READ_CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping golangci-lint."; \
		exit 0; \
	fi; \
	$(GOLANGCI_LINT) run --new-from-merge-base $(QUALITY_BASE_REF) --whole-files ./...

vuln-changed:
	@changed="$(READ_CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping govulncheck."; \
		exit 0; \
	fi; \
	case "$$changed" in \
		*go.mod*|*go.sum*) \
			GOFLAGS="-tags=performance" $(GOVULNCHECK) ./...; \
			;; \
		*) \
			packages=""; performance_changed=0; \
			for file in $$changed; do \
				case "$$file" in \
					tests/performance/*.go) performance_changed=1 ;; \
					*.go) \
						dir=$$(dirname "$$file"); \
						case "$$dir" in \
							.) packages="$$packages ." ;; \
							*) packages="$$packages ./$$dir" ;; \
						esac; \
						;; \
				esac; \
			done; \
			packages=$$(printf '%s\n' $$packages | sort -u | tr '\n' ' '); \
			if [ -z "$$packages" ]; then \
				printf '%s\n' "No changed Go packages. Skipping govulncheck."; \
			else \
				$(GOVULNCHECK) $$packages; \
			fi; \
			if [ "$$performance_changed" = "1" ]; then \
				GOFLAGS="-tags=performance" $(GOVULNCHECK) ./tests/performance; \
			fi; \
			;; \
	esac

secrets-changed:
	@changed="$(READ_CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping gitleaks."; \
		exit 0; \
	fi; \
	for file in $$changed; do \
		if [ -f "$$file" ]; then \
			$(GITLEAKS) dir --no-banner --redact "$$file"; \
		fi; \
	done
