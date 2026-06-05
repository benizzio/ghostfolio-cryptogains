# Contract: Empirical Calculation Tests

## Scope

This contract defines the isolated empirical Go tests that compare project calculation output against normalized hledger oracle fixtures.

## Package

Empirical tests live in:

```text
tests/empirical
```

They must not be placed in `tests/integration`.

## Target Commands

Fixture-backed empirical verification:

```bash
go test ./tests/empirical -count=1 -v
```

Full repository verification:

```bash
make test
make coverage
```

Fixture generation or regeneration must be explicit and documented by implementation tasks. Normal fixture-backed test runs must not require hledger when all required golden fixtures are present.

## Test Boundary Rules

Empirical tests may use:

- `internal/report/calculate`
- `internal/report/model`
- `internal/sync/model`
- `internal/support/decimal`
- `internal/support/math`
- test-local dataset and oracle helpers

Empirical tests must not call or assert:

- Ghostfolio HTTP clients
- Ghostfolio DTO parsing
- TUI rendering
- protected snapshot encryption or decryption
- Markdown rendering
- report output writers
- OS opener behavior
- generated report filenames
- Documents-folder paths

## Comparison Flow

1. Validate `testdata/empirical/financial-dataset.yaml`.
2. Load required golden fixtures from `testdata/empirical/golden/`.
3. Generate a missing fixture with the vendored hledger oracle only when the fixture is absent and generation is allowed by the test helper policy.
4. Translate dataset rows into a `syncmodel.ProtectedActivityCache` equivalent.
5. For each comparable case, create a `reportmodel.ReportRequest` for the fixture year and method.
6. Call `calculate.Calculate`.
7. Normalize `reportmodel.CapitalGainsReport` into `ProjectCalculationOutput`.
8. Compare normalized project output to `OracleOutput`.
9. Report all comparison failures with deterministic non-secret context.

## Required Method Coverage

The empirical suite must compare every supported project cost-basis method:

- FIFO
- LIFO
- HIFO
- Average Cost Basis
- Scope-Local Exact Unit Matching otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order

Scope-local hybrid comparisons must distinguish hledger-backed sub-evidence from project-owned composition rules.

## Required Field Coverage

Comparable output must include at least:

- realized gain or loss
- allocated basis
- closing quantity
- closing basis
- full-liquidation effects where comparable
- method-specific lot or pool evidence where comparable
- zero-priced holding reduction effects

## Precision Rules

- Quantity fields compare by exact decimal equality.
- Financial value fields compare after normalization under the selected decimal policy.
- The default selected policy is the project's production 16-decimal round-half-up internal calculation policy.
- If hledger cannot be configured or normalized to match the default policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to the hledger-established policy before invoking project calculation.
- Production behavior must keep the 16-decimal default when `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` is unset.
- Financial value fields may use documented tight per-field tolerances only for residual hledger/project deviations after decimal-policy alignment.
- Quantity tolerance is always zero.
- Financial tolerances must be small enough to catch material drift and systematic method differences.
- Comparison code must use decimal arithmetic only.
- Floating-point math is invalid in dataset parsing, normalization, and comparison.

## Failure Output Contract

A comparison failure must identify:

- dataset case ID
- cost-basis method
- report year
- asset identity key
- field path
- selected decimal policy
- expected value
- actual value
- difference
- tolerance
- relevant source IDs

Failure output must not include:

- Ghostfolio tokens
- JWTs
- real user data
- raw protected snapshot payloads
- Markdown report content
- TUI text
- output filenames or Documents paths

## Golden Fixture Policy

- Golden fixtures are authoritative for fixture-backed tests.
- hledger generation must not run when required fixtures are present.
- If generation is needed because a fixture is absent, failure to find or execute vendored hledger must produce an actionable setup error.
- Regeneration must update hashes and metadata together with expected values.
- Fixture drift must be visible through changed hashes or changed normalized values.

## Isolation Contract

The empirical suite is supplemental. It must not replace:

- existing contract tests
- existing integration tests
- unit tests needed for isolated edge cases
- `make coverage`
- performance verification

The suite must remain calculation-focused and must not expand into report-format, UI, transport, or storage behavior.
