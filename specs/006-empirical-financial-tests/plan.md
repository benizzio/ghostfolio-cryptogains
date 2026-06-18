# Implementation Plan: Empirical Solidified Financial Tests

**Branch**: `[006-empirical-financial-tests]` | **Date**: 2026-06-05 | **Spec**: `/specs/006-empirical-financial-tests/spec.md`

**Input**: Feature specification from `/specs/006-empirical-financial-tests/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

**Bugfix**: 2026-06-10 — [BUG-001] Updated from bugfix patch.

**Bugfix**: 2026-06-12 — [BUG-002] Updated from bugfix patch.

**Bugfix**: 2026-06-13 — [BUG-003] Updated from bugfix patch.

## Summary

Create internal empirical financial validation infrastructure for the report calculation layer. The implementation introduces a synthetic empirical external dataset, a ~~repository-vendored hledger command-line oracle~~ pinned rotki-backed test-time oracle adapter for FIFO, LIFO, HIFO, and Average Cost aggregate fixtures, a separate Scope-Local Hybrid composite oracle, normalized golden fixtures, and an isolated `tests/empirical` Go integration package that translates the same dataset into this project's calculation inputs and compares pure calculated report results against the oracle output.

The scope deliberately excludes user-facing TUI, Ghostfolio transport, snapshot encryption, Markdown rendering, and report output formatting. ~~hledger is used only as a test-time external command~~ External oracle tooling is used only through test-time boundaries when a required golden fixture is absent or when maintainers explicitly regenerate fixtures. Runtime application code must not link, import, or call ~~hledger,~~ rotki, oracle adapters, or composite oracle helpers.

BUG-001 supersedes the hledger-only oracle direction because the observed fixture-backed empirical run skipped 11 of 13 fixture groups before project calculation and oracle comparison.

BUG-002 supersedes the BUG-001 bootstrap shortcut that accepted committed raw rotki outputs or vendored rotki source as oracle evidence. Explicit golden-fixture regeneration must download or reuse verified pinned rotki source in an untracked project-local folder, execute rotki calculation code through the local adapter boundary, and commit only project-owned normalized golden fixtures plus provenance metadata. Normal fixture-backed empirical tests must not download rotki or invoke oracle generation.

BUG-003 supersedes retained hledger boundary planning. Hledger source, executables, generated journals, command wrappers, normalization paths, hledger-specific tests, fixture metadata, and active documentation references are removal targets; rotki remains the sole external oracle boundary for FIFO, LIFO, HIFO, and Average Cost aggregate regeneration, and the Scope-Local Hybrid composite oracle remains the only non-pure-method oracle path.

## Technical Context

**Language/Version**: Go 1.26.3 for project-owned test infrastructure and oracle normalization code. ~~Vendored hledger remains an external GPL-3.0-or-later command-line tool~~ External oracle materials remain test-time only and are not imported into Go runtime code.

**Primary Dependencies**: Existing Go standard library packages, `github.com/cockroachdb/apd/v3` for exact decimal parsing and comparison, existing report calculation packages under `internal/report`, existing sync models under `internal/sync/model`, repository-controlled rotki source provenance, pinned version or commit, signed-tag verification metadata, adapter constraints, checksums, verified untracked project-local rotki source acquisition for explicit regeneration, the project-owned Python adapter at `tools/empiricaloracle/rotki_adapter.py`, local `python3` or `python` for explicit regeneration, local `git` for remote tag verification during explicit regeneration, ~~and any retained hledger materials for test-time journal generation~~. Hledger materials are out of active scope after BUG-003. No new Go module dependency is planned.

**Storage**: Repository fixtures plus untracked regeneration cache paths. The synthetic dataset, ~~retained generated hledger journals under `testdata/empirical/hledger/`,~~ normalized oracle golden fixtures under `testdata/empirical/golden/`, and external-oracle provenance metadata under `third_party/` are committed under repository-controlled paths. Generated rotki adapter inputs under `.cache/empiricaloracle/oracle-inputs/` and verified rotki source artifacts under `.cache/empiricaloracle/rotki-source/` are untracked explicit-regeneration cache paths and must not be committed. BUG-003 makes `testdata/empirical/hledger/` and `third_party/hledger/` removal targets, not retained fixture or dependency paths. No empirical artifact is written to protected snapshot storage, `setup.json`, user Documents, OS application config, or telemetry. Golden fixtures are synthetic and reproducible from dataset plus oracle metadata.

**Testing**: Go standard `testing`, targeted `go test ./tests/empirical`, full `make test`, full `make coverage`, dataset structural validation, oracle fixture validation, project calculation comparison across all supported cost-basis methods, acceptance failure for unexpected supported fixture skips, OWASP Top 10 review evidence, and secret scanning or fixture-content review for synthetic-only data.

**Empirical Dataset**: New synthetic human-readable dataset at `testdata/empirical/financial-dataset.yaml` with generated normalized golden fixtures under `testdata/empirical/golden/`. This is a dataset-maintenance spec, so dataset creation is allowed here. After this work lands, the dataset becomes read-only for ordinary feature work and may be changed only by later isolated dataset-maintenance specs.

**Target Platform**: Local development and CI test environment for the Go module. External oracle generation is required only where the documented repository-controlled oracle boundary and verified untracked rotki source acquisition path are available and only when a golden fixture is absent or regeneration is explicitly requested. Explicit rotki regeneration also requires local `git` plus `python3` or `python` to verify the pinned tag and execute the project-owned adapter. Existing fixtures allow empirical tests to run without invoking external oracle generation or downloading rotki source.

**Project Type**: Single-module Go terminal application with internal empirical integration tests.

**Performance Goals**: Empirical tests should run in a targeted command without Ghostfolio, TUI, snapshot encryption, Markdown rendering, or filesystem report output. Dataset validation plus fixture-backed comparisons should complete in 30 seconds or less on a normal local development environment when golden fixtures are present. ~~hledger generation~~ External oracle generation has no fixed runtime target and is limited to absent or explicitly regenerated fixtures.

**Constraints**: All dataset records are synthetic. No real tokens, JWTs, user activity, account names, wallet names, proprietary financial records, or copied upstream fixture rows are allowed. Quantities compare by exact decimal equality after normalization under the selected decimal policy. Financial calculated values first align ~~hledger~~ external-oracle and project output under the selected decimal policy, then compare using documented per-field tolerances for residual external-oracle deviations. Quantity tolerance is always `0`. Non-zero financial tolerance must be declared per field in oracle metadata, must not exceed one unit at the selected decimal-policy scale (`0.0000000000000001` for the 16-decimal policy), and must include a fixture note explaining why exact equality is not achievable for that external-oracle-derived value. The default selected policy is the project's production 16-decimal round-half-up policy, which also remains the current empirical default. If ~~hledger~~ the selected external oracle cannot be configured or normalized to match that policy for every valid case, `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` is available as an application-run-scoped override to use the external oracle's established decimal policy for that run. Accepted custom values use `scale=<digits>,rounding=half_up` and should stay at or below a practical maximum scale of 64 for safety. Cross-currency conversion is permanently out of scope; all empirical cases use one currency. ~~hledger vendoring must include GPL-3.0-or-later license text, complete corresponding source under `third_party/hledger/source/`, supported executable artifacts under `third_party/hledger/bin/<goos>-<goarch>/hledger`, upstream URL, version identity, checksums, and platform support notes.~~ External oracle materials must include applicable license text, source provenance, pinned version or commit identity, checksums, adapter constraints, and platform support notes. Runtime application code must not depend on ~~hledger,~~ rotki, or oracle adapters.

BUG-001 edge-case constraints: zero-priced holding reductions are removed from empirical external-oracle fixture scope and must remain covered by non-oracle unit, integration, or contract tests. Supported empirical fixture groups must not be skipped before project calculation and oracle comparison.

BUG-001 dependency constraint: the current checkout cannot rely on a developer-local rotki installation as verifiable acceptance evidence. ~~Phase 6 must first land repository-controlled rotki materials or committed raw rotki outputs with provenance before adapter completion and fixture regeneration are treated as done.~~ BUG-002 supersedes this shortcut: Phase 6 must prove verified untracked pinned rotki source execution before adapter completion and fixture regeneration are treated as done.

BUG-002 dependency constraint: committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, and vendored rotki source are not acceptable oracle evidence. Explicit regeneration must verify and execute pinned rotki source from an untracked project-local folder. Normal fixture-backed tests must fail if they attempt network download or oracle generation while committed golden fixtures are present.

BUG-003 dependency constraint: hledger is not retained as a historical or auxiliary boundary. Active feature implementation must remove hledger vendoring, generated journals, command wrappers, normalization and provenance paths, hledger-specific tests, fixture labels, and active documentation references except historical bug reports or explicit strikethrough rationale.

**Scale/Scope**: At least 150 synthetic activities across at least 3 source-calendar years, all five supported cost-basis methods, required edge cases from `spec.md`, at least one selected report year, dataset validation, oracle fixture generation or reuse, and isolated empirical comparisons in `tests/empirical`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS
Post-design gate status: PASS

- [x] Security: This is internal test infrastructure with synthetic data only. The empirical dataset, ~~hledger journals~~ external-oracle inputs, and golden fixtures must not include real user activity, real account names, wallet names, tokens, JWTs, protected snapshot payloads, proprietary financial records, or copied upstream examples. No artifact is written to protected app storage, user Documents, or OS-specific application config. Token handling is not exercised by the empirical suite. Redaction review focuses on failure output, fixture content, and generated artifacts. OWASP Top 10 review evidence must be recorded before merge. OWASP review scope for implementation covers cryptographic failures only to confirm the suite does not touch protected storage, identification and authentication failures only to confirm no token boundary is involved, insecure design, vulnerable or outdated components, software and data integrity failures, and logging or diagnostic leakage.
- [x] Precision: Project-owned dataset parsing, oracle normalization, and comparisons use decimal strings and `apd.Decimal`; no floating-point math is allowed. The dataset uses one explicit currency for every priced empirical case. No cross-currency conversion or exchange-rate lookup is introduced. The comparison contract uses exact decimal equality for quantities after selected decimal-policy normalization. Financial values are normalized under the selected decimal policy first and then may use documented per-field tolerances for residual ~~hledger/project~~ external-oracle/project deviations. Quantity tolerance is `0`, and any non-zero financial tolerance must not exceed one unit at the selected decimal-policy scale. The default policy is the production 16-decimal round-half-up policy, and current empirical runs use that same default. If ~~hledger~~ the selected external oracle cannot align with that policy for every valid case, `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` provides an application-run-scoped override to the external-oracle-established policy. Accepted custom values use `scale=<digits>,rounding=half_up` and should stay at or below a practical maximum scale of 64 for safety.
- [x] Testing: The feature introduces supplemental integration-style Go tests in `tests/empirical`. They do not replace existing contract, integration, unit, coverage, or performance verification. Coverage verification remains `make coverage`; empirical tests are targeted validation and may be included in `make test` only when they do not require ~~hledger~~ external oracle generation. Unit tests are justified for dataset parsing, schema validation, oracle normalization, comparison formatting, fixture hashing, and external oracle adapter wrapping because those units have deterministic edge cases and failure modes independent of report calculation.
- [x] Empirical financial validation: This spec introduces the empirical suite. The dataset path is `testdata/empirical/financial-dataset.yaml`; golden oracle output path is `testdata/empirical/golden/`; ~~retained generated hledger journal path is `testdata/empirical/hledger/`;~~ generated rotki adapter-input path is `.cache/empiricaloracle/oracle-inputs/`; and verified rotki source-manifest path is `.cache/empiricaloracle/rotki-source/verified-source.json`. BUG-003 removes `testdata/empirical/hledger/` from active scope. This active spec is explicitly dataset-maintenance work, so dataset creation and fixture generation are allowed. After this work completes, ordinary implementation work must treat the empirical external dataset as read-only and change it only through a later isolated dataset-maintenance spec.
- [x] Dependencies and external integrations: ~~hledger is the only new external tool and remains a repository-vendored test-time command. It is justified because the feature requires an external plain-text-accounting oracle and hledger supports lot/cost-basis workflows relevant to FIFO, LIFO, HIFO, average-cost, and gain reporting. hledger is GPL-3.0-or-later, compatible with this GPLv3 repository when source, license notices, URL, version, and checksums are included. The vendoring model is complete corresponding source under `third_party/hledger/source/` plus supported executable artifacts under `third_party/hledger/bin/<goos>-<goarch>/hledger`.~~ BUG-001 replaces hledger-only oracle acceptance with a pinned rotki-based test-time oracle adapter for FIFO, LIFO, HIFO, and Average Cost aggregate comparisons, plus a Scope-Local Hybrid composite oracle. BUG-003 removes retained hledger materials from active feature scope. External oracle license compatibility, source provenance, version or commit identity, adapter constraints, checksums, and platform support must be documented. External oracle code is not a Go module dependency, not linked, not imported, and not used by runtime code.
- [x] Architecture: Dataset schema, oracle generation, oracle normalization, comparison contracts, and tests stay under test-data, test-helper, or `tests/empirical` boundaries. Existing report calculation stays in `internal/report/calculate` and `internal/report/basis`. No Ghostfolio DTO, TUI, snapshot, Markdown, or runtime orchestration behavior is pulled into empirical tests except the report calculation model boundary required to execute pure calculation.

## Project Structure

### Documentation (this feature)

```text
specs/006-empirical-financial-tests/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── dataset-format.md
│   ├── oracle-output.md
│   └── empirical-tests.md
└── tasks.md
```

### Source Code (repository root)

```text
internal/
├── report/
│   ├── basis/
│   ├── calculate/
│   └── model/
├── sync/
│   └── model/
└── support/
    ├── decimal/
    └── math/

tests/
├── empirical/
├── integration/
├── testutil/
└── unit/

testdata/
└── empirical/
    ├── financial-dataset.yaml
    ├── golden/
    └── README.md

tools/
└── empiricaloracle/

.cache/
└── empiricaloracle/
    ├── oracle-inputs/
    └── rotki-source/

third_party/
└── rotki/
    ├── LICENSE.md
    └── README.md
```

BUG-002 supersedes `third_party/rotki/source/` and committed raw rotki fixture paths: rotki source must be downloaded or reused in an untracked project-local folder during explicit regeneration, and that folder must be covered by `.gitignore`.

BUG-003 supersedes `testdata/empirical/hledger/` and `third_party/hledger/`: hledger fixtures, source, executables, metadata, and command or journal tooling are removal targets, not retained historical scope.

**Structure Decision**: Keep the application as one Go module. Add isolated empirical validation under `tests/empirical`, synthetic fixture material under `testdata/empirical`, and a project-owned oracle helper under `tools/empiricaloracle`. ~~Retain hledger materials under `third_party/hledger` only for the auxiliary historical boundary and generated journal artifacts.~~ BUG-003 removes hledger from active scope; do not retain `third_party/hledger`, `testdata/empirical/hledger`, hledger command wrappers, or journal fixtures. Store rotki provenance docs under `third_party/rotki`, generated rotki adapter inputs under `.cache/empiricaloracle/oracle-inputs/`, and verified rotki source artifacts under `.cache/empiricaloracle/rotki-source/`. Do not vendor rotki source or treat committed raw rotki outputs as oracle evidence. Do not put empirical oracle code in `internal/report`, because report packages own production calculation behavior and should not know about ~~hledger,~~ rotki, oracle adapters, or fixture generation.

## Empirical Design

- `testdata/empirical/financial-dataset.yaml` is the canonical synthetic source ledger. It uses a simplified project-owned schema, not copied external-oracle fixture text.
- Dataset rows carry stable source IDs, timestamps, deterministic order, asset identity keys, display labels, activity type, quantity, monetary fields, one currency, fees, source scope, zero-priced reduction explanation, and coverage tags.
- Dataset validation fails if there are fewer than 150 activities, fewer than 3 source-calendar years, missing required method tags, missing required edge-case tags, duplicate source IDs, unstable ordering fields, non-decimal numeric text, missing single-currency identity on priced rows, or real-looking secret material.
- The oracle helper converts supported pure-method cases into generated JSON adapter inputs under `.cache/empiricaloracle/oracle-inputs/`, executes the project-owned Python adapter `tools/empiricaloracle/rotki_adapter.py` against verified pinned rotki source only when fixture generation is required, verifies pinned rotki source in an untracked project-local folder for explicit regeneration, captures external oracle source provenance, version or commit identity, signed-tag verification, adapter arguments, and normalizes results to project-owned JSON fixtures.
- BUG-001 remediation bootstrap ~~stores repository-controlled rotki boundary evidence under `third_party/rotki/` and `testdata/empirical/rotki/` so adapter work and fixture normalization do not depend on unverifiable local installations~~ is superseded by BUG-002: adapter work and fixture normalization must execute verified pinned rotki source from an untracked project-local folder and must not use committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, or vendored rotki source as oracle evidence.
- Golden fixtures include input hash, ~~hledger input hash, hledger version, command arguments~~ external-oracle input hash, external oracle name, version or commit identity, adapter arguments, oracle output hash, normalization version, method, year, asset, realized gain or loss, allocated basis, closing quantity, closing basis, and method-specific evidence where available.
- Comparable oracle fields are identified by case, method, year, asset, source-row segment, expected value, tolerance, and support status. Full-liquidation and method-specific evidence are compared only when the fixture records the evidence source IDs and expected values. Current Scope-Local Hybrid fixtures compare `rotki_backed` arithmetic evidence and persist the remaining project-owned routing or lifecycle segments as `project_composition_only` unsupported segments with documented reasons.
- Unsupported ~~hledger~~ external-oracle representations are explicit. A dataset case that the selected external oracle cannot represent faithfully is marked unsupported for external-oracle comparison with a documented reason instead of being silently converted to a different financial case.
- The Scope-Local Hybrid (`scope_local_hybrid`) method uses ~~hledger-backed per-scope exact-match or average-cost evidence where hledger can model the subproblem~~ rotki-backed arithmetic sub-oracle evidence where valid, plus project-owned documented composition rules for the hybrid lifecycle that no single external oracle provides as one native method.
- Empirical tests read golden fixtures by default. They invoke ~~hledger~~ external oracle generation only when the required fixture is absent or when an explicit regeneration command is used. Fixture-backed runs must not download rotki source, require rotki source availability, or invoke oracle generation.
- Zero-priced holding reductions are excluded from empirical external-oracle fixture scope after BUG-001 and remain covered by non-oracle unit, integration, or contract tests.
- Supported empirical fixture groups must execute project calculation and oracle comparison; broad top-level skips for supported fixtures are acceptance failures.
- Empirical tests translate dataset rows into `syncmodel.ProtectedActivityCache` and `reportmodel.ActivityCalculationInput` equivalents, call `calculate.Calculate`, normalize `reportmodel.CapitalGainsReport`, and compare against `OracleOutput`.
- Failure output identifies dataset case, method, asset, year, field, selected decimal policy, expected value, actual value, difference, tolerance, and source row references. It must not include tokens, real user data, raw protected snapshot content, Markdown, TUI text, output file paths, or report document content.

## Verification Plan

- Dataset validation test proves minimum activity count, year span, required coverage tags, unique IDs, deterministic ordering, single currency, synthetic-only fields, ~~zero-priced reduction explanations~~ zero-priced external-oracle exclusion after BUG-001, and no real-secret patterns.
- Oracle metadata tests prove external oracle name, source URL, version or commit identity, signed-tag verification inputs, adapter arguments, dataset input hash, external-oracle input hash, oracle output hash, normalization version, unsupported-case reasons, and fixture reproducibility metadata are present.
- Empirical comparison tests cover FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Hybrid across every supported golden fixture and fail on unexpected supported fixture skips.
- Precision tests prove exact equality for quantities, 16-decimal round-half-up normalization by default, ~~hledger-aligned~~ external-oracle-aligned application-run-scoped decimal-policy override through `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` when needed, unchanged production and current empirical default behavior when the variable is unset, documented per-field financial tolerances capped at one unit of selected decimal-policy scale for residual deviations, and no floating-point use in parser, normalizer, or comparator code.
- Zero-priced reduction tests prove quantity and basis reduction with zero or absent proceeds, realized gain, and realized loss through non-oracle unit, integration, or contract coverage rather than empirical external-oracle fixtures.
- Isolation tests prove empirical tests do not call Ghostfolio, TUI renderers, protected snapshot encryption, Markdown rendering, report output writers, or OS opener code.
- Dependency tests or review checks prove rotki provenance and adapter materials include applicable license text, source provenance, supported artifact paths, upstream URL, version or commit identity, checksums, platform support, adapter constraints, no prohibited binary-only vendoring, no committed rotki source checkout, and the documented local Python adapter boundary.
- Regeneration-boundary tests prove normal `go test ./tests/empirical -count=1 -v` runs do not download rotki or invoke oracle generation while committed golden fixtures are present, and the explicit regeneration command downloads or reuses only the verified untracked rotki source path, validates the pinned tag with `git ls-remote`, and executes the local Python adapter boundary.
- BUG-003 cleanup verification fails on active non-historical `hledger` or `hleger` references in source, tests, fixtures, active contracts, quickstart, README files, and oracle-output documentation after hledger removal tasks complete.
- Security review checks record OWASP Top 10 review evidence for this empirical test infrastructure before merge.
- Empirical runtime checks record or assert that fixture-backed `go test ./tests/empirical -count=1 -v` completes within the 30-second local-development target when golden fixtures are present.
- Standard verification remains `make test` and `make coverage`; targeted empirical verification is `go test ./tests/empirical -count=1 -v`.

## Complexity Tracking

No constitution violations require justification for this plan.

BUG-001 adds dependency-replacement complexity: hledger-only fixture acceptance is superseded by rotki-backed pure-method fixtures, a separate Scope-Local Hybrid composite oracle, zero-priced external-oracle exclusion, a skip-fail acceptance rule for supported fixture groups, and a repository-controlled rotki-boundary bootstrap because the current checkout does not contain verifiable rotki materials yet.

BUG-002 adds source-acquisition complexity: the rotki-backed oracle boundary must separate normal fixture-backed tests from explicit regeneration, verify pinned rotki source in an untracked project-local folder, reject committed raw rotki outputs or vendored rotki source as oracle evidence, and preserve deterministic committed golden fixtures with provenance metadata.

BUG-003 adds cleanup complexity: prior accepted hledger setup, fixtures, command wrappers, tests, and documentation must be reopened and removed from active scope without weakening the rotki regeneration boundary or Scope-Local Hybrid composite-oracle path.
