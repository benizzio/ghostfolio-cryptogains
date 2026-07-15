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

1. **Given** a report contains the initial `Year`, `Cost Basis Method`, `Generated At`, and `Report Calculation Currency` fields, **When** either output format is generated, **Then** the exact warning `The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.` appears as a standalone fully bold line immediately after those fields and before `Gains-And-Losses Summary`.
2. **Given** report-visible financial values of `1`, `1.004`, `1.005`, and `-1.005`, **When** the report is generated, **Then** they are displayed as `1.00`, `1.00`, `1.01`, and `-1.01` respectively.
3. **Given** a report contains currency-denominated values in summaries, positions, activity rows, liquidation results, or converted amounts, **When** either output format is generated, **Then** every present amount and unit price displays exactly two digits after the decimal separator.
4. **Given** a Currency Conversion Audit row discloses the exchange rate used, **When** either output format is generated, **Then** the rate retains its established provider-published precision so that the disclosed conversion evidence remains reproducible.
5. **Given** a report contains quantity values with zero, fewer than two, or more than two fractional digits, **When** the report is generated, **Then** each quantity retains its established exact report representation and is not rounded or padded to two decimal places by this feature.
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

**Independent Test**: Generate both report formats with Currency Conversion Audit rows containing one, two, and three converted amount entries. Verify a one-to-one mapping between included entries and visible lines, exact spacing around labels and arrows, retained entry order, and two-decimal financial presentation.

**Acceptance Scenarios**:

1. **Given** a conversion audit row contains unit price, gross value, and fee conversions, **When** either output format is generated, **Then** the `Converted Amounts` cell presents the entries on three separate visible lines in this form:

   ```text
   unit_price: 30754.70 -> 28673.04;
   gross_value: 254.76 -> 237.52;
   fee_amount: 1.79 -> 1.67
   ```

2. **Given** a conversion audit row contains more than one included converted amount, **When** it is rendered, **Then** each non-final entry ends with a semicolon followed by a visible line break, the final entry has no trailing semicolon, and entries are not separated only by spaces.
3. **Given** a conversion audit row contains only one included converted amount, **When** it is rendered, **Then** that entry occupies one visible line and has no trailing semicolon.
4. **Given** a conversion component has both an original amount and converted amount of zero, **When** the row is rendered, **Then** that zero-to-zero component remains omitted and this feature does not add a placeholder line.

### Edge Cases

- Present financial values that already have zero, one, or two fractional digits must still display exactly two fractional digits.
- Positive and negative values exactly halfway between two two-decimal results must round HALF UP, away from zero at the tie.
- A negative value that rounds to zero must display as `0.00`, not `-0.00`.
- A missing optional financial value must remain blank and must not be presented as `0.00`.
- Very small, very large, and negative financial values must follow the same two-decimal presentation without changing their sign or currency identity.
- Disclosed exchange-rate ratios with more than two decimal places must retain their established provider-published precision; they are conversion evidence rather than currency-denominated amounts.
- Quantity values such as `2`, `0.1`, and `0.00000001` must retain their established quantity representation rather than becoming `2.00`, `0.10`, or `0.00`.
- A zero-priced holding reduction must have a blank `Original Activity Currency` cell even when the report calculation currency is available; the calculation currency cell must remain populated.
- An audit row with no applicable converted amounts must retain the existing empty-state behavior rather than gaining an empty conversion line.
- A converted amount entry may wrap because of available page or column width, but every subsequent entry must still begin on a new visible line.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST apply all presentation changes in this specification to both Markdown and PDF capital gains and losses report outputs, including their Annex 1 content.
- **FR-002**: The system MUST render the exact warning `The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.` as a standalone line immediately after the initial `Report Calculation Currency` field and immediately before the `Gains-And-Losses Summary` subheading.
- **FR-003**: The complete warning defined by FR-002 MUST be bold, including its final period, and no portion of the warning may be rendered with regular emphasis.
- **FR-004**: Every present report-visible currency-denominated amount and unit price MUST display exactly two digits after the decimal separator, including trailing zeros.
- **FR-005**: Values governed by FR-004 MUST include, at minimum, unit prices, gross values, fees, cost basis values, allocated basis values, liquidation proceeds, gains and losses, summary totals, and original and converted activity amounts wherever they appear in the main report or Annex 1. Disclosed exchange-rate ratios MUST retain their established provider-published precision and are not governed by the two-decimal display rule.
- **FR-006**: When a financial value requires transformation to two decimal places, the system MUST round it using HALF UP rounding.
- **FR-007**: Two-decimal rounding MUST occur only while preparing the final visible report after calculations are complete; a rounded display value MUST NOT be reused for any calculation, total, conversion, comparison, stored value, or subsequent report value.
- **FR-008**: This feature MUST NOT change calculation precision, currency identity, exchange-rate selection, conversion logic, cost-basis allocation, gains-and-losses calculation, or report activity inclusion.
- **FR-009**: Quantity values MUST retain their established report precision and representation and MUST NOT be rounded, padded, or otherwise transformed by the two-decimal financial presentation rule.
- **FR-010**: Missing optional financial values MUST remain blank and MUST NOT be converted into zero-valued output.
- **FR-011**: A financial value that rounds to zero MUST be rendered as unsigned `0.00` and MUST NOT be rendered as negative zero.
- **FR-012**: Every structured boolean value displayed in the main report or Annex 1 MUST use exactly `Yes` for true and `No` for false; the report MUST NOT expose `true` or `false` as boolean values.
- **FR-013**: In a Detailed Per-Asset Audit Report row for an operation that reduces holdings with a zero source unit price, the system MUST leave the `Original Activity Currency` value blank.
- **FR-014**: Applying FR-013 MUST NOT remove or alter the row's `Calculation Currency`, quantity, activity classification, held quantity, basis effects, liquidation evidence, gains or losses, or other applicable audit values.
- **FR-015**: Activities with an applicable non-zero source price MUST continue to show the source activity currency selected for that row as `Original Activity Currency`.
- **FR-016**: Within each Currency Conversion Audit Table `Converted Amounts` cell, every included converted amount entry MUST begin on a separate visible line and remain in the existing entry order.
- **FR-017**: Each converted amount entry MUST use the visible form `<field label>: <original amount> -> <converted amount>`, with exactly one ordinary space after the colon and exactly one ordinary space on each side of the literal `->` arrow.
- **FR-018**: Adjacent converted amount entries MUST be separated by a semicolon followed by a format-appropriate visible line break; the final entry MUST NOT have a trailing semicolon.
- **FR-019**: Converted amount entries MUST retain the labels and order `unit_price`, `gross_value`, and `fee_amount`; an entry whose original and converted values are both zero MUST remain omitted. This feature changes only the visible arrangement, spacing, and two-decimal presentation of included amounts.
- **FR-020**: Shared report content MUST satisfy the same warning, financial formatting, quantity preservation, boolean labeling, audit-currency, and converted-amount requirements in both output formats.
- **FR-021**: This feature MUST NOT add, remove, or reorder report sections, table columns, activities, assets, totals, or audit evidence except for the warning insertion and presentation transformations explicitly required above.

### Scope Boundaries

- **In Scope**: Visible content in generated Markdown and PDF main reports and Annex 1, including the legal-use warning, decimal financial presentation, structured boolean labels, zero-priced holding-reduction currency display, and Converted Amounts line layout.
- **Out of Scope**: Financial calculation algorithms, calculated or stored precision, quantity formatting changes, disclosed exchange-rate precision, exchange-rate selection, currency conversion rules, cost-basis methods, report data selection, report output-format selection, unrelated terminal screens, stored activity data, empirical dataset maintenance, and new external services.

### Financial Calculation Evidence

- **Numeric Representation**: Existing exact-decimal values and explicit currency identities remain authoritative. This feature introduces a two-decimal display representation only for report-visible currency-denominated amounts and unit prices. Exact quantity representation and provider-published exchange-rate precision remain unchanged.
- **Conversion And Rounding**: No new currency conversion boundary or source is authorized. Currency-denominated amounts and unit prices are rounded only for final report display at two decimal places using HALF UP; rounded output must not feed back into calculations, and disclosed exchange-rate ratios remain at their established precision.
- **Empirical Solidified Financial Tests**: Existing empirical financial tests remain applicable as regression evidence that calculations and internal precision do not change. Presentation-specific scenarios supplement rather than replace that evidence.
- **Empirical External Dataset Changes**: The empirical external dataset and generated oracle fixtures remain read-only for this feature.

### Security, Persistence, And Integration Evidence

- **Persistence Impact**: This feature changes existing user-requested cleartext report exports only. It adds no application-managed state, cache, report history, remote persistence, or automatic re-ingestion of generated files.
- **Token Handling Impact**: Existing secret-exclusion requirements remain unchanged. Tokens and reusable authentication material must not appear in report content, errors, diagnostics, examples, or fixtures.
- **External Integration Impact**: No new external data source, remote report service, telemetry destination, or third-party dependency is required by this feature.
- **Security Review Scope**: Review must confirm that the new warning and presentation transformations do not disclose secrets, replace missing values with misleading zeros, or change local report-file handling. The feature adds no authentication, authorization, network, or remote-storage surface.

### Testing Evidence

- **Acceptance Coverage**: Automated report-generation checks must verify every acceptance scenario in both Markdown and PDF outputs, including HALF UP boundaries, negative zero, missing values, full-precision exchange rates, high-precision quantities, both boolean states, zero- and non-zero-priced activity currencies, and one-to-three converted amount entries.
- **Calculation Regression**: Established deterministic and empirical financial evidence must continue to pass without changing expected calculated results or the read-only empirical dataset.
- **Coverage**: All affected behavior and decision outcomes must remain fully covered under the project's required 100% coverage standard.
- **Scale Regression**: The established 10,000-cached-activity report scenario must continue to complete each selected output format independently within its existing two-minute limit.

### Quality Gate Evidence *(mandatory)*

- **Changed Source Inputs**: Changes to report presentation source and related tests are expected. No dependency-file change is expected because this feature requires no new dependency.
- **Quality Gate Command**: `make quality QUALITY_BASE_REF=origin/main` must pass locally or through the successful `quality` GitHub Actions check.
- **No-Source-Change Behavior**: Not expected to apply because report presentation source changes are required; if implementation planning removes all source changes, the gate must still pass with explicit skip messages.

### Key Entities

- **Report-Visible Financial Value**: A currency-denominated amount or unit price, including gross value, fee, basis, proceeds, gain or loss, total, and original or converted activity amount, presented to the report reader with explicit currency identity.
- **Quantity Value**: An asset amount or held/disposed quantity whose established exact report representation is independent of the two-decimal financial display rule.
- **Disclosed Exchange Rate**: The provider-published ratio used for a currency conversion and shown at its established precision so a reviewer can reproduce the conversion evidence.
- **Structured Report Boolean**: A true-or-false report field presented to a reader as `Yes` or `No`, excluding arbitrary note text that happens to contain those words.
- **Zero-Priced Holding Reduction**: An activity that reduces an asset holding while its source unit price is zero, making original activity currency irrelevant to its calculation effect.
- **Converted Amount Entry**: One labeled original-to-converted financial value pair within a Currency Conversion Audit Table row, such as unit price, gross value, or fee amount.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In 100% of generated Markdown and PDF acceptance reports, the exact legal-use warning appears once, fully bold, between `Report Calculation Currency` and `Gains-And-Losses Summary`.
- **SC-002**: In 100% of generated acceptance reports, every present currency-denominated amount and unit price has exactly two decimal places, and all exact-half boundary cases produce the HALF UP result.
- **SC-003**: Across 100% of established calculation and empirical regression cases, calculated financial results, quantities, full-precision disclosed rates, currency identities, and included activities remain unchanged before visible amount formatting.
- **SC-004**: In 100% of quantity-bearing acceptance rows, the displayed quantity matches its established exact report representation and is never changed solely to satisfy the financial two-decimal rule.
- **SC-005**: In 100% of structured boolean fields across both output formats, true is displayed as `Yes`, false is displayed as `No`, and no lowercase `true` or `false` boolean value remains.
- **SC-006**: In 100% of zero-priced holding-reduction audit rows, `Original Activity Currency` is blank and `Calculation Currency` remains present; in 100% of control rows with an applicable non-zero source price, the existing original currency remains present.
- **SC-007**: In every Currency Conversion Audit acceptance row, the number of separately started visible lines in `Converted Amounts` equals the number of included converted amount entries, and every entry uses the required colon and arrow spacing.
- **SC-008**: For identical report inputs, Markdown and PDF outputs agree on 100% of the warning text, displayed financial values, quantity values, boolean labels, applicable currency values, and converted amount entry order.
- **SC-009**: From the `Converted Amounts` cell alone, an acceptance reviewer can correctly identify the field label, original amount, and converted amount for 100% of included unit price, gross value, and fee entries.
- **SC-010**: At the established 10,000-cached-activity scale, one Markdown generation and one PDF generation each continue to complete independently in under two minutes.

## Assumptions

- `Financial value` in issue 45 is interpreted as a currency-denominated amount or unit price. A disclosed exchange-rate ratio is conversion evidence rather than a currency-denominated amount, so its established provider-published precision remains visible for audit reproducibility.
- Exactly two decimal places means trailing zeros are visible, so a displayed whole financial value appears as `1.00`.
- HALF UP applies symmetrically to positive and negative ties, with ties rounded away from zero.
- Missing optional values are not financial zero values and remain blank.
- Values that round to zero use the neutral representation `0.00` to avoid presenting a non-existent negative amount.
- Existing quantity presentation is the compatibility baseline and remains unchanged even when a quantity could visually resemble a monetary amount.
- Converted Amounts retain the existing label order `unit_price`, `gross_value`, and `fee_amount`, with zero-to-zero entries omitted; a format-appropriate line break changes presentation without changing the entry data.
- The scope is limited to generated report documents. Other terminal user-interface values are unchanged.
- Existing report output files remain explicit local user-requested exports outside application-managed persistence.
- Existing report-generation scale targets and supported output formats remain unchanged.
- No new dependency or external integration is needed.
- Existing empirical external datasets and generated oracle fixtures remain unchanged and read-only.
