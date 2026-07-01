GO ?= go
GOCOVERAGEPLUS ?= $(GO) run github.com/Fabianexe/gocoverageplus@v1.2.0
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= $(GO) run golang.org/x/vuln/cmd/govulncheck@v1.5.0
GITLEAKS ?= $(GO) run github.com/zricethezav/gitleaks/v8@v8.30.1
QUALITY_BASE_REF ?= origin/main
ARGS ?=
PRODUCTION_PACKAGES = $(shell $(GO) run ./tools/coverpkg -go $(GO) ./cmd/... ./internal/...)
MODULE_PATH = $(shell $(GO) list -m)
ALL_PACKAGES = $(shell $(GO) list ./...)
NON_EMPIRICAL_PACKAGES = $(filter-out $(MODULE_PATH)/tests/empirical,$(ALL_PACKAGES))
CHANGED_SOURCE_FILES = $$({ git diff --name-only --diff-filter=ACMR $(QUALITY_BASE_REF)...HEAD -- '*.go' 'go.mod' 'go.sum'; git diff --name-only --diff-filter=ACMR -- '*.go' 'go.mod' 'go.sum'; git ls-files --others --exclude-standard -- '*.go' 'go.mod' 'go.sum'; } | sort -u)

.PHONY: run run-dev test test-empirical test-external-integration regenerate-empirical-fixtures coverage quality quality-changed-source-files lint-changed vuln-changed secrets-changed

run:
	$(GO) run ./cmd/ghostfolio-cryptogains $(ARGS)

run-dev:
	$(GO) run ./cmd/ghostfolio-cryptogains --dev-mode $(ARGS)

test: test-empirical
	$(GO) test $(NON_EMPIRICAL_PACKAGES)

test-empirical:
	$(GO) test ./tests/empirical -count=1 -v

test-external-integration:
	GHOSTFOLIO_CRYPTOGAINS_RUN_EXTERNAL_INTEGRATION=1 $(GO) test ./tests/externalintegration -count=1 -v

regenerate-empirical-fixtures:
	$(GO) run ./tools/empiricaloracle --regenerate

coverage:
	mkdir -p dist/coverage
	$(GO) test ./cmd/... ./internal/... ./tests/contract ./tests/empirical ./tests/empirical/fixture ./tests/integration ./tests/unit -covermode=atomic -coverpkg=$(PRODUCTION_PACKAGES) -coverprofile=dist/coverage/coverage.out
	$(GOCOVERAGEPLUS) -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
	$(GO) run ./tools/coveragegate -profile dist/coverage/coverage.out -cobertura dist/coverage/coverage.xml

quality: lint-changed vuln-changed secrets-changed

quality-changed-source-files:
	@changed="$(CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files."; \
	else \
		printf '%s\n' "$$changed"; \
	fi

lint-changed:
	@changed="$(CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping golangci-lint."; \
		exit 0; \
	fi; \
	$(GOLANGCI_LINT) run --new-from-merge-base $(QUALITY_BASE_REF) --whole-files ./...

vuln-changed:
	@changed="$(CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping govulncheck."; \
		exit 0; \
	fi; \
	case "$$changed" in \
		*go.mod*|*go.sum*) \
			$(GOVULNCHECK) ./...; \
			;; \
		*) \
			packages=""; \
			for file in $$changed; do \
				case "$$file" in \
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
			;; \
	esac

secrets-changed:
	@changed="$(CHANGED_SOURCE_FILES)"; \
	if [ -z "$$changed" ]; then \
		printf '%s\n' "No changed source files. Skipping gitleaks."; \
		exit 0; \
	fi; \
	for file in $$changed; do \
		if [ -f "$$file" ]; then \
			$(GITLEAKS) dir --no-banner --redact "$$file"; \
		fi; \
	done
