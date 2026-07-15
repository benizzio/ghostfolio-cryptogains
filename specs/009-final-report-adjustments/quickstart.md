# Quickstart: Final Report Adjustments

This guide validates the final report presentation changes without modifying
financial calculations or empirical fixtures.

## Prerequisites

- Go 1.26.5.
- Repository development tools required by the Makefile quality gates.
- Deterministic report fixtures containing positive, negative, zero, whole,
  high-precision, exact-half, very small, and optional monetary values.
- Fixtures containing high-precision quantities and exchange rates, true and
  false Annex boolean values, zero- and non-zero-priced holding reductions, and
  conversion rows with one, two, and three included amount components.
- No live Ghostfolio, exchange-rate provider, PDF service, browser renderer,
  external PDF binary, or remote font is required.

See `specs/009-final-report-adjustments/contracts/report-rendering.md` for exact
visible contracts and `specs/009-final-report-adjustments/data-model.md` for the
presentation-value boundaries.

## Focused Verification

Run the directly affected report packages first:

```bash
go test ./internal/report/model ./internal/report/calculate ./internal/report/presentation ./internal/report/markdown ./internal/report/pdf ./tests/testutil -count=1
```

Expected result:

- every financial amount and unit price uses two-place HALF UP display
- optional nil values remain blank and rounded negative zero is `0.00`
- source decimals remain unchanged
- quantities and rate values retain canonical precision
- `Full Liquidation Event` uses `Yes` and `No`
- the existing audit currency value remains unchanged before zero-priced
  presentation
- converted entries preserve exact-zero omission and validated canonical order
- duplicate and out-of-order converted kinds are rejected
- PDF uses a fully bold warning operation in the required order
- PDF measurement and drawing produce the same line count for explicit newlines
  and long space-wrapped content
- the isolated PDF finalization seam returns an error without terminating the
  process

Run the report-facing black-box suites:

```bash
go test ./tests/unit ./tests/contract ./tests/integration -count=1
```

Expected result:

- Markdown main output contains the exact fully bold warning once after report
  metadata and before `Gains-And-Losses Summary`
- generated PDF text runs contain the same warning at the same semantic boundary
  and show every warning fragment uses the embedded bold font
- summary, position, activity, liquidation, Annex, and converted monetary values
  agree between formats at two places
- quantities and provider-published rates are unchanged
- zero-priced Annex rows have blank original currency and retain calculation
  currency
- non-zero-priced control rows retain original currency
- one-to-three Converted Amounts entries have exact spacing, semicolons, order,
  and distinct visible-line encodings
  with generated-PDF coordinates proving each entry starts on a later line
- calculated-model assertions retain exact pre-format values and the existing
  audit currency value

## Full Deterministic Suite

Run all maintained deterministic tests:

```bash
make test
```

Expected result:

- package-local, unit, contract, integration, empirical, and tool tests pass
- Markdown and PDF generation preserve existing output bundle shapes
- report rendering and failure paths do not expose secret material
- empirical calculation comparisons pass without fixture changes

## Coverage Gate

Run the canonical coverage gate:

```bash
make coverage
```

Expected result:

- production statement, global line, global branch, per-file line, and per-file
  branch coverage satisfy the repository's 100% requirements
- contract and integration execution contributes to the production package
  coverage profile
- no performance scenario is included in canonical coverage

## Performance Regression

Run the isolated resource-sensitive suite:

```bash
make test-performance
```

Expected result:

- the deterministic fixture contains exactly 10,000 cached activities
- one Markdown generation and one PDF generation are timed independently
- each selected-format generation completes in under two minutes
- presentation formatting does not introduce a lower scale limit for either
  format
- no performance coverage artifact is created

## Changed-Source Quality Gate

Run the changed-source gate against the expected base:

```bash
make quality QUALITY_BASE_REF=origin/main
```

Expected result:

- changed Go source passes `golangci-lint`, `govulncheck`, and `gitleaks`
- `go.mod` and `go.sum` remain unchanged because no dependency is required
- the command exits successfully; if implementation unexpectedly contains no
  source changes, each scanner emits its explicit skip message

The corresponding CI checks are exactly `test / run`, `coverage / run`,
`test-performance / run`, and `quality`.

## Manual Report Verification

Start the application:

```bash
go run ./cmd/ghostfolio-cryptogains
```

Generate Markdown and PDF with the same year, cost-basis method, calculation
currency, and deterministic or development activity history.

Verify the Markdown main report:

- the exact warning is one fully bold standalone line
- it follows `Report Calculation Currency` and precedes
  `Gains-And-Losses Summary`
- every monetary amount and unit price has two fractional digits
- quantities retain their established representation
- the separate Annex file does not repeat the warning

Verify the Markdown Annex:

- `Full Liquidation Event` cells read `Yes` or `No`
- a zero-priced holding reduction has a blank `Original Activity Currency` cell
  and a populated `Calculation Currency` cell
- a non-zero-priced row retains its original activity currency
- each Converted Amounts entry starts after a visible break and follows exact
  colon, arrow, and semicolon spacing
- rate values retain provider precision

Open the generated PDF in a reader that supports selectable text and verify:

- the warning is fully bold, readable, and placed before the summary
- displayed financial, quantity, rate, boolean, and currency values agree with
  Markdown
- each Converted Amounts entry begins on a separate visible line
- long entries may wrap within their own line, but the next entry still starts
  on a new line
- expanded rows do not overlap borders, following rows, or the bottom margin
- every page remains landscape A4 with searchable/selectable text

Manual PDF inspection supplements automated layout-recorder and concrete-PDF
text-run tests for reader-specific visual behavior; the extended project
inspector supplies the automated font and coordinate evidence.

## Calculation Regression Check

Review calculation-level assertions separately from rendered-string assertions:

- exact basis, proceeds, gains, losses, totals, quantities, conversion amounts,
  and rates remain unchanged before rendering
- the existing calculated audit activity-currency value remains unchanged and is
  suppressed only in the qualifying visible presentation row
- summary inclusion continues to use exact zero, so a non-zero value displayed
  as `0.00` remains present
- conversion component omission continues to require exact zero-to-zero values
- zero-priced classification is not inferred from displayed `0.00`

Do not run `make regenerate-empirical-fixtures`. The YAML dataset and generated
oracle fixtures under `testdata/empirical/` are read-only for this feature.

## Security Review

- Confirm generated reports and failures contain no Ghostfolio token, bearer
  material, protected payload bytes, or reusable verifier.
- Confirm controlled Markdown `<br>` and PDF newline delimiters are inserted
  only after dynamic entry components are escaped and sanitized and fixed entry
  syntax is assembled.
- Confirm PDF generation remains in-process and local-only with application
  fonts and no remote service.
- Confirm the constitution-prerequisite PDF finalization error returns through
  normal report error handling before output writing.
- Confirm the existing requested `0600` file mode and reservation/cleanup
  sequence are unchanged.
- Confirm no new dependency, network request, authentication path, telemetry,
  report history, or automatic report re-ingestion was introduced.
