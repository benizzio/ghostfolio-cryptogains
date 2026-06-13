# Contract: Empirical Calculation Tests

**Bugfix**: 2026-06-10 — [BUG-001] Updated empirical test contract for rotki-backed pure-method fixtures, Scope-Local Hybrid composite assertions, and supported-fixture skip failures.

## Scope

This contract defines the isolated empirical Go tests that compare project calculation output against normalized external-oracle and composite-oracle fixtures.

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

Optional missing-fixture generation during a test run:

```bash
GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES=true go test ./tests/empirical -count=1 -v
```

Explicit oracle helper commands:

```bash
go run ./tools/empiricaloracle
go run ./tools/empiricaloracle --regenerate
```

Full repository verification:

```bash
make test
make coverage
```

Fixture generation or regeneration is explicit. Normal fixture-backed test runs must not require external oracle generation when all required golden fixtures are present.

## Test Boundary Rules

Empirical tests may use:

- `internal/report/calculate`
- `internal/report/model`
- `internal/sync/model`
- `internal/support/decimal`
- `internal/support/math`
- test-local dataset and oracle helpers

Empirical tests may trigger the repository-owned oracle helper only through the missing-fixture policy in `tests/empirical/fixture/oracle_generation_policy.go`.

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

## External Oracle Boundary

- Committed fixture-backed runs must load `testdata/empirical/golden/` and must not require `.cache/empiricaloracle/rotki-source/` while those fixtures are present.
- Missing-fixture generation is opt-in only. Without `GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES=true`, a missing committed fixture is a setup error, not an implicit download or regeneration step.
- The automatic generation path used by the tests is `go run ./tools/empiricaloracle`.
- Explicit full regeneration is `go run ./tools/empiricaloracle --regenerate`.
- Regeneration must execute the project-owned Python adapter against verified pinned rotki source in `.cache/empiricaloracle/rotki-source/`.
- Regeneration must not depend on a global `rotki` executable, a vendored `third_party/rotki/source/` checkout, or committed raw payloads under `testdata/empirical/rotki/`.

## Comparison Flow

1. Validate `testdata/empirical/financial-dataset.yaml`.
2. Load required golden fixtures from `testdata/empirical/golden/`.
3. Generate a missing fixture with the documented external oracle or composite oracle only when the fixture is absent and generation is explicitly allowed by the test helper policy; otherwise fail with an actionable setup error.
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
- Scope-Local Hybrid (`scope_local_hybrid`)

Scope-Local Hybrid (`scope_local_hybrid`) comparisons must distinguish rotki-backed arithmetic assertions from project-owned composition rules.

## Required Field Coverage

Comparable output must include at least:

- realized gain or loss
- allocated basis
- closing quantity
- closing basis
- full-liquidation effects when the fixture records source IDs, evidence type, and expected values
- method-specific lot or pool evidence when the fixture records source IDs, evidence type, and expected values
- Average Cost currently compares aggregate yearly values only; committed fixtures omit match evidence and record pool-provenance limits through unsupported segments
- zero-priced holding reduction effects through non-oracle unit, integration, or contract tests only; they are not counted as empirical external-oracle fixture coverage after BUG-001

## Comparability Rules

- A field is comparable only when the oracle fixture contains a normalized expected value for the same case, method, year, asset, and source-row segment.
- A field is not comparable when an unsupported segment covers that field.
- Unsupported fields must be reported as skipped with the unsupported reason; they must not be counted as matched external-oracle assertions.
- Current committed Scope-Local Hybrid fixtures use `rotki_backed` match evidence and `project_composition_only` unsupported segments for project-owned lifecycle behavior.
- The shared fixture schema also permits `project_composition_rule` match evidence. If used later, it must include a stable rule ID and the source-row segment it covers.
- Supported empirical fixture groups must fail if skipped before project calculation and oracle comparison. Only fixture-backed unsupported field-level segments may be skipped with an explicit reason.

## Precision Rules

- Quantity fields compare by exact decimal equality.
- Financial value fields compare after normalization under the selected decimal policy.
- The default selected policy is the project's production 16-decimal round-half-up internal calculation policy.
- Accepted `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` values use the form `scale=<digits>,rounding=half_up`.
- The required accepted value is `scale=16,rounding=half_up`, matching the production default.
- The current committed fixtures all use `scale=16,rounding=half_up`.
- Additional external-oracle-aligned accepted values may be added only when the selected external oracle cannot align with the production default, and each added value must be documented with the oracle name, pinned version or commit, and reason.
- If the selected external oracle cannot be configured or normalized to match the default policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to the external-oracle-established policy before invoking project calculation.
- Production behavior must keep the 16-decimal default when `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` is unset.
- Financial value fields may use documented per-field tolerances only for residual external-oracle/project deviations after decimal-policy alignment.
- Quantity tolerance is always zero.
- Non-zero financial tolerances must not exceed one unit at the selected decimal-policy scale. For the production 16-decimal policy, the maximum is `0.0000000000000001`.
- A non-zero financial tolerance must include a fixture note explaining why exact equality is not achievable for that external-oracle-derived value.
- The current committed fixtures use zero financial tolerance for every compared financial field.
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
- External oracle generation must not run when required fixtures are present.
- If generation is needed because a fixture is absent, failure to find or execute the documented external oracle boundary must produce an actionable setup error.
- The test-owned missing-fixture path calls only `go run ./tools/empiricaloracle`.
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

- The empirical package's static isolation checks must continue to reject imports of Ghostfolio, TUI, snapshot, Markdown, and report-output packages.
- The empirical package's static isolation checks must continue to reject report-output identifiers, generated filename helpers, and Documents-path handling.
