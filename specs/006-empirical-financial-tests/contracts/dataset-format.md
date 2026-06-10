# Contract: Empirical Dataset Format

**Bugfix**: 2026-06-10 — [BUG-001] Updated dataset contract wording for selected external-oracle inputs and zero-priced external-oracle exclusion.

## Scope

This contract defines the synthetic empirical external dataset consumed by selected external-oracle generation and empirical calculation tests.

## Location

Canonical dataset path:

```text
testdata/empirical/financial-dataset.yaml
```

Supporting documentation path:

```text
testdata/empirical/README.md
```

## Top-Level Shape

```yaml
dataset_version: "1"
description: "Synthetic empirical financial validation dataset"
currency: "USD"
supported_years:
  - 2023
  - 2024
  - 2025
supported_methods:
  - fifo
  - lifo
  - hifo
  - average_cost
  - scope_local_hybrid
coverage_tags:
  - multi_year_opening_history
activities:
  - source_id: emp-act-000001
    occurred_at: "2023-01-02T09:00:00Z"
    deterministic_order: 1
    activity_type: BUY
    asset_identity_key: asset-alpha
    asset_symbol: ALPHA
    quantity: "1"
    gross_value: "10"
    unit_price: "10"
    fee_amount: "0"
    currency: USD
    source_scope:
      scope_id: wallet-a
      scope_kind: wallet
      reliability: reliable
      display_name: Synthetic Wallet A
    coverage_tags:
      - fifo
cases:
  - case_id: case-fifo-basic-2024
    description: FIFO disposal consumes oldest lot
    methods:
      - fifo
    year: 2024
    asset_identity_keys:
      - asset-alpha
    activity_source_ids:
      - emp-act-000001
    coverage_tags:
      - fifo
    oracle_support: supported
```

## Required Dataset Rules

- `dataset_version` is required and changes when the dataset schema or intended records change.
- `currency` is required and is the only currency allowed for priced empirical cases.
- `activities` must contain at least 150 rows.
- `supported_years` must include at least 3 source-calendar years derived from `occurred_at` values.
- `supported_methods` must include `fifo`, `lifo`, `hifo`, `average_cost`, and `scope_local_hybrid`.
- `coverage_tags` must include every required category from `spec.md`.
- `cases` must cover every supported method and every required edge-case category.

## Activity Rules

- `source_id` is required, unique, stable, and deterministic.
- `occurred_at` is required and must be RFC3339 with a source offset.
- `deterministic_order` is required and must reproduce same-date project calculation order and selected external-oracle input order.
- `activity_type` is either `BUY` or `SELL`.
- `asset_identity_key` is required and is the calculation grouping key.
- `asset_symbol` is required and is a display label only.
- `quantity` is required and must be a positive decimal string.
- Priced `BUY` and priced `SELL` rows require `currency` and enough monetary values to calculate gross value, fee, basis, and proceeds without cross-currency conversion.
- `fee_amount: "0"` is valid and distinct from a missing fee.
- Zero-priced holding reductions are `SELL` rows with `zero_priced_reduction_explanation`, no proceeds, no realized gain, and no realized loss. After BUG-001, these rows are excluded from empirical external-oracle fixture coverage and remain covered by non-oracle unit, integration, or contract tests.
- Decimal fields are strings. Numeric YAML values are invalid for financial fields.
- Coverage tags on rows and cases must be stable identifiers, not prose.

## Scope Rules

- Reliable scope rows require non-empty `scope_id` and `scope_kind`.
- `scope_kind` is `account` or `wallet`.
- `reliability` is `reliable`, `partial`, or `unavailable`.
- The dataset must include reliable scoped activity and unreliable or unavailable scoped activity.
- Scope-local cases must cover narrowing, broadening, fallback activation, fallback carry-forward until zero, same-scope reset after zero, and independent other-scope state.

## Synthetic-Only Rules

Dataset content must not contain:

- real Ghostfolio security tokens
- bearer tokens
- JWT-like strings
- real user activity
- real account or wallet names
- personally identifying names
- proprietary financial records
- copied upstream hledger, rotki, Ledger, Beancount, or other external-oracle fixture rows

## Read-Only Rule

This feature is a dataset-maintenance spec and may create the dataset. After completion, ordinary feature work must treat `testdata/empirical/financial-dataset.yaml` as read-only and may change it only in a later isolated dataset-maintenance spec.

## Validation Contract

Dataset validation must fail with actionable messages that include:

- dataset path
- row `source_id` or `case_id` when applicable
- field name
- violation kind
- non-secret explanation

Validation messages must not include raw protected payloads, tokens, JWTs, or real user data.
