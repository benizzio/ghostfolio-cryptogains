# Quickstart: Report Base Currency Conversion

This document defines validation flows for the report base-currency conversion feature. Default automated validation must use deterministic fixtures and mocked official providers. Live provider checks belong to a separate opt-in external integration test category.

**Bugfix**: 2026-06-21 — BUG-001 Updated final coverage commands to use maintained `dist/coverage` artifacts.

## Prerequisites

- Go 1.26.3 installed.
- `gocoverageplus` installed for branch and file coverage export: `go install github.com/Fabianexe/gocoverageplus/cmd/gocoverageplus@v1.2.0`
- Existing synced test fixtures containing priced activity in at least three source currencies across at least two report years, including deterministic division, multiplication, previous-available-date, and 16-decimal round-half-up conversion cases.
- Optional external integration validation network access to `https://data-api.ecb.europa.eu` and `https://www.federalreserve.gov`.

## Automated Verification Flow

WU22 implementation note: WU01 through WU21 are recorded as complete and targeted automated tests have passed for the implemented report base-currency conversion work. This note does not claim full final coverage; final full-suite and coverage-gate evidence remains part of the WU23 verification commands.

Superseded by BUG-001: ~~`go test ./... -covermode=atomic -coverprofile=coverage.cov` followed by `gocoverageplus -i coverage.cov -o coverage.xml`~~ wrote generated coverage artifacts to the repository root.

1. Run the maintained coverage gate, which runs the covered package set and writes generated coverage artifacts under `dist/coverage`.

```bash
make coverage
```

2. If running the coverage commands manually instead of `make coverage`, use the maintained generated-output paths.

```bash
mkdir -p dist/coverage
PRODUCTION_PACKAGES=$(go run ./tools/coverpkg -go go ./cmd/... ./internal/...)
go test ./cmd/... ./internal/... ./tests/contract ./tests/empirical ./tests/empirical/fixture ./tests/integration ./tests/unit -covermode=atomic -coverpkg="${PRODUCTION_PACKAGES}" -coverprofile=dist/coverage/coverage.out
gocoverageplus -i dist/coverage/coverage.out -o dist/coverage/coverage.xml
go run ./tools/coveragegate -profile dist/coverage/coverage.out -cobertura dist/coverage/coverage.xml
```

3. Confirm contract and integration coverage includes these scenarios:

- report selection requires one base currency before generation
- only `USD` and `EUR` are selectable as report base currencies
- same-currency rows bypass exchange-rate lookup
- mixed-currency EUR report uses ECB EXR fixture evidence
- mixed-currency USD report uses Federal Reserve H.10 fixture evidence
- Federal Reserve starred and unstarred quote directions produce expected converted amounts
- deterministic conversion fixtures cover at least one division result requiring 16-decimal round-half-up internal bounding and at least one multiplication quote-direction conversion
- the in-memory TUI-session rate cache can reuse evidence across multiple report runs and different security-token unlocks without persisting evidence
- the 10,000-activity responsiveness fixture accepts report base-currency selection and generation confirmation before delayed provider fixture responses are released
- provider lookup requests are bounded by unique `(base currency, source currency, activity source-calendar date)` keys rather than monetary field count
- weekend or non-publication activity date uses the previous provider observation and discloses both dates
- unsupported or malformed source currency fails before final save
- provider outage without current TUI-session evidence fails before final save
- zero-priced holding reduction does not require exchange-rate lookup solely because explicit zero source fields exist
- generated Markdown replaces `NOT APPLICABLE` with the selected report base currency
- converted priced activities include source currency, report base currency, activity date, rate date, authority, rate kind, rate value, original amount, and converted amount
- conversion failure diagnostics exclude Ghostfolio tokens and redact production-mode financial values
- existing single-currency cases preserve prior monetary results when activity currency equals selected base currency

Post-implementation validation notes:

- default automated validation uses deterministic provider fixtures and test-only fixture endpoints instead of live ECB or Federal Reserve traffic
- external integration checks are opt-in and must be enabled through the dedicated external integration environment variable used by the test suite
- external integration checks are limited to fixed historical observations for the official ECB and Federal Reserve HTTP clients
- the implemented in-memory rate evidence cache is process-local to the TUI session and has no persisted rate evidence
- no user-controlled provider URL is accepted by the runtime report-generation path

## Manual TUI Verification Flow

1. Launch the application.

```bash
go run ./cmd/ghostfolio-cryptogains
```

2. Enter the `Sync and Reports` context and unlock or sync a dataset that contains reportable priced activity in more than one currency.

Expected result:

- token entry is masked
- no report action is available before synced report years exist
- the unlocked context shows reportable years without exposing protected raw payload data

3. Start report generation.

Expected result:

- the selection screen shows year, cost basis method, and report base-currency choices
- only `USD` and `EUR` are available as base currencies
- `Generate Report` cannot start without a selected base currency

4. Generate one report with `EUR` as base currency.

Expected result:

- calculation uses ECB-backed EUR conversion for non-EUR priced rows
- same-EUR rows are not converted
- the saved Markdown header shows `Report Calculation Currency: EUR`
- converted rows include conversion audit details with ECB authority and rate dates

5. Generate the same year and method with `USD` as base currency.

Expected result:

- calculation uses Federal Reserve-backed USD conversion for non-USD priced rows
- same-USD rows are not converted
- the saved Markdown header shows `Report Calculation Currency: USD`
- converted rows include conversion audit details with Federal Reserve authority and rate dates
- the USD and EUR reports are separate saved files

6. Run a fixture or development setup where one required source currency is unsupported by the selected provider.

Expected result:

- generation fails before final report save
- no partial cleartext report file remains
- the user remains in the unlocked reporting context
- the failure message identifies source currency, report base currency, and activity date without exposing token material

## External Integration Verification Flow

External integration tests are optional and must be explicitly enabled by the developer or by a CI job dedicated to live provider verification.

The opt-in live checks target only the fixed production hosts `https://data-api.ecb.europa.eu` and `https://www.federalreserve.gov`. Default automated tests must continue to use test-only fixture endpoints and must not depend on network access.

Expected external integration coverage:

- one fixed historical ECB EXR observation through the live ECB HTTP client
- one fixed historical Federal Reserve H.10/Data Download Program observation through the live Federal Reserve HTTP client
- committed expected values for source currency, base currency, activity date, rate date, rate value, quote direction, and provider identity
- no report calculation, no TUI workflow, no token handling, and no broad provider date or currency sweeps

## Fixed-Host Provider Security Review Evidence

Security evidence recorded for the implementation:

- production ECB requests are constrained to `https://data-api.ecb.europa.eu`
- production Federal Reserve requests are constrained to `https://www.federalreserve.gov`
- provider requests do not include Ghostfolio tokens, bearer JWTs, reusable token verifiers, or provider authentication secrets
- query inputs are derived from supported report base currencies, validated source currency codes, and activity dates
- provider response parsing rejects malformed, unsupported, unavailable, ambiguous, or non-decimal rate evidence before final report save
- conversion diagnostics exclude token material and redact production-mode financial values while preserving source currency, report base currency, activity date, and provider category
- exchange-rate evidence is cached only in memory for the current TUI process and is not persisted to setup files, protected snapshots, or report metadata outside the generated cleartext Markdown report
- no persisted rate evidence was introduced, so there is no persisted-rate removal, retention, encryption, or invalidation evidence to review

OWASP Top 10 review coverage for this feature scope:

- Broken Access Control: no new privileged local or remote access path was added for provider lookup; report generation remains inside the existing unlocked reporting workflow.
- Cryptographic Failures: no provider secrets are introduced; existing token-derived protected snapshot handling is unchanged; generated reports remain cleartext output under the existing report-output contract.
- Injection: provider URL construction is not user-controlled; currency and date inputs are validated before request construction; Markdown audit fields are derived from canonical evidence and existing report rendering rules.
- Insecure Design: unsupported, unavailable, malformed, ambiguous, or non-authoritative rate evidence fails before final save instead of falling back to unofficial or hard-coded rates.
- Security Misconfiguration: live provider access is limited to fixed HTTPS production hosts; default automated tests use fixture endpoints and do not require live network access.
- Vulnerable and Outdated Components: no new third-party dependency was added for provider integration.
- Identification and Authentication Failures: provider requests do not reuse Ghostfolio authentication material and do not require provider authentication.
- Software and Data Integrity Failures: rate evidence records authority, provider, rate date, quote direction, and rate value; provider payloads are canonicalized before report calculation consumes them.
- Security Logging and Monitoring Failures: failure diagnostics preserve non-secret context needed for troubleshooting and redact token material and production-mode financial values.
- Server-Side Request Forgery: runtime provider selection has no user-controlled provider URL and is limited to fixed official production hosts.

## Fixture Expectations

Project-owned fixtures should include:

- ECB EXR successful same-day observation
- ECB EXR previous available observation
- ECB EXR unsupported or suspended currency response
- Federal Reserve H.10 unstarred currency-units-per-USD observation
- Federal Reserve H.10 starred USD-per-currency-unit observation
- Federal Reserve H.10 `ND` or missing observation
- provider outage after earlier successful conversion in the same TUI session
- malformed provider payload and malformed decimal value
- mixed-currency activity history with fees and gross values converted together before basis calculation
- explicit zero fee and zero-priced holding reduction cases
- deterministic conversion values that prove 16-decimal round-half-up division bounding
- 10,000-activity responsiveness fixture with repeated rate keys and delayed provider responses

These fixtures must allow CI to validate conversion behavior without a live ECB or Federal Reserve dependency.

External integration fixtures must commit the one fixed historical observation expected for each live provider client endpoint.
