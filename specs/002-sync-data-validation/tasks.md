---

description: "Task list for Sync Data Validation implementation"
---

# Tasks: Sync Data Validation

**Input**: Design documents from `/specs/002-sync-data-validation/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature. Write each story's tests first, make them fail for the targeted behavior, and keep 100% statement coverage from `go test` plus 100% branch and file coverage for project-owned code with the `gocoverageplus` gate.

**Organization**: Tasks are grouped by user story so each story can be implemented and verified independently.

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

- [ ] T001 Initialize the Go module, Bubble Tea dependencies, and module metadata in `go.mod` and `go.sum`
- [ ] T002 Create the executable startup shell in `cmd/ghostfolio-cryptogains/main.go`
- [ ] T003 [P] Pin the coverage gate helper with a build-tagged tools file in `tools/tools.go`

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

**Checkpoint**: The application can start, locate bootstrap configuration, and host screen-specific workflows without business behavior yet.

---

## Phase 3: User Story 1 - Complete Initial Setup (Priority: P1) 🎯 MVP

**Goal**: Let a first-time user choose a Ghostfolio server, persist the bootstrap setup, and skip setup on the next launch.

**Independent Test**: On a fresh machine-local config directory, the user can complete setup with Ghostfolio Cloud or an allowed custom origin, restart the app, and land past setup without entering a token.

### Tests for User Story 1

- [ ] T010 [P] [US1] Add setup-screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/setup_workflow_contract_test.go`
- [ ] T011 [P] [US1] Add first-run setup completion, remembered-setup startup, and no-pre-sync-network integration coverage for clean and remembered bootstrap states in `tests/integration/setup_flow_test.go`
- [ ] T012 [P] [US1] Add bootstrap config store, setup-file protection, and origin validation unit coverage in `tests/unit/config_store_test.go`, `tests/unit/config_permissions_test.go`, and `tests/unit/origin_validator_test.go`

### Implementation for User Story 1

- [ ] T013 [P] [US1] Implement the setup screen menu, labeled custom-origin input, and save gating in `internal/tui/screen/setup_screen.go`
- [ ] T014 [US1] Implement setup workflow state transitions and validation messaging in `internal/tui/flow/setup_flow.go`
- [ ] T015 [P] [US1] Implement startup bootstrap loading and incomplete-setup redirect logic in `internal/app/bootstrap/startup.go`
- [ ] T016 [US1] Wire setup completion, persisted setup reuse, and edit-setup entry into `internal/tui/flow/model.go`

**Checkpoint**: The setup path is independently runnable and remembered across launches.

---

## Phase 4: User Story 2 - Select Sync Data Feature (Priority: P1)

**Goal**: Present `Sync Data` as the only business workflow and prompt for the Ghostfolio token only when that workflow starts.

**Independent Test**: With a completed setup fixture, the user can open the main menu, see only `Sync Data` as the executable business action, enter the sync screen, and reach a masked token prompt without any reporting options.

### Tests for User Story 2

- [ ] T017 [P] [US2] Add main-menu and sync-entry screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/main_menu_workflow_contract_test.go` and `tests/contract/sync_entry_workflow_contract_test.go`
- [ ] T018 [P] [US2] Add main-menu and workflow-selection integration coverage in `tests/integration/main_menu_flow_test.go`
- [ ] T019 [P] [US2] Add focus-aware key routing unit coverage in `tests/unit/key_routing_test.go`

### Implementation for User Story 2

- [ ] T020 [P] [US2] Implement the main menu screen with `Sync Data` as the only business action in `internal/tui/screen/main_menu_screen.go`
- [ ] T021 [P] [US2] Implement the sync validation entry screen with a masked token input and primary actions in `internal/tui/screen/sync_validation_screen.go`
- [ ] T022 [US2] Implement main-menu, edit-setup, and sync-entry navigation in `internal/tui/flow/navigation.go`

**Checkpoint**: The user can reach the sync workflow entry point without storage or report-generation paths.

---

## Phase 5: User Story 3 - Validate Ghostfolio Communication (Priority: P1)

**Goal**: Authenticate against Ghostfolio, probe the activities endpoint, validate the minimal response contract, and show a success or failure result without persisting data.

**Independent Test**: With a reachable compatible server and valid token, `Sync Data` ends with a success message. With rejected auth, connectivity failure, non-2xx responses, or invalid payloads, it ends with a failure message and offers `Validate Again` without repeating setup.

### Tests for User Story 3

- [ ] T023 [P] [US3] Add Ghostfolio auth and activities contract coverage from `specs/002-sync-data-validation/contracts/ghostfolio-sync-validation.md` in `tests/contract/ghostfolio_sync_validation_contract_test.go`
- [ ] T024 [P] [US3] Add validation-result and busy-state screen contract coverage from `specs/002-sync-data-validation/contracts/tui-workflows.md` in `tests/contract/validation_result_workflow_contract_test.go`
- [ ] T025 [P] [US3] Add sync validation success, failure, retry, no-persistence, and in-flight resize responsiveness integration coverage in `tests/integration/sync_validation_flow_test.go`
- [ ] T026 [P] [US3] Add payload validation, token-redaction, and failure-diagnostic coverage in `tests/unit/response_validator_test.go`, `tests/unit/redact_test.go`, and `tests/integration/diagnostic_redaction_test.go`

### Implementation for User Story 3

- [ ] T027 [P] [US3] Implement Ghostfolio auth and activities probe DTOs in `internal/ghostfolio/dto/auth_response.go` and `internal/ghostfolio/dto/activities_probe_response.go`
- [ ] T028 [P] [US3] Implement Ghostfolio response validation rules for auth and activities probes in `internal/ghostfolio/validator/response_validator.go`
- [ ] T029 [US3] Implement the Ghostfolio client for anonymous auth and one-page activities probes in `internal/ghostfolio/client/client.go`
- [ ] T030 [US3] Implement `GhostfolioSession`, `SyncValidationAttempt`, and `ValidationOutcome` orchestration with secret clearing in `internal/app/runtime/sync_service.go`
- [ ] T031 [US3] Implement async busy-state transitions, in-flight resize handling, and retryable sync workflow behavior in `internal/tui/flow/sync_flow.go`
- [ ] T032 [US3] Implement success and failure result screens with no-persistence messaging in `internal/tui/screen/validation_result_screen.go`

**Checkpoint**: Ghostfolio communication validation works end to end and does not persist tokens, JWTs, or payloads.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finish documentation, verification, and release-level checks across all stories.

- [ ] T033 [P] Update the `README.md` sections `Local Setup Storage`, `Removing Local Setup`, `Development Mode`, and `Current Slice Scope`
- [ ] T034 [P] Reconcile the `Launch The Application`, `Remembered Setup Path`, `Sync Validation Failure Paths`, and `Negative Check: No Persistence Beyond Setup` sections in `specs/002-sync-data-validation/quickstart.md`
- [ ] T035 Run `mkdir -p dist/coverage && go test ./... -covermode=atomic -coverprofile=dist/coverage/coverage.out && gocoverageplus -i dist/coverage/coverage.out -o dist/coverage/coverage.xml`, then verify the generated artifacts in `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` report 100% statement coverage plus 100% branch and file coverage for project-owned code
- [ ] T036 [P] Document the OWASP Top 10 review for setup persistence, Ghostfolio token handling, and Ghostfolio API calls in `specs/002-sync-data-validation/plan.md`

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
- T017, T018, and T019 can run in parallel for US2, then T020 and T021 can run in parallel before T022.
- T023, T024, T025, and T026 can run in parallel for US3, then T027 and T028 can run in parallel before T029 through T032.
- T033, T034, and T036 can run in parallel once the released story set is stable.

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
