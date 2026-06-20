# Feature Specification: Report Base Currency Conversion

**Feature Branch**: `[007-currency-conversion-strategy]`

**Created**: 2026-06-20

**Status**: Draft

**Input**: User description: "let's specify a solution for issue https://github.com/benizzio/ghostfolio-cryptogains/issues/5"

**Source Issue**: [Currency Conversion Strategy for Multi-Currency Asset Pricing](https://github.com/benizzio/ghostfolio-cryptogains/issues/5)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Select A Report Base Currency (Priority: P1)

When generating a yearly capital gains and losses report, the user can choose one report base currency so all monetary calculations and report totals are expressed in the same currency.

**Why this priority**: Mixed-currency activity cannot produce defensible cost basis, proceeds, gains, or losses until the report run has one explicit base currency.

**Independent Test**: Generate a report from a synced dataset containing priced activities in more than one currency, choose USD once and EUR once, and verify that each successful report expresses all cross-activity monetary outputs in the selected base currency.

**Acceptance Scenarios**:

1. **Given** synced data contains at least one reportable year, **When** the user starts report generation, **Then** the system requires one report base currency selection before calculation begins.
2. **Given** the user is selecting a report base currency, **When** the available choices are shown, **Then** only USD and EUR are selectable.
3. **Given** a priced activity's selected activity currency already matches the report base currency, **When** the report is calculated, **Then** the activity's monetary values enter cost basis, proceeds, gains, and losses without exchange-rate conversion.
4. **Given** a priced activity's selected activity currency differs from the report base currency, **When** the report is calculated, **Then** the activity's monetary values are converted to the report base currency before they enter cost basis, proceeds, gains, or losses.
5. **Given** the report completes successfully, **When** the user reads the generated report, **Then** the report identifies the selected base currency and all cross-activity monetary totals use that currency instead of `NOT APPLICABLE`.

---

### User Story 2 - Use Official Historical Conversion Rates (Priority: P1)

The user receives a report whose conversions use an official or officially trusted source for the selected base currency and the original activity date, with enough rate detail to audit the calculation.

**Why this priority**: Capital gains reporting must be reproducible and defensible. The user needs to know which authority supplied each rate and which date was used.

**Independent Test**: Use a deterministic dataset with activities in currencies different from the selected base currency, compare each converted amount against expected values derived from the documented official authority source and activity date, and verify that the report exposes the source and rate metadata.

**Acceptance Scenarios**:

1. **Given** the report base currency is EUR, **When** any priced activity must be converted, **Then** conversion rates are sourced from the European Central Bank or a source officially trusted or authorized by the European Central Bank for conversion into EUR.
2. **Given** the report base currency is USD, **When** any priced activity must be converted, **Then** conversion rates are sourced from the Federal Reserve or a source officially trusted or authorized by the Federal Reserve for conversion into USD.
3. **Given** an activity requires conversion, **When** the rate is selected, **Then** the rate is based on the original source-calendar date of that activity, not the report-generation date, sync date, or machine-local date.
4. **Given** the official source publishes only one daily reference or closing rate for the activity date, **When** the rate is selected, **Then** that daily rate is used and the report identifies it as a daily reference or closing rate.
5. **Given** the official source publishes no rate for the activity date, **When** a prior business date has a rate available from the authorized provider, **Then** the most recent previous business-date rate from that provider is used and the report discloses both the activity date and the rate date.
6. **Given** a report contains converted monetary values, **When** the user reviews conversion details, **Then** the report shows the source currency, report base currency, activity date, rate date, rate authority, rate value, original amount, and converted amount for each converted priced activity or for an equivalent per-activity audit section.

---

### User Story 3 - Fail Safely When Conversion Is Not Defensible (Priority: P2)

When an official conversion cannot be obtained or the activity currency is not supported by the selected authority source, the report attempt fails without producing incorrect monetary results.

**Why this priority**: A report with silent fallback rates or missing conversion evidence is worse than no report because it can misstate taxable gains or losses.

**Independent Test**: Use synced data with an unsupported source currency and with a required historical rate intentionally unavailable, start report generation, and verify that no final report file is produced and the user receives an actionable non-secret explanation.

**Acceptance Scenarios**:

1. **Given** a priced activity requires conversion and no official or officially authorized rate is available for its source currency and activity date under the selected report base currency, **When** report generation reaches that activity, **Then** the report attempt fails before the final report file is saved.
2. **Given** the selected authority source is temporarily unavailable and the required rate is not already defensibly available for the report run, **When** report generation attempts conversion, **Then** the report attempt fails with an actionable message rather than using an unofficial fallback.
3. **Given** conversion fails for one activity, **When** the failure is reported, **Then** the user remains inside the unlocked reporting context, no partial cleartext report artifact remains, and the message identifies the affected source currency, report base currency, and activity date without exposing tokens or protected authentication material.
4. **Given** a zero-priced holding reduction contributes no proceeds or acquisition cost, **When** the report is calculated, **Then** it does not require an exchange rate solely because the row reduces quantity or because explicit zero-valued source fields were preserved.

---

### Edge Cases

- A report contains both activities already denominated in the selected base currency and activities that require conversion.
- The same asset has acquisitions in one currency and liquidations in another currency before the selected cost basis method is applied.
- A fee and gross value are both present in one selected activity currency context and must be converted together before they affect basis or proceeds.
- A priced activity has a valid explicit zero fee, so the fee converts to zero and remains valid.
- A selected activity currency code is missing, malformed, or not supported by the selected authority source.
- The activity date falls on a weekend, public holiday, or other date where the authority source publishes no rate, so conversion falls back to the last previous business date where the authorized provider can supply a rate.
- The authority source revises a previously published rate after an earlier report was generated.
- Conversion succeeds for many activities and then fails for one later activity in deterministic history order.
- The report includes only zero-priced holding reductions after the selected year and therefore needs no conversion for those rows.
- The user generates the same year and method twice with different report base currencies and expects separate results and rate audit details.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, financial precision and currency-handling, testing, dependency, and external integration impacts when the feature touches those areas.

### Functional Requirements

- **FR-001**: The system MUST require exactly one report base currency for each capital gains and losses report run.
- **FR-002**: The system MUST support USD and EUR as the only report base currency choices for this feature.
- **FR-003**: The system MUST prevent report calculation from starting until the user has selected a report base currency.
- **FR-004**: The system MUST display the selected report base currency in the generated report.
- **FR-005**: The system MUST first select each priced activity's single-activity monetary context according to the existing report rules before any cross-currency conversion is applied.
- **FR-006**: The system MUST NOT mix monetary tiers within one activity to complete conversion inputs.
- **FR-007**: If a priced activity's selected currency equals the report base currency, the system MUST use that activity's selected monetary values without exchange-rate conversion.
- **FR-008**: If a priced activity's selected currency differs from the report base currency, the system MUST convert every monetary value from that selected activity context that can affect cost basis, proceeds, gains, losses, or report totals before the value enters those calculations.
- **FR-009**: The conversion boundary MUST occur after single-activity monetary context selection and before cost basis integration, proceeds calculation, gain or loss calculation, and cross-activity report totals.
- **FR-010**: For an EUR base-currency report, the system MUST use exchange rates from the European Central Bank or from a source officially trusted or authorized by the European Central Bank for converting all required source currencies into EUR.
- **FR-011**: For a USD base-currency report, the system MUST use exchange rates from the Federal Reserve or from a source officially trusted or authorized by the Federal Reserve for converting all required source currencies into USD.
- **FR-012**: The system MUST NOT silently fall back to unofficial, community, market-data, or application-defined exchange-rate sources when an official or officially authorized source cannot provide the required rate.
- **FR-013**: The system MUST select conversion rates by the original source-calendar date of each activity, using the preserved activity timestamp and its own stored offset.
- **FR-014**: If the authority source publishes only one daily reference or closing rate for the selected activity date, the system MUST use that daily rate and identify the rate kind in the report.
- **FR-015**: If the authority source publishes no rate on the activity date, the system MUST use the most recent previous business-date rate available from the authorized provider and disclose the activity date and actual rate date.
- **FR-016**: The system MUST fail report generation if no official or officially authorized rate is available for a required source currency, report base currency, and activity date under the rules above.
- **FR-017**: The system MUST fail report generation if the selected authority source is unavailable and the required rate is not defensibly available for the report run.
- **FR-018**: The system MUST express all cost basis, proceeds, realized gains, realized losses, and cross-activity monetary totals in the selected report base currency for successful converted reports.
- **FR-019**: The system MUST preserve each priced activity's original selected currency and original selected monetary values in report audit details when conversion occurs.
- **FR-020**: For each converted priced activity, the generated report MUST include or reference audit details containing source currency, report base currency, activity date, rate date, rate authority, rate kind when applicable, rate value, original amount, and converted amount.
- **FR-021**: The report MUST distinguish same-currency activity values from converted activity values so the user can tell whether an exchange rate changed a row's monetary values.
- **FR-022**: Zero-priced holding reductions that create no proceeds and no acquisition cost MUST NOT require exchange-rate lookup solely because they reduce quantity or preserve explicit zero-valued monetary source fields.
- **FR-023**: Explicit zero-valued monetary source fields that are valid under existing report rules MUST remain zero after conversion handling and MUST NOT create gains, losses, proceeds, or fees by conversion.
- **FR-024**: Conversion calculations MUST use exact decimal arithmetic, MUST preserve published rate precision, MUST keep every monetary amount tied to its currency until conversion, and MUST avoid floating-point behavior in financial decisions or assertions.
- **FR-025**: When a required conversion formula divides or otherwise needs a bounded internal decimal result, the system MUST use the existing report internal precision policy of 16 decimal places with round half up handling before later calculations continue.
- **FR-026**: The system MUST make successful report calculations reproducible from synced activity inputs, selected report base currency, selected rate source, rate dates, rate values, and documented rounding rules.
- **FR-027**: If conversion fails before the final report file is saved, the system MUST leave no partial cleartext report artifact behind and MUST keep the user inside the unlocked reporting context with an actionable non-secret error.
- **FR-028**: Conversion failure messages and diagnostics MUST exclude Ghostfolio security tokens, bearer tokens, reusable token verifiers, and raw authentication material.
- **FR-029**: Outside explicit development mode, diagnostics for conversion failures MUST follow the existing production redaction policy for financial-value fields while still identifying the affected activity by non-secret reference, source currency, report base currency, and activity date.
- **FR-030**: The implementation plan MUST record the selected official or officially authorized data source for USD and EUR, the authority relationship, supported currencies, historical coverage, unavailable-date behavior, revision behavior, failure modes, and test evidence before implementation begins.
- **FR-031**: The system MUST treat the existing no-conversion `NOT APPLICABLE` report calculation currency behavior as superseded for reports generated under this feature.

### Financial Calculation Evidence *(include when feature affects financial calculations)*

- **Numeric Representation**: Monetary amounts, quantities, exchange rates, converted amounts, cost basis, proceeds, gains, and losses use exact decimal values. Every monetary value remains tied to an explicit currency before conversion and to the selected report base currency after conversion.
- **Conversion And Rounding**: Conversion is authorized only at the report-generation boundary after one activity monetary context has been selected and before cost basis or gain and loss calculations consume that activity's monetary values. Rates come from the official or officially authorized source for the selected report base currency and the original activity date. Daily reference or closing rates are acceptable when that is the authority source's available historical rate. Non-publication dates use the last previous business date where the authorized provider can supply a conversion rate and must disclose that substitution. Internal decimal bounding uses the existing 16-decimal round half up policy where required.
- **Empirical Solidified Financial Tests**: Existing empirical financial tests for cost-basis methods remain single-currency calculation evidence and MUST NOT be changed in nature to validate conversion behavior. They remain focused on calculation correctness without conversions. New project-owned contract, integration, and unit coverage is required for conversion boundaries, rate-source selection rules, audit details, and failure handling.
- **Empirical External Dataset Changes**: The empirical external dataset remains read-only for this feature. This feature is not a dataset-maintenance specification.

### Key Entities *(include if feature involves data)*

- **Report Base Currency**: The one user-selected currency, USD or EUR, used for all monetary calculations and totals in a single report run.
- **Selected Activity Monetary Context**: The one existing activity-level monetary context chosen before conversion, including its source currency and values that may affect basis, proceeds, fees, gains, or losses.
- **Exchange Rate Evidence**: The authority-backed rate information used to convert one source currency into the report base currency for one activity date, including authority, rate kind, rate date, rate value, and source-to-base direction.
- **Converted Activity Amount**: A monetary value produced from a selected activity monetary value and exchange rate evidence, expressed in the report base currency before entering report calculations.
- **Conversion Audit Entry**: The report-visible evidence that connects an original activity amount, selected currency, activity date, authority source, rate date, rate value, and converted amount.
- **Conversion Failure**: A report-generation failure caused by missing currency identity, unsupported source currency, unavailable authoritative rate evidence, or unavailable authority source.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance testing, 100% of successful mixed-currency reports express cost basis, proceeds, realized gains, realized losses, and report totals in the user-selected base currency.
- **SC-002**: For a deterministic test dataset containing at least 50 priced activities across at least 3 source currencies and 2 report years, 100% of converted activities use the expected authority-backed rate date and produce expected converted values under the documented rounding policy.
- **SC-003**: 100% of converted priced activities in generated reports include audit details identifying source currency, report base currency, activity date, rate date, rate authority, rate value, original amount, and converted amount.
- **SC-004**: 100% of report attempts with missing authoritative rates, unsupported source currencies, or unavailable required authority data fail before final report save and leave no partial cleartext report artifact.
- **SC-005**: Users can select a report base currency and proceed from report setup to generation confirmation in under 30 seconds during normal interactive use when synced reportable data is already available.
- **SC-006**: Existing single-currency report cases where the selected activity currency equals the report base currency preserve the same monetary results as the prior no-conversion behavior in 100% of regression cases.
- **SC-007**: 100% of production-mode conversion failure diagnostics exclude Ghostfolio security tokens, bearer tokens, reusable token verifiers, and unredacted financial-value fields.

## Assumptions

- The user-facing base currency choices for this feature are limited to USD and EUR because the issue explicitly names those currencies and authorities.
- Activity currency identity is expected to use explicit currency codes already preserved by synced activity data.
- The original activity date is the source-calendar date derived from the preserved activity timestamp and its stored offset, consistent with the existing report specification.
- When an authorized provider has no rate for a weekend, public holiday, or other non-publication day, the last previous business date where that provider can supply a conversion rate is the required fallback unless later planning identifies a stricter authority-specific rule.
- The saved Markdown report remains intentionally cleartext after generation, consistent with the existing reporting feature, and must contain enough non-secret conversion evidence for user audit.
- No new long-lived exchange-rate cache is required by this specification. If planning introduces any persisted rate data, that plan must justify storage, protection, invalidation, and user removal.
- Official source access details, supported currency coverage, and authority trust evidence are planning research outputs, not user choices.
- The empirical external dataset is unchanged by this feature.
