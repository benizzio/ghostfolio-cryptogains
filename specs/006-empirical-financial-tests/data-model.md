# Data Model: Empirical Solidified Financial Tests

## Modeling Notes

This feature creates test infrastructure, not product runtime state. Entities below describe repository fixtures, oracle generation metadata, and normalized comparison models. The models must stay isolated from TUI, Ghostfolio transport, protected snapshot encryption, Markdown rendering, and report output.

All decimal fields are stored as canonical decimal strings and parsed into `apd.Decimal` in Go code. No floating-point representation is valid for dataset parsing, oracle normalization, or comparison.

## EmpiricalDataset

Purpose: Complete synthetic ledger and metadata used for empirical financial validation.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `dataset_version` | string | Incremented when schema or intended records change |
| `description` | string | Human-readable dataset purpose |
| `currency` | string | Single currency used by all priced cases |
| `supported_years` | integer array | Source-calendar years present in dataset |
| `supported_methods` | string array | Must include all project cost-basis methods |
| `coverage_tags` | string array | Dataset-level coverage categories |
| `activities` | `EmpiricalActivity[]` | At least 150 synthetic activity rows |
| `cases` | `EmpiricalCase[]` | Method/year/asset groupings used by oracle and comparisons |

Relationships:

- Owns many `EmpiricalActivity` rows.
- Owns many `EmpiricalCase` definitions.
- Produces one or more `OracleInputLedger` files.
- Produces one or more `OracleOutput` golden fixtures.

Validation rules:

- Contains at least 150 activities.
- Spans at least 3 source-calendar years.
- Uses exactly one currency for priced empirical activity.
- Contains no real tokens, JWTs, bearer strings, real account names, wallet names, user activity, or proprietary financial records.
- Includes coverage tags for every required dataset category in `spec.md`.
- Has deterministic source IDs and ordering metadata.
- Becomes read-only after this dataset-maintenance spec is complete.

State transitions:

- `draft -> validated` after schema, synthetic-content, coverage, and deterministic-order checks pass.
- `validated -> oracle_generated` after hledger inputs and normalized golden fixtures are generated.
- `oracle_generated -> read_only_baseline` after implementation completion.

## EmpiricalActivity

Purpose: One synthetic input activity equivalent to a normalized synced activity record.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Stable unique source identifier |
| `occurred_at` | RFC3339 timestamp string | Source timestamp used for year derivation and ordering |
| `deterministic_order` | integer | Stable tie-break within same source date |
| `activity_type` | enum | `BUY` or `SELL` |
| `asset_identity_key` | string | Stable synthetic asset identity |
| `asset_symbol` | string | Synthetic display label |
| `quantity` | decimal string | Positive activity quantity |
| `gross_value` | decimal string nullable | Required for priced activity unless derivable by fixture rule |
| `unit_price` | decimal string nullable | Optional same-tier value used for derivation evidence |
| `fee_amount` | decimal string nullable | Explicit fee. `0` is distinct from missing |
| `currency` | string nullable | Required for priced rows, absent for zero-priced reductions |
| `source_scope` | `EmpiricalScope` nullable | Scope used by Scope-Local Hybrid (`scope_local_hybrid`) cases |
| `zero_priced_reduction_explanation` | string nullable | Required for zero-priced holding reductions |
| `coverage_tags` | string array | Row-level coverage categories |

Relationships:

- Belongs to one `EmpiricalDataset`.
- May reference one `EmpiricalScope`.
- Is translated into one project calculation input.
- May generate one or more hledger postings in `OracleInputLedger`.

Validation rules:

- `source_id` is unique and deterministic.
- `occurred_at` has a valid source-calendar year.
- `activity_type` is supported by the calculation layer.
- `quantity` is a positive decimal string.
- Priced `BUY` and priced `SELL` rows include one currency and enough monetary fields to derive gross value, fee, and unit price according to the dataset contract.
- Zero-priced holding reductions are `SELL` rows with explicit explanation, quantity, no proceeds, and no priced-liquidation treatment.
- Same-source-calendar-date rows use deterministic ordering metadata.
- Activity fields remain synthetic and reviewable.

## EmpiricalScope

Purpose: Synthetic source grouping used to validate scope-local method behavior.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `scope_id` | string | Stable synthetic scope identifier |
| `scope_kind` | enum | `account` or `wallet` |
| `reliability` | enum | `reliable`, `partial`, or `unavailable` |
| `display_name` | string nullable | Synthetic display label |

Relationships:

- May be referenced by many `EmpiricalActivity` rows.
- Determines scope-local narrowing, broadening, and fallback cases.

Validation rules:

- Reliable scopes require stable non-empty `scope_id` and `scope_kind`.
- Partial or unavailable scope data must trigger dataset cases for broadening or fallback.
- Other-scope activity must remain independent in scope-local tests.

## EmpiricalCase

Purpose: Named validation slice for one or more activities, methods, years, and comparable outputs.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `case_id` | string | Stable case identifier used in failures |
| `description` | string | Human-readable reason for the case |
| `methods` | string array | Cost-basis methods covered |
| `year` | integer | Report year under comparison |
| `asset_identity_keys` | string array | Assets included in the case |
| `activity_source_ids` | string array | Dataset rows participating in the case |
| `coverage_tags` | string array | Edge cases proven by this case |
| `oracle_support` | enum | `supported`, `partially_supported`, or `unsupported` |
| `unsupported_reason` | string nullable | Required when not fully supported by hledger |

Relationships:

- References many `EmpiricalActivity` rows.
- Produces one `OracleOutput` segment per method when supported.
- Produces one `EmpiricalComparisonResult` per method when comparable.

Validation rules:

- Every required method and coverage category appears in at least one case.
- Unsupported cases include a reason and are not silently compared as if hledger modeled them.
- Cases do not mutate the dataset.

## OracleInputLedger

Purpose: hledger-compatible representation derived from the empirical dataset for one method or method family.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `ledger_id` | string | Stable identifier for generated hledger input |
| `method` | string | Cost-basis method or method family |
| `case_ids` | string array | Dataset cases represented |
| `hledger_journal_path` | path | Repository path under `testdata/empirical/hledger/` |
| `dataset_input_hash` | string | Hash of the source dataset used to generate the ledger |
| `hledger_input_hash` | string | Hash of the generated hledger journal |
| `generation_notes` | string array | Notes about representation or unsupported fragments |

Relationships:

- Derived from one `EmpiricalDataset`.
- Is consumed by one hledger command invocation.
- Produces one or more `OracleOutput` fixtures.

Validation rules:

- Must not be hand-edited when generated from dataset.
- Must preserve financial meaning of represented dataset cases.
- Must not contain copied upstream hledger examples or real user data.

## HledgerVendoredTool

Purpose: Repository-vendored hledger materials and executable discovery metadata.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `version` | string | Supported hledger version |
| `source_url` | URL | Upstream source URL |
| `license` | string | GPL-3.0-or-later |
| `license_path` | path | Vendored license text path |
| `source_path` | path | Complete corresponding source path under `third_party/hledger/source/` |
| `executable_paths` | path array | Supported executable artifact paths under `third_party/hledger/bin/<goos>-<goarch>/hledger` |
| `executable_checksums` | string map | Checksum for each supported executable artifact |
| `checksum` | string | Checksum for vendored source or artifact |
| `executable_path` | path nullable | Test-time command path when an executable is available |
| `platform_support` | string array | Supported local platforms for generation |

Relationships:

- Used by `OracleGenerationRun` only when hledger generation is required.

Validation rules:

- Binary-only vendoring is invalid.
- License text and complete corresponding source must be present.
- Each supported executable artifact has a matching checksum.
- Runtime application code must not import or execute hledger.
- Missing or unsupported hledger fails fixture generation with an actionable setup error, not normal fixture-backed comparisons.

## OracleGenerationRun

Purpose: One deterministic hledger-backed generation attempt.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `run_id` | string | Deterministic or timestamped generation identifier |
| `hledger_version` | string | Captured from the vendored command |
| `command_arguments` | string array | Exact arguments used |
| `dataset_input_hash` | string | Source dataset hash |
| `hledger_input_hash` | string | Generated ledger hash |
| `decimal_policy` | string | Selected decimal policy used by hledger normalization and project comparison |
| `normalization_version` | string | Project-owned normalizer version |
| `oracle_output_hash` | string | Hash of normalized output |
| `generated_at` | timestamp | Fixture generation time when persisted |

Relationships:

- Uses one `HledgerVendoredTool`.
- Reads one or more `OracleInputLedger` values.
- Writes one or more `OracleOutput` fixtures.

Validation rules:

- Command arguments are recorded exactly.
- Decimal policy is recorded exactly and matches the policy used by empirical project calculation.
- Hashes are present and stable for unchanged inputs.
- Generation runs only when golden fixture is absent or explicit regeneration is requested.

## OracleOutput

Purpose: Normalized assertable expected result derived from hledger output and project-owned normalization.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `fixture_version` | string | Golden fixture schema version |
| `case_id` | string | Empirical case identifier |
| `method` | string | Cost-basis method |
| `year` | integer | Report year |
| `asset_identity_key` | string | Synthetic asset identity |
| `realized_gain_or_loss` | decimal string | Expected yearly realized result |
| `allocated_basis` | decimal string | Expected basis allocated to disposals |
| `closing_quantity` | decimal string | Expected remaining quantity |
| `closing_basis` | decimal string | Expected remaining basis |
| `matches` | `OracleMatchEvidence[]` | Method-specific lot or pool evidence when the fixture records source IDs, evidence type, and expected values |
| `unsupported_segments` | `UnsupportedOracleSegment[]` | Explicit unsupported evidence when needed |
| `metadata` | `OracleGenerationRun` | Generation metadata and hashes |

Relationships:

- Is compared with one `ProjectCalculationOutput` segment.
- Is generated from one `OracleInputLedger` and one `EmpiricalCase`.

Validation rules:

- Decimal fields are canonical decimal strings.
- Quantity fields must be exact.
- Unsupported segments are not included in external-oracle assertions unless a project-owned composition rule explicitly covers them.
- Metadata hashes and hledger version are mandatory.

## OracleMatchEvidence

Purpose: Lot, pool, or scope evidence used to compare method-specific behavior.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `disposed_source_id` | string | Dataset disposal row |
| `acquisition_source_id` | string nullable | Matched lot when hledger provides it |
| `scope_id` | string nullable | Scope for scope-local evidence |
| `matched_quantity` | decimal string | Quantity matched |
| `matched_basis` | decimal string | Basis matched |
| `matched_proceeds` | decimal string nullable | Proceeds evidence when comparable |
| `matched_gain_or_loss` | decimal string nullable | Fragment-level result when comparable |
| `support_label` | enum | `hledger_backed` or `project_composition_rule` |
| `composition_rule_id` | string nullable | Required for `project_composition_rule` evidence |

Relationships:

- Belongs to one `OracleOutput`.
- Maps to project `BasisMatch` evidence where available.

Validation rules:

- Quantities sum to represented disposal quantities.
- Missing hledger evidence is explicit, not inferred silently.
- Scope-Local Hybrid evidence is labeled as hledger-backed evidence or project-owned composition-rule evidence.

## UnsupportedOracleSegment

Purpose: Explicit record that a dataset segment cannot be represented faithfully in hledger.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `case_id` | string | Empirical case identifier |
| `method` | string | Affected method |
| `activity_source_ids` | string array | Rows affected |
| `reason` | string | Why hledger cannot represent the case faithfully |
| `comparison_policy` | enum | `skip_external_oracle`, `project_composition_only`, or `fail_if_selected` |

Relationships:

- Belongs to one `OracleOutput`.
- May be referenced by `EmpiricalComparisonResult` as skipped external evidence.

Validation rules:

- Reason is required.
- Unsupported segments must not fabricate hledger-derived expected values.

## ProjectCalculationOutput

Purpose: Normalized result produced by this project's calculation layer from the empirical dataset.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `case_id` | string | Empirical case identifier |
| `method` | string | Cost-basis method |
| `year` | integer | Report year |
| `asset_identity_key` | string | Synthetic asset identity |
| `realized_gain_or_loss` | decimal string | Normalized from `CapitalGainsReport` |
| `allocated_basis` | decimal string | Normalized from liquidation summaries |
| `closing_quantity` | decimal string | Normalized from detail sections |
| `closing_basis` | decimal string | Normalized from detail sections |
| `matches` | `ProjectMatchEvidence[]` | Project basis-match evidence |

Relationships:

- Derived from one empirical dataset translation and one `calculate.Calculate` call.
- Compared with `OracleOutput`.

Validation rules:

- Uses calculation output only, not Markdown or TUI text.
- Contains no report document structure, saved path, token, JWT, or protected snapshot payload.
- Uses the same method and year as the oracle segment.

## EmpiricalComparisonResult

Purpose: Per-field comparison between oracle output and project calculation output.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `case_id` | string | Empirical case identifier |
| `method` | string | Cost-basis method |
| `year` | integer | Report year |
| `asset_identity_key` | string | Synthetic asset identity |
| `field` | string | Compared field path |
| `expected_value` | decimal string | Oracle value |
| `actual_value` | decimal string | Project value |
| `difference` | decimal string | Absolute or signed difference according to field contract |
| `decimal_policy` | string | Selected comparison policy, such as production 16-decimal round-half-up or hledger-aligned empirical policy |
| `tolerance` | decimal string | Allowed residual difference, zero for quantity fields and at most one unit of selected decimal-policy scale for financial fields |
| `passed` | boolean | Whether comparison passed |
| `diagnostic_context` | string | Non-secret failure context |

Relationships:

- References one `OracleOutput` field and one `ProjectCalculationOutput` field.

Validation rules:

- Quantities compare by exact decimal equality after normalization under `decimal_policy`.
- Financial values compare after normalization under `decimal_policy` with the documented `tolerance` for residual hledger/project deviations.
- If hledger cannot align with the production 16-decimal policy, empirical tests select the hledger-aligned policy through the test-scoped environment variable before project calculation runs.
- Quantity `tolerance` is zero.
- Financial `tolerance` is documented per field and must not exceed one unit at the selected decimal-policy scale.
- A non-zero financial tolerance requires a note explaining why exact equality is not achievable for that hledger-derived value.
- Failure output identifies dataset case, method, asset, year, field, decimal policy, expected, actual, difference, and tolerance.
- Failure output excludes secrets, raw protected payloads, Markdown, report files, and UI text.
