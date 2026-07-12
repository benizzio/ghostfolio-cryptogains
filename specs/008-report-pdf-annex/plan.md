# Implementation Plan: Capital Gains Report PDF And Audit Annex

**Branch**: `[008-report-pdf-annex]` | **Date**: 2026-07-03 | **Spec**: `/specs/008-report-pdf-annex/spec.md`

**Input**: Feature specification from `/specs/008-report-pdf-annex/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add an output-format choice to capital gains report generation so users can generate either the existing Markdown output or a new local landscape A4 text PDF. The existing report calculation rules remain authoritative. The implementation will extend the report model with an audit annex and generated-output bundle, move detailed Currency Conversion Audit rows into Annex 1, render Markdown as a main file plus a separate annex file, render PDF as one searchable text file containing the main report and Annex 1 after a page break, and update TUI/runtime/output flows to report every generated file or fail without presenting partial output.

**Bugfix**: 2026-07-05 — [BUG-001] Updated from bugfix patch. Markdown initial detail verification must assert exact bold classifier label syntax.

**Bugfix**: 2026-07-05 — [BUG-002] Updated from bugfix patch. PDF rendering must format report-domain data through PDF-specific layout and must not pass Markdown source text into the PDF body.

**Bugfix**: 2026-07-07 — [BUG-003] Updated from bugfix patch. PDF rendering must use concrete `gopdf` layout APIs for human-legible headings, styled labels, tables, rows, columns, wrapping, and continuation context rather than plain line dumping.

**Bugfix**: 2026-07-09 — [BUG-004] Updated from bugfix patch. PDF rendering must use landscape A4 pages, keep tables inside printable bounds, preserve non-overlapping vertical spacing, and match section-specific Markdown presentation meanings where required.

**Bugfix**: 2026-07-09 — [BUG-005] Updated from bugfix patch. PDF rendering must allocate the full printable width with balanced outer margins, preserve at least 12 points of section separation, and preflight table rows before the bottom margin.

**Bugfix**: 2026-07-10 — [BUG-006] Updated from bugfix patch. The named main-report subheadings require 24 points of separation, and only actual table continuation pages may render `<section or table context> (continued)`.

## Technical Context

**Language/Version**: Go 1.26.5

**Primary Dependencies**: Existing dependencies: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/Fabianexe/gocoverageplus`, and Go standard library packages. Planned new PDF dependencies: `github.com/signintech/gopdf v0.36.1` for local PDF generation, `golang.org/x/image v0.43.0` for embedded Go font TTF data loaded through `gopdf.AddTTFFontByReader`, and `github.com/phpdave11/gofpdi v1.0.16` as a refreshed indirect dependency if Go module resolution permits it with `gopdf`.

**PDF Renderer API Shape**: The `internal/report/pdf` renderer must construct a `gopdf.GoPdf`, initialize it with landscape A4 page dimensions through `Start(gopdf.Config{PageSize: ...})` using a library-provided landscape A4 page size or an A4 rectangle with width and height swapped, add pages with `AddPage`, register embedded regular and bold fonts with `AddTTFFontByReader`, select fonts with `SetFont`, position content with `SetXY`, `GetX`, and `GetY`, render headings and styled labels with `Text`, `Cell`, or `MultiCell`, and produce structured report tables with `NewTableLayout`, `AddColumn`, `AddRow` or `AddStyledRow`, `CellStyle`, `BorderStyle`, `RGBColor`, and `DrawTable`. Where explicit separators or section boxes are needed, use `Line`, `SetLineWidth`, `SetStrokeColor`, `RectFromUpperLeftWithStyle`, or table border styles. Table widths must be derived from the landscape printable area so columns retain right padding and never clip at the page edge. Vertical flow must add positive spacing before section titles, subheadings, and tables, or advance to a new page when the next block would collide with preceding content. The rendered PDF bytes should be obtained through `WriteTo` or `GetBytesPdf` before output writing. A renderer that only iterates report-domain strings into sequential `Text` or `Cell` lines without table layout, styled cells, wrapped cell content, and continuation context does not satisfy this plan.

**BUG-005 PDF Layout Clarification**: Table widths must consume the full landscape printable width while retaining equal left and right outer margins, right padding, and wrapped cell content without clipping at the page edge. Vertical flow must use at least 12 points of separation before affected section titles, subheadings, and tables. Before drawing a table row or border, the renderer must preflight its complete height against the remaining printable height; if it would cross the bottom margin, it must advance to a continuation page before drawing any part of the row.

**BUG-006 PDF Layout Clarification**: The `Gains-And-Losses Summary`, `Rate Source Summary`, `Reference Section`, `Asset Detail: <asset symbol>`, and `In-Year Activity` subheadings must reserve at least 24 points of vertical separation from preceding same-page content. Only a table that has advanced to a new page may emit continuation context, in the exact format `<section or table context> (continued)` without a `Continued: ` prefix.

**Storage**: User-requested cleartext report export files in the resolved local Documents directory. These exports are outside the application-managed persistence boundary: the application writes them only after an explicit generation request, does not manage them as cache or durable application state, and does not re-ingest them automatically. Markdown output writes one main `.md` file and one Annex 1 `.md` file. PDF output writes one `.pdf` file. No synced-data persistence, protected snapshot persistence, report history cache, rate cache, remote persistence, telemetry, or background report storage is added.

**Testing**: Go `testing` with deterministic contract, integration, unit, and existing empirical suites plus isolated build-tagged performance scenarios. Contract tests cover TUI/output/rendering contracts, including the output-format selection path used as automated evidence for SC-001's 30-second user-start bound. Integration tests cover runtime generation and output cleanup. The isolated performance suite covers the 10,000 cached-activity scale. Targeted unit tests are justified for PDF pagination/text emission, filename construction, label mapping, zero-row filtering, historical-position rendering, and audit-annex model validation. Final verification uses `make test`, `make coverage`, isolated `make test-performance`, `make quality QUALITY_BASE_REF=origin/main`, and supported-OS build/test evidence for Linux, macOS, and Windows unless successful CI checks are cited.

**Empirical Dataset**: Existing `testdata/empirical/` synthetic empirical dataset and oracle fixtures remain read-only. Existing empirical tests continue to guard financial calculation behavior. This feature adds presentation and audit-traceability coverage around calculated results and does not mutate empirical source data or generated oracle fixtures.

**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access. PDF generation runs in-process and local-only.

**Project Type**: Single-module Go terminal UI application.

**Performance Goals**: Preserve the existing report-generation scale target of 10,000 cached activities. Keep report generation asynchronous from the Bubble Tea event loop. Avoid a lower activity-count limit for PDF or Annex 1 generation than Markdown output. Avoid rasterizing report pages into images.

**Constraints**: No floating-point financial logic. Exact decimals and explicit currency identity remain unchanged from the current report calculation pipeline. PDF generation must be local-only, landscape A4-sized, text-based, searchable, and selectable by PDF readers that support text selection. Generated outputs and failures must not expose Ghostfolio tokens, bearer tokens, reusable authentication material, protected payload bytes, or unrelated secrets. Failed render or write attempts must not be reported as successful and must remove partial output files created by the attempt.

**Scale/Scope**: One report output format selection (`Markdown` or `PDF`), one audit annex model, one Markdown main-plus-annex renderer flow, one PDF renderer flow, output bundle writing for one or two files, report-result display of all generated files, and no new external document-generation service.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Generated Markdown and PDF reports are intentional user-requested cleartext local export files outside the application-managed persistence boundary. The application does not manage those exports as cache or durable application state after generation, and does not re-ingest them automatically. No new protected persistence, synced-data persistence, report cache, telemetry, remote storage, or remote document-generation service is introduced. Ghostfolio tokens, bearer tokens, reusable verifiers, and protected payload bytes remain excluded from reports, annexes, result screens, errors, diagnostics, examples, and fixtures. OWASP Top 10 review scope covers broken access control in report workflows, cryptographic failures from token leakage, injection into rendered Markdown/PDF text, insecure design from partial-output success, security misconfiguration from remote PDF services, vulnerable components from the PDF dependency, identification/authentication leakage, software/data integrity of generated audit evidence, logging/diagnostic leakage, and SSRF avoidance by not adding remote document generation.
- [x] Precision: This feature changes output format, report presentation, and audit disclosure only. Financial amounts, quantities, rates, basis, proceeds, gains, and losses remain exact decimals using existing report models and `apd/v3`. Existing selected-currency and report-base-currency rules remain authoritative. No new conversion source, conversion boundary, rounding rule, or calculation method is authorized.
- [x] Testing: Integration-first coverage will verify TUI format selection, runtime output generation for Markdown and PDF, all generated file paths in results, partial-output cleanup, Annex 1 placement/content, and the 10,000 cached-activity scale. Targeted unit tests are justified for renderer and filename behavior that is isolated from external services. Coverage gates remain mandatory.
- [x] Quality gate: Source changes are expected in `*.go`, `go.mod`, and `go.sum`. Final changed-source quality verification is `make quality QUALITY_BASE_REF=origin/main`, or the successful `Quality` GitHub Actions check for this branch/PR. `make test` and `make coverage` remain deterministic project validation commands; resource-sensitive verification runs separately through `make test-performance`.
- [x] Empirical financial validation: Existing empirical tests under `tests/empirical/` and data under `testdata/empirical/` remain read-only and continue to guard financial calculations. This feature does not change financial calculation rules and does not authorize empirical dataset or oracle mutation.
- [x] Dependencies and external integrations: `github.com/signintech/gopdf v0.36.1` is planned because Go standard library PDF generation would create disproportionate code size and higher format risk for A4 pagination, fonts, and text objects. `golang.org/x/image v0.43.0` is planned only for Go project's embedded font bytes so PDF generation does not rely on platform font paths. `github.com/phpdave11/gofpdi v1.0.16` is the gopdf transitive PDF import dependency target. No remote PDF API, browser service, cloud renderer, telemetry, or new external data API is introduced.
- [x] Architecture: Report calculation stays under `internal/report/calculate/`. Report-domain output format, annex, document, and bundle models stay under `internal/report/model/`. Markdown rendering stays under `internal/report/markdown/`. PDF rendering is isolated under a new `internal/report/pdf/` package. Local file writing and cleanup stay under `internal/report/output/`. Runtime coordinates calculation, renderer selection, output writing, and result shaping. TUI captures transient choices and renders workflow state only.

## Project Structure

### Documentation (this feature)

```text
specs/008-report-pdf-annex/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── report-output.md
│   ├── report-rendering.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── app/
│   └── runtime/              # Select renderer, write output bundle, shape result paths and failures
├── report/
│   ├── model/                # Output format, report bundle, audit annex, audit activity, label contracts
│   ├── calculate/            # Preserve existing calculation, emit through-year audit activity evidence
│   ├── markdown/             # Render main Markdown and separate Annex 1 Markdown document
│   ├── pdf/                  # Render local landscape A4 searchable text PDF from calculated report and annex
│   └── output/               # Reserve/write/cleanup one-file PDF or two-file Markdown output bundles
└── tui/
    ├── flow/                 # Output-format selection state and report request construction
    └── screen/               # Selection, busy, and result copy including output format and all saved paths

tests/
├── contract/                 # TUI, report rendering, output filename, and saved-path contracts
├── integration/              # Deterministic runtime generation, failure cleanup, and no-secret outputs
├── performance/              # Build-tagged 10,000-activity report and responsiveness scenarios
├── unit/                     # Renderer, label, filename, annex model, PDF layout unit coverage
└── empirical/                # Existing read-only financial calculation evidence
```

**Structure Decision**: Keep the feature inside the existing single Go module and existing report/runtime/TUI boundaries. Add a dedicated `internal/report/pdf/` package instead of placing PDF layout in runtime, output, TUI, or Markdown packages. Extend report models with output and annex concepts before rendering so Markdown and PDF share the same calculated data and audit evidence.

## Rendering And Output Boundary

Report generation remains a staged pipeline:

1. TUI builds a validated report request with year, cost-basis method, report base currency, and output format.
2. Runtime reads the unlocked protected cache and invokes report calculation.
3. Report calculation emits the existing main report data plus Annex 1 audit evidence for every activity on or before the selected year end.
4. Runtime selects the renderer from output format.
5. Markdown rendering returns a main report document and a separate Annex 1 document.
6. PDF rendering returns one landscape A4 text PDF document with a page break before Annex 1, produced from report-domain data through `gopdf` page, font, text, styled-cell, and table-layout APIs rather than Markdown-rendered body text or plain sequential line dumping.
7. Output writing reserves and writes the complete bundle, cleaning up every file created by the attempt if any write, sync, close, render, or validation step fails before success.
8. Runtime result screens report every generated file path for successful generation.

## PDF Dependency Decision

Planned direct PDF generation dependency: `github.com/signintech/gopdf v0.36.1`.

Recorded evidence:

- Module query returned `v0.36.1`, published from 2026-05-19.
- GitHub repository `signintech/gopdf` is not archived, has about 2,908 stars, and had commits on 2026-05-18 and 2026-05-19.
- Context7 documentation shows A4 page setup through `gopdf.Config{PageSize: *gopdf.PageSizeA4}`, text APIs such as `Text` and `Cell`, table layout APIs, `WritePdf`, and font registration through `AddTTFFontByReader`.
- The library generates PDFs in-process and does not require a browser, external binary, remote service, or CGO boundary for the planned use.
- Required implementation APIs for BUG-003 are `Start`, `AddPage`, `AddTTFFontByReader`, `SetFont`, `SetXY`, `Text`, `Cell` or `MultiCell`, `NewTableLayout`, `AddColumn`, `AddRow` or `AddStyledRow`, `CellStyle`, `BorderStyle`, `RGBColor`, `DrawTable`, and `WriteTo` or `GetBytesPdf`; drawing helpers such as `Line`, `SetLineWidth`, `SetStrokeColor`, and `RectFromUpperLeftWithStyle` may be used for section rules or boxes when table styles are insufficient.

Planned direct font-data dependency: `golang.org/x/image v0.43.0`.

Recorded evidence:

- Module query returned `v0.43.0`, published from 2026-06-15.
- The planned use is limited to Go project's embedded Go regular/bold TTF data so generated PDFs have deterministic embedded fonts across Linux, macOS, and Windows.
- Font bytes are loaded into `gopdf` through `AddTTFFontByReader`; the runtime does not read system font paths and does not fetch fonts remotely.

Planned transitive PDF import dependency target: `github.com/phpdave11/gofpdi v1.0.16`.

Recorded evidence:

- `gopdf v0.36.1` currently requires `github.com/phpdave11/gofpdi` as a transitive dependency.
- Module query returned `v1.0.16`, published from 2026-04-15.
- GitHub release `v1.0.16` and commits on 2026-04-15 show active maintenance and fixes for PDF stream/import handling.

Security posture:

- The selected design uses the dependency only to generate local text PDF output from already calculated in-memory report data.
- The renderer must not import user-provided PDFs, fetch remote resources, execute external binaries, or accept remote templates.
- Final implementation must pin module versions in `go.mod`/`go.sum`, verify the intended `github.com/phpdave11/gofpdi v1.0.16` transitive module when module resolution selects it, and pass the repository changed-source quality gate, including `govulncheck` as run by `make quality`.
- BUG-002 follow-up must verify the selected `gopdf` usage can produce formatted headings, table-like layout, wrapping, and A4 pagination without Markdown passthrough; if it cannot, evaluate another local-only option that preserves the no-browser, no-remote-service, no-OS-print-to-PDF, no-platform-font-path, and no-telemetry constraints.
- BUG-003 follow-up must verify the implementation actually calls the selected `gopdf` layout APIs for headings, styled labels, tables, rows, columns, wrapped cells, and continuation context, and must reject an implementation that only emits sequential report lines.
- BUG-004 follow-up must verify `gopdf` is configured for landscape A4 pages and that renderer layout policy prevents clipped tables, overlapping adjacent text blocks, misplaced summary totals, and generated helper subheadings that are not part of the report contract.
- BUG-005 follow-up must verify table layouts consume the available landscape printable width with balanced outer margins, section transitions retain at least 12 points of separation, and table-row preflight prevents any row or border from crossing the bottom printable margin.
- BUG-006 follow-up must verify the named main-report subheadings retain at least 24 points of separation, a continued table uses only `<section or table context> (continued)`, and an unsplit table emits no continuation label.

Alternatives rejected:

- Manual PDF writing with only the standard library would reduce dependencies but would increase implementation and format risk for pagination, fonts, tables, text objects, and cleanup behavior.
- `pdfcpu/pdfcpu` is actively maintained and useful as PDF tooling, but it is primarily a PDF processor/CLI/library and is not the smallest direct fit for rendering report tables from domain models.
- `johnfercher/maroto` provides a higher-level layout API but adds another abstraction on top of a PDF generator and had only a dependency-update commit in the previous three months.
- `jung-kurt/gofpdf` is archived and therefore fails the constitution's maintenance requirement.
- HTML-to-PDF via browser automation or remote conversion services is rejected because it adds external process/service risk and conflicts with local-only generation.

## Failure Handling

- Invalid output format fails request validation before report generation begins.
- Missing user-friendly label mappings for conversion status or quote direction fail rendering before final output is reported successful.
- Markdown bundle writes must clean up the main file if annex writing fails and must clean up the annex file if a later operation fails before success.
- PDF rendering failures return a non-secret failure and leave no final successful PDF path.
- Automatic-open failures remain non-fatal only after all requested output files have been saved successfully.
- Failure and diagnostic messages identify non-secret context only and must not expose tokens, raw protected payload data, or reusable authentication material.

## Testing Strategy

- Contract tests verify report output format choices, TUI selection copy, output filename patterns, generated file path reporting, main report rendering changes, Annex 1 rendering order, and PDF landscape A4/text-output contract.
- PDF renderer tests must fail if generated PDF presentation text contains Markdown structural syntax such as heading markers, table pipes or separators, or bold markers instead of PDF-formatted report text.
- PDF renderer tests must fail if the PDF path is only a selectable text line dump; tests must require visible heading hierarchy, styled classifier labels, table headers, table rows, table columns, wrapped cell content, and continuation context through the `gopdf` layout boundary.
- PDF renderer tests must fail if pages are not landscape A4, if table columns exceed the printable width or clip at the right edge, if adjacent section text overlaps, if `Overall Yearly Net Total` is outside the Gains-And-Losses Summary table, if Rate Source Summary renders as a table instead of bold label/value lines, if `Reference Table` is generated, or if main-report and Annex 1 asset subheadings have insufficient top margin.
- BUG-005 PDF renderer tests must fail if tables do not consume the available printable width with balanced outer margins, affected section transitions have less than 12 points of separation, or a continued table row or border crosses the bottom margin.
- BUG-006 PDF renderer tests must fail if the named main-report subheadings have less than 24 points of separation, a continued table has a `Continued: ` prefix or an incorrect label, or an unsplit table emits continuation context.
- Markdown renderer and contract tests must assert exact initial detail list-item labels `- **Year:**`, `- **Cost Basis Method:**`, `- **Generated At:**`, and `- **Report Calculation Currency:**` before report values.
- Contract tests verify that the user can select an output format and start generation from the report generation step without waiting on rendering or file IO in the Bubble Tea event loop; this is the automated workflow evidence for SC-001's 30-second bound.
- Integration tests generate Markdown and PDF from the same deterministic protected cache and assert shared report data matches between formats except for pagination, page titles, and Markdown annex splitting.
- Integration tests verify Markdown success creates exactly two files and PDF success creates exactly one file.
- Integration tests cover render/write failures and assert partial files created by the attempt are removed.
- The isolated performance suite covers 10,000 cached activities for Markdown and PDF generation with deterministic currency-rate fixtures from the existing conversion feature; ordinary integration and coverage targets do not run resource-sensitive scenarios.
- Unit tests cover output filename construction, summary zero-row filtering and empty state, historical-position rendering, conversion-status and quote-direction labels, zero-priced `SELL` display as `BLOCKCHAIN OP`, PDF page-break/layout decisions, and audit-annex model validation.
- Supported-OS evidence covers Linux, macOS, and Windows build/test compatibility for the PDF-enabled report packages, using CI matrix results or explicit local cross-OS build checks where CI evidence is unavailable.
- Existing empirical financial tests remain unchanged and continue to verify calculation behavior.

## Complexity Tracking

BUG-004 adds concrete layout edge-case validation for wide tables, section transitions, and section-specific PDF presentation. It does not add a constitution violation.

BUG-005 adds layout-width allocation, readable spacing, and page-bottom row-continuation validation. It does not add a constitution violation.

BUG-006 adds a stricter subheading spacing threshold and a continuation-label predicate. It does not add a constitution violation.

No constitution violations are planned.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
