# Test Coverage Drift Report: Report Base Currency Conversion

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-06-28
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-control.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- Reviewed scope is the active feature slice in `specs/007-currency-conversion-strategy/`, the implementation and test paths named by `tasks.md`, and the repository coverage gate definitions needed to verify the same baseline.

## Coverage Baseline

- `.specify/memory/constitution.md:109-129` requires project-owned code to maintain 100% automated test coverage, both line and branch coverage when tooling distinguishes them, integration tests as the default for user journeys and Ghostfolio-facing workflows, coverage instrumentation that counts black-box contract and integration test packages, targeted unit tests only when integration tests cannot realistically cover the risk, removal of duplicated unit tests, and passing coverage gates before a feature is complete.
- `.specify/memory/constitution.md:207-218` requires every task list to include automated integration testing and coverage verification, and states that adding measurement is a prerequisite when tooling cannot measure a required gate.
- `.specify/templates/tasks-template.md:12-18` requires automated tests for project-owned code, integration coverage, coverage verification, and targeted unit tests only when justified by complexity or integration-test gaps.
- `AGENTS.md:53-55` identifies the Go test stack, coverage tooling, and CI workflow path; `AGENTS.md:115-133` defines the repository test layout and coverage tools; `AGENTS.md:135-137` requires matching coverage in contract, integration, and unit suites when sync behavior changes and identifies `.cov.json` as maintained coverage expectations.
- `spec.md:156-161` requires new project-owned contract, integration, external integration, and unit coverage for conversion boundaries, rate-source selection, audit details, provider HTTP client compatibility, and failure handling while keeping empirical data read-only.
- `spec.md:173-193` defines the feature validation sets and requires final coverage validation to produce and check `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`.
- `plan.md:195-208` defines the feature test strategy across contract, integration, unit, external integration, regression, performance, and maintained coverage-gate validation.
- `quickstart.md:16-36` defines the maintained verification flow through `make coverage`, including `dist/coverage/coverage.out`, `dist/coverage/coverage.xml`, `gocoverageplus`, and `tools/coveragegate`.
- `contracts/rate-provider-integration.md:133-158` requires deterministic default provider fixtures and separate opt-in live external integration tests that are skipped unless explicitly enabled.
- `Makefile:29-33` implements the maintained coverage command with `-coverpkg` over project-owned production packages and black-box contract, empirical, integration, and unit test packages.
- `.github/workflows/test.yml:41-45` runs `make test` and `make coverage` for pull request workflows.
- `.cov.json:1-8` configures Cobertura coverage export and excludes test/tool paths from source-path accounting.
- `tools/coveragegate/main.go:197-244` enforces 100% statement, line, branch, and per-file coverage from the Go cover profile and Cobertura report.
- `dist/coverage/coverage.xml:1` records the current generated line and branch totals as `7177/7177` lines and `1733/1733` branches.
- No derived baseline items were needed because explicit policy, feature, command, and gate references were present.

## Findings

No test-coverage drift was identified in the reviewed active feature slice.

## Notes

- No previous `test-coverage-drift-report.md` existed, so there were no `COV-DRIFT-###` identifiers to preserve.
- Local validation run for this report: `make coverage` passed and refreshed `dist/coverage/coverage.out` plus `dist/coverage/coverage.xml`.
- Local validation run for this report: `go run ./tools/coveragegate -profile dist/coverage/coverage.out -cobertura dist/coverage/coverage.xml` passed with no gate output.
- Local validation run for this report: `go test ./tests/externalintegration -count=1 -v` passed with the expected default skip from `tests/externalintegration/helpers_test.go:11-21`.
- Local validation run for this report: `GHOSTFOLIO_CRYPTOGAINS_RUN_EXTERNAL_INTEGRATION=1 go test ./tests/externalintegration -count=1 -v` passed against the fixed historical observations in `tests/externalintegration/currency_provider_live_test.go:16-66`.
- Feature-specific black-box coverage evidence includes base-currency contract coverage in `tests/contract/report_generation_workflow_contract_test.go:158-179`, provider contract coverage in `tests/contract/rate_provider_integration_contract_test.go:20-138`, Markdown audit and asset-detail contract coverage in `tests/contract/markdown_report_contract_test.go:19-252`, mixed-currency integration coverage in `tests/integration/report_generation_flow_test.go:357-583`, large-fixture responsiveness coverage in `tests/integration/report_generation_responsiveness_test.go:17-83`, conversion failure matrix coverage in `tests/integration/report_failure_flow_test.go:296-400`, and production diagnostic redaction coverage in `tests/integration/diagnostic_redaction_test.go:74-147`.
- No root-level `coverage.cov` or `coverage.xml` artifact was present during this review.
