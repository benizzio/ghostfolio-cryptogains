---

description: "Task list for Store Activity Data implementation"
---

# Tasks: Store Activity Data

**Input**: Design documents from `/specs/003-store-activity-data/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature. Write each story's tests first, make them fail for the targeted behavior, keep 100% statement coverage from `go test` plus 100% branch and file coverage for project-owned code with the `gocoverageplus` gate, add persisted-artifact leakage checks, and keep a documented large-history performance verification path for `SC-006`.

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

- [ ] T001 Update module dependencies for `github.com/cockroachdb/apd/v3` and `golang.org/x/crypto/argon2` in `go.mod` and `go.sum`
- [ ] T002 [P] Create the protected snapshot package skeleton in `internal/snapshot/envelope/`, `internal/snapshot/model/`, and `internal/snapshot/store/`
- [ ] T003 [P] Create the normalized sync package skeleton in `internal/sync/model/`, `internal/sync/normalize/`, and `internal/sync/validate/`
- [ ] T047 [P] Refresh dependency due-diligence evidence for `github.com/cockroachdb/apd/v3` and `golang.org/x/crypto/argon2` in `specs/003-store-activity-data/research.md`
- [ ] T048 [P] Refresh Ghostfolio auth and pagination contract review evidence in `specs/003-store-activity-data/research.md` and `specs/003-store-activity-data/contracts/ghostfolio-sync.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the shared storage, exact-decimal, transport, and runtime primitives that every user story depends on.

**Critical**: Finish this phase before starting story implementation.

- [ ] T004 Implement exact-decimal parsing and canonical string helpers in `internal/support/decimal/decimal.go`
- [ ] T005 [P] Extend Ghostfolio transport DTOs for full paginated activities in `internal/ghostfolio/dto/auth_response.go` and `internal/ghostfolio/dto/activity_page_response.go`
- [ ] T006 [P] Implement full-history response validation helpers for auth and paginated activities in `internal/ghostfolio/validator/response_validator.go`
- [ ] T007 [P] Define normalized activity, scope, cache, and sync-attempt runtime models in `internal/sync/model/activity_record.go`, `internal/sync/model/protected_activity_cache.go`, and `internal/app/runtime/sync_types.go`
- [ ] T008 [P] Define protected snapshot envelope, payload, version, and profile models in `internal/snapshot/model/envelope.go` and `internal/snapshot/model/payload.go`
- [ ] T009 [P] Implement snapshot envelope encoding, server discovery key derivation, and AEAD header authentication helpers in `internal/snapshot/envelope/codec.go`
- [ ] T010 [P] Implement protected snapshot path resolution, candidate enumeration, and atomic file-replacement helpers in `internal/snapshot/store/store.go`
- [ ] T011 Implement runtime dependency wiring for decimal, sync, and snapshot services in `internal/app/runtime/runtime.go`

**Checkpoint**: The codebase has the shared models and infrastructure needed for tests-first implementation of full sync, protected storage, and server-scoped snapshot reuse.

---

## Phase 3: User Story 1 - Sync And Store Full Activity History (Priority: P1) 🎯 MVP

**Goal**: Let the user run `Sync Data`, retrieve the full supported Ghostfolio activity history, normalize and validate it, and store it only as a protected snapshot for future reporting use.

**Independent Test**: With completed setup and a valid token, `Sync Data` retrieves a multi-page or empty history, stores a protected snapshot, and ends with a success result that confirms storage for future use without exposing reporting behavior.

### Tests for User Story 1

- [ ] T012 [P] [US1] Add Ghostfolio full-history contract coverage for auth, pagination, and empty-history success in `tests/contract/ghostfolio_sync_storage_contract_test.go`
- [ ] T013 [P] [US1] Add sync workflow contract coverage for busy-state, success result wording, and non-reporting UI scope in `tests/contract/sync_storage_workflow_contract_test.go`
- [ ] T014 [P] [US1] Add integration coverage for first successful sync, multi-page retrieval, protected snapshot creation, and empty-history success in `tests/integration/sync_storage_flow_test.go`
- [ ] T015 [P] [US1] Add unit coverage for exact-decimal parsing and year-derivation rules in `tests/unit/decimal_test.go` and `tests/unit/year_derivation_test.go`
- [ ] T016 [P] [US1] Add unit coverage for envelope encoding, header authentication, and atomic snapshot replacement in `tests/unit/snapshot_envelope_test.go` and `tests/unit/snapshot_store_test.go`

### Implementation for User Story 1

- [ ] T017 [P] [US1] Implement paginated Ghostfolio activities retrieval with `skip`, `take`, and ascending date order in `internal/ghostfolio/client/client.go`
- [ ] T018 [P] [US1] Implement Ghostfolio activity-to-normalized-record mapping in `internal/ghostfolio/mapper/activity_mapper.go`
- [ ] T019 [P] [US1] Implement chronological normalization, duplicate hashing, and available-year derivation in `internal/sync/normalize/activity_history.go`
- [ ] T020 [P] [US1] Implement supported-history validation for `BUY` and `SELL` activity rules in `internal/sync/validate/activity_history.go`
- [ ] T021 [P] [US1] Implement protected snapshot encryption, decryption, and atomic persistence in `internal/snapshot/store/encrypted_store.go`
- [ ] T022 [US1] Implement full sync orchestration from auth through protected write in `internal/app/runtime/sync_service.go`
- [ ] T023 [US1] Update sync flow busy-state lifecycle and result routing for full-history storage in `internal/tui/flow/sync_flow.go`
- [ ] T024 [US1] Replace validation-only sync entry and result screens with storage-focused wording in `internal/tui/screen/sync_validation_screen.go` and `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: User Story 1 is independently functional. The app can fetch full history and store a protected snapshot without exposing reporting workflows.

---

## Phase 4: User Story 2 - Reuse Token-Locked Stored Data (Priority: P1)

**Goal**: Reuse existing protected snapshots with the correct token, isolate different valid tokens into separate snapshots, and fail safely when tokens or stored-data versions are incompatible.

**Independent Test**: After one successful sync, rerunning `Sync Data` with the same token refreshes the existing snapshot only after success; a different valid token creates a new snapshot; invalid tokens and unsupported stored-data versions leave local data unchanged.

### Tests for User Story 2

- [ ] T025 [P] [US2] Add protected-snapshot discovery and compatibility contract coverage in `tests/contract/protected_snapshot_contract_test.go`
- [ ] T026 [P] [US2] Add integration coverage for same-token refresh, wrong-token denial, different-valid-token isolation, and invalid-token no-change behavior in `tests/integration/snapshot_reuse_flow_test.go`
- [ ] T027 [P] [US2] Add integration coverage for unsupported envelope version, unsupported payload version, and incompatible new sync data retention in `tests/integration/snapshot_compatibility_flow_test.go`
- [ ] T028 [P] [US2] Add unit coverage for server-scoped candidate filtering and payload version checks in `tests/unit/snapshot_discovery_test.go` and `tests/unit/stored_data_version_test.go`

### Implementation for User Story 2

- [ ] T029 [P] [US2] Implement server-scoped snapshot header discovery and candidate filtering in `internal/snapshot/store/discovery.go`
- [ ] T030 [P] [US2] Implement stored-data version compatibility checks for envelope and payload models in `internal/snapshot/model/version.go` and `internal/snapshot/store/compatibility.go`
- [ ] T031 [P] [US2] Implement protected snapshot unlock, active readable snapshot tracking, and isolated snapshot creation for new valid tokens in `internal/app/runtime/sync_service.go`
- [ ] T032 [US2] Implement failure and success result handling for rejected token, unsupported stored-data version, and incompatible new sync data in `internal/tui/flow/sync_flow.go` and `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: User Story 2 is independently functional. Snapshot reuse, token isolation, and compatibility failures behave safely without modifying protected data incorrectly.

---

## Phase 5: User Story 3 - Preserve Data Quality And Server Boundaries (Priority: P1)

**Goal**: Store only defensible normalized histories and prevent silent contamination or replacement when source data is unsupported or the selected server changes.

**Independent Test**: Unsupported activity types, invalid zero-price cases, unstable ordering, non-defensible histories, and declined server replacement all fail safely and preserve the previous readable snapshot.

### Tests for User Story 3

- [ ] T033 [P] [US3] Add normalization and defensibility contract coverage for supported activity rules, below-zero holdings rejection, and scope-reliability outcome categories in `tests/contract/activity_validation_contract_test.go`
- [ ] T034 [P] [US3] Add server-mismatch confirmation workflow contract coverage in `tests/contract/server_replacement_workflow_contract_test.go`
- [ ] T035 [P] [US3] Add integration coverage for unsupported activity history, duplicate removal, deterministic ordering, below-zero holdings rejection, and zero-price rule handling in `tests/integration/activity_validation_flow_test.go`
- [ ] T036 [P] [US3] Add integration coverage for server replacement confirm, cancel, success, and failed-replacement retention in `tests/integration/server_replacement_flow_test.go`
- [ ] T037 [P] [US3] Add unit coverage for duplicate hashing, tie-break ordering, running-quantity defensibility checks, and scope-reliability derivation in `tests/unit/activity_normalization_test.go` and `tests/unit/scope_reliability_test.go`

### Implementation for User Story 3

- [ ] T038 [P] [US3] Implement duplicate hashing, deterministic same-timestamp ordering, running-quantity replay support, and source-scope reliability derivation in `internal/sync/normalize/activity_history.go`
- [ ] T039 [P] [US3] Implement defensibility checks for missing or contradictory normalized fields, below-zero holdings, zero-priced `SELL` comment rules, and unsupported-history rejection in `internal/sync/validate/activity_history.go`
- [ ] T040 [P] [US3] Implement server-mismatch detection and replacement gating against the active readable snapshot in `internal/app/runtime/sync_service.go`
- [ ] T041 [US3] Implement server replacement confirmation screen and navigation in `internal/tui/screen/server_replacement_screen.go` and `internal/tui/flow/sync_flow.go`
- [ ] T042 [US3] Update the main menu and sync entry screens to surface protected-data-exists state without exposing cached activity details in `internal/tui/screen/main_menu_screen.go` and `internal/tui/screen/sync_validation_screen.go`

**Checkpoint**: User Story 3 is independently functional. Invalid histories and server-boundary changes cannot silently replace or contaminate protected data.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, release checks, security verification, and cross-story verification.

- [ ] T043 [P] Update protected-storage, removal, and no-reporting documentation in `README.md`
- [ ] T044 [P] Reconcile `specs/003-store-activity-data/quickstart.md` with the implemented sync result categories, persisted-artifact inspection steps, large-history performance verification steps, and verification commands in `specs/003-store-activity-data/quickstart.md`
- [ ] T045 [P] Document the OWASP Top 10 and Cryptographic Storage review, refreshed dependency and API research evidence, and the `SC-006` performance-verification evidence in `specs/003-store-activity-data/checklists/requirements.md`
- [ ] T049 [P] Add integration coverage that bootstrap files, protected snapshots, and persisted workflow artifacts never store Ghostfolio tokens, raw payload fragments, or transient sync-failure messages in `tests/integration/persistence_security_flow_test.go`
- [ ] T050 [P] Add deterministic large-history performance verification coverage for authenticated retrieval, normalization, validation, and protected replacement in `tests/integration/sync_performance_flow_test.go`
- [ ] T046 Run `make test`, `make coverage`, and the documented large-history performance verification, then verify the generated artifacts in `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks all story work.
- Phase 3, Phase 4, and Phase 5 depend on Phase 2.
- Phase 6 depends on the stories selected for release.

### Dependency Graph

```text
Phase 1 Setup
  -> Phase 2 Foundational
    -> US1 Sync And Store Full Activity History
      -> US2 Reuse Token-Locked Stored Data
      -> US3 Preserve Data Quality And Server Boundaries
        -> Phase 6 Polish

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
- Finish runtime orchestration before TUI messaging and navigation updates.
- Finish the story before moving to polish for that story.

### Parallel Opportunities

- T001, T047, and T048 can run in parallel at the start of Phase 1; T002 and T003 can run in parallel after T001.
- T005 through T010 can run in parallel once T004 defines the decimal primitives.
- T012 through T016 can run in parallel for US1, then T017 through T021 can run in parallel before T022 through T024.
- T025 through T028 can run in parallel for US2, then T029 and T030 can run in parallel before T031 and T032.
- T033 through T037 can run in parallel for US3, then T038 through T040 can run in parallel before T041 and T042.
- T043 through T045 and T049 through T050 can run in parallel once the release scope is stable.

---

## Parallel Example: User Story 1

```bash
Task: T012 Add Ghostfolio full-history contract coverage in tests/contract/ghostfolio_sync_storage_contract_test.go
Task: T013 Add sync workflow contract coverage in tests/contract/sync_storage_workflow_contract_test.go
Task: T014 Add first successful sync and empty-history integration coverage in tests/integration/sync_storage_flow_test.go
Task: T015 Add exact-decimal and year-derivation unit coverage in tests/unit/decimal_test.go and tests/unit/year_derivation_test.go
Task: T016 Add snapshot envelope and atomic-write unit coverage in tests/unit/snapshot_envelope_test.go and tests/unit/snapshot_store_test.go

Task: T017 Implement paginated Ghostfolio retrieval in internal/ghostfolio/client/client.go
Task: T018 Implement activity mapping in internal/ghostfolio/mapper/activity_mapper.go
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
