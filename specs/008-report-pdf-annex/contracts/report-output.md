# Contract: Report Output Files

## Scope

This contract defines generated output files for capital gains and losses reports after adding PDF output and Annex 1.

## Output Formats

Supported output formats:

| Code | Label | Output files |
|------|-------|--------------|
| `markdown` | `Markdown` | Main Markdown file plus separate Annex 1 Markdown file |
| `pdf` | `PDF` | One PDF file containing the main report and Annex 1 |

Rules:

- The user selects exactly one output format before generation begins.
- Unsupported output formats fail request validation before calculation.
- The selected output format is transient and is not persisted to setup or synced-data storage.

## Markdown Output Files

Successful Markdown generation creates exactly two files in the resolved Documents directory:

| Role | Extension | Filename pattern |
|------|-----------|------------------|
| Main report | `.md` | `ghostfolio-capital-gains-<year>-<method-slug>-<YYYY-MM-DD_HH-MM-SS>.md` |
| Annex 1 | `.md` | `ghostfolio-capital-gains-<year>-<method-slug>-annex-1-<YYYY-MM-DD_HH-MM-SS>.md` |

Collision suffix rules:

- If the unsuffixed pair is unavailable, the writer reserves a matching numeric suffix for both files.
- The suffix is appended after the timestamp and before the extension.
- Example main collision filename: `ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2.md`.
- Example annex collision filename: `ghostfolio-capital-gains-2024-fifo-annex-1-2026-05-21_12-34-56-2.md`.

Rules:

- The main Markdown file must not contain the detailed Currency Conversion Audit section.
- The Annex 1 Markdown file must start with `# Annex 1 - Audit`.
- A Markdown success outcome is valid only when both files are written, synced, closed, and recorded.
- If either file fails before success, every file created by that attempt is removed and no successful Markdown path is reported.

## PDF Output File

Successful PDF generation creates exactly one file in the resolved Documents directory:

| Role | Extension | Filename pattern |
|------|-----------|------------------|
| Combined report | `.pdf` | `ghostfolio-capital-gains-<year>-<method-slug>-<YYYY-MM-DD_HH-MM-SS>.pdf` |

Collision suffix rules:

- If the unsuffixed filename is unavailable, the writer appends `-2`, `-3`, and later suffixes after the timestamp and before `.pdf`.
- Example collision filename: `ghostfolio-capital-gains-2024-fifo-2026-05-21_12-34-56-2.pdf`.

Rules:

- The PDF must be A4-sized.
- The PDF must contain the main report first.
- Annex 1 must appear after a page break in the same PDF file.
- The PDF must be text-based and must not rasterize required report text into page images.
- The PDF must be formatted from report-domain data through the PDF renderer, not by embedding Markdown-rendered source text as the report body.
- A PDF that exposes Markdown structural syntax such as heading markers, table pipes or separators, or bold markers as visible report presentation is not a valid successful PDF output.
- A PDF that presents report-domain data as a plain sequential line dump without visible heading hierarchy, styled classifier labels, table headers, table rows, table columns, wrapped cell content, and continuation context is not a valid successful PDF output.
- The PDF renderer must use `github.com/signintech/gopdf` layout primitives for A4 pages, application-supplied fonts, headings, styled text, table rows, table columns, wrapping, and continuation context.
- A PDF success outcome is valid only when the file is written, synced, closed, and recorded.

## Result Screen Path Reporting

Successful outcomes must show all generated output files:

| Output format | Required result paths |
|---------------|-----------------------|
| Markdown | Saved Markdown main path and saved Annex 1 Markdown path |
| PDF | Saved PDF path |

Rules:

- Result messages must identify the selected output format.
- Automatic-open failure after successful save is a warning, not a save failure.
- Automatic-open warnings must still list the saved output paths.

## Failure And Cleanup

Rules:

- Render failures save no report files.
- Write, sync, close, validation, or post-render pre-success failures remove every file created by the attempt.
- Failures before final output completion must not present partial report files as successful.
- Failure messages must not include Ghostfolio tokens, bearer tokens, reusable authentication material, raw protected payload bytes, or other secrets.

## Locality

Rules:

- PDF rendering runs locally in the application process.
- No report data, financial data, tokens, generated Markdown, generated PDF, or annex content may be sent to a remote PDF service, cloud renderer, telemetry service, or remote storage as part of this feature.
- PDF generation must not depend on remote fonts, platform-specific font paths, user-installed fonts, browser rendering, or operating-system print-to-PDF support.
