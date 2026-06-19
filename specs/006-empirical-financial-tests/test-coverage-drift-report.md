# Test Coverage Drift Report: Empirical Solidified Financial Tests

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-06-14
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-control.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Coverage Baseline

- `.specify/memory/constitution.md:109-129`: project-owned code must maintain 100% automated coverage; line and branch coverage must remain 100% when tooling distinguishes them; coverage commands must count execution from black-box contract and integration packages; a feature is incomplete until required tests, coverage gates, and relevant regressions pass.
- `.specify/memory/constitution.md:205-217`: implementation plans and task lists must include coverage verification, and missing measurement is a prerequisite before completing the feature.
- `AGENTS.md:54-56`: repository testing uses Go standard `testing`, `gocoverageplus`, local coverage gate tooling, and CI workflow conventions.
- `AGENTS.md:106-120`: repository test structure includes contract, integration, unit, package-local, and optional empirical integration tests; `tools/coverpkg`, `tools/coveragegate`, and `.cov.json` define the maintained coverage expectation.
- `.specify/templates/tasks-template.md:12-18`: automated tests are mandatory for project-owned code, task lists must include integration coverage and coverage verification, and empirical financial tests are optional only for applicable financial-calculation scopes.
- `Makefile:26-30` (derived gate implementation): `make coverage` runs Go tests with `-coverpkg=$(PRODUCTION_PACKAGES)`, converts the profile through `gocoverageplus`, and runs `tools/coveragegate`.
- `.cov.json:4-8` (derived maintained report scope): generated coverage reports exclude `.cache`, `tests`, and `tools`, so the maintained gate is focused on the production source package set.
- `tools/coveragegate/main.go:197-244`: the local gate rejects non-100% statement, line, branch, file-line, or file-branch coverage.

## Findings

### COV-DRIFT-001: Coverage Gate Fails On Decimal Policy Startup Error Branch

**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:109-129`
- `Makefile:26-30`
- `tools/coveragegate/main.go:197-244`

**Evidence**:

- `internal/app/bootstrap/decimal_policy.go:42-43`
- `internal/app/bootstrap/bootstrap_internal_test.go:74-173`
- `dist/coverage/coverage.xml:2558-2575`
- `Makefile:26-30`
- `tools/coveragegate/main.go:197-244`

**Description**:

The current `make coverage` run fails the repository coverage gate with `statement coverage is 4487/4488`, `line coverage is 5908/5909`, and `branch coverage is 1375/1376`. The uncovered statement is the error return from `supportmath.SetActiveDecimalPolicy(policy)` in `ConfigureProcessDecimalPolicy`. Existing bootstrap tests cover the unset environment path, a valid override, and invalid environment parsing, but they do not exercise the `SetActiveDecimalPolicy` failure path. Because this file is in the production coverage package set, the feature cannot demonstrate the required 100% coverage gate in the current checkout.

## Notes

- Existing `test-coverage-drift-report.md` was not present, so no prior `COV-DRIFT-###` identifiers were available to preserve.
- `make test` passed locally on 2026-06-14.
- `make coverage` failed locally on 2026-06-14 with the metrics recorded in `COV-DRIFT-001`.
- No `.github/workflows/*` files were present in this checkout, so CI workflow coverage evidence was not available for this report.
