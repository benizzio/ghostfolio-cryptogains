# Tasks: Capital Gains Report PDF And Audit Annex

**Input**: Design documents from `/specs/008-report-pdf-annex/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests and Quality Gates**: Automated tests are mandatory for this feature because the specification and quickstart require contract, integration, targeted unit, coverage, and changed-source quality evidence. Existing empirical financial datasets under `testdata/empirical/` remain read-only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing where feasible. Shared report models and bundle plumbing are in the foundational phase because every story depends on the new output and annex contract.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because the task touches different files and has no dependency on another incomplete task.
- **[Story]**: User story label for traceability. Setup, foundational, and polish tasks do not use story labels.
- Every task includes exact file paths.

## Path Conventions

- Go source lives under `cmd/`, `internal/`, `tests/`, and `tools/` at repository root.
- Report calculation remains under `internal/report/calculate/`.
- Report model contracts remain under `internal/report/model/`.
- Markdown rendering remains under `internal/report/markdown/`.
- PDF rendering is added under `internal/report/pdf/`.
- Local output writing remains under `internal/report/output/`.
- Runtime orchestration remains under `internal/app/runtime/`.
- TUI workflow state and rendering remain under `internal/tui/flow/` and `internal/tui/screen/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add planned local PDF dependencies, package structure, and shared fixtures before changing report behavior.

- [ ] T001 Pin planned PDF and font dependencies `github.com/signintech/gopdf@v0.36.1` and `golang.org/x/image@v0.43.0` in `go.mod` and `go.sum`
- [ ] T002 [P] Create the local PDF renderer package skeleton and package documentation in `internal/report/pdf/renderer.go`
- [ ] T003 [P] Add deterministic report-output fixture builders for format, annex, and conversion test data in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared report request, document, output bundle, and annex shell models used by every user story.

**Critical**: No user story can be completed until this phase is complete.

- [ ] T004 Add `ReportOutputFormat` enum, labels, supported-format list, and validation in `internal/report/model/report_output_format.go`
- [ ] T005 Extend `ReportRequest` with required output format and update constructor validation in `internal/report/model/report_request.go`
- [ ] T006 Add document roles, PDF byte payload support, and output bundle models in `internal/report/model/report_document.go`, `internal/report/model/report_output_file.go`, and `internal/report/model/report_output_bundle.go`
- [ ] T007 Add minimal `AuditAnnex` model shell with title and section-order validation in `internal/report/model/audit_annex.go`
- [ ] T008 Update calculated report validation and clone behavior for annex-aware output in `internal/report/model/capital_gains_report.go` and `internal/report/model/report_clone.go`
- [ ] T009 Update runtime report outcome structs for output bundles and selected output format in `internal/app/runtime/report_types.go` and `internal/app/runtime/report_output_outcome.go`
- [ ] T010 Update shared test fixture builders to supply output format and annex shell defaults in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`

**Checkpoint**: Report requests, calculated reports, rendered documents, output files, and runtime outcomes can represent Markdown main-plus-annex output and combined PDF output.

---

## Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP

**Goal**: Users can choose Markdown or PDF before report generation, and successful results list every generated file for the selected format.

**Independent Test**: Generate the same deterministic report inputs once as Markdown and once as PDF, then confirm Markdown creates one main `.md` and one Annex 1 `.md`, PDF creates one `.pdf`, and both successful result screens show the selected format and all saved paths.

### Tests for User Story 1

- [ ] T011 [P] [US1] Add report output-format workflow contract tests for selection, busy, and result copy in `tests/contract/report_generation_workflow_contract_test.go`
- [ ] T012 [P] [US1] Add report output file contract tests for Markdown/PDF file counts, filename patterns, and suffix rules in `tests/contract/report_output_contract_test.go`
- [ ] T013 [P] [US1] Add runtime integration test for generating the same fixture as Markdown and PDF in `tests/integration/report_generation_flow_test.go`
- [ ] T014 [P] [US1] Add output bundle cleanup integration test for render and write failures in `tests/integration/report_failure_flow_test.go`
- [ ] T015 [P] [US1] Add PDF renderer unit tests for A4 configuration, embedded font loading, and text emission seams in `internal/report/pdf/renderer_internal_test.go`
- [ ] T016 [P] [US1] Add output bundle writer unit tests for Markdown pair reservation and PDF filename suffixes in `internal/report/output/writer_internal_test.go`

### Implementation for User Story 1

- [ ] T017 [US1] Implement output-format list state, focus movement, selection, and report request construction in `internal/tui/flow/state.go` and `internal/tui/flow/report_flow.go`
- [ ] T018 [US1] Render output-format choices, selected-format explanations, busy state, and result copy in `internal/tui/flow/view.go`, `internal/tui/flow/help_text.go`, and `internal/tui/screen/report_screen.go`
- [ ] T019 [US1] Implement bundle-aware Markdown rendering entry point returning main and Annex 1 documents in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_annex.go`
- [ ] T020 [US1] Implement initial local A4 PDF renderer for the main report plus Annex 1 shell in `internal/report/pdf/renderer.go`
- [ ] T021 [US1] Implement output bundle reservation, write, sync, close, suffixing, and cleanup for two-file Markdown and one-file PDF output in `internal/report/output/writer.go`
- [ ] T022 [US1] Select renderer by output format, write output bundles, request automatic open, and shape all saved paths in `internal/app/runtime/report_service.go` and `internal/app/runtime/report_output_outcome.go`
- [ ] T023 [US1] Update report result path labels to show Markdown main path plus Annex 1 path or the single PDF path in `internal/tui/screen/report_screen.go`

**Checkpoint**: User Story 1 is fully functional and can be validated independently as the MVP.

---

## Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2)

**Goal**: The main report is shorter and easier to scan without changing financial calculations or losing relevant evidence.

**Independent Test**: Generate reports containing zero net-gain summary rows, rate-source disclosures, assets without report-year activity, zero-priced SELL activities, and conversion statuses, then verify the main report content in Markdown and PDF.

### Tests for User Story 2

- [ ] T024 [P] [US2] Add main report presentation contract tests for bold labels, zero row omission, header rename, and no main conversion audit in `tests/contract/markdown_report_contract_test.go`
- [ ] T025 [P] [US2] Add Markdown renderer unit tests for summary empty state, historical position, conversion status labels, and `BLOCKCHAIN OP` in `internal/report/markdown/renderer_internal_test.go`
- [ ] T026 [P] [US2] Add PDF main report presentation unit tests mirroring shared Markdown content rules in `internal/report/pdf/renderer_internal_test.go`
- [ ] T027 [P] [US2] Add report output integration assertions for clearer main report content in both formats in `tests/integration/report_generation_flow_test.go`

### Implementation for User Story 2

- [ ] T028 [US2] Add closed user-facing render label helpers for conversion status, quote direction, and zero-priced SELL display in `internal/report/model/render_labels.go`
- [ ] T029 [US2] Bold initial detail labels and rate-source classifier labels in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_conversion.go`
- [ ] T030 [US2] Omit zero net-gain summary rows and render the all-zero empty state in `internal/report/markdown/renderer_summary.go`
- [ ] T031 [US2] Rename the reference header and render `Historical Position` for assets without report-year activity in `internal/report/markdown/renderer_details.go`
- [ ] T032 [US2] Remove detailed Currency Conversion Audit from the main Markdown report and use label helpers for visible conversion statuses in `internal/report/markdown/renderer_conversion.go` and `internal/report/markdown/renderer_details.go`
- [ ] T033 [US2] Apply the same main-report presentation, zero-row filtering, historical-position, and label rules in the PDF renderer in `internal/report/pdf/renderer.go`

**Checkpoint**: User Story 2 can be validated independently without changing calculation outputs.

---

## Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3)

**Goal**: Annex 1 contains per-asset activity audit evidence and Currency Conversion Audit evidence for every reported asset through the selected report-year end.

**Independent Test**: Generate a report with multiple assets, historical activity, report-year activity, post-year activity, liquidations, gains or losses, reference-only assets, zero-net assets, and conversions, then verify Annex 1 content and placement in Markdown and PDF.

### Tests for User Story 3

- [ ] T034 [P] [US3] Add Annex 1 rendering contract tests for title, section order, per-asset fields, quote labels, and empty states in `tests/contract/report_annex_contract_test.go`
- [ ] T035 [P] [US3] Add calculation unit tests for annex scope including pre-year, report-year, post-year, zero-net, and reference-only assets in `internal/report/calculate/calculator_internal_test.go`
- [ ] T036 [US3] Add Markdown annex renderer unit tests for separate Annex 1 document content in `internal/report/markdown/renderer_internal_test.go`
- [ ] T037 [US3] Add PDF annex pagination, page-break, and repeated-context unit tests in `internal/report/pdf/renderer_internal_test.go`
- [ ] T038 [P] [US3] Add 10,000 cached-activity Markdown/PDF annex scale integration test in `tests/integration/report_performance_flow_test.go`

### Implementation for User Story 3

- [ ] T039 [US3] Add detailed audit annex models for per-asset sections, audit activity entries, and the conversion audit section in `internal/report/model/audit_annex.go` and `internal/report/model/audit_activity_entry.go`
- [ ] T040 [US3] Capture per-activity post-replay audit evidence through the selected year end in `internal/report/calculate/asset_replay.go` and `internal/report/calculate/artifacts.go`
- [ ] T041 [US3] Build reported-asset annex sections including reference-only assets and excluding post-year activity in `internal/report/calculate/calculator.go` and `internal/report/calculate/artifacts.go`
- [ ] T042 [US3] Render Annex 1 Markdown title, per-asset audit report, and Currency Conversion Audit in `internal/report/markdown/renderer_annex.go`
- [ ] T043 [US3] Append Annex 1 after a PDF page break with table and page continuation context in `internal/report/pdf/renderer.go`
- [ ] T044 [US3] Ensure runtime and output failures from missing label mappings or annex validation save no partial files in `internal/app/runtime/report_service.go` and `internal/report/output/writer.go`

**Checkpoint**: Annex 1 is complete for Markdown and PDF and all user stories are independently testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, security review, formatting, and documentation alignment across all stories.

- [ ] T045 [P] Update validation notes if implementation details differ from the planned workflow in `specs/008-report-pdf-annex/quickstart.md`
- [ ] T046 [P] Run `gofmt` on changed Go files under `internal/`, `tests/contract/`, `tests/integration/`, `tests/unit/`, and `tests/testutil/`
- [ ] T047 Run `make test` using `Makefile` and fix failures in changed files under `internal/` and `tests/`
- [ ] T048 Run `make coverage` using `Makefile` and inspect generated coverage artifacts under `dist/coverage/`
- [ ] T049 Run `make quality QUALITY_BASE_REF=origin/main` using `Makefile` and fix changed-source findings in `*.go`, `go.mod`, and `go.sum`
- [ ] T050 Review generated reports, result messages, diagnostics, and failure paths for token or secret leakage in `internal/report/`, `internal/app/runtime/`, and `internal/tui/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks every user story.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP scope.
- **User Story 2 (Phase 4)**: Depends on Foundational completion. PDF parity tasks depend on the PDF renderer skeleton from T002 and are easiest after US1.
- **User Story 3 (Phase 5)**: Depends on Foundational completion. Final Markdown/PDF placement depends on the bundle and renderer selection completed in US1.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **US1**: No dependency on US2 or US3.
- **US2**: No calculation dependency on US1 or US3, but PDF presentation assertions need the PDF renderer path from US1.
- **US3**: Calculation and model work can start after Foundational. Final output placement is easiest after US1.

### Within Each User Story

- Write story tests first and verify they fail before implementation.
- Complete model changes before calculation, renderer, runtime, and TUI integration changes.
- Complete renderer and output changes before result-screen assertions.
- Validate each story independently at its checkpoint before proceeding.

---

## Parallel Opportunities

- T002 and T003 can run in parallel after T001 because they touch different package/test files.
- T011 through T016 can run in parallel once Foundational models exist because they target separate contract, integration, unit, and writer test files.
- T024 through T027 can run in parallel because they target separate contract, renderer, PDF, and integration assertions.
- T034, T035, and T038 can run in parallel because they target separate contract, calculation, and performance test files.
- US2 Markdown presentation work and US3 calculation evidence work can proceed in parallel after Foundational if US1 runtime/output integration is not being edited at the same time.

---

## Parallel Example: User Story 1

```text
Task: T011 Add workflow contract tests in tests/contract/report_generation_workflow_contract_test.go
Task: T012 Add output file contract tests in tests/contract/report_output_contract_test.go
Task: T013 Add Markdown/PDF runtime integration tests in tests/integration/report_generation_flow_test.go
Task: T014 Add cleanup integration tests in tests/integration/report_failure_flow_test.go
Task: T015 Add PDF renderer unit tests in internal/report/pdf/renderer_internal_test.go
Task: T016 Add output writer unit tests in internal/report/output/writer_internal_test.go
```

## Parallel Example: User Story 2

```text
Task: T024 Add main report contract tests in tests/contract/markdown_report_contract_test.go
Task: T025 Add Markdown renderer unit tests in internal/report/markdown/renderer_internal_test.go
Task: T026 Add PDF main report tests in internal/report/pdf/renderer_internal_test.go
Task: T027 Add integration assertions in tests/integration/report_generation_flow_test.go
```

## Parallel Example: User Story 3

```text
Task: T034 Add Annex 1 contract tests in tests/contract/report_annex_contract_test.go
Task: T035 Add calculation unit tests in internal/report/calculate/calculator_internal_test.go
Task: T038 Add scale integration tests in tests/integration/report_performance_flow_test.go
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 setup.
2. Complete Phase 2 foundational models and bundle plumbing.
3. Complete Phase 3 User Story 1.
4. Validate US1 with its contract, unit, and integration tests.
5. Stop if the goal is only the MVP output-format workflow.

### Incremental Delivery

1. Deliver Setup and Foundational phases.
2. Deliver US1 to make Markdown/PDF selection and file output work.
3. Deliver US2 to improve main report readability without changing calculation behavior.
4. Deliver US3 to add complete Annex 1 audit evidence.
5. Run Phase 6 verification after all desired stories are complete.

### Parallel Team Strategy

1. One developer completes output bundle and runtime integration from US1.
2. One developer works on main report renderer changes from US2 after foundational label helpers exist.
3. One developer works on annex calculation evidence and annex renderer tests from US3 after foundational annex models exist.
4. Coordinate edits to `internal/report/pdf/renderer.go`, `internal/report/markdown/renderer_internal_test.go`, and `tests/integration/report_generation_flow_test.go` because multiple stories touch those files.

---

## Notes

- `[P]` tasks are safe to run in parallel only when the listed files are not already being edited by another active task.
- Every generated report file is intentional cleartext local output and must exclude Ghostfolio tokens, bearer tokens, reusable authentication material, protected payload bytes, and unrelated secrets.
- PDF generation must stay local-only and must not use remote services, browser services, external binaries, platform font paths, or user-installed fonts.
- Exact decimal report calculation behavior must remain unchanged.
- Treat `testdata/empirical/` as read-only for this feature.
