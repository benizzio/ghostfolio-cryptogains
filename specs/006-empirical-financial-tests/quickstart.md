# Quickstart: Empirical Solidified Financial Tests

## Goal

Validate the internal empirical suite that:

- stores a synthetic external dataset
- generates or reuses hledger-backed normalized oracle fixtures
- translates the same dataset into project calculation inputs
- compares pure calculation output for every supported cost-basis method
- avoids Ghostfolio, TUI, snapshot encryption, Markdown rendering, and report output assertions

## Prerequisites

- Go 1.26.3 installed
- repository dependencies available through normal Go module resolution
- vendored hledger materials present under `third_party/hledger/`
- golden fixtures present under `testdata/empirical/golden/` for normal test runs
- no real user data in `testdata/empirical/`

## Validate Dataset And Fixtures

Run:

```bash
go test ./tests/empirical -run TestEmpiricalDatasetValidation -count=1 -v
```

Expected result:

- at least 150 activities are present
- at least 3 source-calendar years are present
- every supported method has coverage
- every required edge-case tag from the specification is present
- all financial values are decimal strings
- priced rows use one currency only
- zero-priced holding reductions have explanations
- no secret-like or real-user fixture content is accepted

## Run Empirical Comparisons

Run:

```bash
go test ./tests/empirical -count=1 -v
```

Expected result:

- existing golden fixtures are loaded
- hledger is not invoked while required fixtures are present
- dataset rows are translated into calculation-layer inputs
- `calculate.Calculate` runs for every comparable method and year
- normalized project output matches oracle output under the documented decimal-policy and financial-tolerance contract
- if hledger cannot match the production 16-decimal policy, the empirical command configures `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to the hledger-established policy before project calculation runs
- failures, if any, identify case, method, year, asset, field, selected decimal policy, expected value, actual value, difference, tolerance, and source IDs

## Run Full Repository Verification

Run:

```bash
make test
make coverage
```

Expected result:

- existing contract, integration, unit, package-local, and empirical tests pass according to the final implementation task configuration
- coverage gate still passes for maintained production packages
- empirical tests remain supplemental and do not replace existing suites

## Missing Golden Fixture Path

1. Remove or withhold one golden fixture in a controlled development branch.
2. Run the documented empirical generation command implemented by this feature.

Expected result:

- the helper resolves the repository-vendored hledger command
- hledger version is checked and recorded
- command arguments are explicit and recorded
- generated hledger input and normalized output hashes are recorded
- missing or unsupported hledger fails with an actionable setup error
- runtime application code remains unaffected

## Inspect hledger Vendoring

Inspect:

```text
third_party/hledger/
```

Expected result:

- GPL-3.0-or-later license text is present
- upstream source URL is documented
- selected version is documented
- checksum is documented
- source or complete corresponding source is present for any executable artifact
- platform support and failure modes are documented
- no binary-only vendoring is used

## Inspect Empirical Artifacts

Inspect:

```text
testdata/empirical/financial-dataset.yaml
testdata/empirical/golden/
testdata/empirical/hledger/
```

Expected result:

- artifacts are synthetic and reviewable
- golden fixtures include hledger version, command arguments, dataset hash, hledger input hash, output hash, and normalization version
- unsupported hledger segments have explicit reasons
- no artifact contains tokens, JWTs, real user data, protected snapshot payloads, generated Markdown reports, TUI text, or output paths

## Manual Failure Inspection

Introduce a controlled mismatch in a local throwaway branch and run:

```bash
go test ./tests/empirical -count=1 -v
```

Expected result:

- the failure identifies the exact dataset case
- the failure identifies method, year, asset, field, selected decimal policy, expected value, actual value, difference, tolerance, and source IDs
- the failure does not print secrets, raw protected payloads, Markdown report content, TUI text, or Documents paths

## Read-Only Policy After Completion

After this feature lands:

- `testdata/empirical/financial-dataset.yaml` is read-only for ordinary feature work
- calculation changes must adapt project code or project-owned tests around the dataset
- dataset corrections or expansions require a later isolated dataset-maintenance spec
