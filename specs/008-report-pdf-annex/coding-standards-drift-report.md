# Coding Standards Drift Report: Capital Gains Report PDF And Audit Annex

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-07-11
**Updated**: 2026-07-11
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-control.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- The reviewed implementation surface was derived from `spec.md`, `plan.md`, `tasks.md`, the supporting feature artifacts, and the feature files changed from `origin/main`.

## Standards Baseline

- [`AGENTS.md`](../../AGENTS.md): reporting, runtime, output, and TUI package boundaries at lines 75-86 and 106-124; descriptive naming, SOLID, SRP, decomposition, DRY, consistency, cognitive-complexity, file-cohesion, and layered-architecture rules at lines 158-182; AI-generated code documentation and attribution requirements at lines 186-210.
- [`.specify/memory/constitution.md`](../../.specify/memory/constitution.md): Clean Architecture and Domain Clarity at lines 162-175, including explicit domain concepts, cohesion, minimized duplication, SOLID boundaries, domain separation from IO, and mandatory consistency.

## Findings

### CODE-STAND-DRIFT-001: Temporary Compatibility Architecture Remains In Production

**Status**: Pending
**Severity**: High
**Diverges from**:

- `AGENTS.md:158-182` - descriptive APIs, SRP, decomposition, DRY, consistency, and layered architecture.
- `.specify/memory/constitution.md:165-173` - explicit domain concepts, cohesive functions, minimized duplication, and mandatory consistency.

**Evidence**:

- `internal/app/runtime/report_service.go:43-61`
- `internal/app/runtime/report_service.go:77-80`
- `internal/app/runtime/report_service.go:151-153`
- `internal/app/runtime/report_service.go:189-271`
- `internal/report/output/writer.go:36-201`
- `internal/report/model/report_request.go:32-81`
- `internal/report/model/report_document.go:36-87`
- `internal/report/model/report_output_file.go:36-86`

**Description**:

The completed feature retains parallel legacy single-document and current output-bundle renderer/writer seams in runtime and output production code. The model constructors similarly accept old and current call shapes through runtime-parsed `...any` arguments. These paths are explicitly described as migration or older-test compatibility code. They duplicate report-generation architecture, mix compatibility dispatch with construction and orchestration, replace compile-time parameter checking with vague dynamic arguments, and leave multiple production paths that must evolve together.

### CODE-STAND-DRIFT-002: Filesystem Output Package Owns Report-Domain Bundle Policy

**Status**: Pending
**Severity**: High
**Diverges from**:

- `AGENTS.md:106-112` - report models own report-document and output metadata validation; output owns local naming, file writing, cleanup, and opening.
- `AGENTS.md:174-182` - layered architecture and file/module SRP.
- `.specify/memory/constitution.md:168-171` - cohesive modules and separation of domain rules from IO and infrastructure.

**Evidence**:

- `internal/report/output/writer.go:311-382`
- `internal/report/model/report_output_bundle.go:57-127`

**Description**:

The filesystem writer validates Markdown/PDF document counts, types, ordering, roles, and shared report metadata before writing. Those are report-domain composition rules, while the same output-shape concepts already exist in model validation. Placing this policy in `internal/report/output` mixes domain validation with filesystem infrastructure and makes document-bundle rules span two package authorities.

### CODE-STAND-DRIFT-003: Markdown And PDF Duplicate Format-Independent Presentation Transformations

**Status**: Pending
**Severity**: High
**Diverges from**:

- `AGENTS.md:158-165` - SRP, decomposition, DRY, and consistency.
- `.specify/memory/constitution.md:165-173` - explicit domain concepts, minimized duplication, domain separation, and mandatory consistency.

**Evidence**:

- `internal/report/pdf/main_report.go:249-304`
- `internal/report/markdown/renderer_details.go:96-155`
- `internal/report/pdf/main_report.go:337-367`
- `internal/report/markdown/renderer_details.go:179-212`
- `internal/report/pdf/annex_report.go:85-162`
- `internal/report/markdown/renderer_annex.go:80-160`
- `internal/report/pdf/annex_report.go:202-251`
- `internal/report/markdown/renderer_conversion.go:64-118`

**Description**:

Both renderers independently canonicalize the same decimal fields, derive the same labels and timestamps, build the same logical rows, and shape equivalent error context. Only the final table encoding is format-specific. Keeping these transformations in both output adapters requires report-visible semantics to remain synchronized by convention and creates clear maintenance risk whenever a row or label changes.

### CODE-STAND-DRIFT-004: Conversion Audit Evidence Has Two Mutable Sources Of Truth

**Status**: Pending
**Severity**: High
**Diverges from**:

- `AGENTS.md:164-165` - avoid duplication and remain consistent.
- `.specify/memory/constitution.md:165-173` - model domain concepts explicitly, minimize duplication, and preserve consistency.

**Evidence**:

- `internal/report/model/report.go:63-78`
- `internal/report/model/audit_annex.go:81-89`
- `internal/report/calculate/calculator.go:187-218`
- `internal/report/model/capital_gains_report.go:160-175`

**Description**:

`CapitalGainsReport` stores conversion audit entries both directly and inside `AuditAnnex`. Calculation populates both slices, while validation copies the top-level slice only when the annex slice is empty and establishes no equality invariant when both are populated. One report value can therefore carry divergent copies of the same domain evidence, contrary to the repository's DRY and consistency requirements.

### CODE-STAND-DRIFT-005: Production Paths Contain Test-Only Control And Transcript Mechanisms

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:75-78` - runtime coordinates application workflows rather than interpreting hidden test configuration.
- `AGENTS.md:158-182` - SRP, cohesion, and layered architecture.
- `.specify/memory/constitution.md:168-171` - cohesive modules and separation from infrastructure concerns.

**Evidence**:

- `internal/app/runtime/report_service.go:9-10`
- `internal/app/runtime/report_service.go:274-299`
- `tests/integration/report_failure_flow_test.go:472-477`
- `internal/report/pdf/gopdf_document.go:36-46`
- `internal/report/pdf/gopdf_document.go:349-379`

**Description**:

Runtime changes production PDF rendering when the test-specific `GHOSTFOLIO_CRYPTOGAINS_PDF_RENDER_FAILURE` environment variable is set, despite existing injectable renderer seams. Separately, the PDF adapter maintains a second text transcript for deterministic assertions and appends it as comments to every production PDF payload. These mechanisms give production orchestration and rendering additional test-support responsibilities instead of keeping failure injection and assertions behind test seams.

### CODE-STAND-DRIFT-006: Report Screen Reconstructs Workflow-Owned Output Format State

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:120-124` - flow owns workflow state and transitions; screen owns rendering.
- `AGENTS.md:158-165` - DRY, consistency, and SRP.
- `.specify/memory/constitution.md:168-173` - cohesive boundaries and mandatory consistency.

**Evidence**:

- `internal/tui/flow/menu_items.go:109-118`
- `internal/tui/flow/report_flow.go:163-185`
- `internal/tui/screen/report_screen.go:214-246`

**Description**:

The flow layer already builds the supported output-format menu and maps stable selection indexes to domain values. The screen layer independently rebuilds the same menu and re-derives a format by comparing display labels and indexes. This duplicates mapping policy and makes rendering interpret workflow state that belongs to `internal/tui/flow`.

### CODE-STAND-DRIFT-007: Shared Test Fixtures Duplicate Policy And Include Unconsumed Subsystems

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:158-182` - descriptive names, SRP, decomposition, DRY, consistency, and file cohesion.
- `.specify/memory/constitution.md:168-169` - cohesive functions/modules and minimized duplication.

**Evidence**:

- `tests/testutil/report_fixtures.go:61-103`
- `tests/testutil/report_fixtures.go:582-655`
- `tests/testutil/report_io_fixtures.go:116-146`
- `tests/testutil/report_io_fixtures.go:460-472`
- `internal/report/output/writer.go:446-480`
- `tests/contract/report_output_contract_test.go:52-64`

**Description**:

The exported Annex fixture types and `DeterministicReportAnnexFixture` have no consumers outside their declaration file, while feature tests construct their own report data. The IO fixture also reimplements production filename construction, and the dedicated filename contract test validates those fixture-generated strings against hard-coded expressions instead of exercising the production naming result. This leaves unused fixture code and a second filename-policy implementation that can pass while production behavior drifts.

### CODE-STAND-DRIFT-008: Two Production Functions Exceed The Cognitive-Complexity Baseline

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:170-173` - production Go functions should remain below cognitive complexity 15 and require SRP/decomposition analysis when they exceed it.

**Evidence**:

- `internal/report/model/audit_activity_entry.go:39-86`
- `internal/report/pdf/main_report.go:164-188`

**Description**:

`gocognit v1.2.1` reports complexity 16 for both `AuditActivityEntry.Validate` and `renderDetailSections`. The validator combines required fields, decimal groups, currency checks, and optional conversion validation. The PDF function combines collection traversal, historical/active classification, four rendering stages, and contextual error shaping. Both exceed the explicit repository baseline and retain multiple responsibilities in one function.

### CODE-STAND-DRIFT-009: PDF Renderers Depend On An Over-Broad Lifecycle Interface

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:161-163` - SOLID, with special importance given to SRP.
- `.specify/memory/constitution.md:168-169` - cohesive modules and SOLID boundaries where they improve clarity and change safety.

**Evidence**:

- `internal/report/pdf/layout_contract.go:5-31`
- `internal/report/pdf/main_report.go:164-188`
- `internal/report/pdf/annex_report.go:167-199`
- `internal/report/pdf/renderer_internal_test.go:851-853`
- `internal/report/pdf/renderer_internal_test.go:922-931`

**Description**:

`pdfLayoutDocument` combines document startup, font registration, report-content layout, page breaking, and byte serialization. Main-report and annex rendering depend on the complete lifecycle interface even though they use only content-layout operations. Test doubles consequently implement unrelated startup, font, and serialization methods as no-ops, providing concrete evidence that the interface is not segregated by consumer responsibility.

### CODE-STAND-DRIFT-010: AI-Authored APIs Lack Required Function-Level Documentation

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:188-210` - AI-generated private functions require purpose documentation, public functions require detailed purpose and usage examples, and agent-touched code requires authoring information at the documented levels.
- `AGENTS.md:222-223` - AI-generated code is the exception to the normal minimal-comment rule.

**Evidence**:

- `internal/report/model/audit_activity_entry.go:37-39`
- `internal/report/model/audit_annex.go:115-143`
- `internal/report/model/report_output_bundle.go:57-60`
- `internal/report/markdown/renderer.go:69-84`
- `internal/report/markdown/renderer_annex.go:22-54`
- `internal/report/output/writer.go:153-156`
- `internal/report/pdf/renderer_internal_test.go:851-893`
- `internal/report/pdf/renderer_internal_test.go:922-954`

**Description**:

Multiple exported, explicitly AI-authored functions and methods have only one-line descriptions and no usage examples, including model validation APIs and Markdown rendering entry points. `WriteReportDocuments` also lacks the detailed usage guidance required for public APIs. In PDF tests, AI-authored recorder and failure-double methods have no function-level purpose comments or attribution at all. File-level or type-level attribution does not satisfy the baseline's explicit method/function-level documentation requirement.

## Notes

- No prior `coding-standards-drift-report.md` existed, so all findings are newly assigned and `Pending`.
- Prerequisite validation used the local Spec Kit task syntax and found no unchecked implementation task in `tasks.md`; previously reopened tasks are checked.
- Cognitive complexity was measured with `go run github.com/uudashr/gocognit/cmd/gocognit@v1.2.1 -over 15` over the active production packages. Test-function results were excluded because `AGENTS.md` exempts tests from the cognitive-complexity rule.
- No finding status was set to `Resolved`, and no remediation plan or task checkbox was added.
