# Contract: Markdown Capital Gains Report

## Scope

This contract defines the generated Markdown report document and final output-file behavior for the `Generate Yearly Gains And Losses Report` slice.

## Output File Contract

### Location

The application saves successful reports in the current OS user's personal Documents folder.

Resolution rule:

```text
linux:
  1. if xdg-user-dirs configuration defines a Documents directory in user-dirs.dirs, use it
  2. otherwise use $HOME/Documents

macOS:
  use the current user's Documents directory under the user's home directory

Windows:
  target the per-user Documents known folder (FOLDERID_Documents)
  whose documented default path is %USERPROFILE%\Documents
```

Best-practice notes:

- Linux user directories can be user-configured; a literal `$HOME/Documents` fallback is only the secondary rule.
- Windows should be treated as a known-folder lookup problem, not just a string concatenation problem.
- macOS Documents is conventionally the per-user `Documents` directory under the home directory.
- If the implementation cannot resolve the OS-appropriate Documents directory using the available standard-library-only rules, it fails instead of guessing another cleartext location.

Failure rules:

- If the user home directory cannot be resolved, report generation fails with `documents folder unavailable`.
- If the Documents directory does not exist, is not a directory, or is not writable, report generation fails with `documents folder unavailable` or `report file write failed`.
- The application must not silently fall back to the current working directory, app-data directory, or OS temp directory.

### Filename

Filename format:

```text
ghostfolio-capital-gains-<year>-<method>-<YYYY-MM-DD_HH-MM-SS>.md
```

Collision rule:

```text
ghostfolio-capital-gains-<year>-<method>-<YYYY-MM-DD_HH-MM-SS>-2.md
ghostfolio-capital-gains-<year>-<method>-<YYYY-MM-DD_HH-MM-SS>-3.md
```

Rules:

- Timestamp uses local time.
- Timestamp order must remain `YYYY-MM-DD_HH-MM-SS` so alphabetical sorting preserves creation-time order for reports with different seconds.
- Existing files must not be overwritten.
- Method text in filenames should be a stable lowercase slug such as `fifo`, `lifo`, `hifo`, `average-cost`, or `scope-local-hybrid`.

### Write And Cleanup

Rules:

- The application renders Markdown in memory before final save.
- The application must not write report content to app-managed storage or OS temp directories before final save.
- The final path should be opened with exclusive-create behavior where supported.
- If writing fails after file creation, the partial file created by that failed attempt must be removed.
- If writing succeeds, the file remains even if the OS open request fails.

### OS Default-App Open

After a successful save, the application requests that the operating system open the Markdown file in the default associated application.

Platform command adapter:

| Platform | Command |
|----------|---------|
| Linux | `xdg-open <path>` |
| macOS | `open <path>` |
| Windows | `rundll32 url.dll,FileProtocolHandler <path>` or an equivalent non-shell opener adapter |

Rules:

- The opener is invoked after the file is fully saved.
- Opener failure is non-fatal.
- The result message must show the saved path and state that automatic opening failed.
- Opener command lines must not include tokens, JWTs, or report content.

## Markdown Document Contract

### General Rules

- The document is plain Markdown.
- The document uses ASCII-compatible section headings and tables.
- The first section is `Gains-And-Losses Summary`.
- The second section is `Reference Section`.
- Per-asset detail sections follow the reference section.
- The report calculation currency label is exactly `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL`.
- Quantities and monetary values use canonical exact-decimal strings with no rounding and only non-significant formatting trimmed.
- Zero is rendered as `0`.
- Losses render with a leading negative sign.
- Activity after the selected year does not appear.
- Ghostfolio token, JWT, raw protected payload bytes, and unredacted diagnostic content must not appear.

### Required Header

The document starts with:

```markdown
# Ghostfolio Capital Gains And Losses Report

- Year: <year>
- Cost Basis Method: <method label>
- Generated At: <local timestamp>
- Report Calculation Currency: NO CURRENCY APPLIES, ALL CONSIDERED EQUAL
```

Rules:

- `Generated At` uses a local timestamp format that includes date and time.
- The method label matches the user's selected method.

### Gains-And-Losses Summary

Section heading:

```markdown
## Gains-And-Losses Summary
```

Required table columns:

```markdown
| Asset | Net Gain Or Loss | Report Calculation Currency |
|-------|------------------|-----------------------------|
```

Rows:

- One row for each asset included in main report sections.
- One final overall yearly net total row.

Rules:

- Asset rows use the display label for the grouped asset identity key.
- Assets with zero selected-year result are included when they meet main-section inclusion rules.
- Net values use selected-year liquidations only.
- The total row label is `Overall Yearly Net Total`.
- Every row uses `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL` in the report calculation currency column.
- If no asset qualifies for the main report sections, render a clear empty-state sentence before the total row and still render `Overall Yearly Net Total` with value `0`.

Example:

```markdown
| Asset | Net Gain Or Loss | Report Calculation Currency |
|-------|------------------|-----------------------------|
| BTC | 1250.5 | NO CURRENCY APPLIES, ALL CONSIDERED EQUAL |
| ETH | -10 | NO CURRENCY APPLIES, ALL CONSIDERED EQUAL |
| Overall Yearly Net Total | 1240.5 | NO CURRENCY APPLIES, ALL CONSIDERED EQUAL |
```

### Reference Section

Section heading:

```markdown
## Reference Section
```

Required table columns:

```markdown
| Asset | Full Liquidation Count Through Year End | Main Section Status |
|-------|-----------------------------------------|---------------------|
```

Rows:

- One row for each asset that reaches zero quantity at least once on or before the selected year end.

Rules:

- Assets fully liquidated before the selected year and not reopened on or before selected year end are marked `reference only`.
- Assets also included in main sections are marked `included in main sections`.
- For the scope-local hybrid method, the count shown in one asset row is the sum of applicable-scope transitions to zero for that asset through the selected-year cutoff.
- If no asset reached full liquidation by year end, render a clear empty-state sentence after the heading instead of a table.

### Per-Asset Detail Sections

Section heading format:

```markdown
## Asset Detail: <display label>
```

Required opening block:

```markdown
### Opening Position

- Quantity: <quantity>
- Cost Basis: <basis>
- Calculation Currency: NO CURRENCY APPLIES, ALL CONSIDERED EQUAL
```

Required activity table columns:

```markdown
### In-Year Activity

| Date | Source ID | Type | Quantity | Gross Value | Fee | Activity Currency | Basis After Row | Calculation Currency | Quantity After Row | Note |
|------|-----------|------|----------|-------------|-----|-------------------|-----------------|----------------------|--------------------|------|
```

Rules:

- Every in-year activity for the included asset appears in this table.
- The table includes acquisitions, priced liquidations, and explained zero-priced holding reductions.
- `Activity Currency` shows the explicit currency code from which that row's `Gross Value` and `Fee` were taken for priced activity rows.
- For explained zero-priced holding reductions, `Gross Value`, `Fee`, and `Activity Currency` are left blank because no activity monetary context is required from that row.
- `Calculation Currency` shows `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL` for calculated row values such as `Basis After Row` in this slice.
- `Note` explains zero-priced holding reductions as holding reductions with zero gain and zero loss.
- The table does not include later activity.
- If an included asset has no in-year activity, render a clear no-in-year-activity sentence instead of this table and omit the liquidation table.

Required liquidation table when the asset has priced in-year liquidations:

```markdown
### Liquidation Calculations

| Date | Source ID | Disposed Quantity | Activity Currency | Allocated Basis | Net Liquidation Proceeds | Gain Or Loss | Calculation Currency |
|------|-----------|-------------------|-------------------|-----------------|--------------------------|--------------|----------------------|
```

Rules:

- Only priced liquidations inside the selected year appear.
- Zero-priced holding reductions do not appear as gain/loss rows.
- `Gain Or Loss` is negative for losses and `0` for zero result.
- `Activity Currency` shows the explicit selected activity currency code used for the matching in-year liquidation row's `Gross Value` and `Fee`.
- `Net Liquidation Proceeds` is rendered in that `Activity Currency` because it is derived from one activity before cross-activity calculation begins.
- `Calculation Currency` shows `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL` because `Allocated Basis` and `Gain Or Loss` are cross-activity calculation outputs in this slice.

Required closing block:

```markdown
### Closing Position

- Quantity: <quantity>
- Cost Basis: <basis>
- Calculation Currency: NO CURRENCY APPLIES, ALL CONSIDERED EQUAL
```

If no asset qualifies for the main report sections, no `Asset Detail: <display label>` sections are rendered.

## Calculation Boundary Contract

Before Markdown rendering, the calculator supplies a complete `CapitalGainsReport` model with:

- selected year
- selected cost basis method
- generated timestamp
- summary entries
- yearly net total
- reference entries
- detail sections
- report calculation currency label

Markdown rendering must not:

- choose currency contexts
- match lots
- calculate basis
- decide asset inclusion
- mutate the protected activity cache
- persist report history

## Security Contract

- The rendered Markdown exists only in memory until final save.
- The final saved Markdown file is intentionally cleartext in Documents.
- After save and OS-open request, the application does not store another cleartext copy.
- Generated report content is not written to diagnostics, setup, protected snapshots, logs, or crash text by project-owned code.
- If save fails, no partial cleartext report artifact remains from that failed attempt.
