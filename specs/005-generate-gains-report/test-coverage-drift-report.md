# Test Coverage Drift Report: Generate Yearly Gains And Losses Report

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-05-24
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-analysis.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- The reviewed feature slice is the active Spec Kit feature in `specs/005-generate-gains-report/`.
- Existing task-state interpretation for this run treats only unchecked `- [ ]` tasks as open tasks. Historical `Reopened` labels on checked `[X]` tasks were not treated as blockers.

## Coverage Baseline

- `.specify/memory/constitution.md:91-108`: explicit baseline. Project-owned code must maintain 100% automated coverage, including both line and branch coverage when tooling distinguishes them. Integration tests are the default for user journeys and Ghostfolio-facing workflows, coverage commands and CI workflows must instrument project-owned packages from black-box tests, unit tests are limited to coverage gaps or materially complex units, duplicated unit tests should be removed, and features are incomplete until required tests, coverage gates, and regressions pass.
- `.specify/memory/constitution.md:176-183`: explicit baseline. Every task list must include automated integration testing and coverage verification, PRs must run the repository test workflow, and missing coverage measurement must be added before completing a feature.
- `AGENTS.md:105-120`: explicit baseline. Contract, integration, and unit suites live under `tests/`; coverage tooling is `tools/coverpkg`, `tools/coveragegate`, and `.cov.json`; sync behavior changes require matching checks in contract, integration, and unit coverage.
- `.specify/templates/tasks-template.md:11-15`: explicit baseline. Tests are mandatory, project-owned code must maintain 100% coverage, integration tests with mocked or stubbed outside services are preferred, unit tests are only for complexity or integration-only gaps, and substantially overlapping unit tests should be removed once integration coverage replaces them.
- `specs/005-generate-gains-report/plan.md:41`: explicit feature baseline. This feature requires `make test`, `make coverage`, integration-first Go suites, targeted unit tests for complex basis and rendering rules, and a gated large-history performance path.
- `specs/005-generate-gains-report/plan.md:184-194`: explicit feature baseline. Verification must include contract workflow coverage, integration coverage for protected snapshots, report generation, output, diagnostics, and artifact leakage checks, unit coverage for calculators and IO branches, wrapped-error diagnostics, 16-decimal internal precision regressions, and `make coverage`.
- `specs/005-generate-gains-report/spec.md:487-499`: explicit feature baseline. `QUAL-001`, `QUAL-001a`, `QUAL-001b`, `INT-001a`, `INT-001b`, and `INT-001c` require automated validation for the Sync and Reports context, report generation, diagnostics, currency-context regressions, wrapped-cause diagnostics, and report-run precondition validation.
- `specs/005-generate-gains-report/tasks.md:31`: explicit feature baseline. Automated tests are mandatory, with integration-first coverage, targeted unit tests, `make test`, `make coverage`, and the opt-in large-history performance path.
- `specs/005-generate-gains-report/tasks.md:201-203`: explicit feature baseline. Completed tasks include opt-in 10,000-activity performance coverage, `make test`, `make coverage`, generated coverage artifacts, and feature-specific regression paths.
- `specs/005-generate-gains-report/quickstart.md:27-41`: explicit feature baseline. Contributor verification commands are `make test`, `make coverage`, and the opt-in performance command, with expected successful coverage-gate artifacts.
- `Makefile:17-21`: explicit gate implementation. `make coverage` runs Go coverage across `./cmd/...`, `./internal/...`, `tests/contract`, `tests/integration`, and `tests/unit`, instruments project-owned production packages via `PRODUCTION_PACKAGES`, writes `dist/coverage/coverage.out`, generates `dist/coverage/coverage.xml`, and runs `tools/coveragegate`.
- `tools/coverpkg/main.go:14-50`: explicit gate implementation. The coverage package helper resolves production package import paths for the `-coverpkg` list.
- `tools/coveragegate/main.go:197-248`: explicit gate implementation. The coverage gate enforces 100% statement, line, branch, and per-file line/branch coverage.
- `.cov.json:1-8`: explicit gate configuration. Cobertura output is generated from repository source while excluding `tests` and `tools` from production coverage reporting.
- `.github/workflows/test.yml:41-45`: explicit CI baseline. The workflow runs `make test` and `make coverage`.
- No derived baseline item was required because the loaded policy, feature, coverage command, and CI references define concrete coverage targets, commands, gates, and test-structure expectations.

## Findings

No test-coverage drift was identified in the reviewed scope.

## Notes

- No prior `test-coverage-drift-report.md` existed for this feature, so there were no existing `COV-DRIFT-###` identifiers to preserve.
- Task completion check found no unchecked `- [ ]` tasks in `specs/005-generate-gains-report/tasks.md`.
- `make coverage` passed during this analysis and regenerated `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`.
- `dist/coverage/coverage.xml:1` reports `line-rate="1.00"`, `branch-rate="1.00"`, `lines-covered="5734"`, `lines-valid="5734"`, `branches-covered="1350"`, and `branches-valid="1350"`.
- `make test` passed during this analysis.
- `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestReportPerformanceFlowLargeHistoryFixture -count=1 -v` passed during this analysis and logged completion in about 6.22 seconds for 10,000 activities across 6 calendar years.
- The reviewed feature test surface includes contract coverage in `tests/contract/main_menu_workflow_contract_test.go:13`, `tests/contract/sync_reports_workflow_contract_test.go:17`, `tests/contract/report_generation_workflow_contract_test.go:19`, `tests/contract/markdown_report_contract_test.go:22-56`, and `tests/contract/report_method_selection_contract_test.go:24`.
- The reviewed feature test surface includes integration coverage in `tests/integration/sync_reports_context_flow_test.go:84-544`, `tests/integration/report_generation_flow_test.go:31-280`, `tests/integration/report_failure_flow_test.go:29-328`, `tests/integration/report_cost_basis_methods_flow_test.go:26-183`, and `tests/integration/report_performance_flow_test.go:22`.
- The reviewed feature test surface includes targeted unit coverage in `tests/unit/report_calculation_test.go:23-562`, `tests/unit/report_activity_input_test.go:20-326`, `tests/unit/report_basis_methods_test.go:18-411`, `tests/unit/report_markdown_test.go:21-108`, and `tests/unit/report_output_test.go:20-205`.
