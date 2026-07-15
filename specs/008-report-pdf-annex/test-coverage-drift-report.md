# Test Coverage Drift Report: Capital Gains Report PDF And Audit Annex

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-07-11
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-control.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Coverage Baseline

- `AGENTS.md:54-66` defines the Go test stack and requires `make test` and `make coverage` for relevant full-project validation. `AGENTS.md:126-144` assigns contract, integration, unit, empirical, coverage-tool, and maintained-expectation responsibilities.
- `specs/008-report-pdf-annex/plan.md:39-41` requires contract, integration, targeted unit, coverage, and unchanged empirical evidence. `specs/008-report-pdf-annex/plan.md:179-195` defines the feature-specific test structure, including real PDF layout assertions, same-input Markdown/PDF parity, render/write failure cleanup, the 10,000-activity path, and integration-first workflow coverage.
- `specs/008-report-pdf-annex/quickstart.md:14-47` requires `make test` and `make coverage` and describes their expected evidence. `specs/008-report-pdf-annex/quickstart.md:60-94` lists the required contract and integration scenarios, including all saved paths, generated-PDF layout and text behavior, continuation context, Markdown/PDF shared values, and both-format scale generation.
- `Makefile:8-11,36-40` is the maintained coverage command referenced by the policy. It instruments all production packages resolved under `cmd/...` and `internal/...` and runs package-local, contract, empirical, integration, and unit suites.
- `tools/coveragegate/main.go:197-245` is a derived baseline because no constitution file is present and `AGENTS.md` does not state percentages. It enforces exact 100% statement, global line, global branch, per-file line, and per-file branch coverage.
- `.cov.json:2-18` is a derived baseline for generated Cobertura scope. It excludes `tests` and `tools` from production coverage reporting.
- `.specify/memory/constitution.md`, `.specify/templates/tasks-template.md`, and the known proprietary instruction files listed by the command are not present. No clauses were derived from absent files.

## Findings

### COV-DRIFT-001: Required Large-History Test Is Skipped And Fails When Enabled

**Severity**: High
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:192-194`, which requires integration coverage for 10,000 cached activities in Markdown and PDF using deterministic currency-rate fixtures.
- `specs/008-report-pdf-annex/quickstart.md:62-94`, which requires both-format 10,000-activity generation as automated contract and integration evidence.

**Evidence**:

- `Makefile:36-40`
- `tests/integration/report_performance_flow_test.go:18-25`
- `tests/integration/report_performance_flow_test.go:54-80`
- `tests/integration/report_performance_flow_test.go:94-124`
- `tests/testutil/report_fixtures.go:545-586`
- `tests/integration/report_generation_responsiveness_test.go:17-58`

**Description**:

The maintained coverage command runs `tests/integration` without setting `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE`, so the only test that generates both Markdown and PDF from the 10,000-activity fixture is skipped. Running `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run '^TestReportPerformanceFlowLargeHistoryFixture$' -count=1 -v` on 2026-07-11 failed after 61.14 seconds because the test expects a raw `PAGE BREAK: Annex 1` marker that the PDF renderer does not emit. The fixture also makes every activity USD-denominated, while the separate 10,000-activity cross-currency test exercises only the default Markdown workflow. The required both-format scale and deterministic conversion path therefore has no passing evidence.

### COV-DRIFT-002: Generated PDF Contract And Markdown/PDF Data Parity Are Not Asserted

**Severity**: High
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:181-184`, which requires contract tests for landscape A4 text output and structured PDF presentation.
- `specs/008-report-pdf-annex/plan.md:189-190`, which requires integration generation from the same protected cache and comparison of shared report data.
- `specs/008-report-pdf-annex/quickstart.md:67-74,91-93`, which requires a combined PDF with text-based report content and identical shared Markdown/PDF values.

**Evidence**:

- `tests/contract/report_output_contract_test.go:124-143`
- `tests/contract/report_output_contract_test.go:201-218`
- `tests/integration/report_generation_flow_test.go:118-164`
- `tests/integration/report_generation_flow_test.go:182-220`
- `internal/report/pdf/renderer_internal_test.go:260-300`
- `internal/report/pdf/renderer_internal_test.go:39-57`

**Description**:

The PDF output contract writes a synthetic `%PDF-1.7` payload instead of invoking the renderer. The integration test generates both formats from the same fixture, but its PDF assertions only require the `%PDF-` prefix and absence of selected raw Markdown strings. The complete renderer test likewise checks only the PDF prefix, while landscape dimensions are asserted against private adapter fields. No contract or integration test extracts generated PDF text, verifies page dimensions from the output, or compares shared report values with the Markdown output. A generated PDF that omits or changes report data can therefore satisfy the current tests despite the integration-first parity requirement.

### COV-DRIFT-003: PDF Table Continuation And Wrapped-Cell Requirements Lack Regression Assertions

**Severity**: High
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:182-186`, which requires wrapped table content, repeated continuation context and headers, exact continuation labels, bottom-margin row preflight, and no label for unsplit tables.
- `specs/008-report-pdf-annex/quickstart.md:72-90`, which requires automated proof of wrapping, complete printable-area rows and borders, repeated context, and exact continuation-label behavior.

**Evidence**:

- `internal/report/pdf/renderer_internal_test.go:60-166`
- `internal/report/pdf/renderer_internal_test.go:670-711`
- `internal/report/pdf/gopdf_document.go:137-172`
- `internal/report/pdf/gopdf_document.go:228-285`

**Description**:

The principal continuation test renders a multi-page table but asserts only that the payload starts with `%PDF-`. It does not assert the exact `<context> (continued)` text, absence of the forbidden prefix, repeated table headers, page placement, or complete row and border placement. The only explicit long wrapped input is a paragraph sent through `MultiCell`; no table test provides a long cell and verifies wrapping within its column. Production contains continuation and chunking logic, but the required regression tests would not detect several specified continuation and table-cell layout regressions.

### COV-DRIFT-004: Successful Markdown Result Coverage Omits The Annex Path

**Severity**: Medium
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:181,190`, which requires result-path reporting and exactly two Markdown files.
- `specs/008-report-pdf-annex/quickstart.md:66-70`, which requires successful result screens to list every generated path.

**Evidence**:

- `tests/contract/report_generation_workflow_contract_test.go:86-118`
- `internal/tui/screen/screen_internal_test.go:286-315`
- `tests/integration/report_generation_flow_test.go:55-77`
- `tests/integration/helpers_test.go:342-357`
- `internal/tui/screen/report_screen.go:291-309`

**Description**:

The workflow contract verifies only a one-file PDF result. The package-local Markdown result test constructs a bundle containing only the main file, and the runtime-backed integration assertion checks only `Saved Markdown Path`; its file helper explicitly filters out Annex 1 files. Production has a distinct `Saved Annex 1 Markdown Path` branch, but no test asserts that label and path in a successful two-file result. This leaves the required all-generated-paths user journey below the specified contract and integration coverage level.

### COV-DRIFT-005: SC-001 Contract Test Bypasses The Bubble Tea Workflow

**Severity**: Medium
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:188`, which designates the output-selection-to-start contract as automated evidence for SC-001.
- `specs/008-report-pdf-annex/tasks.md:164`, which requires the workflow contract to cover selection, start timing, busy state, and result copy.

**Evidence**:

- `tests/contract/report_generation_workflow_contract_test.go:194-237`
- `internal/tui/flow/model_internal_test.go:1063-1072`
- `internal/tui/flow/report_flow.go:219-270`
- `tests/integration/report_generation_responsiveness_test.go:31-44`

**Description**:

The SC-001 contract test starts a timer, constructs a request directly, and calls the busy-screen renderer directly. It does not select the format through the root flow model, activate Generate, observe the returned asynchronous command, or prove that report generation remains outside the transition. Package-local and integration tests cover parts of the real asynchronous transition, so this is not an absence of all behavior evidence. It is a test-structure drift from the explicitly required contract-level workflow proof.

### COV-DRIFT-006: Renderer-Failure Cleanup Evidence Remains Unit-Only

**Severity**: Medium
**Diverges from**:

- `specs/008-report-pdf-annex/plan.md:191`, which requires integration tests for both render and write failures and partial-file cleanup.
- `specs/008-report-pdf-annex/tasks.md:166-168`, which assigns render and write failure cleanup to `tests/integration/report_failure_flow_test.go`.

**Evidence**:

- `tests/integration/report_failure_flow_test.go:449-470`
- `tests/integration/report_failure_flow_test.go:508-529`
- `tests/integration/report_failure_flow_test.go:535-581`
- `internal/app/runtime/report_service_internal_test.go:101-131`

**Description**:

The integration failure suite exercises write failures after file creation and bundle cleanup, but it does not inject a renderer failure and assert that the runtime-backed workflow creates no output files or opener request. Renderer failure behavior is covered only through an injected package-local runtime unit test. This leaves one explicitly required failure journey outside the repository's integration-first feature structure.

## Notes

- `make coverage` passed on 2026-07-11 after regenerating `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`.
- `dist/coverage/coverage.xml:1-4` reports 8,301 of 8,301 lines and 2,085 of 2,085 branches covered. The gate also completed without statement or per-file failures.
- All feature production packages under `internal/app/runtime`, `internal/report`, and `internal/tui` are included by the maintained `-coverpkg` production package set. No package-instrumentation drift was identified.
- The numeric 100% gate does not invalidate the findings above because they concern required scenario assertions and test-layer placement rather than executable statement reachability.
