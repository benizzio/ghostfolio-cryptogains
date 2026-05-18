---

description: "Task list for Store Activity Data implementation"
---

# Tasks: Store Activity Data

**Input**: Design documents from `/specs/003-store-activity-data/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature. Write each story's tests first, make them fail for the targeted behavior, keep 100% statement coverage from `go test` plus 100% branch and file coverage for project-owned code with the `gocoverageplus` gate, add persisted-artifact leakage checks, and keep a documented large-history performance verification path for `SC-006`.

**Bugfix**: 2026-05-15 — [BUG-001] Added synced-data diagnostic-report tasks and security verification coverage.
**Bugfix**: 2026-05-15 — [BUG-002] Reopened deterministic-ordering tasks for Ghostfolio `date` time-of-day fragility.
**Bugfix**: 2026-05-17 — [BUG-003] Reopened currency-context tasks and added mixed-currency storage follow-up work.
**Bugfix**: 2026-05-18 — [BUG-004] Reopened nullable-contract currency-tier work and added follow-up tasks.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Path Conventions

- Executable entrypoint: `cmd/ghostfolio-cryptogains/`
- App wiring and runtime orchestration: `internal/app/`
- Bootstrap persistence: `internal/config/`
- Ghostfolio transport and mapping: `internal/ghostfolio/`
- Protected snapshot storage: `internal/snapshot/`
- Sync normalization and validation: `internal/sync/`
- Bubble Tea screens and flow: `internal/tui/`
- Shared precision and redaction helpers: `internal/support/`
- Automated tests: `tests/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add the new package skeleton, dependency wiring, and traceable research evidence required for full-history sync and protected snapshot storage.

- [X] T001 Update module dependencies for `github.com/cockroachdb/apd/v3` and `golang.org/x/crypto/argon2` in `go.mod` and `go.sum`
- [X] T002 [P] Create the protected snapshot package skeleton in `internal/snapshot/envelope/`, `internal/snapshot/model/`, and `internal/snapshot/store/`
- [X] T003 [P] Create the normalized sync package skeleton in `internal/sync/model/`, `internal/sync/normalize/`, and `internal/sync/validate/`
- [X] T047 [P] Refresh dependency due-diligence evidence for `github.com/cockroachdb/apd/v3` and `golang.org/x/crypto/argon2` in `specs/003-store-activity-data/research.md`
- [X] T048 [P] ⚠️ Reopened Refresh Ghostfolio auth, user, and pagination contract review evidence in `specs/003-store-activity-data/research.md` and `specs/003-store-activity-data/contracts/ghostfolio-sync.md` (reopened — BUG-004: confirm nullable `Order.currency`, optional `SymbolProfile.currency`, optional authenticated `settings.baseCurrency`, and the three-tier currency-context rule are fully propagated before `T045`, `T046`, and `T055` close)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared storage, exact-decimal, transport, and runtime primitives that every user story depends on.

**Critical**: Finish this phase before starting story implementation.

- [X] T004 Implement exact-decimal parsing and canonical string helpers in `internal/support/decimal/decimal.go`
- [X] T005 [P] ⚠️ Reopened Extend Ghostfolio transport DTOs for full paginated activities and per-field currency context in `internal/ghostfolio/dto/auth_response.go` and `internal/ghostfolio/dto/activity_page_response.go` (reopened — BUG-003: capture `currency`, `fee`, `unitPrice`, `value`, `feeInAssetProfileCurrency`, `feeInBaseCurrency`, `unitPriceInAssetProfileCurrency`, `valueInBaseCurrency`, and `SymbolProfile.currency`)
- [X] T006 [P] Implement full-history response validation helpers for auth and paginated activities in `internal/ghostfolio/validator/response_validator.go`
- [X] T007 [P] ⚠️ Reopened Define the persisted normalized activity, scope, and protected-cache models plus the sync-attempt runtime types they feed in `internal/sync/model/activity_record.go`, `internal/sync/model/protected_activity_cache.go`, and `internal/app/runtime/sync_types.go` (reopened — BUG-003: redesign persisted activity monetary fields so each preserved amount keeps explicit order-currency, asset-profile-currency, or base-currency identity, and keep runtime-only sync types persistence-neutral unless snapshot compatibility intentionally changes)
- [X] T008 [P] Define protected snapshot envelope, payload, version, and profile models in `internal/snapshot/model/envelope.go` and `internal/snapshot/model/payload.go`
- [X] T009 [P] Implement snapshot envelope encoding, server discovery key derivation, and AEAD header authentication helpers in `internal/snapshot/envelope/codec.go`
- [X] T010 [P] Implement protected snapshot path resolution, candidate enumeration, and atomic file-replacement helpers in `internal/snapshot/store/store.go`
- [X] T011 Implement runtime dependency wiring for decimal, sync, and snapshot services in `internal/app/runtime/runtime.go`

**Checkpoint**: BUG-004 reopens `T048` for remaining Ghostfolio contract-propagation work.

---

## Phase 3: User Story 1 - Sync And Store Full Activity History (Priority: P1) 🎯 MVP

**Goal**: Let the user run `Sync Data`, retrieve the full supported Ghostfolio activity history, normalize and validate it, and store it only as a protected snapshot for future reporting use.

**Independent Test**: With completed setup and a valid token, `Sync Data` retrieves a multi-page or empty history, stores a protected snapshot, and ends with a success result that confirms storage for future use without exposing reporting behavior.

### Tests for User Story 1

- [X] T012 [P] [US1] ⚠️ Reopened Add Ghostfolio full-history contract coverage for auth, authenticated user retrieval, pagination, empty-history success, mixed-currency activity fields, and nullable or optional currency-definition tiers in `tests/contract/ghostfolio_sync_storage_contract_test.go` (reopened — BUG-004: cover `GET /api/v1/user` `settings.baseCurrency` present and omitted alongside `Order.currency = null` and omitted `SymbolProfile.currency`)
- [X] T013 [P] [US1] Add sync workflow contract coverage for busy-state, success result wording, and non-reporting UI scope in `tests/contract/sync_storage_workflow_contract_test.go`
- [X] T014 [P] [US1] Add integration coverage for first successful sync, multi-page retrieval, protected snapshot creation, and empty-history success in `tests/integration/sync_storage_flow_test.go`
- [X] T015 [P] [US1] Add unit coverage for exact-decimal parsing and year-derivation rules in `tests/unit/decimal_test.go` and `tests/unit/year_derivation_test.go`
- [X] T016 [P] [US1] Add unit coverage for envelope encoding, header authentication, and atomic snapshot replacement in `tests/unit/snapshot_envelope_test.go` and `tests/unit/snapshot_store_test.go`

### Implementation for User Story 1

- [X] T017 [P] [US1] Implement paginated Ghostfolio activities retrieval with `skip`, `take`, and ascending date order in `internal/ghostfolio/client/client.go`
- [X] T018 [P] [US1] ⚠️ Reopened Implement Ghostfolio activity-to-normalized-record mapping in `internal/ghostfolio/mapper/activity_mapper.go` (reopened — BUG-003: keep order-currency, asset-profile-currency, and base-currency values explicitly tied to their source currencies; BUG-004: consume the authenticated-user base-currency context delivered by `T060`, keep those tiers independent, and do not reject a row solely because one tier is uninformed while other tiers remain informed)
- [X] T060 [P] [US1] Extend the authenticated Ghostfolio sync boundary to fetch `GET /api/v1/user` and pass optional `settings.baseCurrency` into mapping and runtime sync state as an independent currency-definition tier in `internal/ghostfolio/client/client.go`, `internal/app/runtime/sync_service.go`, and `internal/app/runtime/sync_types.go`
- [X] T019 [P] [US1] ⚠️ Reopened Implement chronological normalization, same-asset source-calendar-date ordering, duplicate hashing, and available-year derivation in `internal/sync/normalize/activity_history.go` (reopened — BUG-002: preserve original timestamps in stored records while establishing the normalized same-asset ordering rule)
- [X] T020 [P] [US1] Implement supported-history validation for `BUY` and `SELL` activity rules in `internal/sync/validate/activity_history.go`
- [X] T021 [P] [US1] Implement protected snapshot encryption, decryption, and atomic persistence in `internal/snapshot/store/encrypted_store.go`
- [X] T022 [US1] Implement full sync orchestration from auth through protected write in `internal/app/runtime/sync_service.go`
- [X] T023 [US1] Update sync flow busy-state lifecycle and result routing for full-history storage in `internal/tui/flow/sync_flow.go`
- [X] T024 [US1] Replace validation-only sync entry and result screens with storage-focused wording in `internal/tui/screen/sync_validation_screen.go` and `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: BUG-004 reopens `T012`, `T018`, and `T060` for nullable-contract currency-tier handling in User Story 1.

---

## Phase 4: User Story 2 - Reuse Token-Locked Stored Data (Priority: P1)

**Goal**: Reuse existing protected snapshots with the correct token, isolate different valid tokens into separate snapshots, and fail safely when tokens or stored-data versions are incompatible.

**Independent Test**: After one successful sync, rerunning `Sync Data` with the same token refreshes the existing snapshot only after success; a different valid token creates a new snapshot; invalid tokens and unsupported stored-data versions leave local data unchanged.

### Tests for User Story 2

- [X] T025 [P] [US2] Add protected-snapshot discovery and compatibility contract coverage in `tests/contract/protected_snapshot_contract_test.go`
- [X] T026 [P] [US2] Add integration coverage for same-token refresh, wrong-token denial, different-valid-token isolation, and invalid-token no-change behavior in `tests/integration/snapshot_reuse_flow_test.go`
- [X] T027 [P] [US2] Add integration coverage for unsupported envelope version, unsupported payload version, and incompatible new sync data retention in `tests/integration/snapshot_compatibility_flow_test.go`
- [X] T028 [P] [US2] Add unit coverage for server-scoped candidate filtering and payload version checks in `tests/unit/snapshot_discovery_test.go` and `tests/unit/stored_data_version_test.go`
- [X] T051 [P] [US2] Add contract and integration coverage for production opt-in, explicit-development-mode automatic synced-data diagnostic reports, and generated-report path disclosure in `tests/contract/sync_storage_workflow_contract_test.go`, `tests/contract/validation_result_workflow_contract_test.go`, and `tests/integration/sync_diagnostic_report_flow_test.go`

### Implementation for User Story 2

- [X] T029 [P] [US2] Implement server-scoped snapshot header discovery and candidate filtering in `internal/snapshot/store/discovery.go`
- [X] T030 [P] [US2] Implement stored-data version compatibility checks for envelope and payload models in `internal/snapshot/model/version.go` and `internal/snapshot/store/compatibility.go`
- [X] T031 [P] [US2] Implement protected snapshot unlock, active readable snapshot tracking, and isolated snapshot creation for new valid tokens in `internal/app/runtime/sync_service.go`
- [X] T032 [US2] Implement failure and success result handling for rejected token, unsupported stored-data version, and incompatible new sync data in `internal/tui/flow/sync_flow.go` and `internal/tui/screen/validation_result_screen.go`
- [X] T052 [US2] Implement synced-data diagnostic-report policy using the existing explicit-development-mode runtime option, local artifact writes, and result-screen report-location messaging in `internal/app/runtime/sync_service.go`, `internal/tui/flow/sync_flow.go`, and `internal/tui/screen/validation_result_screen.go`
- [X] T056 [P] [US2] Update `activity_model_version` handling and compatibility fixtures for the currency-aware activity-record design in `internal/snapshot/model/version.go`, `tests/integration/snapshot_compatibility_flow_test.go`, and `tests/unit/stored_data_version_test.go` (older pre-BUG-003 snapshots must fail with a compatibility error unless an explicit migration is added)

**Checkpoint**: BUG-003 activity-model compatibility updates for User Story 2 are complete.

---

## Phase 5: User Story 3 - Preserve Data Quality And Server Boundaries (Priority: P1)

**Goal**: Store only defensible normalized histories and prevent silent contamination or replacement when source data is unsupported or the selected server changes.

**Independent Test**: Unsupported activity types, invalid zero-price cases, unstable ordering, non-defensible histories, and declined server replacement all fail safely and preserve the previous readable snapshot.

### Tests for User Story 3

- [X] T033 [P] [US3] ⚠️ Reopened Add normalization and defensibility contract coverage for supported activity rules, below-zero holdings rejection, and scope-reliability outcome categories in `tests/contract/activity_validation_contract_test.go` (reopened — BUG-002: cover same-asset same-day Ghostfolio histories with arbitrary time values)
- [X] T034 [P] [US3] Add server-mismatch confirmation workflow contract coverage in `tests/contract/server_replacement_workflow_contract_test.go`
- [X] T035 [P] [US3] ⚠️ Reopened Add integration coverage for unsupported activity history, duplicate removal, deterministic ordering, below-zero holdings rejection, and zero-price rule handling in `tests/integration/activity_validation_flow_test.go` (reopened — BUG-002: cover same-asset same-day histories that must order `BUY` before `SELL` before `source_id` when Ghostfolio time values are arbitrary)
- [X] T036 [P] [US3] Add integration coverage for server replacement confirm, cancel, success, and failed-replacement retention in `tests/integration/server_replacement_flow_test.go`
- [X] T037 [P] [US3] ⚠️ Reopened Add unit coverage for duplicate hashing, tie-break ordering, running-quantity defensibility checks, and scope-reliability derivation in `tests/unit/activity_normalization_test.go` and `tests/unit/scope_reliability_test.go` (reopened — BUG-002: tie-break ordering must use source calendar date, `activity_type`, then `source_id`)
- [X] T057 [P] [US3] ⚠️ Reopened Add contract, integration, and unit coverage for mixed-currency activities, nullable `Order.currency`, optional `SymbolProfile.currency`, optional authenticated `settings.baseCurrency`, tier-specific currency-context outcomes, and end-to-end authenticated-user base-currency propagation through the sync path in `tests/contract/activity_validation_contract_test.go`, `tests/integration/activity_validation_flow_test.go`, and `tests/unit/activity_normalization_test.go` (reopened — BUG-004: valid rows with one uninformed tier must succeed when other tiers remain informed, and integration coverage must exercise present and omitted `settings.baseCurrency` across client retrieval, mapping, and validation)
- [X] T061 [P] [US3] Add reusable Ghostfolio fixture permutations for `Order.currency = null`, missing `SymbolProfile.currency`, and missing authenticated `settings.baseCurrency` in `tests/testutil/testutil.go`, `tests/integration/helpers_test.go`, and the affected currency-context test files

### Implementation for User Story 3

- [X] T038 [P] [US3] ⚠️ Reopened Implement running-quantity replay support and source-scope reliability derivation in `internal/sync/normalize/activity_history.go` so they consume the same-asset source-calendar-date ordering established in `T019` (reopened — BUG-002: replay must use source calendar date, `activity_type`, then `source_id`)
- [X] T039 [P] [US3] ⚠️ Reopened Implement general defensibility checks for missing or internally inconsistent normalized fields, below-zero holdings, zero-priced `SELL` comment rules, and unsupported-history rejection in `internal/sync/validate/activity_history.go` (reopened — BUG-002: defensibility must evaluate same-asset ordering after source calendar date, `activity_type`, and `source_id` tie-breaking; BUG-004 follow-up is isolated in `T058` so this task keeps the remaining defensibility checks)
- [X] T053 [US3] Extend mapping, normalization, and validation failures to surface offending-record diagnostic context and production/dev redaction inputs in `internal/ghostfolio/mapper/activity_mapper.go`, `internal/sync/normalize/activity_history.go`, `internal/sync/validate/activity_history.go`, and `internal/app/runtime/sync_service.go`
- [X] T058 [US3] ⚠️ Reopened Implement validation that rejects currency context only when all three independent tiers are uninformed for preserved monetary data and classifies the failure for the existing synced-data diagnostic-report policy in `internal/sync/validate/activity_history.go`
- [X] T059 [US3] ⚠️ Reopened Implement offending-record diagnostic details for all-tier-uninformed currency-context failures, reusing the existing production redaction and explicit-development-mode detail rules, in `internal/sync/validate/activity_history.go` and `internal/app/runtime/sync_service.go`
- [X] T040 [P] [US3] Implement server-mismatch detection and replacement gating against the active readable snapshot in `internal/app/runtime/sync_service.go`
- [X] T041 [US3] Implement server replacement confirmation screen and navigation in `internal/tui/screen/server_replacement_screen.go` and `internal/tui/flow/sync_flow.go`
- [X] T042 [US3] Update the main menu and sync entry screens to surface protected-data-exists state without exposing cached activity details in `internal/tui/screen/main_menu_screen.go` and `internal/tui/screen/sync_validation_screen.go`

**Checkpoint**: BUG-004 reopens `T057`, `T058`, `T059`, and `T061` for three-tier currency-context validation follow-up in User Story 3. `T039` remains the broader defensibility task in the same validator.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, release checks, security verification, and cross-story verification.

- [X] T043 [P] Update protected-storage, diagnostic-report, removal, and no-reporting documentation in `README.md`
- [X] T044 [P] Reconcile `specs/003-store-activity-data/quickstart.md` with the implemented sync result categories, diagnostic-report generation and inspection steps, persisted-artifact inspection steps, large-history performance verification steps, and verification commands in `specs/003-store-activity-data/quickstart.md`
- [X] T062 [P] Reconcile `specs/003-store-activity-data/data-model.md` `ActivityRecord.order_currency`, `ActivityRecord.asset_profile_currency`, `ActivityRecord.base_currency`, and `GhostfolioSession.user_base_currency` notes with BUG-004 nullable and optional currency-definition tiers plus authenticated-user `settings.baseCurrency` sourcing
- [X] T045 [P] ⚠️ Reopened Refresh the documented OWASP Top 10 and Cryptographic Storage review summary, dependency and API research evidence, and the `SC-006` performance-verification evidence in `specs/003-store-activity-data/checklists/requirements.md` after the BUG-004 Ghostfolio contract review, data-model alignment, and final verification rerun are complete
- [X] T049 [P] Add integration coverage that bootstrap files, protected snapshots, generated diagnostic reports, and persisted workflow artifacts never store Ghostfolio tokens, raw payload fragments, transient sync-failure messages, or production-disallowed financial-value fields in `tests/integration/persistence_security_flow_test.go`
- [X] T050 [P] Add deterministic large-history performance verification coverage for authenticated retrieval, normalization, validation, and protected replacement in `tests/integration/sync_performance_flow_test.go`
- [X] T046 ⚠️ Reopened Run `make test`, `make coverage`, and the documented large-history performance verification after the BUG-004 remediation tasks, then verify the generated artifacts in `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`
- [X] T054 ⚠️ Conditional rerun task: if the final BUG-004 verification rerun exposes coverage gaps, add targeted tests to address them and rerun verification until all gates are satisfied, following defined test approaches
- [X] T055 ⚠️ Reopened After the final BUG-004 verification rerun, certify that the coverage gates are met again; if the rerun exposes gaps, reopen `T054`, add the required tests, and rerun verification

**Checkpoint**: BUG-004 Phase 6 follow-up now includes `T062` plus reruns in `T045`, `T046`, and `T055`.

---

## Phase 7: Test Coverage Drift Remediation

**Purpose**: Restore the documented 100% coverage gate enforcement and close the remaining uncovered feature-owned branches identified in `specs/003-store-activity-data/test-coverage-drift-report.md`.

- [X] T063 [P] COV-DRIFT-001 High: Enforce the maintained 100% coverage gate in `Makefile` and `.github/workflows/test.yml` so the documented verification path and PR workflow fail when `dist/coverage/coverage.xml` remains below target, following `specs/003-store-activity-data/test-coverage-drift-report.md#cov-drift-001-coverage-gate-is-not-enforced-in-the-maintained-verification-path`
- [X] T064 [P] COV-DRIFT-002 High: Add integration-first and targeted unit coverage for mapper diagnostic fallback paths, currency-resolution branches, and protected-snapshot runtime nil/error guards in `tests/integration/sync_storage_flow_test.go`, `tests/integration/snapshot_reuse_flow_test.go`, `tests/unit/activity_amount_resolution_test.go`, `tests/unit/snapshot_lifecycle_test.go`, and `tests/unit/active_snapshot_state_test.go` to cover the evidence in `internal/ghostfolio/mapper/activity_mapper.go`, `internal/sync/model/activity_amount_resolution.go`, `internal/app/runtime/snapshot_lifecycle.go`, `internal/app/runtime/active_snapshot_state.go`, and `dist/coverage/coverage.xml`, following `specs/003-store-activity-data/test-coverage-drift-report.md#cov-drift-002-active-feature-code-still-measures-below-the-required-100-target`
- [X] T065 Re-run `make test` and `make coverage`, verify `dist/coverage/coverage.out` plus `dist/coverage/coverage.xml` report 100% statement, branch, and file coverage for the Store Activity Data slice, and confirm the enforced verification path closes `COV-DRIFT-001` and `COV-DRIFT-002` in `specs/003-store-activity-data/test-coverage-drift-report.md`

**Checkpoint**: Coverage drift remediation is complete only after `T063` through `T065` restore enforcement and the measured coverage gates pass again.

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks all story work.
- Phase 3, Phase 4, and Phase 5 depend on Phase 2.
- Phase 6 depends on the stories selected for release.
- Phase 7 depends on Phase 6 and runs only after the implementation phases are complete.
- Final Phase 6 evidence and verification closure for this slice now depends on the remaining nullable-contract follow-up and unchecked validation work, especially `T048`, `T012`, `T018`, `T039`, `T057`, `T058`, `T059`, `T060`, `T061`, `T062`, and the rerun tasks `T045`, `T046`, and `T055`.
- Final coverage-drift closure now depends on `T063`, `T064`, and `T065`.

### Dependency Graph

```text
Phase 1 Setup
  -> Phase 2 Foundational
    -> US1 Sync And Store Full Activity History
      -> US2 Reuse Token-Locked Stored Data
      -> US3 Preserve Data Quality And Server Boundaries
        -> Phase 6 Polish
          -> Phase 7 Test Coverage Drift Remediation

Cross-story runtime dependencies:
  US1 provides the initial full-history sync, protected snapshot write, and success result path.
  US2 extends the same runtime path with snapshot reuse, token isolation, and compatibility retention.
  US3 extends the same runtime path with stricter history validation and server replacement gating.
```

### User Story Dependencies

- US1 depends only on Foundational work and is the MVP slice for this feature.
- US2 depends on US1 because snapshot reuse and isolated refresh require the first protected snapshot flow to exist.
- US3 depends on US1 for the protected sync path and shares runtime orchestration with US2, but its validation and server-boundary work can be developed in parallel once the US1 skeleton is stable.

### Within Each User Story

- Write the listed tests first and confirm they fail for the targeted behavior.
- Finish transport and model changes before sync orchestration.
- For BUG-004 in US1, close `T060` before closing `T018` so mapper behavior includes authenticated-user base-currency context.
- For BUG-004 in US3, close `T057` and `T061` before certifying `T058` and `T059` so the reopened fixtures define the expected validation and diagnostic outcomes.
- Finish runtime orchestration before TUI messaging and navigation updates.
- Finish the story before moving to polish for that story.

### Parallel Opportunities

- T001, T047, and T048 can run in parallel at the start of Phase 1; T002 and T003 can run in parallel after T001.
- T005 through T010 can run in parallel once T004 defines the decimal primitives.
- T012 through T016 can run in parallel for US1, then T017 and T060 can run in parallel; `T018` starts after `T060` defines authenticated-user base-currency plumbing, while `T019` through `T021` can proceed in parallel before `T022` through `T024`.
- T025 through T028 and T051 can run in parallel for US2, then T029 and T030 can run in parallel before T031, T032, T052, and T056.
- T033 through T037, T057, and T061 can run in parallel for US3; `T038` through `T040` and `T053` can proceed after the shared fixtures exist, and `T058` plus `T059` close after `T057` and `T061` establish the BUG-004 validation and diagnostic expectations, before `T041` and `T042`.
- T043, T044, T049, T050, and T062 can run in parallel once the release scope is stable; `T045`, `T046`, and `T055` are the verification reruns that close the BUG-004 follow-up work after the reopened tasks complete.
- T063 and `T064` can run in parallel after Phase 6; `T065` runs after both remediation tasks complete.

---

## Parallel Example: User Story 1

```bash
Task: T012 Add Ghostfolio full-history contract coverage in tests/contract/ghostfolio_sync_storage_contract_test.go
Task: T013 Add sync workflow contract coverage in tests/contract/sync_storage_workflow_contract_test.go
Task: T014 Add first successful sync and empty-history integration coverage in tests/integration/sync_storage_flow_test.go
Task: T015 Add exact-decimal and year-derivation unit coverage in tests/unit/decimal_test.go and tests/unit/year_derivation_test.go
Task: T016 Add snapshot envelope and atomic-write unit coverage in tests/unit/snapshot_envelope_test.go and tests/unit/snapshot_store_test.go

Task: T017 Implement paginated Ghostfolio retrieval in internal/ghostfolio/client/client.go
Task: T060 Extend the authenticated Ghostfolio sync boundary for optional settings.baseCurrency in internal/ghostfolio/client/client.go and internal/app/runtime/
Task: T018 Implement activity mapping in internal/ghostfolio/mapper/activity_mapper.go after T060 exposes authenticated-user base-currency context
Task: T019 Implement normalization and year derivation in internal/sync/normalize/activity_history.go
Task: T020 Implement supported-history validation in internal/sync/validate/activity_history.go
Task: T021 Implement encrypted snapshot persistence in internal/snapshot/store/encrypted_store.go
```

## Parallel Example: User Story 2

```bash
Task: T025 Add protected-snapshot discovery contract coverage in tests/contract/protected_snapshot_contract_test.go
Task: T026 Add same-token refresh and token-isolation integration coverage in tests/integration/snapshot_reuse_flow_test.go
Task: T027 Add compatibility-retention integration coverage in tests/integration/snapshot_compatibility_flow_test.go
Task: T028 Add snapshot discovery and stored-data-version unit coverage in tests/unit/snapshot_discovery_test.go and tests/unit/stored_data_version_test.go

Task: T029 Implement server-scoped snapshot discovery in internal/snapshot/store/discovery.go
Task: T030 Implement stored-data compatibility checks in internal/snapshot/model/version.go and internal/snapshot/store/compatibility.go
```

## Parallel Example: User Story 3

```bash
Task: T033 Add activity validation contract coverage in tests/contract/activity_validation_contract_test.go
Task: T034 Add server replacement workflow contract coverage in tests/contract/server_replacement_workflow_contract_test.go
Task: T035 Add unsupported-history and normalization integration coverage in tests/integration/activity_validation_flow_test.go
Task: T036 Add server replacement integration coverage in tests/integration/server_replacement_flow_test.go
Task: T037 Add normalization and scope-reliability unit coverage in tests/unit/activity_normalization_test.go and tests/unit/scope_reliability_test.go

Task: T038 Implement deterministic normalization details in internal/sync/normalize/activity_history.go
Task: T039 Implement defensibility and zero-price validation in internal/sync/validate/activity_history.go
Task: T040 Implement server-mismatch detection in internal/app/runtime/sync_service.go
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1.
2. Complete Phase 2.
3. Complete Phase 3.
4. Validate full-history sync, protected snapshot creation, and empty-history success before expanding reuse and replacement behavior.

### Incremental Delivery

1. Deliver US1 so the application can fetch and store protected full-history data.
2. Deliver US2 so protected snapshots can be reused safely across runs and isolated by token.
3. Deliver US3 so invalid histories and server changes cannot silently replace good data.
4. Finish Phase 6 to lock documentation, security review, and coverage evidence.
5. Finish Phase 7 to restore enforced 100% coverage gating and close the recorded coverage drift.

### Parallel Team Strategy

1. One contributor owns Ghostfolio transport and runtime orchestration while another owns snapshot storage primitives and another owns TUI workflow updates.
2. After Phase 2, one contributor can drive US1 tests and runtime changes while another prepares US3 validation logic in parallel against fixtures.
3. Merge at `internal/app/runtime/sync_service.go` and `internal/tui/flow/sync_flow.go` only after each story's tests pass independently.

---

## Notes

- `[P]` tasks touch different files and can run in parallel.
- `[US1]`, `[US2]`, and `[US3]` labels map tasks directly to the feature specification user stories.
- Each user story remains independently testable from the command-line and TUI workflows described in `quickstart.md`.
- No task introduces reporting, report preview, or cached-activity browsing because those behaviors are explicitly out of scope for this slice.
