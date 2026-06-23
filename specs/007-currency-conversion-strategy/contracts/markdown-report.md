# Contract: Markdown Report Currency Conversion

## Scope

This contract extends the generated Markdown capital gains report so successful reports disclose the selected report base currency and exchange-rate audit evidence.

**Bugfix**: 2026-06-24 — BUG-004 Updated compact Currency Conversion Audit table contract.

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
- Explicit zero monetary values remain `0` after conversion handling.
- Zero-priced holding reductions with no proceeds and no acquisition cost do not require conversion audit entries.

## Conversion Audit Section

When the report contains converted priced activities, it includes a conversion audit section or an equivalent per-activity audit subsection.

Recommended section heading:

```markdown
## Currency Conversion Audit
```

Required table columns or equivalent fields:

Superseded by BUG-004: ~~Old required columns were `Date`, `Source ID`, `Asset`, `Source Currency`, `Report Base Currency`, `Rate Date`, `Rate Authority`, `Rate Kind`, `Quote Direction`, `Rate Value`, `Amount Kind`, `Original Amount`, and `Converted Amount`.~~ BUG-004 moves provider-level authority and rate kind to `Rate Source Summary` and changes the rendered audit column order.

```markdown
| Date | Source ID | Asset | Amount Kind | Rate Date | Source Currency | Original Amount | Report Base Currency | Converted Amount | Quote Direction | Rate Value |
|------|-----------|-------|-------------|-----------|-----------------|-----------------|----------------------|------------------|-----------------|------------|
```

Rules:

- Every converted priced activity must have audit details.
- ~~Audit details must include source currency, report base currency, activity date, rate date, rate authority, rate kind when applicable, rate value, original amount, and converted amount.~~ Superseded by BUG-004 because provider-level authority and rate kind are disclosed in `Rate Source Summary`.
- Audit details must include source currency, report base currency, activity date, amount kind, rate date, quote direction, rate value, original amount, and converted amount.
- The audit table must not include `Rate Authority` or `Rate Kind` columns; those provider-level fields are disclosed in `Rate Source Summary`.
- Audit details must preserve provider-published rate precision.
- When the rate date differs from the activity date, both dates must be visible.
- The audit section must be reproducible from synced activity inputs, selected base currency, provider, rate date, rate value, quote direction, and rounding rules.
- Same-currency rows may be excluded from the conversion audit section if the activity detail tables clearly mark them as same-currency.

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
