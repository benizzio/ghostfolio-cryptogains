# Tasks: Capital Gains Report PDF And Audit Annex

**Input**: Design documents from `/specs/008-report-pdf-annex/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests and Quality Gates**: Automated tests are mandatory for this feature because the specification and quickstart require contract, integration, targeted unit, coverage, and changed-source quality evidence. Existing empirical financial datasets under `testdata/empirical/` remain read-only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing where feasible. Shared report models and bundle plumbing are in the foundational phase because every story depends on the new output and annex contract. The context-orchestration work-unit ledger below is the execution control plane for parent agents and clean-context subagents.

**Bugfix**: 2026-07-05 — [BUG-001] Updated from bugfix patch.

**Bugfix**: 2026-07-05 — [BUG-002] Updated from bugfix patch.

**Bugfix**: 2026-07-07 — [BUG-003] Updated from bugfix patch.

**Bugfix**: 2026-07-09 — [BUG-004] Updated from bugfix patch.

**Bugfix**: 2026-07-09 — [BUG-005] Updated from bugfix patch.

**Bugfix**: 2026-07-10 — [BUG-006] Updated from bugfix patch.

**Bugfix**: 2026-07-13 — [BUG-007] Updated from bugfix patch.

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

## Context Orchestration Process

### Parent Orchestrator Rules

- Execute work units in ledger order unless the ledger explicitly marks units as parallel candidates and their prerequisites are verified.
- The task checkboxes remain authoritative; a ledger unit is complete only when every referenced task is checked after parent verification.
- Use a clean subagent session for each delegated work unit.
- Include all required handoff context in the subagent prompt; do not rely on prior conversation state.
- Keep subagents inside the listed scope and require them to stop before editing outside allowed paths.
- Require fail-first behavior for test tasks.
- Parent must inspect diffs, run targeted verification, check for unrelated changes, and fix or re-delegate inconsistencies before starting a dependent unit.
- Parent owns final validation and must rerun final gates even if a subagent helped triage command output.

### Required Subagent Handoff Packet

- Work unit ID, phase, task IDs, exact task descriptions, and exact paths.
- Relevant spec sources and contract files from `specs/008-report-pdf-annex/`, including `spec.md`, `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, and files under `contracts/` when applicable.
- Non-negotiable project constraints: PDF rendering stays local-only, landscape A4-sized, text-based, searchable, selectable, and uses application-supplied font data; no remote PDF service, browser service, external binary, platform font path, telemetry, or new persistence is allowed.
- Non-negotiable report constraints: financial calculations keep exact decimals and existing cost-basis behavior; presentation and audit evidence must not expose Ghostfolio tokens, bearer tokens, protected payload bytes, raw secrets, snake_case report labels, or partial-output success.
- Non-negotiable repository constraints: `testdata/empirical/` and generated oracle fixtures remain read-only; generated `dist/` artifacts are not implementation inputs; report models, calculation, Markdown, PDF, output writing, runtime, and TUI stay in their planned package boundaries.
- Current implementation status from previously verified units.
- Allowed edit paths and forbidden paths, including a requirement to stop before editing outside the unit scope.
- Tests or validation commands to run, including expected fail-first evidence for test units.
- Required final response fields: files changed, task IDs completed, tests run with results, expected failures, assumptions, and parent follow-up.

### Parent Verification Gate

- Inspect `git diff -- <unit paths>` plus any extra paths reported by the subagent.
- Confirm edits are inside unit scope or are justified adjacent changes required by the touched public boundary.
- Run targeted tests or the closest compiling package test listed in the ledger.
- Re-read relevant contracts or data-model sections for public types, rendering output, persistence behavior, security behavior, diagnostics, or TUI workflow changes.
- Confirm forbidden generated, fixture, empirical, secret, and persistence paths remain unchanged when such restrictions are present.
- Mark task checkboxes only after the gate passes.

### Context Compaction Recovery

- Read this orchestration process, the work-unit ledger, and the checklist before editing.
- Run `git status --short` and inspect existing diffs.
- Resume at the first ledger unit with unchecked referenced tasks unless an earlier partial diff exists.
- Reconstruct prior state from checked tasks, current diffs, and targeted tests.
- Finish and verify a partial unit before opening a new subagent.

### Work Unit Ledger

| Unit | Phase | Tasks | Atomic scope and touched paths | Prerequisites | Required handoff sources | Parent verification |
|------|-------|-------|--------------------------------|---------------|--------------------------|---------------------|
| WU01 | Phase 1: Setup (Shared Infrastructure) | T001 | Pin local PDF and font dependencies in `go.mod` and `go.sum`, and verify the planned `gofpdi` transitive PDF dependency selected by module resolution. | None | `specs/008-report-pdf-annex/tasks.md`, `plan.md`, `research.md`, `quickstart.md` | Inspect `git diff -- go.mod go.sum`; run `go list -m github.com/signintech/gopdf golang.org/x/image github.com/phpdave11/gofpdi`; confirm no unrelated module churn. |
| WU02 | Phase 1: Setup (Shared Infrastructure) | T002 | Create PDF renderer package skeleton and package documentation in `internal/report/pdf/renderer.go`. | WU01 | `tasks.md`, `plan.md`, `research.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/pdf/renderer.go`; run `go test ./internal/report/pdf`; confirm package boundary stays under `internal/report/pdf/`. |
| WU03 | Phase 1: Setup (Shared Infrastructure) | T003 | Add deterministic report-output fixture builders in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`. | WU01 | `tasks.md`, `data-model.md`, `quickstart.md`, `contracts/report-output.md` | Inspect `git diff -- tests/testutil/report_fixtures.go tests/testutil/report_io_fixtures.go`; run `go test ./tests/testutil`; confirm fixtures contain no tokens and do not touch `testdata/empirical/`. |
| WU04 | Phase 2: Foundational (Blocking Prerequisites) | T004, T005, T006, T007, T008 | Add output format, request, document payload, output-file metadata, bundle, annex shell, validation, and clone model support in `internal/report/model/report_output_format.go`, `internal/report/model/report_request.go`, `internal/report/model/report_document.go`, `internal/report/model/report_output_file.go`, `internal/report/model/report_output_bundle.go`, `internal/report/model/audit_annex.go`, `internal/report/model/capital_gains_report.go`, and `internal/report/model/report_clone.go`. | WU02, WU03 | `tasks.md`, `plan.md`, `spec.md`, `data-model.md`, `contracts/report-output.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/model`; run `go test ./internal/report/model`; re-read `data-model.md` output format, request, document, output-file metadata, bundle, and annex validation rules. |
| WU05 | Phase 2: Foundational (Blocking Prerequisites) | T009 | Update runtime report outcome structs for output bundles and selected output format in `internal/app/runtime/report_types.go` and `internal/app/runtime/report_output_outcome.go`. | WU04 | `tasks.md`, `plan.md`, `data-model.md`, `contracts/tui-workflows.md`, `contracts/report-output.md` | Inspect `git diff -- internal/app/runtime/report_types.go internal/app/runtime/report_output_outcome.go`; run `go test ./internal/app/runtime`; confirm runtime types do not write files or expose secrets. |
| WU06 | Phase 2: Foundational (Blocking Prerequisites) | T010 | Update shared test fixture builders for output format and annex shell defaults in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`. | WU04 | `tasks.md`, `data-model.md`, `quickstart.md`, `contracts/report-output.md` | Inspect `git diff -- tests/testutil/report_fixtures.go tests/testutil/report_io_fixtures.go`; run `go test ./tests/testutil`; confirm fixture defaults align with model validation. |
| WU07 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T011 | Add fail-first output-format workflow and SC-001 selection-to-start contract tests in `tests/contract/report_generation_workflow_contract_test.go`. | WU04, WU05, WU06 | `tasks.md`, `spec.md`, `quickstart.md`, `contracts/tui-workflows.md` | Inspect `git diff -- tests/contract/report_generation_workflow_contract_test.go`; run `go test ./tests/contract -run ReportGenerationWorkflow`; confirm failure is the expected missing US1 behavior before implementation and covers the 30-second start-generation workflow bound without synchronous rendering or file IO in the TUI path. |
| WU08 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T012 | Add fail-first report output file contract tests in `tests/contract/report_output_contract_test.go`. | WU04, WU05, WU06 | `tasks.md`, `spec.md`, `quickstart.md`, `contracts/report-output.md` | Inspect `git diff -- tests/contract/report_output_contract_test.go`; run `go test ./tests/contract -run ReportOutput`; confirm failure is the expected missing bundle or filename behavior. |
| WU09 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T013, T014 | Add fail-first runtime generation and cleanup integration tests in `tests/integration/report_generation_flow_test.go` and `tests/integration/report_failure_flow_test.go`. | WU04, WU05, WU06 | `tasks.md`, `spec.md`, `quickstart.md`, `contracts/report-output.md`, `contracts/tui-workflows.md` | Inspect `git diff -- tests/integration/report_generation_flow_test.go tests/integration/report_failure_flow_test.go`; run `go test ./tests/integration -run ReportGeneration` and `go test ./tests/integration -run ReportFailure`; confirm failures target missing Markdown/PDF bundle and cleanup behavior. |
| WU10 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T015 | Add fail-first PDF renderer unit tests in `internal/report/pdf/renderer_internal_test.go`, including landscape A4, embedded font, text-emission, `gopdf` table/styled-layout, printable-width, non-overlapping vertical layout, and no-Markdown-source seams. | WU02, WU04, WU06 | `tasks.md`, `plan.md`, `research.md`, `quickstart.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/pdf/renderer_internal_test.go`; run `go test ./internal/report/pdf`; confirm failure is the expected missing landscape A4, font, text-emission seam, table/styled-layout seam, no-Markdown-source boundary, right-boundary clipping prevention, non-overlapping vertical layout, or line-dump rejection. |
| WU11 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T016 | Add fail-first output bundle writer unit tests in `internal/report/output/writer_internal_test.go`. | WU04, WU05, WU06 | `tasks.md`, `quickstart.md`, `contracts/report-output.md` | Inspect `git diff -- internal/report/output/writer_internal_test.go`; run `go test ./internal/report/output`; confirm failure is the expected missing Markdown pair or PDF suffix behavior. |
| WU12 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T017, T018 | Implement TUI output-format selection state, request construction, visible choices, explanations, busy state, and result copy in `internal/tui/flow/state.go`, `internal/tui/flow/report_flow.go`, `internal/tui/flow/view.go`, `internal/tui/flow/help_text.go`, and `internal/tui/screen/report_screen.go`. | WU07, WU08, WU09, WU10, WU11 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/tui-workflows.md` | Inspect `git diff -- internal/tui/flow/state.go internal/tui/flow/report_flow.go internal/tui/flow/view.go internal/tui/flow/help_text.go internal/tui/screen/report_screen.go`; run `go test ./internal/tui/flow ./internal/tui/screen ./tests/contract -run ReportGenerationWorkflow`; confirm no report content preview or token exposure. |
| WU13 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T019 | Implement bundle-aware Markdown rendering entry point returning main and Annex 1 documents in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_annex.go`. | WU07, WU08, WU09, WU10, WU11 | `tasks.md`, `data-model.md`, `contracts/report-rendering.md`, `contracts/report-output.md` | Inspect `git diff -- internal/report/markdown/renderer.go internal/report/markdown/renderer_annex.go`; run `go test ./internal/report/markdown`; re-read Markdown rendering and output contracts. |
| WU14 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T020 | Implement initial local landscape A4 PDF renderer for the main report plus Annex 1 shell through `gopdf` page, font, text, styled-cell, table-layout, printable-width, and non-overlapping vertical-flow APIs in `internal/report/pdf/renderer.go`. | WU07, WU08, WU09, WU10, WU11 | `tasks.md`, `plan.md`, `research.md`, `contracts/report-rendering.md`, `quickstart.md` | Inspect `git diff -- internal/report/pdf/renderer.go`; run `go test ./internal/report/pdf`; confirm renderer stays local-only, text-based, landscape A4-sized, font-data based, free of Markdown body passthrough, not clipped at the right edge, and not a plain line dump. |
| WU15 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T021 | Implement output bundle reservation, write, sync, close, suffixing, and cleanup in `internal/report/output/writer.go`. | WU07, WU08, WU09, WU10, WU11 | `tasks.md`, `data-model.md`, `contracts/report-output.md`, `quickstart.md` | Inspect `git diff -- internal/report/output/writer.go`; run `go test ./internal/report/output`; confirm failed attempts remove all created files and file paths stay in the Documents directory. |
| WU16 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T022 | Select renderer by output format, write output bundles, request automatic open, and shape saved paths in `internal/app/runtime/report_service.go` and `internal/app/runtime/report_output_outcome.go`. | WU12, WU13, WU14, WU15 | `tasks.md`, `data-model.md`, `contracts/report-output.md`, `contracts/tui-workflows.md`, `quickstart.md` | Inspect `git diff -- internal/app/runtime/report_service.go internal/app/runtime/report_output_outcome.go`; run `go test ./internal/app/runtime ./tests/integration -run ReportGeneration`; confirm failures remain non-secret and partial saves are not reported as success. |
| WU17 | Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP | T023 | Update report result path labels for Markdown main plus Annex 1 paths or a single PDF path in `internal/tui/screen/report_screen.go`. | WU16 | `tasks.md`, `contracts/tui-workflows.md`, `contracts/report-output.md` | Inspect `git diff -- internal/tui/screen/report_screen.go`; run `go test ./internal/tui/screen ./tests/contract -run ReportGenerationWorkflow`; confirm result screen lists every saved path for the selected format. |
| WU18 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T024 | Add fail-first main report presentation contract tests, including exact Markdown initial detail bold-label lines, in `tests/contract/markdown_report_contract_test.go`. | WU17 | `tasks.md`, `spec.md`, `quickstart.md`, `contracts/report-rendering.md` | Inspect `git diff -- tests/contract/markdown_report_contract_test.go`; run `go test ./tests/contract -run MarkdownReport`; confirm failure is the expected missing main-report presentation behavior. |
| WU19 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T025 | Add fail-first Markdown renderer unit tests in `internal/report/markdown/renderer_internal_test.go`. | WU17 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/markdown/renderer_internal_test.go`; run `go test ./internal/report/markdown`; confirm failure targets summary, historical position, label, or `BLOCKCHAIN OP` behavior. |
| WU20 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T026 | Add fail-first PDF main report presentation unit tests for shared content, visible heading hierarchy, styled classifier labels, table headers, rows, columns, wrapped content, landscape table fit, non-overlapping section spacing, Rate Source Summary label/value formatting, and summary-total table placement in `internal/report/pdf/renderer_internal_test.go`. | WU17 | `tasks.md`, `spec.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/pdf/renderer_internal_test.go`; run `go test ./internal/report/pdf`; confirm failure mirrors shared Markdown content rules, rejects Markdown structural syntax in PDF output, rejects line-dump presentation, and catches BUG-004 main-report layout defects. |
| WU21 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T027 | Add fail-first integration assertions for clearer main report content in `tests/integration/report_generation_flow_test.go`. | WU17 | `tasks.md`, `quickstart.md`, `contracts/report-rendering.md`, `contracts/report-output.md` | Inspect `git diff -- tests/integration/report_generation_flow_test.go`; run `go test ./tests/integration -run ReportGeneration`; confirm failure is limited to US2 presentation assertions. |
| WU22 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T028 | Add closed user-facing render label helpers in `internal/report/model/render_labels.go`. | WU18, WU19, WU20, WU21 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/model/render_labels.go`; run `go test ./internal/report/model`; confirm unmapped labels fail before output success and snake_case labels are not exposed. |
| WU23 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T029, T030, T031, T032 | Implement Markdown main-report presentation rules, including the initial details block bold-label path, in `internal/report/markdown/renderer.go`, `internal/report/markdown/renderer_conversion.go`, `internal/report/markdown/renderer_summary.go`, and `internal/report/markdown/renderer_details.go`. | WU22 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/markdown/renderer.go internal/report/markdown/renderer_conversion.go internal/report/markdown/renderer_summary.go internal/report/markdown/renderer_details.go`; run `go test ./internal/report/markdown ./tests/contract -run MarkdownReport`; confirm calculation outputs are not changed. |
| WU24 | Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2) | T033 | Apply shared main-report presentation, styled classifier labels, readable landscape table layout, Rate Source Summary label/value formatting, summary-total table placement, and non-overlapping section spacing in the PDF renderer through `gopdf` layout primitives in `internal/report/pdf/renderer.go`. | WU22, WU23 | `tasks.md`, `spec.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/pdf/renderer.go`; run `go test ./internal/report/pdf`; confirm PDF data values match Markdown aside from allowed pagination and page-title differences, without Markdown body passthrough, line dumping, clipped tables, or overlapping section text. |
| WU25 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T034 | Add fail-first Annex 1 rendering contract tests for required visible audit fields in `tests/contract/report_annex_contract_test.go`. | WU24 | `tasks.md`, `spec.md`, `quickstart.md`, `contracts/report-rendering.md` | Inspect `git diff -- tests/contract/report_annex_contract_test.go`; run `go test ./tests/contract -run ReportAnnex`; confirm failure is the expected missing Annex 1 evidence behavior and covers every FR-019 field or allowed equivalent visible field. |
| WU26 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T035 | Add fail-first calculation unit tests for annex scope in `internal/report/calculate/calculator_internal_test.go`. | WU24 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/calculate/calculator_internal_test.go`; run `go test ./internal/report/calculate`; confirm failures target annex scope and do not require empirical fixture mutation. |
| WU27 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T036 | Add fail-first Markdown annex renderer unit tests in `internal/report/markdown/renderer_internal_test.go`. | WU24 | `tasks.md`, `contracts/report-rendering.md`, `contracts/report-output.md` | Inspect `git diff -- internal/report/markdown/renderer_internal_test.go`; run `go test ./internal/report/markdown`; confirm failure is limited to separate Annex 1 document content. |
| WU28 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T037 | Add fail-first PDF annex pagination, page-break, repeated-context, table header, row, column, wrapped-cell, and Annex 1 asset-heading spacing unit tests in `internal/report/pdf/renderer_internal_test.go`. | WU24 | `tasks.md`, `contracts/report-rendering.md`, `quickstart.md` | Inspect `git diff -- internal/report/pdf/renderer_internal_test.go`; run `go test ./internal/report/pdf`; confirm failure targets Annex 1 page-break, repeated-context behavior, table readability, same-page asset-heading spacing, and no Markdown structural syntax. |
| WU29 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T038, T087 | Add fail-first 10,000 cached-activity Markdown/PDF annex scale coverage, corrected by BUG-007 to assert one independent timing interval per selected format in `tests/performance/report_performance_flow_test.go`. | WU24 | `tasks.md`, `quickstart.md`, `contracts/report-rendering.md`, `contracts/report-output.md`, `bugs/BUG-007.md` | Inspect `git diff -- tests/performance/report_performance_flow_test.go`; run `make test-performance`; confirm each failure identifies its output format and measured duration, generated scale fixtures stay project-owned, and `testdata/empirical/` remains unchanged. |
| WU30 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T039 | Add detailed audit annex models, including every required FR-019 audit activity field, in `internal/report/model/audit_annex.go` and `internal/report/model/audit_activity_entry.go`. | WU25, WU26, WU27, WU28, WU29 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/model/audit_annex.go internal/report/model/audit_activity_entry.go`; run `go test ./internal/report/model`; re-read `data-model.md` annex and audit entry validation rules. |
| WU31 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T040, T041 | Capture post-replay audit evidence and build reported-asset annex sections in `internal/report/calculate/asset_replay.go`, `internal/report/calculate/artifacts.go`, and `internal/report/calculate/calculator.go`. | WU30 | `tasks.md`, `spec.md`, `data-model.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/calculate/asset_replay.go internal/report/calculate/artifacts.go internal/report/calculate/calculator.go`; run `go test ./internal/report/calculate`; confirm existing cost-basis and exact-decimal calculation behavior is preserved. |
| WU32 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T042 | Render Annex 1 Markdown title, per-asset audit report, and Currency Conversion Audit in `internal/report/markdown/renderer_annex.go`. | WU31 | `tasks.md`, `contracts/report-rendering.md`, `contracts/report-output.md` | Inspect `git diff -- internal/report/markdown/renderer_annex.go`; run `go test ./internal/report/markdown ./tests/contract -run ReportAnnex`; confirm Annex 1 is separate Markdown output with required title and section order. |
| WU33 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T043 | Append Annex 1 after a PDF page break with `gopdf` table rows, table columns, wrapped cells, page continuation context, and adequate same-page asset-heading top margin in `internal/report/pdf/renderer.go`. | WU31 | `tasks.md`, `research.md`, `contracts/report-rendering.md`, `quickstart.md` | Inspect `git diff -- internal/report/pdf/renderer.go`; run `go test ./internal/report/pdf`; confirm Annex 1 starts on a new page, required report text remains selectable text, Markdown syntax is not rendered as PDF body text, Annex 1 is not a line dump, and asset subheadings do not touch previous tables. |
| WU34 | Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3) | T044 | Ensure runtime and output failures from missing label mappings or annex validation save no partial files in `internal/app/runtime/report_service.go` and `internal/report/output/writer.go`. | WU32, WU33 | `tasks.md`, `spec.md`, `contracts/report-output.md`, `contracts/tui-workflows.md`, `quickstart.md` | Inspect `git diff -- internal/app/runtime/report_service.go internal/report/output/writer.go`; run `go test ./internal/app/runtime ./internal/report/output` and `go test ./tests/integration -run ReportFailure`; confirm failures are non-secret and no partial files are reported. |
| WU35 | Phase 6: Polish & Cross-Cutting Concerns | T045 | Update validation notes if implementation details differ in `specs/008-report-pdf-annex/quickstart.md`. | WU34 | `tasks.md`, `quickstart.md`, `plan.md`, `spec.md` | Inspect `git diff -- specs/008-report-pdf-annex/quickstart.md`; run `make test` if quickstart validation claims changed; confirm notes track observed implementation behavior and do not change contracts or task checklist. |
| WU36 | Bugfix BUG-002: PDF rendering boundary | T052, T053, T054 | Add no-Markdown-source and no-line-dump PDF regression coverage, prove or refactor direct PDF-domain rendering through `gopdf` layout APIs, and record dependency reassessment in `internal/report/pdf/renderer_internal_test.go`, `internal/report/pdf/renderer.go`, and `specs/008-report-pdf-annex/research.md`. | WU10, WU14, WU20, WU24, WU28, WU33 | `specs/008-report-pdf-annex/bugs/BUG-002.md`, `specs/008-report-pdf-annex/bugs/BUG-003.md`, `tasks.md`, `spec.md`, `plan.md`, `research.md`, `contracts/report-rendering.md` | Inspect `git diff -- internal/report/pdf/renderer_internal_test.go internal/report/pdf/renderer.go specs/008-report-pdf-annex/research.md`; run `go test ./internal/report/pdf`; confirm PDF output is local-only, landscape A4-sized, text-based, font-data based, uses `gopdf` layout APIs, does not render Markdown source syntax, and is not a plain line dump. |
| WU37 | Bugfix BUG-003/BUG-004/BUG-005/BUG-006: Human-legible gopdf PDF layout | T056, T057, T058, T059, T060, T061, T062, T063, T064, T065, T066, T067 | Add fail-first PDF layout regression coverage, replace line-oriented PDF output with `gopdf` layout primitives, enforce landscape A4 layout with balanced full-width tables, ~~at least 12-point section separation~~ **at least 24-point section separation under the superseding BUG-006 definition**, concise continuation labels only after actual table continuations, and bottom-margin row preflight, and validate generated layout against the expected preview using `internal/report/pdf/renderer_internal_test.go`, `internal/report/pdf/renderer.go`, `specs/008-report-pdf-annex/pdf-expected.png`, and `specs/008-report-pdf-annex/pdf-preview.png`. | WU10, WU14, WU20, WU24, WU28, WU33, WU36 | `specs/008-report-pdf-annex/bugs/BUG-003.md`, `specs/008-report-pdf-annex/bugs/BUG-004.md`, `specs/008-report-pdf-annex/bugs/BUG-005.md`, `specs/008-report-pdf-annex/bugs/BUG-006.md`, `tasks.md`, `spec.md`, `plan.md`, `research.md`, `contracts/report-rendering.md`, `quickstart.md` | Inspect `git diff -- internal/report/pdf/renderer_internal_test.go internal/report/pdf/renderer.go`; run `go test ./internal/report/pdf`; confirm the implementation uses landscape A4 page setup, `AddPage`, `AddTTFFontByReader`, `SetFont`, `Text`/`Cell`/`MultiCell`, `NewTableLayout`, `AddColumn`, `AddRow`/`AddStyledRow`, and `DrawTable`, fails for line dumps, uses the full printable width with balanced outer margins, preserves at least 24 points before the named main-report subheadings, emits `<section or table context> (continued)` only for actual table continuations, keeps rows and borders above the bottom margin through preflight, and does not expose Markdown syntax. |
| WU38 | Phase 6: Polish & Cross-Cutting Concerns | T046, T047, T048, T049, T050, T051, T055 | Parent-owned final validation: run formatting, full tests, coverage, changed-source quality, supported-OS evidence, and secret-leakage review for `internal/`, `tests/contract/`, `tests/integration/`, `tests/unit/`, `tests/testutil/`, `.github/workflows/`, `Makefile`, `dist/coverage/`, `*.go`, `go.mod`, `go.sum`, `internal/report/`, `internal/app/runtime/`, and `internal/tui/`. | WU35, WU36, WU37 | `tasks.md`, `plan.md`, `quickstart.md`, `Makefile`, `contracts/report-output.md`, `contracts/report-rendering.md`, `contracts/tui-workflows.md` | Parent must run `gofmt` on changed Go files, `go test ./internal/report/pdf`, `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`; inspect generated coverage artifacts under `dist/coverage/`; cite CI matrix results or run explicit Linux, macOS, and Windows build/test checks for PDF-enabled report packages; review report, result, diagnostic, and failure diffs for token or secret leakage before checking T055. |
| WU39 | Phase 7: Coding Standards Drift Remediation | T068, T069, T070, T071, T072, T073, T074, T075, T076, T077, T078, T079 | Execute the recorded `CODE-STAND-DRIFT-001` through `CODE-STAND-DRIFT-010` remediation plans in task order, validate the complete behavior-preserving refactor, and update only successfully remediated findings. | WU38 | `tasks.md`, `coding-standards-drift-report.md`, `plan.md`, `AGENTS.md`, `.specify/memory/constitution.md` | Inspect all remediation diffs against each report topic, run the commands in T078, and confirm T079 follows the report status-update rules with concrete resolution evidence. |
| WU40 | Phase 8: Test Coverage Drift Remediation | T080, T081, T082, T083, T084, T085, T086, T088, T089 | Remediate `COV-DRIFT-001` through `COV-DRIFT-006`, correct BUG-007's shared Markdown/PDF timing interval, then validate the required contract, integration, generated-PDF, large-history, cleanup, full-suite, performance-suite, and coverage-gate evidence. | WU39, WU29 | `tasks.md`, `test-coverage-drift-report.md`, `plan.md`, `quickstart.md`, `Makefile`, `.cov.json`, `bugs/BUG-007.md` | Inspect each remediation against its linked report topic; run the targeted commands while implementing T080-T085 and T088, then run every command in T089 and confirm `make test-performance` reports independent Markdown and PDF durations under two minutes. |

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add planned local PDF dependencies, package structure, and shared fixtures before changing report behavior.

- [X] T001 Pin planned PDF and font dependencies `github.com/signintech/gopdf@v0.36.1` and `golang.org/x/image@v0.43.0` in `go.mod` and `go.sum`, then verify `github.com/phpdave11/gofpdi@v1.0.16` is the selected transitive dependency when module resolution includes it
- [X] T002 [P] Create the local PDF renderer package skeleton and package documentation in `internal/report/pdf/renderer.go`
- [X] T003 [P] Add deterministic report-output fixture builders for format, annex, and conversion test data in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared report request, document, output bundle, and annex shell models used by every user story.

**Critical**: No user story can be completed until this phase is complete.

- [X] T004 Add `ReportOutputFormat` enum, labels, supported-format list, and validation in `internal/report/model/report_output_format.go`
- [X] T005 Extend `ReportRequest` with required output format and update constructor validation in `internal/report/model/report_request.go`
- [X] T006 Add document roles, PDF byte payload support, and output bundle models in `internal/report/model/report_document.go` and `internal/report/model/report_output_bundle.go`; keep `internal/report/model/report_output_file.go` limited to persisted save metadata
- [X] T007 Add minimal `AuditAnnex` model shell with title and section-order validation in `internal/report/model/audit_annex.go`
- [X] T008 Update calculated report validation and clone behavior for annex-aware output in `internal/report/model/capital_gains_report.go` and `internal/report/model/report_clone.go`
- [X] T009 Update runtime report outcome structs for output bundles and selected output format in `internal/app/runtime/report_types.go` and `internal/app/runtime/report_output_outcome.go`
- [X] T010 Update shared test fixture builders to supply output format and annex shell defaults in `tests/testutil/report_fixtures.go` and `tests/testutil/report_io_fixtures.go`

**Checkpoint**: Report requests, calculated reports, rendered documents, output files, and runtime outcomes can represent Markdown main-plus-annex output and combined PDF output.

---

## Phase 3: User Story 1 - Choose Report Output Format (Priority: P1) MVP

**Goal**: Users can choose Markdown or PDF before report generation, and successful results list every generated file for the selected format.

**Independent Test**: Generate the same deterministic report inputs once as Markdown and once as PDF, then confirm Markdown creates one main `.md` and one Annex 1 `.md`, PDF creates one `.pdf`, and both successful result screens show the selected format and all saved paths.

### Tests for User Story 1

- [X] T011 [P] [US1] Add report output-format workflow contract tests for selection, SC-001 selection-to-start timing evidence, busy state, and result copy in `tests/contract/report_generation_workflow_contract_test.go`
- [X] T012 [P] [US1] Add report output file contract tests for Markdown/PDF file counts, filename patterns, and suffix rules in `tests/contract/report_output_contract_test.go`
- [X] T013 [P] [US1] Add runtime integration test for generating the same fixture as Markdown and PDF in `tests/integration/report_generation_flow_test.go`
- [X] T014 [P] [US1] Add output bundle cleanup integration test for render and write failures in `tests/integration/report_failure_flow_test.go`
- [X] T015 [P] [US1] ⚠️ Reopened Add or tighten PDF renderer unit tests for landscape A4 configuration, embedded font loading through `gopdf.AddTTFFontByReader`, text emission seams, `gopdf` table/styled-layout seams, right-boundary clipping prevention, non-overlapping vertical layout, and no Markdown structural syntax in emitted PDF presentation text in `internal/report/pdf/renderer_internal_test.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T016 [P] [US1] Add output bundle writer unit tests for Markdown pair reservation and PDF filename suffixes in `internal/report/output/writer_internal_test.go`

### Implementation for User Story 1

- [X] T017 [US1] Implement output-format list state, focus movement, selection, and report request construction in `internal/tui/flow/state.go` and `internal/tui/flow/report_flow.go`
- [X] T018 [US1] Render output-format choices, selected-format explanations, busy state, and result copy in `internal/tui/flow/view.go`, `internal/tui/flow/help_text.go`, and `internal/tui/screen/report_screen.go`
- [X] T019 [US1] Implement bundle-aware Markdown rendering entry point returning main and Annex 1 documents in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_annex.go`
- [X] T020 [US1] ⚠️ Reopened Implement initial local landscape A4 PDF renderer for the main report plus Annex 1 shell through `gopdf` page, font, text, styled-cell, table-layout, printable-width, and non-overlapping vertical-flow APIs instead of Markdown-rendered body text or plain line dumps in `internal/report/pdf/renderer.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T021 [US1] Implement output bundle reservation, write, sync, close, suffixing, and cleanup for two-file Markdown and one-file PDF output in `internal/report/output/writer.go`
- [X] T022 [US1] Select renderer by output format, write output bundles, request automatic open, and shape all saved paths in `internal/app/runtime/report_service.go` and `internal/app/runtime/report_output_outcome.go`
- [X] T023 [US1] Update report result path labels to show Markdown main path plus Annex 1 path or the single PDF path in `internal/tui/screen/report_screen.go`

**Checkpoint**: User Story 1 is reopened by BUG-002, BUG-003, BUG-004, BUG-005, and BUG-006 until T015, T020, T052, T053, T054, T056, T057, T058, T059, T060, T061, T062, T063, T064, T065, T066, and T067 are completed, then can be validated independently as the MVP.

---

## Phase 4: User Story 2 - Read A Clearer Main Report (Priority: P2)

**Goal**: The main report is shorter and easier to scan without changing financial calculations or losing relevant evidence.

**Independent Test**: Generate reports containing zero net-gain summary rows, rate-source disclosures, assets without report-year activity, zero-priced SELL activities, and conversion statuses, then verify the main report content in Markdown and PDF.

### Tests for User Story 2

- [X] T024 [P] [US2] ⚠️ Reopened Add or tighten main report presentation contract tests for exact initial detail bold-label lines (`- **Year:**`, `- **Cost Basis Method:**`, `- **Generated At:**`, and `- **Report Calculation Currency:**`), zero row omission, header rename, and no main conversion audit in `tests/contract/markdown_report_contract_test.go` (reopened — BUG-001)
- [X] T025 [P] [US2] Add Markdown renderer unit tests for summary empty state, historical position, conversion status labels, and `BLOCKCHAIN OP` in `internal/report/markdown/renderer_internal_test.go`
- [X] T026 [P] [US2] ⚠️ Reopened Add PDF main report presentation unit tests mirroring shared Markdown content rules, requiring visible heading hierarchy, styled classifier labels, table headers, table rows, table columns, wrapped content, landscape table fit, non-overlapping section spacing, Rate Source Summary label/value formatting, and `Overall Yearly Net Total` inside the summary table, and rejecting Markdown heading, table, and bold markers in rendered PDF text in `internal/report/pdf/renderer_internal_test.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T027 [P] [US2] Add report output integration assertions for clearer main report content in both formats in `tests/integration/report_generation_flow_test.go`

### Implementation for User Story 2

- [X] T028 [US2] Add closed user-facing render label helpers for conversion status, quote direction, and zero-priced SELL display in `internal/report/model/render_labels.go`
- [X] T029 [US2] ⚠️ Reopened Bold the Markdown initial details block path with exact label-and-colon markup, plus rate-source classifier labels, in `internal/report/markdown/renderer.go` and `internal/report/markdown/renderer_conversion.go` (reopened — BUG-001)
- [X] T030 [US2] Omit zero net-gain summary rows and render the all-zero empty state in `internal/report/markdown/renderer_summary.go`
- [X] T031 [US2] Rename the reference header and render `Historical Position` for assets without report-year activity in `internal/report/markdown/renderer_details.go`
- [X] T032 [US2] Remove detailed Currency Conversion Audit from the main Markdown report and use label helpers for visible conversion statuses in `internal/report/markdown/renderer_conversion.go` and `internal/report/markdown/renderer_details.go`
- [X] T033 [US2] ⚠️ Reopened Apply the same main-report presentation, zero-row filtering, historical-position, label rules, styled classifier labels, readable landscape table layout, Rate Source Summary label/value formatting, summary-total table footer placement, and non-overlapping section spacing in the PDF renderer through `gopdf` layout primitives without Markdown passthrough or line dumping in `internal/report/pdf/renderer.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)

**Checkpoint**: User Story 2 PDF presentation is reopened by BUG-002, BUG-003, BUG-004, BUG-005, and BUG-006 until T026, T033, T052, T053, T054, T056, T057, T058, T059, T060, T061, T062, T063, T064, T065, T066, and T067 are completed, then can be validated independently without changing calculation outputs.

---

## Phase 5: User Story 3 - Review Annex 1 Audit Evidence (Priority: P3)

**Goal**: Annex 1 contains per-asset activity audit evidence and Currency Conversion Audit evidence for every reported asset through the selected report-year end.

**Independent Test**: Generate a report with multiple assets, historical activity, report-year activity, post-year activity, liquidations, gains or losses, reference-only assets, zero-net assets, and conversions, then verify Annex 1 content and placement in Markdown and PDF.

### Tests for User Story 3

- [X] T034 [P] [US3] Add Annex 1 rendering contract tests for title, section order, every required FR-019 per-asset visible audit field, quote labels, and empty states in `tests/contract/report_annex_contract_test.go`
- [X] T035 [P] [US3] Add calculation unit tests for annex scope including pre-year, report-year, post-year, zero-net, and reference-only assets in `internal/report/calculate/calculator_internal_test.go`
- [X] T036 [P] [US3] Add Markdown annex renderer unit tests for separate Annex 1 document content in `internal/report/markdown/renderer_internal_test.go`
- [X] T037 [P] [US3] ⚠️ Reopened Add PDF annex pagination, page-break, repeated-context, table header, row, column, wrapped-cell, Annex 1 asset-heading spacing, and no Markdown structural syntax unit tests in `internal/report/pdf/renderer_internal_test.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T038 [P] [US3] ⚠️ Reopened ~~Add 10,000 cached-activity Markdown/PDF annex scale integration test in `tests/integration/report_performance_flow_test.go`~~ Add fail-first assertions to the superseding build-tagged scenario in `tests/performance/report_performance_flow_test.go` that independently time Markdown and PDF generation against the same two-minute threshold (reopened — BUG-007)

### Implementation for User Story 3

- [X] T039 [US3] Add detailed audit annex models for per-asset sections, audit activity entries with every required FR-019 field, and the conversion audit section in `internal/report/model/audit_annex.go` and `internal/report/model/audit_activity_entry.go`
- [X] T040 [US3] Capture per-activity post-replay audit evidence through the selected year end in `internal/report/calculate/asset_replay.go` and `internal/report/calculate/artifacts.go`
- [X] T041 [US3] Build reported-asset annex sections including reference-only assets and excluding post-year activity in `internal/report/calculate/calculator.go` and `internal/report/calculate/artifacts.go`
- [X] T042 [US3] Render Annex 1 Markdown title, per-asset audit report, and Currency Conversion Audit in `internal/report/markdown/renderer_annex.go`
- [X] T043 [US3] ⚠️ Reopened Append Annex 1 after a PDF page break with `gopdf` table rows, table columns, wrapped cells, page continuation context, and sufficient top margin before same-page `Asset: <asset symbol>` subheadings through PDF-specific layout, not Markdown-rendered body text or line dumping, in `internal/report/pdf/renderer.go` (reopened — BUG-002; reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T044 [US3] Ensure runtime and output failures from missing label mappings or annex validation save no partial files in `internal/app/runtime/report_service.go` and `internal/report/output/writer.go`

**Checkpoint**: Annex 1 PDF placement is reopened by BUG-002, BUG-003, BUG-004, BUG-005, and BUG-006 until T037, T043, T052, T053, T054, T056, T057, T058, T059, T060, T061, T062, T063, T064, T065, T066, and T067 are completed, then Annex 1 is complete for Markdown and PDF and all user stories are independently testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, security review, formatting, and documentation alignment across all stories.

- [X] T045 [P] Update validation notes if implementation details differ from the planned workflow in `specs/008-report-pdf-annex/quickstart.md`
- [X] T046 [P] Run `gofmt` on changed Go files under `internal/`, `tests/contract/`, `tests/integration/`, `tests/unit/`, and `tests/testutil/`
- [X] T047 Run `make test` using `Makefile` and fix failures in changed files under `internal/` and `tests/`
- [X] T048 Run `make coverage` using `Makefile` and inspect generated coverage artifacts under `dist/coverage/`
- [X] T049 Run `make quality QUALITY_BASE_REF=origin/main` using `Makefile` and fix changed-source findings in `*.go`, `go.mod`, and `go.sum`
- [X] T050 Review generated reports, result messages, diagnostics, and failure paths for token or secret leakage in `internal/report/`, `internal/app/runtime/`, and `internal/tui/`
- [X] T051 Run or cite supported-OS validation for Linux, macOS, and Windows PDF-enabled report generation using `.github/workflows/`, `Makefile`, and changed packages under `internal/report/pdf/`, `internal/report/markdown/`, `internal/report/output/`, `internal/app/runtime/`, and `internal/tui/`
- [X] T052 [P] [US1] ⚠️ Reopened Add fail-first PDF no-Markdown-source regression tests in `internal/report/pdf/renderer_internal_test.go` that fail when PDF presentation emits Markdown heading markers, bold markers, table pipes, Markdown table separators, or a plain sequential line dump as report text (reopened — BUG-003)
- [X] T053 [US1] ⚠️ Reopened Rework or prove `internal/report/pdf/renderer.go` consumes report-domain data and `gopdf` page, font, styled text, and table-layout APIs directly instead of Markdown-rendered output or line-oriented output for main report and Annex 1 PDF content (reopened — BUG-003)
- [X] T054 [P] [US1] ⚠️ Reopened Reassess the selected PDF dependency in `specs/008-report-pdf-annex/research.md` for the exact `gopdf` APIs used for formatted headings, styled labels, tables, rows, columns, wrapping, and A4 pagination without Markdown passthrough or line dumping; evaluate a local-only alternative only if `gopdf` cannot satisfy the boundary (reopened — BUG-003)
- [X] T055 ⚠️ Reopened Run BUG-002, BUG-003, BUG-004, BUG-005, and BUG-006 final validation after PDF renderer changes using `gofmt` on changed Go files, `go test ./internal/report/pdf`, `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`; cite supported-OS evidence or run build/test checks if the PDF dependency decision changes (reopened — BUG-003; reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T056 [P] [US1] ⚠️ Reopened Add fail-first PDF layout regression tests in `internal/report/pdf/renderer_internal_test.go` that fail for simple line dumps and require landscape A4 pages, visible heading hierarchy, styled classifier labels, table headers, table rows, table columns within printable bounds, wrapped cell content, non-overlapping vertical flow, and continued table context (reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T057 [US1] ⚠️ Reopened Replace line-oriented PDF output with `gopdf` layout primitives in `internal/report/pdf/renderer.go`, using landscape A4 page setup, `AddPage`, `AddTTFFontByReader`, `SetFont`, `Text`/`Cell`/`MultiCell`, `NewTableLayout`, `AddColumn`, `AddRow`/`AddStyledRow`, `CellStyle`, and `DrawTable` for headings, styled text, custom fonts, table rows, columns, printable-width layout, and non-overlapping section flow (reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T058 [US1] ⚠️ Reopened Validate generated PDF layout against `specs/008-report-pdf-annex/pdf-expected.png` and `specs/008-report-pdf-annex/pdf-preview.png` by manual inspection or renderer seams, including landscape A4 orientation, table fit, right padding, no clipped columns, row and column readability, wrapping, continuation pages, selectable text, absence of Markdown syntax, no overlapping section text, correct summary total placement, Rate Source Summary label/value formatting, no `Reference Table` subheading, and adequate main-report and Annex 1 asset-heading top margins (reopened — BUG-004; reopened — BUG-005; reopened — BUG-006)
- [X] T059 [P] [US1] [US2] [US3] ⚠️ Reopened Add fail-first BUG-004 PDF layout regression tests in `internal/report/pdf/renderer_internal_test.go` for landscape A4 page dimensions, printable right-boundary padding, no clipped table columns, non-overlapping `Report Calculation Currency` and `Gains-And-Losses Summary` spacing, `Overall Yearly Net Total` as the final summary table row or footer, Rate Source Summary label/value formatting, absence of `Reference Table`, and top margin before `Asset Detail`, `In-Year Activity`, and Annex 1 `Asset: <asset symbol>` subheadings (reopened — BUG-005; reopened — BUG-006)
- [X] T060 [US1] [US2] [US3] ⚠️ Reopened Update `internal/report/pdf/renderer.go` to enforce BUG-004 layout rules for landscape A4 page setup, printable-width table sizing, positive vertical spacing between sections, Gains-And-Losses Summary total placement, Rate Source Summary label/value rendering, Reference Section subheading suppression, and main-report plus Annex 1 asset-heading margins (reopened — BUG-005; reopened — BUG-006)
- [X] T061 [US1] [US2] [US3] ⚠️ Reopened Validate BUG-004 generated PDF layout through renderer seams or manual evidence against the itemized defects in `specs/008-report-pdf-annex/bugs/BUG-004.md`, then rerun `go test ./internal/report/pdf` before final validation (reopened — BUG-005; reopened — BUG-006)
- [X] T062 [P] [US1] [US2] [US3] ⚠️ Reopened Add fail-first BUG-005 PDF layout regression tests in `internal/report/pdf/renderer_internal_test.go` for full printable-width table allocation with equal left and right margins, ~~at least 12 points of separation~~ **at least 24 points under the superseding BUG-006 definition** before affected main-report subheadings, and page-break preflight that keeps every continued Annex 1 row and border above the bottom margin (reopened — BUG-006)
- [X] T063 [US1] [US2] [US3] ⚠️ Reopened Update `internal/report/pdf/renderer.go` to allocate full printable-width tables with balanced outer margins, enforce ~~at least 12 points of section separation~~ **at least 24 points under the superseding BUG-006 definition**, and advance a table row before drawing any part that would cross the bottom printable margin (reopened — BUG-006)
- [X] T064 [US1] [US2] [US3] ⚠️ Reopened Validate BUG-005 generated PDF layout through renderer seams or manual evidence against `specs/008-report-pdf-annex/bugs/BUG-005.md`, including wide-table margin balance, ~~at least 12 points of affected section spacing~~ **at least 24 points under the superseding BUG-006 definition**, and multi-page Annex 1 table rows and borders, then rerun `go test ./internal/report/pdf` before final validation (reopened — BUG-006)
- [X] T065 [P] [US1] [US2] [US3] Add fail-first BUG-006 PDF layout regressions in `internal/report/pdf/renderer_internal_test.go` for at least 24 points of separation before the named main-report subheadings, exact `<section or table context> (continued)` labels on actual table continuation pages, and no continuation label for unsplit tables.
- [X] T066 [US1] [US2] [US3] Update `internal/report/pdf/renderer.go` to preserve at least 24 points of spacing before the named main-report subheadings and emit `<section or table context> (continued)` only after an actual table continuation.
- [X] T067 [US1] [US2] [US3] Validate BUG-006 generated PDF layout through renderer seams or manual evidence against `specs/008-report-pdf-annex/bugs/BUG-006.md`, including 24-point named-subheading separation, concise continued-table labels, and no label for unsplit tables, then rerun `go test ./internal/report/pdf` before final validation.

---

## Phase 7: Coding Standards Drift Remediation

**Purpose**: Resolve pending coding-standards drift after feature implementation without changing report calculations, generated-report semantics, output safety, or TUI workflow behavior.

- [X] T068 [CODE-STAND-DRIFT-001] [High] Implement the "Temporary Compatibility Architecture Remains In Production" remediation plan in `coding-standards-drift-report.md#code-stand-drift-001-temporary-compatibility-architecture-remains-in-production`: replace variadic constructors and parallel single-document paths with typed bundle-only APIs in `internal/app/runtime/report_service.go`, `internal/report/output/writer.go`, `internal/report/model/report_request.go`, `internal/report/model/report_document.go`, `internal/report/model/report_output_file.go`, and their call sites while preserving naming, collision, cleanup, opening, result-path, and generated-time behavior
- [X] T069 [CODE-STAND-DRIFT-002] [High] Implement the "Filesystem Output Package Owns Report-Domain Bundle Policy" remediation plan in `coding-standards-drift-report.md#code-stand-drift-002-filesystem-output-package-owns-report-domain-bundle-policy`: move rendered-document count, order, type, role, and shared-metadata validation from `internal/report/output/writer.go` to a model-owned validator alongside `internal/report/model/report_output_bundle.go`, move the validation matrix to model tests, and retain focused no-filesystem-work coverage in output tests
- [X] T070 [CODE-STAND-DRIFT-003] [High] Implement the "Markdown And PDF Duplicate Format-Independent Presentation Transformations" remediation plan in `coding-standards-drift-report.md#code-stand-drift-003-markdown-and-pdf-duplicate-format-independent-presentation-transformations`: add typed format-neutral row builders under `internal/report/presentation/` and migrate the duplicated transformations in `internal/report/pdf/main_report.go`, `internal/report/markdown/renderer_details.go`, `internal/report/pdf/annex_report.go`, `internal/report/markdown/renderer_annex.go`, and `internal/report/markdown/renderer_conversion.go` while leaving format-specific escaping and layout in each renderer
- [X] T071 [CODE-STAND-DRIFT-004] [High] Implement the "Conversion Audit Evidence Has Two Mutable Sources Of Truth" remediation plan in `coding-standards-drift-report.md#code-stand-drift-004-conversion-audit-evidence-has-two-mutable-sources-of-truth`: make `AuditAnnex.ConversionAuditEntries` the sole conversion-audit source in `internal/report/model/report.go`, `internal/report/model/audit_annex.go`, `internal/report/model/capital_gains_report.go`, and `internal/report/calculate/calculator.go`, then migrate validation, fixtures, and renderer assertions without changing rate-source checks or Annex output
- [X] T072 [CODE-STAND-DRIFT-005] [Medium] Implement the "Production Paths Contain Test-Only Control And Transcript Mechanisms" remediation plan in `coding-standards-drift-report.md#code-stand-drift-005-production-paths-contain-test-only-control-and-transcript-mechanisms`: remove environment-driven PDF failure behavior from `internal/app/runtime/report_service.go`, remove transcript state and appended comments from `internal/report/pdf/gopdf_document.go`, and migrate `tests/integration/report_failure_flow_test.go` plus PDF tests to injectable renderer failures, test-only layout recording, and real PDF structure/output assertions
- [X] T073 [CODE-STAND-DRIFT-006] [Medium] Implement the "Report Screen Reconstructs Workflow-Owned Output Format State" remediation plan in `coding-standards-drift-report.md#code-stand-drift-006-report-screen-reconstructs-workflow-owned-output-format-state`: pass the selected output-format domain value and flow-owned menu items through `internal/tui/flow/report_flow.go` and `internal/tui/flow/view.go` to `internal/tui/screen/report_screen.go`, delete screen-owned reconstruction helpers, and preserve index highlighting, empty guidance, and fallback behavior
- [X] T074 [CODE-STAND-DRIFT-007] [Medium] Implement the "Shared Test Fixtures Duplicate Policy And Include Unconsumed Subsystems" remediation plan in `coding-standards-drift-report.md#code-stand-drift-007-shared-test-fixtures-duplicate-policy-and-include-unconsumed-subsystems`: delete unused Annex fixtures from `tests/testutil/report_fixtures.go`, replace duplicated filename algorithms in `tests/testutil/report_io_fixtures.go` with literal collision/bundle oracle data, and remove only the fixture-policy assertion from `tests/contract/report_output_contract_test.go` while retaining production-writer naming and suffix contracts for `internal/report/output/writer.go`
- [X] T075 [CODE-STAND-DRIFT-008] [Medium] Implement the "Two Production Functions Exceed The Cognitive-Complexity Baseline" remediation plan in `coding-standards-drift-report.md#code-stand-drift-008-two-production-functions-exceed-the-cognitive-complexity-baseline`: decompose `AuditActivityEntry.Validate` in `internal/report/model/audit_activity_entry.go` and `renderDetailSections` in `internal/report/pdf/main_report.go` into cohesive ordered helpers, preserve first-error precedence and PDF rendering order, and prove every resulting production function remains below complexity 15
- [X] T076 [CODE-STAND-DRIFT-009] [Medium] Implement the "PDF Renderers Depend On An Over-Broad Lifecycle Interface" remediation plan in `coding-standards-drift-report.md#code-stand-drift-009-pdf-renderers-depend-on-an-over-broad-lifecycle-interface`: segregate the private lifecycle, content-layout, Annex-page-break, and serialization contracts in `internal/report/pdf/layout_contract.go`, narrow consumers in `internal/report/pdf/main_report.go` and `internal/report/pdf/annex_report.go`, preserve orchestration order, and remove unrelated no-op lifecycle methods from content recorders in `internal/report/pdf/renderer_internal_test.go`
- [X] T077 [CODE-STAND-DRIFT-010] [Medium] Implement the "AI-Authored APIs Lack Required Function-Level Documentation" remediation plan in `coding-standards-drift-report.md#code-stand-drift-010-ai-authored-apis-lack-required-function-level-documentation`: after T068-T076, add detailed usage examples and accurate attribution to every surviving cited public API in `internal/report/model/audit_activity_entry.go`, `internal/report/model/audit_annex.go`, `internal/report/model/report_output_bundle.go`, `internal/report/markdown/renderer.go`, `internal/report/markdown/renderer_annex.go`, and `internal/report/output/writer.go`, and add purpose comments plus attribution to surviving private PDF test-double methods in `internal/report/pdf/renderer_internal_test.go`
- [X] T078 Validate T068-T077 with `gofmt` on changed Go files, `go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 14 internal/report/model/audit_activity_entry.go internal/report/pdf/main_report.go`, `go test ./internal/app/runtime ./internal/report/model ./internal/report/presentation ./internal/report/markdown ./internal/report/pdf ./internal/report/output ./internal/tui/flow ./internal/tui/screen ./tests/testutil ./tests/unit ./tests/contract ./tests/integration ./tests/empirical/...`, followed by `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`
- [X] T079 Update `coding-standards-drift-report.md` after T078 by changing only the `Status` of `CODE-STAND-DRIFT-001` through `CODE-STAND-DRIFT-010` findings whose matching remediation completed and passed its required validation to `Resolved`, and add or update each such finding's `Resolution` notes with the implemented change and verification evidence; leave unsuccessful or incomplete findings `Pending`

---

## Phase 8: Test Coverage Drift Remediation

**Purpose**: Restore the feature-specific scenario assertions and required contract and integration test placement identified by the current test-coverage drift report without changing report calculation behavior or weakening the maintained coverage gate.

**Supersession (2026-07-13)**: T080 and T086 preserve historical remediation evidence, but issue #46 now owns resource-sensitive scenarios under build-tagged `tests/performance`. The 10,000-activity Markdown/PDF scenario runs only through `make test-performance`; canonical `make test` and `make coverage` remain deterministic and have no performance coverage profile or context.

- [X] T080 [COV-DRIFT-001] [High] ⚠️ Reopened ~~Implement the "Required Large-History Test Is Skipped And Fails When Enabled" remediation from `test-coverage-drift-report.md#cov-drift-001-required-large-history-test-is-skipped-and-fails-when-enabled`: update `tests/integration/report_performance_flow_test.go` and `tests/testutil/report_fixtures.go` so the 10,000-activity fixture exercises deterministic cross-currency activity in both Markdown and PDF, replace the obsolete raw `PAGE BREAK: Annex 1` expectation with assertions against the generated PDF contract, and update `Makefile` so the passing large-history scenario runs as maintained `make coverage` evidence while preserving the separate responsiveness coverage in `tests/integration/report_generation_responsiveness_test.go`.~~ Revalidate the superseding build-tagged scenario in `tests/performance/report_performance_flow_test.go` so it preserves the deterministic cross-currency fixture and generated-PDF output-contract evidence while T088 applies separate per-format timers; keep the resource-sensitive scenario exclusive to `make test-performance` and outside `make test` and `make coverage` (reopened — BUG-007)
- [X] T081 [COV-DRIFT-002] [High] Implement the "Generated PDF Contract And Markdown/PDF Data Parity Are Not Asserted" remediation from `test-coverage-drift-report.md#cov-drift-002-generated-pdf-contract-and-markdownpdf-data-parity-are-not-asserted`: make `tests/contract/report_output_contract_test.go` invoke the concrete PDF renderer instead of a synthetic `%PDF-1.7` payload, add project-owned generated-PDF inspection support under `tests/testutil/`, and strengthen `tests/integration/report_generation_flow_test.go` plus `internal/report/pdf/renderer_internal_test.go` to assert landscape A4 page dimensions, searchable report text, and shared report values matching the Markdown output from the same protected cache
- [X] T082 [COV-DRIFT-003] [High] Implement the "PDF Table Continuation And Wrapped-Cell Requirements Lack Regression Assertions" remediation from `test-coverage-drift-report.md#cov-drift-003-pdf-table-continuation-and-wrapped-cell-requirements-lack-regression-assertions`: extend `internal/report/pdf/renderer_internal_test.go` with long table-cell fixtures and generated-PDF or layout-recorder assertions for wrapped content within its column, repeated table headers, exact `<section or table context> (continued)` text without the forbidden prefix, no continuation label for unsplit tables, continuation-page placement, and complete rows and borders above the bottom margin; fix `internal/report/pdf/gopdf_document.go` only if those fail-first regressions expose a production defect
- [X] T083 [COV-DRIFT-004] [Medium] Implement the "Successful Markdown Result Coverage Omits The Annex Path" remediation from `test-coverage-drift-report.md#cov-drift-004-successful-markdown-result-coverage-omits-the-annex-path`: add a successful two-file Markdown result contract to `tests/contract/report_generation_workflow_contract_test.go`, use a complete main-plus-Annex bundle in `internal/tui/screen/screen_internal_test.go`, and update `tests/integration/report_generation_flow_test.go` plus `tests/integration/helpers_test.go` to assert both `Saved Markdown Path` and `Saved Annex 1 Markdown Path` with their generated paths from `internal/tui/screen/report_screen.go`
- [X] T084 [COV-DRIFT-005] [Medium] Implement the "SC-001 Contract Test Bypasses The Bubble Tea Workflow" remediation from `test-coverage-drift-report.md#cov-drift-005-sc-001-contract-test-bypasses-the-bubble-tea-workflow`: rewrite `tests/contract/report_generation_workflow_contract_test.go` to select PDF through the root Bubble Tea flow model, activate Generate, assert the immediate busy-screen transition and returned asynchronous command within the SC-001 bound, and prove report generation is not executed synchronously, using the established transition behavior in `internal/tui/flow/model_internal_test.go`, `internal/tui/flow/report_flow.go`, and `tests/integration/report_generation_responsiveness_test.go`
- [X] T085 [COV-DRIFT-006] [Medium] Implement the "Renderer-Failure Cleanup Evidence Remains Unit-Only" remediation from `test-coverage-drift-report.md#cov-drift-006-renderer-failure-cleanup-evidence-remains-unit-only`: add a runtime-backed renderer-failure journey to `tests/integration/report_failure_flow_test.go` using a deterministic injectable renderer failure equivalent to the seam covered by `internal/app/runtime/report_service_internal_test.go`, then assert the failure result, empty output directory, absence of partial Markdown or PDF files, and no opener request; expose only the smallest testable runtime assembly seam needed under `internal/app/runtime/`
- [X] T086 [COV-DRIFT-001] [COV-DRIFT-002] [COV-DRIFT-003] [COV-DRIFT-004] [COV-DRIFT-005] [COV-DRIFT-006] ⚠️ Reopened ~~Validate all remediation topics in `test-coverage-drift-report.md#findings` with `gofmt` on changed Go files, `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run '^TestReportPerformanceFlowLargeHistoryFixture$' -count=1 -v`, targeted `go test` runs for `./internal/report/pdf`, `./tests/contract`, and `./tests/integration`, followed by `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`; inspect `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` against `.cov.json` and confirm the maintained coverage run no longer skips the required both-format large-history scenario.~~ Revalidate the deterministic COV-DRIFT-002 through COV-DRIFT-006 topics with `gofmt` on changed Go files, targeted `go test` runs for `./internal/report/pdf`, `./tests/contract`, and `./tests/integration`, followed by `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`; inspect `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` against `.cov.json`, while T080 preserves the isolated scale scenario and T089 exclusively validates its per-format performance evidence through `make test-performance` (reopened — BUG-007)

### BUG-007 Performance Timing Correction

- [X] T087 [P] [US1] [US3] Add fail-first assertions in `tests/performance/report_performance_flow_test.go` that start and stop one timer for Markdown generation and another timer for PDF generation, assert each elapsed duration independently against two minutes, and include the selected output format and measured duration in each failure
- [X] T088 [US1] [US3] Update `tests/performance/report_performance_flow_test.go` so each selected-format `Generate` call has its own timing interval covering request validation, calculation, selected rendering, final save, and opener invocation, while shared fixture setup and output-contract assertions remain outside the interval where appropriate
- [X] T089 Validate BUG-007 with `gofmt` on `tests/performance/report_performance_flow_test.go`, `make test-performance`, `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main`; confirm the performance output identifies independent Markdown and PDF elapsed durations and that each is under two minutes

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup completion and blocks every user story.
- **User Story 1 (Phase 3)**: Depends on Foundational completion and is the MVP scope.
- **User Story 2 (Phase 4)**: Depends on Foundational completion. PDF parity tasks depend on the PDF renderer skeleton from T002 and are easiest after US1.
- **User Story 3 (Phase 5)**: Depends on Foundational completion. Final Markdown/PDF placement depends on the bundle and renderer selection completed in US1.
- **Polish (Phase 6)**: Depends on all desired user stories and BUG-002, BUG-003, BUG-004, BUG-005, and BUG-006 PDF rendering-boundary follow-up being complete.
- **Coding Standards Drift Remediation (Phase 7)**: Depends on Phase 6 completion. Execute T068 through T077 in order because several remediations share runtime, model, renderer, and test seams; T078 validates the combined result before T079 updates report statuses.
- **Test Coverage Drift Remediation (Phase 8)**: Depends on Phase 7 completion. Execute T080 through T085 in order because the generated-PDF inspection support and overlapping contract and integration fixtures are shared; T086 validates the combined result and maintained coverage instrumentation.
- **BUG-007 Performance Timing Correction**: Reopens T038, T080, and T086. Execute T087 fail-first, T088 implementation, and T089 validation in order; `make test-performance` remains isolated from deterministic test and coverage aggregates.

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

- Work-unit ordering controls orchestration; `[P]` markers are local hints only and do not override ledger prerequisites or parent verification.
- WU02 and WU03 can run in parallel after WU01 because they touch the PDF package skeleton and shared test fixtures separately.
- WU07 through WU11 can run in parallel once WU04 through WU06 are verified because they target separate US1 contract, integration, PDF unit, and writer unit test files.
- WU18 through WU21 can run in parallel after WU17 because they target separate US2 contract, renderer, PDF, and integration assertions.
- WU25 through WU29 can run in parallel after WU24 because they target separate US3 contract, calculation, renderer, PDF, and performance test files.
- WU23 and WU31 are independent by package after their own prerequisites, but parent verification must check shared report contracts before any dependent Markdown, PDF, runtime, or output unit starts.
- Phase 8 tasks run sequentially because T080-T085 overlap `tests/integration/`, `tests/contract/`, generated-PDF inspection support, and runtime test seams; T086 runs only after all six drift remediations pass their targeted tests.
- BUG-007 tasks run sequentially because T087 and T088 edit the same performance scenario and T089 validates the completed correction.

---

## Subagent Handoff Examples

```text
Test-oriented handoff example:
Delegate WU10 only after WU02, WU04, and WU06 are parent-verified. Include T015's exact task text, allowed path `internal/report/pdf/renderer_internal_test.go`, `contracts/report-rendering.md`, `quickstart.md`, PDF locality and text-output constraints, and the command `go test ./internal/report/pdf`. Require fail-first evidence and a final response with files changed, task IDs completed, tests run with results, expected failures, assumptions, and parent follow-up.
```

```text
Implementation-oriented handoff example:
Delegate WU15 only after WU07 through WU11 are parent-verified. Include T021's exact task text, allowed path `internal/report/output/writer.go`, `contracts/report-output.md`, `data-model.md`, failure-cleanup and Documents-directory constraints, and the command `go test ./internal/report/output`. Require the subagent to stop before editing runtime, TUI, renderer, generated coverage, or empirical fixture paths.
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

### Context-Orchestrated Subagent Strategy

1. Parent agent delegates one ledger work unit at a time with a complete handoff packet and no reliance on prior conversation state.
2. Subagents may run only within the unit's allowed paths and must stop before touching adjacent packages unless they report the need for parent approval.
3. Parent verification runs the ledger command, inspects scoped diffs, checks forbidden paths, and re-reads affected contracts before checking task boxes.
4. Parent retains WU38 final validation ownership even if a subagent helps interpret command output.

---

## Notes

- `[P]` tasks are safe to run in parallel only when the listed files are not already being edited by another active task.
- Every generated report file is intentional cleartext local output and must exclude Ghostfolio tokens, bearer tokens, reusable authentication material, protected payload bytes, and unrelated secrets.
- PDF generation must stay local-only and must not use remote services, browser services, external binaries, platform font paths, or user-installed fonts.
- Exact decimal report calculation behavior must remain unchanged.
- Treat `testdata/empirical/` as read-only for this feature.
- The work-unit ledger controls subagent delegation; do not mark ledger units or task checkboxes complete until parent verification passes.
- A compacted parent session must resume from the first ledger row with unchecked referenced tasks and inspect existing diffs before delegating more work.
