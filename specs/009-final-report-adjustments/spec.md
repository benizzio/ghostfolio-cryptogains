# Feature Specification: Final Report Adjustments

**Feature Branch**: `[009-final-report-adjustments]`

**Created**: 2026-07-15

**Status**: Draft

**Input**: User description: "https://github.com/benizzio/ghostfolio-cryptogains/issues/45: Final report adjustments"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Read Release-Ready Report Values (Priority: P1)

As a user reviewing a capital gains and losses report, I want a prominent legal-use warning and consistently rounded financial values so that I understand the report's limitations and can read its monetary information without inconsistent decimal precision.

**Why this priority**: The warning prevents the report from being mistaken for country-specific tax-return guidance, while consistent financial presentation is required across every report section for the MVP release.

**Independent Test**: Generate the same representative report as Markdown and PDF with positive, negative, zero, whole-number, high-precision, and exact-half currency-denominated values. Verify the warning's exact text, location, and emphasis; verify every displayed amount and unit price has two decimal places using HALF UP rounding; verify quantities, disclosed exchange-rate precision, and calculated results remain unchanged.

**Acceptance Scenarios**:

1. **Given** a report contains the initial `Year`, `Cost Basis Method`, `Generated At`, and `Report Calculation Currency` fields, **When** either output format is generated, **Then** the exact warning `The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.` appears as one standalone fully bold logical paragraph immediately after those fields and before `Gains-And-Losses Summary`, even if PDF width causes multiple physical lines or text runs.
2. **Given** report-visible financial values of `1`, `1.004`, `1.005`, and `-1.005`, **When** the report is generated, **Then** they are displayed as `1.00`, `1.00`, `1.01`, and `-1.01` respectively.
3. **Given** a report contains currency-denominated values in summaries, positions, activity rows, liquidation results, or converted amounts, **When** either output format is generated, **Then** every present amount and unit price displays exactly two digits after the decimal separator.
4. **Given** a Currency Conversion Audit row discloses a normalized provider rate whose source spelling was `0.86010`, `16.9140`, or `1.0946`, **When** either output format is generated, **Then** its canonical visible text is `0.8601`, `16.914`, or `1.0946` respectively, with every significant digit and the selected quote direction unchanged.
5. **Given** exact report quantities mathematically equal to `0`, `2.000`, `0.1000`, and `0.00000001`, **When** the report is generated, **Then** their FR-009 canonical text is `0`, `2`, `0.1`, and `0.00000001` in both formats and is not rounded or padded to two decimal places.
6. **Given** the same source activities and report request are used before and after this feature, **When** the report is calculated, **Then** all quantities, basis values, proceeds, gains, losses, rates, and totals before visible formatting remain identical.

---

### User Story 2 - Interpret Audit Values Clearly (Priority: P2)

As a user reviewing Annex 1, I want readable boolean labels and only meaningful currency information so that audit rows do not expose implementation-style values or imply a source currency where a zero price had no calculation effect.

**Why this priority**: Audit evidence must communicate its meaning directly and must not present a misleading original currency for zero-priced holding reductions.

**Independent Test**: Generate both report formats with true and false audit states, a zero-priced holding reduction, and an ordinarily priced activity. Verify boolean labels, blank and retained original-currency cells, and unchanged calculation-currency evidence.

**Acceptance Scenarios**:

1. **Given** a structured report boolean value is true, **When** either output format is generated, **Then** the value is shown as `Yes` and not `true`.
2. **Given** a structured report boolean value is false, **When** either output format is generated, **Then** the value is shown as `No` and not `false`.
3. **Given** a Detailed Per-Asset Audit Report row reduces holdings with a zero source unit price, **When** Annex 1 is generated, **Then** its `Original Activity Currency` cell is blank while its `Calculation Currency` and all other applicable audit evidence remain visible.
4. **Given** a Detailed Per-Asset Audit Report row has an applicable non-zero source price, **When** Annex 1 is generated, **Then** the row continues to show its selected source activity currency as `Original Activity Currency`.

---

### User Story 3 - Scan Converted Amounts (Priority: P3)

As a user reviewing the Annex 1 Currency Conversion Audit Table, I want each converted amount on a separate, consistently spaced line so that I can distinguish unit price, gross value, and fee conversions without parsing a dense inline list.

**Why this priority**: The conversion evidence is already complete, but its current inline presentation is difficult to scan, especially in constrained table columns.

**Independent Test**: Generate both report formats with Currency Conversion Audit rows containing zero, one, two, and three converted amount entries and every canonical subsequence defined by FR-019. Verify a one-to-one mapping between included entries and logical entry starts, exact spacing around labels and arrows, retained entry order, and two-decimal financial presentation. Width-driven physical wraps inside one entry do not create another entry start.

**Acceptance Scenarios**:

1. **Given** a conversion audit row contains unit price, gross value, and fee conversions, **When** either output format is generated, **Then** the `Converted Amounts` cell presents three separate logical entry starts in this form, even if an entry later wraps physically because of width:

   ```text
   unit_price: 30754.70 -> 28673.04;
   gross_value: 254.76 -> 237.52;
   fee_amount: 1.79 -> 1.67
   ```

2. **Given** a conversion audit row contains more than one included converted amount, **When** it is rendered, **Then** each non-final entry ends with a semicolon followed by a visible line break, the final entry has no trailing semicolon, and entries are not separated only by spaces.
3. **Given** a conversion audit row contains only one included converted amount, **When** it is rendered, **Then** that entry starts at the cell origin, has no renderer-controlled preceding line break or trailing semicolon, and may occupy multiple physical lines only through width-driven wrapping.
4. **Given** a conversion component has both an original amount and converted amount of zero, **When** the row is rendered, **Then** that zero-to-zero component remains omitted and this feature does not add a placeholder line.
5. **Given** conversion rows collectively contain each FR-019 canonical subsequence, **When** both formats are rendered, **Then** every included label retains subsequence order and the empty subsequence retains the existing empty-cell result without a synthetic line.

### Edge Cases

- Present financial values that already have zero, one, or two fractional digits must still display exactly two fractional digits.
- Positive and negative values exactly halfway between two two-decimal results must round HALF UP, away from zero at the tie.
- A negative value that rounds to zero must display as `0.00`, not `-0.00`.
- A missing optional financial value must remain blank and must not be presented as `0.00`.
- Within the FR-004a accepted domain, very small, very large, and negative financial values must follow the same two-decimal presentation without changing their sign or currency identity; values outside that domain follow FR-022 instead.
- Disclosed exchange-rate ratios use the canonical normalized provider-rate representation: provider lexical trailing zeros are removed, but no significant digit is rounded or discarded.
- Quantity values such as `2.000`, `0.1000`, and `0.00000001` must use canonical fixed-point text `2`, `0.1`, and `0.00000001` rather than financial two-place formatting.
- A zero-priced holding reduction must have a blank `Original Activity Currency` cell even when the report calculation currency is available; the calculation currency cell must remain populated.
- An audit row with no applicable converted amounts must retain the existing empty-state behavior rather than gaining an empty conversion line.
- A converted amount entry may wrap into additional physical lines because of available page or column width, but every subsequent entry must still begin after a new renderer-controlled logical line boundary.
- A PDF row that does not fit the remaining row area but fits on a fresh page must move before any part of the row is drawn and must remain whole.
- A PDF row that cannot fit in the row area of a fresh page must fail rendering rather than split, clip, overlap, or repeatedly advance through empty pages.

### Failure And Recovery Acceptance Scenarios

1. **Given** a present report-visible decimal is non-finite or cannot be represented in the required visible grammar, **When** the selected renderer prepares the report, **Then** generation returns an actionable non-secret render failure, returns no rendered document, reports no successful output path, and invokes neither output writing nor automatic opening.
2. **Given** PDF text measurement, explicit-newline wrapping, page-fit preparation, or drawing fails, **When** PDF generation is attempted, **Then** generation returns a contextual layout failure before finalization, returns no successful PDF document or path, and leaves the application available for another report attempt.
3. **Given** a completed in-memory PDF cannot be finalized into PDF bytes, **When** finalization is attempted, **Then** the error returns through normal report-failure handling without terminating the application, partial bytes are discarded, no output path is reserved or written, no opener is invoked, and no successful result is shown.
4. **Given** Markdown output has exclusively reserved its current-attempt main path, **When** reservation of the matching Annex path encounters an unrecoverable non-collision IO error after ordinary collision suffix selection has been handled, **Then** the reserved main path is closed and removed, no Annex path is treated as reserved, neither path is reported as saved, and the attempt returns failure. An ordinary collision instead abandons that candidate safely and retries the matched pair under FR-025.
5. **Given** Markdown output has reserved both current-attempt paths and the main file has been fully written, **When** Annex writing, syncing, closing, validation, or bundle recording fails, **Then** both current-attempt paths are closed and removed, neither path is reported as saved, and the attempt returns failure.
6. **Given** PDF output has reserved or partly written its current-attempt path, **When** writing, syncing, closing, validation, or bundle recording fails, **Then** that current-attempt path is removed, no PDF path is reported as saved, and the attempt returns failure.
7. **Given** the Documents directory already contains files whose names collide with the attempted Markdown pair or PDF, **When** the current attempt later fails, **Then** cleanup removes only paths exclusively reserved by the current attempt and every pre-existing file retains its original path and contents.
8. **Given** every selected-format output file has been successfully written, synced, closed, validated, and recorded, **When** automatic opening fails, **Then** the result is success-with-warning, every saved path is reported, and no saved file is removed.
9. **Given** a wrapped `Converted Amounts` row is taller than the remaining current-page row area but fits the fresh-page row area, **When** PDF pagination preflights the row, **Then** the page advances before any row text, cell, or border is drawn and the complete row is drawn once with the required header and applicable continuation context.
10. **Given** a wrapped `Converted Amounts` row is taller than the fresh-page row area, **When** PDF pagination preflights the row, **Then** PDF rendering returns a contextual layout failure, leaves the application running, finalizes and saves no PDF, and produces no row fragment, clipping, overlap, or repeated empty continuation page.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST apply all presentation changes in this specification to both Markdown and PDF capital gains and losses report outputs, including their Annex 1 content.
- **FR-002**: The system MUST render the exact warning `The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.` as one standalone warning paragraph at the logical document-block level, immediately after the initial `Report Calculation Currency` field and immediately before the `Gains-And-Losses Summary` subheading. Standalone means no metadata value, heading, table content, or other prose shares or interrupts that paragraph; it does not require one physical typeset line. Markdown MUST encode one source paragraph on one source line between paragraph boundaries. PDF MAY wrap it into multiple physical lines or text runs solely because of available width; all ordered fragments remain one occurrence of the same paragraph.
- **FR-003**: The complete warning defined by FR-002 MUST be bold, including its final period, and no portion may use regular emphasis. Every PDF fragment produced by width-driven wrapping remains part of the warning and MUST use the bold font.
- **FR-004**: Every present report-visible currency-denominated amount and unit price MUST use fixed-point ASCII text matching `^-?[0-9]+\.[0-9]{2}$`: an optional leading `-`, one or more decimal digits, the literal `.` separator, and exactly two fractional digits including trailing zeros. A leading `+`, grouping separator, alternate decimal separator, whitespace, or exponent notation is forbidden.
- **FR-004a**: The accepted financial-formatting input domain is every finite decimal whose adjusted base-10 exponent, defined as source exponent plus source coefficient digit count minus one, is between `-100000` and `100000` inclusive, whose correctly HALF UP quantized scale-2 result including any carry also has adjusted exponent no greater than `100000`, and whose required quantization precision fits unsigned 32-bit precision. Required precision is the source coefficient digit count plus any coefficient expansion needed to reach exponent `-2`, plus one possible carry digit, and MUST NOT exceed `4294967295`. Every value in this domain MUST produce an FR-004 string under FR-006, subject to later document-layout validation; a value outside it, including an upper-bound value whose rounding carry would produce adjusted exponent `100001`, MUST return the FR-022 unrepresentable-value error before visible output. Successful-display requirements, including FR-011 and the successful numeric vectors, apply only within this domain.
- **FR-005**: Values governed by FR-004 comprise unit prices, gross values, fees, opening, closing, and historical cost basis values, basis-after-activity values, allocated basis values, liquidation proceeds, gains and losses, per-asset and yearly summary totals, and original and converted activity amounts wherever they appear in the main report or Annex 1. A disclosed `Rate Value` is not governed by FR-004. Its authoritative source is the exact positive normalized provider-rate value selected for the conversion together with its quote direction. The system MUST render that value as a canonical fixed-point decimal without rounding, padding, inversion, reciprocal derivation, rescaling, exponent notation, or grouping separators. Canonical rate text removes fractional trailing zeros and omits the decimal separator for an integral value, so provider source spellings `1.094600`, `16.9140`, and `2.00` render as `1.0946`, `16.914`, and `2`. In this specification, provider-published precision means fidelity to that normalized numeric value, not preservation of the provider field's lexical scale.
- **FR-006**: When a financial value requires transformation to two decimal places, the system MUST round it using HALF UP rounding.
- **FR-007**: Two-decimal rounding MUST occur only while preparing the final visible report after calculations are complete; a rounded display value MUST NOT be reused for any calculation, total, conversion, comparison, stored value, or subsequent report value.
- **FR-008**: This feature MUST NOT change calculation precision, currency identity, exchange-rate selection, conversion logic, cost-basis allocation, gains-and-losses calculation, or report activity inclusion.
- **FR-009**: Every quantity MUST be rendered from the exact finite quantity in the pre-presentation calculated report model using this canonical fixed-point representation: expand the complete base-10 value without exponent notation; use `.` only when a fractional part remains; remove trailing zeros only from the fractional part; remove the decimal separator when that fractional part becomes empty; render numeric zero as `0`; and use no grouping separator, leading `+`, rounding, or fractional padding. Thus values mathematically equal to `2.000`, `0.1000`, and `0.00000001` render as `2`, `0.1`, and `0.00000001`. This baseline is derived without consulting current or previous generated output and MUST NOT alter the model value.
- **FR-010**: Missing optional financial values MUST remain blank and MUST NOT be converted into zero-valued output.
- **FR-011**: An FR-004a accepted financial value that rounds to zero MUST be rendered as unsigned `0.00` and MUST NOT be rendered as negative zero.
- **FR-012**: Every structured boolean value displayed in the main report or Annex 1 MUST use exactly `Yes` for true and `No` for false; the report MUST NOT expose `true` or `false` as boolean values.
- **FR-013**: Source-price predicates MUST be evaluated from normalized exact source data before conversion and visible formatting. A `zero-priced holding reduction` is the inherited classification for a normalized `SELL` whose positive quantity reduces holdings without making them negative, whose trimmed explanatory comment is non-empty, whose normalized source unit price is present and numerically zero, and whose other present source monetary fields across order, asset-profile, and base tiers are finite and numerically zero. Numeric zero includes any exponent, trailing-zero scale, or negative-zero sign; a missing value is not itself evidence of zero. In a Detailed Per-Asset Audit Report row carrying that exact pre-format classification, the system MUST leave `Original Activity Currency` blank and MUST NOT infer the classification from a displayed price, activity label, rounded amount, or currency value.
- **FR-014**: Applying FR-013 MUST NOT remove or alter the row's `Calculation Currency`, quantity, activity classification, held quantity, basis effects, liquidation evidence, gains or losses, or other applicable audit values.
- **FR-015**: An `applicable non-zero source price` exists when the selected pre-conversion single-tier monetary context has a non-empty source currency and a present finite unit price, either source-provided or derived within that same tier, that is numerically greater than zero. Such a row MUST retain the selected source activity currency as `Original Activity Currency`, including when the positive unit price is small enough to display as `0.00`. A row satisfying neither FR-013 nor this predicate retains its inherited original-currency presentation; this feature suppresses no other row.
- **FR-016**: Within each Currency Conversion Audit Table `Converted Amounts` cell, every included converted amount entry MUST have one distinct logical entry start and remain in the existing entry order. The first entry starts at the cell origin; every later entry starts immediately after the renderer-controlled line break required by FR-018. Width-driven physical wrapping inside an entry creates no additional logical entry start and does not satisfy the boundary required before a later entry.
- **FR-017**: Each converted amount entry MUST use the visible form `<field label>: <original amount> -> <converted amount>`, with exactly one ordinary space after the colon and exactly one ordinary space on each side of the literal `->` arrow.
- **FR-018**: Adjacent converted amount entries MUST be separated by a semicolon followed by a format-appropriate visible line break; the final entry MUST NOT have a trailing semicolon.
- **FR-019**: Rendering MUST preserve each included converted entry's existing supported label and relative order after exact zero-to-zero omission. The canonical presentation-fixture subsequences are `[]`, `[unit_price]`, `[gross_value]`, `[fee_amount]`, `[unit_price, gross_value]`, `[unit_price, fee_amount]`, `[gross_value, fee_amount]`, and `[unit_price, gross_value, fee_amount]`; this closed acceptance set is exactly the order-preserving subsets of that three-label sequence and does not impose a new calculator-output or model-validity rule. For any received sequence whose individual kinds pass inherited validation, the renderer MUST NOT sort, deduplicate, synthesize, or reorder entries and MUST NOT add a failure because supported kinds are duplicated or non-canonical. For inherited calculator outputs and direct presentation fixtures alike, this feature changes only visible arrangement, spacing, and two-decimal presentation.
- **FR-020**: Shared report content MUST satisfy the same warning, financial formatting, quantity preservation, boolean labeling, audit-currency, and converted-amount requirements in both output formats.
- **FR-021**: This feature MUST NOT add, remove, or reorder report sections, table columns, activities, assets, totals, or audit evidence except for the warning insertion and presentation transformations explicitly required above.
- **FR-022**: If final report presentation encounters a non-finite decimal, a finite decimal that cannot be represented in the required visible grammar, a formatting precision or exponent limit, an unexpected decimal condition, or a PDF measurement, wrapping, page-fit, drawing, or byte-finalization failure, the system MUST return an actionable, contextual, non-secret report-render error through the normal report-failure flow. It MUST NOT return a rendered document, report successful generation, or fall back to another output format.
- **FR-023**: PDF byte finalization is part of rendering and MUST use an error-returning path. A finalization failure MUST NOT terminate the application process. Rendering, including finalization, MUST complete before output paths are reserved; therefore an FR-022 render failure MUST return no PDF byte payload or report document, MUST invoke neither output writing nor automatic opening, and MUST create no report output file for that attempt.
- **FR-024**: The report-output success, collision, and cleanup rules in `specs/008-report-pdf-annex/contracts/report-output.md` remain normative. Every newly reserved report path MUST be requested with owner-read and owner-write permissions only, mode `0600`, before content is written. A selected-format output is successful only after every required file, meaning both Markdown files or the one combined PDF, has been exclusively reserved, fully written, synced, closed, validated, and recorded in one valid output bundle. A candidate-name collision follows FR-025 suffix selection and is not a terminal reservation failure. If an unrecoverable reservation error or any write, sync, close, output validation, or bundle-recording step fails before success, the attempt MUST return failure, MUST report no saved output path, and MUST close and remove every path reserved or created by that attempt, including a fully written first Markdown file when the second Markdown file fails.
- **FR-025**: Cleanup under FR-024 MUST be limited to paths reserved by the failed attempt. Files that existed before the attempt and files produced by earlier successful attempts MUST retain their paths and contents; a filename collision MUST select a new suffix and MUST NOT overwrite, truncate, or remove an existing file. A failure to open the primary file after FR-024 success remains a success-with-warning and MUST retain and report every saved file.
- **FR-026**: Before drawing any PDF table row, the system MUST measure the complete row using the same explicit-newline and width-driven wrapping rules used for drawing and MUST reserve the bottom margin, repeated table header, and required continuation context. If the complete row does not fit the remaining current-page row area but does fit the row area of a fresh page, the system MUST advance before drawing any part of the row and MUST draw the complete row exactly once on that page. A continued table MUST use its inherited continuation context and repeated header; a table-start relocation before its first row MUST NOT emit a continuation label.
- **FR-027**: If a measured PDF table row cannot fit in the row area of a fresh page after required margins, table header, spacing, and continuation context are reserved, rendering MUST fail with the FR-022 PDF-layout error. The renderer MUST NOT split the logical row across pages, clip or overlap its text or borders, emit repeated empty continuation pages, finalize a PDF, or report success.

### Scope Boundaries

- **In Scope**: Visible content in generated Markdown and PDF main reports and Annex 1, including the legal-use warning, decimal financial presentation, structured boolean labels, zero-priced holding-reduction currency display, and Converted Amounts line layout.
- **Out of Scope**: Financial calculation algorithms, calculated or stored precision, quantity-format behavior changes beyond documenting its inherited canonical baseline, provider-rate numeric precision changes, exchange-rate selection, currency conversion rules, cost-basis methods, report data selection, new converted-amount list cardinality, duplicate-kind, or ordering validation, report output-format selection, unrelated terminal screens, stored activity data, empirical dataset maintenance, and new external services.

### Accessibility And Readability Evidence

- **ACC-001 — Retained Readability Baseline**: This feature MUST preserve the inherited report readability contract. Required PDF report text MUST remain text-based, searchable, and selectable in readers that support those operations. The legal-use warning MUST remain complete in text, standalone, and fully bold; its caution is conveyed by the sentence itself and MUST NOT depend on emphasis alone. Warning wrapping and width-driven wrapping within converted entries MUST preserve complete text and source order. Every later converted entry MUST retain a distinct logical line start, and multiline rows MUST NOT clip or overlap text, borders, following rows, or printable margins.
- **ACC-002 — Assistive-Consumption Scope**: Acceptance is limited to the existing plain-text nature of Markdown output and text-based, searchable, selectable PDF content. This feature does not establish or claim tagged-PDF structure, PDF/UA conformance, semantic table-header associations, a guaranteed assistive reading order, or compatibility or certification for any screen reader or other assistive technology. Searchability or text selection MUST NOT be represented as evidence of those excluded capabilities.

### Financial Calculation Evidence

- **Numeric Representation**: Existing exact-decimal values and explicit currency identities remain authoritative. This feature introduces a two-decimal display representation only for report-visible currency-denominated amounts and unit prices. Quantities use the FR-009 canonical baseline, while rate evidence uses the FR-005 canonical normalized representation.
- **Conversion And Rounding**: No new currency conversion boundary or source is authorized. Currency-denominated amounts and unit prices are rounded only for final report display at two decimal places using HALF UP; rounded output must not feed back into calculations. Disclosed rates retain every significant digit of the normalized selected evidence and never enter the two-decimal formatter.
- **AUD-001 — Pre-Presentation Audit Baseline**: The validated exact calculated report immediately before presentation is the acceptance baseline. For each Markdown and PDF render, every visible financial string MUST derive from its corresponding baseline decimal and currency identity, with differences limited to presentation transformations explicitly authorized here. Presentation MUST NOT mutate any baseline decimal, currency identity, quantity, rate, rate date, quote direction, converted amount, inclusion or omission state, or classification. FR-013 may suppress only the visible original-currency cell and MUST NOT mutate its baseline value. A rendered or rounded value MUST NOT become an input to calculation, conversion, totals, comparison, omission, inclusion, classification, persistence, or a later report value.
- **AUD-002 — Normalized Provider-Rate Representation**: The authoritative disclosed rate is the validated exact decimal after provider-boundary normalization; provider response spelling is not retained as audit data. Its visible value MUST be the FR-005 canonical fixed-point text, MUST preserve every significant digit, and MUST remain paired with the same source currency, report base currency, rate date, and quote direction. Provider text `1.094600` normalizes to `1.0946`, while `1.0900` normalizes to `1.09`.
- **Empirical Solidified Financial Tests**: Existing empirical financial tests remain applicable as regression evidence that calculations and internal precision do not change. Presentation-specific scenarios supplement rather than replace that evidence.
- **Empirical External Dataset Changes**: The empirical external dataset and generated oracle fixtures remain read-only for this feature.

### Security, Persistence, And Integration Evidence

- **Persistence Impact**: This feature changes existing user-requested cleartext report exports only. It adds no application-managed state, cache, report history, remote persistence, or automatic re-ingestion of generated files.
- **SEC-001 — Confidentiality**: Generated Markdown main documents, Markdown Annex documents, combined PDFs, result and error text including wrapped or dependency-originated causes, diagnostic artifacts and fields, documentation or test examples, and committed or generated test fixtures MUST NOT contain any real Ghostfolio or security token, bearer or JWT value, token-derived verifier or key, other material reusable for authentication or decryption, or raw protected-payload material. Raw protected-payload material includes serialized encrypted snapshot or envelope bytes, decrypted serialized payload bytes or fragments, and reversible encodings of any of them. Project-owned boundaries MUST redact or suppress prohibited material before display, return, logging, persistence, or artifact capture. Clearly synthetic, non-reusable sentinels MAY be used only to verify redaction and MUST NOT be copied from user credentials or protected payloads. This does not prohibit contracted report fields or inherited mode-authorized modeled non-secret diagnostic fields; those retain their existing redaction rules and MUST NOT be copied as raw payload serialization.
- **External Integration Impact**: No new external data source, remote report service, telemetry destination, or third-party dependency is required by this feature.
- **DEP-001 — Capability-Failure Gate**: Sufficiency of the currently pinned local, in-process dependencies and project-owned inspection facilities is a planning precondition. If implementation or acceptance evidence shows that they cannot satisfy any requirement, work MUST stop before adding or upgrading a dependency, introducing a browser, external binary, platform service, new network call, remote renderer, remote storage or telemetry path, weakening a requirement, or accepting reduced evidence. The capability gap and local-only alternatives MUST be documented in research; the specification, implementation plan, security, dependency and integration impacts, constitution checks, task list, and acceptance-evidence plan MUST be reviewed and updated before implementation resumes. Every candidate dependency or integration MUST receive the constitution-required review and preserve SEC-001 and local-only report processing. If no compliant plan satisfies the requirement, the feature MUST remain incomplete and MUST NOT be merged.
- **Security Review Scope**: Review must confirm that the new warning and presentation transformations do not disclose secrets, replace missing values with misleading zeros, or change local report-file handling. The feature adds no authentication, authorization, network, or remote-storage surface.

### Financial Presentation Acceptance Matrix

Each row below is a distinct financial field class. Main-report classes MUST be
verified in the Markdown main document and PDF main section. Annex classes MUST
be verified in the separate Markdown Annex and PDF Annex section. A successful
shared-formatter assertion, field class, or format does not satisfy another row
or output boundary.

| Financial field class | Markdown boundary | PDF boundary | Required vector |
|-----------------------|-------------------|--------------|-----------------|
| Per-asset net gain or loss; overall yearly net total | Main | Main | Signed |
| Opening, closing, and historical-position cost basis | Main | Main | Non-negative |
| In-Year Activity unit price, gross value, fee, and basis after row | Main | Main | Non-negative; absent where nullable |
| Liquidation allocated basis | Main | Main | Non-negative; absent where nullable |
| Liquidation net proceeds and gain or loss | Main | Main | Signed; absent where nullable |
| Detailed Per-Asset Audit unit price, gross value, fee, and basis after activity | Annex | Annex | Non-negative; absent where nullable |
| Detailed Per-Asset Audit allocated basis | Annex | Annex | Non-negative; absent where nullable |
| Detailed Per-Asset Audit net proceeds and gain or loss | Annex | Annex | Signed; absent where nullable |
| Original and converted `unit_price`, `gross_value`, and `fee_amount` | Annex | Annex | Non-negative |

The Non-negative vector is the following closed set:

| Exact pre-format value | Required visible value | Boundary |
|------------------------|------------------------|----------|
| `0` | `0.00` | Exact zero |
| `0.00000001` | `0.00` | Exact non-zero below visible scale |
| `1` | `1.00` | Whole value |
| `1.2` | `1.20` | One fractional digit |
| `1.23` | `1.23` | Already two fractional digits |
| `1.004` | `1.00` | Below positive tie |
| `1.005` | `1.01` | Positive exact tie |
| `1.006` | `1.01` | Above positive tie |
| `9.995` | `10.00` | Carry into a new whole digit |
| `12345678901234567890.123456789` | `12345678901234567890.12` | Very large high-precision value |

The Signed vector contains the Non-negative vector and these additional cases:

| Exact pre-format value | Required visible value | Boundary |
|------------------------|------------------------|----------|
| `-1` | `-1.00` | Negative whole value |
| `-1.004` | `-1.00` | Below negative tie in magnitude |
| `-1.005` | `-1.01` | Negative exact tie |
| `-1.006` | `-1.01` | Above negative tie in magnitude |
| `-9.995` | `-10.00` | Negative carry |
| `-0` | `0.00` | Signed exact zero |
| `-0.004` | `0.00` | Negative value rounded to neutral zero |
| `-0.005` | `-0.01` | Negative zero-adjacent tie |
| `-12345678901234567890.123456789` | `-12345678901234567890.12` | Very large negative value |

Every nullable class also includes an absent source value whose required visible
value is blank. Acceptance coverage is the complete cross-product of each field
class, both specified output boundaries, and every vector member applicable to
that field's signedness, nullability, and inherited visibility rule. Exact-zero
per-asset summary rows retain inherited omission behavior; the exact non-zero
`0.00000001` case remains included and displays `0.00`. Each assertion MUST
compare the exact pre-format value with the actual semantic field text, not only
with an isolated formatter result or punctuation-stripped aggregate PDF text.
These vectors are the complete successful rounding-boundary set. FR-004a
representability limits are exception-flow boundaries rather than visible-field
vector members: formatter evidence verifies both inclusive adjusted-exponent
bounds, an accepted upper-bound non-carry result, and checked precision
arithmetic, while Markdown and PDF render-failure evidence covers adjusted
exponents immediately outside either bound, an upper-bound carry to `100001`,
and required precision above `4294967295` with no successful output.

### Performance Acceptance Definition

- **Multiline 10,000-Activity Fixture**: The authoritative project-owned deterministic fixture contains exactly 10,000 quantity-`1` priced activities for two assets. Each asset has 2,500 `BUY` activities distributed across 2020 through 2024 and 2,500 `SELL` activities in 2025. The combined currency counts are exactly 3,334 USD, 3,333 EUR, and 3,333 GBP activities. Every activity has non-zero gross value and fee and a deterministically same-tier-derived unit price. The fixed report request uses year 2025, HIFO, USD, and generated-at timestamp `2026-05-21T10:00:00Z`. A local deterministic rate service supplies exact rate `1.1` without provider network access. Exactly 6,666 non-USD activities require conversion, and each resulting conversion row contains included `unit_price`, `gross_value`, and `fee_amount` entries.
- **Performance Acceptance Environment**: Authoritative release evidence is the `test-performance / run` check on its GitHub-hosted Ubuntu runner, using the Go version declared by `go.mod` and the exact `make test-performance` command, which invokes `go test -tags=performance ./tests/performance -count=1 -v -parallel=1 -timeout=10m`. The run uses temporary local application and Documents directories, local deterministic rate evidence, and a recorded stub opener, with no provider or rendering network. The evidence MUST record the concrete runner image/version, architecture, available CPU count, and Go version. Local runs are supplemental unless they record equivalent conditions.
- **Per-Format Timing Boundary**: Fixture construction, compilation, temporary-directory setup, protected-snapshot seeding and unlock, request construction, post-generation inspection, and cleanup occur outside measured intervals. For each format, a fresh timer starts immediately before the selected-format generation operation and stops immediately after it returns. The interval includes generation-time validation, calculation, selected rendering, Markdown multiline assembly or PDF multiline measurement and pagination, PDF byte finalization where applicable, output reservation, write, sync, close, bundle validation, and opener invocation. Markdown and PDF run as separate requests with separate timers, threshold assertions, labels, and elapsed-duration records; no assertion aggregates their durations.

### Testing Evidence

- **Acceptance Coverage**: Automated report-generation checks must verify every acceptance scenario and the complete Financial Presentation Acceptance Matrix in both output formats, normalized rates, FR-009 quantities, both boolean states, exact source-price controls, and zero-to-three converted entries.
- **Calculation Regression**: Established deterministic and empirical financial evidence must continue to pass without changing expected calculated results or the read-only empirical dataset.
- **Audit Integrity**: Each renderer must be followed by equality checks against the AUD-001 baseline for exact decimals, currencies, quantities, rates and rate metadata, included and omitted entries, and classifications. Rate cells must equal the AUD-002 canonical baseline.
- **Coverage**: All affected behavior and decision outcomes must remain fully covered under the project's required 100% coverage standard.
- **Scale Regression**: The Performance Acceptance Definition governs the two-minute limit and the multiline pagination evidence; performance evidence remains isolated from canonical coverage.

### Quality Gate Evidence *(mandatory)*

- **Changed Source Inputs**: Changes to report presentation source and related tests are expected. No dependency-file change is expected because this feature requires no new dependency.
- **Quality Gate Command**: `make quality QUALITY_BASE_REF=origin/main` must pass locally or through the successful `quality` GitHub Actions check.
- **No-Source-Change Behavior**: Not expected to apply because report presentation source changes are required; if implementation planning removes all source changes, the gate must still pass with explicit skip messages.

### Key Entities

- **Report-Visible Financial Value**: A currency-denominated amount or unit price, including gross value, fee, basis, proceeds, gain or loss, total, and original or converted activity amount, presented to the report reader with explicit currency identity.
- **Quantity Value**: A finite asset amount or held/disposed quantity whose visible baseline is the canonical fixed-point representation defined by FR-009 and derived from the exact pre-presentation report value rather than a source lexeme or existing output.
- **Disclosed Exchange Rate**: The exact positive normalized decimal selected as conversion evidence and paired with its quote direction. Its canonical fixed-point text preserves all significant digits but not provider-response lexical scale, whitespace, exponent spelling, leading zeros, or non-significant fractional trailing zeros.
- **Structured Report Boolean**: A true-or-false report field presented to a reader as `Yes` or `No`, excluding arbitrary note text that happens to contain those words.
- **Zero-Priced Holding Reduction**: The exact pre-format inherited classification defined by FR-013, not a conclusion drawn from visible `0.00`.
- **Converted Amount Entry**: One labeled original-to-converted financial value pair within a Currency Conversion Audit Table row, such as unit price, gross value, or fee amount.

## Success Criteria *(mandatory)*

### Acceptance Populations And Counting Rules

The closed rendering acceptance case set `A` is the union of these deterministic
case IDs:

- `warning/wrapped`;
- `financial/<matrix-row>/<vector-case>` for every applicable cross-product in the two closed Financial Presentation Acceptance Matrix tables, including each nullable `absent` case and the inherited exact-zero per-asset-summary omission case;
- `quantity/{zero,whole-trailing-zero,fraction-trailing-zero,small,large}` using exact values `0`, `2.000`, `0.1000`, `0.00000001`, and `12345678901234567890.123456789`;
- `rate/{0.86010,16.9140,1.094600,1.0900,2.00}`;
- `boolean/{true,false}`;
- `currency/{zero-priced,nonzero,tiny-positive,derived-positive,zero-with-nonzero-source-field}`; and
- `converted/{empty,unit-price,gross-value,fee-amount,unit-price-gross-value,unit-price-fee-amount,gross-value-fee-amount,all}` corresponding one-to-one with the eight FR-019 subsequences.

Each `A` case contains one exact pre-presentation report model and contributes
exactly two format attempts: one Markdown main-plus-Annex bundle and one combined
PDF. The case schemas above expand only over the closed tables and literal sets
in this specification; cases cannot be added, removed, or filtered during an
acceptance run. A listed attempt that fails generation remains in every
applicable denominator and fails its assertions. Repeated rows and fields are
separate semantic occurrences identified by case ID, document role, section,
asset, source or row identity, field name, amount kind, and amount ordinal;
substring counts are not denominators. Every population below MUST be non-empty.

The established calculation regression population `R` is fixed by feature
baseline `b7de13e597332ca8a1c36af3e05685217ab25f18`. It comprises every top-level
Go test without child subtests and every leaf `t.Run` subtest at that commit in
`internal/report/basis`, `internal/report/calculate`, and `tests/empirical`. A
parent containing child subtests is not counted separately. Each baseline test
or leaf subtest is identified by its fully qualified package, test, and subtest
name and remains in `R` if it fails after this feature; no renderer, output, or
performance case is imported into `R`. Rebasing requires a reviewed
specification update to the baseline identifier before acceptance.

The success-criterion populations are:

- `A`, rendering acceptance cases: the closed case set enumerated above.
- `W`, warning-bearing views: one Markdown main document and one PDF main section per acceptance input; the separate Markdown Annex is excluded.
- `V`, present financial field occurrences: every field expected to be visible in either format, including every applicable Financial Presentation Acceptance Matrix occurrence; absent values and inherited omitted exact-zero summary rows are excluded.
- `R`, established calculation regression cases: the fixed population defined above.
- `M`, model-integrity comparisons: one comparison of the exact pre-presentation model before and after each of the two renderer attempts for every `A` case.
- `Q`, quantity occurrences: every expected opening, closing, historical-position, activity, quantity-after-row or quantity-after-activity, and disposed quantity in both formats.
- `B`, structured boolean occurrences: every visible report-model boolean field in both formats. For this feature this is every `Full Liquidation Event`; arbitrary prose is excluded, and true and false must each occur in each format.
- `Z`, zero-priced controls: every Annex activity occurrence in both formats satisfying FR-013; at least one has a non-empty pre-format activity currency.
- `N`, non-zero-price controls: every Annex activity occurrence in both formats satisfying FR-015.
- `C`, conversion-row occurrences: every expected Currency Conversion Audit row in both formats, including rows with zero included entries; the population contains at least one row for each of the eight canonical subsequences enumerated by FR-019.
- `P`, parity items: each warning, financial field, quantity, disclosed rate and metadata set, boolean, applicable currency field, and converted-entry sequence matched between Markdown and PDF for the same input.
- `E`, included converted-entry occurrences: every logical entry in both formats whose exact original and converted values are not both zero; each supported label occurs in each format.

### Measurable Outcomes

- **SC-001**: For 100% of `W`, the warning occurs exactly once as one logical standalone paragraph, has the exact required text, is fully bold, and is semantically located between `Report Calculation Currency` and `Gains-And-Losses Summary`. PDF width-driven fragments do not create additional occurrences.
- **SC-002**: For 100% of `V`, the visible field has exactly two fractional digits and equals the required HALF UP result for its exact pre-format value. `V` includes every applicable output-boundary, field-class, and numeric-boundary combination in the Financial Presentation Acceptance Matrix.
- **SC-003**: For 100% of `R`, the baseline test or leaf subtest passes without changing its baseline expected calculation result or empirical fixture. For 100% of `M`, post-render comparison under AUD-001 shows zero mutation to financial results, quantities, normalized rates and metadata, currency identities, included or omitted activities or entries, and audit classifications. Every displayed rate in `A` equals its AUD-002 canonical baseline, and exact non-zero probes that display as `0.00` retain their original decisions.
- **SC-004**: For 100% of `Q`, the displayed quantity equals the FR-009 canonical representation computed from the exact pre-presentation quantity without consulting generated output.
- **SC-005**: For 100% of `B`, source true renders exactly `Yes`, source false renders exactly `No`, and no field exposes lowercase `true` or `false`. Both states have non-zero counts in Markdown and PDF.
- **SC-006**: For 100% of `Z`, `Original Activity Currency` is blank and `Calculation Currency` remains present. For 100% of `N`, the selected original activity currency remains present. Both populations are non-empty in each format.
- **SC-007**: For 100% of `C`, the number of logical converted-entry starts equals the number of exact included entries, every later entry starts after a renderer-controlled logical boundary, and each entry uses the required colon and arrow spacing. Width-driven wraps inside one entry do not increase its entry-start count.
- **SC-008**: For 100% of `P`, matched Markdown and PDF semantic items agree. A missing, extra, changed, or reordered item or changed applicability state is a mismatch. Markdown syntax, PDF styling and pagination, page titles, and Markdown Annex file separation are not parity items.
- **SC-009**: For 100% of `E`, the `Converted Amounts` cell alone identifies exactly one supported field label, one original two-decimal amount, and one converted two-decimal amount using the required grammar without relying on another column or internal model inspection.
- **SC-010**: In the Performance Acceptance Environment using the Multiline 10,000-Activity Fixture, one Markdown request and one PDF request each complete their independently measured Per-Format Timing Boundary in strictly less than two minutes. Each format and elapsed duration is recorded separately; no combined duration is evaluated.
- **SC-011**: In the Multiline 10,000-Activity Fixture, the Markdown Annex and PDF Annex each contain all 6,666 expected Currency Conversion Audit rows and all three expected entries per row. Markdown retains controlled entry boundaries. The PDF conversion audit occupies at least two pages, measures and draws every entry without clipping or omission, and repeats the inherited table header and continuation context on each actual continuation page. PDF multiline measurement, pagination, finalization, and save are inside the SC-010 interval; post-generation inspection is outside it.

## Assumptions

- `Financial value` in issue 45 is interpreted as a currency-denominated amount or unit price. A disclosed exchange-rate ratio is conversion evidence rather than a currency-denominated amount, so its canonical normalized value remains visible without monetary two-place formatting.
- Exactly two decimal places means trailing zeros are visible, so a displayed whole financial value appears as `1.00`.
- HALF UP applies symmetrically to positive and negative ties, with ties rounded away from zero.
- Missing optional values are not financial zero values and remain blank.
- Values that round to zero use the neutral representation `0.00` to avoid presenting a non-existent negative amount.
- FR-009 defines the inherited quantity compatibility baseline independently of generated output; this feature does not change it even when a quantity could visually resemble a monetary amount.
- Converted Amounts preserve every received supported-label sequence after zero-to-zero omission. The eight FR-019 canonical subsequences define presentation acceptance coverage, not new calculator output, cardinality, duplicate, order, or model-validity constraints.
- The scope is limited to generated report documents. Other terminal user-interface values are unchanged.
- Existing report output files remain explicit local user-requested exports outside application-managed persistence.
- Existing report-generation scale targets and supported output formats remain unchanged; SC-010 and SC-011 define their acceptance boundary for this feature.
- Existing pinned dependencies and local report capabilities are assumed sufficient only while required evidence confirms that assumption. Contrary evidence invokes DEP-001 and does not authorize an unplanned dependency, integration, or requirement reduction.
- Existing empirical external datasets and generated oracle fixtures remain unchanged and read-only.
