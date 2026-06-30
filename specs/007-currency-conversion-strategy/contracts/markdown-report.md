# Contract: Markdown Report Currency Conversion

## Scope

This contract extends the generated Markdown capital gains report so successful reports disclose the selected report base currency and exchange-rate audit evidence.

**Bugfix**: 2026-06-24 — BUG-004 Updated compact Currency Conversion Audit table contract.

**Bugfix**: 2026-06-24 — BUG-005 Updated Currency Conversion Audit row cardinality and zero-to-zero amount suppression contract.

**Bugfix**: 2026-06-28 — BUG-006 Clarified that audited converted activities must not render as same-currency rows in asset detail sections.

**Bugfix**: 2026-06-28 — BUG-007 Clarified Asset Detail table currency-column naming, ordering, and liquidation-column omission.

## Header

The report header includes:

```markdown
# Ghostfolio Capital Gains And Losses Report

- Year: <year>
- Cost Basis Method: <method label>
- Generated At: <local timestamp>
- Report Calculation Currency: <USD or EUR>
```

Rules:

- `Report Calculation Currency` must equal the selected report base currency.
- `NOT APPLICABLE` is superseded for reports generated under this feature.
- The selected base currency is not inferred from activities.

## Monetary Tables

Rules:

- Cost basis, proceeds, realized gains, realized losses, open-position basis, closing-position basis, per-asset net totals, and overall yearly net total are rendered in the selected report base currency.
- Same-currency activity rows remain distinguishable from converted rows.
- Activity currency columns continue to show the selected activity currency before conversion.
- Calculation currency columns show the selected report base currency.
- Asset detail sections must render same-currency versus converted status from the selected activity currency or explicit conversion status preserved before conversion, not from post-conversion report-base amount currency.
- `Asset Detail` `In-Year Activity` tables must render columns in this exact order: `Date`, `Source ID`, `Type`, `Quantity`, `Unit Price`, `Gross Value`, `Fee`, `Quantity After Row`, `Basis After Row`, `Original Activity Currency`, `Calculation Currency`, `Conversion Status`, and `Note`.
- `Asset Detail` `In-Year Activity` tables must use `Original Activity Currency`, not `Activity Currency`, for the selected activity currency before conversion.
- `Asset Detail` `Liquidation Calculations` tables must not include an `Activity Currency` column.
- Any `Source ID` present in `Currency Conversion Audit` must not be labeled `same currency` in asset detail sections.
- Explicit zero monetary values remain `0` after conversion handling, but zero-to-zero converted amount slots do not render as standalone audit rows or grouped amount items.
- Zero-priced holding reductions with no proceeds and no acquisition cost do not require conversion audit entries.

## Conversion Audit Section

When the report contains converted priced activities, it includes a conversion audit section or an equivalent per-activity audit subsection.

Recommended section heading:

```markdown
## Currency Conversion Audit
```

Required grouped table columns or equivalent per-activity fields:

Superseded by BUG-004: ~~Old required columns were `Date`, `Source ID`, `Asset`, `Source Currency`, `Report Base Currency`, `Rate Date`, `Rate Authority`, `Rate Kind`, `Quote Direction`, `Rate Value`, `Amount Kind`, `Original Amount`, and `Converted Amount`.~~ BUG-004 moves provider-level authority and rate kind to `Rate Source Summary` and changes the rendered audit column order.

Superseded by BUG-005: ~~BUG-004 compact per-amount columns were `Date`, `Source ID`, `Asset`, `Amount Kind`, `Rate Date`, `Source Currency`, `Original Amount`, `Report Base Currency`, `Converted Amount`, `Quote Direction`, and `Rate Value`.~~ BUG-005 groups amount-kind conversions by converted source activity.

```markdown
| Date | Source ID | Asset | Rate Date | Source Currency | Report Base Currency | Converted Amounts | Quote Direction | Rate Value |
|------|-----------|-------|-----------|-----------------|----------------------|-------------------|-----------------|------------|
```

Rules:

- Every converted priced activity must have audit details.
- ~~Audit details must include source currency, report base currency, activity date, rate date, rate authority, rate kind when applicable, rate value, original amount, and converted amount.~~ Superseded by BUG-004 because provider-level authority and rate kind are disclosed in `Rate Source Summary`.
- Audit details must include source currency, report base currency, activity date, amount kind, rate date, quote direction, rate value, original amount, and converted amount.
- The rendered audit must emit one table row or equivalent subsection per converted source activity, not one row per converted amount kind.
- Multiple amount kinds for the same converted source activity must be grouped in `Converted Amounts` or an equivalent field, using a stable representation such as `unit_price: <original> -> <converted>; gross_value: <original> -> <converted>`.
- Amount slots where both original amount and converted amount are exactly zero must be omitted from the rendered audit grouping.
- The audit table must not include `Rate Authority` or `Rate Kind` columns; those provider-level fields are disclosed in `Rate Source Summary`.
- Audit details must preserve provider-published rate precision.
- When the rate date differs from the activity date, both dates must be visible.
- The audit section must be reproducible from synced activity inputs, selected base currency, provider, rate date, rate value, quote direction, and rounding rules.
- Same-currency rows may be excluded from the conversion audit section if the activity detail tables clearly mark them as same-currency.
- A `Source ID` included in this section is definitive converted-activity evidence for asset detail rendering and must not be contradicted by a `same currency` label elsewhere in the report.

## Rate Source Summary

Successful reports include a concise source summary identifying the provider used for the selected base currency.

Required content for EUR reports:

- authority: European Central Bank
- provider: ECB Data Portal `EXR`
- rate kind: daily euro foreign exchange reference rate
- unavailable-date rule: most recent previous available ECB observation

Required content for USD reports:

- authority: Federal Reserve
- provider: Federal Reserve Board H.10/Data Download Program
- rate kind: noon buying rate in New York for cable transfers payable in listed currencies
- unavailable-date rule: most recent previous available H.10 observation

Rules:

- `Rate Source Summary` is the disclosure location for provider-level authority and rate-kind metadata that is shared by the selected report base currency provider.

## Empty Audit Behavior

If no priced activity required conversion:

- The report still identifies the selected report base currency.
- The report may omit `Currency Conversion Audit` or render a clear empty-state sentence such as `No activity required exchange-rate conversion.`
- Same-currency regression reports must preserve prior monetary results when the selected activity currency equals the selected report base currency.

## Secret Handling

The Markdown report must not contain:

- Ghostfolio security tokens
- bearer JWTs
- reusable token verifiers
- raw protected payload bytes
- provider request headers containing secrets

The final report intentionally contains cleartext financial values and conversion evidence because it is the user-requested audit artifact.
