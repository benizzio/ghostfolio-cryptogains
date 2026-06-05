# Implementation Plan: Empirical Solidified Financial Tests

**Branch**: `[006-empirical-financial-tests]` | **Date**: 2026-06-05 | **Spec**: `/specs/006-empirical-financial-tests/spec.md`

**Input**: Feature specification from `/specs/006-empirical-financial-tests/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Create internal empirical financial validation infrastructure for the report calculation layer. The implementation introduces a synthetic empirical external dataset, a repository-vendored hledger command-line oracle that produces normalized golden fixtures, and an isolated `tests/empirical` Go integration package that translates the same dataset into this project's calculation inputs and compares pure calculated report results against the oracle output.

The scope deliberately excludes user-facing TUI, Ghostfolio transport, snapshot encryption, Markdown rendering, and report output formatting. hledger is used only as a test-time external command when a required golden fixture is absent or when maintainers explicitly regenerate fixtures. Runtime application code must not link, import, or call hledger.

## Technical Context

**Language/Version**: Go 1.26.3 for project-owned test infrastructure and oracle normalization code. Vendored hledger remains an external GPL-3.0-or-later command-line tool and is not imported into Go runtime code.

**Primary Dependencies**: Existing Go standard library packages, `github.com/cockroachdb/apd/v3` for exact decimal parsing and comparison, existing report calculation packages under `internal/report`, existing sync models under `internal/sync/model`, and repository-vendored hledger source or corresponding source materials for test-time oracle generation. No new Go module dependency is planned.

**Storage**: Repository fixtures only. The synthetic dataset, generated hledger journal inputs, normalized oracle golden fixtures, and hledger vendoring metadata are committed under repository-controlled paths. No empirical artifact is written to protected snapshot storage, `setup.json`, user Documents, OS application config, or telemetry. Golden fixtures are synthetic and reproducible from dataset plus oracle metadata.

**Testing**: Go standard `testing`, targeted `go test ./tests/empirical`, full `make test`, full `make coverage`, dataset structural validation, oracle fixture validation, project calculation comparison across all supported cost-basis methods, and secret scanning or fixture-content review for synthetic-only data.

**Empirical Dataset**: New synthetic human-readable dataset at `testdata/empirical/financial-dataset.yaml` with generated normalized golden fixtures under `testdata/empirical/golden/`. This is a dataset-maintenance spec, so dataset creation is allowed here. After this work lands, the dataset becomes read-only for ordinary feature work and may be changed only by later isolated dataset-maintenance specs.

**Target Platform**: Local development and CI test environment for the Go module. hledger execution is required only on platforms where the repository-vendored hledger command is supported and only when a golden fixture is absent or regeneration is explicitly requested. Existing fixtures allow empirical tests to run without invoking hledger.

**Project Type**: Single-module Go terminal application with internal empirical integration tests.

**Performance Goals**: Empirical tests should run in a targeted command without Ghostfolio, TUI, snapshot encryption, Markdown rendering, or filesystem report output. Dataset validation plus fixture-backed comparisons should remain suitable for normal local development runs; hledger generation may be slower and is limited to absent or explicitly regenerated fixtures.

**Constraints**: All dataset records are synthetic. No real tokens, JWTs, user activity, account names, wallet names, proprietary financial records, or copied upstream fixture rows are allowed. Quantities compare by exact decimal equality after normalization under the selected decimal policy. Financial calculated values first align hledger and project output under the selected decimal policy, then compare using documented tight per-field tolerances for residual external-oracle deviations. The default selected policy is the project's production 16-decimal round-half-up policy. If hledger cannot be configured or normalized to match that policy for every valid case, empirical tests set the test-scoped `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` environment variable to hledger's established decimal policy while production behavior keeps the 16-decimal default when the variable is unset. Cross-currency conversion is permanently out of scope; all empirical cases use one currency. hledger vendoring must include GPL-3.0-or-later license text, source or complete corresponding source for any executable artifact, upstream URL, version identity, checksum, and platform support notes. Runtime application code must not depend on hledger.

**Scale/Scope**: At least 150 synthetic activities across at least 3 source-calendar years, all five supported cost-basis methods, required edge cases from `spec.md`, at least one selected report year, dataset validation, oracle fixture generation or reuse, and isolated empirical comparisons in `tests/empirical`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: This is internal test infrastructure with synthetic data only. The empirical dataset, hledger journals, and golden fixtures must not include real user activity, real account names, wallet names, tokens, JWTs, protected snapshot payloads, proprietary financial records, or copied upstream examples. No artifact is written to protected app storage, user Documents, or OS-specific application config. Token handling is not exercised by the empirical suite. Redaction review focuses on failure output, fixture content, and generated artifacts. OWASP review scope for implementation covers cryptographic failures only to confirm the suite does not touch protected storage, identification and authentication failures only to confirm no token boundary is involved, insecure design, vulnerable or outdated components, software and data integrity failures, and logging or diagnostic leakage.
- [x] Precision: Project-owned dataset parsing, oracle normalization, and comparisons use decimal strings and `apd.Decimal`; no floating-point math is allowed. The dataset uses one explicit currency for every priced empirical case. No cross-currency conversion or exchange-rate lookup is introduced. The comparison contract uses exact decimal equality for quantities after selected decimal-policy normalization. Financial values are normalized under the selected decimal policy first and then may use documented tight per-field tolerances for residual hledger/project deviations. The default policy is the production 16-decimal round-half-up policy. If hledger cannot align with that policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to the hledger-established policy, and production behavior must remain unchanged when the variable is unset.
- [x] Testing: The feature introduces supplemental integration-style Go tests in `tests/empirical`. They do not replace existing contract, integration, unit, coverage, or performance verification. Coverage verification remains `make coverage`; empirical tests are targeted validation and may be included in `make test` only when they do not require hledger generation. Unit tests are justified for dataset parsing, schema validation, oracle normalization, comparison formatting, fixture hashing, and hledger command wrapping because those units have deterministic edge cases and failure modes independent of report calculation.
- [x] Empirical financial validation: This spec introduces the empirical suite. The dataset path is `testdata/empirical/financial-dataset.yaml`; golden oracle output path is `testdata/empirical/golden/`; generated hledger input path is `testdata/empirical/hledger/`. This active spec is explicitly dataset-maintenance work, so dataset creation and fixture generation are allowed. After this work completes, ordinary implementation work must treat the empirical external dataset as read-only and change it only through a later isolated dataset-maintenance spec.
- [x] Dependencies and external integrations: hledger is the only new external tool and remains a repository-vendored test-time command. It is justified because the feature requires an external plain-text-accounting oracle and hledger supports lot/cost-basis workflows relevant to FIFO, LIFO, HIFO, average-cost, and gain reporting. hledger is GPL-3.0-or-later, compatible with this GPLv3 repository when source, license notices, URL, version, and checksum are included. It is not a Go module dependency, not linked, not imported, and not used by runtime code.
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
    ├── hledger/
    └── README.md

tools/
└── empiricaloracle/

third_party/
└── hledger/
```

**Structure Decision**: Keep the application as one Go module. Add isolated empirical validation under `tests/empirical`, synthetic fixture material under `testdata/empirical`, and a project-owned oracle helper under `tools/empiricaloracle`. Vendor hledger materials under `third_party/hledger` with license and source metadata. Do not put empirical oracle code in `internal/report`, because report packages own production calculation behavior and should not know about hledger or fixture generation.

## Empirical Design

- `testdata/empirical/financial-dataset.yaml` is the canonical synthetic source ledger. It uses a simplified project-owned schema, not copied hledger, Ledger, or Beancount fixture text.
- Dataset rows carry stable source IDs, timestamps, deterministic order, asset identity keys, display labels, activity type, quantity, monetary fields, one currency, fees, source scope, zero-priced reduction explanation, and coverage tags.
- Dataset validation fails if there are fewer than 150 activities, fewer than 3 source-calendar years, missing required method tags, missing required edge-case tags, duplicate source IDs, unstable ordering fields, non-decimal numeric text, missing single-currency identity on priced rows, or real-looking secret material.
- The oracle helper converts dataset cases into hledger-compatible journal files per method or method family, invokes the repository-vendored hledger command only when fixture generation is required, captures hledger version and command arguments, and normalizes results to project-owned JSON fixtures.
- Golden fixtures include input hash, hledger input hash, hledger version, command arguments, oracle output hash, normalization version, method, year, asset, realized gain or loss, allocated basis, closing quantity, closing basis, and method-specific evidence where available.
- Unsupported hledger representations are explicit. A dataset case that hledger cannot represent faithfully is marked unsupported for external-oracle comparison with a documented reason instead of being silently converted to a different financial case.
- The scope-local hybrid method uses hledger-backed per-scope exact-match or average-cost evidence where hledger can model the subproblem, plus project-owned documented composition rules for the hybrid lifecycle that hledger does not provide as one native method.
- Empirical tests read golden fixtures by default. They invoke hledger generation only when the required fixture is absent or when an explicit regeneration command is used.
- Empirical tests translate dataset rows into `syncmodel.ProtectedActivityCache` and `reportmodel.ActivityCalculationInput` equivalents, call `calculate.Calculate`, normalize `reportmodel.CapitalGainsReport`, and compare against `OracleOutput`.
- Failure output identifies dataset case, method, asset, year, field, selected decimal policy, expected value, actual value, difference, tolerance, and source row references. It must not include tokens, real user data, raw protected snapshot content, Markdown, TUI text, output file paths, or report document content.

## Verification Plan

- Dataset validation test proves minimum activity count, year span, required coverage tags, unique IDs, deterministic ordering, single currency, synthetic-only fields, zero-priced reduction explanations, and no real-secret patterns.
- Oracle metadata tests prove hledger version, command arguments, dataset input hash, hledger input hash, oracle output hash, normalization version, unsupported-case reasons, and fixture reproducibility metadata are present.
- Empirical comparison tests cover FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Hybrid across every supported golden fixture.
- Precision tests prove exact equality for quantities, 16-decimal round-half-up normalization by default, hledger-aligned decimal-policy configuration through `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` when needed, unchanged production default behavior when the variable is unset, documented tight per-field financial tolerances for residual deviations, and no floating-point use in parser, normalizer, or comparator code.
- Zero-priced reduction tests prove quantity and basis reduction with zero or absent proceeds, realized gain, and realized loss according to the normalized comparison contract.
- Isolation tests prove empirical tests do not call Ghostfolio, TUI renderers, protected snapshot encryption, Markdown rendering, report output writers, or OS opener code.
- Dependency tests or review checks prove hledger vendoring includes GPL-3.0-or-later license text, source or complete corresponding source, upstream URL, version, checksum, platform support, and no binary-only vendoring.
- Standard verification remains `make test` and `make coverage`; targeted empirical verification is `go test ./tests/empirical -count=1 -v`.

## Complexity Tracking

No constitution violations require justification for this plan.
