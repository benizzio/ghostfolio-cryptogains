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
- documented external oracle materials present under `third_party/rotki/`
- committed golden fixtures present under `testdata/empirical/golden/` for normal fixture-backed test runs
- local Python available as `python3` or `python` only when explicit regeneration is required
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
- committed unsupported segments are logged as informational `skip_external_oracle` or `project_composition_only` cases without skipping supported fixture groups
- dataset rows are translated into calculation-layer inputs
- `calculate.Calculate` runs for every comparable method and year
- normalized project output matches oracle output under the documented decimal-policy and financial-tolerance contract
- the committed fixtures and current empirical runs use `scale=16,rounding=half_up`, which also matches the normal production default
- quantity tolerance is zero, and the committed fixtures currently use zero financial tolerance for every compared financial field
- failures, if any, identify case, method, year, asset, field, selected decimal policy, expected value, actual value, difference, tolerance, and source IDs
- observed in this checkout on 2026-06-13: `go test ./tests/empirical -count=1 -v` completed in `0.105s` wall-clock (`ok ... 0.042s`), with no fixture-generation or rotki-download step

## Decimal Policy Override

If oracle alignment requires a different decimal policy for one application run, set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` in the process environment before running `go test` or `go run ./tools/empiricaloracle`.

Example:

```bash
GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY=scale=18,rounding=half_up go test ./tests/empirical -count=1 -v
```

Expected result:

- when the variable is unset, normal production runs and current empirical runs keep `scale=16,rounding=half_up`
- custom documented values are allowed when needed to align with an oracle
- accepted values use `scale=<digits>,rounding=half_up`
- practical custom scale values should stay at or below 64 for safety

## Run Full Repository Verification

Run:

```bash
make test
make coverage
```

Expected result:

- existing contract, integration, unit, package-local, and empirical tests pass according to the final implementation task configuration
- existing contract, integration, unit, package-local, and empirical tests pass
- coverage gate still passes for maintained production packages
- empirical tests remain supplemental and do not replace existing suites

## Missing Golden Fixture Path

1. Remove or withhold one golden fixture in a controlled development branch.
1. Generate only the missing fixture with:

```bash
go run ./tools/empiricaloracle
```

1. If you want the test run itself to opt in to missing-fixture generation instead of failing with setup guidance, run:

```bash
GHOSTFOLIO_CRYPTOGAINS_GENERATE_MISSING_FIXTURES=true go test ./tests/empirical -count=1 -v
```

1. For an explicit full refresh of committed oracle artifacts, run:

```bash
go run ./tools/empiricaloracle --regenerate
```

Expected result:

- the helper writes deterministic rotki adapter inputs under `.cache/empiricaloracle/oracle-inputs/`
- the helper resolves the verified rotki source boundary only when a golden fixture is absent or `--regenerate` is used
- external oracle name, source URL, pinned version or commit, and adapter constraints are checked and recorded
- rotki source checksum is checked and recorded
- adapter or command arguments are explicit and recorded
- generated external-oracle input and normalized output hashes are recorded
- missing or unsupported external oracle boundaries fail with an actionable setup error
- runtime application code remains unaffected

## Inspect External Oracle Materials

Inspect:

```text
third_party/rotki/
tools/empiricaloracle/
```

Expected result:

- applicable license text is present
- upstream source URL is documented
- selected version or commit is documented
- source checksum is documented
- adapter constraints for FIFO, LIFO, HIFO, and Average Cost aggregate fixtures are documented
- Scope-Local Hybrid composite-rule provenance is documented
- the verified rotki source cache path is `.cache/empiricaloracle/rotki-source/`
- no vendored rotki source checkout or committed raw rotki output is used as oracle evidence

## Inspect Empirical Artifacts

Inspect:

```text
testdata/empirical/financial-dataset.yaml
testdata/empirical/golden/
```

Expected result:

- artifacts are synthetic and reviewable
- golden fixtures include external oracle name, source URL, source checksum, pinned version or commit, adapter arguments, dataset hash, external-oracle input hash, output hash, and normalization version
- unsupported external-oracle field-level segments have explicit reasons and zero-priced holding reductions are not counted as supported external-oracle coverage
- comparable fields are identified by case, method, year, asset, source-row segment, expected value, tolerance, and support status
- current Scope-Local Hybrid fixtures use `rotki_backed` match evidence plus `project_composition_only` unsupported segments for project-owned lifecycle assertions
- no artifact contains tokens, JWTs, real user data, protected snapshot payloads, generated Markdown reports, TUI text, or output paths

## OWASP Top 10 Review Evidence

Recorded review evidence for the implemented empirical boundary:

- A02 Cryptographic Failures: the empirical suite does not read or write protected snapshot storage; `tests/empirical/isolation_test.go` forbids `internal/snapshot` imports, and the suite works only from synthetic dataset and fixture files.
- A07 Identification and Authentication Failures: the empirical suite does not call Ghostfolio authentication or transport code; `tests/empirical/isolation_test.go` forbids `internal/ghostfolio`, and synthetic-content validation rejects token-like and JWT-like fixture content.
- A04 Insecure Design: normal fixture-backed runs load committed golden fixtures and do not require oracle generation; `tests/empirical/fixture/oracle_generation_policy.go` allows generation only after an explicit missing-fixture opt-in, and `tools/empiricaloracle/main_test.go` verifies the default helper path does not resolve rotki when fixtures already exist.
- A06 Vulnerable and Outdated Components: the rotki boundary is pinned to release `v1.43.1`, commit `a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb`, and source checksum `sha256:8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b` in `third_party/rotki/README.md`, and regeneration revalidates that pin in `tools/empiricaloracle/rotki_source.go`.
- A08 Software and Data Integrity Failures: oracle fixtures record and validate `dataset_input_hash`, `external_oracle_input_hash`, `source_checksum`, and `oracle_output_hash`; `tests/empirical/fixture/oracle_output.go` revalidates canonical JSON and the stable fixture hash, and rotki boundary tests reject vendored source and committed raw rotki outputs as regeneration evidence.
- A09 Security Logging and Monitoring Failures: comparison failures are formatted only with case, method, year, asset, field, expected value, actual value, difference, tolerance, decimal policy, and source IDs; the empirical isolation tests forbid report-output filenames and Documents-path handling, and fixture validation rejects secret-like content.

Expected result:

- the suite stays outside protected storage and token boundaries
- persisted fixtures remain synthetic and non-secret
- external-oracle provenance, adapter, composite-oracle, and fixture-integrity controls are documented and validated
- failure output stays limited to non-secret comparison context

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
