# Tasks: Report Base Currency Conversion

**Input**: Design documents from `/specs/007-currency-conversion-strategy/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature because the specification requires project-owned contract, integration, unit, external integration, redaction, performance, and regression coverage. Write the listed test tasks first and verify they fail before implementation tasks make them pass.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independently reviewable increment after the shared foundation is complete. The context-orchestration work-unit ledger below is the execution control plane for parent agents and clean-context subagents.

**Bugfix**: 2026-06-21 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-06-23 — BUG-002 Updated from bugfix patch

**Bugfix**: 2026-06-23 — BUG-003 Updated from bugfix patch

**Bugfix**: 2026-06-24 — BUG-004 Updated from bugfix patch

**Bugfix**: 2026-06-24 — BUG-005 Updated from bugfix patch

**Bugfix**: 2026-06-28 — BUG-006 Updated from bugfix patch

**Bugfix**: 2026-06-28 — BUG-007 Updated from bugfix patch

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it targets a different file and has no dependency on another incomplete task in the same phase.
- **[Story]**: User story label for traceability. Setup, foundational, and polish tasks do not use story labels.
- Every task includes an exact repository path.

## Context Orchestration Process

**Purpose**: Keep implementation context bounded by delegating atomic work units to clean-context subagents while a parent orchestrator preserves phase order, cross-unit consistency, and final verification.

### Parent Orchestrator Rules

- Execute work units in ledger order. Do not start a later phase until every required earlier-phase unit has been parent-verified.
- Treat individual task checkboxes in this file as authoritative. The ledger is complete only when every referenced task is checked after parent verification.
- Use a clean subagent session for each work unit. WU23 may use a subagent for command output triage, but the parent must rerun final validation commands before marking T059 through T062, T071, and T072 complete.
- Include all required handoff context in the subagent prompt. Do not rely on the subagent having prior conversation state.
- Keep subagents inside the listed scope. If a subagent needs to edit outside the work-unit paths, it must stop and report the required path and reason.
- Require fail-first behavior for test tasks. For test-only units, successful completion means the tests exist, were run, and fail only for the expected missing implementation.
- After each subagent returns, the parent must inspect the diff, run the targeted verification, check for unrelated changes, and fix or re-delegate any inconsistency before starting the next unit.
- Do not allow provider DTOs or network details to cross into report calculation, cost-basis, TUI, runtime diagnostics, or Markdown rendering except through canonical currency integration models.
- Do not add third-party dependencies, persisted exchange-rate caches, user-supplied provider URLs, floating-point financial logic, or changes under `testdata/empirical/`.

### Required Subagent Handoff Packet

Every handoff must include:

- Work unit ID, phase, task IDs, exact task descriptions, and exact paths from the ledger.
- The relevant spec sources: `spec.md`, `plan.md`, `data-model.md`, `research.md`, and the specific contract files named by the unit.
- The non-negotiable rules for the unit: exact decimals only, fixed official provider hosts, no persisted rate evidence, no unofficial fallback rates, fail before final save on non-defensible conversion, and production-safe diagnostics.
- Current implementation status from previously verified units, including any public types, interfaces, helpers, or fixtures that the subagent must reuse.
- Allowed edit paths and any explicitly forbidden paths.
- Tests to run before implementation for test tasks and after implementation for mixed or implementation units.
- Required final response: files changed, task IDs completed, tests run with results, expected failures if any, assumptions made, and any requested parent follow-up.

### Parent Verification Gate

After each work unit, the parent must:

- Inspect `git diff -- <unit paths>` plus any extra paths reported by the subagent.
- Confirm all edits are within the unit scope or are justified adjacent changes.
- Run the unit's targeted tests or the closest compiling package test if the unit only adds scaffolding.
- Re-read the relevant contract or data-model section when a boundary type, public service, report output, or diagnostic behavior changed.
- Confirm `testdata/empirical/` remains unchanged.
- Confirm no exchange-rate evidence is persisted outside the final Markdown report.
- Mark the referenced task checkboxes only after the unit passes the verification gate.

### Context Compaction Recovery

If the parent context is compacted or a new parent agent resumes work:

- Read this `Context Orchestration Process`, the `Work Unit Ledger`, and the task checklist before editing.
- Run `git status --short` and inspect any existing diffs before selecting a unit.
- Identify the first ledger unit with unchecked referenced tasks. Resume there unless the current diff shows an incomplete earlier unit.
- Reconstruct prior state from checked task IDs, current diffs, and targeted tests. Do not assume a previous subagent result is valid without parent verification.
- If a unit has partial work, finish and verify that unit before opening a new subagent for another unit.

### Work Unit Ledger

Process units in this order. Units marked as parallel candidates may run concurrently only when their prerequisites are verified and the parent can review and merge both results before the next dependent unit.

| Unit | Phase | Tasks | Atomic scope and touched paths | Prerequisites | Required handoff sources | Parent verification |
|------|-------|-------|--------------------------------|---------------|--------------------------|---------------------|
| WU01 | Phase 1 Setup | T001, T002, T003 | Create feature package docs and skeletal fixtures in `internal/integration/currency/doc.go`, `tests/testutil/currency_provider_fixtures.go`, and `tests/externalintegration/helpers_test.go`. | None. | `plan.md` summary and project structure, `contracts/rate-provider-integration.md` test double and external integration sections. | Compile the touched packages or closest packages and inspect helper APIs for future reuse. |
| WU02 | Phase 2 Foundation | T004, T005, T006, T007 | Define report-owned base-currency, request, report, and conversion audit model changes in `internal/report/model/`. | WU01. | `spec.md` FR-001 through FR-004 and FR-019 through FR-021, `data-model.md` ReportBaseCurrency, ReportRequest, ConversionAuditEntry, and CapitalGainsReport, `contracts/markdown-report.md`. | Run `go test ./internal/report/model` and inspect public model validation. |
| WU03 | Phase 2 Foundation | T008, T009, T010 | Define currency integration canonical evidence, lookup service contract, and in-memory session cache in `internal/integration/currency/`. | WU01. | `data-model.md` CurrencyRateService, RateLookupRequest, CurrencyRateSessionCache, and ExchangeRateEvidence, `contracts/rate-provider-integration.md`, `research.md` anticorruption and revision behavior. | Run `go test ./internal/integration/currency` and inspect for no persistence or provider DTO leakage. |
| WU04 | Phase 2 Foundation | T011, T012 | Add report calculation rate-service seam and runtime dependency wiring seams in `internal/report/calculate/currency_rate_service.go`, `internal/app/runtime/report_service.go`, and `internal/app/runtime/runtime.go`. | WU02 and WU03. | `plan.md` architecture and conversion boundary, `data-model.md` ReportRequest and CurrencyRateService. | Run `go test ./internal/report/calculate ./internal/app/runtime` or the closest compiling package set. |
| WU05 | Phase 3 US1 Tests | T014, T015, T017 | Add fail-first TUI screen, TUI flow, and report request validation tests in `internal/tui/screen/report_screen_internal_test.go`, `internal/tui/flow/model_internal_test.go`, and `internal/report/model/report_internal_test.go`. | WU04. | `contracts/tui-workflows.md`, `data-model.md` ReportBaseCurrency and ReportRequest, `spec.md` US1. | Run targeted package tests and confirm failures are only expected missing US1 behavior. |
| WU06 | Phase 3 US1 Tests | T013, T016, T018, T063 | Add fail-first workflow, integration, Markdown, and same-currency regression tests in `tests/contract/report_generation_workflow_contract_test.go`, `tests/integration/report_generation_flow_test.go`, and `tests/contract/markdown_report_contract_test.go`. | WU04. | `contracts/tui-workflows.md`, `contracts/markdown-report.md`, `quickstart.md` automated verification flow, `spec.md` Mixed-Currency Acceptance Matrix and Single-Currency Regression Suite. | Run targeted contract and integration tests and record expected failures. |
| WU07 | Phase 3 US1 Implementation | T019, T020, T021, T022, T023, T024 | Implement base-currency selection state, rendering, flow navigation, disabled generation, and request construction in `internal/tui/flow/` and `internal/tui/screen/report_screen.go`. | WU05 and WU06 fail-first tests verified. | `contracts/tui-workflows.md`, `data-model.md` ReportBaseCurrency state transitions, previously verified WU02 model API. | Run `go test ./internal/tui/flow ./internal/tui/screen` plus affected US1 contract tests. |
| WU08 | Phase 3 US1 Implementation | T025, T026, T027 | Use request base currency in calculation, implement same-currency bypass and conversion boundary through the rate-service seam, and propagate report currency into artifacts in `internal/report/calculate/`. | WU07. | `plan.md` Conversion Boundary And Rounding, `data-model.md` SelectedActivityMonetaryContext and ConvertedActivityAmount, WU04 rate-service seam. | Run `go test ./internal/report/calculate ./tests/integration` with the US1 targeted tests. |
| WU09 | Phase 4 US2 Tests | T028, T029, T030, T031, T068, T073 | Add fail-first provider contract, ECB, Federal Reserve, Federal Reserve DDP direct-download, conversion math, and session-cache revision tests in `tests/contract/rate_provider_integration_contract_test.go` and `internal/integration/currency/*_internal_test.go`. | WU08. | `contracts/rate-provider-integration.md`, `research.md` provider source decisions, supported currency coverage, quote direction, unavailable-date rule, and revision behavior. | Run `go test ./internal/integration/currency ./tests/contract` and confirm failures are limited to missing US2 provider behavior. |
| WU10 | Phase 4 US2 Tests | T032, T064, T065, T067, T069 | Add fail-first deterministic mixed-currency, source-calendar date, no-tier-mixing, zero-valued field, and 10,000-activity responsiveness tests in `tests/integration/report_generation_flow_test.go`, `tests/integration/report_generation_responsiveness_test.go`, and `internal/report/calculate/calculator_internal_test.go`. | WU08. | `spec.md` Deterministic Conversion Fixture and 10,000-Activity Responsiveness Fixture, `plan.md` Performance Validation, `data-model.md` SelectedActivityMonetaryContext and ConvertedActivityAmount. | Run targeted integration and calculation tests and record expected failures, including zero-to-zero audit suppression preconditions. |
| WU11 | Phase 4 US2 Tests | T033, T066, T076, T077, T078, T079, T082, T083, T093, T094, T097, T098, T099, T100 | Add fail-first Markdown audit compact-table, rate source summary cardinality, field-boundary, same-currency versus converted-row, grouped audit row cardinality, zero-to-zero suppression, audit-to-detail Source ID consistency, Asset Detail table header and liquidation-column omission, and renderer aggregation tests in `tests/contract/markdown_report_contract_test.go` and `internal/report/markdown/renderer_internal_test.go`. | WU08. | `contracts/markdown-report.md`, `data-model.md` ConversionAuditEntry and CapitalGainsReport, `spec.md` FR-020, FR-021, FR-023, FR-034, FR-035, SC-003, and SC-009. | Run targeted Markdown contract and renderer tests and record expected failures. |
| WU12 | Phase 4 US2 Implementation | T034, T035, T040 | Implement exact conversion formulas, provider registry and public lookup service, and session cache lookups/writes in `internal/integration/currency/conversion.go`, `service.go`, and `session_cache.go`. | WU09 fail-first tests verified. | `research.md` Conversion And Rounding and Anticorruption Layer, `contracts/rate-provider-integration.md` canonical lookup and cache contracts, WU03 contracts. | Run `go test ./internal/integration/currency` and inspect exact-decimal division/multiplication handling. |
| WU13 | Phase 4 US2 Implementation | T036, T037 | Implement ECB EXR HTTP client and response canonicalization in `internal/integration/currency/ecb_client.go` and `ecb_mapper.go`. | WU12. Parallel candidate with WU14. | `research.md` EUR Base Currency Source and EUR Source Coverage, `contracts/rate-provider-integration.md` ECB EXR Contract. | Run ECB-focused `go test ./internal/integration/currency` tests and inspect fixed HTTPS endpoint use. |
| WU14 | Phase 4 US2 Implementation | T038, T039, T074 | Implement Federal Reserve H.10 HTTP client, DDP direct-download URL construction, live package CSV mapping, and quote-direction canonicalization in `internal/integration/currency/federal_reserve_client.go` and `federal_reserve_mapper.go`. | WU12. Parallel candidate with WU13. | `research.md` USD Base Currency Source and USD Source Coverage, `contracts/rate-provider-integration.md` Federal Reserve H.10 Contract. | Run Federal-Reserve-focused `go test ./internal/integration/currency` tests and inspect DDP URL shape plus starred/unstarred quote handling. |
| WU15 | Phase 4 US2 Implementation | T041, T042, T085, T095 | Resolve provider evidence per unique rate key before asset replay, preserve conversion classification, and record grouped conversion audit entries in `internal/report/calculate/calculator.go` and `artifacts.go`. | WU13 and WU14 verified. | `plan.md` Conversion Boundary And Rounding and Performance Validation, `data-model.md` ConvertedActivityAmount and ConversionAuditEntry. | Run `go test ./internal/report/calculate ./tests/integration` with WU10 targeted tests. |
| WU16 | Phase 4 US2 Implementation | T043, T044, T045, T080, T081, T084, T096, T101 | Validate conversion audit/rate source report construction, retain provider evidence for validation, and render provider-level rate source summary plus grouped converted activity audit table, asset detail conversion labels, and BUG-007 Asset Detail column contracts in `internal/report/model/capital_gains_report.go`, `internal/report/model/conversion_audit.go`, `internal/report/markdown/renderer.go`, and `internal/report/markdown/renderer_details.go`. | WU15 and WU11 tests verified. | `contracts/markdown-report.md`, `data-model.md` ConversionAuditEntry and CapitalGainsReport. | Run `go test ./internal/report/model ./internal/report/markdown ./tests/contract`. |
| WU17 | Phase 4 US2 Implementation | T046, T047, T075 | Wire concrete currency rate service into runtime report generation, add opt-in live provider checks, and keep Federal Reserve expected observations aligned with current DDP package evidence in `internal/app/runtime/report_service.go` and `tests/externalintegration/currency_provider_live_test.go`. | WU16. | `contracts/rate-provider-integration.md` external integration contract, `quickstart.md` external integration verification flow, `plan.md` runtime architecture. | Run runtime and externalintegration tests with default skip behavior, then run WU10 responsiveness target. |
| WU18 | Phase 5 US3 Tests | T048, T049, T050, T051 | Add fail-first conversion failure matrix, production diagnostic redaction, provider failure classification, and zero-priced no-lookup tests in `tests/integration/report_failure_flow_test.go`, `tests/integration/diagnostic_redaction_test.go`, `internal/integration/currency/errors_internal_test.go`, and `internal/report/calculate/calculator_internal_test.go`. | WU17. | `spec.md` US3 and Conversion Failure Matrix, `plan.md` Failure Handling, `contracts/rate-provider-integration.md` Failure Contract, `contracts/tui-workflows.md` result screen failure rules. | Run targeted integration, currency, and calculation tests and record expected failures. |
| WU19 | Phase 5 US3 Implementation | T052, T053 | Implement conversion failure errors, safe message shaping, and malformed/missing/unsupported/mismatched evidence rejection in `internal/integration/currency/errors.go` and `service.go`. | WU18 fail-first tests verified. | `data-model.md` ConversionFailure, `contracts/rate-provider-integration.md` Failure Contract, `research.md` Failure Modes. | Run `go test ./internal/integration/currency` and inspect safe messages for no secrets or financial values. |
| WU20 | Phase 5 US3 Implementation | T054, T055 | Map conversion failures into report calculation errors and keep zero-priced no-cost holding reductions out of rate lookup in `internal/report/calculate/errors.go` and `currency_conversion.go`. | WU19. | `plan.md` Failure Handling and Conversion Boundary, `spec.md` FR-022, FR-027, and FR-028. | Run `go test ./internal/report/calculate` and WU18 calculation targets. |
| WU21 | Phase 5 US3 Implementation | T056, T057 | Include source currency, base currency, and activity date in runtime failure copy, and show selected base currency on failure result screens in `internal/app/runtime/report_service.go` and `internal/tui/screen/report_screen.go`. | WU20. | `contracts/tui-workflows.md` Report Result Screen, `spec.md` FR-027 through FR-029, `data-model.md` ConversionFailure. | Run `go test ./internal/app/runtime ./internal/tui/screen ./tests/integration` with WU18 failure and redaction targets. |
| WU22 | Phase 6 Polish | T058, T070 | Update validation notes, BUG-002 Federal Reserve DDP direct-download evidence, and fixed-host provider security review evidence in `specs/007-currency-conversion-strategy/quickstart.md`. | WU21. | `quickstart.md`, `plan.md` Security Review and Testing Strategy, `research.md` Security Review. | Re-read quickstart for automated/manual/external validation, Federal Reserve DDP direct-download evidence, and fixed-host/no-user-controlled-provider evidence. |
| WU23 | Phase 6 Polish | T059, T060, T061, T062, T071, T072 | Run final coverage, coverage XML export, empirical unchanged check, external integration default skip and explicit-enabled verification, stale root-level coverage artifact cleanup, and maintained coverage-gate verification using ~~`coverage.cov`, `coverage.xml`~~ `dist/coverage/coverage.out`, `dist/coverage/coverage.xml`, `testdata/empirical/`, and `tests/externalintegration/currency_provider_live_test.go`; root-level coverage artifacts are superseded by BUG-001. | WU22. | `quickstart.md` Automated Verification Flow and External Integration Verification Flow, `.cov.json` if coverage gate behavior is inspected. | Run final commands, inspect generated coverage artifacts under `dist/coverage`, verify default-skip and explicitly enabled external integration outcomes, verify root-level coverage artifacts are absent, verify `testdata/empirical/` has no diff, and fix any failed gate before completion. |

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish feature-specific package and fixture locations without changing behavior.

- [X] T001 Create currency integration package documentation in `internal/integration/currency/doc.go`
- [X] T002 [P] Add deterministic provider fixture builder skeleton in `tests/testutil/currency_provider_fixtures.go`
- [X] T003 [P] Add opt-in external integration guard helper in `tests/externalintegration/helpers_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define shared models, service contracts, and runtime seams required before user-story work.

**CRITICAL**: No user story work can begin until this phase is complete.

- [X] T004 Add `ReportBaseCurrency` enum and validation in `internal/report/model/report_base_currency.go`
- [X] T005 Extend `ReportRequest` with required report base currency in `internal/report/model/report_request.go`
- [X] T006 Extend calculated report model fields for conversion audit data in `internal/report/model/report.go`
- [X] T007 [P] Add conversion audit models and validators in `internal/report/model/conversion_audit.go`
- [X] T008 [P] Add canonical rate evidence types in `internal/integration/currency/rate_evidence.go`
- [X] T009 [P] Add lookup request and public rate service contracts in `internal/integration/currency/service.go`
- [X] T010 [P] Add in-memory TUI-session rate cache in `internal/integration/currency/session_cache.go`
- [X] T011 Add report calculation rate-service seam in `internal/report/calculate/currency_rate_service.go`
- [X] T012 Prepare runtime report service dependency wiring in `internal/app/runtime/report_service.go` and `internal/app/runtime/runtime.go`

**Checkpoint**: Foundation ready. User-story implementation can now begin.

---

## Phase 3: User Story 1 - Select A Report Base Currency (Priority: P1) MVP

**Goal**: The user must choose USD or EUR for each report run, and report calculations/totals use that selected base currency.

**Independent Test**: Generate reports from a synced mixed-currency dataset once with USD and once with EUR, then verify each report request, calculation currency, and rendered output uses the selected base currency.

### Tests for User Story 1 (MANDATORY)

- [X] T013 [P] [US1] Add contract tests for required USD/EUR report base-currency choices in `tests/contract/report_generation_workflow_contract_test.go`
- [X] T014 [P] [US1] Add report screen render tests for base-currency menu, busy state, and result labels in `internal/tui/screen/report_screen_internal_test.go`
- [X] T015 [P] [US1] Add flow tests for focus movement and disabled generation before base-currency selection in `internal/tui/flow/model_internal_test.go`
- [X] T016 [P] [US1] Add integration test proving USD and EUR report requests reach runtime generation in `tests/integration/report_generation_flow_test.go`
- [X] T017 [P] [US1] Add request validation tests for missing and invalid report base currency in `internal/report/model/report_internal_test.go`
- [X] T018 [P] [US1] Add Markdown contract test that selected base currency replaces `NOT APPLICABLE` in `tests/contract/markdown_report_contract_test.go`
- [X] T063 [P] [US1] Add single-currency regression tests proving same-currency report monetary results preserve prior no-conversion behavior in `tests/integration/report_generation_flow_test.go`

### Implementation for User Story 1

- [X] T019 [US1] Add report base-currency indexes and selected value state in `internal/tui/flow/state.go`
- [X] T020 [US1] Render the USD/EUR base-currency menu in `internal/tui/screen/report_screen.go`
- [X] T021 [US1] Pass base-currency selection parameters from flow to screens in `internal/tui/flow/view.go`
- [X] T022 [US1] Add base-currency focus navigation and selection handling in `internal/tui/flow/report_flow.go`
- [X] T023 [US1] Disable `Generate Report` until year, method, and base currency are selected in `internal/tui/flow/menu_items.go`
- [X] T024 [US1] Build validated report requests with selected report base currency in `internal/tui/flow/report_flow.go`
- [X] T025 [US1] Use request base currency as the report calculation currency in `internal/report/calculate/calculator.go`
- [X] T026 [US1] Implement same-currency bypass and cross-currency conversion boundary using the rate-service seam in `internal/report/calculate/currency_conversion.go`
- [X] T027 [US1] Propagate selected report currency into summary, detail, and liquidation artifacts in `internal/report/calculate/artifacts.go`

**Checkpoint**: User Story 1 is functional with a deterministic test rate service and no official-provider dependency.

---

## Phase 4: User Story 2 - Use Official Historical Conversion Rates (Priority: P1)

**Goal**: Converted report amounts use official ECB or Federal Reserve historical rate evidence and disclose enough metadata to audit each conversion.

**Independent Test**: Use deterministic ECB and Federal Reserve fixtures with expected rate dates, quote directions, rate values, and converted values, then verify calculated amounts and Markdown audit output.

### Tests for User Story 2 (MANDATORY)

- [X] T028 [P] [US2] Add official rate-provider contract tests with deterministic fixtures covering supported and unsupported source currencies in `tests/contract/rate_provider_integration_contract_test.go`
- [X] T029 [P] [US2] Add ECB EXR client and mapper unit tests in `internal/integration/currency/ecb_client_internal_test.go`
- [X] T030 [P] [US2] Add Federal Reserve H.10 client and mapper unit tests in `internal/integration/currency/federal_reserve_client_internal_test.go`
- [X] T031 [P] [US2] Add exact conversion math and session-cache unit tests in `internal/integration/currency/conversion_internal_test.go`
- [X] T032 [P] [US2] Add deterministic mixed-currency conversion integration test with at least 50 priced activities, 3 source currencies, 2 report years, ECB division, H.10 unstarred division, H.10 starred multiplication, and previous-available-rate fallback in `tests/integration/report_generation_flow_test.go`
- [X] T033 [P] [US2] ⚠️ Reopened ~~Add Markdown audit and rate source summary contract tests proving the Rate Source Summary renders once per selected base-currency provider and excludes rate-specific `Quote Direction` and `Rate Value` in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-003)~~ Add Markdown audit and rate source summary contract tests proving the Rate Source Summary renders once per selected base-currency provider, excludes rate-specific `Quote Direction` and `Rate Value`, the `Currency Conversion Audit` uses grouped per-source activity rows without `Rate Authority`, `Rate Kind`, or zero-to-zero amount slots, and BUG-007 Asset Detail currency-column contracts are covered in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-005; reopened — BUG-007)
- [X] T064 [P] [US2] Add offset-sensitive source-calendar date rate-selection tests where UTC date differs from the preserved activity offset date in `tests/integration/report_generation_flow_test.go`
- [X] T065 [P] [US2] Add single-activity monetary-context no-tier-mixing conversion tests in `internal/report/calculate/calculator_internal_test.go`
- [X] T066 [P] [US2] ⚠️ Reopened Add Markdown contract tests distinguishing same-currency rows from converted rows, including cross-section checks that audited `Source ID` values are not labeled `same currency` in asset detail sections, exact `In-Year Activity` currency-column placement, and no `Activity Currency` in liquidation calculations, in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-006; reopened — BUG-007)
- [X] T067 [P] [US2] ⚠️ Reopened Add explicit zero-valued monetary field conversion tests proving valid zero fees and amounts remain zero and do not become report-visible zero-to-zero audit amount items in `internal/report/calculate/calculator_internal_test.go` (reopened — BUG-005)
- [X] T068 [P] [US2] Add session-cache revision behavior tests proving same-key cached evidence is reused within one process and new service instances fetch currently published provider values in `internal/integration/currency/session_cache_internal_test.go`
- [X] T069 [P] [US2] Add 10,000-activity responsiveness integration test with delayed provider fixtures, asynchronous busy-state assertion, and bounded lookup-count assertions in `tests/integration/report_generation_responsiveness_test.go`
- [X] T073 [P] [US2] Add Federal Reserve DDP direct-download URL and live `seriesrow` package CSV fixture regression tests in `internal/integration/currency/federal_reserve_client_internal_test.go`
- [X] T076 [P] [US2] ⚠️ Reopened ~~Add Markdown contract regression that keeps `Quote Direction` and `Rate Value` in `Currency Conversion Audit` or equivalent per-activity details while excluding them from `Rate Source Summary` in `tests/contract/markdown_report_contract_test.go`~~ Add Markdown contract regression that keeps `Quote Direction` and `Rate Value` in the compact `Currency Conversion Audit`, excludes `Rate Authority` and `Rate Kind` from audit columns, and keeps authority/rate-kind disclosure in `Rate Source Summary` in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-004)
- [X] T077 [P] [US2] Add renderer unit coverage for provider-level `Rate Source Summary` aggregation across multiple rate values in `internal/report/markdown/renderer_internal_test.go`
- [X] T078 [P] [US2] ⚠️ Reopened ~~Add Markdown contract assertion for exact `Currency Conversion Audit` compact header order in `tests/contract/markdown_report_contract_test.go`~~ Add Markdown contract assertion for the grouped `Currency Conversion Audit` header, grouped converted-amount field without one row per amount kind, and exact `Asset Detail` `In-Year Activity` header order in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-005; reopened — BUG-007)
- [X] T079 [P] [US2] ⚠️ Reopened ~~Add renderer unit assertion for compact audit row order and absence of `Rate Authority` and `Rate Kind` columns in `internal/report/markdown/renderer_internal_test.go`~~ Add renderer unit assertion for grouped audit row order, absence of `Rate Authority` and `Rate Kind` columns, no zero-to-zero amount items, and exact `Asset Detail` currency-column rendering in `internal/report/markdown/renderer_internal_test.go` (reopened — BUG-005; reopened — BUG-007)
- [X] T082 [P] [US2] Add Markdown contract regression proving a source activity with unit price, gross value, and zero fee renders one conversion audit row with non-zero amount kinds grouped and no zero-to-zero fee slot in `tests/contract/markdown_report_contract_test.go`
- [X] T083 [P] [US2] ⚠️ Reopened Add renderer unit regression for grouped converted amount display, zero-to-zero amount omission, audited converted `Source ID` values not rendering as `same currency` in asset detail sections, and BUG-007 Asset Detail table column clarity in `internal/report/markdown/renderer_internal_test.go` (reopened — BUG-006; reopened — BUG-007)
- [X] T093 [P] [US2] ⚠️ Reopened Add Markdown contract regression that cross-checks asset detail sections against `Currency Conversion Audit` by `Source ID` and verifies BUG-007 `Asset Detail` table currency-column contracts in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-007)
- [X] T094 [P] [US2] ⚠️ Reopened Add renderer unit regression that fails when a `Source ID` present in `Currency Conversion Audit` is rendered with a `same currency` label, when `In-Year Activity` omits `Original Activity Currency`, or when `Liquidation Calculations` repeats `Activity Currency` in `internal/report/markdown/renderer_internal_test.go` (reopened — BUG-007)
- [X] T097 [P] [US2] Add Markdown contract tests for exact `Asset Detail` `In-Year Activity` header order and `Original Activity Currency` naming in `tests/contract/markdown_report_contract_test.go`
- [X] T098 [P] [US2] Add renderer unit tests for exact `Asset Detail` `In-Year Activity` header order and currency/status column placement in `internal/report/markdown/renderer_internal_test.go`
- [X] T099 [P] [US2] Add Markdown contract tests proving `Asset Detail` `Liquidation Calculations` omits `Activity Currency` in `tests/contract/markdown_report_contract_test.go`
- [X] T100 [P] [US2] Add renderer unit tests proving `Asset Detail` `Liquidation Calculations` omits `Activity Currency` while `In-Year Activity` keeps `Original Activity Currency` in `internal/report/markdown/renderer_internal_test.go`

### Implementation for User Story 2

- [X] T034 [US2] Implement exact source-to-base conversion formulas in `internal/integration/currency/conversion.go`
- [X] T035 [US2] Implement provider registry and public lookup service in `internal/integration/currency/service.go`
- [X] T036 [US2] Implement ECB EXR HTTP client in `internal/integration/currency/ecb_client.go`
- [X] T037 [US2] Implement ECB EXR response canonicalization in `internal/integration/currency/ecb_mapper.go`
- [X] T038 [US2] ⚠️ Reopened Implement Federal Reserve H.10 HTTP client using the DDP direct `Output.aspx` CSV package endpoint in `internal/integration/currency/federal_reserve_client.go` (reopened — BUG-002)
- [X] T039 [US2] ⚠️ Reopened Implement Federal Reserve quote-direction canonicalization for the live DDP `seriesrow` package CSV layout in `internal/integration/currency/federal_reserve_mapper.go` (reopened — BUG-002)
- [X] T074 [US2] Update Federal Reserve DDP client URL construction and mapper parsing for `Output.aspx` H.10 package CSV metadata rows while preserving starred and unstarred quote-direction mapping in `internal/integration/currency/federal_reserve_client.go` and `internal/integration/currency/federal_reserve_mapper.go`
- [X] T040 [US2] Apply session cache lookups and writes around provider requests in `internal/integration/currency/session_cache.go`
- [X] T041 [US2] Resolve provider evidence per unique rate key before asset replay in `internal/report/calculate/calculator.go`
- [X] T042 [US2] Record conversion audit entries during report artifact creation in `internal/report/calculate/artifacts.go`
- [X] T043 [US2] Add conversion audit and rate source validation to report construction in `internal/report/model/capital_gains_report.go`
- [X] T044 [US2] ⚠️ Reopened ~~Render rate source summary in `internal/report/markdown/renderer_currency.go`~~ Render provider-level Rate Source Summary once per report from selected base-currency provider metadata in `internal/report/markdown/renderer.go` (reopened — BUG-003)
- [X] T045 [US2] ⚠️ Reopened ~~Render converted activity audit table in `internal/report/markdown/renderer_currency.go`~~ Render grouped converted activity audit table with one row or equivalent subsection per converted source activity, no `Rate Authority` or `Rate Kind` columns, no zero-to-zero amount items, asset detail converted labels aligned to audited `Source ID` values, and BUG-007 Asset Detail currency-column contracts in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_details.go` (reopened — BUG-005; reopened — BUG-006; reopened — BUG-007)
- [X] T080 [US2] Remove `Rate Authority` and `Rate Kind` from rendered `Currency Conversion Audit` columns while preserving retained rate evidence in `internal/report/markdown/renderer.go`
- [X] T081 [US2] Verify report model validation keeps provider authority and rate kind as retained evidence, not rendered audit-table column requirements, in `internal/report/model/capital_gains_report.go` and `internal/report/model/conversion_audit.go`
- [X] T084 [US2] ⚠️ Reopened Update report model validation to keep `ConversionAuditEntry` grouped per converted source activity, preserve the conversion status needed by asset detail rendering, and avoid requiring report-visible rows for zero-to-zero converted amount slots in `internal/report/model/capital_gains_report.go` and `internal/report/model/conversion_audit.go` (reopened — BUG-006)
- [X] T085 [US2] ⚠️ Reopened Verify report artifact creation records one `ConversionAuditEntry` per converted source activity with grouped `ConvertedActivityAmount` values and preserves converted versus same-currency status by `Source ID` in `internal/report/calculate/artifacts.go` (reopened — BUG-006)
- [X] T095 [US2] Preserve converted versus same-currency status from calculated artifacts into report detail artifacts in `internal/report/calculate/artifacts.go`
- [X] T096 [US2] ⚠️ Reopened Render asset detail conversion labels from preserved conversion status, render exact BUG-007 `In-Year Activity` column order with `Original Activity Currency`, and omit `Activity Currency` from `Liquidation Calculations` in `internal/report/markdown/renderer_details.go` (reopened — BUG-007)
- [X] T101 [US2] Update `Asset Detail` Markdown rendering so `In-Year Activity` uses the exact BUG-007 column order and `Liquidation Calculations` omits `Activity Currency` in `internal/report/markdown/renderer_details.go`
- [X] T046 [US2] Wire concrete currency rate service into runtime report generation in `internal/app/runtime/report_service.go`
- [X] T047 [US2] ⚠️ Reopened Add opt-in live ECB and Federal Reserve client checks against the official provider endpoints in `tests/externalintegration/currency_provider_live_test.go` (reopened — BUG-002)
- [X] T075 [US2] Update the opt-in Federal Reserve live expected EUR `2024-01-05` observation to current DDP package evidence in `tests/externalintegration/currency_provider_live_test.go`

**Checkpoint**: ~~User Stories 1 and 2 both work with official-provider test doubles and auditable Markdown output.~~ Reopened by BUG-002 until Federal Reserve DDP direct-download tasks pass; reopened by BUG-003 until provider-level Rate Source Summary remediation passes; reopened by BUG-004 until compact Currency Conversion Audit table remediation passes; reopened by BUG-005 until grouped Currency Conversion Audit rendering and zero-to-zero suppression pass; reopened by BUG-006 until asset detail converted labels are consistent with `Currency Conversion Audit` by `Source ID`; reopened by BUG-007 until Asset Detail table currency-column contracts pass.

---

## Phase 5: User Story 3 - Fail Safely When Conversion Is Not Defensible (Priority: P2)

**Goal**: Unsupported currencies, missing rates, malformed evidence, provider outage without cache evidence, and unsafe diagnostics fail before final report save.

**Independent Test**: Run failure fixtures for unsupported source currency, unavailable historical rate, provider outage, malformed currency, and a late conversion failure, then verify no final report file is produced and the user sees a non-secret actionable message.

### Tests for User Story 3 (MANDATORY)

- [X] T048 [P] [US3] Add conversion failure matrix integration tests in `tests/integration/report_failure_flow_test.go`
- [X] T049 [P] [US3] Add production diagnostic redaction tests for conversion failures in `tests/integration/diagnostic_redaction_test.go`
- [X] T050 [P] [US3] Add provider failure classification unit tests in `internal/integration/currency/errors_internal_test.go`
- [X] T051 [P] [US3] Add zero-priced holding reduction no-lookup unit tests in `internal/report/calculate/calculator_internal_test.go`

### Implementation for User Story 3

- [X] T052 [US3] Implement conversion failure errors and safe message shaping in `internal/integration/currency/errors.go`
- [X] T053 [US3] Reject malformed, missing, unsupported, and mismatched rate evidence in `internal/integration/currency/service.go`
- [X] T054 [US3] Map conversion failures into report calculation errors and diagnostic context in `internal/report/calculate/errors.go`
- [X] T055 [US3] Keep zero-priced no-cost holding reductions out of rate lookup in `internal/report/calculate/currency_conversion.go`
- [X] T056 [US3] Include source currency, report base currency, and activity date in runtime failure copy in `internal/app/runtime/report_service.go`
- [X] T057 [US3] Show selected report base currency on report failure result screens in `internal/tui/screen/report_screen.go`

**Checkpoint**: ~~All user stories are independently functional, and unsafe conversions fail before report save.~~ Reopened by BUG-002 until Federal Reserve DDP live-provider verification passes; User Story 3 safety behavior remains unchanged.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation alignment, and coverage-gate checks.

- [X] T058 [P] ⚠️ Reopened Update automated and manual validation notes after BUG-002 Federal Reserve DDP remediation in `specs/007-currency-conversion-strategy/quickstart.md` (reopened — BUG-002)
- [X] T059 ⚠️ Reopened ~~Run full Go coverage test command and create `coverage.cov`~~ Run full Go coverage test command with the maintained output path and create `dist/coverage/coverage.out` (reopened — BUG-001)
- [X] T060 ⚠️ Reopened ~~Run coverage XML export and create `coverage.xml`~~ Run coverage XML export from `dist/coverage/coverage.out` and create `dist/coverage/coverage.xml` (reopened — BUG-001)
- [X] T061 Verify empirical dataset files remain unchanged under `testdata/empirical/`
- [X] T062 ⚠️ Reopened Run opt-in external integration tests only when explicitly enabled, verify default skip behavior, and verify explicitly enabled ECB and Federal Reserve live checks pass in `tests/externalintegration/currency_provider_live_test.go` (reopened — BUG-002)
- [X] T070 Record fixed-host provider integration security review evidence, including OWASP Top 10 coverage and no user-controlled provider URL review, in `specs/007-currency-conversion-strategy/quickstart.md`
- [X] T071 Remove stale root-level generated coverage artifacts `coverage.cov` and `coverage.xml` if present in the repository root
- [X] T072 Run `make coverage` from the repository root and verify the coverage gate consumes `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`

---

## Phase 7: Coding Standards Drift Remediation

**Purpose**: Remediate coding-standards drift findings recorded in `coding-standards-drift-report.md` after all implementation and polish tasks are complete.

- [X] T086 Remediate `CODE-STAND-DRIFT-001` (High): Report Models Depend On Integration Provider Types by replacing integration-owned provider, authority, and quote-direction types in report-domain models with report-owned evidence concepts, then updating the integration-to-report mapping and Markdown rendering in `internal/report/model/conversion_audit.go`, `internal/report/calculate/currency_conversion.go`, `internal/report/calculate/artifacts.go`, and `internal/report/markdown/renderer.go`; source: `coding-standards-drift-report.md#code-stand-drift-001-report-models-depend-on-integration-provider-types`; evidence: `internal/report/model/conversion_audit.go:11-59`, `internal/report/model/conversion_audit.go:92-137`, `internal/report/model/conversion_audit.go:172-223`, `internal/report/markdown/renderer.go:10-14`, `internal/report/markdown/renderer.go:385-404`
- [X] T087 Remediate `CODE-STAND-DRIFT-002` (Medium): Currency Service File Mixes Contract, Orchestration, Transport, Classification, And Parsing by splitting `internal/integration/currency/service.go` so the public service contract remains in `service.go`, failure classification moves to `internal/integration/currency/service_failure.go`, fixed-provider HTTP fetching moves to `internal/integration/currency/provider_http.go`, and rate parsing moves to `internal/integration/currency/rate_parser.go`; source: `coding-standards-drift-report.md#code-stand-drift-002-currency-service-file-mixes-contract-orchestration-transport-classification-and-parsing`; evidence: `internal/integration/currency/service.go:35-85`, `internal/integration/currency/service.go:175-215`, `internal/integration/currency/service.go:239-300`, `internal/integration/currency/service.go:315-342`, `internal/integration/currency/service.go:385-429`
- [X] T088 Remediate `CODE-STAND-DRIFT-003` (Medium): Markdown Renderer Accumulates Multiple Rendering Responsibilities In One File by splitting `internal/report/markdown/renderer.go` into cohesive section files, keeping top-level orchestration in `renderer.go`, moving summary/reference rendering to `internal/report/markdown/renderer_summary.go`, detail/liquidation/activity rendering to `internal/report/markdown/renderer_details.go`, conversion audit and rate source summary rendering to `internal/report/markdown/renderer_conversion.go`, and formatting/sanitization helpers to `internal/report/markdown/renderer_format.go`; source: `coding-standards-drift-report.md#code-stand-drift-003-markdown-renderer-accumulates-multiple-rendering-responsibilities-in-one-file`; evidence: `internal/report/markdown/renderer.go:19-28`, `internal/report/markdown/renderer.go:76-107`, `internal/report/markdown/renderer.go:119-231`, `internal/report/markdown/renderer.go:233-369`, `internal/report/markdown/renderer.go:406-534`
- [X] T089 Remediate `CODE-STAND-DRIFT-004` (Medium): Production Functions Exceed Cognitive Complexity Threshold by decomposing `MapECBEXRCSVToEvidence` in `internal/integration/currency/ecb_mapper.go` and `ConversionAuditEntry.Validate` in `internal/report/model/conversion_audit.go` into cohesive helpers until production cognitive complexity is at or below the repository threshold; source: `coding-standards-drift-report.md#code-stand-drift-004-production-functions-exceed-cognitive-complexity-threshold`; evidence: `internal/integration/currency/ecb_mapper.go:19-64`, `internal/report/model/conversion_audit.go:175-223`
- [X] T090 Verify `CODE-STAND-DRIFT-004` remediation by running `go run github.com/uudashr/gocognit/cmd/gocognit@latest -test=false -over 15 internal` from the repository root and confirming no production functions remain above the threshold; source: `coding-standards-drift-report.md#code-stand-drift-004-production-functions-exceed-cognitive-complexity-threshold`
- [X] T091 Verify `CODE-STAND-DRIFT-001`, `CODE-STAND-DRIFT-002`, and `CODE-STAND-DRIFT-003` remediation by running `go test ./internal/report/model ./internal/report/calculate ./internal/report/markdown ./internal/integration/currency ./tests/contract` from the repository root after the refactors; source: `coding-standards-drift-report.md#findings`
- [X] T092 Verify `CODE-STAND-DRIFT-001`, `CODE-STAND-DRIFT-002`, `CODE-STAND-DRIFT-003`, and `CODE-STAND-DRIFT-004` remediation with the maintained coverage command `make coverage` from the repository root and confirm the coverage artifacts remain under `dist/coverage/`; source: `coding-standards-drift-report.md#findings`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. Can start immediately.
- **Foundational (Phase 2)**: Depends on Setup. Blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational. MVP scope.
- **User Story 2 (Phase 4)**: The ledger starts US2 after US1 to preserve phase order. Provider adapter work is technically independent after Foundational, but full report integration depends on User Story 1 conversion boundary and request model.
- **User Story 3 (Phase 5)**: Depends on User Story 1 and User Story 2 error surfaces.
- **Polish (Phase 6)**: Depends on all targeted user stories.
- **Coding Standards Drift Remediation (Phase 7)**: Depends on all implementation and polish tasks. Run only after every earlier task checkbox is checked and `coding-standards-drift-report.md` exists.

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational. No dependency on official live provider clients when tested with deterministic rate-service seams.
- **US2 (P1)**: Starts after US1 in this orchestration ledger. Complete end-to-end behavior depends on US1 request and conversion boundary tasks.
- **US3 (P2)**: Starts after US1 and US2 establish conversion lookup, provider errors, and runtime failure handling.

### Within Each User Story

- Tests are written first and should fail before implementation.
- Models and service contracts precede orchestration changes.
- Provider DTO and mapping code stays inside `internal/integration/currency/`.
- Report calculation consumes canonical rate evidence, not provider DTOs.
- TUI captures only the selected base currency and displays non-secret outcomes.

### Parallel Opportunities

- Work-unit ordering controls orchestration. Task-level `[P]` markers are local parallelism hints only when the parent can still verify the full work unit before any dependent unit starts.
- WU02 and WU03 can run in parallel after WU01 if the parent verifies both before WU04.
- WU05 and WU06 can run in parallel after WU04 because both are US1 fail-first test units.
- WU09, WU10, and WU11 can run in parallel after WU08 because they are US2 fail-first test units. The parent must verify all expected failures before WU12 starts.
- WU13 and WU14 can run in parallel after WU12 because ECB and Federal Reserve adapter files are separate. The parent must verify both before WU15 starts.
- WU22 is documentation-only after WU21. WU23 is the parent-controlled final validation gate and should not be bypassed by a subagent result alone.

---

## Subagent Handoff Example: US1 Tests

```bash
Subagent: "WU05 Phase 3 US1 Tests. Add fail-first tests for T014, T015, and T017 only. Use contracts/tui-workflows.md, data-model.md ReportBaseCurrency and ReportRequest, and spec.md US1. Edit only internal/tui/screen/report_screen_internal_test.go, internal/tui/flow/model_internal_test.go, and internal/report/model/report_internal_test.go unless you stop and report a needed extra path. Run targeted package tests and report expected failures."
```

## Subagent Handoff Example: US2 Providers

```bash
Subagent: "WU13 Phase 4 US2 Implementation. Implement only T036 and T037 for ECB EXR after WU12 is verified. Use research.md EUR sections and contracts/rate-provider-integration.md ECB EXR Contract. Edit only internal/integration/currency/ecb_client.go and internal/integration/currency/ecb_mapper.go unless you stop and report a needed extra path. Run ECB-focused internal/integration/currency tests."

Subagent: "WU14 Phase 4 US2 Implementation. Implement only T038 and T039 for Federal Reserve H.10 after WU12 is verified. Use research.md USD sections and contracts/rate-provider-integration.md Federal Reserve H.10 Contract. Edit only internal/integration/currency/federal_reserve_client.go and internal/integration/currency/federal_reserve_mapper.go unless you stop and report a needed extra path. Run Federal-Reserve-focused internal/integration/currency tests."
```

## Subagent Handoff Example: US3 Tests

```bash
Subagent: "WU18 Phase 5 US3 Tests. Add fail-first tests for T048, T049, T050, and T051 only. Use spec.md US3, plan.md Failure Handling, contracts/rate-provider-integration.md Failure Contract, and contracts/tui-workflows.md result screen rules. Run targeted integration, currency, and calculation tests and report expected failures."
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundation.
3. Complete Phase 3 User Story 1 with deterministic rate-service seams.
4. Validate US1 independently with T013 through T018, T063, and targeted report generation tests.
5. Stop before official-provider work if an MVP review is needed.

### Incremental Delivery

1. Complete Setup and Foundational phases.
2. Deliver US1 for explicit base-currency selection and conversion boundary.
3. Deliver US2 for official provider evidence, caching, and audit rendering.
4. Deliver US3 for failure safety, redaction, and no-partial-save behavior.
5. Complete Polish validation and coverage gates.

### Context-Orchestrated Subagent Strategy

1. The parent orchestrator owns phase order, handoff completeness, diff review, targeted verification, and final coverage gates.
2. Subagents own one ledger unit at a time with a clean context and the exact handoff packet required by that unit.
3. Test-only units must be completed and parent-verified as fail-first before implementation units for the same story begin.
4. Parallel subagents are allowed only for ledger-approved parallel candidates, and the parent must verify all parallel results before any dependent unit starts.

---

## Notes

- Treat `testdata/empirical/` as read-only for this feature.
- Do not persist exchange-rate evidence to snapshots, setup files, temp files, or app-data caches.
- Do not add third-party dependencies for provider integration unless the implementation plan is updated first.
- Keep provider DTOs and HTTP details inside `internal/integration/currency/`.
- Keep financial arithmetic exact-decimal only and preserve provider-published rate precision in audit evidence.
