# Quickstart: Report Base Currency Conversion

This document defines validation flows for the report base-currency conversion feature. Automated validation must use deterministic fixtures and mocked official providers. Live provider checks are optional manual validation only.

## Prerequisites

- Go 1.26.3 installed.
- `gocoverageplus` installed for branch and file coverage export: `go install github.com/Fabianexe/gocoverageplus/cmd/gocoverageplus@v1.2.0`
- Existing synced test fixtures containing priced activity in at least three source currencies across at least two report years.
- Optional manual validation network access to `https://data-api.ecb.europa.eu` and `https://www.federalreserve.gov`.

## Automated Verification Flow

1. Run the full automated test suite.

```bash
go test ./... -covermode=atomic -coverprofile=coverage.cov
```

2. Generate the branch and file coverage report required by the constitution.

```bash
gocoverageplus -i coverage.cov -o coverage.xml
```

3. Confirm contract and integration coverage includes these scenarios:

- report selection requires one base currency before generation
- only `USD` and `EUR` are selectable as report base currencies
- same-currency rows bypass exchange-rate lookup
- mixed-currency EUR report uses ECB EXR fixture evidence
- mixed-currency USD report uses Federal Reserve H.10 fixture evidence
- Federal Reserve starred and unstarred quote directions produce expected converted amounts
- weekend or non-publication activity date uses the previous provider observation and discloses both dates
- unsupported or malformed source currency fails before final save
- provider outage without current-run evidence fails before final save
- zero-priced holding reduction does not require exchange-rate lookup solely because explicit zero source fields exist
- generated Markdown replaces `NOT APPLICABLE` with the selected report base currency
- converted priced activities include source currency, report base currency, activity date, rate date, authority, rate kind, rate value, original amount, and converted amount
- conversion failure diagnostics exclude Ghostfolio tokens and redact production-mode financial values
- existing single-currency cases preserve prior monetary results when activity currency equals selected base currency

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

## Fixture Expectations

Project-owned fixtures should include:

- ECB EXR successful same-day observation
- ECB EXR previous available observation
- ECB EXR unsupported or suspended currency response
- Federal Reserve H.10 unstarred currency-units-per-USD observation
- Federal Reserve H.10 starred USD-per-currency-unit observation
- Federal Reserve H.10 `ND` or missing observation
- provider outage after earlier successful conversion in the same report run
- malformed provider payload and malformed decimal value
- mixed-currency activity history with fees and gross values converted together before basis calculation
- explicit zero fee and zero-priced holding reduction cases

These fixtures must allow CI to validate conversion behavior without a live ECB or Federal Reserve dependency.
