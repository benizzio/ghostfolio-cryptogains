---

description: "Task list for Sync Data Validation implementation"
---

# Tasks: Sync Data Validation

**Input**: Design documents from `/specs/002-sync-data-validation/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature. Write the story tests first, make them fail for the targeted behavior, and keep 100% coverage for project-owned code with `go test` plus the `gocoverageplus` gate.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

## Path Conventions

- Executable entrypoint: `cmd/ghostfolio-cryptogains/`
- App wiring and runtime orchestration: `internal/app/`
- Setup persistence: `internal/config/`
- Ghostfolio integration: `internal/ghostfolio/`
- Bubble Tea screens and flow: `internal/tui/`
- Secret-safe helpers: `internal/support/`
- Automated tests: `tests/`

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Bootstrap the Go module and the executable shell.

- [ ] T001 Initialize the Go module and Bubble Tea dependencies in `go.mod`
- [ ] T002 Create the executable startup shell in `cmd/ghostfolio-cryptogains/main.go`
- [ ] T003 [P] Document the local Go run, test, and coverage commands in `README.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core runtime, config, shared TUI, and redaction infrastructure required by every story.

**Critical**: Finish this phase before starting story implementation.

- [ ] T004 Implement runtime options and application assembly for config paths and development mode in `internal/app/bootstrap/options.go` and `internal/app/runtime/runtime.go`
- [ ] T005 [P] Implement `AppSetupConfig` and origin canonicalization rules in `internal/config/model/app_setup_config.go`
- [ ] T006 [P] Implement atomic JSON setup persistence and restrictive file permissions in `internal/config/store/store.go` and `internal/config/store/json_store.go`
- [ ] T007 [P] Implement shared full-screen theme, layout, and hotkey help components in `internal/tui/component/theme.go`, `internal/tui/component/layout.go`, and `internal/tui/component/help.go`
- [ ] T008 Implement the root Bubble Tea model and screen router in `internal/tui/flow/model.go`
- [ ] T009 [P] Implement token-safe redaction helpers for diagnostics and transient status text in `internal/support/redact/redact.go`

**Checkpoint**: The application can start, locate bootstrap configuration, and host screen-specific workflows without any business behavior yet.

---

## Phase 3: User Story 1 - Complete Initial Setup (Priority: P1)

**Goal**: Let a first-time user choose a Ghostfolio server, persist the bootstrap setup, and skip setup on the next launch.

**Independent Test**: On a fresh machine-local config directory, the user can complete setup with Ghostfolio Cloud or an allowed custom origin, restart the app, and land past setup without entering a token.

### Tests for User Story 1

- [ ] T010 [P] [US1] Add first-run and remembered-setup integration coverage in `tests/integration/setup_flow_test.go`
- [ ] T011 [P] [US1] Add bootstrap config store and origin validation unit coverage in `tests/unit/config_store_test.go` and `tests/unit/origin_validator_test.go`

### Implementation for User Story 1

- [ ] T012 [P] [US1] Implement the setup screen menu, labeled custom-origin input, and save gating in `internal/tui/screen/setup_screen.go`
- [ ] T013 [US1] Implement setup workflow state transitions and validation messaging in `internal/tui/flow/setup_flow.go`
- [ ] T014 [P] [US1] Implement startup bootstrap loading and incomplete-setup redirect logic in `internal/app/bootstrap/startup.go`
- [ ] T015 [US1] Wire setup completion, persisted setup reuse, and edit-setup entry into `internal/tui/flow/model.go`

**Checkpoint**: The setup path is independently runnable and remembered across launches.

---

## Phase 4: User Story 2 - Select Sync Data Feature (Priority: P1)

**Goal**: Present `Sync Data` as the only business workflow and prompt for the Ghostfolio token only when that workflow starts.

**Independent Test**: With a completed setup fixture, the user can open the main menu, see only `Sync Data` as the executable business action, enter the sync screen, and reach a masked token prompt without any reporting options.

### Tests for User Story 2

- [ ] T016 [P] [US2] Add main-menu and workflow-selection integration coverage in `tests/integration/main_menu_flow_test.go`
- [ ] T017 [P] [US2] Add focus-aware key routing unit coverage in `tests/unit/key_routing_test.go`

### Implementation for User Story 2

- [ ] T018 [P] [US2] Implement the main menu screen with `Sync Data` as the only business action in `internal/tui/screen/main_menu_screen.go`
- [ ] T019 [P] [US2] Implement the sync validation entry screen with a masked token input and primary actions in `internal/tui/screen/sync_validation_screen.go`
- [ ] T020 [US2] Implement main-menu, edit-setup, and sync-entry navigation in `internal/tui/flow/navigation.go`

**Checkpoint**: The user can reach the sync workflow entry point without any storage or report-generation paths.

---

## Phase 5: User Story 3 - Validate Ghostfolio Communication (Priority: P1)

**Goal**: Authenticate against Ghostfolio, probe the activities endpoint, validate the minimal response contract, and show a success or failure result without persisting data.

**Independent Test**: With a reachable compatible server and valid token, `Sync Data` ends with a success message. With rejected auth, connectivity failure, non-2xx responses, or invalid payloads, it ends with a failure message and offers `Validate Again`.

### Tests for User Story 3

- [ ] T021 [P] [US3] Add Ghostfolio auth and activities contract coverage in `tests/contract/ghostfolio_sync_validation_contract_test.go`
- [ ] T022 [P] [US3] Add sync validation success, failure, and retry integration coverage in `tests/integration/sync_validation_flow_test.go`
- [ ] T023 [P] [US3] Add payload validation and secret-redaction unit coverage in `tests/unit/response_validator_test.go` and `tests/unit/redact_test.go`

### Implementation for User Story 3

- [ ] T024 [P] [US3] Implement Ghostfolio auth and activities probe DTOs in `internal/ghostfolio/dto/auth_response.go` and `internal/ghostfolio/dto/activities_probe_response.go`
- [ ] T025 [P] [US3] Implement Ghostfolio response validation rules for auth and activities probes in `internal/ghostfolio/validator/response_validator.go`
- [ ] T026 [US3] Implement the Ghostfolio client for anonymous auth and one-page activities probes in `internal/ghostfolio/client/client.go`
- [ ] T027 [US3] Implement sync validation attempt orchestration and secret clearing in `internal/app/runtime/sync_service.go`
- [ ] T028 [US3] Implement busy-state transitions and retryable sync workflow behavior in `internal/tui/flow/sync_flow.go`
- [ ] T029 [US3] Implement success and failure result screens with no-persistence messaging in `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: Ghostfolio communication validation works end to end and does not persist tokens, JWTs, or payloads.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, verification, and release-level checks across all stories.

- [ ] T030 [P] Update the operator and contributor guidance for local setup storage, development mode, and sync-only scope in `README.md`
- [ ] T031 [P] Reconcile manual verification steps with the implemented screens and commands in `specs/002-sync-data-validation/quickstart.md`
- [ ] T032 Run `go test ./...`, generate the coverage profile in `dist/coverage/coverage.out`, and verify the `gocoverageplus` gate against `dist/coverage/coverage.out`

---

## Dependencies & Execution Order

### Phase Dependencies

- Phase 1 has no dependencies.
- Phase 2 depends on Phase 1 and blocks all story work.
- Phase 3, Phase 4, and Phase 5 depend on Phase 2.
- Phase 6 depends on the stories selected for release.

### User Story Dependencies

- US1 depends only on Foundational work and is the smallest MVP increment.
- US2 depends only on Foundational work for isolated development by loading a completed setup fixture, but integrated delivery fits after US1.
- US3 depends only on Foundational work for stubbed development, but integrated delivery fits after US2 because the sync entry screen is its primary launch point.

### Suggested Completion Order

- US1 -> US2 -> US3

### Within Each User Story

- Write the listed tests first and confirm they fail for the targeted behavior.
- Finish model or contract primitives before orchestration and screen wiring.
- Finish the story before moving to polish for that story.

### Parallel Opportunities

- T003 can run while T001 and T002 are in progress.
- T005, T006, T007, and T009 can run in parallel after T004 defines runtime options.
- T010 and T011 can run in parallel for US1, then T012 and T014 can run in parallel before T013 and T015.
- T016 and T017 can run in parallel for US2, then T018 and T019 can run in parallel before T020.
- T021, T022, and T023 can run in parallel for US3, then T024 and T025 can run in parallel before T026 through T029.
- T030 and T031 can run in parallel once the released story set is stable.

---

## Parallel Example: User Story 1

```bash
Task: T010 Add first-run and remembered-setup integration coverage in tests/integration/setup_flow_test.go
Task: T011 Add bootstrap config store and origin validation unit coverage in tests/unit/config_store_test.go and tests/unit/origin_validator_test.go

Task: T012 Implement the setup screen menu, labeled custom-origin input, and save gating in internal/tui/screen/setup_screen.go
Task: T014 Implement startup bootstrap loading and incomplete-setup redirect logic in internal/app/bootstrap/startup.go
```

## Parallel Example: User Story 2

```bash
Task: T016 Add main-menu and workflow-selection integration coverage in tests/integration/main_menu_flow_test.go
Task: T017 Add focus-aware key routing unit coverage in tests/unit/key_routing_test.go

Task: T018 Implement the main menu screen with Sync Data as the only business action in internal/tui/screen/main_menu_screen.go
Task: T019 Implement the sync validation entry screen with a masked token input and primary actions in internal/tui/screen/sync_validation_screen.go
```

## Parallel Example: User Story 3

```bash
Task: T021 Add Ghostfolio auth and activities contract coverage in tests/contract/ghostfolio_sync_validation_contract_test.go
Task: T022 Add sync validation success, failure, and retry integration coverage in tests/integration/sync_validation_flow_test.go
Task: T023 Add payload validation and secret-redaction unit coverage in tests/unit/response_validator_test.go and tests/unit/redact_test.go

Task: T024 Implement Ghostfolio auth and activities probe DTOs in internal/ghostfolio/dto/auth_response.go and internal/ghostfolio/dto/activities_probe_response.go
Task: T025 Implement Ghostfolio response validation rules for auth and activities probes in internal/ghostfolio/validator/response_validator.go
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
