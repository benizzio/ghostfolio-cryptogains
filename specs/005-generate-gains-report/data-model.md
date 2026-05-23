# Data Model: Generate Yearly Gains And Losses Report

## Modeling Notes

This slice reuses the protected snapshot model from `specs/003-store-activity-data/` and adds report-generation runtime models. The final Markdown file is the only cleartext persisted output created by this feature. Report requests, calculations, rendered content, output paths, and open results are not persisted back into setup or protected snapshots.

The existing stored activity model must expose a stable Ghostfolio asset identity key. If current persisted records only expose display symbol or name, implementation must update the stored activity model and compatibility version before report generation can safely group assets.

All financial quantities and values are represented with exact decimals in runtime code and canonical exact-decimal strings at report boundaries. No floating-point representation is valid for these entities.

## Existing ProtectedActivityCache

Purpose: Existing token-derived encrypted synced activity dataset that gates and drives report generation.

Fields used by this slice:

| Field | Type | Notes |
|-------|------|-------|
| `synced_at` | timestamp | Last successful sync time displayed beside `Sync Data` after unlock |
| `activity_count` | integer | Determines whether synced activity data exists |
| `available_report_years` | integer array | Candidate years shown in report-generation workflow |
| `scope_reliability` | enum | Determines whether scope-local method can narrow by source scope or must broaden to asset |
| `activities` | `ActivityRecord[]` | Chronological normalized activity history |

Relationships:

- Belongs to one protected snapshot unlocked by the active Ghostfolio token.
- Supplies zero or more `ActivityRecord` values to one `ReportRequest`.

Validation rules:

- Report generation is unavailable when the cache is absent or `available_report_years` is empty.
- `available_report_years` contains source calendar years derived from each stored `occurred_at` timestamp using that timestamp's own offset.
- A source calendar year remains reportable even when that year's in-scope activity contains only acquisitions or only explained zero-priced holding reductions.
- Cache contents remain protected until `Sync and Reports` is unlocked by token.
- Cache data is read for reports but not modified by report generation.

State transitions:

- `absent -> present` after a successful sync.
- `present -> refreshed` after `Sync Data` succeeds inside the unlocked context.
- `present -> unchanged` after report generation.

## ActivityRecord

Purpose: Existing normalized stored Ghostfolio activity used as the source event for holdings and basis replay.

Fields used or required by this slice:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Deterministic ordering tie-break and detail-row source reference |
| `occurred_at` | RFC3339 timestamp string | Source timestamp, preserving offset for year derivation |
| `activity_type` | enum | `BUY` or `SELL` |
| `asset_identity_key` | string | Stable stored Ghostfolio asset identity used for all grouping and calculations |
| `asset_symbol` | string | Rendering label only |
| `asset_name` | string nullable | Rendering label only |
| `quantity` | decimal | Acquired, liquidated, or reduced quantity |
| `order_currency` | string nullable | Currency context for order-tier amounts |
| `order_unit_price` | decimal nullable | Order-tier unit price |
| `order_gross_value` | decimal nullable | Order-tier gross value |
| `order_fee_amount` | decimal nullable | Order-tier fee |
| `asset_profile_currency` | string nullable | Currency context for asset-profile-tier amounts |
| `asset_profile_unit_price` | decimal nullable | Asset-profile-tier unit price |
| `asset_profile_fee_amount` | decimal nullable | Asset-profile-tier fee |
| `base_currency` | string nullable | Currency context for base-tier amounts |
| `base_gross_value` | decimal nullable | Base-tier gross value |
| `base_fee_amount` | decimal nullable | Base-tier fee |
| `comment` | string nullable | Required explanation for zero-priced `SELL` records admitted by sync |
| `source_scope` | `SourceScope` nullable | Optional account, wallet, or equivalent scope information |
| `raw_hash` | string | Deterministic duplicate-removal hash already created by sync |

Relationships:

- Belongs to one `ProtectedActivityCache`.
- May reference one `SourceScope`.
- Is transformed into one `ActivityCalculationInput` during report generation.

Validation rules:

- `asset_identity_key` is mandatory for report generation. Symbol or name must not replace it as a grouping key.
- Only `BUY` and `SELL` are supported.
- `BUY` records create acquisitions and require a positive selected unit price.
- Priced `SELL` records create liquidations and require positive quantity.
- Explained zero-priced `SELL` records create holding reductions with zero gain and zero loss.
- Activity after the selected year is ignored for the report run.

## SourceScope

Purpose: Existing optional source grouping used by the scope-local hybrid cost-basis method.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `scope_id` | string | Source account or wallet identity |
| `scope_name` | string nullable | Rendering or diagnostic label |
| `scope_kind` | enum | `account`, `wallet`, or `unknown` |
| `reliability` | enum | `reliable`, `partial`, or `unavailable` |

Relationships:

- May be referenced by many `ActivityRecord` values.
- Contributes to `ApplicableScope` resolution.

Validation rules:

- Reliable scope data can narrow scope-local reporting.
- Missing, partial, or contradictory scope data broadens the scope-local method to the whole asset instead of failing solely for missing scope detail.
- Report generation reuses this reliability classification and does not try to infer a narrower scope from contradictory or partial rows.

## SyncAndReportsContext

Purpose: Active token-unlocked workflow state for sync and report actions.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `server_origin` | string | runtime only | Canonical selected Ghostfolio server |
| `security_token` | secret string | runtime only | User-entered token reused only while this context is active |
| `active_readable_snapshot` | `ActiveReadableSnapshot` nullable | runtime only | Current run's decrypted protected snapshot, as defined by the existing `003` derived runtime concept |
| `protected_activity_cache` | `ProtectedActivityCache` nullable | runtime only | Current readable cache after unlock or sync |
| `entered_at` | timestamp | runtime only | Context creation time |

Relationships:

- Uses one bootstrap `AppSetupConfig`.
- May hold one readable protected snapshot.
- Starts zero or more `Sync Data` attempts.
- Starts zero or more report-generation attempts.

Validation rules:

- Token must be explicitly entered before actions are exposed.
- Token must be cleared when leaving the context or when the app exits.
- Protected cache metadata, report years, and last-sync timestamp must not be displayed before unlock.
- Snapshot miss alone must not activate the context.
- If no selected-server snapshot unlocks and Ghostfolio rejects the informed token, the workflow must remain on the unlock screen with `access denied`, `Unlock` disabled, `Back` as the only available action, the rejected token value preserved only for that failed screen instance, and that field cleared only after leaving and re-entering the unlock screen.

State transitions:

- `locked -> unlocked` after selected-server snapshot unlock.
- `locked -> authenticating_new_context -> unlocked` after selected-server snapshot miss and Ghostfolio authentication success for a new isolated local-user context.
- `locked -> rejected_token` after selected-server snapshot miss and Ghostfolio authentication rejection.
- `rejected_token -> cleared` when the user leaves the unlock screen with `Back`.
- `cleared -> locked` when the user later re-enters `Sync and Reports`.
- `unlocked -> syncing -> unlocked` after sync success or failure.
- `unlocked -> selecting_report -> generating_report -> unlocked` after report success or failure.
- `unlocked -> cleared` when the user leaves the context.

## ReportRequest

Purpose: User-selected inputs for one report-generation run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `year` | integer | Must be present in `ProtectedActivityCache.available_report_years` |
| `cost_basis_method` | enum | `fifo`, `lifo`, `hifo`, `average_cost`, or `scope_local_hybrid` |
| `requested_at` | timestamp | Local generation request time used for output naming |

Relationships:

- Belongs to one active `SyncAndReportsContext`.
- Consumes one `ProtectedActivityCache`.
- Produces one `CapitalGainsReport` or one non-secret error.

Validation rules:

- The year must be selected from synced data, not free text.
- The method must be one of the supported methods.
- One method applies consistently to all included liquidations in the run.
- Generation may still succeed when the chosen year later yields no main-section assets, in which case the report uses the documented empty states.

State transitions:

- `draft -> validated -> calculated -> rendered -> saved` on success.
- `draft -> failed` when inputs or data cannot support calculation.

## CostBasisMethod

Purpose: Enumeration of supported basis allocation methods and user-visible explanations.

Values:

| Value | User label | Summary |
|-------|------------|---------|
| `fifo` | FIFO | Consume oldest open acquisitions first |
| `lifo` | LIFO | Consume newest open acquisitions first |
| `hifo` | HIFO | Consume highest unit-cost open acquisitions first, with deterministic tie-breaks |
| `average_cost` | Average Cost Basis | Maintain one moving weighted-average pool per asset |
| `scope_local_hybrid` | Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order | Use reliable scope and exact matching when defensible, otherwise scope-local average cost until full liquidation |

Relationships:

- Selected by one `ReportRequest`.
- Determines the `BasisState` implementation used for every asset or scope partition.

Validation rules:

- Explanation text must be visible before generation.
- Scope-local fallback consistency is tracked per `(asset_identity_key, applicable_scope)` until quantity reaches zero.
- Once one applicable scope reaches zero, a later reacquisition in that same scope starts a new scope-local state whose exact-identification eligibility is evaluated again.

## ActivityCalculationInput

Purpose: One normalized activity after selecting the single-activity currency context for calculation.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Copied from `ActivityRecord` |
| `occurred_at` | timestamp | Parsed from `ActivityRecord.occurred_at` |
| `source_year` | integer | Derived from source timestamp offset |
| `activity_type` | enum | `BUY` or `SELL` |
| `asset_identity_key` | string | Calculation grouping key |
| `display_label` | string | Symbol or name used for rendering only |
| `quantity` | decimal | Exact activity quantity |
| `gross_value` | decimal nullable | Selected from one complete activity currency context for priced activities; may remain as preserved explicit `0` for explained zero-priced holding reductions |
| `fee_amount` | decimal nullable | Selected from the same activity currency context for priced activities; may remain as preserved explicit `0` for explained zero-priced holding reductions |
| `unit_price` | decimal nullable | Selected or derived within the same activity context when exact for priced activities; may remain as preserved explicit `0` for explained zero-priced holding reductions |
| `selected_currency_context` | enum nullable | `order`, `asset_profile`, or `base` when a priced activity requires one |
| `selected_currency_code` | string nullable | Explicit currency code carried from the selected activity context when one is required |
| `source_scope` | `SourceScope` nullable | Preserved source scope |
| `is_zero_priced_holding_reduction` | boolean | True only for explained zero-priced `SELL` rows |
| `comment` | string nullable | Explanation shown for holding reductions when present |

Relationships:

- Derived from one `ActivityRecord`.
- Applied to one `AssetPositionTimeline`.

Validation rules:

- A `BUY` requires gross acquisition value and fee from one chosen context.
- A priced `SELL` requires gross liquidation value and fee from one chosen context.
- An explained zero-priced holding reduction requires no activity monetary inputs and therefore no selected currency context, even when preserved explicit zero-valued `unit_price`, `gross_value`, or `fee_amount` remain present.
- An explicit fee value of `0` is valid. A missing fee is not equivalent to zero.
- Priced activity quantity must be greater than zero.
- If unit price is not stored explicitly for a priced activity, it may be derived only when the needed division terminates exactly.
- Values from different tiers must not be mixed inside one input.
- `selected_currency_code` must match the chosen tier and remain explicit when a chosen tier exists.
- After input creation, currency identity remains explicit for priced rows. No conversion occurs, and successful rendered cross-activity monetary outputs require one shared report calculation currency.

## ApplicableScope

Purpose: Runtime grouping used only by the scope-local hybrid method.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_identity_key` | string | Parent asset identity |
| `scope_key` | string | Reliable source scope key or asset identity when broadened |
| `scope_kind` | enum | `account`, `wallet`, or `asset` |
| `broadened_to_asset` | boolean | True when scope data is unreliable or unavailable |

Relationships:

- Belongs to one `AssetPositionTimeline`.
- Owns one method-specific scope-local basis state.

Validation rules:

- A missing or unreliable source scope broadens to asset-level scope.
- Once scope-local average fallback occurs in an open scope, later disposals in that scope use fallback until quantity reaches zero.
- Reacquisition in a different applicable scope does not alter another open scope's state.

## BasisLot

Purpose: Runtime lot fragment used by FIFO, LIFO, HIFO, exact matching, and provenance queues.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Acquisition source identifier |
| `asset_identity_key` | string | Owning asset |
| `applicable_scope` | `ApplicableScope` nullable | Present for scope-local state |
| `acquired_at` | timestamp | Acquisition ordering timestamp |
| `deterministic_order` | integer | Stable order from normalized history |
| `remaining_quantity` | decimal | Open lot quantity |
| `remaining_basis` | decimal | Open lot basis |

Relationships:

- Created from an acquisition input.
- Consumed by one or more `BasisMatch` values.

Validation rules:

- Remaining quantity and basis must never become negative.
- When remaining quantity reaches zero, the lot is closed.

## BasisMatch

Purpose: One matched acquisition fragment consumed by a liquidation or holding reduction.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `acquisition_source_id` | string | Source lot consumed |
| `matched_quantity` | decimal | Quantity consumed from the lot or pool |
| `matched_basis` | decimal | Basis allocated to this matched quantity |
| `matched_proceeds` | decimal nullable | Proportional proceeds for priced liquidations |
| `matched_gain_or_loss` | decimal nullable | Proportional result for priced liquidations |

Relationships:

- Belongs to one `LiquidationCalculation` or holding-reduction detail row.

Validation rules:

- Match quantities sum to the liquidation or reduction quantity.
- Matched basis is removed from open basis exactly.
- Holding reductions have no matched proceeds and no gain or loss.

## AssetPositionTimeline

Purpose: Derived per-asset timeline through the selected year end.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_identity_key` | string | Grouping key |
| `display_label` | string | Rendering label |
| `opening_quantity` | decimal | Position at start of selected year |
| `opening_basis` | decimal | Basis carried into selected year |
| `in_year_rows` | `AssetActivityDetailRow[]` | Every in-year activity row for included assets |
| `closing_quantity` | decimal | Position at selected year end |
| `closing_basis` | decimal | Basis at selected year end |
| `yearly_gain_or_loss` | decimal | Net result from priced in-year liquidations only |
| `full_liquidation_count_through_year_end` | integer | Count used by reference section |
| `had_in_year_full_liquidation` | boolean | Main-section inclusion criterion |
| `reopened_on_or_before_year_end` | boolean | Reference-only exclusion decision |
| `has_in_year_activity` | boolean | Whether any activity for the asset occurs inside the selected year |

Relationships:

- Derived from many `ActivityCalculationInput` values for one asset.
- Contains zero or more `LiquidationCalculation` values.
- Produces one `AssetSummaryEntry` when included in main sections.

Validation rules:

- Activity after the selected year is excluded.
- Activity before the selected year may affect opening quantity and basis.
- Main section inclusion requires an open year-end position or a full liquidation during the selected year.
- The timeline may still be included when `has_in_year_activity` is false, in which case the rendered section uses the documented empty-state instead of in-year tables.

## AssetActivityDetailRow

Purpose: One rendered in-year activity row in an included per-asset section.

Fields:

| Field                           | Type                              | Notes                                                                                                                                                             |
|---------------------------------|-----------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `source_id`                     | string                            | Source activity identifier                                                                                                                                        |
| `occurred_at`                   | timestamp                         | Activity timestamp                                                                                                                                                |
| `activity_type`                 | enum                              | `BUY` or `SELL`                                                                                                                                                   |
| `quantity`                      | decimal                           | Activity quantity                                                                                                                                                 |
| `unit_price`                    | decimal nullable                  | Selected or derived unit price for priced rows; may remain as preserved explicit `0` when a detail layout carries it for explained zero-priced holding reductions |
| `gross_value`                   | decimal nullable                  | Selected gross value for acquisitions and priced liquidations, or preserved explicit `0` for explained zero-priced holding reductions when present                |
| `fee_amount`                    | decimal nullable                  | Selected fee for priced rows, or preserved explicit `0` for explained zero-priced holding reductions when present                                                 |
| `activity_currency`             | string nullable                   | Explicit currency code from the row's single-activity currency context when one exists                                                                            |
| `calculation_currency`          | string                            | Explicit shared report calculation currency for calculated row values, or `NOT APPLICABLE` in this slice                                                          |
| `basis_after_row`               | decimal                           | Remaining basis after applying row                                                                                                                                |
| `quantity_after_row`            | decimal                           | Remaining quantity after applying row                                                                                                                             |
| `holding_reduction_explanation` | string nullable                   | Present for zero-priced holding reductions                                                                                                                        |
| `liquidation_calculation`       | `LiquidationCalculation` nullable | Present for priced in-year liquidations                                                                                                                           |

Relationships:

- Belongs to one `AssetPositionTimeline`.

Validation rules:

- Every in-year activity for an included asset appears as one row.
- Zero-priced holding reductions show basis and quantity effects without realized gain or loss.
- For priced rows, `activity_currency` records the explicit currency code used for `gross_value` and `fee_amount` in that row.
- For explained zero-priced holding reductions, `activity_currency` remains blank because no selected currency context exists. `unit_price`, `gross_value`, and `fee_amount` stay nullable so missing values remain blank while preserved explicit `0` values remain distinguishable.
- `calculation_currency` records the explicit shared report calculation currency used for calculated row values such as `basis_after_row`.

## LiquidationCalculation

Purpose: Per-liquidation calculation details for priced in-year liquidations.

Fields:

| Field                      | Type           | Notes                                                                                                            |
|----------------------------|----------------|------------------------------------------------------------------------------------------------------------------|
| `source_id`                | string         | Liquidation source identifier                                                                                    |
| `disposed_quantity`        | decimal        | Liquidated quantity                                                                                              |
| `allocated_basis`          | decimal        | Basis consumed by the selected method                                                                            |
| `net_liquidation_proceeds` | decimal        | Gross liquidation value minus liquidation fee                                                                    |
| `gain_or_loss`             | decimal        | Net proceeds minus allocated basis                                                                               |
| `activity_currency`        | string         | Explicit currency code selected for that liquidation activity                                                    |
| `calculation_currency`     | string         | Explicit shared report calculation currency for liquidation calculation values or `NOT APPLICABLE` in this slice |
| `matches`                  | `BasisMatch[]` | Lot or pool fragments used by the selected method                                                                |

Relationships:

- Belongs to one `AssetActivityDetailRow`.
- Contributes to one `AssetSummaryEntry` and the report yearly total.

Validation rules:

- Only liquidations inside the selected year contribute to gains and losses.
- `net_liquidation_proceeds` remains rendered in `activity_currency` because it is derived from one activity before cross-activity calculations.
- `allocated_basis` and `gain_or_loss` remain rendered in `calculation_currency` because they are cross-activity calculation outputs.
- Losses are negative values and zero is rendered as `0`.

## AssetSummaryEntry

Purpose: One row in the first report section.

Fields:

| Field                         | Type    | Notes                                                                      |
|-------------------------------|---------|----------------------------------------------------------------------------|
| `asset_identity_key`          | string  | Grouping key, not necessarily rendered                                     |
| `display_label`               | string  | Label shown to the user                                                    |
| `net_gain_or_loss`            | decimal | Selected-year result for the asset                                         |
| `result_kind`                 | enum    | `gain`, `loss`, or `zero`                                                  |
| `report_calculation_currency` | string  | Explicit shared report calculation currency `NOT APPLICABLE` in this slice |

Relationships:

- Belongs to one `CapitalGainsReport`.
- Corresponds to one included `AssetPositionTimeline`.

Validation rules:

- Include zero-result assets when they meet main-section inclusion rules.
- Summary order must be deterministic.
- The report may contain zero `AssetSummaryEntry` rows when the selected year remains reportable but no asset qualifies for main sections.

## ReferenceLiquidationEntry

Purpose: One reference-section entry for an asset that reached zero quantity by the selected year end.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_identity_key` | string | Grouping key |
| `display_label` | string | Label shown to the user |
| `full_liquidation_count_through_year_end` | integer | Number of full liquidations through cutoff |
| `main_section_status` | enum | `included_in_main_sections` or `reference_only` |

Relationships:

- Belongs to one `CapitalGainsReport`.

Validation rules:

- Every asset with at least one full liquidation on or before year end appears here.
- Assets fully liquidated before the selected year and not reopened on or before year end are reference-only.
- For the scope-local hybrid method, one entry's count is the sum of applicable-scope transitions to zero for that asset.

## CapitalGainsReport

Purpose: Final calculated yearly report before Markdown rendering.

Fields:

| Field                         | Type                          | Notes                                                                      |
|-------------------------------|-------------------------------|----------------------------------------------------------------------------|
| `year`                        | integer                       | Selected report year                                                       |
| `cost_basis_method`           | enum                          | Method used for the whole run                                              |
| `generated_at`                | timestamp                     | Local generation time                                                      |
| `report_calculation_currency` | string                        | Explicit shared report calculation currency `NOT APPLICABLE` in this slice |
| `summary_entries`             | `AssetSummaryEntry[]`         | First report section rows                                                  |
| `yearly_net_total`            | decimal                       | Sum of included assets' yearly results                                     |
| `reference_entries`           | `ReferenceLiquidationEntry[]` | Second report section rows                                                 |
| `detail_sections`             | `AssetPositionTimeline[]`     | Per-asset detail sections                                                  |

Relationships:

- Derived from one `ReportRequest` and one `ProtectedActivityCache`.
- Rendered into one `ReportDocument`.

Validation rules:

- Section order is summary, reference, then per-asset details.
- The report calculation currency label appears in required report-wide contexts.
- Report contains no activity after the selected year.
- The report may contain no detail sections when no asset qualifies for the main report sections.

## ReportDocumentType

Purpose: Enumeration that defines the generated report document format.

Values:

| Value | Notes |
|-------|-------|
| `markdown` | The only supported value in this slice |

Validation rules:

- This slice supports only `markdown`.
- Adding future report formats extends this enum without renaming the document entity.

## ReportDocument

Purpose: Rendered report content before final save.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `document_type` | `ReportDocumentType` | runtime only until final save | `markdown` in this slice |
| `content` | string | runtime only until final save | Rendered report body |
| `year` | integer | runtime only | Used for filename descriptor |
| `cost_basis_method` | enum | runtime only | Used for filename descriptor |
| `generated_at` | timestamp | runtime only | Used for filename timestamp |

Relationships:

- Rendered from one `CapitalGainsReport`.
- Saved as one `ReportOutputFile`.

Validation rules:

- Content must not be written to app-managed storage or OS temp locations.
- Content must not include the Ghostfolio token or JWT.
- `document_type` must be `markdown` in this slice.

## ReportOutputFile

Purpose: Final cleartext Markdown file saved in the user's Documents folder.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `documents_directory` | path | filesystem | Current user's Documents folder |
| `filename` | string | filesystem | Timestamped `YYYY-MM-DD_HH-MM-SS` prefix plus descriptor and optional suffix |
| `path` | path | filesystem | Full output path |
| `saved_at` | timestamp | runtime result only | Transient success message data |
| `open_requested` | boolean | runtime result only | Whether OS open was attempted |
| `open_error` | string nullable | runtime result only | Non-secret opener failure text |

Relationships:

- Created from one `ReportDocument`.
- Reported back through one transient `ReportGenerationOutcome`.

Validation rules:

- Existing files must not be overwritten.
- Partial files from failed saves must be removed.
- Successful files remain even if automatic opening fails.

## ReportFailureDiagnostics

Purpose: Runtime diagnostics-generation state and persisted-source context for one eligible report-generation failure.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `eligible` | boolean | runtime only | True only for report-generation failures that occur before the final Markdown file is saved |
| `status` | enum | runtime only | `not_applicable`, `prompt_required`, `declined`, `generating`, `generated`, or `generation_failed` |
| `failure_category` | enum nullable | runtime only | Primary report-generation failure category that drives diagnostics eligibility and user messaging |
| `artifact_path` | path nullable | runtime only | Present only when a diagnostics artifact is written successfully |
| `generation_message` | string nullable | runtime only | Non-secret status text for diagnostics success or failure |
| `offending_activity_record` | `ActivityRecord` nullable | runtime only | Original persisted record included only when the report failure is activity-specific |
| `redaction_mode` | enum nullable | runtime only | `production_redacted` or `explicit_development_detail` when diagnostics are generated |

Relationships:

- Belongs to one `ReportGenerationOutcome`.
- Writes zero or one local `.diagnostic.json` artifact under the application-owned diagnostics directory.

Validation rules:

- Diagnostics are eligible only for calculation, validation, rendering, or output-preparation failures that happen before the final Markdown file is saved.
- A successful save followed by automatic-open failure is not diagnostics-eligible.
- When `offending_activity_record` is present, it must mirror the original persisted `ActivityRecord` rather than selected activity inputs, rendered rows, or other derived report values.
- Nullable source fields in serialized offending-record output must render as explicit `null` values.
- Production-mode diagnostics redact financial-value fields. Explicit-development-mode diagnostics may include the full non-secret offending persisted record context.

## ReportGenerationOutcome

Purpose: Transient user-visible result and optional diagnostics state after generation finishes.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `success` | boolean | runtime only | Whether report saved successfully |
| `saved_path` | path nullable | runtime only | Present after successful save |
| `open_failed` | boolean | runtime only | True when save succeeded but opener failed |
| `message` | string | runtime only | Non-secret result or actionable error |
| `failure_category` | enum nullable | runtime only | Primary actionable report-generation failure category, absent on success and opener-only warnings |
| `diagnostics` | `ReportFailureDiagnostics` nullable | runtime only | Present for eligible report-generation failures or diagnostics-generation attempts |

Relationships:

- Belongs to one active `SyncAndReportsContext` until the result is dismissed.
- May own one `ReportFailureDiagnostics` state object while the result screen remains active.

Validation rules:

- Outcome is not persisted.
- `failure_category` remains absent when the report saves successfully, including opener-only warning cases.
- Diagnostics prompt state, diagnostics path disclosure, and any offending persisted activity context remain transient and are cleared when the result is dismissed or the user leaves the context.
- Error messages must not include token, JWT, raw protected payloads, or unredacted financial details beyond what the final report intentionally contains after save.
