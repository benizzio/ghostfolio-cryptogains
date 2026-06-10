# Quickstart: Empirical Solidified Financial Tests

**Bugfix**: 2026-06-10 — [BUG-001] Updated quickstart expectations for rotki-backed pure-method fixtures, Scope-Local Hybrid composite fixtures, and zero-priced external-oracle exclusion.

## Goal

Validate the internal empirical suite that:

- stores a synthetic external dataset
- generates or reuses normalized oracle fixtures from rotki-backed pure-method and Scope-Local Hybrid composite sources
- translates the same dataset into project calculation inputs
- compares pure calculation output for every supported cost-basis method
- avoids Ghostfolio, TUI, snapshot encryption, Markdown rendering, and report output assertions

## Prerequisites

- Go 1.26.3 installed
- repository dependencies available through normal Go module resolution
- documented external oracle materials present under `third_party/rotki/` and any retained hledger materials under `third_party/hledger/`
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
- zero-priced holding reductions are excluded from empirical external-oracle fixture coverage and remain covered by non-oracle unit, integration, or contract tests
- no secret-like or real-user fixture content is accepted

## Run Empirical Comparisons

Run:

```bash
go test ./tests/empirical -count=1 -v
```

Expected result:

- existing golden fixtures are loaded
- external oracle generation is not invoked while required fixtures are present
- dataset rows are translated into calculation-layer inputs
- `calculate.Calculate` runs for every comparable method and year
- normalized project output matches oracle output under the documented decimal-policy and financial-tolerance contract
- if the selected external oracle cannot match the production 16-decimal policy, the empirical command configures `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to the external-oracle-established policy before project calculation runs
- accepted decimal-policy values use the form `scale=<digits>,rounding=half_up`; `scale=16,rounding=half_up` is required and matches the production default
- quantity tolerance is zero, and any non-zero financial tolerance is capped at one unit of the selected decimal-policy scale
- failures, if any, identify case, method, year, asset, field, selected decimal policy, expected value, actual value, difference, tolerance, and source IDs
- fixture-backed empirical tests complete in 30 seconds or less when required golden fixtures are present

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

- the helper resolves the documented external oracle or adapter boundary
- external oracle name, source URL, pinned version or commit, and adapter constraints are checked and recorded
- adapter or command arguments are explicit and recorded
- generated external-oracle input and normalized output hashes are recorded
- missing or unsupported external oracle boundaries fail with an actionable setup error
- runtime application code remains unaffected

## Inspect External Oracle Materials

Inspect:

```text
third_party/rotki/
third_party/hledger/       # if retained for historical or auxiliary test-time material
tools/empiricaloracle/
```

Expected result:

- applicable license text is present
- upstream source URL is documented
- selected version or commit is documented
- source and artifact checksums are documented
- adapter constraints for FIFO, LIFO, HIFO, and Average Cost aggregate fixtures are documented
- Scope-Local Hybrid composite-rule provenance is documented
- supported artifact paths, platform support, and failure modes are documented
- no prohibited binary-only vendoring is used

## Inspect Empirical Artifacts

Inspect:

```text
testdata/empirical/financial-dataset.yaml
testdata/empirical/golden/
testdata/empirical/rotki/
testdata/empirical/hledger/       # if retained for historical or auxiliary test-time material
```

Expected result:

- artifacts are synthetic and reviewable
- golden fixtures include external oracle name, source URL, pinned version or commit, adapter arguments, dataset hash, external-oracle input hash, output hash, and normalization version
- unsupported external-oracle field-level segments have explicit reasons and zero-priced holding reductions are not counted as supported external-oracle coverage
- comparable fields are identified by case, method, year, asset, source-row segment, expected value, tolerance, and support status
- Scope-Local Hybrid assertions are labeled as `rotki_backed` arithmetic evidence or `project_composition_rule`
- no artifact contains tokens, JWTs, real user data, protected snapshot payloads, generated Markdown reports, TUI text, or output paths

## OWASP Top 10 Review Evidence

Before merge, record the OWASP Top 10 categories reviewed for this empirical test infrastructure and the resulting evidence. The review must cover at least the categories already scoped in `plan.md`: cryptographic failures, identification and authentication failures, insecure design, vulnerable and outdated components, software and data integrity failures, and logging or diagnostic leakage.

Expected result:

- the review confirms the suite does not touch protected storage or token boundaries
- the review confirms persisted fixtures are synthetic and non-secret
- the review records external oracle provenance, adapter, composite-oracle, and generated-fixture integrity controls
- the review records failure-output leakage controls

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
