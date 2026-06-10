# Skipped Empirical Financial Tests Report

**Generated**: 2026-06-10  
**Command used**: `go test ./tests/empirical -count=1 -v`  
**Primary skip source**: `tests/empirical/empirical_calculation_test.go:71-73`, with skip policy in `empiricalCaseComparisonSkipReason` at `tests/empirical/empirical_calculation_test.go:178-190`.

## Summary

The empirical calculation suite currently loads 13 golden fixture subtests. Only 2 calculation fixtures execute comparisons. The other 11 are skipped before project calculation and oracle comparison run.

| Status | Count | Fixture groups |
| --- | ---: | --- |
| Executed | 2 | FIFO standard case, LIFO standard case |
| Skipped | 11 | Average Cost, HIFO, Scope-Local Hybrid, zero-priced FIFO |

This undermines the spec goal because most high-risk calculation methods and edge cases are not empirically compared against their oracle fixtures.

## Runtime Skips

These skips were observed directly from the verbose test run.

| Subtest | Skip reason |
| --- | --- |
| `average_cost/case-average-cost-delta-2024/2024/asset-delta` | `average_cost empirical comparison is skipped because report output does not preserve case-slice pool provenance` |
| `average_cost/case-average-cost-reset-delta-2024/2024/asset-delta` | `average_cost empirical comparison is skipped because report output does not preserve case-slice pool provenance` |
| `average_cost/case-post-year-ignore-delta-2024/2024/asset-delta` | `average_cost empirical comparison is skipped because report output does not preserve case-slice pool provenance` |
| `average_cost/case-zero-priced-delta-2025/2025/asset-delta` | `average_cost empirical comparison is skipped because report output does not preserve case-slice pool provenance` |
| `fifo/case-zero-priced-gamma-2024/2024/asset-gamma` | `zero-priced empirical comparison is skipped because report output does not preserve comparable zero-priced lifecycle provenance` |
| `hifo/case-hifo-gamma-2024/2024/asset-gamma` | `hifo empirical comparison is skipped because persisted oracle precision still differs from calculation-layer financial normalization` |
| `hifo/case-zero-priced-gamma-2024/2024/asset-gamma` | `hifo empirical comparison is skipped because persisted oracle precision still differs from calculation-layer financial normalization` |
| `scope_local_hybrid/case-scope-local-broadening-gamma-2024/2024/asset-delta` | `scope_local_hybrid empirical comparison is skipped because report output does not preserve comparable composition-rule provenance` |
| `scope_local_hybrid/case-scope-local-broadening-gamma-2024/2024/asset-gamma` | `scope_local_hybrid empirical comparison is skipped because report output does not preserve comparable composition-rule provenance` |
| `scope_local_hybrid/case-scope-local-reliable-epsilon-2024/2024/asset-epsilon` | `scope_local_hybrid empirical comparison is skipped because report output does not preserve comparable composition-rule provenance` |
| `scope_local_hybrid/case-scope-local-reset-epsilon-2024/2024/asset-epsilon` | `scope_local_hybrid empirical comparison is skipped because report output does not preserve comparable composition-rule provenance` |

## Reason Groups

### Average Cost Provenance Gap

Affected fixtures: 4.

The current skip policy skips every `average_cost` empirical comparison before calculation runs. The stated reason is that report output does not preserve case-slice pool provenance.

Relevant cases from `testdata/empirical/financial-dataset.yaml`:

| Case | Dataset support | Dataset description |
| --- | --- | --- |
| `case-average-cost-delta-2024` | `supported` | Synthetic average-cost partial disposal with opening history |
| `case-average-cost-reset-delta-2024` | `supported` | Synthetic average-cost full liquidation followed by reacquisition |
| `case-post-year-ignore-delta-2024` | `supported` | Synthetic 2024 report slice that includes after-year activity references for ignore-path review |
| `case-zero-priced-delta-2025` | `supported` | Synthetic zero-priced holding reduction with missing optional upstream source fields |

Detailed implication: the dataset and oracle fixtures mark these cases as available for validation, but the project comparison path cannot isolate the average-cost pool evidence for the selected empirical case slice from the report model output. The test therefore does not verify average-cost calculation output empirically.

Additional fixture metadata for `case-zero-priced-delta-2025` records a narrower oracle limitation: `journal omitted zero-priced reduction handling for emp-act-000140 because lot mode AVERAGE does not support native zero-priced handling`, with comparison policy `project_composition_only`. That metadata is not reached in the current run because the broader average-cost skip happens first.

### HIFO Oracle Precision Gap

Affected fixtures: 2.

The current skip policy skips every `hifo` empirical comparison before calculation runs. The stated reason is that persisted oracle precision still differs from calculation-layer financial normalization.

Relevant cases from `testdata/empirical/financial-dataset.yaml`:

| Case | Dataset support | Dataset description |
| --- | --- | --- |
| `case-hifo-gamma-2024` | `supported` | Synthetic HIFO slice with deterministic tie-breaking evidence |
| `case-zero-priced-gamma-2024` | `supported` | Synthetic zero-priced holding reduction with explicit upstream zero-value semantics |

Detailed implication: the project has golden fixtures for HIFO, but the persisted oracle values and the calculation-layer normalization policy are not aligned closely enough for empirical comparison. The skipped zero-priced HIFO case is classified under this HIFO precision skip because method-level HIFO skipping is checked before zero-priced lifecycle skipping.

### Scope-Local Hybrid Composition Provenance Gap

Affected fixtures: 4.

The current skip policy skips every `scope_local_hybrid` empirical comparison before calculation runs. The stated reason is that report output does not preserve comparable composition-rule provenance.

Relevant cases from `testdata/empirical/financial-dataset.yaml` and golden fixture metadata:

| Case | Asset | Dataset support | Detailed unsupported reason |
| --- | --- | --- | --- |
| `case-scope-local-broadening-gamma-2024` | `asset-delta` | `partially_supported` | Hybrid broadening and fallback activation remain partly project-owned composition rules |
| `case-scope-local-broadening-gamma-2024` | `asset-gamma` | `partially_supported` | Hybrid broadening and fallback activation remain partly project-owned composition rules |
| `case-scope-local-reliable-epsilon-2024` | `asset-epsilon` | `partially_supported` | Hybrid lifecycle composition remains project-owned outside the hledger-backed scope slice |
| `case-scope-local-reset-epsilon-2024` | `asset-epsilon` | `partially_supported` | Hybrid reset and independent-scope assertions require project-owned composition rules |

Detailed implication: the fixtures acknowledge that hledger can back only part of the hybrid behavior. The project-owned composition rules are expected to complete the validation story, but the report output does not currently expose comparable provenance for those composition assertions. The test therefore does not verify scope-local hybrid calculation output empirically.

### Zero-Priced Lifecycle Provenance Gap

Affected fixtures: 1 direct runtime skip.

The FIFO zero-priced case is skipped because report output does not preserve comparable zero-priced lifecycle provenance.

Relevant case from `testdata/empirical/financial-dataset.yaml`:

| Case | Method | Dataset support | Dataset description |
| --- | --- | --- | --- |
| `case-zero-priced-gamma-2024` | `fifo` | `supported` | Synthetic zero-priced holding reduction with explicit upstream zero-value semantics |

Detailed implication: the dataset and oracle fixture are available, but the comparison path cannot prove the project lifecycle handling for zero-priced reductions from report output provenance. The same dataset case also exists for HIFO, but that HIFO subtest is skipped earlier by the HIFO precision rule.

## Executed Empirical Calculation Fixtures

Only these calculation fixtures currently run through project calculation and comparison:

| Subtest | Notes |
| --- | --- |
| `fifo/case-fifo-alpha-2024/2024/asset-alpha` | Standard FIFO liquidation fixture. |
| `lifo/case-lifo-beta-2024/2024/asset-beta` | Standard LIFO fixture with fees and negative yearly total. |

## Additional Comparison-Level Skip Metadata

The comparator supports informational skips through `OracleOutput.UnsupportedSegments` and `EmpiricalComparisonOutcome.Skips`. Current runtime skips happen before `CompareProjectCalculationOutput` is called for the affected fixtures, so this metadata is not surfaced by `TestEmpiricalCalculationFixtures` today.

Golden fixtures that contain unsupported segment metadata:

| Fixture | Policy | Reason |
| --- | --- | --- |
| `testdata/empirical/golden/average-cost/case-zero-priced-delta-2025.json` | `project_composition_only` | journal omitted zero-priced reduction handling for `emp-act-000140` because lot mode AVERAGE does not support native zero-priced handling |
| `testdata/empirical/golden/scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-delta.json` | `project_composition_only` | Hybrid broadening and fallback activation remain partly project-owned composition rules |
| `testdata/empirical/golden/scope-local-hybrid/case-scope-local-broadening-gamma-2024--asset-gamma.json` | `project_composition_only` | Hybrid broadening and fallback activation remain partly project-owned composition rules |
| `testdata/empirical/golden/scope-local-hybrid/case-scope-local-reliable-epsilon-2024.json` | `project_composition_only` | Hybrid lifecycle composition remains project-owned outside the hledger-backed scope slice |
| `testdata/empirical/golden/scope-local-hybrid/case-scope-local-reset-epsilon-2024.json` | `project_composition_only` | Hybrid reset and independent-scope assertions require project-owned composition rules |

## Latent Match-Evidence Reduction

`TestEmpiricalCalculationFixtures` also contains a comparison reduction path at `tests/empirical/empirical_calculation_test.go:92-95`: when `shouldSkipCaseMatchEvidence` returns true, both expected and actual match evidence are set to `nil` before comparison.

That currently applies to average-cost, scope-local hybrid, and zero-priced cases by policy. Because those cases are already skipped earlier, this reduction is not the current observed runtime skip mechanism. If the top-level skips are removed, match evidence may still be silently excluded unless the report model exposes comparable provenance.

## Planned Test Fixes

### Remove Zero-Priced Holding Reductions From Empirical Oracle Scope

Decision: remove empirical cases that contain zero-priced holding reductions from the external-oracle dataset, golden fixtures, generated hledger journals, and empirical covered-case expectations.

Reason: zero-priced holding reduction is a project-specific lifecycle rule. The purpose of this empirical oracle suite is to verify calculation correctness and precision against an external financial oracle. When the rule cannot be represented faithfully by the external oracle, keeping it in this suite causes skip paths instead of useful empirical validation.

Affected empirical cases and fixtures:

| Case | Method fixtures | Current skip or limitation |
| --- | --- | --- |
| `case-zero-priced-gamma-2024` | `fifo`, `hifo` | FIFO is skipped for missing zero-priced lifecycle provenance; HIFO is skipped by the broader HIFO precision rule before the zero-priced limitation is reached. |
| `case-zero-priced-delta-2025` | `average_cost` | Average Cost is skipped by the broader average-cost provenance rule; fixture metadata also says hledger lot mode AVERAGE does not support native zero-priced handling for `emp-act-000140`. |

Required cleanup when applying this fix:

1. Remove the affected case definitions from `testdata/empirical/financial-dataset.yaml`.
2. Remove dedicated zero-priced reduction coverage tags from empirical coverage expectations when they are only satisfied by these cases.
3. Remove affected golden fixtures under `testdata/empirical/golden/`.
4. Remove affected generated hledger journals under `testdata/empirical/hledger/`.
5. Keep zero-priced holding reduction behavior covered in the existing traditional test suites, such as unit, integration, or contract tests that validate project-specific calculation rules without requiring an external oracle.

Expected result: zero-priced holding reductions stop contributing empirical-oracle skips. The empirical suite remains focused on externally verifiable calculation correctness and precision, while project-specific zero-priced behavior remains covered by non-oracle tests.

## Current Blocking Themes

1. Report output lacks provenance needed to compare method-specific lifecycle slices for average-cost, scope-local hybrid, and zero-priced reductions.
2. HIFO oracle fixtures have persisted precision that does not yet match calculation-layer financial normalization.
3. Fixture-level unsupported segment metadata exists, but the main empirical calculation test does not currently surface it because broad method-level skips short-circuit earlier.
