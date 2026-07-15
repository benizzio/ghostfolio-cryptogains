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
  conversion rows covering zero, one, two, and three entries and all eight
  FR-019 canonical subsequences.
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
- quantities follow the FR-009 fixed-point baseline, and normalized rates such
  as provider spellings `1.094600` and `1.0900` render as `1.0946` and `1.09`
- FR-004a adjusted-exponent and precision guards accept their inclusive bounds
  at the formatter seam and reject out-of-domain Markdown and PDF attempts
  without visible or saved output
- `Full Liquidation Event` uses `Yes` and `No`
- the existing audit currency value remains unchanged before zero-priced
  presentation
- converted entries preserve exact-zero omission and received order without new
  duplicate-kind or supported-kind order validation
- PDF uses a fully bold warning operation in the required order
- PDF measurement and drawing produce the same line count for explicit newlines
  and long space-wrapped content
- the isolated PDF finalization seam returns an error without terminating the
  process, invoking output writing, or invoking the opener
- a row that fits only on a fresh PDF page advances before drawing and remains
  whole, while a row exceeding fresh-page capacity returns a layout error and
  finalizes no PDF

Run the report-facing black-box suites:

```bash
go test ./tests/unit ./tests/contract ./tests/integration -count=1
```

Expected result:

- Markdown main output contains the exact fully bold warning once after report
  metadata and before `Gains-And-Losses Summary`
- generated PDF text runs contain the same warning at the same semantic boundary
  and show every warning fragment uses the embedded bold font
- every Financial Presentation Acceptance Matrix field/vector/output combination
  has the required two-place semantic value in generated Markdown and PDF
- quantities equal their FR-009 canonical baseline, and rates equal normalized
  AUD-002 evidence with metadata unchanged
- zero-priced Annex rows have blank original currency and retain calculation
  currency
- non-zero-priced control rows retain original currency
- a positive source price such as `0.004` retains original currency even though
  it displays as `0.00`; all-missing, mixed missing-and-zero, and explicit-zero
  report-level compatibility cases are classified without manufacturing values,
  while contradictory non-zero cases are not misclassified and sync admission
  remains unchanged
- zero-to-three Converted Amounts entries cover all eight canonical subsequences
  with exact spacing, semicolons, order, and distinct visible-line encodings
  with generated-PDF coordinates proving each entry starts on a later line
- calculated-model assertions retain exact pre-format values and the existing
  audit currency value, with population numerator and denominator counts reported

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
- Markdown second-file and PDF save failures remove only current-attempt files,
  preserve pre-existing colliding sentinel files, and report no partial success
- opener failure after complete save retains all files and reports a warning

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

- the deterministic fixture contains exactly 10,000 quantity-`1` priced
  activities across two assets, 3,334 USD, 3,333 EUR, and 3,333 GBP rows
- the request is 2025/HIFO/USD at `2026-05-21T10:00:00Z`, and the local rate
  service supplies exact `1.1` without network access
- exactly 6,666 conversion rows contain all three converted entries in both
  formats, and the PDF audit spans continuation pages without clipping or loss
- one Markdown generation and one PDF generation are timed independently from
  immediately before generation through save, bundle validation, and opener
  return; fixture setup and output inspection remain outside each interval
- each selected-format generation completes in strictly under two minutes and
  records its format and elapsed duration separately
- `test-performance / run` records the Ubuntu runner image/version, architecture,
  available CPU count, and Go version used for authoritative evidence
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
- any unexpected dependency-file change, executable prerequisite, network path,
  or inability to produce required evidence is a DEP-001 stop signal; passing
  quality checks does not replace required review and replanning

The corresponding CI checks are exactly `test / run`, `coverage / run`,
`test-performance / run`, and `quality`.

## Manual Report Verification

Start the application:

```bash
go run ./cmd/ghostfolio-cryptogains
```

Generate Markdown and PDF with the same year, cost-basis method, calculation
currency, and deterministic or development activity history.

Verify each successful result, including an opener-warning result:

- it identifies the output as cleartext financial data
- it lists every saved path, meaning both Markdown paths or the combined PDF path
- it tells the user to delete every listed file to remove the exported data
- leaving the result flow retains no report history, reopen catalog, or path list

Verify the Markdown main report:

- the exact warning is one fully bold standalone logical paragraph; PDF may wrap
  it into multiple ordered physical lines or text runs without creating another
  occurrence
- it follows `Report Calculation Currency` and precedes
  `Gains-And-Losses Summary`
- every monetary amount and unit price has two fractional digits
- quantities equal the FR-009 canonical representation computed from exact model
  values
- the separate Annex file does not repeat the warning

Verify the Markdown Annex:

- `Full Liquidation Event` cells read `Yes` or `No`
- a zero-priced holding reduction has a blank `Original Activity Currency` cell
  and a populated `Calculation Currency` cell
- all-missing classified monetary fields remain blank, while present exact-zero
  fields display as `0.00`
- a non-zero-priced row retains its original activity currency
- each Converted Amounts entry starts after a visible break and follows exact
  colon, arrow, and semicolon spacing
- rate values retain every significant digit of normalized provider evidence
  while provider lexical trailing zeros remain omitted

Open the generated PDF in a reader that supports selectable text and verify:

- the warning is fully bold, readable, and placed before the summary
- displayed financial, quantity, rate, boolean, and currency values agree with
  Markdown
- each Converted Amounts entry has one logical start; physical wrapping within
  one entry does not count as another start
- long entries may wrap within their own line, but the next entry still starts
  on a new line
- expanded rows do not overlap borders, following rows, or the bottom margin
- a row that fits only a fresh page moves before any part is drawn; an
  intentionally overheight row fails instead of splitting or clipping
- relocating a table before its first row does not emit a continuation label;
  actual continuation pages retain their inherited label and repeated header
- every page remains landscape A4 with searchable/selectable text

These checks do not establish tagged-PDF, PDF/UA, semantic table association,
assistive reading-order, or screen-reader conformance and must not be reported as
such.

Manual PDF inspection supplements automated layout-recorder and concrete-PDF
text-run tests for reader-specific visual behavior; the extended project
inspector supplies the automated font and coordinate evidence.

## Calculation Regression Check

Review calculation-level assertions separately from rendered-string assertions:

- exact basis, proceeds, gains, losses, totals, quantities, conversion amounts,
  rates, rate metadata, currencies, inclusion and omission states, and
  classifications remain unchanged before and after each renderer
- the existing calculated audit activity-currency value remains unchanged and is
  suppressed only in the qualifying visible presentation row
- summary inclusion continues to use exact zero, so a non-zero value displayed
  as `0.00` remains present
- conversion component omission continues to require exact zero-to-zero values
- zero-priced classification is not inferred from displayed `0.00`
- report-level all-missing and mixed zero/missing compatibility inputs preserve
  missing values, while the sync validator's amount-resolution admission rule is
  unchanged

Do not run `make regenerate-empirical-fixtures`. The YAML dataset and generated
oracle fixtures under `testdata/empirical/` are read-only for this feature.

## Security Review

- Inspect generated documents, result and error text including wrapped causes,
  diagnostics, documentation examples, and generated test artifacts. Confirm
  they contain no real credential, bearer or JWT value, reusable authentication
  or decryption material, or raw encrypted/decrypted protected-payload
  serialization. Confirm real user financial data appears only in contracted
  user-export fields and separately authorized diagnostic modes, not errors,
  logs, screenshots, examples, or committed/generated fixtures. Redaction tests
  and all examples and fixtures use only clearly synthetic non-reusable data.
- Confirm controlled Markdown `<br>` and PDF newline delimiters are inserted
  only after dynamic entry components are escaped and sanitized and fixed entry
  syntax is assembled.
- Confirm PDF generation remains in-process and local-only with application
  fonts and no remote service.
- Confirm the constitution-prerequisite PDF finalization error returns through
  normal report error handling before output reservation or writing, leaves the
  process running, and invokes no opener.
- Confirm the existing requested `0600` file mode and reservation/cleanup
  sequence removes only current-attempt paths on failure and preserves colliding
  pre-existing files and earlier successful bundles.
- Confirm cleartext status, every saved path, and deletion guidance are shown for
  normal and opener-warning success, and leaving the result flow retains no
  report copy, path state, history, reopen catalog, or automatic re-ingestion.
- Confirm no new dependency, network request, authentication path, telemetry,
  report history, or automatic report re-ingestion was introduced.
- Confirm new OWASP Cryptographic Storage evidence remains N/A because no
  financial or person-linked data enters application-managed persistence and
  existing token-derived encrypted snapshots are unchanged.
