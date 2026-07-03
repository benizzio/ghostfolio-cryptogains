# Implementation Plan: Capital Gains Report PDF And Audit Annex

**Branch**: `[008-report-pdf-annex]` | **Date**: 2026-07-03 | **Spec**: `/specs/008-report-pdf-annex/spec.md`

**Input**: Feature specification from `/specs/008-report-pdf-annex/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add an output-format choice to capital gains report generation so users can generate either the existing Markdown output or a new local A4 text PDF. The existing report calculation rules remain authoritative. The implementation will extend the report model with an audit annex and generated-output bundle, move detailed Currency Conversion Audit rows into Annex 1, render Markdown as a main file plus a separate annex file, render PDF as one searchable text file containing the main report and Annex 1 after a page break, and update TUI/runtime/output flows to report every generated file or fail without presenting partial output.

## Technical Context

**Language/Version**: Go 1.26.3

**Primary Dependencies**: Existing dependencies: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/Fabianexe/gocoverageplus`, and Go standard library packages. Planned new PDF dependencies: `github.com/signintech/gopdf v0.36.1` for local PDF generation, `golang.org/x/image v0.43.0` for embedded Go font TTF data loaded through `gopdf.AddTTFFontByReader`, and `github.com/phpdave11/gofpdi v1.0.16` as a refreshed indirect dependency if Go module resolution permits it with `gopdf`.

**Storage**: User-requested cleartext report files in the resolved local Documents directory. Markdown output writes one main `.md` file and one Annex 1 `.md` file. PDF output writes one `.pdf` file. No synced-data persistence, protected snapshot persistence, report history cache, rate cache, remote persistence, telemetry, or background report storage is added.

**Testing**: Go `testing` with contract, integration, unit, and existing empirical suites. Contract tests cover TUI/output/rendering contracts. Integration tests cover runtime generation, output cleanup, and 10,000 cached-activity scale. Targeted unit tests are justified for PDF pagination/text emission, filename construction, label mapping, zero-row filtering, historical-position rendering, and audit-annex model validation. Final verification uses `make test`, `make coverage`, and `make quality QUALITY_BASE_REF=origin/main` unless successful CI checks are cited.

**Empirical Dataset**: Existing `testdata/empirical/` synthetic empirical dataset and oracle fixtures remain read-only. Existing empirical tests continue to guard financial calculation behavior. This feature adds presentation and audit-traceability coverage around calculated results and does not mutate empirical source data or generated oracle fixtures.

**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access. PDF generation runs in-process and local-only.

**Project Type**: Single-module Go terminal UI application.

**Performance Goals**: Preserve the existing report-generation scale target of 10,000 cached activities. Keep report generation asynchronous from the Bubble Tea event loop. Avoid a lower activity-count limit for PDF or Annex 1 generation than Markdown output. Avoid rasterizing report pages into images.

**Constraints**: No floating-point financial logic. Exact decimals and explicit currency identity remain unchanged from the current report calculation pipeline. PDF generation must be local-only, A4-sized, text-based, searchable, and selectable by PDF readers that support text selection. Generated outputs and failures must not expose Ghostfolio tokens, bearer tokens, reusable authentication material, protected payload bytes, or unrelated secrets. Failed render or write attempts must not be reported as successful and must remove partial output files created by the attempt.

**Scale/Scope**: One report output format selection (`Markdown` or `PDF`), one audit annex model, one Markdown main-plus-annex renderer flow, one PDF renderer flow, output bundle writing for one or two files, report-result display of all generated files, and no new external document-generation service.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Generated Markdown and PDF reports are intentional user-requested cleartext local output files. No new protected persistence, synced-data persistence, report cache, telemetry, remote storage, or remote document-generation service is introduced. Ghostfolio tokens, bearer tokens, reusable verifiers, and protected payload bytes remain excluded from reports, annexes, result screens, errors, diagnostics, examples, and fixtures. OWASP Top 10 review scope covers broken access control in report workflows, cryptographic failures from token leakage, injection into rendered Markdown/PDF text, insecure design from partial-output success, security misconfiguration from remote PDF services, vulnerable components from the PDF dependency, identification/authentication leakage, software/data integrity of generated audit evidence, logging/diagnostic leakage, and SSRF avoidance by not adding remote document generation.
- [x] Precision: This feature changes output format, report presentation, and audit disclosure only. Financial amounts, quantities, rates, basis, proceeds, gains, and losses remain exact decimals using existing report models and `apd/v3`. Existing selected-currency and report-base-currency rules remain authoritative. No new conversion source, conversion boundary, rounding rule, or calculation method is authorized.
- [x] Testing: Integration-first coverage will verify TUI format selection, runtime output generation for Markdown and PDF, all generated file paths in results, partial-output cleanup, Annex 1 placement/content, and the 10,000 cached-activity scale. Targeted unit tests are justified for renderer and filename behavior that is isolated from external services. Coverage gates remain mandatory.
- [x] Quality gate: Source changes are expected in `*.go`, `go.mod`, and `go.sum`. Final changed-source quality verification is `make quality QUALITY_BASE_REF=origin/main`, or the successful `Quality` GitHub Actions check for this branch/PR. `make test` and `make coverage` remain the relevant full project validation commands.
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
│   ├── pdf/                  # Render local A4 searchable text PDF from calculated report and annex
│   └── output/               # Reserve/write/cleanup one-file PDF or two-file Markdown output bundles
└── tui/
    ├── flow/                 # Output-format selection state and report request construction
    └── screen/               # Selection, busy, and result copy including output format and all saved paths

tests/
├── contract/                 # TUI, report rendering, output filename, and saved-path contracts
├── integration/              # Runtime generation, failure cleanup, scale, and no-secret outputs
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
6. PDF rendering returns one A4 text PDF document with a page break before Annex 1.
7. Output writing reserves and writes the complete bundle, cleaning up every file created by the attempt if any write, sync, close, render, or validation step fails before success.
8. Runtime result screens report every generated file path for successful generation.

## PDF Dependency Decision

Planned direct PDF generation dependency: `github.com/signintech/gopdf v0.36.1`.

Recorded evidence:

- Module query returned `v0.36.1`, published from 2026-05-19.
- GitHub repository `signintech/gopdf` is not archived, has about 2,908 stars, and had commits on 2026-05-18 and 2026-05-19.
- Context7 documentation shows A4 page setup through `gopdf.Config{PageSize: *gopdf.PageSizeA4}`, text APIs such as `Text` and `Cell`, table layout APIs, `WritePdf`, and font registration through `AddTTFFontByReader`.
- The library generates PDFs in-process and does not require a browser, external binary, remote service, or CGO boundary for the planned use.

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
- Final implementation must pin module versions in `go.mod`/`go.sum` and pass the repository changed-source quality gate, including `govulncheck` as run by `make quality`.

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

- Contract tests verify report output format choices, TUI selection copy, output filename patterns, generated file path reporting, main report rendering changes, Annex 1 rendering order, and PDF A4/text-output contract.
- Integration tests generate Markdown and PDF from the same deterministic protected cache and assert shared report data matches between formats except for pagination, page titles, and Markdown annex splitting.
- Integration tests verify Markdown success creates exactly two files and PDF success creates exactly one file.
- Integration tests cover render/write failures and assert partial files created by the attempt are removed.
- Integration/performance tests cover 10,000 cached activities for Markdown and PDF generation with deterministic currency-rate fixtures from the existing conversion feature.
- Unit tests cover output filename construction, summary zero-row filtering and empty state, historical-position rendering, conversion-status and quote-direction labels, zero-priced `SELL` display as `BLOCKCHAIN OP`, PDF page-break/layout decisions, and audit-annex model validation.
- Existing empirical financial tests remain unchanged and continue to verify calculation behavior.

## Complexity Tracking

No constitution violations are planned.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
