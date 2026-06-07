---
description: "Task list for Empirical Solidified Financial Tests"
---

# Tasks: Empirical Solidified Financial Tests

**Input**: Design documents from `/specs/006-empirical-financial-tests/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature because the specification explicitly creates internal empirical validation infrastructure. Test tasks are listed before implementation tasks in each objective phase.

**Organization**: This specification intentionally has internal validation objectives instead of user-facing stories. The objectives are mapped to story labels for checklist traceability: `US1` = Maintain An Empirical External Dataset, `US2` = Produce An hledger-Backed Oracle, `US3` = Add Empirical Solidified Financial Integration Tests.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because the task touches different files and has no dependency on another incomplete task.
- **[Story]**: Maps the task to the internal validation objective phase.
- **File paths**: Every task includes exact repository paths to create, update, test, or verify.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the repository locations and documentation anchors needed by all empirical validation work.

- [X] T001 Create empirical directory skeleton at `testdata/empirical/golden/`, `testdata/empirical/hledger/`, `tests/empirical/`, `tests/empirical/fixture/`, `tools/empiricaloracle/`, `third_party/hledger/bin/`, and `third_party/hledger/source/`
- [X] T002 [P] Add empirical artifact operating notes in `testdata/empirical/README.md`
- [X] T003 [P] Add hledger vendoring compliance notes for complete source, supported executable artifact paths, checksums, platform support, and no binary-only vendoring in `third_party/hledger/README.md`
- [X] T004 [P] Add compilable empirical oracle command skeleton in `tools/empiricaloracle/main.go` and `tools/empiricaloracle/doc.go`
- [X] T005 [P] Add empirical test package documentation in `tests/empirical/doc.go` and `tests/empirical/fixture/doc.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Implement shared decimal-policy and fixture-helper foundations that all objective phases depend on.

**Critical**: No objective phase should begin until this phase is complete.

- [X] T006 [P] Add decimal policy configuration tests for the default production policy and documented accepted `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` values in `internal/support/math/rounding_internal_test.go`
- [X] T007 Implement decimal policy selection in `internal/support/math/decimal_policy.go` and update `internal/support/math/decimal_ops.go` to keep the 16-decimal round-half-up default when the environment variable is unset
- [X] T008 [P] Add shared empirical model tests for dataset, activity, case, oracle, and comparison structs in `tests/empirical/fixture/model_test.go`
- [X] T009 Implement shared empirical model structs in `tests/empirical/fixture/model.go`
- [X] T010 [P] Add decimal string parsing and canonicalization tests in `tests/empirical/fixture/decimal_test.go`
- [X] T011 Implement decimal string parsing and canonicalization helpers in `tests/empirical/fixture/decimal.go`
- [X] T012 [P] Add synthetic-content scanner tests for token, JWT, bearer, real-name, and copied-fixture patterns in `tests/empirical/fixture/security_test.go`
- [X] T013 Implement synthetic-content scanner helpers in `tests/empirical/fixture/security.go`

**Checkpoint**: Shared empirical helpers compile, decimal-policy behavior is covered, and objective work can begin.

---

## Phase 3: User Story 1 - Maintain An Empirical External Dataset (Priority: P1) MVP

**Goal**: Add a synthetic, human-readable empirical dataset that validates independently without hledger or project calculation execution.

**Independent Test**: `go test ./tests/empirical -run TestEmpiricalDatasetValidation -count=1 -v` parses and validates `testdata/empirical/financial-dataset.yaml`, confirms at least 150 activities across at least 3 source-calendar years, confirms required method and edge-case coverage, confirms deterministic ordering, confirms one currency, and rejects non-synthetic fixture content.

### Tests for User Story 1

- [X] T014 [P] [US1] Add dataset parser contract tests for top-level fields, activity fields, case fields, string-only decimals, scopes, and zero-priced reductions in `tests/empirical/fixture/dataset_parser_test.go`
- [X] T015 [P] [US1] Add dataset validation contract tests for activity count, year span, supported methods, deterministic source IDs, ordering metadata, single currency, and synthetic-only content in `tests/empirical/dataset_validation_test.go`
- [X] T016 [P] [US1] Add required coverage tag tests for every method and edge-case category from `specs/006-empirical-financial-tests/spec.md` in `tests/empirical/fixture/dataset_coverage_test.go`

### Implementation for User Story 1

- [X] T017 [US1] Implement the constrained project-owned YAML parser for `testdata/empirical/financial-dataset.yaml` in `tests/empirical/fixture/dataset_parser.go`
- [X] T018 [US1] Implement dataset validation rules for counts, years, methods, deterministic ordering, currency, zero-priced reductions, scopes, and synthetic-only content in `tests/empirical/fixture/dataset_validator.go`
- [X] T019 [US1] Implement required method and edge-case coverage validation in `tests/empirical/fixture/dataset_coverage.go`
- [X] T020 [US1] Populate `testdata/empirical/financial-dataset.yaml` with at least 150 synthetic activities across at least 3 source-calendar years covering FIFO, LIFO, HIFO, average cost, Scope-Local Hybrid (`scope_local_hybrid`), fees, gains, losses, zero-result liquidations, zero-priced reductions, same-date ordering, pre-year positions, in-year activity, after-year ignored activity, full liquidation followed by reacquisition, and assets excluded from selected-year main results
- [X] T021 [US1] Update `testdata/empirical/README.md` with the dataset schema fields, stable coverage tag index, synthetic-only policy, and read-only policy after this dataset-maintenance feature completes
- [X] T022 [US1] Wire `tests/empirical/dataset_validation_test.go` to load and validate `testdata/empirical/financial-dataset.yaml`
- [X] T023 [US1] Run `go test ./tests/empirical -run TestEmpiricalDatasetValidation -count=1 -v` for `tests/empirical/dataset_validation_test.go` and `testdata/empirical/financial-dataset.yaml`

**Checkpoint**: The dataset is independently valid and reviewable without hledger or project calculation output.

---

## Phase 4: User Story 2 - Produce An hledger-Backed Oracle (Priority: P2)

**Goal**: Add the vendored hledger boundary, generate hledger inputs from the empirical dataset, and persist normalized oracle golden fixtures with reproducibility metadata.

**Independent Test**: Running the oracle command against `testdata/empirical/financial-dataset.yaml` creates deterministic `testdata/empirical/hledger/` journals and `testdata/empirical/golden/` JSON fixtures with hledger version, command arguments, decimal policy, dataset hash, hledger input hash, oracle output hash, normalization version, supported methods, unsupported reasons, and no binary-only vendoring.

### Tests for User Story 2

- [ ] T024 [P] [US2] Add hledger vendoring contract tests for license text, source metadata, source checksum, supported executable artifact checksum, source presence, executable path, platform support, and runtime prohibition in `tests/empirical/hledger_vendoring_test.go`
- [ ] T025 [P] [US2] Add oracle fixture schema tests for metadata, decimal strings, tolerances, hashes, methods, years, matches, and unsupported segments in `tests/empirical/fixture/oracle_output_test.go`
- [ ] T026 [P] [US2] Add vendored hledger command wrapper tests for version detection, explicit file arguments, missing executable errors, unsupported version errors, and environment isolation in `tools/empiricaloracle/command_test.go`
- [ ] T027 [P] [US2] Add hledger journal rendering tests for acquisitions, liquidations, fees, zero-priced reductions, same-date ordering, scope evidence, and unsupported cases in `tools/empiricaloracle/journal_test.go`
- [ ] T028 [P] [US2] Add oracle normalization and stable hash tests for hledger output, dataset input hash, hledger input hash, oracle output hash, and normalization version in `tools/empiricaloracle/oracle_output_test.go`

### Implementation for User Story 2

- [ ] T029 [US2] Add GPL-compatible hledger license, source metadata, executable metadata, platform support notes, and checksums in `third_party/hledger/LICENSE`, `third_party/hledger/SOURCE.md`, and `third_party/hledger/README.md`
- [ ] T030 [US2] Add hledger complete corresponding source under `third_party/hledger/source/` and supported executable artifacts under `third_party/hledger/bin/<goos>-<goarch>/hledger`
- [ ] T031 [US2] Implement vendored hledger discovery from `third_party/hledger/bin/<goos>-<goarch>/hledger`, version capture, platform checks, explicit argument handling, and actionable setup errors in `tools/empiricaloracle/command.go`
- [ ] T032 [US2] Implement dataset-to-hledger journal rendering in `tools/empiricaloracle/journal.go`
- [ ] T033 [US2] Implement normalized oracle output JSON generation and stable hashing in `tools/empiricaloracle/oracle_output.go`
- [ ] T034 [US2] Implement explicit unsupported-segment detection and serialization in `tools/empiricaloracle/unsupported.go`
- [ ] T035 [US2] Implement CLI generation and explicit regeneration flow in `tools/empiricaloracle/main.go`
- [ ] T036 [US2] Generate hledger journal fixtures in `testdata/empirical/hledger/` from `testdata/empirical/financial-dataset.yaml`
- [ ] T037 [US2] Generate normalized golden fixtures for FIFO, LIFO, HIFO, average cost, and Scope-Local Hybrid (`scope_local_hybrid`) under `testdata/empirical/golden/`
- [ ] T038 [US2] Implement golden fixture loading and validation helpers in `tests/empirical/fixture/oracle_output.go`
- [ ] T039 [US2] Run `go test ./tools/empiricaloracle ./tests/empirical/fixture -run 'TestHledger|TestOracle|TestJournal' -count=1 -v` for `tools/empiricaloracle` and `tests/empirical/fixture/oracle_output_test.go`

**Checkpoint**: Oracle fixtures are persisted, reproducible, metadata-complete, and generated only through the documented vendored hledger boundary.

---

## Phase 5: User Story 3 - Add Empirical Solidified Financial Integration Tests (Priority: P3)

**Goal**: Add isolated empirical Go integration tests that translate the dataset into project calculation inputs and compare normalized pure calculation output to oracle fixtures for every supported method.

**Independent Test**: `go test ./tests/empirical -count=1 -v` loads existing golden fixtures, does not invoke hledger while fixtures are present, translates dataset records into calculation-layer inputs, calls `calculate.Calculate`, normalizes `CapitalGainsReport`, and reports deterministic non-secret comparison failures.

### Tests for User Story 3

- [ ] T040 [P] [US3] Add dataset-to-project translation tests for `syncmodel.ProtectedActivityCache`, activity ordering, scope reliability, selected currency context, and zero-priced holding reductions in `tests/empirical/fixture/project_translation_test.go`
- [ ] T041 [P] [US3] Add project calculation output normalization tests for realized gain or loss, allocated basis, closing quantity, closing basis, comparable full-liquidation effects, comparable matches, and reference-only assets in `tests/empirical/fixture/project_output_test.go`
- [ ] T042 [P] [US3] Add decimal comparator tests for exact quantity equality, capped per-field financial tolerances, selected decimal policy, difference formatting, and failure context in `tests/empirical/fixture/comparison_test.go`
- [ ] T043 [P] [US3] Add isolation boundary tests that reject Ghostfolio, TUI, snapshot, Markdown, report output writer, OS opener, filename, and Documents-path usage in `tests/empirical/isolation_test.go`
- [ ] T044 [P] [US3] Add fixture-backed empirical integration test skeleton for all supported methods and comparable cases in `tests/empirical/empirical_calculation_test.go`

### Implementation for User Story 3

- [ ] T045 [US3] Implement dataset-to-`syncmodel.ProtectedActivityCache` translation in `tests/empirical/fixture/project_translation.go`
- [ ] T046 [US3] Implement project calculation runner for every supported method, case, and report year in `tests/empirical/fixture/project_calculation.go`
- [ ] T047 [US3] Implement `reportmodel.CapitalGainsReport` normalization into project comparison output in `tests/empirical/fixture/project_output.go`
- [ ] T048 [US3] Implement decimal comparison, per-field tolerance handling, and non-secret failure formatting in `tests/empirical/fixture/comparison.go`
- [ ] T049 [US3] Implement hledger generation policy guard that skips execution when fixtures exist and permits generation only for missing fixtures in `tests/empirical/fixture/oracle_generation_policy.go`
- [ ] T050 [US3] Complete the empirical integration flow that validates dataset, loads fixtures, conditionally generates missing fixtures, runs project calculation, normalizes output, and compares every comparable case while reporting unsupported fields with reasons in `tests/empirical/empirical_calculation_test.go`
- [ ] T051 [US3] Implement static isolation assertions for forbidden package imports and forbidden output artifacts in `tests/empirical/isolation_test.go`
- [ ] T052 [US3] Run `go test ./tests/empirical -count=1 -v` for `tests/empirical/empirical_calculation_test.go`

**Checkpoint**: Empirical calculation tests run as an isolated supplemental suite and compare calculation output only.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, repository verification, formatting, coverage wiring, and fixture review.

- [ ] T053 [P] Update final empirical verification commands and oracle generation command in `specs/006-empirical-financial-tests/quickstart.md`
- [ ] T054 [P] Update final fixture names, comparability labels, unsupported-case policy, and hledger metadata examples in `specs/006-empirical-financial-tests/contracts/oracle-output.md`
- [ ] T055 [P] Update empirical isolation notes in `specs/006-empirical-financial-tests/contracts/empirical-tests.md`
- [ ] T056 Add `./tests/empirical` to the coverage test package list in `Makefile` while keeping `-coverpkg=$(PRODUCTION_PACKAGES)` unchanged
- [ ] T057 Run `gofmt` on Go files under `internal/support/math/`, `tests/empirical/`, and `tools/empiricaloracle/`
- [ ] T058 [P] Run synthetic and secret-content fixture review for `testdata/empirical/financial-dataset.yaml`, `testdata/empirical/golden/`, and `testdata/empirical/hledger/`
- [ ] T059 Run `make test` from the repository root for `Makefile`
- [ ] T060 Run `make coverage` from the repository root for `Makefile`
- [ ] T061 [P] Record OWASP Top 10 review evidence for empirical test infrastructure in `specs/006-empirical-financial-tests/quickstart.md`
- [ ] T062 [P] Verify fixture-backed empirical test runtime target of 30 seconds or less with `go test ./tests/empirical -count=1 -v` and document the observed result in `specs/006-empirical-financial-tests/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all objective phases.
- **US1 Dataset (Phase 3)**: Depends on Foundational completion.
- **US2 Oracle (Phase 4)**: Depends on US1 because hledger journals and golden fixtures are generated from the validated dataset.
- **US3 Empirical Tests (Phase 5)**: Depends on US1 and US2 because comparisons require the validated dataset and golden fixtures.
- **Polish (Phase 6)**: Depends on all selected objective phases.

### Objective Dependency Graph

```text
Setup -> Foundational -> US1 Dataset -> US2 Oracle -> US3 Empirical Tests -> Polish
```

### Parallel Opportunities

- T002, T003, T004, and T005 can run in parallel after T001 creates directories.
- T006, T008, T010, and T012 can run in parallel because they add tests in different files.
- T014, T015, and T016 can run in parallel once Phase 2 is complete.
- T024, T025, T026, T027, and T028 can run in parallel once US1 is complete.
- T040, T041, T042, T043, and T044 can run in parallel once US2 is complete.
- T053, T054, T055, T058, T061, and T062 can run in parallel during Polish.

---

## Parallel Example: User Story 1

```bash
Task: "Add dataset parser contract tests for top-level fields, activity fields, case fields, string-only decimals, scopes, and zero-priced reductions in tests/empirical/fixture/dataset_parser_test.go"
Task: "Add dataset validation contract tests for activity count, year span, supported methods, deterministic source IDs, ordering metadata, single currency, and synthetic-only content in tests/empirical/dataset_validation_test.go"
Task: "Add required coverage tag tests for every method and edge-case category from specs/006-empirical-financial-tests/spec.md in tests/empirical/fixture/dataset_coverage_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "Add hledger vendoring contract tests for license text, source metadata, checksum, source presence, platform support, and runtime prohibition in tests/empirical/hledger_vendoring_test.go"
Task: "Add oracle fixture schema tests for metadata, decimal strings, tolerances, hashes, methods, years, matches, and unsupported segments in tests/empirical/fixture/oracle_output_test.go"
Task: "Add vendored hledger command wrapper tests for version detection, explicit file arguments, missing executable errors, unsupported version errors, and environment isolation in tools/empiricaloracle/command_test.go"
Task: "Add hledger journal rendering tests for acquisitions, liquidations, fees, zero-priced reductions, same-date ordering, scope evidence, and unsupported cases in tools/empiricaloracle/journal_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "Add dataset-to-project translation tests for syncmodel.ProtectedActivityCache, activity ordering, scope reliability, selected currency context, and zero-priced holding reductions in tests/empirical/fixture/project_translation_test.go"
Task: "Add project calculation output normalization tests for realized gain or loss, allocated basis, closing quantity, closing basis, matches, and reference-only assets in tests/empirical/fixture/project_output_test.go"
Task: "Add decimal comparator tests for exact quantity equality, financial tolerances, selected decimal policy, difference formatting, and failure context in tests/empirical/fixture/comparison_test.go"
Task: "Add isolation boundary tests that reject Ghostfolio, TUI, snapshot, Markdown, report output writer, OS opener, filename, and Documents-path usage in tests/empirical/isolation_test.go"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: US1 Dataset.
4. Stop and validate with `go test ./tests/empirical -run TestEmpiricalDatasetValidation -count=1 -v`.

### Incremental Delivery

1. Deliver US1 so the dataset is validated independently.
2. Deliver US2 so hledger inputs and golden fixtures are reproducible from the dataset.
3. Deliver US3 so project calculation output is compared to oracle fixtures.
4. Run Polish verification with `go test ./tests/empirical -count=1 -v`, `make test`, and `make coverage`.

### Parallel Team Strategy

1. Complete Setup and Foundational phases first.
2. Assign parallel test-writing tasks inside each objective phase before implementation tasks.
3. Keep phase order sequential because the oracle depends on the dataset and the empirical comparison suite depends on both dataset and golden fixtures.

---

## Notes

- Keep hledger as a separate test-time command. Do not import, link, or execute it from runtime application code.
- Keep empirical artifacts synthetic. Do not add real tokens, JWTs, user activity, real account names, wallet names, proprietary financial records, raw protected snapshots, generated Markdown reports, TUI text, output filenames, or Documents paths.
- Use `apd.Decimal` through existing decimal helpers. Do not introduce floating-point math in dataset parsing, oracle normalization, or comparison code.
- Treat `testdata/empirical/financial-dataset.yaml` as read-only after this dataset-maintenance feature is complete.
- Add required OpenCode authoring documentation to new Go package, type, and function comments when implementing these tasks.
