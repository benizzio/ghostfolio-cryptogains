# Quickstart: Capital Gains Report PDF And Audit Annex

This document defines validation flows for the PDF output and Annex 1 feature. Automated validation should use deterministic project-owned fixtures and existing mocked currency-rate provider fixtures where conversion evidence is needed.

## Prerequisites

- Go 1.26.3 installed.
- Development tools installed as required by the repository quality gates.
- A synced or fixture protected activity cache with reportable years, at least one asset with no report-year activity, at least one zero net-gain summary row, at least one zero-priced SELL activity, and at least one converted priced activity.
- No live PDF service, browser service, or external document-generation service is required.

## Automated Verification Flow

1. Run the full test suite.

```bash
make test
```

Expected result:

- report output-format request validation passes for Markdown and PDF and rejects unsupported formats
- TUI contract tests show output-format selection and selected format on busy/result screens
- Markdown renderer tests produce a main document and Annex 1 document
- PDF renderer tests produce A4 text PDF bytes through the local renderer
- rendering tests cover bold classifier labels, zero summary row omission, summary empty state, `Historical Full Liquidation Count`, `Historical Position`, exact conversion status labels `Same currency` and `Converted`, exact quote direction labels `Source currency per base currency` and `Base currency per source currency`, and `BLOCKCHAIN OP`
- runtime tests verify Markdown creates exactly two files and PDF creates exactly one file
- failure tests verify partial output cleanup
- no generated report or failure text contains token material

1. Run the maintained coverage gate.

```bash
make coverage
```

Expected result:

- project-owned code remains at the repository coverage threshold
- coverage includes contract, integration, unit, and empirical suites as configured by repository tooling
- generated coverage artifacts are written under `dist/coverage`

1. Run the changed-source quality gate from the expected base branch.

```bash
make quality QUALITY_BASE_REF=origin/main
```

Expected result:

- changed Go source and dependency files pass the repository changed-source quality gate
- dependency and vulnerability checks include any new `go.mod` or `go.sum` entries introduced for PDF rendering

## Required Automated Scenarios

Contract and integration coverage should prove these scenarios:

- User can select `Markdown` or `PDF` before generation.
- Markdown output remains available.
- Markdown success creates exactly one main report file and exactly one Annex 1 file.
- PDF success creates exactly one PDF file containing the main report and Annex 1.
- PDF output uses `.pdf` while preserving the main report filename pattern.
- Markdown Annex 1 filename inserts `-annex-1-` immediately before the date segment.
- Successful result screens list all generated paths.
- PDF generation works without user-installed fonts, platform-specific font paths, browser rendering, operating-system print-to-PDF support, or remote font resources.
- Main report omits detailed Currency Conversion Audit rows.
- Annex 1 title is `Annex 1 - Audit`.
- Annex 1 renders per-asset audit evidence before Currency Conversion Audit.
- Annex 1 includes activity on or before report-year end for every reported asset, including reference-only reported assets, and excludes post-year activity.
- Annex 1 includes an explicit Currency Conversion Audit empty state when no converted activity exists.
- PDF Annex 1 starts after a page break.
- Required PDF report text is generated as text, not as raster page images.
- Missing conversion-status or quote-direction label mappings fail before output success.
- PDF and Markdown shared main report sections contain the same required data values for identical inputs.
- 10,000 cached-activity report generation succeeds for both output formats using deterministic fixtures.

## Manual TUI Verification Flow

1. Launch the application.

```bash
go run ./cmd/ghostfolio-cryptogains
```

Expected result:

- application starts in the terminal UI without requiring a PDF service

1. Enter the `Sync and Reports` context and unlock or sync a dataset with reportable activity.

Expected result:

- token entry remains masked
- report generation is unavailable until reportable years exist
- unlocked context shows reportable years without exposing protected raw payload data

1. Start report generation and inspect the selection screen.

Expected result:

- year, cost-basis method, report base currency, and output format are visible
- output formats are exactly `Markdown` and `PDF`
- generation cannot start without a selected output format

1. Generate a Markdown report.

Expected result:

- result screen shows a saved main Markdown path and a saved Annex 1 Markdown path
- main Markdown report contains no detailed Currency Conversion Audit rows
- Annex 1 Markdown report starts with `Annex 1 - Audit`
- Annex 1 per-asset audit appears before Currency Conversion Audit

1. Generate the same report inputs as PDF.

Expected result:

- result screen shows one saved `.pdf` path
- PDF filename preserves the main report filename pattern and uses `.pdf`
- PDF opens locally if automatic open succeeds
- PDF generation does not require installing fonts or using OS-specific font paths
- PDF text can be selected and searched in a PDF reader that supports text selection
- Annex 1 appears in the PDF after a page break

1. Run a fixture or development setup that forces PDF render or output write failure.

Expected result:

- generation fails before final success
- no partial output from the failed attempt remains in the Documents directory
- failure message is actionable and contains no token material

## Security Review Checklist

- PDF rendering is local-only and does not call remote services.
- Report output files are cleartext because they are user-requested generated reports.
- Generated files, result messages, diagnostics, examples, and fixtures exclude Ghostfolio tokens, bearer tokens, reusable authentication material, and raw protected payload bytes.
- Dependency versions for PDF generation are pinned and reviewed in `go.mod` and `go.sum`.
- `make quality QUALITY_BASE_REF=origin/main` passes or the successful `Quality` GitHub Actions check is cited.
- Failed render/write attempts clean up partial output files before reporting failure.

## Empirical Dataset Policy

Existing empirical financial tests remain applicable as calculation regression evidence. The active feature must not mutate `testdata/empirical/` datasets or generated oracle fixtures.
