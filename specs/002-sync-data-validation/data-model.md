# Data Model: Sync Data Validation

## Modeling Notes

This slice persists only the bootstrap setup state needed before Ghostfolio token entry. Ghostfolio tokens, returned JWTs, and Ghostfolio activity payloads remain runtime-only and are cleared when the validation attempt ends.

The persisted setup model below is intentionally narrow. It is sufficient to gate first-run setup and reuse the selected Ghostfolio origin between runs, but it does not create any local user profile, encrypted activity cache, or report-ready ledger.

This data model also contains validation-only runtime entities that exist only because this slice stops at communication verification. Later slices that implement real Ghostfolio retrieval, normalization, and protected persistence must not keep those probe entities permanently if equivalent real-sync models replace them.

## Forward Slice Evolution

| Entity | Transitional Status | Next-slice expectation |
|--------|---------------------|------------------------|
| `AppSetupConfig` | Keep, but narrow | Retain only as bootstrap configuration readable before token entry. If later setup fields become person-linked or financial, move them into future token-protected setup or profile models instead of expanding this entity unsafely |
| `GhostfolioSession` | Keep and expand | Reuse as the authenticated runtime context for real sync and persistence workflows |
| `SyncValidationAttempt` | Update or rename | Align with a broader future `SyncAttempt` model that covers retrieval, normalization, validation, and persistence states |
| `ActivitiesProbeResponse` | Remove | Replace with the real Ghostfolio retrieval DTOs and pagination inputs used by full sync |
| `ActivityProbeEntry` | Remove | Replace with normalized `ActivityRecord`-style models and related source-scope data once real sync exists |
| `ValidationOutcome` | Update or merge | Replace with the broader sync-result and UI-status model used once successful communication can also produce persisted data and later report readiness |

## AppSetupConfig

Purpose: Machine-local bootstrap configuration loaded before any Ghostfolio token prompt.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `schema_version` | integer | persisted | Version for future bootstrap-file migrations |
| `setup_complete` | boolean | persisted | Gates access to the main workflow selection screen |
| `server_mode` | enum | persisted | `ghostfolio_cloud` or `custom_origin` |
| `server_origin` | string | persisted | Canonical absolute origin, default `https://ghostfol.io` |
| `allow_dev_http` | boolean | persisted | True only when the stored origin was accepted in explicit development mode |
| `updated_at` | timestamp | persisted | Last successful setup save time |

Relationships:

- Loaded by the application on startup.
- Read by every `GhostfolioSession` and `SyncValidationAttempt`.

Validation rules:

- `server_origin` must contain scheme, host, and optional port only.
- `server_origin` must not contain path, query, fragment, or user info.
- Production usage rejects `http` origins.
- Development mode may allow `http` only when explicitly enabled at startup.
- This entity must never include the Ghostfolio token, a token verifier, JWTs, or Ghostfolio-returned activity payloads.

State transitions:

- `absent -> complete` after the first successful setup save.
- `complete -> updated` when the user changes the selected server origin.
- `complete -> deleted` when the user removes local bootstrap state.

Future slice note:

- This entity is expected to remain as bootstrap-only configuration, but later slices must not turn it into a container for user-linked or financial data. Those fields belong in the future protected user and setup models.

## GhostfolioSession

Purpose: Ephemeral authenticated runtime state for one communication-validation run.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `server_origin` | string | runtime only | Selected canonical origin for this attempt |
| `security_token` | secret string | runtime only | User-entered Ghostfolio token |
| `auth_token` | secret string nullable | runtime only | JWT returned by anonymous auth |
| `started_at` | timestamp | runtime only | Attempt start time |
| `authenticated_at` | timestamp nullable | runtime only | Set only after auth succeeds |

Relationships:

- Created from one `AppSetupConfig`.
- Used by one `SyncValidationAttempt`.

Validation rules:

- Never persisted.
- Cleared on success, failure, cancellation, or application exit.
- Must never be written to logs, traces, dumps, or user-visible messages.

State transitions:

- `prompted -> authenticated -> cleared`
- `prompted -> failed -> cleared`

Future slice note:

- This entity is expected to remain, but later slices must expand it beyond the validation probe so it can support full sync execution and later protected-data workflows.

## SyncValidationAttempt

Purpose: Ephemeral workflow state for one execution of the `Sync Data` feature.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `attempt_id` | string | runtime only | Correlation identifier for in-memory workflow state only |
| `status` | enum | runtime only | `idle`, `authenticating`, `requesting_activities`, `validating_payload`, `success`, `failure` |
| `failure_reason` | enum nullable | runtime only | `invalid_token`, `connectivity`, `unexpected_status`, `invalid_payload`, `incompatible_server` |
| `started_at` | timestamp | runtime only | Attempt start |
| `completed_at` | timestamp nullable | runtime only | Final outcome time |

Relationships:

- Uses one `GhostfolioSession`.
- Produces one `ValidationOutcome`.

Validation rules:

- A validation attempt can start only when `AppSetupConfig.setup_complete == true`.
- Failure must not modify persisted setup state unless the user explicitly changed setup first.
- Failure must allow a new attempt without forcing setup to be repeated.
- No Ghostfolio payload is persisted after the attempt ends.

State transitions:

- `idle -> authenticating -> requesting_activities -> validating_payload -> success`
- `idle -> authenticating -> failure`
- `idle -> requesting_activities -> failure`
- `idle -> validating_payload -> failure`

Future slice note:

- This entity is transitional. Later slices should evolve or replace it with the broader real-sync lifecycle model instead of preserving a separate validation-only attempt abstraction indefinitely.

## ActivitiesProbeResponse

Purpose: Minimal validated representation of the first page returned by `GET /api/v1/activities` for this slice.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `count` | integer | runtime only | Total number of activities reported by the server |
| `activities` | `ActivityProbeEntry[]` | runtime only | First page fetched with `take=1` |

Relationships:

- Owned by one `SyncValidationAttempt` during payload validation.
- Contains zero or one `ActivityProbeEntry` in this slice.

Validation rules:

- `count` must be a non-negative integer.
- `activities` must be present and must be an array.
- With `take=1`, `len(activities)` must be `0` or `1`.
- If `count == 0`, `activities` may be empty and the response still counts as successful communication.
- If `count > 0`, the response must contain one `ActivityProbeEntry` on the first page.
- Unknown extra fields are ignored.

Future slice note:

- This entity is validation-only and must be removed when later slices introduce full-history retrieval and pagination models for real sync.

## ActivityProbeEntry

Purpose: Minimal activity item shape required to treat Ghostfolio communication as structurally compatible in this slice.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `id` | string | runtime only | Source activity identifier |
| `date` | string | runtime only | Parseable timestamp string from Ghostfolio |
| `type` | string | runtime only | Non-empty source activity type |

Relationships:

- Belongs to one `ActivitiesProbeResponse`.

Validation rules:

- `id` must be a non-empty string.
- `date` must be a non-empty string and parse as a timestamp.
- `type` must be a non-empty string.
- This slice does not yet enforce full domain support such as `BUY` or `SELL`-only normalization across the retrieved history.

Future slice note:

- This entity is validation-only and must be replaced by the normalized activity models used for real sync, persistence, and later reporting.

## ValidationOutcome

Purpose: User-visible result produced after a sync validation attempt finishes.

Fields:

| Field | Type | Persistence | Notes |
|-------|------|-------------|-------|
| `success` | boolean | runtime only | True only when auth and minimal activities validation both succeed |
| `summary_message` | string | runtime only | Primary user-facing result message |
| `detail_reason` | enum | runtime only | `communication_ok`, `invalid_token`, `connectivity_failure`, `unexpected_response`, `invalid_payload` |
| `follow_up_note` | string nullable | runtime only | Additional user guidance such as "data was not stored" |

Relationships:

- Produced by one `SyncValidationAttempt`.

Validation rules:

- Success messages must explicitly state that no Ghostfolio data was stored and no reporting flow is available yet.
- Failure messages must explain the failure category without exposing the Ghostfolio token, JWT, or raw unprotected payload details.
- Outcomes are transient and are not shown again after restart unless the workflow fails again.

Future slice note:

- This entity should not remain as a standalone long-term domain model once later slices add real sync completion, cache update, and report-readiness states. It should evolve into or merge with the broader workflow-status model.
