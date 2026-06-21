# Tasks: Report Base Currency Conversion

**Input**: Design documents from `/specs/007-currency-conversion-strategy/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature because the specification requires project-owned contract, integration, unit, external integration, redaction, performance, and regression coverage. Write the listed test tasks first and verify they fail before implementation tasks make them pass.

**Organization**: Tasks are grouped by user story so each story can be implemented and tested as an independently reviewable increment after the shared foundation is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it targets a different file and has no dependency on another incomplete task in the same phase.
- **[Story]**: User story label for traceability. Setup, foundational, and polish tasks do not use story labels.
- Every task includes an exact repository path.

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish feature-specific package and fixture locations without changing behavior.

- [ ] T001 Create currency integration package documentation in `internal/integration/currency/doc.go`
- [ ] T002 [P] Add deterministic provider fixture builder skeleton in `tests/testutil/currency_provider_fixtures.go`
- [ ] T003 [P] Add opt-in external integration guard helper in `tests/externalintegration/helpers_test.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Define shared models, service contracts, and runtime seams required before user-story work.

**CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T004 Add `ReportBaseCurrency` enum and validation in `internal/report/model/report_base_currency.go`
- [ ] T005 Extend `ReportRequest` with required report base currency in `internal/report/model/report_request.go`
- [ ] T006 Extend calculated report model fields for conversion audit data in `internal/report/model/report.go`
- [ ] T007 [P] Add conversion audit models and validators in `internal/report/model/conversion_audit.go`
- [ ] T008 [P] Add canonical rate evidence types in `internal/integration/currency/rate_evidence.go`
- [ ] T009 [P] Add lookup request and public rate service contracts in `internal/integration/currency/service.go`
- [ ] T010 [P] Add in-memory TUI-session rate cache in `internal/integration/currency/session_cache.go`
- [ ] T011 Add report calculation rate-service seam in `internal/report/calculate/currency_rate_service.go`
- [ ] T012 Prepare runtime report service dependency wiring in `internal/app/runtime/report_service.go` and `internal/app/runtime/runtime.go`

**Checkpoint**: Foundation ready. User-story implementation can now begin.

---

## Phase 3: User Story 1 - Select A Report Base Currency (Priority: P1) MVP

**Goal**: The user must choose USD or EUR for each report run, and report calculations/totals use that selected base currency.

**Independent Test**: Generate reports from a synced mixed-currency dataset once with USD and once with EUR, then verify each report request, calculation currency, and rendered output uses the selected base currency.

### Tests for User Story 1 (MANDATORY)

- [ ] T013 [P] [US1] Add contract tests for required USD/EUR report base-currency choices in `tests/contract/report_generation_workflow_contract_test.go`
- [ ] T014 [P] [US1] Add report screen render tests for base-currency menu, busy state, and result labels in `internal/tui/screen/report_screen_internal_test.go`
- [ ] T015 [P] [US1] Add flow tests for focus movement and disabled generation before base-currency selection in `internal/tui/flow/model_internal_test.go`
- [ ] T016 [P] [US1] Add integration test proving USD and EUR report requests reach runtime generation in `tests/integration/report_generation_flow_test.go`
- [ ] T017 [P] [US1] Add request validation tests for missing and invalid report base currency in `internal/report/model/report_internal_test.go`
- [ ] T018 [P] [US1] Add Markdown contract test that selected base currency replaces `NOT APPLICABLE` in `tests/contract/markdown_report_contract_test.go`

### Implementation for User Story 1

- [ ] T019 [US1] Add report base-currency indexes and selected value state in `internal/tui/flow/state.go`
- [ ] T020 [US1] Render the USD/EUR base-currency menu in `internal/tui/screen/report_screen.go`
- [ ] T021 [US1] Pass base-currency selection parameters from flow to screens in `internal/tui/flow/view.go`
- [ ] T022 [US1] Add base-currency focus navigation and selection handling in `internal/tui/flow/report_flow.go`
- [ ] T023 [US1] Disable `Generate Report` until year, method, and base currency are selected in `internal/tui/flow/menu_items.go`
- [ ] T024 [US1] Build validated report requests with selected report base currency in `internal/tui/flow/report_flow.go`
- [ ] T025 [US1] Use request base currency as the report calculation currency in `internal/report/calculate/calculator.go`
- [ ] T026 [US1] Implement same-currency bypass and cross-currency conversion boundary using the rate-service seam in `internal/report/calculate/currency_conversion.go`
- [ ] T027 [US1] Propagate selected report currency into summary, detail, and liquidation artifacts in `internal/report/calculate/artifacts.go`

**Checkpoint**: User Story 1 is functional with a deterministic test rate service and no official-provider dependency.

---

## Phase 4: User Story 2 - Use Official Historical Conversion Rates (Priority: P1)

**Goal**: Converted report amounts use official ECB or Federal Reserve historical rate evidence and disclose enough metadata to audit each conversion.

**Independent Test**: Use deterministic ECB and Federal Reserve fixtures with expected rate dates, quote directions, rate values, and converted values, then verify calculated amounts and Markdown audit output.

### Tests for User Story 2 (MANDATORY)

- [ ] T028 [P] [US2] Add official rate-provider contract tests with deterministic fixtures in `tests/contract/rate_provider_integration_contract_test.go`
- [ ] T029 [P] [US2] Add ECB EXR client and mapper unit tests in `internal/integration/currency/ecb_client_internal_test.go`
- [ ] T030 [P] [US2] Add Federal Reserve H.10 client and mapper unit tests in `internal/integration/currency/federal_reserve_client_internal_test.go`
- [ ] T031 [P] [US2] Add exact conversion math and session-cache unit tests in `internal/integration/currency/conversion_internal_test.go`
- [ ] T032 [P] [US2] Add deterministic mixed-currency conversion integration test in `tests/integration/report_generation_flow_test.go`
- [ ] T033 [P] [US2] Add Markdown audit and rate source summary contract tests in `tests/contract/markdown_report_contract_test.go`

### Implementation for User Story 2

- [ ] T034 [US2] Implement exact source-to-base conversion formulas in `internal/integration/currency/conversion.go`
- [ ] T035 [US2] Implement provider registry and public lookup service in `internal/integration/currency/service.go`
- [ ] T036 [US2] Implement ECB EXR HTTP client in `internal/integration/currency/ecb_client.go`
- [ ] T037 [US2] Implement ECB EXR response canonicalization in `internal/integration/currency/ecb_mapper.go`
- [ ] T038 [US2] Implement Federal Reserve H.10 HTTP client in `internal/integration/currency/federal_reserve_client.go`
- [ ] T039 [US2] Implement Federal Reserve quote-direction canonicalization in `internal/integration/currency/federal_reserve_mapper.go`
- [ ] T040 [US2] Apply session cache lookups and writes around provider requests in `internal/integration/currency/session_cache.go`
- [ ] T041 [US2] Resolve provider evidence per unique rate key before asset replay in `internal/report/calculate/calculator.go`
- [ ] T042 [US2] Record conversion audit entries during report artifact creation in `internal/report/calculate/artifacts.go`
- [ ] T043 [US2] Add conversion audit and rate source validation to report construction in `internal/report/model/capital_gains_report.go`
- [ ] T044 [US2] Render rate source summary in `internal/report/markdown/renderer_currency.go`
- [ ] T045 [US2] Render converted activity audit table in `internal/report/markdown/renderer_currency.go`
- [ ] T046 [US2] Wire concrete currency rate service into runtime report generation in `internal/app/runtime/report_service.go`
- [ ] T047 [US2] Add opt-in live ECB and Federal Reserve client checks in `tests/externalintegration/currency_provider_live_test.go`

**Checkpoint**: User Stories 1 and 2 both work with official-provider test doubles and auditable Markdown output.

---

## Phase 5: User Story 3 - Fail Safely When Conversion Is Not Defensible (Priority: P2)

**Goal**: Unsupported currencies, missing rates, malformed evidence, provider outage without cache evidence, and unsafe diagnostics fail before final report save.

**Independent Test**: Run failure fixtures for unsupported source currency, unavailable historical rate, provider outage, malformed currency, and a late conversion failure, then verify no final report file is produced and the user sees a non-secret actionable message.

### Tests for User Story 3 (MANDATORY)

- [ ] T048 [P] [US3] Add conversion failure matrix integration tests in `tests/integration/report_failure_flow_test.go`
- [ ] T049 [P] [US3] Add production diagnostic redaction tests for conversion failures in `tests/integration/diagnostic_redaction_test.go`
- [ ] T050 [P] [US3] Add provider failure classification unit tests in `internal/integration/currency/errors_internal_test.go`
- [ ] T051 [P] [US3] Add zero-priced holding reduction no-lookup unit tests in `internal/report/calculate/calculator_internal_test.go`

### Implementation for User Story 3

- [ ] T052 [US3] Implement conversion failure errors and safe message shaping in `internal/integration/currency/errors.go`
- [ ] T053 [US3] Reject malformed, missing, unsupported, and mismatched rate evidence in `internal/integration/currency/service.go`
- [ ] T054 [US3] Map conversion failures into report calculation errors and diagnostic context in `internal/report/calculate/errors.go`
- [ ] T055 [US3] Keep zero-priced no-cost holding reductions out of rate lookup in `internal/report/calculate/currency_conversion.go`
- [ ] T056 [US3] Include source currency, report base currency, and activity date in runtime failure copy in `internal/app/runtime/report_service.go`
- [ ] T057 [US3] Show selected report base currency on report failure result screens in `internal/tui/screen/report_screen.go`

**Checkpoint**: All user stories are independently functional, and unsafe conversions fail before report save.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation alignment, and coverage-gate checks.

- [ ] T058 [P] Update automated and manual validation notes after implementation in `specs/007-currency-conversion-strategy/quickstart.md`
- [ ] T059 Run full Go coverage test command and create `coverage.cov`
- [ ] T060 Run coverage XML export and create `coverage.xml`
- [ ] T061 Verify empirical dataset files remain unchanged under `testdata/empirical/`
- [ ] T062 Run opt-in external integration tests only when explicitly enabled and verify default skip behavior in `tests/externalintegration/currency_provider_live_test.go`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies. Can start immediately.
- **Foundational (Phase 2)**: Depends on Setup. Blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational. MVP scope.
- **User Story 2 (Phase 4)**: Provider client work can start after Foundational. Full report integration depends on User Story 1 conversion boundary and request model.
- **User Story 3 (Phase 5)**: Depends on User Story 1 and User Story 2 error surfaces.
- **Polish (Phase 6)**: Depends on all targeted user stories.

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational. No dependency on official live provider clients when tested with deterministic rate-service seams.
- **US2 (P1)**: Starts after Foundational for provider-layer work. Complete end-to-end behavior depends on US1 request and conversion boundary tasks.
- **US3 (P2)**: Starts after US1 and US2 establish conversion lookup, provider errors, and runtime failure handling.

### Within Each User Story

- Tests are written first and should fail before implementation.
- Models and service contracts precede orchestration changes.
- Provider DTO and mapping code stays inside `internal/integration/currency/`.
- Report calculation consumes canonical rate evidence, not provider DTOs.
- TUI captures only the selected base currency and displays non-secret outcomes.

### Parallel Opportunities

- T002 and T003 can run in parallel after T001 is not required.
- T007, T008, T009, and T010 can run in parallel after T004 through T006 decisions are understood.
- US1 test tasks T013 through T018 can run in parallel.
- US2 provider tests T028 through T031 can run in parallel with Markdown and integration test work T032 and T033.
- US2 provider clients T036 and T038 can run in parallel, as can provider mappers T037 and T039 after their client tests exist.
- US3 test tasks T048 through T051 can run in parallel.
- Polish documentation and external skip validation T058 and T062 can run in parallel after implementation is complete.

---

## Parallel Example: User Story 1

```bash
Task: "T013 Add contract tests for required USD/EUR report base-currency choices in tests/contract/report_generation_workflow_contract_test.go"
Task: "T014 Add report screen render tests for base-currency menu, busy state, and result labels in internal/tui/screen/report_screen_internal_test.go"
Task: "T015 Add flow tests for focus movement and disabled generation before base-currency selection in internal/tui/flow/model_internal_test.go"
Task: "T017 Add request validation tests for missing and invalid report base currency in internal/report/model/report_internal_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "T029 Add ECB EXR client and mapper unit tests in internal/integration/currency/ecb_client_internal_test.go"
Task: "T030 Add Federal Reserve H.10 client and mapper unit tests in internal/integration/currency/federal_reserve_client_internal_test.go"
Task: "T036 Implement ECB EXR HTTP client in internal/integration/currency/ecb_client.go"
Task: "T038 Implement Federal Reserve H.10 HTTP client in internal/integration/currency/federal_reserve_client.go"
```

## Parallel Example: User Story 3

```bash
Task: "T048 Add conversion failure matrix integration tests in tests/integration/report_failure_flow_test.go"
Task: "T049 Add production diagnostic redaction tests for conversion failures in tests/integration/diagnostic_redaction_test.go"
Task: "T050 Add provider failure classification unit tests in internal/integration/currency/errors_internal_test.go"
Task: "T051 Add zero-priced holding reduction no-lookup unit tests in internal/report/calculate/calculator_internal_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundation.
3. Complete Phase 3 User Story 1 with deterministic rate-service seams.
4. Validate US1 independently with T013 through T018 and targeted report generation tests.
5. Stop before official-provider work if an MVP review is needed.

### Incremental Delivery

1. Complete Setup and Foundational phases.
2. Deliver US1 for explicit base-currency selection and conversion boundary.
3. Deliver US2 for official provider evidence, caching, and audit rendering.
4. Deliver US3 for failure safety, redaction, and no-partial-save behavior.
5. Complete Polish validation and coverage gates.

### Parallel Team Strategy

1. One developer owns TUI and report request changes in US1.
2. One developer owns provider integration and canonical evidence in US2.
3. One developer owns failure classification, diagnostics, and redaction in US3 after US2 errors are shaped.
4. Test authors can start contract and unit tests from the beginning of each story phase.

---

## Notes

- Treat `testdata/empirical/` as read-only for this feature.
- Do not persist exchange-rate evidence to snapshots, setup files, temp files, or app-data caches.
- Do not add third-party dependencies for provider integration unless the implementation plan is updated first.
- Keep provider DTOs and HTTP details inside `internal/integration/currency/`.
- Keep financial arithmetic exact-decimal only and preserve provider-published rate precision in audit evidence.
