# Contract: Report Rendering And Annex 1

## Scope

This contract defines user-visible report content for the main capital gains and losses report and Annex 1 after adding PDF output and audit annex support.

## Shared Main Report Content

The main report must include the existing report sections except where this contract changes presentation.

Required presentation changes:

- Initial report detail labels are bold.
- `Gains-And-Losses Summary` omits rows whose `Net Gain Or Loss` is exactly zero.
- If all summary rows are omitted, the summary section renders a clear empty-state message instead of an empty table.
- `Rate Source Summary` classifier labels are bold.
- The reference section header uses `Historical Full Liquidation Count` instead of `Full Liquidation Count Through Year End`.
- `Currency Conversion Audit` detailed rows are not rendered in the main report.

Rules:

- Shared data values between Markdown and PDF main reports must match for the same report inputs.
- Format-specific page breaks, page titles, and Markdown annex separation may differ.
- Report text and table values must continue to use exact-decimal canonical formatting.
- Report content must not include token material or raw protected payload data.

## Asset Detail Rendering

For assets with at least one report-year activity:

- Render the normal asset detail structure.
- Keep the report-year `In-Year Activity` section.
- Keep `Opening Position` and `Closing Position` sections.
- Keep liquidation calculations when present.

For assets with no report-year activity:

- Render a single `Historical Position` section.
- `Historical Position` shows the same quantity, cost basis, and calculation currency facts that would otherwise be shown in `Closing Position`.
- Omit separate `Opening Position`, `In-Year Activity`, and `Closing Position` subsections for that asset.

Activity row label rules:

- Conversion status values must be user-friendly labels and must not expose `same_currency`, `converted`, or other snake_case/internal code values.
- Zero-priced `SELL` rows render Type as `BLOCKCHAIN OP`.
- Missing conversion-status label mappings fail rendering before final output success.

## Annex 1 Structure

Every successful report includes Annex 1.

Required title:

```markdown
Annex 1 - Audit
```

Required section order:

1. Detailed per-asset audit report.
2. Currency Conversion Audit.

Rules:

- Annex 1 is a separate Markdown file for Markdown output.
- Annex 1 appears in the same PDF after a page break for PDF output.
- Annex sections render explicit empty-state text when they contain no rows.
- Annex content must not include activity after the selected report-year end.

## Per-Asset Audit Section

The detailed per-asset audit report must group activity evidence by asset.

Required entry fields or equivalent visible fields:

| Field | Requirement |
|-------|-------------|
| Activity date/time | Render the activity occurrence timestamp consistently with report contracts |
| Source ID | Render non-secret source activity reference |
| Activity type | Render user-friendly type, using `BLOCKCHAIN OP` for zero-priced SELL rows |
| Quantity | Render activity quantity |
| Unit price | Render when applicable |
| Gross value | Render when applicable |
| Fee | Render when applicable |
| Original activity currency | Render when applicable |
| Calculation currency | Render when monetary values are calculated |
| Quantity after activity | Render held quantity after the activity is applied |
| Basis after activity | Render open cost basis after the activity is applied |
| Full liquidation event | Render whether the activity fully liquidated the asset |
| Allocated basis | Render when basis was allocated by a disposal |
| Net liquidation proceeds | Render for priced liquidations |
| Gain or loss | Render for activities that realize gain or loss |
| Conversion status | Render user-friendly label when applicable |
| Note | Render sanitized note when present |

Rules:

- Include every available activity for the asset on or before selected report-year end.
- Exclude every activity after selected report-year end.
- Preserve deterministic report replay ordering.
- Financial values in Annex 1 must trace to the calculated report and existing basis replay rules.

## Currency Conversion Audit In Annex 1

Currency Conversion Audit is the second Annex 1 section.

Required grouped table columns or equivalent visible fields:

```markdown
| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |
|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|
```

Rules:

- Converted priced activities must have audit details.
- Same-currency rows may be excluded from Currency Conversion Audit if main or annex activity rows identify them clearly as same-currency.
- If no activity required conversion, render a clear empty-state message such as `No converted activity was present for this report.`
- Quote direction must be a user-friendly label and must not expose `source_per_base`, `base_per_source`, or other code-style values.
- Missing quote-direction label mappings fail rendering before final output success.
- Provider-level authority and rate-kind metadata remain in `Rate Source Summary`; per-row audit details focus on activity, amount, quote direction, rate date, and rate value.
- Provider-published rate precision is preserved.

## PDF Rendering Contract

PDF output must satisfy these additional rules:

- Page size is A4.
- Required report text is emitted as selectable text, not as page images.
- Annex 1 starts after a page break.
- Long tables may continue across pages with repeated or clear table context.
- PDF generation runs locally without remote services or external document-generation APIs.

## Markdown Rendering Contract

Markdown output must satisfy these additional rules:

- Main report and Annex 1 are separate files.
- Main report heading remains `# Ghostfolio Capital Gains And Losses Report`.
- Annex report heading is `# Annex 1 - Audit`.
- Markdown table content may wrap only according to Markdown renderer rules; data values must match PDF values for the same report inputs.
