# Data Model: Capital Gains Report PDF And Audit Annex

## Modeling Notes

This feature extends report-generation runtime models and rendered report documents. It does not add protected snapshot fields, synced-data storage, remote storage, or financial calculation methods. Generated report files remain intentional cleartext user output.

All financial amounts, quantities, exchange rates, converted amounts, basis values, proceeds, gains, and losses remain exact decimals. Currency identity remains explicit before and after any existing base-currency conversion boundary.

## ReportOutputFormat

Purpose: The user-selected document format for one report-generation run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `code` | enum | `markdown` or `pdf` |
| `label` | string | User-facing label, `Markdown` or `PDF` |

Relationships:

- Selected by one `ReportRequest`.
- Determines which renderer runtime invokes.
- Determines the generated output-file bundle shape.

Validation rules:

- Exactly one output format is required before generation starts.
- Only `markdown` and `pdf` are valid.
- The selection is transient and is not persisted.

State transitions:

- `unselected -> selected` when the user chooses an output format.
- `selected -> submitted` when generation starts.
- `submitted -> rendered_markdown` or `submitted -> rendered_pdf` after successful rendering.

## ReportRequest

Purpose: User-selected inputs for one report-generation run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `year` | integer | Must be present in `ProtectedActivityCache.available_report_years` |
| `cost_basis_method` | enum | Existing supported method set |
| `report_base_currency` | enum | Existing `USD` or `EUR` base currency selection |
| `output_format` | `ReportOutputFormat` | Required `markdown` or `pdf` |
| `requested_at` | timestamp | Generation request time used for output naming |

Relationships:

- Consumes one unlocked `ProtectedActivityCache`.
- Produces one `CapitalGainsReport` plus one `AuditAnnex` on successful calculation.
- Produces one `ReportOutputBundle` on successful rendering and save.

Validation rules:

- Year, method, base currency, output format, and requested timestamp are required.
- Year must be selected from synced data, not free text.
- Method must be one of the supported cost-basis methods.
- Base currency must be `USD` or `EUR`.
- Output format must be `markdown` or `pdf`.

State transitions:

- `draft -> validated -> calculated -> rendered -> saved` on success.
- `draft -> failed` when request validation fails.
- `calculated -> render_failed` when Markdown, PDF, label, or annex rendering fails.
- `rendered -> save_failed` when bundle writing fails.

## MainCapitalGainsReport

Purpose: The primary calculated report content rendered before Annex 1.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `year` | integer | Selected report year |
| `cost_basis_method` | enum | Selected method |
| `generated_at` | timestamp | Report generation timestamp |
| `report_calculation_currency` | string | Selected report base currency label |
| `summary_entries` | list of `AssetSummaryEntry` | May contain zero net rows before rendering filters |
| `yearly_net_total` | decimal | Existing exact-decimal yearly net total |
| `reference_entries` | list of `ReferenceLiquidationEntry` | Existing reference evidence |
| `detail_sections` | list of `AssetDetailSection` | Existing main asset details |
| `rate_sources` | list of `ExchangeRateEvidence` | Provider-level source summary evidence |

Relationships:

- Produced by report calculation from a `ReportRequest` and protected cache.
- Shares source activity identities and conversion evidence with `AuditAnnex`.
- Rendered into Markdown main output and the first part of PDF output.

Validation rules:

- Existing report validation rules remain applicable.
- Main report must not include detailed Currency Conversion Audit rows after this feature.
- Renderer must omit zero net-gain summary rows and render an empty state if all rows are omitted.
- Renderer must use user-friendly labels for conversion status.
- Renderer must show zero-priced `SELL` activity rows as `BLOCKCHAIN OP` in the Type column.
- Renderer must render `Historical Position` instead of opening/activity/closing subsections for assets without report-year activity.

## AuditAnnex

Purpose: Annex 1 of every successful report, titled `Annex 1 - Audit`.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `title` | string | Must be `Annex 1 - Audit` |
| `per_asset_sections` | list of `PerAssetAuditSection` | First annex section |
| `currency_conversion_section` | `CurrencyConversionAuditSection` | Second annex section |

Relationships:

- Produced with one `MainCapitalGainsReport`.
- Rendered as a separate Markdown file for Markdown output.
- Rendered in the same PDF after a page break for PDF output.

Validation rules:

- Annex is required for every successful report generation.
- Per-asset audit report must render before Currency Conversion Audit.
- Empty annex sections render explicit empty-state text instead of disappearing.
- Annex content must not include activity after selected report-year end.

State transitions:

- `calculated -> rendered_markdown_annex` for Markdown output.
- `calculated -> rendered_pdf_annex_section` for PDF output.

## PerAssetAuditSection

Purpose: A grouped Annex 1 section that traces one reported asset's activity history through the selected year end.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_identity_key` | string | Stable internal asset identity used for ordering and traceability |
| `display_label` | string | User-facing asset label |
| `entries` | list of `AuditActivityEntry` | Activity entries on or before year end |

Relationships:

- Belongs to one `AuditAnnex`.
- Groups `AuditActivityEntry` values for one asset.
- Uses the same ordering as report calculation replay.

Validation rules:

- Asset identity key is required.
- Display label must be present or derive from the identity key during rendering.
- Entries must be ordered deterministically by existing report replay ordering.
- Entries must exclude post-year activity.

## AuditActivityEntry

Purpose: One Annex 1 row or subsection describing one activity after it has been applied to report cost-basis state.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Non-secret activity reference |
| `occurred_at` | timestamp | Stored activity timestamp rendered in deterministic UTC or report contract form |
| `activity_type` | enum | Report-facing activity type, with zero-priced SELL displayed as `BLOCKCHAIN OP` |
| `quantity` | decimal | Activity quantity |
| `unit_price` | decimal nullable | Selected or converted unit price when applicable |
| `gross_value` | decimal nullable | Selected or converted gross value when applicable |
| `fee_amount` | decimal nullable | Selected or converted fee when applicable |
| `activity_currency` | string nullable | Original selected activity currency when applicable |
| `calculation_currency` | string | Report base currency when monetary values are calculated |
| `quantity_after_activity` | decimal | Held quantity after applying the activity |
| `basis_after_activity` | decimal | Open cost basis after applying the activity |
| `full_liquidation_event` | boolean | True when the activity brings holdings to zero |
| `allocated_basis` | decimal nullable | Present for disposal activities that allocate basis |
| `net_liquidation_proceeds` | decimal nullable | Present for priced liquidations |
| `gain_or_loss` | decimal nullable | Present when the activity realizes a gain or loss |
| `conversion_status` | enum nullable | Same-currency or converted label source when applicable |
| `note` | string | Redacted explanatory note when present |

Relationships:

- Belongs to one `PerAssetAuditSection`.
- May reference one `LiquidationCalculation` result.
- May be cross-referenced by a `ConversionAuditEntry` through `source_id`.

Validation rules:

- Source ID, timestamp, activity type, quantity, quantity-after, and basis-after are required.
- Monetary values must use exact decimals and explicit currency context.
- `gain_or_loss` must be present for priced liquidation activities.
- Zero-priced holding reductions must not claim proceeds and must render as `BLOCKCHAIN OP`.
- Notes must be sanitized through existing report rendering/redaction rules.

## CurrencyConversionAuditSection

Purpose: The Annex 1 section that discloses currency conversion evidence for converted activities.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `entries` | list of `ConversionAuditEntry` | Existing converted activity evidence |
| `empty_message` | string | Rendered when no converted activity is present |

Relationships:

- Belongs to one `AuditAnnex`.
- Uses existing `ConversionAuditEntry`, `ConvertedActivityAmount`, and `ExchangeRateEvidence` models from the base-currency conversion feature.
- No longer renders in the main report.

Validation rules:

- Section is required for every annex.
- If entries are empty, render a clear no-converted-activity message.
- Quote direction must render through a user-friendly label map.
- Missing quote-direction label mapping fails rendering before final output.
- Provider-level authority and rate kind remain in Rate Source Summary, not repeated as per-row audit columns unless existing model validation requires retained evidence.

## RenderLabelMap

Purpose: Closed mapping from internal report enum values to user-facing report labels.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `conversion_status_labels` | map | Maps `same_currency` and `converted` to user-friendly labels |
| `quote_direction_labels` | map | Maps quote direction enums to user-friendly labels |
| `activity_type_labels` | map | Maps zero-priced SELL display override to `BLOCKCHAIN OP` |

Relationships:

- Used by Markdown and PDF renderers.
- Enforced before output bundle writing.

Validation rules:

- Renderers must not expose unmapped enum constants or snake_case values.
- Missing mapping returns a render error and no final output success.

## ReportDocument

Purpose: One rendered document before final output-file writing.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `document_type` | enum | `markdown` or `pdf` |
| `role` | enum | `main`, `annex`, or `combined` |
| `content` | bytes or string | Markdown text or PDF bytes |
| `year` | integer | Selected report year |
| `cost_basis_method` | enum | Selected method |
| `generated_at` | timestamp | Timestamp used for naming |

Relationships:

- Markdown output creates two documents: main and annex.
- PDF output creates one combined document.
- Written by `ReportOutputBundle` writer.

Validation rules:

- Document type and role must be compatible.
- Markdown documents must contain non-empty text.
- PDF documents must contain non-empty PDF bytes and use `.pdf` output.
- Generated timestamp, year, and method are required for naming.

## ReportOutputBundle

Purpose: The final generated-file outcome for one successful report generation.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `output_format` | `ReportOutputFormat` | Selected output format |
| `files` | list of `ReportOutputFile` | One file for PDF, two files for Markdown |
| `saved_at` | timestamp | Save completion timestamp |
| `open_requested` | boolean | Whether automatic open was requested |
| `open_error` | string | Non-empty only after successful save and failed open |

Relationships:

- Returned by runtime in `ReportOutcome`.
- Rendered by TUI result screen.

Validation rules:

- Markdown output must contain exactly two files: main report and Annex 1.
- PDF output must contain exactly one file: combined main report and Annex 1.
- Every file path must be inside the resolved Documents directory.
- Failed bundle writes must clean up every file created by the attempt.
- Open error requires an open request and must not convert a successful save into a failed save.

## ReportOutputFile

Purpose: One saved file within a report output bundle.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `documents_directory` | string | Resolved local Documents directory |
| `filename` | string | Generated safe filename |
| `path` | string | Absolute saved path |
| `role` | enum | `main`, `annex`, or `combined` |
| `media_type` | string | `text/markdown` or `application/pdf` |
| `saved_at` | timestamp | Save timestamp |

Relationships:

- Belongs to one `ReportOutputBundle`.
- Displayed in TUI result screen on success.

Validation rules:

- Filename and path are required.
- Filename must follow the report output filename contract.
- File mode remains owner-only as currently enforced by output writer policy.
- Paths and filenames must not contain secrets.
