# Data Model: Report Base Currency Conversion

## Modeling Notes

This feature extends report-generation runtime models. It does not add a long-lived exchange-rate cache and does not persist rate evidence into protected snapshots. The final Markdown report remains the intentional cleartext audit artifact.

All financial amounts, quantities, exchange rates, converted amounts, basis values, proceeds, gains, and losses are exact decimals. Every monetary value has an explicit currency before and after conversion.

## ReportBaseCurrency

Purpose: The one user-selected fiat currency used for all report calculations and monetary totals in a report run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `code` | enum | `USD` or `EUR` |
| `label` | string | User-visible label |
| `authority` | enum | `federal_reserve` for `USD`, `european_central_bank` for `EUR` |

Relationships:

- Selected by one `ReportRequest`.
- Selects one `OfficialRateProvider`.
- Becomes the `ReportCalculationCurrency` on successful calculated and rendered reports.

Validation rules:

- Exactly one base currency is required before report calculation starts.
- Only `USD` and `EUR` are valid for this feature.
- The selected base currency applies to every included asset and row in the report.

State transitions:

- `unselected -> selected` when the user chooses `USD` or `EUR`.
- `selected -> submitted` when report generation starts.

## ReportRequest

Purpose: User-selected inputs for one report-generation run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `year` | integer | Must be present in `ProtectedActivityCache.available_report_years` |
| `cost_basis_method` | enum | Existing supported method set |
| `report_base_currency` | `ReportBaseCurrency` | Required `USD` or `EUR` |
| `requested_at` | timestamp | Local generation request time used for output naming |

Relationships:

- Belongs to one active `SyncAndReportsContext`.
- Consumes one `ProtectedActivityCache`.
- Uses one `OfficialRateProvider` selected by `report_base_currency`.
- Produces one `CapitalGainsReport` or one non-secret failure.

Validation rules:

- Year, method, and base currency are all required.
- The year must be selected from synced data, not free text.
- The method must be one of the supported methods.
- The base currency must be `USD` or `EUR`.

State transitions:

- `draft -> validated -> resolving_rates -> calculated -> rendered -> saved` on success.
- `draft -> failed` when request validation fails.
- `resolving_rates -> failed` when official rate evidence is unavailable or non-defensible.

## SelectedActivityMonetaryContext

Purpose: Existing report calculation input after selecting exactly one activity monetary tier and before cross-currency conversion.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Non-secret activity reference |
| `occurred_at` | timestamp | Parsed from stored activity timestamp with source offset preserved |
| `activity_date` | date | Source-calendar date derived from `occurred_at` and its stored offset |
| `selected_currency_context` | enum nullable | `order`, `asset_profile`, or `base` for priced rows |
| `selected_currency_code` | string nullable | Source currency for selected monetary values |
| `unit_price` | decimal nullable | Selected or same-tier derived unit price |
| `gross_value` | decimal nullable | Selected or same-tier derived gross amount |
| `fee_amount` | decimal nullable | Selected fee amount, explicit zero allowed |
| `is_zero_priced_holding_reduction` | boolean | Existing zero-priced `SELL` handling |

Relationships:

- Derived from one stored `ActivityRecord`.
- Produces zero or more `ConvertedActivityAmount` values.
- May produce one `ConversionAuditEntry` when conversion occurs.

Validation rules:

- Values from different monetary tiers must not be mixed.
- A priced activity requires a selected currency code.
- A malformed, empty, or unsupported selected currency code fails the report when conversion would be required.
- Explained zero-priced holding reductions do not require a selected currency context solely because preserved zero-valued fields exist.

State transitions:

- `selected -> same_currency` when selected currency equals report base currency.
- `selected -> conversion_required` when selected currency differs from report base currency.
- `selected -> no_conversion_required` for zero-priced holding reductions with no proceeds and no acquisition cost.

## OfficialRateProvider

Purpose: Fixed official or officially authorized source selected by report base currency.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `provider_id` | enum | `ecb_exr` or `federal_reserve_h10` |
| `authority` | enum | `european_central_bank` or `federal_reserve` |
| `base_currency` | enum | `EUR` or `USD` |
| `rate_kind` | string | Daily reference rate or noon buying rate |
| `endpoint_identity` | string | Fixed provider host and dataset identity, not user-provided |
| `supported_currencies` | string set | Provider-supported source currencies mapped to stored currency codes |

Relationships:

- Selected by `ReportBaseCurrency`.
- Resolves `RateLookupRequest` into `ExchangeRateEvidence`.
- Emits `ConversionFailure` when official evidence cannot be returned.

Validation rules:

- Provider host must be fixed in code and HTTPS.
- Provider must not receive Ghostfolio tokens, JWTs, or protected payload data.
- Provider responses must be mapped into canonical evidence before calculation.
- Unsupported source currencies fail rather than falling back to another source.

State transitions:

- `idle -> fetching -> mapped` on successful provider response.
- `fetching -> failed` on network, HTTP status, parse, unsupported currency, no observation, or authority mismatch failure.

## RateLookupRequest

Purpose: Canonical request for one required source-to-base conversion rate.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_currency` | string | Selected activity currency |
| `base_currency` | string | Report base currency |
| `activity_date` | date | Original source-calendar activity date |
| `provider_id` | enum | Selected from base currency |

Relationships:

- Created from one `SelectedActivityMonetaryContext` requiring conversion.
- Resolved by one `OfficialRateProvider`.
- Produces one `ExchangeRateEvidence` or one `ConversionFailure`.

Validation rules:

- Source and base currencies must be non-empty uppercase currency codes from supported sets.
- `activity_date` must come from the activity timestamp, not report generation time, sync time, or machine-local date.
- Requests where source currency equals base currency are not created.

State transitions:

- `new -> cached_in_run` if evidence already exists for the same report run key.
- `new -> provider_lookup -> resolved` on success.
- `provider_lookup -> failed` when no authoritative evidence is available.

## ExchangeRateEvidence

Purpose: Canonical authority-backed rate evidence used for one conversion date.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_currency` | string | Activity selected currency |
| `base_currency` | string | Report base currency |
| `activity_date` | date | Original source-calendar activity date |
| `rate_date` | date | Actual provider observation date used |
| `authority` | enum | `european_central_bank` or `federal_reserve` |
| `provider_id` | enum | `ecb_exr` or `federal_reserve_h10` |
| `rate_kind` | string | Daily reference rate or noon buying rate |
| `quote_direction` | enum | `source_per_base` or `base_per_source` |
| `rate_value` | decimal | Provider-published rate value with precision preserved |
| `dataset_reference` | string | Non-secret dataset/series identity |

Relationships:

- Resolved from one `RateLookupRequest`.
- Used by one or more same-key `ConvertedActivityAmount` values in the same report run.
- Referenced by `ConversionAuditEntry`.

Validation rules:

- `rate_date` must be equal to or earlier than `activity_date`.
- `rate_value` must be positive and exact-decimal parseable.
- `quote_direction` must be explicit.
- Provider identity must match the selected report base currency.
- Evidence must not be persisted to protected snapshots.

State transitions:

- `mapped -> validated -> applied` on success.
- `mapped -> rejected` when the provider response is inconsistent or non-defensible.

## ConvertedActivityAmount

Purpose: A selected activity monetary amount converted into report base currency before report calculations consume it.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Activity reference |
| `amount_kind` | enum | `unit_price`, `gross_value`, or `fee_amount` |
| `original_currency` | string | Selected activity currency |
| `original_amount` | decimal | Selected exact amount before conversion |
| `report_base_currency` | string | `USD` or `EUR` |
| `converted_amount` | decimal | Amount used by basis/proceeds/gain/loss calculation |
| `exchange_rate_evidence` | `ExchangeRateEvidence` nullable | Present only when currencies differ |
| `conversion_status` | enum | `same_currency` or `converted` |

Relationships:

- Derived from one `SelectedActivityMonetaryContext` amount.
- Uses one `ExchangeRateEvidence` when conversion occurs.
- Feeds existing acquisition, liquidation, fee, and basis calculations.

Validation rules:

- Original and converted currencies must both be explicit.
- Same-currency amounts preserve the original amount exactly.
- Converted amounts are calculated according to canonical quote direction.
- Explicit zero amounts remain zero and do not create fees, proceeds, gains, or losses by conversion.

State transitions:

- `original -> same_currency` when no exchange rate is needed.
- `original -> converted` when canonical evidence is applied.
- `original -> failed` when required evidence is unavailable.

## ConversionAuditEntry

Purpose: Report-visible evidence connecting original activity values to converted values.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Activity reference |
| `asset_label` | string | Rendered asset display label |
| `activity_date` | date | Original source-calendar activity date |
| `source_currency` | string | Selected activity currency |
| `report_base_currency` | string | Selected report base currency |
| `rate_date` | date | Actual provider observation date |
| `rate_authority` | string | ECB or Federal Reserve |
| `rate_kind` | string | Daily reference rate or noon buying rate |
| `rate_value` | decimal | Published rate value |
| `quote_direction` | string | Canonical quote direction |
| `amounts` | `ConvertedActivityAmount[]` | Original and converted unit price, gross value, and fee values as applicable |

Relationships:

- Belongs to one `CapitalGainsReport`.
- References one priced activity that required conversion.
- Uses one `ExchangeRateEvidence`.

Validation rules:

- Required for every converted priced activity.
- Not required for same-currency rows, but same-currency rows must remain distinguishable in report detail tables.
- Must not expose tokens, JWTs, raw protected payloads, or production-mode diagnostic financial values outside the intentional final report content.

State transitions:

- `created -> rendered` when the final report is saved.

## ConversionFailure

Purpose: A report-generation failure caused by non-defensible conversion conditions.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string nullable | Non-secret affected activity reference when known |
| `source_currency` | string nullable | Affected selected activity currency |
| `report_base_currency` | string | Selected report base currency |
| `activity_date` | date nullable | Affected source-calendar date |
| `provider_id` | enum nullable | Provider selected by base currency |
| `reason` | enum | `unsupported_currency`, `missing_rate`, `provider_unavailable`, `malformed_rate`, `ambiguous_quote`, `invalid_activity_currency`, or `authority_mismatch` |
| `safe_message` | string | User-visible non-secret explanation |

Relationships:

- Created by provider mapping, conversion validation, or calculation boundary.
- Returned through runtime report failure handling.
- May contribute redacted diagnostic context.

Validation rules:

- Safe message must exclude Ghostfolio tokens, bearer tokens, reusable token verifiers, raw auth material, and raw protected payloads.
- Production diagnostics redact financial amounts but preserve source currency, base currency, activity date, and non-secret activity reference.
- A conversion failure prevents final report save.

State transitions:

- `raised -> reported` after runtime turns it into a user-visible result.
- `raised -> diagnostic_available` when the existing diagnostic policy allows a report-failure diagnostic.

## CapitalGainsReport

Purpose: Existing calculated report model extended with report base currency and conversion audit evidence.

Fields added or changed:

| Field | Type | Notes |
|-------|------|-------|
| `report_calculation_currency` | string | Selected report base currency, no longer `NOT APPLICABLE` under this feature |
| `conversion_audit_entries` | `ConversionAuditEntry[]` | One per converted priced activity |
| `rate_sources` | `ExchangeRateEvidence[]` or summarized equivalent | Provider/rate details needed to reproduce conversions |

Relationships:

- Produced from one `ReportRequest`.
- Contains zero or more `ConversionAuditEntry` values.
- Supplies Markdown rendering with same-currency versus converted-row status.

Validation rules:

- `report_calculation_currency` must equal the request base currency.
- Successful mixed-currency reports express cost basis, proceeds, gains, losses, and totals in `report_calculation_currency`.
- Converted activity audit details must be complete enough to reproduce conversion from synced activity inputs, selected base currency, provider, rate date, rate value, quote direction, and rounding rules.

State transitions:

- `calculated -> rendered -> saved` on success.
