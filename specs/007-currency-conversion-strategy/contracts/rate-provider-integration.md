# Contract: Official Rate Provider Integration

## Scope

This contract defines the external exchange-rate integration boundary for report base-currency conversion. It covers official provider selection, canonical rate evidence, provider failure behavior, deterministic test doubles, and opt-in external integration tests.

## Provider Selection

| Report base currency | Provider ID | Authority | Source |
|----------------------|-------------|-----------|--------|
| `EUR` | `ecb_exr` | European Central Bank | ECB Data Portal `EXR` daily reference-rate data |
| `USD` | `federal_reserve_h10` | Federal Reserve | Federal Reserve Board H.10/Data Download Program data |

Rules:

- Provider selection is internal to the currency integration service and is derived only from the validated report base currency.
- Provider hosts are fixed implementation constants and must use HTTPS.
- User input must not alter provider host, scheme, or authority relationship.
- Ghostfolio tokens, bearer JWTs, and protected snapshot payloads must never be sent to rate providers.
- No unofficial fallback provider is allowed.

## Canonical Lookup Contract

Input:

```text
source_currency: uppercase currency code from selected activity monetary context
base_currency: USD or EUR
activity_date: source-calendar date from the activity timestamp and its stored offset
```

Successful output:

```text
source_currency
base_currency
activity_date
rate_date
authority
provider_id
rate_kind
quote_direction
rate_value
dataset_reference
```

Rules:

- The public lookup request must not include `provider_id`; provider identity is selected internally from `base_currency`.
- `rate_date` must be the activity date or the most recent previous provider date with an available observation.
- `rate_value` must be a positive exact decimal parsed without floating point.
- `quote_direction` must be either `source_per_base` or `base_per_source`.
- The currency integration service must reject responses that do not match the requested source currency, base currency, or date range.
- The currency integration service must reject missing observations, `ND` observations, malformed decimals, and ambiguous quote direction.

## ECB EXR Contract

Endpoint family:

```text
https://data-api.ecb.europa.eu/service/data/EXR/D.<SOURCE>.EUR.SP00.A
```

Required query behavior:

```text
startPeriod=<lookback date>
endPeriod=<activity date>
detail=dataonly
format=<csvdata or jsondata>
```

Rules:

- The implementation may choose CSV or JSON but must parse into the same canonical model.
- `D.<SOURCE>.EUR.SP00.A` values are `source currency units per 1 EUR`.
- Source-to-EUR conversion uses division by the rate value.
- If source currency is `EUR`, the provider is not called.
- If no observation exists on the activity date, the adapter chooses the latest observation before that date from the same series.
- `RUB` or any suspended, unsupported, or absent source currency fails unless the provider later supplies authoritative current production observations.

## Federal Reserve H.10 Contract

Endpoint family:

```text
https://www.federalreserve.gov/datadownload/...
https://www.federalreserve.gov/releases/h10/...
```

Required data behavior:

- Use H.10/Data Download Program or H.10 historical country data as currently published by the Federal Reserve Board.
- Resolve observations for the requested source currency on or before the activity date.
- Map Federal Reserve country/monetary-unit labels to stored currency codes explicitly.
- Preserve H.10 quote direction in canonical evidence.

Rules:

- H.10 unstarred rows are currency units per USD and source-to-USD conversion uses division.
- H.10 starred rows are USD per currency unit and source-to-USD conversion uses multiplication.
- If source currency is `USD`, the provider is not called.
- `ND` values and absent observations are not valid rates.
- Past release archive pages are not the regenerated-report source of truth when they differ from current DDP/historical data.

## In-Memory TUI-Session Cache

Rules:

- The application may cache canonical rate evidence only in memory while the TUI session is executing.
- The cache may be reused across multiple report runs, years, cost-basis methods, and different security-token unlocks in the same process.
- Public cache identity is `(source_currency, base_currency, activity_date)`; the implementation may include internally selected provider identity in a private key.
- The cache must not include security-token values, token-derived verifiers, Ghostfolio tokens, bearer JWTs, or protected payload data in keys or values.
- Cached evidence must not be written to protected snapshots, setup files, app-data caches, or temp files.
- Final report output may disclose the evidence because the report is the intentional cleartext audit artifact.

## Failure Contract

Provider lookup fails the report before final save when any condition applies:

- Unsupported source currency for selected provider.
- Malformed or missing selected activity currency.
- Provider request fails and no current TUI-session evidence exists for the key.
- Provider returns non-success HTTP status.
- Provider response cannot be parsed.
- Provider response contains no current or prior available observation.
- Observation value is `ND`, empty, non-positive, or not exact-decimal parseable.
- Quote direction is ambiguous.
- Response authority, provider, source currency, base currency, or date range does not match the lookup request.

User-visible failure messages must identify source currency, report base currency, and activity date when known, and must exclude secrets.

## Test Double Contract

Default automated contract, integration, and unit tests must use local fixtures or `httptest.Server` implementations that can simulate:

- ECB same-day observation.
- ECB previous available observation for a weekend or TARGET closing day.
- ECB unsupported or suspended currency.
- Federal Reserve unstarred `source_per_base` row.
- Federal Reserve starred `base_per_source` row.
- Federal Reserve `ND` or missing observation.
- Provider outage after some successful conversions.
- Malformed decimal and malformed payload.

Live ECB or Federal Reserve calls must not be required for default CI or coverage-gate execution.

## External Integration Test Contract

External integration tests are a separate opt-in category for validating that official provider HTTP clients still match live provider API behavior.

Rules:

- Tests target only the HTTP client layer under `internal/integration/currency/`; they do not run report calculation or TUI workflows.
- Each unique client endpoint uses exactly one fixed historical observation whose expected provider response values are committed in the project.
- Tests assert source currency, base currency, activity date, rate date, rate value, quote direction, and provider identity for the fixed observation.
- Tests must avoid broad date ranges, randomized dates, loops over currency sets, or repeated calls that create unnecessary provider load.
- Tests must be skipped unless explicitly enabled by the developer or CI job dedicated to external integration verification.
