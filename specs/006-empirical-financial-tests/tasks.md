---
description: "Task list for Empirical Solidified Financial Tests"
---

# Tasks: Empirical Solidified Financial Tests

**Input**: Design documents from `/specs/006-empirical-financial-tests/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Automated tests are mandatory for this feature because the specification explicitly creates internal empirical validation infrastructure. Test tasks are listed before implementation tasks in each objective phase.

**Organization**: This specification intentionally has internal validation objectives instead of user-facing stories. The objectives are mapped to story labels for checklist traceability: `US1` = Maintain An Empirical External Dataset, `US2` = Produce An ~~hledger-Backed~~ External Oracle, `US3` = Add Empirical Solidified Financial Integration Tests.

**Bugfix**: 2026-06-10 — [BUG-001] Updated from bugfix patch, including explicit repository-controlled rotki-boundary bootstrap tasks.

**Bugfix**: 2026-06-12 — [BUG-002] Updated from bugfix patch, including untracked verified rotki source regeneration tasks and reopened raw-output shortcut completions.

**Bugfix**: 2026-06-13 — [BUG-003] Updated from bugfix patch, including hledger boundary removal tasks and reopened hledger-retention completions.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because the task touches different files and has no dependency on another incomplete task.
- **[Story]**: Maps the task to the internal validation objective phase.
- **File paths**: Every task includes exact repository paths to create, update, test, or verify.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create the repository locations and documentation anchors needed by all empirical validation work.

- [X] T001 ⚠️ Reopened (reopened — BUG-003): ~~Create empirical directory skeleton at `testdata/empirical/golden/`, `testdata/empirical/hledger/`, `tests/empirical/`, `tests/empirical/fixture/`, `tools/empiricaloracle/`, `third_party/hledger/bin/`, and `third_party/hledger/source/`~~ Remove obsolete hledger setup paths from the empirical skeleton and keep only active empirical dataset, golden fixture, `tests/empirical/`, `tools/empiricaloracle/`, and rotki provenance/cache paths
- [X] T002 [P] Add empirical artifact operating notes in `testdata/empirical/README.md`
- [X] T003 [P] ⚠️ Reopened (reopened — BUG-003): ~~Add hledger vendoring compliance notes for complete source, supported executable artifact paths, checksums, platform support, and no binary-only vendoring in `third_party/hledger/README.md`~~ Remove hledger vendoring compliance notes and active `third_party/hledger/README.md` references; keep only historical bug report references or strikethrough rationale
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

**Goal**: Add a synthetic, human-readable empirical dataset that validates independently without ~~hledger~~ external oracle generation or project calculation execution.

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

**Checkpoint**: The dataset is independently valid and reviewable without ~~hledger~~ external oracle generation or project calculation output.

---

## Phase 4: User Story 2 - Produce An ~~hledger-Backed~~ External Oracle (Priority: P2)

**Goal**: Add the ~~vendored hledger boundary, generate hledger inputs~~ external oracle boundary, generate external-oracle inputs from the empirical dataset, and persist normalized oracle golden fixtures with reproducibility metadata.

**Independent Test**: Running the oracle command against `testdata/empirical/financial-dataset.yaml` creates deterministic ~~`testdata/empirical/hledger/` journals~~ external-oracle inputs and `testdata/empirical/golden/` JSON fixtures with external oracle name, version or commit identity, adapter arguments, decimal policy, dataset hash, external-oracle input hash, oracle output hash, normalization version, supported methods, unsupported reasons, and no prohibited binary-only vendoring.

### Tests for User Story 2

- [X] T024 [P] [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add hledger vendoring contract tests for license text, source metadata, source checksum, supported executable artifact checksum, source presence, executable path, platform support, and runtime prohibition in `tests/empirical/hledger_vendoring_test.go`~~ Remove hledger vendoring contract tests and keep only rotki/composite-oracle boundary checks required by the current oracle model
- [X] T025 [P] [US2] Add oracle fixture schema tests for metadata, decimal strings, tolerances, hashes, methods, years, matches, and unsupported segments in `tests/empirical/fixture/oracle_output_test.go`
- [X] T026 [P] [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add vendored hledger command wrapper tests for version detection, explicit file arguments, missing executable errors, unsupported version errors, and environment isolation in `tools/empiricaloracle/command_test.go`~~ Remove hledger command wrapper tests and verify no hledger command wrapper remains in `tools/empiricaloracle/`
- [X] T027 [P] [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add hledger journal rendering tests for acquisitions, liquidations, fees, zero-priced reductions, same-date ordering, scope evidence, and unsupported cases in `tools/empiricaloracle/journal_test.go`~~ Remove hledger journal rendering tests and any active journal-rendering fixtures from `tools/empiricaloracle/`
- [X] T028 [P] [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add oracle normalization and stable hash tests for hledger output, dataset input hash, hledger input hash, oracle output hash, and normalization version in `tools/empiricaloracle/oracle_output_test.go`~~ Remove hledger-output normalization expectations and keep stable hash tests tied to rotki/composite oracle outputs

### Implementation for User Story 2

- [X] T029 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add GPL-compatible hledger license, source metadata, executable metadata, platform support notes, and checksums in `third_party/hledger/LICENSE`, `third_party/hledger/SOURCE.md`, and `third_party/hledger/README.md`~~ Remove obsolete hledger license, source metadata, executable metadata, platform notes, and checksums from active project artifacts
- [X] T030 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Add hledger complete corresponding source under `third_party/hledger/source/` and supported executable artifacts under `third_party/hledger/bin/<goos>-<goarch>/hledger`~~ Remove obsolete hledger source and executable artifacts from `third_party/hledger/`
- [X] T031 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Implement vendored hledger discovery from `third_party/hledger/bin/<goos>-<goarch>/hledger`, version capture, platform checks, explicit argument handling, and actionable setup errors in `tools/empiricaloracle/command.go`~~ Remove hledger discovery and command execution code from `tools/empiricaloracle/command.go`
- [X] T032 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Implement dataset-to-hledger journal rendering in `tools/empiricaloracle/journal.go`~~ Remove dataset-to-hledger journal rendering code from `tools/empiricaloracle/journal.go`
- [X] T033 [US2] Implement normalized oracle output JSON generation and stable hashing in `tools/empiricaloracle/oracle_output.go`
- [X] T034 [US2] Implement explicit unsupported-segment detection and serialization in `tools/empiricaloracle/unsupported.go`
- [X] T035 [US2] Implement CLI generation and explicit regeneration flow in `tools/empiricaloracle/main.go`
- [X] T036 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Generate hledger journal fixtures in `testdata/empirical/hledger/` from `testdata/empirical/financial-dataset.yaml`~~ Remove generated hledger journal fixtures from `testdata/empirical/hledger/`
- [X] T037 [US2] ⚠️ Reopened (reopened — BUG-002): Generate normalized golden fixtures for FIFO, LIFO, HIFO, average cost, and Scope-Local Hybrid (`scope_local_hybrid`) under `testdata/empirical/golden/`; ~~hledger-only fixtures~~ committed raw rotki outputs are insufficient unless fixtures are regenerated from verified pinned rotki source execution through the local adapter boundary
- [X] T038 [US2] Implement golden fixture loading and validation helpers in `tests/empirical/fixture/oracle_output.go`
- [X] T039 [US2] ⚠️ Reopened (reopened — BUG-003): ~~Run `go test ./tools/empiricaloracle ./tests/empirical/fixture -run 'TestHledger|TestOracle|TestJournal' -count=1 -v` for `tools/empiricaloracle` and `tests/empirical/fixture/oracle_output_test.go`~~ Run updated oracle tests that exclude hledger and journal-rendering expectations for `tools/empiricaloracle` and `tests/empirical/fixture/oracle_output_test.go`

**Checkpoint**: Oracle fixtures are persisted, reproducible, metadata-complete, and generated only through the documented ~~vendored hledger~~ external oracle boundary.

---

## Phase 5: User Story 3 - Add Empirical Solidified Financial Integration Tests (Priority: P3)

**Goal**: Add isolated empirical Go integration tests that translate the dataset into project calculation inputs and compare normalized pure calculation output to oracle fixtures for every supported method.

**Independent Test**: `go test ./tests/empirical -count=1 -v` loads existing golden fixtures, does not invoke ~~hledger~~ external oracle generation while fixtures are present, translates dataset records into calculation-layer inputs, calls `calculate.Calculate`, normalizes `CapitalGainsReport`, and reports deterministic non-secret comparison failures.

### Tests for User Story 3

- [X] T040 [P] [US3] Add dataset-to-project translation tests for `syncmodel.ProtectedActivityCache`, activity ordering, scope reliability, selected currency context, and zero-priced holding reductions in `tests/empirical/fixture/project_translation_test.go`
- [X] T041 [P] [US3] Add project calculation output normalization tests for realized gain or loss, allocated basis, closing quantity, closing basis, comparable full-liquidation effects, comparable matches, and reference-only assets in `tests/empirical/fixture/project_output_test.go`
- [X] T042 [P] [US3] Add decimal comparator tests for exact quantity equality, capped per-field financial tolerances, selected decimal policy, difference formatting, and failure context in `tests/empirical/fixture/comparison_test.go`
- [X] T043 [P] [US3] Add isolation boundary tests that reject Ghostfolio, TUI, snapshot, Markdown, report output writer, OS opener, filename, and Documents-path usage in `tests/empirical/isolation_test.go`
- [X] T044 [P] [US3] Add fixture-backed empirical integration test skeleton for all supported methods and comparable cases in `tests/empirical/empirical_calculation_test.go`

### Implementation for User Story 3

- [X] T045 [US3] Implement dataset-to-`syncmodel.ProtectedActivityCache` translation in `tests/empirical/fixture/project_translation.go`
- [X] T046 [US3] Implement project calculation runner for every supported method, case, and report year in `tests/empirical/fixture/project_calculation.go`
- [X] T047 [US3] Implement `reportmodel.CapitalGainsReport` normalization into project comparison output in `tests/empirical/fixture/project_output.go`
- [X] T048 [US3] Implement decimal comparison, per-field tolerance handling, and non-secret failure formatting in `tests/empirical/fixture/comparison.go`
- [X] T049 [US3] ⚠️ Reopened (reopened — BUG-003): ~~Implement hledger generation policy guard that skips execution when fixtures exist and permits generation only for missing fixtures in `tests/empirical/fixture/oracle_generation_policy.go`~~ Replace hledger-specific generation policy wording with rotki/composite-oracle fixture generation policy in `tests/empirical/fixture/oracle_generation_policy.go`
- [X] T050 [US3] ⚠️ Reopened (reopened — BUG-002): Complete the empirical integration flow that validates dataset, loads fixtures, conditionally generates missing fixtures only through the verified untracked rotki source boundary, runs project calculation, normalizes output, and compares every comparable supported case while reporting unsupported field-level segments with reasons in `tests/empirical/empirical_calculation_test.go`
- [X] T051 [US3] Implement static isolation assertions for forbidden package imports and forbidden output artifacts in `tests/empirical/isolation_test.go`
- [X] T052 [US3] ⚠️ Reopened (reopened — BUG-002): Run `go test ./tests/empirical -count=1 -v` for `tests/empirical/empirical_calculation_test.go` and verify supported fixture groups do not skip before project calculation and oracle comparison, and fixture-backed runs do not download rotki or invoke oracle generation

**Checkpoint**: Empirical calculation tests run as an isolated supplemental suite and compare calculation output only.

---

## Phase 6: BUG-001/BUG-002 Oracle Remediation (Blocking Before Polish Acceptance)

**Purpose**: Replace the ~~hledger-only~~ empirical oracle acceptance path and BUG-001 raw-output shortcut, restore comparison breadth, require verified untracked pinned rotki source execution for regeneration, and fail on unexpected supported fixture skips.

**Dependencies**: T064 before T067; T065 before T073 and T075; T075 before T076; T076 before T077, T078, and T080; T077 before T066; T073 before T074; T066, T074, and T078 before T067, T068, T069, T070, and T079; T067, T068, T069, T070, T071, T072, T074, T078, T079, and T080 before reopened T052, Phase 7 hledger removal, and Phase 8 polish verification.

- [X] T064 [US2] Remove zero-priced holding reduction cases from empirical external-oracle dataset scope, generated oracle inputs, golden fixtures, and empirical covered-case expectations while preserving zero-priced behavior coverage in non-oracle unit, integration, or contract tests across `testdata/empirical/financial-dataset.yaml`, `testdata/empirical/golden/`, ~~`testdata/empirical/hledger/`,~~ `tests/empirical/`, `tests/unit/`, and `tests/integration/`
- [X] T065 [P] [US2] ⚠️ Reopened (reopened — BUG-002): Create and document the non-vendored rotki source boundary policy, license text, source provenance, pinned version or commit, checksums, adapter constraints, platform support, untracked source directory, and ~~hledger-only~~ oracle supersession in `third_party/rotki/README.md` and `specs/006-empirical-financial-tests/research.md`
- [X] T073 [US2] ⚠️ Reopened (reopened — BUG-002): Replace repository-controlled raw rotki oracle evidence under `third_party/rotki/` and `testdata/empirical/rotki/` with committed project-owned provenance metadata and normalized golden fixtures only, so BUG-002 remediation does not depend on committed raw rotki outputs, hand-authored rotki datasets, developer-local rotki installation, or vendored rotki source
- [X] T074 [P] [US2] ⚠️ Reopened (reopened — BUG-002): Add boundary verification tests or checks that fail with an actionable setup error when the required verified untracked rotki source acquisition path, provenance, checksum, or adapter constraint is missing, and reject committed raw rotki outputs in `tools/empiricaloracle/` and `tests/empirical/`
- [X] T066 [US2] ⚠️ Reopened (reopened — BUG-002): Implement the pinned rotki-based test-time oracle adapter for FIFO, LIFO, HIFO, and Average Cost aggregate fixtures against verified untracked pinned rotki source execution, not committed raw rotki outputs or developer-global installations, in `tools/empiricaloracle/` and `tests/empirical/fixture/`
- [X] T067 [US2] ⚠️ Reopened (reopened — BUG-002): Regenerate pure external-oracle golden fixtures from verified untracked pinned rotki source execution after zero-priced external-oracle cases are removed under `testdata/empirical/golden/` and update fixture metadata for source URL, pinned version or commit, source checksum, adapter arguments, hashes, and decimal policy
- [X] T068 [US2] ⚠️ Reopened (reopened — BUG-002): Replace HIFO ~~hledger fixtures~~ oracle fixtures with rotki HIFO fixtures regenerated from verified untracked pinned rotki source execution and add or preserve a deterministic non-zero-priced HIFO tie-break case in `testdata/empirical/golden/` and `testdata/empirical/financial-dataset.yaml`
- [X] T069 [US2] Limit Average Cost empirical comparisons to aggregate realized gain or loss, allocated basis, closing quantity, and closing basis until project-compatible pool provenance exists in `tests/empirical/fixture/comparison.go` and `tests/empirical/fixture/project_output.go`
- [X] T070 [US2] ⚠️ Reopened (reopened — BUG-002): Add a separate Scope-Local Hybrid composite oracle package that combines rotki-backed arithmetic sub-oracles regenerated from verified untracked pinned rotki source execution with documented project-owned composition-rule assertions under `tools/empiricaloracle/` and `tests/empirical/fixture/`
- [X] T071 [US3] Remove broad top-level supported-fixture skip policies from `tests/empirical/empirical_calculation_test.go` while preserving only fixture-backed unsupported field-level skips with explicit reasons
- [X] T072 [P] [US3] Add an acceptance check that fails when any supported empirical fixture group is skipped before project calculation and oracle comparison in `tests/empirical/empirical_calculation_test.go`
- [X] T075 [US2] Define the untracked project-local rotki source directory, `.gitignore` coverage, cleanup policy, and normal-test no-network rule in `.gitignore`, `third_party/rotki/README.md`, `testdata/empirical/README.md`, and `specs/006-empirical-financial-tests/research.md`
- [X] T076 [US2] Implement pinned rotki source download, checksum verification, commit or tag verification, and actionable setup errors in `tools/empiricaloracle/`
- [X] T077 [US2] Implement local test-time adapter execution that directly accesses rotki calculation code from the verified untracked source checkout or archive extraction path in `tools/empiricaloracle/`
- [X] T078 [P] [US2] Add regeneration guard tests that reject committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, and vendored rotki source in `tools/empiricaloracle/` and `tests/empirical/`
- [X] T079 [US2] Regenerate committed golden fixtures from the verified rotki source execution path and record source URL, pinned version or commit, source checksum, adapter arguments, input hash, output hash, and normalization version in `testdata/empirical/golden/`
- [X] T080 [P] [US3] Add tests proving normal `go test ./tests/empirical -count=1 -v` runs do not download rotki, while the explicit regeneration command downloads or reuses only the verified untracked source path in `tests/empirical/` and `tools/empiricaloracle/`

**Checkpoint**: Supported empirical fixtures execute project calculation and oracle comparison across FIFO, LIFO, HIFO, Average Cost aggregate values, and Scope-Local Hybrid composite assertions without unexpected supported fixture skips, and rotki-backed regenerated fixtures come only from verified untracked pinned rotki source execution.

---

## Phase 7: BUG-003 hledger Boundary Removal (Blocking Before Polish Acceptance)

**Purpose**: Remove obsolete hledger dependency, fixture, tooling, metadata, and documentation surface from active feature scope while preserving historical bug report references and explicit strikethrough rationale.

**Dependencies**: T081, T082, T083, and T084 depend on BUG-001/BUG-002 remediation fixture and rotki boundary tasks. T085 depends on T081 through T084. Reopened T058 and T063 depend on T085.

- [X] T081 [US2] Remove `third_party/hledger/` source, executable, license, README, source metadata, executable metadata, platform notes, and checksums from active project artifacts
- [X] T082 [US2] Remove `testdata/empirical/hledger/` generated journals and any active documentation that treats hledger journals as retained or auxiliary fixtures
- [X] T083 [US2] Remove hledger command, journal rendering, normalization, provenance, generation-policy, and vendoring tests or code from `tools/empiricaloracle/` and `tests/empirical/` unless a later non-oracle spec explicitly reintroduces a separate use
- [X] T084 [US2] Remove hledger-specific metadata and support labels from `testdata/empirical/golden/`, `specs/006-empirical-financial-tests/contracts/`, `specs/006-empirical-financial-tests/quickstart.md`, `testdata/empirical/README.md`, and oracle-output documentation except historical bug report references or explicit strikethrough rationale
- [X] T085 [P] [US2] Add a cleanup verification check that fails on active non-historical `hledger` or `hleger` references in source, tests, fixtures, active contracts, quickstart, README files, and oracle-output documentation after BUG-003 cleanup

**Checkpoint**: Hledger is no longer present as an active oracle, fixture, dependency, tooling, metadata, or documentation boundary.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Finalize documentation, repository verification, formatting, coverage wiring, fixture review, BUG-001/BUG-002 oracle-remediation evidence, and BUG-003 hledger-removal evidence after Phase 6 and Phase 7 are complete.

- [X] T053 [P] Update final empirical verification commands and oracle generation command after BUG-001 and BUG-002 remediation in `specs/006-empirical-financial-tests/quickstart.md`
- [X] T054 [P] Update final fixture names, comparability labels, unsupported-case policy, rotki source execution metadata examples, composite-oracle metadata examples, and superseded ~~hledger/raw-rotki-output~~ oracle-evidence metadata notes in `specs/006-empirical-financial-tests/contracts/oracle-output.md`
- [X] T055 [P] Update empirical isolation notes for the rotki adapter, untracked source acquisition boundary, normal-test no-network rule, and composite oracle in `specs/006-empirical-financial-tests/contracts/empirical-tests.md`
- [X] T056 Add `./tests/empirical` to the coverage test package list in `Makefile` while keeping `-coverpkg=$(PRODUCTION_PACKAGES)` unchanged
- [X] T057 Run `gofmt` on Go files under `internal/support/math/`, `tests/empirical/`, and `tools/empiricaloracle/`
- [X] T058 [P] ⚠️ Reopened (reopened — BUG-003): ~~Run synthetic and secret-content fixture review for `testdata/empirical/financial-dataset.yaml`, `testdata/empirical/golden/`, `testdata/empirical/hledger/`, `testdata/empirical/rotki/`, and `third_party/rotki/`, and verify no rotki source checkout is committed~~ Run synthetic and secret-content fixture review after hledger removal for `testdata/empirical/financial-dataset.yaml`, `testdata/empirical/golden/`, `testdata/empirical/rotki/`, and `third_party/rotki/`, and verify no hledger artifacts or rotki source checkout are committed
- [X] T059 Run `make test` from the repository root for `Makefile` after BUG-001 and BUG-002 remediation is complete
- [X] T060 Run `make coverage` from the repository root for `Makefile` after BUG-001 and BUG-002 remediation is complete
- [X] T061 [P] Record OWASP Top 10 review evidence for empirical test infrastructure and the BUG-001/BUG-002 oracle replacement boundary in `specs/006-empirical-financial-tests/quickstart.md`
- [X] T062 [P] Verify fixture-backed empirical test runtime target of 30 seconds or less with `go test ./tests/empirical -count=1 -v` after BUG-001 and BUG-002 remediation, verify no rotki download occurs, and document the observed result in `specs/006-empirical-financial-tests/quickstart.md`
- [X] T063 ⚠️ Reopened (reopened — BUG-003): Update `specs/006-empirical-financial-tests/spec.md`, `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, and `contracts/oracle-output.md` to reflect the actually implemented external oracle provenance, rotki adapter constraints, verified untracked source execution boundary, Scope-Local Hybrid composite oracle, and ~~superseded hledger-only/raw-rotki-output planning assumptions~~ BUG-003 removal of hledger from active oracle, fixture, dependency, and documentation scope

---

## Phase 9: Coding Standards Drift Remediation

**Purpose**: Remediate the coding-standards drift findings recorded in `specs/006-empirical-financial-tests/coding-standards-drift-report.md` after the normal implementation task list is complete.

**Dependencies**: All prior setup, foundational, objective, bugfix, hledger-removal, and polish tasks must remain complete before starting this phase.

- [X] T086 Remediate CODE-STAND-DRIFT-001 (High) Shared Math Helper Reads Process Environment from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-001-shared-math-helper-reads-process-environment` by moving `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` selection out of the shared math helper boundary and into an explicit caller/test boundary while preserving decimal-policy behavior in `internal/support/math/decimal_policy.go` and `internal/support/math/decimal_ops.go`
- [X] T087 Remediate CODE-STAND-DRIFT-002 (Medium) Empirical Fixture Duplicates Runtime Scope-Reliability Rules from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-002-empirical-fixture-duplicates-runtime-scope-reliability-rules` by making `tests/empirical/fixture/project_translation.go` reuse a single runtime scope-reliability rule source instead of duplicating logic from `internal/sync/normalize/activity_history.go`
- [X] T088 Remediate CODE-STAND-DRIFT-003 (Medium) Oracle Command `run` Mixes Multiple Responsibilities from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-003-oracle-command-run-mixes-multiple-responsibilities` by splitting CLI parsing, repository path resolution, dataset and fixture loading, rotki source setup, generation routing, artifact writing, and user-facing reporting responsibilities in `tools/empiricaloracle/main.go`
- [X] T089 Remediate CODE-STAND-DRIFT-004 (Medium) Rotki Adapter Boundary Uses Generic Payloads And Mixed Logic from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-004-rotki-adapter-boundary-uses-generic-payloads-and-mixed-logic` by replacing generic payload interpretation with explicit adapter input/output parsing and splitting oracle execution, normalization, match-evidence, closing-state, and response construction in `tools/empiricaloracle/rotki_adapter.py`
- [X] T090 Remediate CODE-STAND-DRIFT-005 (Medium) Coverage Target Omits The Empirical Fixture Subpackage from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-005-coverage-target-omits-the-empirical-fixture-subpackage` by updating `Makefile` coverage package execution so `./tests/empirical/fixture` tests such as `tests/empirical/fixture/model_test.go` run during `make coverage`
- [X] T091 Remediate CODE-STAND-DRIFT-006 (Medium) External Source Boundary Documentation Omits Required Integration Details from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-006-external-source-boundary-documentation-omits-required-integration-details` by documenting the authentication model, expected external failure modes, and security implications for the GitHub archive download and `git ls-remote` verification boundary in `third_party/rotki/README.md`, aligned with `tools/empiricaloracle/rotki_source.go`
- [X] T092 Remediate CODE-STAND-DRIFT-007 (Low) AI-Authored Feature Packages Lack Package-Level Documentation from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-007-ai-authored-feature-packages-lack-package-level-documentation` by adding or correcting package-level documentation and OpenCode authoring blocks for `tools/empiricaloracle/main.go`, `tests/empirical/dataset_validation_test.go`, `tests/empirical/fixture/model.go`, and the corresponding package `doc.go` files when present
- [X] T093 Remediate CODE-STAND-DRIFT-008 (Low) Dataset Validation Test Documentation Is Stale from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-008-dataset-validation-test-documentation-is-stale` by updating comments in `tests/empirical/dataset_validation_test.go` so parser and validator hook documentation reflects the current `fixture.LoadEmpiricalDataset` and `fixture.ValidateEmpiricalDataset` implementation
- [X] T094 Remediate CODE-STAND-DRIFT-009 (Low) Active Oracle Code Retains Journal Terminology from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-009-active-oracle-code-retains-journal-terminology` by replacing active journal and lot-mode naming with rotki/composite-oracle terminology in `tools/empiricaloracle/oracle_helpers.go`, `tools/empiricaloracle/unsupported.go`, and `specs/006-empirical-financial-tests/contracts/dataset-format.md`
- [X] T095 Remediate CODE-STAND-DRIFT-010 (Low) Exported Fixture Helper Lacks Public Usage Documentation from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-010-exported-fixture-helper-lacks-public-usage-documentation` by expanding the public documentation for `NormalizeProjectCalculationOutputForCase` in `tests/empirical/fixture/project_output.go` with usage guidance suitable for callers in other packages
- [X] T096 Remediate CODE-STAND-DRIFT-011 (Low) Dead Helper Code Remains In Empirical Test Slice from `specs/006-empirical-financial-tests/coding-standards-drift-report.md#code-stand-drift-011-dead-helper-code-remains-in-empirical-test-slice` by removing the unused `allRequiredCoverageTags` helper from `tests/empirical/fixture/dataset_coverage_test.go` and the unused `calculate_open_state` helper from `tools/empiricaloracle/rotki_adapter.py`, adjusting tests only if a hidden dependency is found
- [X] T097 Run `gofmt` on Go files changed by coding-standards drift remediation under `internal/support/math/`, `internal/sync/normalize/`, `tests/empirical/`, and `tools/empiricaloracle/`
- [X] T098 Run coding-standards drift remediation verification with `go test ./tests/empirical/... ./tools/empiricaloracle ./internal/support/math ./internal/sync/normalize -count=1 -v`, `make test`, and `make coverage` from the repository root

**Checkpoint**: Coding-standards drift findings are remediated and verified after the implementation feature work is complete.

---

## Phase 10: Test Coverage Drift Remediation

**Purpose**: Remediate the test-coverage drift findings recorded in `specs/006-empirical-financial-tests/test-coverage-drift-report.md` after the normal implementation and coding-standards drift task lists are complete.

**Dependencies**: All prior implementation, bugfix, hledger-removal, polish, and coding-standards drift remediation tasks must remain complete before starting this phase. T100 depends on T099.

- [ ] T099 Remediate COV-DRIFT-001 (High) Coverage Gate Fails On Decimal Policy Startup Error Branch from `specs/006-empirical-financial-tests/test-coverage-drift-report.md#cov-drift-001-coverage-gate-fails-on-decimal-policy-startup-error-branch` by adding package-local bootstrap coverage for the `supportmath.SetActiveDecimalPolicy(policy)` failure path in `internal/app/bootstrap/decimal_policy.go` and `internal/app/bootstrap/bootstrap_internal_test.go`, preserving the maintained coverage gate instrumentation in `Makefile` and `tools/coveragegate/main.go`
- [ ] T100 Verify COV-DRIFT-001 (High) Coverage Gate Fails On Decimal Policy Startup Error Branch from `specs/006-empirical-financial-tests/test-coverage-drift-report.md#cov-drift-001-coverage-gate-fails-on-decimal-policy-startup-error-branch` by running `make coverage` from the repository root and confirming the coverage gate no longer reports the uncovered statement, line, or branch recorded in `dist/coverage/coverage.xml`

**Checkpoint**: Test-coverage drift findings are remediated and the maintained coverage gate passes after all implementation and prior drift-remediation tasks remain complete.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks all objective phases.
- **US1 Dataset (Phase 3)**: Depends on Foundational completion.
- **US2 Oracle (Phase 4)**: Depends on US1 because ~~hledger journals~~ external-oracle inputs and golden fixtures are generated from the validated dataset.
- **US3 Empirical Tests (Phase 5)**: Depends on US1 and US2 because comparisons require the validated dataset and golden fixtures.
- **BUG-001/BUG-002 Remediation (Phase 6)**: Depends on US1, reopened US2 oracle fixture work, reopened US3 empirical comparison work, a non-vendored verified rotki source acquisition boundary, and rejection of committed raw rotki outputs as oracle evidence; blocks BUG-003 hledger removal, polish acceptance, and reopened T052.
- **BUG-003 hledger Removal (Phase 7)**: Depends on BUG-001/BUG-002 remediation because rotki and composite-oracle paths must remain after hledger cleanup; blocks polish acceptance, reopened T058, and reopened T063.
- **Polish (Phase 8)**: Depends on all selected objective phases, BUG-001/BUG-002 remediation completion, and BUG-003 hledger removal completion.
- **Coding Standards Drift Remediation (Phase 9)**: Depends on all prior implementation, bugfix, hledger-removal, and polish tasks being complete.
- **Test Coverage Drift Remediation (Phase 10)**: Depends on all prior implementation, bugfix, hledger-removal, polish, and coding-standards drift remediation tasks being complete.

### Objective Dependency Graph

```text
Setup -> Foundational -> US1 Dataset -> US2 Oracle -> US3 Empirical Tests -> BUG-001/BUG-002 Remediation -> BUG-003 hledger Removal -> Polish -> Coding Standards Drift Remediation -> Test Coverage Drift Remediation
```

### Parallel Opportunities

- T002, T003, T004, and T005 can run in parallel after T001 resolves setup paths.
- T006, T008, T010, and T012 can run in parallel because they add tests in different files.
- T014, T015, and T016 can run in parallel once Phase 2 is complete.
- T024, T025, T026, T027, and T028 can run in parallel once US1 is complete.
- T040, T041, T042, T043, and T044 can run in parallel once US2 is complete.
- T053, T054, T055, T061, and T062 can run in parallel during Polish after BUG-003 removal tasks complete.
- T065 and T072 can run in parallel with independent implementation work after BUG-001 remediation starts.
- T074 and T078 can run in parallel with source-boundary implementation after T076 establishes the verified rotki source acquisition path.
- T080 can run in parallel with final fixture regeneration after T076 because it verifies the normal-test no-network rule and explicit-regeneration source path.
- T081, T082, T083, and T084 can run in parallel after BUG-001/BUG-002 remediation establishes the active rotki and composite-oracle paths.
- T090, T091, T093, T095, and T096 can run in parallel during coding-standards drift remediation because they touch independent files or documentation surfaces.

---

## Parallel Example: User Story 1

```bash
Task: "Add dataset parser contract tests for top-level fields, activity fields, case fields, string-only decimals, scopes, and zero-priced reductions in tests/empirical/fixture/dataset_parser_test.go"
Task: "Add dataset validation contract tests for activity count, year span, supported methods, deterministic source IDs, ordering metadata, single currency, and synthetic-only content in tests/empirical/dataset_validation_test.go"
Task: "Add required coverage tag tests for every method and edge-case category from specs/006-empirical-financial-tests/spec.md in tests/empirical/fixture/dataset_coverage_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "Remove hledger vendoring contract tests and keep only rotki/composite-oracle boundary checks required by the current oracle model"
Task: "Add oracle fixture schema tests for metadata, decimal strings, tolerances, hashes, methods, years, matches, and unsupported segments in tests/empirical/fixture/oracle_output_test.go"
Task: "Remove hledger command wrapper tests and verify no hledger command wrapper remains in tools/empiricaloracle/"
Task: "Remove hledger journal rendering tests and any active journal-rendering fixtures from tools/empiricaloracle/"
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
2. Deliver US2 so ~~hledger inputs~~ external-oracle inputs and golden fixtures are reproducible from the dataset.
3. Deliver US3 so project calculation output is compared to oracle fixtures.
4. Complete BUG-003 hledger removal.
5. Run Polish verification with `go test ./tests/empirical -count=1 -v`, `make test`, and `make coverage`.
6. Complete coding-standards drift remediation after all normal implementation tasks are complete.
7. Complete test-coverage drift remediation and verify with `make coverage`.

### Parallel Team Strategy

1. Complete Setup and Foundational phases first.
2. Assign parallel test-writing tasks inside each objective phase before implementation tasks.
3. Keep phase order sequential because the oracle depends on the dataset and the empirical comparison suite depends on both dataset and golden fixtures.

---

## Notes

- Keep external oracle tooling behind a separate test-time boundary. Do not import, link, or execute ~~hledger,~~ rotki, or oracle adapters from runtime application code.
- Keep empirical artifacts synthetic. Do not add real tokens, JWTs, user activity, real account names, wallet names, proprietary financial records, raw protected snapshots, generated Markdown reports, TUI text, output filenames, or Documents paths.
- Use `apd.Decimal` through existing decimal helpers. Do not introduce floating-point math in dataset parsing, oracle normalization, or comparison code.
- Treat `testdata/empirical/financial-dataset.yaml` as read-only after this dataset-maintenance feature is complete.
- Add required OpenCode authoring documentation to new Go package, type, and function comments when implementing these tasks.
