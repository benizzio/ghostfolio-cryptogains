# Empirical Artifacts

This directory stores the synthetic empirical dataset and derived oracle artifacts
used by the repository's supplemental empirical financial tests.

## Dataset Layout

Canonical dataset path:

```text
testdata/empirical/financial-dataset.yaml
```

Top-level fields:

- `dataset_version`: schema and dataset intent version.
- `description`: human-readable synthetic dataset purpose.
- `currency`: the one allowed priced-row currency for this dataset.
- `supported_years`: source-calendar years present in `activities`.
- `supported_methods`: `fifo`, `lifo`, `hifo`, `average_cost`, and `scope_local_hybrid`.
- `coverage_tags`: stable coverage-tag index for required methods and edge-case categories.
- `activities`: synthetic normalized activity rows.
- `cases`: named validation slices used by oracle generation and comparison tests.

Activity fields:

- `source_id`
- `occurred_at`
- `deterministic_order`
- `activity_type`
- `asset_identity_key`
- `asset_symbol`
- `quantity`
- `gross_value`
- `unit_price`
- `fee_amount`
- `currency`
- `source_scope.scope_id`
- `source_scope.scope_kind`
- `source_scope.reliability`
- `source_scope.display_name`
- `zero_priced_reduction_explanation`

Case fields:

- `case_id`
- `description`
- `methods`
- `year`
- `asset_identity_keys`
- `activity_source_ids`
- `coverage_tags`
- `oracle_support`
- `unsupported_reason`

Decimal-policy rules:

- Financial decimal fields are stored as canonical quoted strings.
- Numeric YAML scalars are invalid for decimal fields.
- Zero-priced holding reductions omit priced monetary fields and currency.

## Stable Coverage Tag Index

Method coverage tags:

- `fifo`
- `lifo`
- `hifo`
- `average_cost`
- `scope_local_hybrid`

Edge-case coverage tags:

- `acquisitions`
- `partial_liquidations`
- `full_liquidations`
- `gain_cases`
- `loss_cases`
- `zero_result_liquidations`
- `fees_on_priced_activity`
- `same_source_calendar_date_ordering`
- `pre_year_opening_positions`
- `multi_year_opening_history`
- `selected_year_in_year_activity`
- `post_selected_year_ignored_activity`
- `full_liquidation_followed_by_reacquisition`
- `excluded_assets_from_selected_year_main_results`
- `selected_year_single_lot_liquidation`
- `selected_year_multi_lot_liquidation`
- `hifo_deterministic_tie_breaking`
- `average_cost_multiple_acquisitions`
- `average_cost_partial_disposal`
- `average_cost_full_disposal`
- `average_cost_pool_reset_after_zero`
- `average_cost_reacquisition_after_zero`
- `scope_local_reliable_activity`
- `scope_local_narrowing`
- `scope_local_unreliable_or_unavailable_activity`
- `scope_local_broadening`
- `scope_local_fallback_activation`
- `scope_local_fallback_carry_forward_until_zero`
- `scope_local_same_scope_reset_after_zero`
- `scope_local_independent_other_scope_state`
- `zero_priced_holding_reduction_explicit_zero_fields`
- `zero_priced_holding_reduction_missing_optional_fields`
- `rounded_internal_division_or_allocation`
- `negative_yearly_totals`

## Operating Notes

- Keep all content synthetic. Do not add real user activity, account names, wallet names, tokens, JWTs, proprietary financial records, snapshot payloads, Markdown reports, TUI text, or copied upstream fixture rows.
- `financial-dataset.yaml` is the canonical empirical source dataset.
- `golden/` stores normalized oracle JSON fixtures generated from the dataset.
- Generated rotki adapter inputs are written only under `.cache/empiricaloracle/oracle-inputs/` during explicit regeneration and must not be committed.
- Cases marked `oracle_support: unsupported` stay in the dataset for structural and non-oracle coverage, but do not require committed `golden/` fixtures.
- Generate or refresh derived artifacts only through `tools/empiricaloracle`. Do not hand-edit generated fixtures.
- Regeneration obtains rotki data only by verifying and executing pinned rotki source from the untracked project-local cache path `.cache/empiricaloracle/rotki-source/`.
- Normal fixture-backed empirical test runs must not download rotki, require the untracked source cache, or invoke oracle generation while committed golden fixtures are present.
- Explicit regeneration may download or reuse the pinned source archive only under `.cache/empiricaloracle/rotki-source/`. Clean up that cache by removing the directory when you need to force a fresh verification pass.
- Runtime application code must not read or write empirical fixture artifacts.

## Synthetic-Only Policy

- Use synthetic asset identifiers, synthetic source IDs, synthetic timestamps, and synthetic scope display names only.
- Do not add bearer tokens, JWTs, real account or wallet names, personally identifying names, proprietary financial records, or copied upstream ledger fixture text.
- Validation failures must stay non-secret and actionable.

## Current State

- `financial-dataset.yaml` is the repository-backed synthetic empirical dataset baseline used by empirical validation.
- `golden/` contains the current normalized oracle fixture baseline for the supported empirical fixture groups.
- `.cache/empiricaloracle/oracle-inputs/` is the untracked regeneration-only location for generated rotki adapter inputs.
