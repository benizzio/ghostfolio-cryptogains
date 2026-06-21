# Research: Report Base Currency Conversion

## Research Inputs

- Feature spec: `/specs/007-currency-conversion-strategy/spec.md`
- Source issue: `https://github.com/benizzio/ghostfolio-cryptogains/issues/5`
- Referenced PR discussion: `https://github.com/benizzio/ghostfolio-cryptogains/pull/1#discussion_r3191770497`
- ECB Data Portal API documentation: `https://data.ecb.europa.eu/help/api/overview`, `https://data.ecb.europa.eu/help/api/data`, and `https://data.ecb.europa.eu/help/api/data-examples`
- ECB euro foreign exchange reference rates page: `https://www.ecb.europa.eu/stats/policy_and_exchange_rates/euro_reference_exchange_rates/html/index.en.html`
- Federal Reserve H.10 release and Data Download Program pages: `https://www.federalreserve.gov/releases/h10/default.htm`, `https://www.federalreserve.gov/releases/h10/about.htm`, `https://www.federalreserve.gov/releases/h10/current/default.htm`, `https://www.federalreserve.gov/releases/h10/hist/default.htm`, and `https://www.federalreserve.gov/datadownload/Choose.aspx?rel=H10`
- Federal Reserve Data Download Program help: `https://www.federalreserve.gov/datadownload/help/default.htm`
- OWASP Top 10 current release page: `https://owasp.org/www-project-top-ten/`

## EUR Base Currency Source

Decision: Use the European Central Bank Data Portal `EXR` dataflow for EUR-base conversions.

Rationale: The ECB Data Portal is operated by the European Central Bank and exposes SDMX 2.1 RESTful web services over HTTPS. The documented exchange-rate series key shape for daily euro reference rates is `D.<SOURCE>.EUR.SP00.A`, for example `D.USD.EUR.SP00.A`. The API supports date ranges through `startPeriod` and `endPeriod`, data-only responses, CSV and JSON formats, and history/revision parameters. The ECB euro foreign exchange reference-rates page states that reference rates are usually updated around 16:00 CET every working day except TARGET closing days, are based on the daily concertation procedure between central banks across Europe, and quote all currencies against the euro as base currency.

Alternatives considered: Use downloaded ECB XML/ZIP time-series files, but the Data Portal API provides narrower date and series queries. Use commercial FX APIs, but they are not official ECB sources and would violate the no-unofficial-fallback requirement. Use Ghostfolio prices, but that would not satisfy the issue's official authority requirement.

## EUR Source Coverage And Rate Semantics

Decision: Treat ECB EXR `D.<SOURCE>.EUR.SP00.A` values as `source currency units per 1 EUR`, and convert source amounts into EUR by division.

Rationale: The ECB page explicitly says all listed currencies are quoted against the euro base currency. Current listed currencies include `USD`, `JPY`, `CZK`, `DKK`, `GBP`, `HUF`, `PLN`, `RON`, `SEK`, `CHF`, `ISK`, `NOK`, `TRY`, `AUD`, `BRL`, `CAD`, `CNY`, `HKD`, `IDR`, `ILS`, `INR`, `KRW`, `MXN`, `MYR`, `NZD`, `PHP`, `SGD`, `THB`, and `ZAR`. `RUB` is suspended. Historical coverage is available through ECB time series and the Data Portal, but implementation must verify that the requested source currency and activity date range produce an observation.

Alternatives considered: Invert the quote globally without recording direction, but that would obscure auditability. Use cross-rates between non-EUR currencies from other data sources, but EUR reports can be satisfied directly by ECB EUR reference rates and unsupported currencies should fail.

## USD Base Currency Source

Decision: Use the Federal Reserve Board H.10 foreign exchange rates through the Federal Reserve Data Download Program and H.10 historical country data for USD-base conversions.

Rationale: The H.10 pages are served from `federalreserve.gov`, a U.S. government `.gov` HTTPS host operated by the Board of Governors of the Federal Reserve System. The H.10 about page states that the release contains daily rates of exchange of major currencies against the U.S. dollar, that the data are noon buying rates in New York for cable transfers payable in listed currencies, and that the rates have been certified by the Federal Reserve Bank of New York for customs purposes as required by section 522 of the amended Tariff Act of 1930. The Data Download Program provides H.10 data in downloadable CSV/XML/SDMX forms and documents a direct download URL for automated systems.

Alternatives considered: Use the FRED API H.10 release because FRED is operated by a Federal Reserve Bank, but FRED API use commonly involves an API key and adds a second trust and operational boundary. Use Treasury or commercial datasets, but the issue names the Federal Reserve as the authority. Use the current H.10 HTML page only, but historical report generation needs historical observations and stable programmatic downloads.

## USD Source Coverage And Rate Semantics

Decision: Canonicalize Federal Reserve H.10 observations with explicit quote direction before conversion.

Rationale: The current H.10 release states: `Rates in currency units per U.S. dollar except as noted by an asterisk`, and the footnote says starred rows are `U.S. dollars per currency unit`. Current H.10 country coverage includes Australia, Brazil, Canada, China, Denmark, EMU member countries, Hong Kong, India, Japan, Malaysia, Mexico, New Zealand, Norway, Singapore, South Africa, South Korea, Sri Lanka, Sweden, Switzerland, Taiwan, Thailand, United Kingdom, and Venezuela. H.10 historical country data lists 2000-present, 1990-99, and through-1989 groupings. The adapter must map each currency to an ISO-like code used by stored activity data, identify whether the quote is `source per USD` or `USD per source`, and reject unsupported or ambiguous rows.

Alternatives considered: Assume every H.10 row is foreign units per USD, but starred rows such as EMU/EUR, Australia, New Zealand, and United Kingdom use the opposite direction. Normalize by scraping display text from only the current HTML table, but that is brittle and insufficient for historical dates.

## Unavailable-Date Rule

Decision: For both providers, use the most recent previous provider observation when the activity source-calendar date has no published observation.

Rationale: The spec requires prior-business-date fallback when the official source publishes no rate for the activity date. ECB explicitly skips TARGET closing days. Federal Reserve H.10 is released weekly and contains daily business-week observations; current release rows can contain `ND` for no data. Selecting the previous available observation from the same provider gives a deterministic rule that supports weekends, public holidays, and no-publication days without inventing an unofficial rate.

Alternatives considered: Use the next business date, but the spec requires previous available date. Use interpolation, but that is an unofficial derived rate. Fail on weekends and holidays, but the spec requires fallback when prior official observations exist.

## Revision Behavior

Decision: Regenerated reports use currently published provider observations at generation time and disclose the authority, provider, rate date, rate value, and quote direction used.

Rationale: The feature clarification requires current authorized rates at generation time. ECB Data Portal supports retrieving current production values and has `updatedAfter`/history concepts for changes and revisions. Federal Reserve DDP help says data are available as currently published, while H.10 release archives may not reflect later revisions. Using current provider data avoids persisting stale rates and keeps regenerated reports auditable by their disclosed rate evidence.

Alternatives considered: Persist rate evidence in the encrypted snapshot, but the spec assumes no long-lived cache and requires regenerated reports to use current published rates. Use archived release pages for historical authenticity, but H.10 explicitly warns that past releases are not revised and may not reflect subsequent revisions.

## Anticorruption Layer And Canonical Model

Decision: Build a report exchange-rate anticorruption layer that converts each provider response into canonical `ExchangeRateEvidence` before calculation.

Rationale: The original issue states that obtaining conversion rates involves multiple sources for the same data and must follow a proper anticorruption layer with a canonical model. ECB and Federal Reserve differ in endpoint shape, payload format, quote direction, release cadence, supported currency coverage, and missing-data markers. Canonical evidence protects `internal/report/calculate/`, cost-basis methods, TUI, and Markdown rendering from provider-specific DTOs.

Alternatives considered: Put provider parsing directly inside the calculator, but that would mix IO and financial calculation rules. Create generic support helpers under `internal/support/`, but provider mapping is report-domain-specific and should stay under `internal/report/`. Store provider DTOs in report models, but that leaks external representations across domain boundaries.

## Conversion And Rounding

Decision: Convert selected activity monetary values after single-activity monetary-context selection and before cost-basis/proceeds/gain/loss calculation.

Rationale: Existing report rules already select one complete activity monetary tier and prohibit mixing monetary tiers. Converting after selection preserves those rules while ensuring every cross-activity basis and proceeds calculation consumes one report base currency. Same-currency activities bypass lookup. Zero-priced holding reductions that create no proceeds and no acquisition cost bypass lookup. Division and bounded internal decimal results use the existing 16-decimal round-half-up policy; published rate precision is preserved in audit evidence.

Alternatives considered: Convert during sync normalization, but the constitution prohibits undocumented cross-currency conversion during ingestion/storage and regenerated reports must use current rates. Convert after cost-basis allocation, but that mixes currencies in basis state and produces indefensible gains/losses. Use binary floating point for rates, but the constitution prohibits floating-point financial logic.

## Failure Modes

Decision: Fail report generation before final save when authoritative conversion is unavailable or non-defensible.

Rationale: The spec requires safe failure instead of silent unofficial fallbacks. Failure causes include unsupported source currency, malformed or missing selected activity currency, provider outage without current-run evidence, no prior available official observation, ambiguous quote direction, malformed provider response, non-decimal observation value, and provider response whose currency/date does not match the request. The runtime must keep the user inside the unlocked reporting context and leave no partial cleartext report artifact.

Alternatives considered: Emit partial reports with skipped rows, but that can misstate taxable gains/losses. Use hard-coded rates for missing data, but that is unofficial and unauditable. Save diagnostic cleartext automatically in production, but production diagnostics must follow redaction policy and report diagnostics are separate from final reports.

## Security Review

Decision: Treat this as a fixed-host outbound HTTP integration with no provider authentication and no additional persistence.

Rationale: ECB and Federal Reserve rate requests do not need Ghostfolio tokens. The integration must not include user-controllable provider URLs, which keeps SSRF risk bounded to fixed hosts. Query parameters are derived from validated currency codes and dates. Provider response parsing must enforce expected content type/shape and decimal parsing without using floating point. Diagnostics must redact tokens, bearer JWTs, reusable token verifiers, and production-mode financial values.

Alternatives considered: Let users configure arbitrary provider URLs, but that increases SSRF and integrity risk and violates the official-source requirement. Persist a cache for offline report generation, but that creates invalidation, removal, and protection work not required by the spec.

## Dependency Decision

Decision: Add no third-party dependencies for exchange-rate integration.

Rationale: Go standard library support is sufficient for HTTPS requests, CSV, JSON, XML, ZIP, time parsing, and deterministic tests with `httptest.Server`. Avoiding new dependencies satisfies the constitution's minimal dependency principle and reduces review scope.

Alternatives considered: Add an SDMX client, but it would require maintenance, security, and release-freshness research and is not necessary for the limited EXR/H.10 subset. Add a commercial FX SDK, but it is not official-authority evidence.

## Test Evidence Strategy

Decision: Automated tests use project-owned deterministic provider fixtures and do not depend on live ECB or Federal Reserve availability.

Rationale: Integration tests must be deterministic and CI-friendly. Fixture responses can prove endpoint mapping, quote direction, prior-date fallback, conversion formulas, audit evidence, and failure handling without network flakiness. Manual quickstart validation can optionally exercise live official endpoints.

Alternatives considered: Run live provider calls in CI, but provider availability and published-rate revisions would make tests nondeterministic. Use only unit tests for conversion math, but the feature also changes runtime workflow, report output, and failure cleanup.
