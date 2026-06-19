# Research: Empirical Solidified Financial Tests

**Bugfix**: 2026-06-10 — [BUG-001] Replaced stale hledger-only oracle research with rotki-backed pure-method oracle and Scope-Local Hybrid composite-oracle decisions.

## Decision: Store The Empirical Dataset As Project-Owned YAML

Use `testdata/empirical/financial-dataset.yaml` as the canonical empirical external dataset. The file is synthetic, simplified, human-readable, and validates with project-owned Go code before any oracle or calculation comparison runs.

**Rationale**: The specification requires a clear maintainable format without hidden transformation logic. YAML supports explicit named fields and readable activity records better than positional CSV while staying reviewable in Git. A single canonical dataset avoids separate hand-maintained inputs for ~~hledger~~ external-oracle adapters and project calculation.

**Alternatives considered**:

- CSV: rejected because nested source scope, coverage tags, and optional monetary fields become hard to review without hidden column conventions.
- JSON: rejected because it is strict and machine-friendly but less maintainable for a large hand-reviewed synthetic ledger.
- ~~hledger journal as the canonical dataset~~ External-oracle-native input as the canonical dataset: rejected because the project also needs source IDs, scope reliability, zero-priced-reduction explanations where applicable, and project-specific ordering metadata that should not be hidden in external tool syntax.

## Decision: Keep Dataset Creation In This Dedicated Dataset-Maintenance Spec

Create and populate the empirical dataset in this feature, then treat it as read-only for ordinary later work.

**Rationale**: The constitution allows empirical dataset changes only in isolated specs whose explicit purpose includes dataset creation, correction, or expansion. This specification is such a dataset-maintenance spec and records the read-only policy after completion.

**Alternatives considered**:

- Spread dataset additions across ordinary calculation changes: rejected because it violates the constitution's dataset isolation rule.
- Defer dataset creation to a later feature after adding tests: rejected because empirical tests need a stable external dataset and golden fixtures to be meaningful.
- Make the dataset generated only from code: rejected because the spec requires a human-readable external dataset without hidden transformation logic.

## Decision: Use ~~hledger As A Vendored Test-Time External Oracle~~ A Rotki-Backed Test-Time External Oracle For Pure Methods

~~Use hledger as a separate repository-vendored command-line tool for oracle generation.~~ BUG-001 supersedes the hledger-only oracle decision. Use a pinned rotki-based test-time oracle adapter for FIFO, LIFO, HIFO, and Average Cost aggregate fixture generation. The Go code invokes external oracle tooling only from test-helper or tool code when a required golden fixture is absent or when maintainers explicitly regenerate fixtures.

**Rationale**: ~~The spec requires hledger as the external oracle. hledger is an actively maintained plain-text-accounting engine with cost-basis and lot support, command-line output formats, and GPL-3.0-or-later licensing.~~ The BUG-001 evidence shows the hledger-backed fixture set skipped 11 of 13 supported groups before project calculation and oracle comparison. Rotki is the planned external oracle for pure FIFO, LIFO, HIFO, and Average Cost aggregate expected values because it can provide the method coverage needed by the patched acceptance criteria. Pinning source version or commit, adapter constraints, arguments, and hashes keeps empirical tests reproducible and avoids dependence on the developer's system installation.

**Alternatives considered**:

- Use the superseded plain-text-accounting boundary from `PATH`: rejected because version drift would make oracle output unstable and because the earlier single-oracle output is no longer sufficient for BUG-001 acceptance.
- Link or import external-oracle runtime libraries into production code: rejected because runtime and Go test code must not embed external-oracle runtime code; the oracle boundary is test-time only.
- Use Beancount or Ledger as the primary oracle: rejected because they do not resolve the BUG-001 method coverage gap. Beancount and Ledger can remain research references only.
- Hand-author all expected values: rejected because this would not be an external oracle and would provide weaker drift detection.

## Decision: Vendor Or Document License-Compatible External Oracle Materials, Not Binary-Only Artifacts

Document external oracle provenance materials under `third_party/` with applicable license text, upstream source URL, pinned version or commit, source checksum, platform support notes, adapter constraints, and supported artifact paths. Rotki source provenance and licensing must be documented before fixture regeneration.

**Rationale**: The repository is GPLv3, so external oracle materials are acceptable only when applicable license and source-distribution obligations are preserved. The spec explicitly prohibits undocumented binary-only vendoring and requires source provenance, license text, version or commit identity, checksums, and platform support notes.

**Alternatives considered**:

- Commit only a downloaded executable: rejected because binary-only vendoring is prohibited unless source-distribution obligations are satisfied and documented.
- Fetch external oracle source during normal fixture-backed tests: rejected because it harms reproducibility and can make CI dependent on network availability. BUG-002 permits only the explicit fixture-regeneration command to download or reuse verified pinned rotki source in the untracked source cache.
- Document manual installation instead of repository-controlled provenance: rejected because fixture generation must be reproducible from documented, pinned oracle materials.

## Decision: Use Verified Untracked rotki Source For Fixture Regeneration

Pin rotki to release tag `v1.43.1`, resolving to commit `a2e00be49a0ea36e7563a5d235cfa6a7c91edbfb`, and record the upstream source archive URL `https://github.com/rotki/rotki/archive/refs/tags/v1.43.1.tar.gz` with SHA-256 `8434b653104f8d5b0638e98d88a5ef256fac7720cc459eb33b729e2848900e3b`. Copy the upstream `LICENSE.md` text from `https://raw.githubusercontent.com/rotki/rotki/v1.43.1/LICENSE.md`, whose fetched SHA-256 is `eb6f58a98d8bdb6d3c8fee3817543589f3cd0921d14748fa0630edff2d4c08b0`, into `third_party/rotki/LICENSE.md` as repository-controlled license evidence.

BUG-002 supersedes the BUG-001 bootstrap shortcut that accepted committed raw rotki outputs or repository-controlled normalization inputs as oracle evidence. Explicit fixture regeneration must download or reuse the pinned rotki source under `.cache/empiricaloracle/rotki-source/`, verify the configured source URL, release tag, peeled commit, signed tag object, checksum, and adapter constraints, then execute rotki calculation code through the project-owned local adapter boundary. The implementation writes the pinned archive to `.cache/empiricaloracle/rotki-source/rotki-v1.43.1.tar.gz`, extracts it to `.cache/empiricaloracle/rotki-source/rotki-1.43.1`, records the reusable verification manifest at `.cache/empiricaloracle/rotki-source/verified-source.json`, and generates per-case adapter inputs under `.cache/empiricaloracle/oracle-inputs/`. The rotki cache paths are untracked and covered by `.gitignore`. Normal fixture-backed empirical tests must continue to rely on committed golden fixtures and must not download rotki, require the source cache, or invoke oracle generation.

The implemented verification method is `archive_sha256+git_ls_remote_tag`. The regeneration boundary shells out to local `git` for `ls-remote` tag verification and to local `python3` or `python` to execute `tools/empiricaloracle/rotki_adapter.py`. The adapter loads `rotkehlchen/fval.py` and `rotkehlchen/accounting/cost_basis/base.py` directly from the verified source tree and uses project-owned support stubs instead of a developer-global rotki installation.

Committed raw rotki payloads, bootstrap manifests, and hand-authored adapter inputs are not authoritative regeneration evidence and must not be used as the source of regenerated oracle data.

**Rationale**: BUG-002 requires regenerated fixtures to derive from executed pinned rotki source while avoiding vendored rotki code and developer-global installations. A verified untracked source cache keeps the repository free of rotki source while still making the regeneration boundary reproducible from documented provenance and checksums.

**Alternatives considered**:

- Claim local prototype outputs as repository evidence: rejected because they are not reproducible from version-controlled materials in this checkout.
- Commit raw rotki responses or hand-authored rotki datasets: rejected because BUG-002 requires regenerated oracle data to come from executed pinned rotki source through the local adapter boundary.
- Require maintainers to install rotki locally: rejected because developer-global installations are not reproducible oracle evidence.
- Vendor the full rotki source tree: rejected because BUG-002 requires a non-vendored untracked source acquisition path.

Normal fixture-backed `go test ./tests/empirical -count=1 -v` runs must stay no-network when committed golden fixtures are present. Only explicit regeneration may download the pinned source archive, and it must write or reuse artifacts only below `.cache/empiricaloracle/rotki-source/`.

Co-authored by: OpenCode

## Decision: Persist Normalized Oracle Output As Golden Fixtures

Store normalized oracle outputs under `testdata/empirical/golden/` and make empirical tests consume those fixtures by default. Generate fixtures only when missing or explicitly requested by maintainers.

**Rationale**: Persisted fixtures make normal tests deterministic and avoid invoking external oracle generation on every run. Input and output hashes expose drift when the dataset, external-oracle input, adapter code, composite-rule version, or pinned oracle identity changes.

**Alternatives considered**:

- Generate oracle results on every test run: rejected because it requires external oracle tooling for all empirical runs and increases runtime and platform fragility.
- Never generate in tests: rejected because the spec permits external oracle generation when a required fixture is absent.
- Store only human-readable external-oracle terminal output: rejected because tests need normalized assertable data rather than terminal formatting.

## Decision: Normalize Oracle Output To Project-Owned JSON

Use project-owned JSON golden fixtures with decimal values represented as strings and explicit metadata for version, command, hashes, normalization, method, year, asset, fields, and unsupported cases.

**Rationale**: JSON is better for machine assertions than external-oracle terminal output and is easier to hash deterministically than loosely formatted text. Decimal strings preserve exactness and avoid floating-point parsing.

**Alternatives considered**:

- Compare external-oracle terminal output directly: rejected because terminal output is presentation, not a stable calculation contract.
- Store Go source fixtures: rejected because it would hide expected data in code and make review harder.
- Store binary fixtures: rejected because they are not human-reviewable.

## Decision: Translate Dataset To `ProtectedActivityCache`-Equivalent Inputs

Empirical tests translate dataset rows into project calculation inputs equivalent to synced protected activity records, then call the pure report calculation layer.

**Rationale**: The spec requires calculation-layer comparison only. Existing `internal/report/calculate.Calculate` accepts `syncmodel.ProtectedActivityCache`, so tests can exercise the same calculation boundary without Ghostfolio, snapshots, TUI, Markdown rendering, or output writers.

**Alternatives considered**:

- Drive the TUI workflow: rejected because this suite must not assert UI text or report output boundaries.
- Parse generated Markdown reports: rejected because empirical comparisons must use pure normalized calculation output.
- Call lower-level basis states only: rejected because the suite should validate the calculation layer integration from normalized activity history through report model output.

## Decision: Use A Dedicated `tests/empirical` Package

Place empirical solidified financial tests under `tests/empirical` as a completely isolated Go test package.

**Rationale**: The spec explicitly requires an isolated empirical package separate from `tests/integration`. Isolation prevents the empirical suite from becoming coupled to existing Ghostfolio, TUI, snapshot, or report-format integration tests while preserving integration-style execution of calculation packages.

**Alternatives considered**:

- Add tests to `tests/integration`: rejected by clarification and because it would mix empirical oracle behavior with product workflow tests.
- Add package-local tests under `internal/report`: rejected because the dataset and oracle comparison span multiple packages and should stay isolated.
- Add only unit tests: rejected because the objective is empirical integration validation of the calculation layer.

## Decision: Cover Scope-Local Hybrid With Rotki-Backed Arithmetic Plus Project Composition Rules

For Scope-Local Hybrid (`scope_local_hybrid`), use a separate composite oracle. The implemented composite oracle executes the rotki adapter with rotki method `average_cost` for the comparable arithmetic slice, then records the remaining project-owned routing and lifecycle segments as explicit `project_composition_only` unsupported segments with documented reasons for fallback activation, carry-forward until zero, reset after zero, and independent scope state.

**Rationale**: No single selected external oracle models this project's full hybrid lifecycle as one native method. Using rotki-backed arithmetic where valid keeps external evidence where possible while avoiding false claims that any external tool directly implements the project's hybrid method. Recording the remaining hybrid segments as explicit unsupported composite-only slices matches the current committed fixture behavior more accurately than claiming first-class `project_composition_rule` evidence that the implementation does not yet persist.

**Alternatives considered**:

- Treat any selected external oracle as a native scope-local-hybrid oracle: rejected because that would be inaccurate.
- Exclude Scope-Local Hybrid from empirical testing: rejected because the spec requires every supported method.
- Hand-author all hybrid expected results without external-oracle-backed arithmetic evidence where available: rejected because it weakens the empirical external-oracle objective.

## Decision: Mark Faithfully Unrepresentable Cases As Unsupported For External Comparison

If a dataset case cannot be represented in the selected external oracle or composite oracle without changing its financial meaning, the oracle marks that case unsupported for external-oracle comparison with a reason. BUG-001 removes zero-priced holding reductions from empirical external-oracle fixture scope instead of carrying them as supported fixture groups or superseded-boundary-specific unsupported segments.

**Rationale**: Silent approximation would create false confidence and could hide drift. Explicit unsupported markers preserve dataset coverage while making oracle limits visible.

**Alternatives considered**:

- Fabricate expected values for unsupported selected-oracle cases: rejected because it is not external-oracle evidence.
- Remove unsupported project edge cases from all testing: rejected because the project must still cover project-supported edge cases through non-oracle unit, integration, or contract tests when they are outside empirical external-oracle scope.
- Fail the whole suite on any unsupported case: rejected because some project-specific semantics may legitimately lack a direct selected-oracle representation.

## Decision: Align Decimal Policy First, Then Apply Tight Financial Tolerances

Quantities compare by exact decimal equality after normalization. Financial fields are compared after one selected decimal policy is applied to both external-oracle output and project calculation output. The oracle first attempts to configure or normalize external-oracle-derived values to this project's production internal policy: 16 decimal places with round-half-up handling for required non-terminating divisions and proportional allocations. If the selected external oracle cannot be configured to match that policy for every otherwise valid empirical case, the implementation must expose an application-run-scoped environment-variable override, expected as `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY`, so a given empirical run or helper invocation can use the external-oracle-established decimal policy. Normal production runs and current empirical runs keep the `scale=16,rounding=half_up` default when the variable is unset. Custom documented values are allowed when needed to align with an oracle, with a practical maximum scale of 64 for safety. After decimal-policy alignment, calculated financial values may use documented per-field tolerances for residual external-oracle deviations. Quantity tolerance is `0`. Non-zero financial tolerances must be declared per field, must not exceed one unit at the selected decimal-policy scale, and must include a note explaining why exact equality is not achievable for that external-oracle-derived value.

**Rationale**: Empirical validation should minimize avoidable differences by aligning decimal policy before comparison. A selected external oracle may still produce small residual deviations because it is an external accounting engine with its own internal representation and reporting behavior. A documented per-field tolerance prevents immaterial oracle/tooling differences from failing the suite, while the decimal-policy alignment step and one-unit-at-scale cap keep the comparison sensitive to actual calculation drift.

**Alternatives considered**:

- Use broad global comparison deltas: rejected because they can hide systematic method differences.
- Use tolerance without first aligning decimal policy: rejected because avoidable precision mismatches would consume the tolerance budget and weaken drift detection.
- Require exact equality for every financial value after policy alignment: rejected because small residual external-oracle deviations may still occur and should not fail otherwise valid empirical cases.
- Require exact equality under the production 16-decimal policy regardless of selected-oracle precision behavior: rejected because it could reject valid external evidence when the selected oracle's established policy cannot be made to match the project's production default.
- Use floating-point comparisons: rejected by the constitution.

## Decision: Keep Currency Conversion Permanently Out Of Scope

Use one currency for every empirical dataset case and block any currency conversion logic in the empirical suite.

**Rationale**: The suite exists to validate cost-basis and gains/losses behavior. Currency conversion has different source, rounding, and audit requirements and would make oracle comparison ambiguous.

**Alternatives considered**:

- Include multiple currencies with no conversion: rejected because cross-activity financial outputs would become ambiguous.
- Add exchange-rate fixtures: rejected because the spec permanently excludes currency conversion from this suite.
- Reuse product conversion logic later: rejected for this suite; future conversion work needs a separate dedicated test scope.

## Decision: Validate Synthetic-Only Fixture Content

Add validation that rejects real-looking tokens, bearer prefixes, JWT-like values, real account names, and unreviewed copied upstream fixture text patterns in the empirical dataset and oracle fixtures.

**Rationale**: Empirical fixtures are persisted in the repository and must remain non-secret and synthetic. Validation lowers the risk that review misses pasted real data.

**Alternatives considered**:

- Rely on code review only: rejected because persisted financial fixtures create recurring leakage risk.
- Encrypt fixtures: rejected because they are synthetic and must be reviewable.
- Use production redaction on fixtures: rejected because the goal is to prevent secret-like content from entering the fixture set.
