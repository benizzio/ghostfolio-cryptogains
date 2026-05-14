# Data Model: Store Activity Data

## Modeling Notes

This slice keeps the plaintext bootstrap setup model from `specs/002-sync-data-validation/` and adds a separate protected snapshot model for all user-specific sync state and activity history. Bootstrap state stays readable before token entry. Protected snapshot data stays unreadable without the matching Ghostfolio security token.

All persisted activity timestamps keep the source offset that Ghostfolio provided so available report years can be derived from the source calendar date rather than from machine-local or forced UTC conversion.

No raw Ghostfolio payloads, Ghostfolio tokens, JWTs, or reusable token verifiers are persisted.

## Forward Slice Evolution

| Entity | Status From `002` | `003` Decision |
|--------|-------------------|----------------|
| `AppSetupConfig` | Keep | Remains bootstrap-only and must not gain user-linked or financial fields |
| `GhostfolioSession` | Expand | Grows from validation-only auth state into full sync runtime state |
| `ActivitiesProbeResponse` | Remove | Replaced by full-history Ghostfolio DTOs and mappers |
| `ActivityProbeEntry` | Remove | Replaced by normalized stored `ActivityRecord` values |
| `SyncValidationAttempt` | Evolve | Replaced by broader `SyncAttempt` lifecycle states covering retrieval, normalization, validation, and protected write |
| `ValidationOutcome` | Evolve | Becomes a broader sync result model that can report storage and compatibility outcomes |

## AppSetupConfig

Purpose: Startup-readable machine-local bootstrap configuration loaded before any Ghostfolio token prompt.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `schema_version` | integer | persisted | Bootstrap file version |
| `setup_complete` | boolean | persisted | Gates access to the main menu |
| `server_mode` | enum | persisted | `ghostfolio_cloud` or `custom_origin` |
| `server_origin` | string | persisted | Canonical selected Ghostfolio origin |
| `allow_dev_http` | boolean | persisted | True only when an `http` origin was accepted in explicit development mode |
| `updated_at` | timestamp | persisted | Last successful setup save time |

Relationships:

- Loaded on application startup.
- Drives server-scoped snapshot discovery.
- Remains separate from the protected snapshot model.

Validation rules:

- Must remain limited to bootstrap-only fields.
- Must not include activity data, available years, local-user metadata created by this slice, or any stored-data version markers.
- A valid bootstrap config is required before any sync attempt starts.

State transitions:

- `absent -> complete` after first successful setup save.
- `complete -> updated` when the selected server changes.
- `complete -> deleted` when the user removes bootstrap setup.

## EncryptedSnapshotEnvelope

Purpose: Token-locked on-disk container for one protected snapshot.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `file_id` | UUID string | cleartext filename | Random opaque file identifier |
| `format_version` | integer | cleartext header | Envelope compatibility marker |
| `server_discovery_key` | hex string | cleartext header | `SHA-256(canonical_server_origin)` used to limit unlock attempts to the selected server |
| `kdf_name` | enum | cleartext header | Baseline `argon2id` |
| `kdf_memory_kib` | integer | cleartext header | KDF memory parameter |
| `kdf_iterations` | integer | cleartext header | KDF iteration parameter |
| `kdf_parallelism` | integer | cleartext header | KDF parallelism parameter |
| `salt` | bytes | cleartext header | Fresh random salt on every rewrite |
| `nonce` | bytes | cleartext header | Fresh random GCM nonce on every rewrite |
| `ciphertext` | bytes | encrypted payload | Includes the authenticated payload and tag |

Relationships:

- Wraps exactly one `SnapshotPayload`.

Validation rules:

- Header bytes are authenticated as AEAD additional authenticated data.
- `server_discovery_key` must be derived from the canonical bootstrap server origin, not from the Ghostfolio token.
- The cleartext header must not include the Ghostfolio token, a token hash, a token verifier, available years, or user-readable profile data.
- Unsupported `format_version` produces a compatibility failure before decrypt.

## StoredDataVersion

Purpose: Version markers persisted with the protected snapshot so the application can reject unsupported stored data safely.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `envelope_format_version` | integer | Duplicates the envelope compatibility marker inside the payload for traceability |
| `payload_schema_version` | integer | Version of the decrypted snapshot payload layout |
| `activity_model_version` | integer | Version of the normalized stored-activity model |
| `written_by_app_version` | string | Application version that wrote the snapshot |

Relationships:

- Belongs to one `SnapshotPayload`.

Validation rules:

- Unsupported versions fail safely and leave the snapshot untouched.
- The application must not partially load or auto-overwrite a snapshot whose stored-data version is unsupported.

## SnapshotPayload

Purpose: Decrypted protected state for one isolated local user context.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `stored_data_version` | `StoredDataVersion` | Required compatibility metadata |
| `registered_local_user` | `RegisteredLocalUser` | Protected local user metadata |
| `setup_profile` | `SetupProfile` | Protected selected-server reference and related sync profile data |
| `protected_activity_cache` | `ProtectedActivityCache` | Normalized stored history and sync metadata |
| `available_report_years` | integer array | Distinct years derived from stored activity timestamps |

Relationships:

- Contains exactly one `StoredDataVersion`.
- Contains exactly one `RegisteredLocalUser`.
- Contains exactly one `SetupProfile`.
- Contains exactly one `ProtectedActivityCache`.

Validation rules:

- Persist only after successful auth, full retrieval, normalization, validation, and protected write preparation.
- Rewrite atomically as one whole payload.
- Empty history remains valid when `available_report_years` is empty and the cache reflects a successful empty sync.

## RegisteredLocalUser

Purpose: Protected local user metadata created only after a full successful sync.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `local_user_id` | UUID string | Stable internal identifier for this protected snapshot context |
| `created_at` | timestamp | First successful protected snapshot creation time |
| `updated_at` | timestamp | Last successful protected rewrite time |
| `last_successful_sync_at` | timestamp | Last successful full sync completion time |

Relationships:

- Owns one `SetupProfile`.
- Owns one `ProtectedActivityCache`.

Validation rules:

- Created only after the first full successful sync.
- Bound to the Ghostfolio token only through unlockability. No stored token copy, token hash, or reusable token verifier is allowed.

State transitions:

- `absent -> persisted` after successful first sync.
- `persisted -> replaced` after confirmed server replacement sync completes successfully.
- `persisted -> unchanged` after failed refresh, invalid token, wrong-token unlock denial, or incompatible new data.

## SetupProfile

Purpose: Protected selected-server reference and related sync profile data stored with the protected activity cache.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `server_origin` | string | Canonical Ghostfolio origin for the protected snapshot |
| `server_mode` | enum | `ghostfolio_cloud` or `custom_origin` |
| `last_validated_at` | timestamp | Last successful server compatibility check |
| `source_api_base_path` | string | Baseline `api/v1` |

Relationships:

- Belongs to one `RegisteredLocalUser`.

Validation rules:

- Used for server-mismatch comparison when a readable snapshot is active in memory.
- Must match the bootstrap server origin before a non-replacement refresh starts.

State transitions:

- `current -> pending_replacement` when the active readable snapshot's protected server origin differs from the current bootstrap origin.
- `pending_replacement -> current` only after replacement sync succeeds and the new snapshot is written atomically.

## ProtectedActivityCache

Purpose: Normalized, deduplicated, validated activity history and sync metadata reused across sessions.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `synced_at` | timestamp | Time of the last successful sync |
| `retrieved_count` | integer | Number of source records retrieved before normalization |
| `activity_count` | integer | Number of stored activities after normalization and duplicate removal |
| `scope_reliability` | enum | `reliable`, `partial`, `unavailable` |
| `activities` | `ActivityRecord[]` | Chronologically ordered normalized activity history |

Relationships:

- Belongs to one `RegisteredLocalUser`.
- Contains many `ActivityRecord` values.

Validation rules:

- Persist only after full pagination succeeds.
- Persist only after chronological sorting, duplicate removal, exact-decimal parsing, activity-type validation, zero-price rule validation, available-year derivation, and defensibility checks complete.
- A valid empty history uses `retrieved_count = 0`, `activity_count = 0`, and an empty `activities` list.

## ActivityRecord

Purpose: One normalized Ghostfolio `BUY` or `SELL` event stored for future reporting use.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `source_id` | string | Ghostfolio activity identifier |
| `occurred_at` | RFC3339 timestamp string | Stored with the source offset preserved |
| `activity_type` | enum | `BUY` or `SELL` only |
| `asset_symbol` | string | Asset identifier used for future reporting |
| `asset_name` | string nullable | Optional human-readable asset name |
| `base_currency` | string nullable | Source base currency label when provided |
| `quantity` | decimal string | Exact quantity value |
| `unit_price` | decimal string | Exact unit price |
| `gross_value` | decimal string | Exact gross value |
| `fee_amount` | decimal string | Exact fee value |
| `comment` | string nullable | Required for zero-priced `SELL` explanation |
| `data_source` | string nullable | Optional opaque source-system label |
| `source_scope` | `SourceHoldingScope` nullable | Optional source grouping data |
| `raw_hash` | string | Exact-duplicate detection hash from normalized fields |

Relationships:

- Optionally references one `SourceHoldingScope`.

Validation rules:

- `source_id`, `occurred_at`, `activity_type`, `asset_symbol`, `quantity`, `unit_price`, and `gross_value` are mandatory.
- `activity_type` must be `BUY` or `SELL`.
- `BUY` requires `unit_price > 0`.
- `SELL` with `unit_price = 0` is valid only when `comment` is present and is treated as a non-taxable holding reduction for future reporting.
- Monetary and quantity fields must remain exact decimal strings and must not be rounded or converted in this slice.
- `raw_hash` is computed after normalization and is used only for exact-duplicate removal.

## SourceHoldingScope

Purpose: Preserved source grouping data from Ghostfolio that a future reporting slice may use to narrow or broaden scope-local reporting behavior.

Fields:

| Field | Type | Notes |
|-------|------|-------|
| `scope_id` | string | Source account identifier or future equivalent |
| `scope_name` | string nullable | Optional human-readable label |
| `scope_kind` | enum | `account`, `wallet`, `unknown` |
| `reliability` | enum | `reliable`, `partial`, `unavailable` |

Relationships:

- May be referenced by many `ActivityRecord` values.

Validation rules:

- Missing or unreliable scope data does not fail sync by itself.
- `reliability` informs future reporting decisions but does not expose any reporting choice in this slice.

## GhostfolioSession

Purpose: Ephemeral authenticated runtime state for one full sync attempt.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `server_origin` | string | runtime only | Canonical selected server for the attempt |
| `security_token` | secret string | runtime only | User-entered Ghostfolio security token |
| `auth_token` | secret string | runtime only | Bearer JWT returned by Ghostfolio |
| `started_at` | timestamp | runtime only | Session start time |
| `authenticated_at` | timestamp nullable | runtime only | Set after anonymous auth succeeds |

Relationships:

- Uses one `AppSetupConfig`.
- May unlock or create one `EncryptedSnapshotEnvelope`.
- Is used by one `SyncAttempt`.

Validation rules:

- Never persisted.
- Cleared on success, failure, cancel, or application exit.
- Must never be written to logs, user-visible messages, or diagnostics.

## SyncAttempt

Purpose: Ephemeral workflow state for one `Sync Data` execution.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `attempt_id` | string | runtime only | Correlation identifier for in-memory workflow state |
| `status` | enum | runtime only | `started`, `discovering_snapshot`, `unlocking_snapshot`, `authenticating`, `retrieving_history`, `normalizing`, `validating`, `persisting`, `success`, `failed`, `aborted` |
| `failure_reason` | enum nullable | runtime only | User-visible non-secret outcome category |
| `started_at` | timestamp | runtime only | Attempt start time |
| `completed_at` | timestamp nullable | runtime only | Attempt end time |
| `server_mismatch_confirmed` | boolean | runtime only | True only after the user confirms replacement |

Relationships:

- Uses one `GhostfolioSession`.
- May read one active `SnapshotPayload`.
- May produce one replacement `SnapshotPayload`.

Validation rules:

- A failed or aborted attempt must not create or overwrite a protected snapshot.
- A successful attempt persists only after `persisting` completes atomically.
- Incompatible new data leaves the currently active readable snapshot unchanged.

State transitions:

- `started -> discovering_snapshot -> unlocking_snapshot -> authenticating -> retrieving_history -> normalizing -> validating -> persisting -> success`
- `started -> aborted` when replacement is declined.
- `* -> failed` on any auth, retrieval, normalization, validation, compatibility, or write failure.

## Derived Runtime Concepts

- `ServerScopedCandidateSet`: the set of snapshot headers whose `server_discovery_key` matches the current bootstrap `server_origin`. Only this set may be tried with the supplied token.
- `ActiveReadableSnapshot`: the decrypted protected snapshot currently held in memory for the running application instance. Server-mismatch confirmation compares this snapshot's `setup_profile.server_origin` with the current bootstrap `server_origin`.
- `ReplacementWriteCandidate`: the fully validated replacement payload prepared in memory before the temp-file write. It becomes visible only after atomic rename succeeds.
