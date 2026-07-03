# Research: Capital Gains Report PDF And Audit Annex

## Research Inputs

- Feature spec: `/specs/008-report-pdf-annex/spec.md`
- Prior base-currency conversion plan: `/specs/007-currency-conversion-strategy/plan.md`
- Prior conversion contracts: `/specs/007-currency-conversion-strategy/contracts/markdown-report.md` and `/specs/007-currency-conversion-strategy/contracts/tui-workflows.md`
- Current report code: `internal/report/model/`, `internal/report/calculate/`, `internal/report/markdown/`, `internal/report/output/`, `internal/app/runtime/`, and `internal/tui/`
- Dependency evidence from GitHub MCP and `go list -m -json` for `github.com/signintech/gopdf`, `github.com/phpdave11/gofpdi`, `golang.org/x/image`, `github.com/pdfcpu/pdfcpu`, and `github.com/johnfercher/maroto/v2`
- Context7 documentation for `/signintech/gopdf`

## Output Format Selection

Decision: Add a report output format to the report request and TUI workflow with two valid values: Markdown and PDF.

Rationale: The user must choose the output format before generation begins, and Markdown must remain available. Modeling the choice in the report request keeps runtime generation deterministic and avoids hidden renderer defaults. The format is transient to one report run and is not persisted.

Alternatives considered: Use a command-line flag only, but current report generation is a TUI workflow. Generate both formats every time, but the spec requires a user choice and would create extra files the user did not request.

## Local PDF Generation Strategy

Decision: Use an in-process Go PDF renderer under `internal/report/pdf/` backed by `github.com/signintech/gopdf v0.36.1`.

Rationale: The standard library does not provide a high-level PDF writer. Manual PDF generation would be possible but would add significant local format code for page dictionaries, content streams, embedded fonts, table layout, wrapping, and object references. `gopdf` directly supports A4 pages, text operations, table layout, embedded TTF fonts, and writing PDF files locally. GitHub evidence shows the repository is not archived, had a 2026-05-19 module release, had commits in May 2026, and has broad adoption signals. The planned usage does not require importing user PDFs, remote services, CGO, browser automation, or external binaries.

Alternatives considered: Manual PDF writing with standard library only was rejected for higher format risk and larger implementation scope. `pdfcpu/pdfcpu` was rejected as the primary renderer because it is better suited to PDF processing/tooling than direct report generation from domain models. `johnfercher/maroto` was rejected because it adds a higher-level dependency above a PDF generator and is not necessary for the report layout. `jung-kurt/gofpdf` was rejected because it is archived. Browser or service-based HTML-to-PDF was rejected because PDF generation must be local-only and must not send report data to a remote document service.

## PDF Dependency Research

Decision: Pin and verify `github.com/signintech/gopdf v0.36.1`, `golang.org/x/image v0.43.0`, and a refreshed `github.com/phpdave11/gofpdi v1.0.16` indirect dependency if compatible with module resolution.

Rationale: `go list -m -json github.com/signintech/gopdf@latest` returned `v0.36.1`, time `2026-05-19T16:39:12Z`, and Go version `1.13`. GitHub search showed `signintech/gopdf` as a non-archived Go repository with about 2,908 stars and recent updates. Context7 documentation shows A4 page creation, text APIs, table APIs, `WritePdf`, and `AddTTFFontByReader`. `gopdf v0.36.1` depends on `github.com/phpdave11/gofpdi`; `go list -m -json github.com/phpdave11/gofpdi@latest` returned `v1.0.16`, time `2026-04-15T23:48:25Z`, and GitHub release `v1.0.16` documents PDF stream/import fixes. `go list -m -json golang.org/x/image@latest` returned `v0.43.0`, time `2026-06-15T23:26:03Z`; its use is limited to embedded Go font TTF data.

Alternatives considered: Add no dependency and hand-write PDFs, but that would move low-level PDF correctness into application code. Add `pdfcpu` for validation or text extraction, but that would be a second PDF dependency and is not required for the initial renderer plan. Vendor a TTF file directly into the repository, but using Go project's font bytes through `x/image` avoids OS font path dependence without adding a binary asset to the repo.

## PDF Text Accessibility

Decision: Generate PDF pages with text drawing operations and embedded TTF fonts, never as rasterized full-page images.

Rationale: The spec requires text-based searchable PDF output with selectable report text. `gopdf` text APIs emit text into PDF content streams and can embed fonts from `io.Reader`. Rendering from report-domain rows into text/table operations preserves searchability better than rendering screenshots or canvases. Tests can verify that required report strings are emitted through the PDF renderer's text path and that the renderer does not create page image replacements.

Alternatives considered: Render Markdown to an image and place it in a PDF, but that would violate the selectable text requirement. Use an external `pdftotext`-style binary in tests, but that would add a platform dependency outside Go test control. Add a PDF text-extraction dependency immediately, but the first implementation can validate renderer behavior without another PDF dependency unless test evidence later proves insufficient.

## Embedded Font Strategy

Decision: Load deterministic embedded Go font bytes through `gopdf.AddTTFFontByReader` rather than relying on local system font files.

Rationale: The application targets Linux, macOS, and Windows terminals. Relying on `/usr/share/fonts`, Windows font paths, or user-installed fonts would make PDF generation environment-dependent and could fail on otherwise valid installations. The Go font package provides local font bytes that can be embedded with the binary and loaded from memory.

Alternatives considered: Require users to install a font, but that adds setup burden and support risk. Check several OS-specific font paths, but that creates non-deterministic behavior and complicates tests. Vendor a TTF asset in the repository, but that adds binary asset governance and license tracking work.

## Audit Annex Model

Decision: Add explicit Annex 1 report-domain models for per-asset audit sections, audit activity entries, and the currency conversion audit section.

Rationale: The existing main report detail sections only store selected-year activity rows. Annex 1 needs all activity evidence on or before report-year end, including historical pre-year activity and excluding post-year activity. Recording explicit annex models after report replay lets Markdown and PDF share the same evidence and prevents renderer-specific reconstruction of financial state. The model can reuse existing exact-decimal values and liquidation results without changing cost-basis methods.

Alternatives considered: Reconstruct the annex entirely inside renderers from protected cache data, but renderers should not own financial replay rules or protected-cache interpretation. Duplicate the main asset detail sections, but those sections intentionally omit pre-year activity rows and cannot satisfy the annex traceability requirement.

## Calculation Boundary For Annex Evidence

Decision: Extend report calculation replay to record audit evidence after every activity applied through the selected year end, while preserving existing cost-basis calculations and yearly totals.

Rationale: The replay already applies historical activity through the selected-year cutoff to calculate opening, closing, basis, liquidation, and yearly net values. Capturing post-activity held quantity, basis after row, full liquidation status, and gain/loss contribution at that point gives Annex 1 auditable evidence without recalculating in a separate renderer. Activities after the report year are already outside the selected cutoff and must remain excluded.

Alternatives considered: Run a second replay only for Annex 1, but that duplicates financial state transitions and increases drift risk. Store annex evidence in protected snapshots, but this is generated report output and does not require new persistence.

## Main Report Presentation Changes

Decision: Implement requested main report changes in renderer/model contracts without changing financial calculation inputs or totals.

Rationale: Bold classifier labels, zero net-gain row omission, the `Historical Full Liquidation Count` header, `Historical Position` for assets without report-year activity, user-friendly conversion status labels, and `BLOCKCHAIN OP` for zero-priced SELL rows are presentation rules. Keeping them in rendering/model validation avoids altering basis state or normalized activity storage.

Alternatives considered: Change activity type normalization for zero-priced SELL rows, but that would rewrite domain activity semantics. Remove zero rows during calculation, but the renderer can suppress them while preserving calculated evidence for tests and alternate renderers.

## Markdown Annex Output

Decision: Keep Markdown main report output as a Markdown file and write Annex 1 as a separate Markdown file using the annex filename pattern.

Rationale: The spec requires separate Markdown Annex 1 output whose name preserves the main report filename pattern and inserts `-annex-1-` before the date segment. Separate documents avoid bloating the main Markdown report while keeping the annex deterministic and discoverable.

Alternatives considered: Append Annex 1 to the main Markdown file, but the spec requires a separate Markdown annex file. Generate only the annex when detailed evidence exists, but the spec requires Annex 1 for every successful report.

## PDF Annex Output

Decision: Render the main report and Annex 1 into one PDF document, with Annex 1 starting after a page break.

Rationale: The spec requires a single PDF file containing both the main report and Annex 1. A page break creates a clear audit boundary while keeping one output artifact for PDF workflows.

Alternatives considered: Generate separate main and annex PDFs, but the spec requires one PDF file. Embed the Markdown annex as an attachment, but that would not satisfy visible selectable Annex 1 text in the PDF body.

## Output Bundle Writing

Decision: Replace single-document report writing with output-bundle writing that can atomically treat Markdown main-plus-annex as one success outcome.

Rationale: Markdown success now means exactly two files, while PDF success means exactly one file. Runtime result screens must report all generated files. If one Markdown file succeeds and the other fails, presenting the first file as a successful report would be misleading. Bundle writing centralizes reservation, suffix selection, file modes, sync/close, cleanup, and post-save opening behavior.

Alternatives considered: Call the current single-file writer twice from runtime, but cleanup and result composition would be error-prone and spread output policy across runtime. Write a temporary zip bundle, but the spec asks for direct Markdown and PDF report files.

## User-Friendly Label Mapping

Decision: Define explicit report-facing labels for conversion status and quote direction and fail rendering if a mapping is missing.

Rationale: The spec forbids exposing code-style or snake_case values. A closed mapping makes regressions testable and avoids accidental leakage of enum names such as `same_currency` or `source_per_base`.

Alternatives considered: Replace underscores at render time, but that hides unmapped enum values instead of failing safely. Change enum constants to display strings, but those constants are used as domain identifiers and should remain stable.

## Testing Evidence Strategy

Decision: Use deterministic project-owned fixtures for automated tests and keep existing empirical financial data read-only.

Rationale: Report output format, annex, and PDF rendering can be tested without live services. Existing conversion-provider deterministic fixtures from the prior feature remain applicable for converted activity evidence. The financial empirical dataset continues to guard calculation behavior and should not be mutated for a presentation/output feature.

Alternatives considered: Use live PDF viewers or OS text extraction tools in automated tests, but those are platform-dependent. Change empirical fixtures to include annex-specific assertions, but the active feature is not a dataset-maintenance spec.

## Security Review

Decision: Treat this as local cleartext output generation with dependency risk review and no new remote integration.

Rationale: Reports intentionally contain financial values, but only as user-requested local files. PDF rendering must not upload data, fetch remote templates, call a remote service, or include token material. Dependency risk is managed by version pinning, recent-maintenance research, and the repository changed-source quality gate with `govulncheck`.

Alternatives considered: Remote PDF services are rejected by the spec and constitution. Persisting generated report metadata or report history is unnecessary and would add storage/security scope.
