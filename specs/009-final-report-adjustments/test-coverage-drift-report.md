# Test Coverage Drift Report: Final Report Adjustments

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-07-18
**Updated**: 2026-07-19
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-control.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence for newly observed findings reflects the current implementation tree. Evidence retained with historical findings remains a point-in-time record from the review that captured it.

## Coverage Baseline

- `.specify/memory/constitution.md:149-169` requires 100% project-owned line and branch coverage where distinguished, black-box contract and integration execution to contribute to production-package coverage, integration tests as the default for user journeys, justified unit tests only, removal of substantially duplicated unit tests, and passing required tests and regressions.
- `.specify/memory/constitution.md:267-286` requires integration testing and coverage verification in feature tasks and makes missing coverage measurement a prerequisite rather than an accepted limitation.
- `AGENTS.md:58-67` requires `make test` and `make coverage` for full validation and isolates resource-sensitive performance evidence from deterministic tests and canonical coverage.
- `AGENTS.md:129-160` assigns package/unit, contract, integration, empirical, and performance suite responsibilities; defines the `cmd/` and `internal/` production denominator; requires 100% statement, global line, global branch, per-file line, and per-file branch coverage; and prohibits performance coverage artifacts from canonical coverage.
- `.specify/templates/tasks-template.md:12-22` requires automated tests, integration coverage, coverage verification, and export-boundary evidence for project-owned feature code.
- `spec.md:212-240` requires the complete semantic financial matrix, affected behavior and decision outcomes, contract and integration evidence, empirical regressions, and performance isolation.
- `spec.md:282-320` defines the closed `R`, `V`, `M`, `Q`, `B`, `Z`, `N`, `C`, `P`, and `E` populations and requires 100% evidence for SC-001 through SC-012, including deterministic ownership of the 10,000-activity document-content checks.
- `plan.md:53-64` and `plan.md:283-347` assign package, contract, integration, empirical, performance, failure-recovery, and generated-document evidence and require `make test`, `make coverage`, and isolated `make test-performance` validation.
- `contracts/report-rendering.md:355-431` requires semantic-field acceptance evidence, every FR-004a rejection boundary in both renderers, runtime failure behavior, confidentiality sentinels, deterministic scale-content checks, and performance-only timing evidence.
- `quickstart.md:101-143` defines the canonical 100% coverage command, cross-package contribution, performance exclusion, deterministic content ownership, and the absence of a performance coverage artifact.
- `Makefile:8-16,55-81`, `.cov.json:1-18`, `tools/coverpkg/main.go:28-49`, and `tools/coveragegate/main.go:197-247` implement the production-package denominator and canonical statement, line, branch, and per-file gates used by this review.
- `.github/workflows/test.yml:18-35` and `.github/workflows/test-suite.yml:23-49` run the independent `test / run`, `coverage / run`, and `test-performance / run` checks on pull requests.

## Findings

### COV-DRIFT-001: Acceptance Accounting Credits Unverified Semantic Occurrences

**Status**: Resolved
**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:149-169`
- `spec.md:212-219,294-317`
- `contracts/report-rendering.md:365-399`

**Evidence**:

- `tests/testutil/report_presentation_financial.go:15-114`
- `tests/testutil/report_presentation_occurrence.go:3-51`
- `tests/testutil/report_presentation_scalar.go:24-45`
- `tests/contract/report_rendering_acceptance_test.go:157-175`
- `tests/contract/report_rendering_acceptance_test.go:318-390`
- `tests/contract/report_rendering_acceptance_test.go:448-467`
- `tests/contract/report_rendering_acceptance_test.go:489-509`
- `tests/contract/report_rendering_acceptance_test.go:531-695`
- `tests/contract/report_rendering_acceptance_test.go:775-787`

**Description**:

The manifest declares distinct semantic occurrence keys, but a successful format attempt records every key for that case after only a document-wide expected-string check. The financial fixture declares opening, closing, and historical cost-basis occurrences, while the case control changes only opening and closing values. Nullable cases remove complete rows instead of proving blank values at present semantic fields, and exact-zero summary omission skips the generic visible-text assertion. Quantity cases change and count only `OpeningQuantity`, although `Q` is defined to include every opening, closing, historical, activity, post-activity, and disposed quantity. Rate cases assign the same value to all rate fields, and parity compares only the warning plus one expected value before crediting all `P` keys. The reported `V=664`, `Q=10`, and `P=491` completion therefore does not demonstrate the semantic-field coverage required by the feature.

**Remediation plan**:

Replace attempt-wide occurrence crediting with test-owned Markdown and PDF observations keyed by the full semantic occurrence identity. Extend the closed fixture only as needed to keep nullable rows present with blank fields, exercise historical positions, cover every required quantity and rate-metadata field, and distinguish omitted from visible values; then credit each `V`, `Q`, `B`, `Z`, `N`, `C`, `P`, and `E` item only after its own exact assertion and compare parity from the observed maps. Add negative harness cases proving a missing, misplaced, blank, or mismatched field cannot receive credit. Preserve warning, converted-entry, failed-attempt denominator, AUD-001 model-integrity, exact-value inclusion, and empirical-fixture behavior, and validate the focused testutil, contract, and integration owners before `make coverage`.

### COV-DRIFT-002: Deterministic Scale-Content Coverage Is Missing

**Status**: Resolved
**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:149-169`
- `AGENTS.md:129-160`
- `spec.md:318-320`
- `contracts/report-rendering.md:421-431`

**Evidence**:

- `tests/performance/helpers_test.go:44-79`
- `tests/performance/report_performance_flow_test.go:20-97`
- `tests/integration/report_converted_amounts_flow_test.go:93-187`
- `tests/integration/report_generation_flow_test.go:536-566`

**Description**:

The only Feature 009 fixture with exactly 10,000 activities is build-tagged performance support. Its performance test checks workload composition, timing, non-empty files, and opener invocation, but intentionally performs no document-content inspection. Deterministic integration coverage uses six-activity and 54-activity fixtures. No deterministic package, contract, or integration owner verifies that the named 10,000-activity report produces exactly 6,666 conversion rows with three entries each, controlled Markdown boundaries, a multi-page PDF Annex, repeated headers and continuation context, and no clipping or omission as required by SC-011. The required content evidence is absent rather than isolated from performance evidence.

**Remediation plan**:

Move the exact named 10,000-activity fixture into shared runtime-flow test support and reuse it from the isolated performance suite without restoring document assertions there. Add one deterministic runtime-backed integration owner that generates both formats and verifies all 6,666 source rows and three entries per row, controlled Markdown boundaries, multi-page PDF Annex headers and continuation context, searchable content within printable bounds, and no duplicates, clipping, or omission. Correct the stale performance work-unit wording while preserving the existing workload, request, timers, opener evidence, suite isolation, and absence of performance coverage artifacts. Validate focused PDF and converted-amount contracts, the new deterministic integration case, `make test`, `make coverage`, and separate `make test-performance`.

### COV-DRIFT-003: FR-004a Failure Boundaries Stop Before Runtime Output

**Status**: Resolved
**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:149-169`
- `plan.md:283-307`
- `contracts/report-rendering.md:373-376,410-415`

**Evidence**:

- `internal/report/presentation/financial_test.go:194-260`
- `tests/contract/report_rendering_values_contract_test.go:180-216`
- `internal/app/runtime/report_service_internal_test.go:440-459`
- `tests/integration/report_converted_amounts_flow_test.go:89-187`

**Description**:

Formatter tests cover adjusted exponents immediately below and above the accepted range, upper-bound carry, and checked precision limits. Generated-document coverage exercises only adjusted exponent `100001` through Markdown and PDF. The runtime service test injects a preconstructed generic render error, while the runtime-backed integration test exercises a concrete PDF layout failure. There is no selected-format runtime journey for the lower exponent boundary, upper-bound carry, or precision overflow proving renderer rejection, no saved path, no opener request, no alternate renderer, and a usable retry. The feature therefore cannot demonstrate its explicitly required end-to-end failure coverage for every FR-004a rejection class.

**Remediation plan**:

Add immutable, constructor-injected report-pipeline and renderer-scoped financial-formatting test options that retain the current concrete production defaults and make the precision-overflow path resource-safe without constructing a multi-gigabyte coefficient or using process-global fault mutation. Extend both-format generated-document and runtime-backed integration coverage for adjusted exponents `-100001` and `100001`, upper-bound carry to `100001`, and required precision above `2147383649`, invoking only the actual selected renderer. Each case must prove contextual redacted failure, no alternate renderer, writer, opener, document, bundle, path, or file, followed by a successful same-format retry through the same service. Preserve calculation, protected snapshots, exact inclusion, output transactions, and TUI disclosure ownership, and validate the formatter, renderer, runtime, contract, and integration paths before canonical coverage.

### COV-DRIFT-004: Export Disclosure Is Not Covered by a Runtime-Backed Journey

**Status**: Resolved
**Severity**: Medium
**Diverges from**:

- `.specify/memory/constitution.md:153-166`
- `spec.md:320`
- `plan.md:312-314`
- `contracts/report-rendering.md:404-406`

**Evidence**:

- `internal/tui/screen/report_screen_internal_test.go:247-323`
- `tests/contract/report_generation_workflow_contract_test.go:152-261`
- `tests/integration/report_generation_flow_test.go:40-73`
- `tests/integration/report_generation_flow_test.go:187-237`

**Description**:

Package and contract tests construct successful outcomes or screen parameters directly and verify cleartext disclosure, all saved paths, and deletion guidance. The runtime-backed normal-success journey checks saved paths and request metadata but not the disclosure or deletion copy. The opener-warning journey checks only its operational warning and manual-open instruction. No integration journey proves that real generation, output bundling, and flow routing produce every SC-012 disclosure for normal and opener-warning outcomes, which diverges from the repository's integration-first user-journey baseline.

**Remediation plan**:

Keep production copy ownership unchanged and extend the existing runtime-backed normal-success and opener-warning journeys with shared flow assertions. Verify the TUI-owned cleartext disclosure and deletion guidance exactly once, every saved Markdown or PDF path exactly once, the expected two-file and one-file bundle shapes, and absence of prior paths and disclosure after result-flow exit. Preserve opener failure as success-with-warning, retained user-owned files, runtime operational-only messages, and non-retention. Validate the focused runtimeflow and integration journeys plus the existing component, screen, flow, and workflow-contract tests.

### COV-DRIFT-005: Concrete PDF Finalization Failure Lacks Integration Coverage

**Status**: Resolved
**Severity**: Medium
**Diverges from**:

- `.specify/memory/constitution.md:153-169`
- `plan.md:341-347`
- `contracts/report-rendering.md:334-339,410-411`

**Evidence**:

- `internal/report/pdf/renderer_05_byte_finalization_internal_test.go:34-71`
- `internal/app/runtime/report_service_internal_test.go:399-537`
- `tests/integration/report_converted_amounts_flow_test.go:89-187`

**Description**:

The PDF package injects a concrete finalization failure and proves that the renderer returns no partial payload. Runtime package tests separately inject a generic `renderBundle` error labeled as finalization and prove no writer or opener call plus retry. The integration suite covers the same runtime consequences only for a concrete PDF layout failure. No runtime-backed integration journey carries the actual PDF finalization seam through process survival, no output reservation, no opener request, redacted failure context, and successful retry. The behavior is split across isolated seams instead of satisfying the required integration-first recovery evidence.

**Remediation plan**:

Replace the process-global finalization fault hook with an immutable renderer-scoped byte-finalizer option whose production default remains `GetBytesPdfReturnErr`, and expose only the narrow default-preserving runtime pipeline injection needed by shared integration support. Add a PDF-only runtime-backed journey that fails at the concrete document finalization boundary with partial bytes and a synthetic secret-bearing cause, then proves partial-byte discard, process survival, redacted finalization context, no alternate renderer, writer, opener, reservation, output metadata, path, or file, and one successful retry through the same service. Preserve layout and font completion order, `errors.Is` behavior, local `0600` export behavior on retry, and TUI ownership of disclosure copy. Validate focused PDF and runtime tests, shared test support, and the new integration journey.

### COV-DRIFT-006: Markdown Unit Tests Substantially Duplicate Broader Coverage

**Status**: Resolved
**Severity**: Medium
**Diverges from**:

- `.specify/memory/constitution.md:162-166`
- `.specify/templates/tasks-template.md:12-16`

**Evidence**:

- `tests/unit/report_markdown_test.go:132-300`
- `tests/unit/report_markdown_test.go:631-650`
- `tests/contract/report_rendering_values_contract_test.go:86-132`
- `tests/contract/report_annex_contract_test.go:17-47`
- `tests/integration/report_value_presentation_flow_test.go:20-70`
- `tests/integration/report_value_presentation_flow_test.go:84-155`
- `tests/integration/report_audit_presentation_flow_test.go:151-177`

**Description**:

The Feature 009 black-box Markdown unit tests repeat warning occurrence and placement, financial formatting, nil handling, canonical quantities and rates, source immutability, and classified Annex currency behavior that are also asserted through contract and runtime-backed integration boundaries. These tests call the exported renderers rather than isolating a seam that broader tests cannot reach. This is substantial behavioral duplication under the constitution's unit-test rule. The current acceptance-accounting gap in COV-DRIFT-001 must not be hidden by treating duplicate unit coverage as a substitute for correct contract or integration evidence.

**Remediation plan**:

After COV-DRIFT-001 supplies passing per-occurrence contract evidence, remove only the cited Feature 009 black-box Markdown tests and helpers used exclusively by them. Retain inherited Markdown behavior, unique invalid-render failure coverage, package-local syntax and error seams, corrected generated-document contracts, and runtime-backed integration journeys. If canonical production coverage drops, add the missing execution to the broader owning contract or integration layer rather than restoring duplicate assertions or changing the coverage denominator. Validate focused unit, contract, and integration owners followed by `make coverage`.

### COV-DRIFT-007: Regression Population Is Miscounted and Re-Executed Across Suites

**Status**: Resolved
**Severity**: Medium
**Diverges from**:

- `.specify/memory/constitution.md:153-166`
- `AGENTS.md:129-143`
- `spec.md:282-290`

**Evidence**:

- `tests/contract/testdata/report_calculation_regression_baseline.txt:85-95`
- `tests/empirical/empirical_calculation_test.go:66-140`
- `tests/contract/report_calculation_regression_contract_test.go:78-106`
- `tests/contract/report_calculation_regression_contract_test.go:231-262`
- `tests/contract/report_calculation_regression_contract_test.go:291-336`
- `tests/contract/report_rendering_acceptance_test.go:47-62`
- `tests/contract/report_rendering_acceptance_test.go:846-864`
- `Makefile:29-44`

**Description**:

The frozen `R` baseline includes `TestEmpiricalCalculationFixtures` even though that test creates child subtests. The discovery code marks only the path before the final slash as a parent, so slash-bearing subtest names do not mark the top-level test as non-leaf. This contradicts the feature's explicit exclusion of parents containing child subtests. The contract suite also launches the basis, calculation, and empirical owner suites once to discover `R` and a second time from aggregate acceptance, while `make test` already runs their designated package and empirical targets. The result is an incorrect denominator plus recursive duplicate execution across suite ownership boundaries.

**Remediation plan**:

Correct the pinned baseline without rebasing by removing only the invalid `TestEmpiricalCalculationFixtures` parent, changing `R` from 102 to 101, and retaining all leaf identities, source fingerprints, artifact hashes, and the baseline commit. Add a baseline invariant rejecting any recorded parent identity that is a proper ancestor of another case, remove both nested `go test` executions and synthetic `baseline/NNN` acceptance crediting, and make the contract layer statically validate identities, fingerprints, and empirical artifact hashes while maintained package and empirical targets remain the sole execution owners. Preserve `Makefile`, coverage instrumentation, calculation expectations, and read-only empirical data. Validate the focused regression and acceptance contracts, direct owner packages, and aggregate `make test` and `make coverage` paths.

### COV-DRIFT-008: Successful-Document Financial Confidentiality Check Is Vacuous

**Status**: Resolved
**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:77-82,149-169`
- `spec.md:153-161,234-240`
- `contracts/report-rendering.md:311-324,416-417`

**Evidence**:

- `tests/contract/report_rendering_confidentiality_test.go:21-77`
- `tests/contract/report_rendering_confidentiality_test.go:80-163`
- `tests/contract/report_rendering_confidentiality_test.go:234-243`

**Description**:

The successful Markdown/PDF test injects credential and protected-payload sentinels into a note and injects an allowed numeric amount into contracted financial fields. Its shared absence helper also checks `syntheticFinancialSentinel`, but that sentinel is never supplied to the successful report; it appears only in error and diagnostic tests. Its absence from successful documents is therefore guaranteed regardless of renderer behavior and does not prove the required suppression of unrelated financial material from a successful export. A required security-sensitive coverage claim remains unexercised.

**Remediation plan**:

Make the successful-document check non-vacuous with a test-local synthetic probe in valid non-contracted financial provenance, such as `BasisMatch.AcquisitionSourceID`, and assert the probe exists in the pre-render model before proving it is absent from both Markdown documents and PDF searchable text. Keep the separate synthetic export amount present in contracted financial fields, and retain every credential, protected-payload, error, diagnostic, and fixture assertion. Change no production renderer or redaction behavior and introduce no real user data or reusable secret. Validate the focused confidentiality contract tests.

### COV-DRIFT-009: Stale Performance Coverage Profile Survives Canonical Validation

**Status**: Resolved
**Severity**: Low
**Diverges from**:

- `AGENTS.md:66,134,152,160`
- `quickstart.md:101-143`

**Evidence**:

- `dist/coverage/performance.out:1-5`
- `Makefile:49-60`

**Description**:

The workspace contains an atomic performance coverage profile even though repository policy states that no performance profile or artifact exists. `make test-performance` does not create a profile, and `make coverage` excludes performance tests, so canonical coverage is not contaminated. However, canonical cleanup removes only `coverage.out` and `coverage.xml`, allowing the forbidden stale `performance.out` artifact to survive a successful gate. Its timestamp predates Feature 009, so this finding records current validation and cleanup drift rather than claiming that the feature generated the file.

**Remediation plan**:

Extend the canonical `coverage` target's exact generated-artifact cleanup to remove only `dist/coverage/performance.out` in addition to the canonical outputs. Preserve permitted diagnostic leaf profiles, deterministic package instrumentation, independent CI jobs, and the rule that no performance coverage target, profile, merge, job, or context exists. Validate that `make coverage` removes the stale file and that a separate `make test-performance` run does not recreate it. Do not change `.cov.json`, coverage-gate tooling, workflows, or performance suite membership.

## Resolution Evidence

- **COV-DRIFT-001**: Acceptance accounting records exact Markdown and PDF semantic occurrences. Nullable blanks remain parity/applicability controls but are excluded from `V`, non-applicable liquidation absence cases are excluded, and PDF warning credit requires exactly one complete, fully bold, correctly placed run sequence. The focused testutil and contract owners pass with `A=146/146`, `W=292/292`, `V=664/664`, `M=292/292`, `Q=80/80`, `B=16/16`, `Z=2/2`, `N=4/4`, `C=16/16`, `P=596/596`, and `E=24/24`.
- **COV-DRIFT-002**: The exact 10,000-activity fixture is shared through `tests/testutil/runtimeflow`; deterministic scale-content integration verifies 6,666 conversion rows, three entries per row, Markdown boundaries, and PDF continuation content. `TestReportScaleContentFlow` and the isolated performance suite pass.
- **COV-DRIFT-003**: Renderer-scoped financial-formatting options exercise all FR-004a rejection classes through both selected renderers and runtime retry without unsafe allocations or output side effects. Formatter, renderer, contract, and integration tests pass.
- **COV-DRIFT-004**: Runtime-backed Markdown and PDF success journeys now verify exact-once cleartext disclosure, deletion guidance, saved paths, bundle shape, opener-warning retention, and post-exit transient-state clearing. The integration, TUI, and workflow owners pass.
- **COV-DRIFT-005**: PDF finalization uses an immutable renderer-scoped byte finalizer with the production error-returning default. The runtime-backed fault journey proves redacted failure, discarded partial bytes, no output boundary calls, process survival, and successful retry.
- **COV-DRIFT-006**: Duplicated Feature 009 black-box Markdown tests and exclusive helpers were removed while contract, package, and runtime-backed ownership remains green. `make coverage` passes the canonical 100% gates.
- **COV-DRIFT-007**: The frozen regression population excludes the invalid empirical parent, reports `R=101/101`, validates identities and artifacts statically, and leaves direct basis, calculation, and empirical packages as execution owners.
- **COV-DRIFT-008**: Successful-document confidentiality coverage now injects and verifies a synthetic non-contracted provenance sentinel before proving its absence from Markdown and PDF, while retaining the contracted export amount and all other redaction controls.
- **COV-DRIFT-009**: Canonical coverage cleanup removes `dist/coverage/performance.out`; repeated `make coverage` and separate `make test-performance` runs leave the artifact absent.

## Notes

- `make coverage` completed successfully on 2026-07-18. `dist/coverage/coverage.xml:1` reports global line coverage `8497/8497` and branch coverage `2148/2148`; the statement and per-file gates also exited successfully.
- The canonical command instruments all `cmd/` and `internal/` production packages and includes package/unit, contract, integration, and empirical execution under the same `-coverpkg` denominator. No production-package instrumentation drift was identified.
- `make test-performance` was not rerun. This review inspected its test ownership and artifact isolation only; timing evidence is not used as production coverage evidence.
- No prior `test-coverage-drift-report.md` existed, so all identifiers were assigned in this review and all findings remain `Pending`.
- No additional proprietary agent-instruction files were present in repository or feature scope beyond `AGENTS.md`.
