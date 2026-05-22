---

description: "Task list for Generate Yearly Gains And Losses Report implementation"
---

# Tasks: Generate Yearly Gains And Losses Report

**Input**: Design documents from `/specs/005-generate-gains-report/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Bugfix**: 2026-05-22 — [BUG-001] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-002] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-003] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-004] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-005] Updated from bugfix patch

**Tests**: Automated tests are mandatory for this feature. The feature specification marks User Scenarios & Testing as mandatory and `plan.md` requires integration-first coverage, targeted unit tests for calculation and IO rules, `make test`, `make coverage`, and an opt-in large-history performance path.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Path Conventions

- Executable entrypoint: `cmd/ghostfolio-cryptogains/`
- App wiring and orchestration: `internal/app/`
- Report calculation, rendering, and output: `internal/report/`
- Protected snapshots and synced activity models: `internal/snapshot/` and `internal/sync/`
- Bubble Tea screens and flow: `internal/tui/`
- Shared precision and redaction helpers: `internal/support/`
- Automated tests: `tests/`
- Feature documents: `specs/005-generate-gains-report/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the report package skeleton and reusable fixture locations needed by later story work.

- [X] T001 [P] Create report package skeleton and package documentation in `internal/report/model/model.go`, `internal/report/basis/doc.go`, `internal/report/calculate/doc.go`, `internal/report/markdown/doc.go`, and `internal/report/output/doc.go`
- [X] T002 [P] Create reusable deterministic report ledger fixtures in `tests/testutil/report_fixtures.go`
- [X] T003 [P] Create reusable report filesystem and opener test helpers in `tests/testutil/report_io_fixtures.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish stored asset identity, readable cache summaries, report service boundaries, and exact calculation primitives required before story implementation.

**Critical**: Finish this phase before starting user story work.

- [X] T004 [P] ⚠️ Reopened Persist a stable Ghostfolio asset identity key from non-empty nested `SymbolProfile.id` into normalized activities and fail safe when required reporting rows lack that key in `internal/ghostfolio/dto/activity_page_response.go`, `internal/ghostfolio/mapper/activity_mapper.go`, and `internal/sync/model/activity_record.go` (reopened — BUG-002)
- [X] T005 ⚠️ Reopened Reevaluate snapshot compatibility for the asset identity model change against the corrected upstream `SymbolProfile.id` dependency, including whether `ActivityModelVersion`, refresh handling, and older-snapshot expectations need revision in `internal/snapshot/model/payload.go`, `tests/unit/stored_data_version_test.go`, and `tests/integration/snapshot_compatibility_flow_test.go` (reopened — BUG-002)
- [X] T067 [P] Add regression coverage that verifies upstream `Activity.SymbolProfile.id` maps into `AssetIdentityKey` end-to-end through DTO decoding and activity mapping in `internal/ghostfolio/mapper/activity_mapper_internal_test.go`
- [X] T068 Assess and refresh synced-activity fixtures, snapshot fixtures, and compatibility expectations that may still encode the superseded `symbolProfileId` assumption in `tests/testutil/testutil.go`, `tests/unit/stored_data_version_test.go`, and `tests/integration/snapshot_compatibility_flow_test.go`
- [X] T006 [P] Update shared synced-activity test fixtures to include non-display `AssetIdentityKey` values in `tests/testutil/testutil.go`
- [X] T007 Extend readable protected-data summaries with activity count, last successful sync timestamp, available report years, and unlocked cache access in `internal/app/runtime/sync_types.go`, `internal/app/runtime/active_snapshot_state.go`, and `internal/app/runtime/snapshot_lifecycle.go`
- [X] T008 [P] Define report runtime request, outcome, failure reason, and service interface types in `internal/app/runtime/report_types.go`
- [X] T009 [P] Define report request, document, output file, summary, reference, detail, activity row, and liquidation calculation models in `internal/report/model/report.go`
- [X] T010 [P] Define activity calculation input, selected currency context, and cost basis method enum skeleton in `internal/report/model/activity_input.go` and `internal/report/model/cost_basis_method.go`
- [X] T011 [P] Reuse `internal/support/decimal` for exact division and canonical formatting, then add only report-specific decimal helpers for multiplication, zero checks, and comparisons in `internal/report/calculate/decimal_math.go`
- [X] T012 Add `ReportService` dependency slots to runtime and TUI dependency assembly without enabling report generation yet in `internal/app/runtime/runtime.go` and `internal/tui/flow/model.go`

**Checkpoint**: Stored activity data has report-safe asset identity, and runtime/TUI code can receive report services without exposing report workflows.

---

## Phase 3: User Story 1 - Enter The Sync And Reports Context (Priority: P1) MVP

**Goal**: After setup is complete, the user can open `Sync and Reports`, unlock the token-scoped working context once, and choose between `Sync Data` and `Generate Capital Gains Report` while seeing synced-data readiness.

**Independent Test**: Start from the main menu with completed setup, open `Sync and Reports`, provide the token once, verify that the contextual menu shows `Sync Data`, `Generate Capital Gains Report`, and the correct synced-data readiness state, then enter `Sync Data` and verify that the screen explains token reuse without rendering token input.

### Tests for User Story 1

- [X] T013 [P] [US1] ⚠️ Reopened Add main-menu and Sync and Reports workflow contract coverage from `contracts/tui-workflows.md` in `tests/contract/main_menu_workflow_contract_test.go` and `tests/contract/sync_reports_workflow_contract_test.go` so the context opens only after selected-server snapshot unlock or Ghostfolio-authenticated new-context acceptance, a Ghostfolio-rejected token stays on the unlock screen with `access denied`, `Unlock` blocked, `Back` as the only available action, deferred token-field clearing after leaving and re-entering, the in-context `Sync Data` screen explains token reuse and does not show `Ghostfolio Security Token` input, and the unavailable `Generate Capital Gains Report` action remains visible but is skipped by keyboard focus in `Sync and Reports` (reopened — BUG-003, BUG-004, BUG-005)
- [X] T014 [P] [US1] ⚠️ Reopened Add integration coverage for token unlock, selected-server snapshot discovery, valid-new-context activation after selected-server snapshot miss, rejected-token refusal that keeps the unlock screen in a back-only `access denied` state, no-data readiness, existing-data readiness, last-sync timestamp display, token reuse after sync completion, context exit token clearing, and the rule that `Sync Data` inside the unlocked context cannot replace the stored runtime token through the screen itself, while Up and Down navigation skip the unavailable report action instead of landing on it, in `tests/integration/sync_reports_context_flow_test.go` (reopened — BUG-003, BUG-004, BUG-005)
- [X] T015 [P] [US1] Add screen rendering coverage for Sync and Reports unlock and context menu states in `internal/tui/screen/sync_reports_screen_internal_test.go`
- [X] T073 [P] [US1] Add unlock-screen rendering regression coverage for the rejected-token branch in `internal/tui/screen/sync_reports_screen_internal_test.go` so `access denied` is shown, `Unlock` is disabled, `Back` is the only available action, the rejected token field remains populated for that failed screen instance, and the field is cleared only after leaving and re-entering the unlock screen
- [X] T069 [P] [US1] Add screen rendering regression coverage for the in-context `Sync Data` screen so it hides any token label or input and shows only the existing-context-token explanation in `internal/tui/screen/sync_entry_screen_internal_test.go`
- [X] T071 [P] [US1] Add internal flow regression coverage that `Up` and `Down` skip disabled `Generate Capital Gains Report` actions in `Sync and Reports` when report generation is unavailable in `internal/tui/flow/model_internal_test.go`

### Implementation for User Story 1

- [X] T016 [US1] Replace the main menu business entry with `Sync and Reports` and remove pre-unlock protected-data status rendering in `internal/tui/screen/main_menu_screen.go` and `internal/tui/flow/model.go`
- [X] T017 [P] [US1] Add Sync and Reports unlock and context menu screen renderers in `internal/tui/screen/sync_reports_screen.go`
- [X] T070 [US1] Add a dedicated in-context `Sync Data` confirmation or explicit non-token mode so `internal/tui/screen/sync_entry_screen.go` does not render editable token input after context unlock
- [X] T018 [US1] Add active unlocked context state for runtime token, selected server, protected cache summary, report result scratch data, and no report history in `internal/tui/flow/model.go`
- [X] T019 [US1] ⚠️ Reopened Implement runtime selected-server protected snapshot unlock for context entry without forcing a sync in `internal/app/runtime/sync_service.go` and `internal/app/runtime/snapshot_lifecycle.go`, distinguishing `snapshot unlocked`, `authenticated new isolated local-user context`, and `rejected token` outcomes so selected-server snapshot miss alone never activates the context (reopened — BUG-005)
- [X] T020 [US1] ⚠️ Reopened Implement unlock screen key handling, masked token validation, selected-server unlock attempt, Ghostfolio-authenticated new-context activation, rejected-token `access denied` handling, blocked repeated `Unlock`, and token-field clearing only after leaving and re-entering the unlock screen in `internal/tui/flow/sync_flow.go` and `internal/tui/flow/model.go` (reopened — BUG-005)
- [X] T021 [US1] ⚠️ Reopened Route `Sync Data` from the unlocked context using the stored context token and return to Sync and Reports after success or failure in `internal/tui/flow/sync_flow.go`, `internal/tui/flow/model.go`, and `internal/tui/flow/navigation.go` without reusing the unlock token-entry UI or allowing token display, focus, or editing on the in-context sync screen (reopened — BUG-003)
- [X] T022 [US1] Route server-replacement cancellation and success back to the unlocked context without requiring another token prompt in `internal/tui/flow/sync_flow.go`, `internal/tui/flow/model.go`, and `internal/tui/flow/navigation.go`
- [X] T023 [US1] Render `Sync Data` last successful sync timestamp, `no synced data available`, and report-generation unavailable reasons in `internal/tui/screen/sync_reports_screen.go`
- [X] T072 [US1] Update `internal/tui/flow/navigation.go` so `updateSyncReportsMenu()` advances to the previous or next enabled menu item and never lands on disabled `Generate Capital Gains Report` rows when report generation is unavailable
- [X] T024 [US1] Clear runtime token and in-memory report scratch state when leaving Sync and Reports or quitting from the context in `internal/tui/flow/model.go` and `internal/tui/flow/navigation.go`

**Checkpoint**: User Story 1 is independently functional and testable without calculating or saving reports.

---

## Phase 4: User Story 2 - Generate A Yearly Gains And Losses Markdown Report (Priority: P1)

**Goal**: With synced data available in the active unlocked context, the user can choose a year and cost basis method, generate a yearly gains-and-losses report, save it to Documents, request OS opening, and return to Sync and Reports.

**Independent Test**: Using a deterministic multi-year synced dataset, select an available year and a supported cost basis method, generate the report, verify the output file contents and location, and confirm that the workflow returns to Sync and Reports without asking for the token again.

### Tests for User Story 2

- [X] T025 [P] [US2] ⚠️ Reopened Add Markdown document and output-file contract coverage from `contracts/markdown-report.md` in `tests/contract/markdown_report_contract_test.go` to distinguish preserved explicit zero-valued explained zero-priced holding-reduction fields from blank values (reopened — BUG-001)
- [X] T026 [P] [US2] Add report selection, busy, result, and failure workflow contract coverage from `contracts/tui-workflows.md` in `tests/contract/report_generation_workflow_contract_test.go`
- [X] T027 [P] [US2] Add integration coverage for deterministic multi-year report generation, available-year selection, Documents save, one opener request on success, opener failure warning, and return to unlocked context in `tests/integration/report_generation_flow_test.go`
- [X] T028 [P] [US2] Add integration coverage for empty-main-section reports with `NOT APPLICABLE` calculation currency, incomplete monetary context failure, Documents unavailable failure, partial-file cleanup, and app-managed storage leakage checks in `tests/integration/report_failure_flow_test.go`
- [X] T029 [P] [US2] ⚠️ Reopened Add unit coverage for selected-year cutoffs, first-acquisition exclusion, main-section inclusion, reference-only exclusion, same-source-calendar-date BUY-before-SELL reopening behavior, full-liquidation counts, zero-result included assets, negative losses, zero-priced holding reductions, and production-shaped explained zero-priced `SELL` rows with explicit zero `unit_price`, `gross_value`, and `fee_amount` in `tests/unit/report_calculation_test.go` (reopened — BUG-001)
- [X] T030 [P] [US2] Add unit coverage for Documents directory resolution, timestamped filename slugs, same-second suffixes, exclusive creation, write cleanup, and platform opener commands in `tests/unit/report_output_test.go`
- [X] T031 [P] [US2] ⚠️ Reopened Add unit coverage for Markdown header and section order, required tables, empty states, canonical exact decimal rendering, explicit report calculation currency label or `NOT APPLICABLE`, activity currency columns, secret exclusion, and explicit `0` rendering for preserved explained zero-priced holding-reduction fields in `tests/unit/report_markdown_test.go` (reopened — BUG-001)
- [X] T032 [P] [US2] ⚠️ Reopened Add unit coverage for single-activity currency context priority, explicit zero fee, missing fee, positive priced quantity, exact unit-price derivation, no cross-tier mixing, and explained zero-priced rows with preserved explicit zero-valued source fields in `tests/unit/report_activity_input_test.go` (reopened — BUG-001)
- [X] T065 [P] [US2] Add reusable report fixtures for production-shaped explained zero-priced `SELL` rows that preserve explicit zero `unit_price`, `gross_value`, and `fee_amount` in `tests/testutil/report_fixtures.go`

### Implementation for User Story 2

- [X] T033 [P] [US2] ⚠️ Reopened Implement single-activity currency context selection, selected-currency tracking, and calculation-input validation in `internal/report/calculate/activity_input.go` so explained zero-priced holding reductions can preserve explicit zero-valued source fields without becoming priced inputs or requiring activity currency context (reopened — BUG-001)
- [X] T034 [P] [US2] Implement report model constructors and validation helpers for request, report, summary, reference, detail, document, and output outcome structures in `internal/report/model/report.go`
- [X] T066 [P] [US2] Extend report activity-row models to distinguish missing values from preserved explicit zero-valued `unit_price`, `gross_value`, and `fee_amount` for explained zero-priced holding reductions in `internal/report/model/report.go`
- [X] T035 [P] [US2] Implement FIFO, LIFO, and HIFO lot basis state with exact arithmetic and deterministic lot ordering in `internal/report/basis/lot_methods.go`
- [X] T036 [P] [US2] Implement Average Cost Basis pool state and zero-quantity pool reset in `internal/report/basis/average_cost.go`
- [X] T037 [US2] Implement report calculation engine for asset timelines, source-year cutoff, opening and closing basis, inclusion rules, same-date reopening behavior, reference entries, summary entries, yearly net total, and shared report-calculation-currency enforcement in `internal/report/calculate/calculator.go`
- [X] T038 [US2] ⚠️ Reopened Implement priced liquidation proceeds, proportional allocation, explained zero-priced holding reductions, and basis removal details in `internal/report/calculate/calculator.go`, preserving explicit zero-valued source fields without creating proceeds, gains, losses, or priced-liquidation behavior (reopened — BUG-001)
- [X] T039 [US2] Implement non-secret report calculation error taxonomy with offending activity source ID and display label references in `internal/report/model/errors.go` and `internal/report/calculate/calculator.go`
- [X] T040 [P] [US2] ⚠️ Reopened Implement Markdown rendering for the required header, summary, reference section, per-asset detail sections, activity rows, liquidation tables, empty states, explicit report-calculation-currency labels, canonical decimals, and preserved explicit zero-valued explained zero-priced holding-reduction fields in `internal/report/markdown/renderer.go` (reopened — BUG-001)
- [X] T041 [P] [US2] Implement Documents directory resolution using Linux XDG user-dirs, macOS home Documents, and Windows user Documents conventions in `internal/report/output/documents.go`
- [X] T042 [P] [US2] Implement timestamped filename slugging, suffix reservation, exclusive final write, and failed-write cleanup in `internal/report/output/writer.go`
- [X] T043 [P] [US2] Implement OS default-app opener command adapter for Linux, macOS, and Windows with one post-save open request per successful run in `internal/report/output/opener.go`
- [X] T044 [US2] Implement runtime report service orchestration for request validation, calculation, rendering, save, opener warning, failure cleanup, saved-path removal guidance, and transient outcome creation in `internal/app/runtime/report_service.go`
- [X] T045 [US2] Wire the concrete report service into application assembly and TUI dependencies in `internal/app/runtime/runtime.go` and `internal/tui/flow/model.go`
- [X] T046 [P] [US2] Add report selection, report generation busy, and report result screen renderers in `internal/tui/screen/report_screen.go`
- [X] T047 [US2] Implement report year selection, method selection shell, async generation command, result routing, `Generate Another Report`, and `Back To Sync and Reports` behavior in `internal/tui/flow/report_flow.go`
- [X] T048 [US2] Enforce no in-application report history by clearing saved path, rendered content, and outcome state on result dismissal and context exit in `internal/tui/flow/report_flow.go` and `internal/tui/flow/model.go`

**Checkpoint**: User Story 2 can generate, save, and open a Markdown report from protected synced data and recover safely from calculation or output failures.

---

## Phase 5: User Story 3 - Choose And Understand A Cost Basis Method (Priority: P2)

**Goal**: Before generating the report, the user can review all supported cost basis methods, read a short explanation for each one, and choose the method that governs the report run.

**Independent Test**: Open the report-generation workflow with synced multi-year data, move through each method choice, verify the explanatory text, and compare method-specific outcomes against controlled expected ledgers.

### Tests for User Story 3

- [X] T049 [P] [US3] Add cost-basis method selection contract coverage for exact labels, stable slugs, and highlighted explanation text in `tests/contract/report_method_selection_contract_test.go`
- [X] T050 [P] [US3] Add integration coverage comparing controlled expected ledgers for FIFO, LIFO, HIFO, Average Cost Basis, and scope-local hybrid report outcomes in `tests/integration/report_cost_basis_methods_flow_test.go`
- [X] T051 [P] [US3] Add unit coverage for HIFO unit-cost tie-breaking, exact-division failure, scope-local reliable scope resolution, fallback activation, fallback carry-forward until zero, same-scope reset after reacquisition, and independent other-scope state in `tests/unit/report_basis_methods_test.go`

### Implementation for User Story 3

- [X] T052 [US3] Implement exact supported method labels, stable filename slugs, and plain-language explanation text in `internal/report/model/cost_basis_method.go`
- [X] T053 [US3] Update report selection flow so changing the highlighted method updates explanation text before generation in `internal/tui/screen/report_screen.go` and `internal/tui/flow/report_flow.go`
- [X] T054 [P] [US3] Implement applicable-scope resolution for reliable wallet or account scope and broaden-to-asset fallback in `internal/report/calculate/scope.go`
- [X] T055 [US3] Implement scope-local exact unit matching, scope-local average-cost fallback, oldest-acquired deemed-disposal order, fallback carry-forward until scope reaches zero, and same-scope post-zero reset in `internal/report/basis/scope_local_hybrid.go`
- [X] T056 [US3] Implement HIFO cross-multiplication comparison and deterministic older-lot tie-breaks without unnecessary division in `internal/report/basis/lot_methods.go`
- [X] T057 [US3] Extend deterministic report fixtures with expected per-method summaries, reference counts, detail ledgers, and shared report-calculation-currency expectations for all five supported methods in `tests/testutil/report_fixtures.go`

**Checkpoint**: All supported methods are visible, explained, selectable, and validated against controlled method-specific expected outcomes.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, security review, performance evidence, and release-level verification across all stories.

- [X] T058 [P] Update report workflow, protected-storage boundary, Documents output behavior, user file-removal guidance, and no report history documentation in `README.md`
- [X] T059 [P] Reconcile implemented commands, manual scenarios, mixed-currency, user file-removal guidance, artifact inspection, and output layout in `specs/005-generate-gains-report/quickstart.md`
- [X] T060 [P] Add OWASP Top 10, cryptographic-storage boundary, report cleartext output, and dependency/API review evidence in `specs/005-generate-gains-report/checklists/requirements.md`
- [X] T061 [P] Add opt-in deterministic 10,000-activity report performance coverage for one timed run of request validation, calculation, Markdown rendering, save, and opener stub invocation in `tests/integration/report_performance_flow_test.go`
- [X] T062 Run `make test` and `make coverage`, then verify report-feature coverage artifacts in `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`
- [X] T063 Run `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestReportPerformanceFlowLargeHistoryFixture -count=1 -v` and record the single-run outcome in `specs/005-generate-gains-report/quickstart.md`
- [X] T064 Inspect generated test artifacts and application-managed storage for cleartext report leakage, then document the result in `specs/005-generate-gains-report/checklists/requirements.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks all user stories.
- Phase 3, Phase 4, and Phase 5 depend on Phase 2.
- Phase 4 depends on Phase 3 for the user-facing unlocked context, but its report domain tests can start after Phase 2 using direct runtime and package-level seams.
- Phase 5 depends on Phase 4 report selection and calculation seams.
- Phase 6 depends on all selected user stories being complete.

### Dependency Graph

```text
Phase 1 Setup
  -> Phase 2 Foundational
    -> US1 Sync And Reports Context
      -> US2 Markdown Report Generation
        -> US3 Cost Basis Method Choice
          -> Phase 6 Polish

Parallel-capable after Phase 2:
  -> US1 TUI context work
  -> US2 report-domain package tests and implementation seams
```

### User Story Dependencies

- US1 depends only on Foundational work and is the MVP scope.
- US2 depends on US1 for end-to-end TUI access, but report calculation, rendering, and output packages can be built and tested independently once Foundational work is complete.
- US3 depends on US2 because method explanations and method-specific ledgers sit on the report selection and calculation flow.

### Within Each User Story

- Write the listed tests first and confirm they fail for the targeted behavior.
- Complete models before calculation services.
- Complete calculation and rendering before runtime orchestration.
- Complete runtime orchestration before TUI navigation.
- Finish each story checkpoint before moving to the next priority story.

### Parallel Opportunities

- T001, T002, and T003 can run in parallel.
- T004, T006, T008, T009, T010, T011, and T067 can run in parallel after Phase 1; T005, T007, T012, and T068 close after the related model, snapshot, and fixture decisions exist.
- T013, T014, T015, T069, T071, and T073 can run in parallel for US1 before T016 through T024, T070, and T072.
- T025 through T032 and T065 can run in parallel for US2; T033 through T036, T040 through T043, and T066 can then run in parallel before T037, T038, T044, T045, T047, and T048.
- T049 through T051 can run in parallel for US3; T052, T054, and T056 can run in parallel before T053, T055, and T057 close the story.
- T058, T059, T060, and T061 can run in parallel after story implementation; T062, T063, and T064 run after documentation and performance fixtures are in place.

---

## Parallel Example: User Story 1

```bash
Task: T013 Add main-menu and Sync and Reports workflow contract coverage in tests/contract/main_menu_workflow_contract_test.go and tests/contract/sync_reports_workflow_contract_test.go
Task: T014 Add Sync and Reports context integration coverage in tests/integration/sync_reports_context_flow_test.go
Task: T015 Add Sync and Reports screen rendering coverage in internal/tui/screen/sync_reports_screen_internal_test.go
Task: T073 Add rejected-token unlock-screen rendering regression coverage in internal/tui/screen/sync_reports_screen_internal_test.go
Task: T069 Add in-context Sync Data screen rendering regression coverage in internal/tui/screen/sync_entry_screen_internal_test.go
Task: T071 Add internal flow regression coverage for skip-disabled Sync and Reports navigation in internal/tui/flow/model_internal_test.go

Task: T017 Add Sync and Reports screen renderers in internal/tui/screen/sync_reports_screen.go
Task: T070 Add in-context Sync Data non-token screen mode in internal/tui/screen/sync_entry_screen.go
Task: T019 Implement runtime selected-server snapshot unlock in internal/app/runtime/sync_service.go and internal/app/runtime/snapshot_lifecycle.go
Task: T072 Update Sync and Reports navigation so disabled report actions are skipped in internal/tui/flow/navigation.go
```

## Parallel Example: User Story 2

```bash
Task: T025 Add Markdown report contract coverage in tests/contract/markdown_report_contract_test.go
Task: T027 Add report generation integration coverage in tests/integration/report_generation_flow_test.go
Task: T029 Add report calculation unit coverage in tests/unit/report_calculation_test.go
Task: T030 Add report output unit coverage in tests/unit/report_output_test.go
Task: T031 Add report Markdown unit coverage in tests/unit/report_markdown_test.go
Task: T032 Add activity input unit coverage in tests/unit/report_activity_input_test.go
Task: T065 Add explicit-zero explained SELL fixtures in tests/testutil/report_fixtures.go

Task: T033 Implement activity input selection in internal/report/calculate/activity_input.go
Task: T066 Extend report activity-row models in internal/report/model/report.go
Task: T035 Implement lot basis methods in internal/report/basis/lot_methods.go
Task: T036 Implement average cost basis in internal/report/basis/average_cost.go
Task: T040 Implement Markdown renderer in internal/report/markdown/renderer.go
Task: T041 Implement Documents resolver in internal/report/output/documents.go
Task: T042 Implement report writer in internal/report/output/writer.go
Task: T043 Implement opener adapter in internal/report/output/opener.go
```

## Parallel Example: User Story 3

```bash
Task: T049 Add method selection contract coverage in tests/contract/report_method_selection_contract_test.go
Task: T050 Add method outcome integration coverage in tests/integration/report_cost_basis_methods_flow_test.go
Task: T051 Add basis method unit coverage in tests/unit/report_basis_methods_test.go

Task: T052 Implement method labels, slugs, and explanations in internal/report/model/cost_basis_method.go
Task: T054 Implement applicable scope resolution in internal/report/calculate/scope.go
Task: T056 Implement HIFO comparison refinements in internal/report/basis/lot_methods.go
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1.
2. Complete Phase 2.
3. Complete Phase 3 for User Story 1.
4. Stop and validate the Sync and Reports context independently before implementing report calculation.

### Incremental Delivery

1. Deliver US1 so the application has the unlocked sync/report context and token reuse.
2. Deliver US2 so the report can be calculated, rendered, saved, opened, and cleaned up safely.
3. Deliver US3 so every supported cost basis method is explained and verified against controlled ledgers.
4. Finish Phase 6 to lock documentation, security review, coverage, and performance evidence.

### Parallel Team Strategy

1. One contributor owns the TUI context and token lifecycle while another owns report calculation and another owns output/Markdown behavior.
2. Merge at `internal/app/runtime/report_service.go` only after calculation, rendering, and output package tests pass independently.
3. Merge at `internal/tui/flow/model.go` only after US1 context routing and US2 report routing tests pass independently.

---

## Notes

- `[P]` tasks touch different files or package seams and can run in parallel after their listed dependencies are satisfied.
- `[US1]`, `[US2]`, and `[US3]` labels map tasks directly to user stories in `spec.md`.
- Report generation must not call Ghostfolio and must use only the protected activity cache from the unlocked context.
- Report content must remain in memory until the final Documents file is saved.
- No task should persist report content, generated report paths, or report history into setup or protected snapshots.
