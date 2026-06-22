# Implementation Plan: Report Base Currency Conversion

**Branch**: `[007-currency-conversion-strategy]` | **Date**: 2026-06-20 | **Spec**: `/specs/007-currency-conversion-strategy/spec.md`

**Input**: Feature specification from `/specs/007-currency-conversion-strategy/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

**Bugfix**: 2026-06-21 — BUG-001 Updated from bugfix patch

**Bugfix**: 2026-06-23 — BUG-002 Updated from bugfix patch

## Summary

Add a required report base-currency selection to yearly capital gains report generation, limited to `USD` and `EUR`. The report pipeline will keep the existing single-activity monetary-context selection rules, then convert every priced activity whose selected currency differs from the report base currency before cost-basis, proceeds, gain/loss, and report-total calculations consume the values. EUR-base conversions use the ECB Data Portal `EXR` daily reference-rate data. USD-base conversions use the Federal Reserve Board H.10/Data Download Program data. Provider-specific payloads, quote directions, provider selection, and canonicalization stay behind `internal/integration/currency/`, which exposes a public rate lookup service consumed by the report calculator and disclosed in the Markdown report through canonical rate evidence.

## Technical Context

**Language/Version**: Go 1.26.3

**Primary Dependencies**: Existing dependencies only: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/Fabianexe/gocoverageplus`, and Go standard library packages including `net/http`, `encoding/csv`, `encoding/json`, `encoding/xml`, `archive/zip`, `time`, and `context`.

**Storage**: No new exchange-rate persistence. Existing protected snapshots remain token-derived encrypted local data. Canonical rate evidence may be held in an in-memory TUI-session cache that survives multiple report runs and different security-token unlocks in the same process. The cache is not written to disk and is disclosed only through the generated cleartext Markdown report saved under the existing report-output contract.

**Testing**: Go `testing` with contract, integration, unit, and external integration suites. Default contract, integration, and unit tests cover provider behavior with deterministic `httptest.Server` fixtures and golden response snippets, not live ECB or Federal Reserve calls. External integration tests are a separate opt-in category that targets only official-provider HTTP clients with one fixed historical observation per unique client endpoint and committed expected values. Coverage remains enforced with ~~`go test ./... -covermode=atomic -coverprofile=coverage.cov` and `gocoverageplus`/repository coverage gates~~ `make coverage`, which writes `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` and runs the repository coverage gate; the root-level coverage command is superseded by BUG-001 because it writes generated artifacts outside `dist/coverage`.

**Federal Reserve DDP Compatibility**: USD live-provider access uses the Data Download Program `Output.aspx` direct CSV package endpoint for H.10 daily rates, not the interactive DDP landing page. Federal Reserve deterministic fixtures and mapper tests must mirror the live `layout=seriesrow` package CSV shape, including metadata such as `Descriptions:`, `Unit:`, `Multiplier:`, `Currency:`, `Unique Identifier:`, and `Series Name:` before date observations.

**Empirical Dataset**: Existing `testdata/empirical/` synthetic empirical dataset and oracle fixtures remain read-only. Existing empirical tests continue to validate single-currency cost-basis behavior and are not repurposed for exchange-rate integration.

**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access and outbound HTTPS access to fixed official rate-provider hosts during manual/live report generation.

**Project Type**: Single-module Go terminal UI application.

**Performance Goals**: Keep the Bubble Tea UI responsive during provider lookup and calculation; avoid per-monetary-field network requests by resolving rates per `(base currency, source currency, activity source-calendar date)` in one in-memory TUI-session cache; preserve the existing report scale target of up to 10,000 cached activities.

**Constraints**: No floating-point financial logic; all amounts and rates use exact decimals with explicit currency identity. Conversion occurs only after selected activity monetary context selection and before cost-basis integration. Same-currency rows do not request rates. Zero-priced holding reductions that create no proceeds and no acquisition cost do not request rates. Missing, unsupported, unavailable, malformed, or non-authoritative rate evidence fails the report before final save. Provider endpoints are fixed official HTTPS origins; no user-supplied rate-provider URLs are accepted. Failure diagnostics redact tokens and production-mode financial values while preserving non-secret activity reference, source currency, report base currency, and activity date.

**Scale/Scope**: Two report base currencies (`USD`, `EUR`), one canonical exchange-rate evidence model, two initial provider adapters (ECB EXR and Federal Reserve H.10/Data Download Program), existing five cost-basis methods, Markdown output audit details for every converted priced activity, and no changes to Ghostfolio sync contracts except consuming the already stored selected activity currency fields.

**Sync Currency Identity Traceability**: This feature relies on the existing sync contract from `specs/003-store-activity-data/spec.md` FR-018 and `specs/003-store-activity-data/contracts/ghostfolio-sync.md`. Protected `internal/sync/model.ActivityRecord` values already preserve order-tier, asset-profile-tier, and base-tier currency identity as `OrderCurrency`, `AssetProfileCurrency`, and `BaseCurrency` with their same-tier monetary fields. Report calculation already selects one tier as `SelectedCurrencyContext`; conversion consumes that selected currency identity and does not require a protected snapshot schema change or sync persistence migration.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: No new persisted provider cache is introduced. Ghostfolio tokens and JWTs remain runtime-only. Official rate-provider requests do not use Ghostfolio secrets. Provider hosts are fixed HTTPS origins to avoid user-controlled outbound destinations. The most recent published OWASP Top 10 review scope is OWASP Top 10 2025 and covers broken access control, cryptographic failure, injection into provider URLs or rendered Markdown, insecure design, security misconfiguration, vulnerable components, authentication/session token leakage, integrity failure in rate data handling, logging/diagnostic leakage, and server-side request forgery from dynamic provider URLs.
- [x] Precision: Monetary amounts, quantities, exchange rates, converted amounts, cost basis, proceeds, gains, and losses use `apd/v3` exact decimals. Every financial value carries source currency before conversion and report base currency after conversion. Conversion boundary, quote-direction handling, prior-business-date fallback, and 16-decimal round-half-up internal bounding are defined in this plan and `research.md`.
- [x] Testing: Integration-first coverage will verify TUI report base-currency selection, runtime generation, currency integration anticorruption behavior with mocked provider responses, Markdown audit sections, failure handling, and no-partial-file cleanup. Targeted unit tests are justified for quote-direction formulas, prior available date selection, canonical model validation, and redaction-sensitive failure shaping. External integration tests are limited to official-provider HTTP clients with one committed historical observation per unique endpoint. Coverage gates remain mandatory.
- [x] Empirical financial validation: Existing empirical financial tests under `tests/empirical/` and data under `testdata/empirical/` remain read-only and single-currency. New conversion coverage uses project-owned contract, integration, external integration, and unit fixtures.
- [x] Dependencies and external integrations: No new third-party dependency is planned. ECB Data Portal and Federal Reserve Board H.10/Data Download Program are new external HTTP integrations and are documented in `research.md` with authority relationship, supported currencies, historical coverage, unavailable-date behavior, revision behavior, failure modes, and test evidence strategy.
- [x] Architecture: Report-domain conversion, cost-basis, proceeds, gain/loss, audit, and rendering rules live under `internal/report/`. Provider IO, provider DTOs, provider selection, quote-direction mapping, and canonicalization live under `internal/integration/currency/`, exposed through a public rate lookup service. `internal/report/calculate/` calls that service and consumes canonical rate evidence rather than provider DTOs. TUI only captures the report base-currency choice. Runtime coordinates dependencies without embedding provider integration details.

## Project Structure

### Documentation (this feature)

```text
specs/007-currency-conversion-strategy/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── markdown-report.md
│   ├── rate-provider-integration.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── app/
│   └── runtime/              # Compose report service, currency integration service, diagnostics, and output lifecycle
├── integration/
│   └── currency/             # Rate lookup service, provider clients, anticorruption mapping, canonical rate evidence, session cache
├── report/
│   ├── model/                # Report request base currency, converted amounts, audit entries, validation
│   ├── calculate/            # Conversion boundary before basis/proceeds/gain/loss replay
│   ├── basis/                # Existing cost-basis methods consume converted report-base amounts
│   ├── markdown/             # Report currency header and conversion audit rendering
│   └── output/               # Existing final Markdown write and cleanup rules
└── tui/
    ├── flow/                 # Base-currency selection state and generation request construction
    └── screen/               # Selection, busy, and result copy including report base currency

tests/
├── contract/                 # TUI and Markdown contract tests
├── integration/              # Runtime report generation with mocked rate providers
├── externalintegration/      # Opt-in live HTTP client checks with one fixed historical record per endpoint
├── unit/                     # Canonical rate, conversion math, provider mapping, and failure classification tests
└── empirical/                # Existing read-only single-currency empirical evidence
```

**Structure Decision**: Keep the feature inside the existing single Go module. Use a new `internal/integration/currency/` layer for provider-specific HTTP, provider DTOs, provider selection from report base currency, quote-direction mapping, canonical rate evidence, and the in-memory TUI-session rate cache. Keep report calculation orchestration in `internal/report/calculate/`, report-owned request/audit/output models in `internal/report/model/`, and user interaction in `internal/tui/`. Do not move provider DTOs or network details into TUI, runtime diagnostics, cost-basis state implementations, or report models.

## Official Rate Source Decisions

### EUR Report Base Currency

- Source: European Central Bank Data Portal `EXR` dataflow through `https://data-api.ecb.europa.eu/service/data/EXR/...`.
- Authority relationship: ECB-operated Data Portal and ECB euro foreign exchange reference rates.
- Rate kind: Daily euro foreign exchange reference rate.
- Quote direction: Units of source currency per 1 EUR for `D.<SOURCE>.EUR.SP00.A`.
- Source-to-base formula: when source is not `EUR`, `converted_eur = source_amount / quote`; when source is `EUR`, no lookup and no conversion.
- Unavailable date behavior: use the most recent previous available ECB observation and disclose both activity date and rate date.
- Revision behavior: use the currently published ECB production observation when no same-key evidence is defensibly available in the current TUI-session cache, and disclose the rate date and value.

### USD Report Base Currency

- Source: Federal Reserve Board H.10 foreign exchange rates through the Federal Reserve Data Download Program and H.10 historical country data.
- Direct download URL shape: `https://www.federalreserve.gov/datadownload/Output.aspx?rel=H10&series=60f32914ab61dfab590e0e470153e3ae&lastobs=&from=<YYYY-MM-DD>&to=<YYYY-MM-DD>&filetype=csv&label=include&layout=seriesrow&type=package`.
- Authority relationship: Board of Governors `.gov` release; H.10 rates are certified by the Federal Reserve Bank of New York for customs purposes.
- Rate kind: Daily noon buying rate in New York for cable transfers payable in listed currencies.
- Quote direction: H.10 publishes most rates as currency units per USD, with starred rows as USD per currency unit. The provider adapter must map the quote direction explicitly into canonical evidence.
- Source-to-base formula: for `source per USD`, `converted_usd = source_amount / quote`; for `USD per source`, `converted_usd = source_amount * quote`; when source is `USD`, no lookup and no conversion.
- Unavailable date behavior: use the most recent previous available H.10 observation and disclose both activity date and rate date.
- Revision behavior: use the currently published H.10/DDP data when no same-key evidence is defensibly available in the current TUI-session cache. Past release archive pages are not the source of truth for regenerated reports because they may not reflect subsequent revisions.

## Supported Currency Coverage

- Same-currency activity rows are always supported without provider lookup when the selected activity currency equals the report base currency.
- EUR-base provider coverage is limited to ECB EXR daily reference-rate source currency series that use `D.<SOURCE>.EUR.SP00.A`. The initial supported source currency set is `AUD`, `BRL`, `CAD`, `CHF`, `CNY`, `CZK`, `DKK`, `GBP`, `HKD`, `HUF`, `IDR`, `ILS`, `INR`, `ISK`, `JPY`, `KRW`, `MXN`, `MYR`, `NOK`, `NZD`, `PHP`, `PLN`, `RON`, `SEK`, `SGD`, `THB`, `TRY`, `USD`, and `ZAR`. `RUB` and any other suspended, absent, malformed, or unmapped ECB source currency fail as unsupported for this feature.
- USD-base provider coverage is limited to Federal Reserve H.10 rows with an unambiguous ISO-like stored currency mapping and quote direction. The initial supported source currency set is `AUD`, `BRL`, `CAD`, `CHF`, `CNY`, `DKK`, `EUR`, `GBP`, `HKD`, `INR`, `JPY`, `KRW`, `LKR`, `MXN`, `MYR`, `NOK`, `NZD`, `SEK`, `SGD`, `THB`, `TWD`, and `ZAR`. Venezuela rows and any other redenomination-sensitive, absent, malformed, or unmapped H.10 rows fail as unsupported until a date-bounded currency-code mapping is explicitly planned.
- Historical coverage is observation-based. A source currency in the supported set still fails for a specific report when the selected provider cannot supply an activity-date observation or a previous available observation for the required date.
- Unsupported, ambiguous, suspended, malformed, or missing selected activity currency values fail report generation before final save and are reported through the non-secret failure path.

## Conversion Boundary And Rounding

1. Normalize and order synced activities using existing sync rules.
2. For each reportable activity, select exactly one activity monetary context using existing `order -> asset_profile -> base` priority and same-tier completeness rules.
3. If the activity is an explained zero-priced holding reduction with no proceeds and no acquisition cost, skip exchange-rate lookup.
4. If the selected activity currency equals the report base currency, preserve selected monetary values and mark the row as same-currency.
5. If the selected activity currency differs from the report base currency, resolve canonical rate evidence by source currency, report base currency, and source-calendar activity date.
6. In the report calculation tier, convert every selected monetary value that can affect cost basis, proceeds, fees, gains, losses, or totals before it enters basis state or liquidation calculations.
7. Use the existing 16-decimal round-half-up policy only when division or another bounded internal decimal result is required. Preserve provider-published rate precision in audit evidence.

## Performance Validation

The 10,000-activity responsiveness fixture validates the scale target with deterministic provider fixtures and no live-provider dependency.

Required validation:

- Base-currency selection and generation confirmation do not perform provider lookup.
- Provider lookup and report calculation run through the asynchronous report-generation workflow and do not block TUI input or rendering while provider responses are delayed.
- Provider lookup requests are bounded by unique `(base currency, source currency, activity source-calendar date)` keys, not by monetary field count.
- Same-currency rows and zero-priced no-cost holding reductions create no provider lookup requests.
- Report calculation succeeds for 10,000 cached activities using deterministic provider evidence.

## Integration Anticorruption Layer

Provider adapters must expose a canonical currency integration application service that returns rate evidence independent of provider payload shape. The public lookup request includes source currency, base currency, and activity date. The canonical model includes source currency, base currency, activity date, rate date, authority, provider, rate kind, quote direction, rate value, and source URL or dataset identity. ECB and Federal Reserve DTOs do not cross into `internal/report/calculate/`, `internal/report/basis/`, TUI, Markdown rendering, or report-owned models.

The issue requirement for scalability is handled by keeping provider selection behind the currency integration service's base-currency registry. Adding a future base currency should require adding one provider adapter and registry entry, not changing cost-basis methods or report rendering logic beyond supported currency labels.

## Failure Handling

- Unsupported or malformed selected activity currency fails report generation before final save.
- Missing official rate evidence for a required `(source currency, base currency, activity date)` fails before final save.
- Provider unavailability fails before final save unless the required evidence is already present in the current in-memory TUI-session cache.
- No unofficial market-data, community, Ghostfolio price, or hard-coded fallback rates are used.
- Failure messages identify non-secret activity reference, source currency, base currency, activity date, and provider category.
- Production diagnostics redact financial values and exclude Ghostfolio tokens, bearer tokens, reusable verifiers, and raw authentication material.
- If conversion or rendering fails before final report save, no partial cleartext report artifact remains.

## Testing Strategy

- Contract tests verify that report selection requires one base currency, only `USD` and `EUR` are selectable, and generated Markdown replaces `NOT APPLICABLE` with the selected base currency.
- Integration tests run report generation with mocked ECB and Federal Reserve provider responses behind the currency integration service, including same-currency rows, converted rows, mixed source currencies, prior-business-date fallback, provider outage, unsupported currency, malformed currency, and no-partial-file cleanup.
- Unit tests cover ECB EXR mapping, Federal Reserve H.10 quote-direction mapping, canonical rate evidence validation, conversion formulas, 16-decimal division bounding, in-memory TUI-session rate reuse, and redaction-safe failure construction.
- External integration tests directly exercise each official-provider HTTP client endpoint with one fixed historical observation and committed expected rate data, avoiding repeated live-provider load and avoiding report-domain setup.
- Federal Reserve external integration tests must call the DDP `Output.aspx` direct CSV package endpoint and commit expected values from that same current DDP package source; BUG-002 updates the EUR `2024-01-05` expectation to `1.0957`.
- Regression tests verify existing single-currency report cases produce the same monetary results when selected activity currency equals the chosen report base currency.
- Performance validation uses the 10,000-activity responsiveness fixture to assert asynchronous TUI behavior, bounded provider lookups by unique rate key, no per-monetary-field network requests, and successful calculation at scale with deterministic provider fixtures.
- Final coverage-gate validation uses the maintained repository output paths `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`; root-level `coverage.cov` and `coverage.xml` are not valid final verification artifacts for this feature.
- Existing empirical tests remain unchanged and continue to guard cost-basis methods without conversion.

## Complexity Tracking

No constitution violations are planned.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
