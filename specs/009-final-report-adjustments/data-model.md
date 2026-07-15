# Data Model: Final Report Adjustments

## Modeling Notes

This feature adds no persisted schema, protected snapshot field, normalized
activity field, calculation algorithm, report section, or output format. It adds
one transient zero-priced-reduction classification to `AuditActivityEntry` and
defines derived presentation values over the existing exact report model.
Generated Markdown and PDF files remain intentional cleartext user exports under
EXP-001: they are direct local requests, every path and the cleartext
financial-data status are disclosed, and users are told to delete every listed
file to remove the data. No temporary cleartext file, additional copy, report
history, durable path state, reopen catalog, or automatic re-ingestion is added.

The authoritative source remains `CapitalGainsReport` and its existing nested
models. Every source monetary amount, quantity, exchange rate, and currency
identity remains unchanged. Derived strings exist only during rendering and are
never passed back into calculation, conversion, comparison, or persistence.

The validated `CapitalGainsReport` immediately before presentation is the
AUD-001 comparison baseline. Rendering leaves every exact decimal, currency,
quantity, rate and metadata field, inclusion or omission state, and
classification equal to that baseline. Intentional cleartext export status
authorizes only contracted report fields; it does not authorize real user
financial data or other SEC-001-prohibited material in errors, logs, screenshots,
documentation examples, committed or generated fixtures, or unrelated document
content. Diagnostics retain their separately reviewed mode-specific policy.

## LegalUseWarning

Purpose: The fixed caution shown once in each main report before the first
financial summary.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `text` | constant string | Exact sentence ending in a period |
| `placement` | enum | After `Report Calculation Currency`, before `Gains-And-Losses Summary` |
| `emphasis` | enum | Entire sentence is bold |
| `document_scope` | enum | Markdown main document and combined PDF main section; not the separate Markdown Annex |

Validation rules:

- Text is exactly `The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.`
- The complete text, including the final period, is one bold standalone
  paragraph.
- Standalone is a logical block property. PDF width-driven physical lines or
  text runs remain ordered fragments of the same single warning occurrence.
- It occurs exactly once per generated report output.
- It is not stored in calculated or persisted report data.

State transitions:

- `shared_constant -> markdown_bold_paragraph`
- `shared_constant -> pdf_bold_wrapped_paragraph`

## ReportVisibleFinancialValue

Purpose: A final visible representation of one present currency-denominated
amount or unit price.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_value` | `apd.Decimal` | Existing exact finite calculated value; read-only |
| `currency_identity` | string or surrounding column | Existing explicit currency context; unchanged |
| `display_exponent` | constant integer | `-2` |
| `rounding_mode` | constant enum | `apd.RoundHalfUp` |
| `display_value` | `apd.Decimal` | Defensive quantized copy used only to create text |
| `rendered_text` | string | Fixed-point text with exactly two fractional digits |

Relationships:

- Derived from monetary fields in `AssetSummaryEntry`, `CapitalGainsReport`,
  `AssetDetailSection`, `AssetActivityRow`, `LiquidationCalculation`,
  `AuditActivityEntry`, and `ConvertedActivityAmount`.
- Rendered identically by Markdown and PDF.
- Keeps the existing explicit report, activity, original, converted, or
  calculation currency context associated with its source field.

Validation rules:

- Source value must be finite.
- Formatting must not mutate `source_value` or its coefficient storage.
- Quantization starts from a copy of `apd.BaseContext` so the normative adjusted-
  exponent bounds `-100000` through `100000` and default traps remain active.
- Required precision is derived with checked arithmetic from source digit count,
  expansion to exponent `-2`, and one possible carry digit; a result above
  unsigned 32-bit maximum `4294967295` is rejected before quantization.
- A source at adjusted exponent `100000` is accepted only when HALF UP
  quantization cannot carry its result to adjusted exponent `100001`.
- Only `Rounded` and `Inexact` operation conditions are accepted; every other
  condition or error fails rendering.
- Every rendered present value matches `^-?[0-9]+\.[0-9]{2}$`.
- Exact halves round away from zero under HALF UP.
- A quantized zero has no negative sign and renders exactly `0.00`.
- A non-zero source that displays as `0.00` remains non-zero for all model and
  inclusion decisions.

State transitions:

- `exact_present -> defensive_copy -> quantized_scale_2 -> zero_normalized -> rendered_text`
- `non_finite -> render_error`

## OptionalReportVisibleFinancialValue

Purpose: Preserve the distinction between an absent monetary value and a
present exact zero.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_value` | nullable `apd.Decimal` | Existing optional report-model field |
| `rendered_text` | string | Blank when absent; otherwise a `ReportVisibleFinancialValue` string |

Validation rules:

- `nil` renders as an empty cell or value, never `0.00`.
- Present exact zero renders as `0.00`.
- Presence is evaluated before formatting.

State transitions:

- `absent -> blank`
- `present -> ReportVisibleFinancialValue`

## QuantityValue

Purpose: An exact asset amount whose established canonical representation is
not a monetary display value.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_value` | `apd.Decimal` | Existing quantity or held/disposed quantity |
| `rendered_text` | string | FR-009 canonical fixed-point representation |

Relationships:

- Includes activity quantity, quantity after activity/row, disposed quantity,
  opening quantity, and closing quantity.
- May appear adjacent to `ReportVisibleFinancialValue` fields but never adopts
  their display scale.

Validation rules:

- Derive text from the exact finite pre-presentation quantity, not a source
  lexeme or current generated output.
- Expand without exponent notation or grouping, trim only fractional trailing
  zeros, omit an empty decimal part, use no leading `+`, and render zero as `0`.
- Do not round or pad solely for this feature.
- Values mathematically equal to `2.000`, `0.1000`, and `0.00000001` render as
  `2`, `0.1`, and `0.00000001`.

State transitions:

- `exact_quantity -> canonical_text`

## DisclosedExchangeRate

Purpose: Normalized exact provider conversion evidence retained without loss of
significant digits for reproducibility.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `rate_value` | `apd.Decimal` | Existing normalized exact `ConversionAuditEntry.RateValue`; provider lexical scale is not retained |
| `quote_direction` | enum and display label | Existing source/base direction |
| `source_currency` | string | Existing source currency |
| `report_base_currency` | string | Existing base currency |
| `rendered_text` | string | Canonical fixed-point rate representation; never scale-2 quantized |

Validation rules:

- Do not quantize to two decimal places.
- Preserve every significant digit and quote direction from the normalized rate
  evidence. Do not preserve non-significant provider lexical trailing zeros.
- Render `0.86010`, `16.9140`, `1.094600`, and `2.00` source spellings as
  normalized exact values `0.8601`, `16.914`, `1.0946`, and `2`.
- Do not reuse a displayed financial amount to derive or verify a rate.

State transitions:

- `exact_provider_rate -> canonical_text`

## StructuredReportBoolean

Purpose: A reader-facing label for a true-or-false report field.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_value` | boolean | Existing structured model value |
| `rendered_label` | enum string | `Yes` or `No` |

Relationships:

- Currently maps `AuditActivityEntry.FullLiquidationEvent` into
  `AnnexActivityRow.FullLiquidationEvent`.
- Does not transform arbitrary note or descriptive text.

Validation rules:

- `true` renders exactly `Yes`.
- `false` renders exactly `No`.
- Renderers consume the shared label directly and do not convert it back to a
  boolean.

State transitions:

- `true -> Yes`
- `false -> No`

## ZeroPricedHoldingReductionAuditPresentation

Purpose: The Annex 1 currency applicability rule for an activity carrying the
inherited report-level zero-priced holding-reduction classification.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `is_zero_priced_holding_reduction` | boolean | Existing exact input classification copied into `AuditActivityEntry` |
| `pre_format_activity_currency` | string | Existing `AuditActivityEntry.ActivityCurrency`; retained unchanged without asserting source provenance |
| `visible_original_activity_currency` | nullable string | Blank only for the classified operation |
| `calculation_currency` | string | Existing required report calculation currency |
| `audit_entry` | `AuditActivityEntry` | Existing activity, quantity, basis, liquidation, conversion, and note evidence |

Relationships:

- Constructed from the same `ActivityCalculationInput` and basis application
  result already used for Annex audit evidence.
- The calculated audit entry retains its existing activity-currency value and
  the copied classification. Zero-priced rows may have no selected source
  context, so the retained value is not reclassified as source provenance.
- Shared presentation derives the visible currency consumed by both renderers.

Validation rules:

- The copied classification is authoritative for presentation and is computed
  before conversion or formatting from the inherited normalized `SELL`, positive
  holding-reduction quantity, non-empty explanation, nonnegative running holdings,
  and the rule that every present source monetary field across the order,
  asset-profile, and base tiers is finite and numerically zero.
- No monetary field must be present for the inherited report-level compatibility
  classification. All-missing, mixed missing-and-zero, explicit-zero-unit-price,
  and all-explicit-zero shapes qualify. Missing remains nil and visible blank; it
  is not converted into a numeric zero.
- Sync admission remains unchanged and continues to require resolvable amount
  evidence for newly synced activities.
- A positive selected or same-tier-derived source unit price remains non-zero
  even when it displays as `0.00`.
- A classified zero-priced holding reduction retains
  `pre_format_activity_currency` but has blank
  `visible_original_activity_currency`.
- `calculation_currency` remains populated.
- Quantity, activity classification, held quantity, basis, proceeds, gains or
  losses, conversion status, and notes are not changed by this rule.
- An applicable non-zero-priced activity retains its selected source activity
  currency.
- Applicability is never inferred from a two-decimal display string.

State transitions:

- `pre_format_currency_and_classification -> classified_zero_priced_reduction -> visible_currency_blank`
- `pre_format_currency_and_classification -> other_activity -> visible_currency_retained`

## ConvertedAmountEntryPresentation

Purpose: One ordered original-to-converted financial pair in a Currency
Conversion Audit row.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `amount_kind` | enum | Existing `unit_price`, `gross_value`, or `fee_amount` |
| `original_amount` | `apd.Decimal` | Existing exact amount |
| `converted_amount` | `apd.Decimal` | Existing exact converted amount |
| `original_currency` | string | Existing explicit source currency |
| `report_base_currency` | string | Existing explicit converted currency |
| `included` | boolean | False only when both exact amounts are zero |
| `rendered_entry` | string | `<label>: <original> -> <converted>` with two-place amounts |

Relationships:

- Derived in the received `ConversionAuditEntry.Amounts` order.
- Collected by one `ConvertedAmountsCellPresentation`.
- Original and converted values each follow `ReportVisibleFinancialValue`.

Validation rules:

- Presentation treats the inherited `ConversionAuditEntry.Amounts` sequence as
  read-only and preserves the relative order of included entries.
- This feature adds no duplicate-kind or supported-kind list-order validation;
  inherited validation of each individual amount kind remains unchanged.
- Omit only when both exact values have sign zero.
- Include exact non-zero values even when both visible values become `0.00`.
- Use exactly one ordinary space after `:` and around `->`.
- The logical entry itself has no semicolon or renderer-specific line break.

State transitions:

- `exact_zero_to_zero -> omitted`
- `included -> two_decimal_pair -> logical_entry`

## ConvertedAmountsCellPresentation

Purpose: Format-specific visible arrangement of zero or more converted amount
entries in one existing table cell.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `entries` | ordered list of `ConvertedAmountEntryPresentation` | Included entries only |
| `markdown_text` | string | Entries joined by `;<br>` |
| `pdf_text` | string | Entries joined by `;\n` |

Validation rules:

- Every included entry has one logical start. The first is the cell origin and
  each later start follows a renderer-controlled boundary; physical width wraps
  inside one entry do not create another logical start.
- Every non-final entry ends with a semicolon before the visible line break.
- The final entry has no trailing semicolon.
- One entry has no separator.
- Zero entries preserve the existing empty-cell behavior.
- Renderer-specific delimiters are added only after each logical entry is
  sanitized as single-line content.
- Markdown escapes HTML-sensitive and table-delimiter characters in each dynamic
  label or amount component before assembling the fixed literal `: ` and ` -> `
  syntax and adding the controlled `<br>` delimiter.
- PDF row-height measurement applies the same indicator-sensitive space break
  option as table drawing, accounts for explicit newlines, and completes before
  page preflight and drawing.
- A measured row that fits only on a fresh page advances before any part of the
  row is drawn and is drawn whole exactly once.
- A measured row that exceeds the fresh-page row area fails rendering and is
  never split, clipped, overlapped, finalized, or saved.

State transitions:

- `logical_entries -> markdown_joined_cell`
- `logical_entries -> pdf_joined_cell -> drawing_equivalent_measurement -> drawn_cell`
- `measured_row -> fits_fresh_page_only -> page_advanced -> drawn_whole`
- `measured_row -> exceeds_fresh_page_area -> render_error`

## PDF Finalization

PDF finalization serializes a completed in-memory layout into the bytes used to
construct the combined PDF `ReportDocument`. It is part of rendering and occurs
before output reservation.

State transitions:

- `pdf_layout_complete -> pdf_bytes_finalized -> ReportDocument`
- `pdf_layout_or_finalization_error -> render_failed -> no ReportDocument`

## Existing ReportDocument And Output Bundle

The existing output models do not change:

- Markdown still produces one main `ReportDocument` and one Annex
  `ReportDocument`.
- PDF still produces one combined `ReportDocument`.
- Output writing still reserves all paths, requests mode `0600`, and runs its
  existing cleanup sequence after a failed bundle attempt.
- Success requires all selected-format files to be reserved, written, synced,
  closed, validated, and recorded in one valid bundle.
- Any pre-success output failure closes and removes every path reserved or
  created by that attempt and reports no saved path. Cleanup is limited to that
  attempt and leaves pre-existing colliding files and earlier successful bundles
  unchanged.
- Opener failure after bundle success retains every file and produces a
  success-with-warning result.
- Normal success and opener-warning success identify every saved file as a
  cleartext financial-data export and instruct the user to delete all listed
  paths to remove the exported data.
- Leaving the result flow clears transient report content and paths. No report
  history, reopen catalog, durable path state, additional cleartext copy, or
  automatic re-ingestion is created.
- The warning and derived display strings are document content only and do not
  add fields to `ReportDocument`, `ReportOutputFile`, or `ReportOutputBundle`.

Output state transitions:

- `reserved -> all_files_written_synced_closed -> bundle_validated -> success`
- `any_pre_success_failure -> remove_current_attempt_reservations -> failure`
- `success -> opener_failure -> success_with_warning_and_files_retained`
- `success_or_opener_warning -> disclose_paths_and_deletion -> user_managed_files`

Overall transient flow:

```text
exact CapitalGainsReport
  -> format-neutral presentation values
  -> Markdown main + Annex documents OR combined PDF document
  -> existing local output bundle writer
```
