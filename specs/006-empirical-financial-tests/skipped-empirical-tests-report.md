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

### Replace hledger With Rotki as the Empirical Oracle

Decision: replace hledger with a test-time rotki-based oracle adapter for empirical methods that rotki can model: `fifo`, `lifo`, `hifo`, and `average_cost` through rotki ACB.

Reason: hledger cannot currently produce faithful fixtures for this project's moving weighted-average pool. Disposable experiments showed both `lots: AVERAGE` and `lots: AVERAGEALL` differ materially from project average-cost output. A disposable rotki prototype matched project aggregate output for FIFO, LIFO, HIFO, and ACB within the configured tolerance.

Prototype evidence:

| Method | Case | Result | Notes |
| --- | --- | --- | --- |
| `fifo` | `case-fifo-alpha-2024` | Pass | Aggregate values matched within tolerance. |
| `lifo` | `case-lifo-beta-2024` | Pass | Aggregate values matched within tolerance. |
| `hifo` | `case-hifo-gamma-2024` | Pass | Aggregate values matched within tolerance; add a focused deterministic tie-break check before relying on HIFO match provenance. |
| `average_cost` / rotki ACB | `case-average-cost-delta-2024` | Pass | Aggregate values matched within tolerance. |
| `average_cost` / rotki ACB | `case-average-cost-reset-delta-2024` | Pass | Aggregate values matched within tolerance. |

Prototype tolerance standard:

| Field type | Required tolerance | Observed prototype result |
| --- | --- | --- |
| Quantity fields | Exact `0` difference | Passed for tested cases. |
| Financial fields | Difference no greater than `0.0000000000000001` | Passed for tested cases. |

Implementation approach:

1. Pin a specific rotki commit in oracle metadata.
2. Add a test-time rotki oracle adapter under repository tooling, isolated from production application code.
3. Record rotki AGPL-3.0 license text, source provenance, commit identity, and adapter constraints.
4. Generate new golden fixtures from rotki for `fifo`, `lifo`, `hifo`, and `average_cost` after zero-priced cases are removed from empirical oracle scope.
5. Remove hledger generated journals, hledger golden fixture provenance, and hledger vendoring checks once rotki replacement fixtures are available.
6. Update empirical fixture metadata from hledger-specific fields to generic external-oracle fields that can record rotki command or adapter provenance.
7. Keep runtime application code independent from rotki. The adapter must remain test-time or fixture-generation-only.

### Fix Average Cost With Rotki ACB Aggregate Comparisons

Decision: use rotki ACB as the external oracle for `average_cost` aggregate values only.

Reason: rotki's ACB implementation tracks current amount and current total cost basis, then calculates disposal cost from `current_total_acb / current_amount`. This matches the project moving weighted-average pool model more closely than hledger's lot-shaped `AVERAGE` output.

Average-cost comparison policy after applying this fix:

| Comparison area | Planned behavior |
| --- | --- |
| `values.realized_gain_or_loss` | Compare against rotki ACB. |
| `values.allocated_basis` | Compare against rotki ACB. |
| `values.closing_quantity` | Compare exactly against rotki ACB. |
| `values.closing_basis` | Compare against rotki ACB. |
| `matches` | Do not compare for average cost unless a later adapter exposes project-compatible pool provenance. |

Expected result: remove the broad `average_cost` runtime skip after fixtures are regenerated from rotki and zero-priced average-cost cases are removed. Average-cost empirical validation then covers aggregate calculation correctness and precision, while pool-provenance evidence remains out of scope for the external oracle.

### Fix HIFO Precision With Rotki HIFO Fixtures

Decision: replace hledger HIFO fixtures with rotki HIFO fixtures.

Reason: the current HIFO skip is caused by persisted hledger oracle precision diverging from calculation-layer financial normalization. The disposable rotki prototype matched project HIFO aggregate values within the required tolerance for the standard HIFO case.

HIFO comparison policy after applying this fix:

| Comparison area | Planned behavior |
| --- | --- |
| Aggregate financial values | Compare against rotki HIFO fixtures. |
| Closing quantity | Compare exactly against rotki HIFO fixtures. |
| Match evidence | Compare only after adding a focused deterministic tie-break fixture proving rotki and project acquisition ordering agree for equal-cost HIFO candidates. |

Required HIFO follow-up: add or preserve a targeted HIFO deterministic tie-break empirical case that does not include zero-priced reductions. This is needed because rotki HIFO prioritizes highest acquisition rate, while the project has explicit deterministic tie-breaking requirements.

Expected result: remove the `hifo` runtime skip after rotki HIFO fixtures replace hledger fixtures and HIFO tie-break behavior is explicitly verified or match evidence is limited to aggregate comparisons.

## Current Blocking Themes

1. hledger must be replaced by a rotki-based external oracle adapter before Average Cost and HIFO skips can be removed safely.
2. Zero-priced holding reductions must be removed from empirical oracle scope and retained in traditional project tests.
3. Average Cost should compare aggregate values only until project-compatible pool provenance exists.
4. HIFO match evidence should remain aggregate-only or be gated by a deterministic tie-break fixture.
5. Scope-Local Hybrid still lacks a credible external oracle because its composition rules are project-specific.
