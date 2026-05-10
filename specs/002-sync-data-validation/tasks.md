---

description: "Task list for Sync Data Validation implementation"
---

# Tasks: Sync Data Validation

**Input**: Design documents from `/specs/002-sync-data-validation/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Bugfix**: 2026-05-10 — [BUG-001] Updated focused-input workflow coverage and reopened false-complete key-routing work.
**Bugfix**: 2026-05-10 — [BUG-002] Added paste-handling coverage and implementation tasks for focused setup and sync inputs.
**Bugfix**: 2026-05-10 — [BUG-003] Added persistent application-identity header coverage and implementation tasks.
**Bugfix**: 2026-05-10 — [BUG-004] Added maintained `Makefile` target and documentation-alignment tasks.

**Tests**: Automated tests are mandatory for this feature. Write each story's tests first, make them fail for the targeted behavior, and keep 100% statement coverage from `go test` plus 100% branch and file coverage for project-owned code with the `gocoverageplus` gate.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Requirement Traceability

- `FR-001` to `FR-005`, `FR-017` to `FR-021`, `FR-023` to `FR-025`, `SEC-005`, `QUAL-001`, `QUAL-002`, `QUAL-005`, `SC-001`, `SC-007`, and `SC-008` trace primarily to `T010` to `T016`, `T037`, `T038`, `T040`, `T041`, `T033`, and `T034`.
- `FR-006` to `FR-007`, `FR-014`, `FR-023` to `FR-025`, `QUAL-002`, `QUAL-004`, `QUAL-005`, `SC-005`, `SC-007`, and `SC-008` trace primarily to `T017` to `T022`, `T038`, `T039`, `T041`, and `T042`.
- `FR-008` to `FR-016`, `FR-022` to `FR-025`, `SEC-001` to `SEC-004`, `QUAL-001`, `QUAL-003`, `QUAL-005`, `INT-001`, and `SC-002` to `SC-008` trace primarily to `T023` to `T032`, `T039`, `T041`, and `T042`.
- `QUAL-006` and `SC-009` trace primarily to `T043` and `T044`.
- Release-level evidence for coverage and security review traces to `T035`, `T036`, and `T044`.

## Path Conventions

- Executable entrypoint: `cmd/ghostfolio-cryptogains/`
- App wiring and runtime orchestration: `internal/app/`
- Setup persistence: `internal/config/`
- Ghostfolio integration: `internal/ghostfolio/`
- Bubble Tea screens and flow: `internal/tui/`
- Secret-safe helpers: `internal/support/`
- Tool dependency pinning: `tools/`
- Automated tests: `tests/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Bootstrap the Go module, tool pinning, and the executable shell.

- [X] T001 Initialize the Go module, Bubble Tea dependencies, and module metadata in `go.mod` and `go.sum`
- [X] T002 Create the executable startup shell in `cmd/ghostfolio-cryptogains/main.go`
- [X] T003 [P] Pin the coverage gate helper with a build-tagged tools file in `tools/tools.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core runtime, config, shared TUI, and redaction infrastructure required by every story.

**Critical**: Finish this phase before starting story implementation.

- [X] T004 Implement runtime options and application assembly for config paths and development mode in `internal/app/bootstrap/options.go` and `internal/app/runtime/runtime.go`
- [X] T005 [P] Implement `AppSetupConfig` and origin canonicalization rules in `internal/config/model/app_setup_config.go`
- [X] T006 [P] Implement atomic JSON setup persistence and restrictive file permissions in `internal/config/store/store.go` and `internal/config/store/json_store.go`
- [X] T007 [P] Implement shared full-screen theme, layout, and hotkey help components in `internal/tui/component/theme.go`, `internal/tui/component/layout.go`, and `internal/tui/component/help.go`
- [X] T008 Implement the root Bubble Tea model and screen router in `internal/tui/flow/model.go`
- [X] T009 [P] Implement token-safe redaction helpers for diagnostics and transient status text in `internal/support/redact/redact.go`

**Checkpoint**: The application can start, locate bootstrap configuration, and host screen-specific workflows without business behavior yet.

---

## Phase 3: User Story 1 - Complete Initial Setup (Priority: P1) 🎯 MVP

**Goal**: Let a first-time user choose a Ghostfolio server, persist the bootstrap setup, and skip setup on the next launch.

**Independent Test**: On a fresh machine-local config directory, the user can complete setup with Ghostfolio Cloud or an allowed custom origin, restart the app, and land past setup without entering a token.

### Tests for User Story 1

- [X] T010 [P] [US1] Add setup-screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/setup_workflow_contract_test.go`
- [X] T011 [P] [US1] Add first-run setup completion, remembered-setup startup, invalid-remembered-setup startup fallback, setup-file removal-after-load behavior, and no-pre-sync-network integration coverage for clean and remembered bootstrap states in `tests/integration/setup_flow_test.go`
- [X] T012 [P] [US1] Add bootstrap config store, setup-file protection, startup-readable field validation, and origin validation unit coverage in `tests/unit/config_store_test.go`, `tests/unit/config_permissions_test.go`, and `tests/unit/origin_validator_test.go`
- [ ] T037 [P] [US1] Add focused custom-origin input contract and integration coverage for `Enter` return-to-menu behavior and paste-safe text entry in `tests/contract/setup_workflow_contract_test.go` and `tests/integration/setup_flow_test.go`

### Implementation for User Story 1

- [X] T013 [P] [US1] Implement the setup screen menu, labeled custom-origin input, and save gating in `internal/tui/screen/setup_screen.go`
- [X] T014 [US1] Implement setup workflow state transitions and validation messaging in `internal/tui/flow/setup_flow.go`
- [X] T015 [P] [US1] Implement startup bootstrap loading and incomplete-setup redirect logic in `internal/app/bootstrap/startup.go`
- [X] T016 [US1] Wire setup completion, persisted setup reuse, and edit-setup entry into `internal/tui/flow/model.go`
- [ ] T040 [US1] Implement focused custom-origin input `Enter` return-to-menu behavior and paste-safe input handling in `internal/tui/screen/setup_screen.go` and `internal/tui/flow/setup_flow.go`

**Checkpoint**: The setup path is independently runnable and remembered across launches.

---

## Phase 4: User Story 2 - Select Sync Data Feature (Priority: P1)

**Goal**: Present `Sync Data` as the only business workflow and prompt for the Ghostfolio token only when that workflow starts.

**Independent Test**: With a completed setup fixture, the user can open the main menu, see only `Sync Data` as the executable business action, enter the sync screen, and reach a masked token prompt without any reporting options.

### Tests for User Story 2

- [X] T017 [P] [US2] Add main-menu and sync-entry screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/main_menu_workflow_contract_test.go` and `tests/contract/sync_entry_workflow_contract_test.go`
- [X] T018 [P] [US2] Add main-menu, workflow-selection, and non-reporting-outcome integration coverage in `tests/integration/main_menu_flow_test.go`
- [ ] T019 [P] [US2] ⚠️ Reopened Add focus-aware key routing unit coverage for focused-input `Enter` release and paste-safe routing in `tests/unit/key_routing_test.go` (reopened — BUG-001)
- [ ] T038 [P] [US2] Add persistent application-identity header contract coverage for setup, main-menu, and sync-entry screens in `tests/contract/setup_workflow_contract_test.go`, `tests/contract/main_menu_workflow_contract_test.go`, and `tests/contract/sync_entry_workflow_contract_test.go`
- [ ] T039 [P] [US2] Add focused Ghostfolio security-token input contract and integration coverage for `Enter` return-to-menu behavior and paste-safe text entry in `tests/contract/sync_entry_workflow_contract_test.go` and `tests/integration/main_menu_flow_test.go`

### Implementation for User Story 2

- [X] T020 [P] [US2] Implement the main menu screen with `Sync Data` as the only business action in `internal/tui/screen/main_menu_screen.go`
- [X] T021 [P] [US2] Implement the sync validation entry screen with a masked token input and primary actions in `internal/tui/screen/sync_validation_screen.go`
- [X] T022 [US2] Implement main-menu, edit-setup, and sync-entry navigation in `internal/tui/flow/navigation.go`
- [ ] T041 [US2] Implement a persistent ASCII-safe application-identity header for setup, main-menu, and sync-entry screens in `internal/tui/component/layout.go`, `internal/tui/component/theme.go`, and the affected screen files
- [ ] T042 [US2] Implement focused Ghostfolio security-token input `Enter` return-to-menu behavior and paste-safe input handling in `internal/tui/screen/sync_validation_screen.go` and `internal/tui/flow/navigation.go`

**Checkpoint**: The user can reach the sync workflow entry point without storage or report-generation paths.

---

## Phase 5: User Story 3 - Validate Ghostfolio Communication (Priority: P1)

**Goal**: Authenticate against Ghostfolio, probe the activities endpoint, validate the minimal response contract, and show a success or failure result without persisting data.

**Independent Test**: With a reachable compatible server and valid token, `Sync Data` ends with a success message. With rejected auth, connectivity failure, non-2xx responses, or invalid payloads, it ends with a failure message and offers `Validate Again` without repeating setup.

### Tests for User Story 3

- [X] T023 [P] [US3] Add Ghostfolio auth and activities contract coverage from `specs/002-sync-data-validation/contracts/ghostfolio-sync-validation.md` in `tests/contract/ghostfolio_sync_validation_contract_test.go`
- [X] T024 [P] [US3] Add validation-result and busy-state screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/validation_result_workflow_contract_test.go`
- [X] T025 [P] [US3] Add sync validation success, categorized failure outcomes, retry after success and failure, no-persistence, abandoned-attempt, and in-flight resize responsiveness integration coverage in `tests/integration/sync_validation_flow_test.go`
- [X] T026 [P] [US3] Add payload validation, contradictory one-page probe validation, token-redaction, and failure-diagnostic coverage in `tests/unit/response_validator_test.go`, `tests/unit/redact_test.go`, and `tests/integration/diagnostic_redaction_test.go`

### Implementation for User Story 3

- [X] T027 [P] [US3] Implement Ghostfolio auth and activities probe DTOs in `internal/ghostfolio/dto/auth_response.go` and `internal/ghostfolio/dto/activities_probe_response.go`
- [X] T028 [P] [US3] Implement Ghostfolio response validation rules for auth and activities probes in `internal/ghostfolio/validator/response_validator.go`
- [X] T029 [US3] Implement the Ghostfolio client for anonymous auth and one-page activities probes in `internal/ghostfolio/client/client.go`
- [X] T030 [US3] Implement `GhostfolioSession`, `SyncValidationAttempt`, and `ValidationOutcome` orchestration with secret clearing in `internal/app/runtime/sync_service.go`
- [X] T031 [US3] Implement async busy-state transitions, in-flight resize handling, and retryable sync workflow behavior in `internal/tui/flow/sync_flow.go`
- [X] T032 [US3] Implement success and failure result screens with no-persistence messaging in `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: Ghostfolio communication validation works end to end and does not persist tokens, JWTs, or payloads.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, verification, and release-level checks across all stories.

- [X] T033 [P] Update the `README.md` sections `Local Setup Storage`, `Removing Local Setup`, `Development Mode`, and `Current Slice Scope`, including Linux, macOS, and Windows setup-file location expectations and protection notes
- [X] T034 [P] Reconcile the `Launch The Application`, `Remembered Setup Path`, `Sync Validation Failure Paths`, and `Negative Check: No Persistence Beyond Setup` sections in `specs/002-sync-data-validation/quickstart.md`, including the supported failure categories and invalid-remembered-setup behavior
- [X] T035 Run `mkdir -p dist/coverage && go test ./... -covermode=atomic -coverprofile=dist/coverage/coverage.out && gocoverageplus -i dist/coverage/coverage.out -o dist/coverage/coverage.xml`, then verify the generated artifacts in `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` report 100% statement coverage plus 100% branch and file coverage for project-owned code
- [X] T036 [P] Document the OWASP Top 10 review for setup persistence, Ghostfolio token handling, and Ghostfolio API calls in `specs/002-sync-data-validation/checklists/requirements.md`
- [ ] T043 [P] Add maintained `make run`, `make test`, and `make coverage` targets in `Makefile` for the slice's launch, test, and coverage workflows
- [ ] T044 [P] Update `README.md`, `specs/002-sync-data-validation/quickstart.md`, and Phase 6 verification references to use `make run`, `make test`, and `make coverage`, then verify coverage artifacts through the `make coverage` path

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
    -> US1 Complete Initial Setup
      -> US2 Select Sync Data Feature
        -> US3 Validate Ghostfolio Communication
          -> Phase 6 Polish

Parallel-capable after Phase 2:
  -> US2 with a completed setup fixture
  -> US3 with a completed setup fixture and Ghostfolio stub
```

### User Story Dependencies

- US1 depends only on Foundational work and is the smallest MVP increment.
- US2 depends only on Foundational work for isolated development by loading a completed setup fixture, but integrated delivery fits after US1.
- US3 depends only on Foundational work for stubbed development, but integrated delivery fits after US2 because the sync entry screen is its primary launch point.

### Within Each User Story

- Write the listed tests first and confirm they fail for the targeted behavior.
- Finish model or contract primitives before orchestration and screen wiring.
- Finish the story before moving to polish for that story.

### Parallel Opportunities

- T002 and T003 can run in parallel after T001.
- T005, T006, T007, and T009 can run in parallel once T004 establishes runtime expectations.
- T010, T011, and T012 can run in parallel for US1, then T013 and T015 can run in parallel before T014 and T016.
- T017, T018, T019, T037, and T038 can run in parallel for US1 and US2 coverage expansion, then T020, T021, T040, and T041 can run in parallel before T022 and T042.
- T023, T024, T025, and T026 can run in parallel for US3, then T027 and T028 can run in parallel before T029 through T032.
- T033, T034, T036, T043, and T044 can run in parallel once the released story set is stable.

---

## Parallel Example: User Story 1

```bash
Task: T010 Add setup-screen contract coverage in tests/contract/setup_workflow_contract_test.go
Task: T011 Add first-run setup integration coverage in tests/integration/setup_flow_test.go
Task: T012 Add config store and origin validation unit coverage in tests/unit/config_store_test.go and tests/unit/origin_validator_test.go

Task: T013 Implement the setup screen menu in internal/tui/screen/setup_screen.go
Task: T015 Implement startup bootstrap loading in internal/app/bootstrap/startup.go
```

## Parallel Example: User Story 2

```bash
Task: T017 Add main-menu and sync-entry contract coverage in tests/contract/main_menu_workflow_contract_test.go and tests/contract/sync_entry_workflow_contract_test.go
Task: T018 Add main-menu integration coverage in tests/integration/main_menu_flow_test.go
Task: T019 Add focus-aware key routing unit coverage in tests/unit/key_routing_test.go

Task: T020 Implement the main menu screen in internal/tui/screen/main_menu_screen.go
Task: T021 Implement the sync validation entry screen in internal/tui/screen/sync_validation_screen.go
```

## Parallel Example: User Story 3

```bash
Task: T023 Add Ghostfolio contract coverage in tests/contract/ghostfolio_sync_validation_contract_test.go
Task: T024 Add validation-result workflow contract coverage in tests/contract/validation_result_workflow_contract_test.go
Task: T025 Add sync validation integration coverage in tests/integration/sync_validation_flow_test.go
Task: T026 Add validator and redaction coverage in tests/unit/response_validator_test.go and tests/unit/redact_test.go

Task: T027 Implement Ghostfolio probe DTOs in internal/ghostfolio/dto/auth_response.go and internal/ghostfolio/dto/activities_probe_response.go
Task: T028 Implement Ghostfolio response validators in internal/ghostfolio/validator/response_validator.go
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1.
2. Complete Phase 2.
3. Complete Phase 3.
4. Validate the remembered-setup journey before expanding the workflow surface.

### Incremental Delivery

1. Deliver US1 so the application boots, persists setup, and gates workflow entry.
2. Deliver US2 so the main menu exposes only the sync-validation path.
3. Deliver US3 so the application reaches the first real Ghostfolio communication outcome.
4. Finish Phase 6 to lock documentation and verification.

### Parallel Team Strategy

1. One contributor owns runtime and config foundations while another builds shared TUI components.
2. After Phase 2, one contributor can build setup flow, one can build main-menu and sync-entry screens against fixtures, and one can build the Ghostfolio client and validators against `httptest` stubs.
3. Merge at the flow-routing layer only after each story's tests pass independently.
