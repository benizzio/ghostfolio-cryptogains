# Research: Empirical Solidified Financial Tests

## Decision: Store The Empirical Dataset As Project-Owned YAML

Use `testdata/empirical/financial-dataset.yaml` as the canonical empirical external dataset. The file is synthetic, simplified, human-readable, and validates with project-owned Go code before any oracle or calculation comparison runs.

**Rationale**: The specification requires a clear maintainable format without hidden transformation logic. YAML supports explicit named fields and readable activity records better than positional CSV while staying reviewable in Git. A single canonical dataset avoids separate hand-maintained inputs for hledger and project calculation.

**Alternatives considered**:

- CSV: rejected because nested source scope, coverage tags, and optional monetary fields become hard to review without hidden column conventions.
- JSON: rejected because it is strict and machine-friendly but less maintainable for a large hand-reviewed synthetic ledger.
- hledger journal as the canonical dataset: rejected because the project also needs source IDs, scope reliability, zero-priced-reduction explanations, and project-specific ordering metadata that should not be hidden in hledger syntax.

## Decision: Keep Dataset Creation In This Dedicated Dataset-Maintenance Spec

Create and populate the empirical dataset in this feature, then treat it as read-only for ordinary later work.

**Rationale**: The constitution allows empirical dataset changes only in isolated specs whose explicit purpose includes dataset creation, correction, or expansion. This specification is such a dataset-maintenance spec and records the read-only policy after completion.

**Alternatives considered**:

- Spread dataset additions across ordinary calculation changes: rejected because it violates the constitution's dataset isolation rule.
- Defer dataset creation to a later feature after adding tests: rejected because empirical tests need a stable external dataset and golden fixtures to be meaningful.
- Make the dataset generated only from code: rejected because the spec requires a human-readable external dataset without hidden transformation logic.

## Decision: Use hledger As A Vendored Test-Time External Oracle

Use hledger as a separate repository-vendored command-line tool for oracle generation. The Go code invokes it only from test-helper or tool code when a required golden fixture is absent or when maintainers explicitly regenerate fixtures.

**Rationale**: The spec requires hledger as the external oracle. hledger is an actively maintained plain-text-accounting engine with cost-basis and lot support, command-line output formats, and GPL-3.0-or-later licensing. The latest GitHub release observed during planning is `1.52.1`, published 2026-04-28, with release assets carrying SHA-256 digests. Vendoring keeps empirical tests reproducible and avoids dependence on the developer's system installation.

**Alternatives considered**:

- Use system hledger from `PATH`: rejected because version drift would make oracle output unstable and would fail the repository-vendored-tool requirement.
- Link or import hledger libraries: rejected because runtime and Go test code must not embed hledger or hledger-lib; the oracle boundary is a separate command.
- Use Beancount or Ledger as the primary oracle: rejected because the spec identifies hledger as the primary tool for FIFO, LIFO, HIFO, average-cost, and lot evidence. Beancount and Ledger can remain research references only.
- Hand-author all expected values: rejected because this would not be an external oracle and would provide weaker drift detection.

## Decision: Vendor GPL-Compliant Source Materials, Not Binary-Only Artifacts

Vendor hledger materials under `third_party/hledger` with GPL-3.0-or-later license text, upstream source URL, selected version, checksum, platform support notes, and source or complete corresponding source for any executable artifact.

**Rationale**: hledger's README identifies GPLv3-or-later licensing, and the upstream repository includes a GPLv3 license. The repository is GPLv3, so vendoring is compatible only when license and corresponding-source obligations are preserved. The spec explicitly prohibits binary-only vendoring.

**Alternatives considered**:

- Commit only a downloaded executable: rejected because binary-only vendoring is prohibited and would not provide corresponding source.
- Fetch hledger at test time: rejected because it harms reproducibility and can make CI dependent on network availability.
- Document manual installation instead of vendoring: rejected because the spec requires a repository-vendored test-time tool.

## Decision: Persist Normalized Oracle Output As Golden Fixtures

Store normalized oracle outputs under `testdata/empirical/golden/` and make empirical tests consume those fixtures by default. Generate fixtures only when missing or explicitly requested by maintainers.

**Rationale**: Persisted fixtures make normal tests deterministic and avoid invoking hledger on every run. Input and output hashes expose drift when the dataset, hledger input, oracle code, or hledger version changes.

**Alternatives considered**:

- Generate oracle results on every test run: rejected because it requires hledger for all empirical runs and increases runtime and platform fragility.
- Never generate in tests: rejected because the spec permits hledger generation when a required fixture is absent.
- Store only human-readable hledger terminal output: rejected because tests need normalized assertable data rather than terminal formatting.

## Decision: Normalize Oracle Output To Project-Owned JSON

Use project-owned JSON golden fixtures with decimal values represented as strings and explicit metadata for version, command, hashes, normalization, method, year, asset, fields, and unsupported cases.

**Rationale**: JSON is better for machine assertions than hledger's terminal output and is easier to hash deterministically than loosely formatted text. Decimal strings preserve exactness and avoid floating-point parsing.

**Alternatives considered**:

- Compare hledger terminal output directly: rejected because terminal output is presentation, not a stable calculation contract.
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

## Decision: Cover Scope-Local Hybrid With hledger-Backed Sub-Evidence Plus Project Composition Rules

For scope-local hybrid, use hledger to validate representable per-scope exact matching and average-cost portions, then apply documented project-owned composition rules for fallback activation, carry-forward until zero, reset after zero, and independent scope state.

**Rationale**: hledger does not model this project's full hybrid lifecycle as one native method. Using hledger for representable subproblems keeps external evidence where possible while avoiding false claims that hledger directly implements the project's hybrid method.

**Alternatives considered**:

- Treat hledger as a native scope-local-hybrid oracle: rejected because that would be inaccurate.
- Exclude scope-local hybrid from empirical testing: rejected because the spec requires every supported method.
- Hand-author all hybrid expected results without hledger evidence: rejected because it weakens the empirical external-oracle objective.

## Decision: Mark Faithfully Unrepresentable Cases As Unsupported For External Comparison

If a dataset case cannot be represented in hledger without changing its financial meaning, the oracle marks that case unsupported for external-oracle comparison with a reason.

**Rationale**: Silent approximation would create false confidence and could hide drift. Explicit unsupported markers preserve dataset coverage while making oracle limits visible.

**Alternatives considered**:

- Fabricate expected values for unsupported hledger cases: rejected because it is not external-oracle evidence.
- Remove unsupported edge cases from the dataset: rejected because the dataset must cover project-supported edge cases.
- Fail the whole suite on any unsupported case: rejected because some project-specific semantics may legitimately lack a direct hledger representation.

## Decision: Compare Quantities Exactly And Financial Values With Documented Tight Tolerances Only When Needed

Quantities compare by exact decimal equality after normalization. Financial fields compare exactly after 16-decimal round-half-up normalization when possible. If hledger cannot align exactly for an otherwise valid case, use a documented per-field tolerance in the fixture contract.

**Rationale**: Quantities should not drift. Financial calculations may involve hledger formatting or precision behavior that does not exactly match the project's internal 16-decimal policy in every valid case, but tolerances must remain explicit and tight enough to detect material drift.

**Alternatives considered**:

- Use broad global tolerance: rejected because it can hide systematic method differences.
- Require exact equality for every financial value regardless of hledger precision behavior: rejected because it could reject valid external evidence when precision policies differ by an immaterial unit.
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
