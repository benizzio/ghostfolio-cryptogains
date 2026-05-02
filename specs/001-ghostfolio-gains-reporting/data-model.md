# Data Model: Ghostfolio Gains Reporting

## Modeling Notes

All persisted user-related data is stored inside a single encrypted snapshot payload per registered local user. Only non-secret envelope metadata required to derive the key and verify the ciphertext remains outside the encrypted payload.

## EncryptedSnapshotEnvelope

Purpose: Cryptographic container stored on disk for one registered local user.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `file_id` | UUID string | cleartext filename | Random opaque identifier, not user-derived |
| `format_version` | integer | cleartext header | Outer envelope version |
| `kdf_name` | enum | cleartext header | Baseline `argon2id` |
| `kdf_memory_kib` | integer | cleartext header | Baseline 19456 KiB |
| `kdf_iterations` | integer | cleartext header | Baseline 2 |
| `kdf_parallelism` | integer | cleartext header | Baseline 1 |
| `salt` | bytes | cleartext header | Fresh random salt each rewrite |
| `nonce` | bytes | cleartext header | Fresh random GCM nonce each rewrite |
| `ciphertext` | bytes | encrypted payload | Includes authentication tag |

Relationships:

- Wraps exactly one `SnapshotPayload`.

Validation rules:

- Header bytes are authenticated as AEAD additional authenticated data.
- `file_id` must never encode user identity or server host data.
- Envelope parse or decrypt failure is surfaced as a generic unlock failure.

## SnapshotPayload

Purpose: Versioned decrypted state for one registered local user.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `schema_version` | integer | Inner payload version for migration |
| `registered_local_user` | `RegisteredLocalUser` | Core local profile metadata |
| `setup_profile` | `SetupProfile` | Selected Ghostfolio server configuration |
| `protected_activity_cache` | `ProtectedActivityCache` nullable | Normalized activity history and sync metadata |
| `available_report_years` | integer array | Derived from cached activity dates |

Relationships:

- Contains one `RegisteredLocalUser`.
- Contains one `SetupProfile`.
- Contains zero or one `ProtectedActivityCache`.

Validation rules:

- Persist only after successful Ghostfolio authentication, full retrieval, normalization, and validation.
- Rewrite atomically as a whole snapshot; no partial-record updates.

## RegisteredLocalUser

Purpose: Local profile metadata for one successfully registered user.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `profile_id` | UUID string | Internal stable identifier inside the payload |
| `created_at` | timestamp | First successful registration time |
| `updated_at` | timestamp | Last successful snapshot rewrite |
| `last_successful_sync_at` | timestamp nullable | Null until the first completed sync |
| `report_base_currency` | string nullable | Base currency used for report output when available from Ghostfolio |

Relationships:

- Owns one `SetupProfile`.
- Owns zero or one `ProtectedActivityCache`.

Validation rules:

- This entity does not persist until the initial auth and sync workflow succeeds.
- Binding to the Ghostfolio token is implicit through successful decrypt of the snapshot; no token, token hash, or verifier is stored.

State transitions:

- `absent -> persisted` after successful first sync.
- `persisted -> replaced` after confirmed server-mismatch sync completes successfully.
- `persisted -> deleted` when the user removes the local profile.

## SetupProfile

Purpose: Protected per-user configuration that identifies the selected Ghostfolio server and local transport policy.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `setup_complete` | boolean | Gates report workflows |
| `server_mode` | enum | `ghostfolio_cloud` or `custom_origin`; first-run default is `ghostfolio_cloud` |
| `server_origin` | string | Canonical scheme, host, and optional port only; defaults to `https://ghostfol.io` |
| `allow_insecure_http` | boolean | Default false; true only for explicitly permitted local-development origins |
| `last_validated_at` | timestamp nullable | Updated after a successful connectivity/auth check |

Relationships:

- Belongs to one `RegisteredLocalUser`.

Validation rules:

- `server_origin` must be an absolute origin without path, query, or fragment.
- HTTPS is required unless the origin is an explicitly permitted local-development address. Production-like HTTP origins are rejected with a blocking error.
- Changing `server_origin` after a cache exists requires a server-mismatch confirmation flow.

State transitions:

- `incomplete -> complete` after the first successful sync.
- `complete -> pending_replacement` when the stored server origin and current configured origin differ.
- `pending_replacement -> complete` after confirmed replacement sync succeeds.

## GhostfolioSession

Purpose: Ephemeral authenticated runtime state for one application run and one registered local user.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `registered_local_user_profile_id` | UUID string | Runtime reference to the selected local user context for this session |
| `server_origin` | string | Active Ghostfolio origin for this run |
| `security_token` | secret string | Runtime-only user input |
| `bearer_jwt` | secret string | Runtime-only auth token returned by Ghostfolio |
| `started_at` | timestamp | Session start time |
| `authenticated_at` | timestamp nullable | Set after `POST /auth/anonymous` succeeds |

Relationships:

- Belongs to one `RegisteredLocalUser` for the duration of the active run.
- Used by one `SyncAttempt`.

Validation rules:

- Never persisted.
- A session must not exist without a selected or unlocked `RegisteredLocalUser` context.
- Cleared on sync success, sync failure, or application exit.
- Token and JWT must not be logged, printed, or written to crash artifacts intentionally produced by the app.

State transitions:

- `prompted -> authenticated -> cleared`.
- `prompted -> failed -> cleared`.

## SyncAttempt

Purpose: Ephemeral workflow state for one sync execution.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `attempt_id` | UUID string | Correlates in-memory workflow events only |
| `server_origin` | string | Active server for the attempt |
| `started_at` | timestamp | Attempt start |
| `completed_at` | timestamp nullable | Set on success or failure |
| `status` | enum | `started`, `authenticated`, `retrieved`, `normalized`, `validated`, `persisted`, `failed`, `aborted` |
| `failure_reason` | enum nullable | User-visible non-secret failure category |

Relationships:

- Uses one `GhostfolioSession`.
- Produces zero or one new `ProtectedActivityCache`.

Validation rules:

- Failed attempts do not create or retain a new `RegisteredLocalUser`.
- Replacement sync after server mismatch must write the new snapshot only after the attempt reaches `validated`.

State transitions:

- `started -> authenticated -> retrieved -> normalized -> validated -> persisted`.
- `started -> failed`.
- `started -> aborted` when the user declines server replacement.

## ProtectedActivityCache

Purpose: Normalized, deduplicated, validated activity history reused across sessions.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `synced_at` | timestamp | Last successful refresh time |
| `source_api_base_path` | string | Baseline `api/v1` |
| `activity_count` | integer | Count after normalization and deduplication |
| `activities` | `ActivityRecord[]` | Chronological normalized records |
| `available_years` | integer array | Distinct years present in the normalized timeline |
| `scope_reliability` | enum | `reliable`, `partial`, `unavailable` |

Relationships:

- Belongs to one `RegisteredLocalUser`.
- Contains many `ActivityRecord` entries.

Validation rules:

- Persist only after chronological sorting, exact duplicate removal, and defensibility validation complete.
- Unsupported event types that affect holdings cause the whole sync to fail.
- Missing or contradictory data that prevents defensible basis calculation causes the whole sync to fail.

## ActivityRecord

Purpose: One normalized acquisition, disposal, or movement event derived from Ghostfolio activity history.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Ghostfolio activity identifier |
| `occurred_at` | timestamp | Normalized event time |
| `activity_type` | enum | Supported Ghostfolio event type |
| `asset_symbol` | string | Asset identifier used in reporting |
| `asset_name` | string nullable | Human-readable symbol profile name |
| `base_currency` | string nullable | Report currency label when provided by source data; not converted in this feature slice |
| `quantity` | decimal string | Exact asset quantity |
| `unit_price` | decimal string | Exact unit price, may be zero for non-fiat movement semantics |
| `gross_value` | decimal string | Source value before fee treatment as used by the domain |
| `fee_amount` | decimal string | Fee in the report currency or source base currency |
| `comment` | string nullable | Required for interpreting zero-priced movements safely |
| `data_source` | string nullable | Preserve source system identity as opaque data |
| `source_scope` | `SourceHoldingScope` nullable | Optional account-derived wallet-equivalent scope |
| `raw_hash` | string | Hash of normalized source fields used for exact-duplicate detection |

Relationships:

- Optionally references one `SourceHoldingScope`.

Validation rules:

- `occurred_at`, `activity_type`, `asset_symbol`, and `quantity` are mandatory.
- Unit price `0` is valid only when the business rules can interpret the movement from direction and explanatory context.
- Monetary inputs are consumed as provided and are not cross-currency converted in this feature slice.
- Exact duplicates are removed by `raw_hash` after canonical normalization.
- Unsupported holding-affecting event types fail the sync instead of being skipped.

## SourceHoldingScope

Purpose: Ghostfolio account, or equivalent future source grouping, normalized as the application's wallet concept when wallet-scoped cost-basis matching is selected.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `scope_id` | string | Stable account identifier, or wallet identifier if future upstream data adds it |
| `scope_name` | string nullable | Optional human-readable label |
| `scope_kind` | enum | `account`, `wallet`, `unknown` |
| `reliability` | enum | `reliable`, `partial`, `unavailable` |

Relationships:

- May be referenced by many `ActivityRecord` entries.

Validation rules:

- `reliability != reliable` for the relevant asset timeline forces fallback from account-derived wallet matching to asset-level FIFO.

## ReportRequest

Purpose: User-selected parameters for one report-generation run.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `tax_year` | integer | Must exist in `available_years` |
| `cost_basis_method` | enum | `fifo`, `lifo`, `hifo`, `average_cost`, `unit_by_unit_wallet_scoped` |
| `output_format` | enum | Baseline always `pdf` |

Relationships:

- Consumes one `ProtectedActivityCache`.
- Produces one `CapitalGainsReport`.

Validation rules:

- Report generation is blocked until setup is complete and a successful sync exists.
- The TUI must show a jurisdiction-neutral informational message whenever the cost basis method changes.

State transitions:

- `draft -> calculated -> rendered`.
- `draft -> failed`.

## AssetPositionTimeline

Purpose: Derived per-asset chronological ledger used to calculate basis and report inclusion or exclusion.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_symbol` | string | Grouping key |
| `opening_position_before_year` | decimal string | Holdings carried into the report year |
| `in_year_events` | `ActivityRecord[]` | Events within the selected year |
| `closing_position_end_of_year` | decimal string | Holdings after in-year processing |
| `liquidated_before_year` | boolean | Drives reference-list inclusion |
| `liquidated_during_year` | boolean | Drives main report inclusion |

Relationships:

- Derived from many `ActivityRecord` entries.

Validation rules:

- Activity after the selected year is ignored for that report run.
- Assets with open positions at year end or liquidations during the year remain in the main report sections.

## AssetReportEntry

Purpose: Summary row in the gains and losses section for one asset included in the main report sections.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `asset_symbol` | string | Asset identifier shown in the summary |
| `asset_name` | string nullable | Optional human-readable asset name |
| `net_gain_loss_amount` | decimal string | Net gain, loss, or zero result for the selected year in the report currency |
| `result_kind` | enum | `gain`, `loss`, `zero` |

Relationships:

- Belongs to one `CapitalGainsReport`.

Validation rules:

- Exactly one entry exists for each asset included in the main report sections.
- A zero result remains a valid summary entry and must not be omitted.

## CapitalGainsReport

Purpose: Final calculated and rendered yearly report.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `tax_year` | integer | Selected report year |
| `cost_basis_method` | enum | Method used consistently across included disposals |
| `generated_at` | timestamp | Render time |
| `summary_entries` | `AssetReportEntry[]` | Per-asset gains and losses summary entries, always first section |
| `previously_liquidated_assets` | string array | Reference-only list |
| `detail_sections` | `AssetPositionTimeline[]` | Grouped by asset |
| `pdf_output_path` | string | User-selected or generated path |

Relationships:

- Derived from one `ReportRequest` and one `ProtectedActivityCache`.
- Contains many `AssetReportEntry`.
- Contains many `AssetPositionTimeline`.

Validation rules:

- Gains/losses summary must appear before detailed sections.
- Assets fully liquidated before the selected year and not reopened must be excluded from main sections and listed only in the reference list.
