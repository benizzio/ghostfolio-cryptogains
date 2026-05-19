# Contract: Ghostfolio Sync And Protected Storage Boundary

## Scope

This contract defines the external Ghostfolio HTTP behavior and the local protected-storage behavior required for the `Store Activity Data` slice.

Reference sources of truth:

- `specs/001-ghostfolio-gains-reporting/contracts/ghostfolio-sync.md`
- `specs/002-sync-data-validation/contracts/ghostfolio-sync-validation.md`
- `specs/003-store-activity-data/spec.md`

Observed upstream review evidence refreshed on 2026-05-17:

- Ghostfolio release `3.3.0` published on 2026-05-14.
- `apps/api/src/app/auth/auth.controller.ts` exposes `@Post('anonymous')` and returns `{ authToken }` on success while rejected access tokens surface as HTTP `403 Forbidden`.
- `apps/api/src/app/activities/activities.controller.ts` exposes `GET /activities` with `skip`, `take`, `sortColumn`, and `sortDirection` query parameters and returns `{ activities, count }`.
- `apps/api/src/app/activities/activities.service.ts` applies `skip` and `take`, defaults to date ordering, and appends `id` as a deterministic tie-break when explicit sorting is requested.
- Ghostfolio activity capture is date-based, so the `/api/v1/activities` `date` field must be preserved exactly for storage and year derivation but its time-of-day precision is not treated as authoritative for same-asset local ordering.
- `apps/api/src/app/user/user.controller.ts` exposes authenticated `GET /user`.
- `libs/common/src/lib/interfaces/user.interface.ts` defines `User.settings`.
- `libs/common/src/lib/interfaces/user-settings.interface.ts` defines `UserSettings.baseCurrency?: string`.
- `libs/common/src/lib/interfaces/activities.interface.ts` exposes `Activity extends Order` plus `feeInAssetProfileCurrency`, `feeInBaseCurrency`, `unitPriceInAssetProfileCurrency`, `value`, `valueInBaseCurrency`, optional `account`, optional `error`, optional `tagIds`, optional `tags`, optional `updateAccountBalance`, and required `SymbolProfile`.
- `prisma/schema.prisma` defines the underlying `Order.currency` column as nullable (`String?`), while `Order.comment`, `Order.accountId`, and `Order.accountUserId` are also nullable and `Order.fee`, `Order.quantity`, and `Order.unitPrice` are non-null in persisted orders.
- `libs/common/src/lib/interfaces/enhanced-symbol-profile.interface.ts` exposes `EnhancedSymbolProfile.currency?: string`, `name?: string`, and other optional display fields, so the public response contract does not guarantee `SymbolProfile.currency` on every activity row even though the Prisma `SymbolProfile.currency` column is non-null.
- `apps/api/src/app/activities/activities.service.ts` computes derived currency-converted amounts using `order.currency ?? order.SymbolProfile.currency`, which empirically confirms Ghostfolio can emit activities where `Order.currency` is null and the service falls back to symbol-profile currency for conversion work.

## Compatibility Rules

- The selected Ghostfolio origin still comes from the bootstrap setup model introduced in `002`.
- The client targets the observed `api/v1` base path.
- The Ghostfolio security token is sent only to the currently selected canonical Ghostfolio origin.
- Production-like custom origins require `https`. Development-only `http` origins remain allowed only when the app is launched in explicit development mode.
- Sync succeeds only when all of these complete successfully in one workflow:
  - token entry
  - anonymous auth
  - authenticated user retrieval for sync base-currency context
  - full paginated activity retrieval
  - normalization and validation of the complete supported history
  - atomic protected snapshot write or refresh
- Reporting, cached-data browsing, report preview, gains-or-losses calculation, and PDF generation remain unavailable in this slice even after sync succeeds.

## User-Visible Outcome Categories

This slice may surface these non-secret outcome categories:

- `rejected token`
- `timeout`
- `connectivity problem`
- `unsuccessful server response`
- `incompatible server contract`
- `unsupported activity history`
- `unsupported stored-data version`
- `incompatible new sync data`
- `server replacement cancelled`

Each finished sync shows exactly one final outcome category.

## Authentication Contract

### Request

`POST /api/v1/auth/anonymous`

Headers:

- `Content-Type: application/json`

Body:

```json
{
  "accessToken": "<ghostfolio-security-token>"
}
```

### Successful Response

HTTP `200 OK`

```json
{
  "authToken": "<jwt>"
}
```

### Runtime Validation Rules

- The response must declare a JSON-compatible content type.
- The body must parse as JSON.
- `authToken` must exist and be a non-empty string.
- The returned JWT remains runtime-only for the active sync workflow.

### Failure Handling Rules

- HTTP `403 Forbidden` maps to `rejected token`.
- Transport timeout maps to `timeout`.
- Network reachability failures map to `connectivity problem`.
- Other non-2xx responses map to `unsuccessful server response` unless the response itself proves contract incompatibility.
- Unsupported content type, malformed JSON, missing `authToken`, or empty `authToken` maps to `incompatible server contract`.

## Authenticated User Contract

### Request

`GET /api/v1/user`

Headers:

- `Authorization: Bearer <jwt>`

### Successful Response Shape

HTTP `200 OK`

```json
{
  "settings": {
    "baseCurrency": "USD"
  }
}
```

### Runtime Validation Rules

- The response must declare a JSON-compatible content type.
- The body must parse as JSON.
- `settings` must exist.
- `settings.baseCurrency` is optional in the verified public `UserSettings` interface.
- `settings.baseCurrency` is used as the sync base-currency context when base-valued activity fields are present.
- If this slice receives base-valued activity amounts but authenticated user data still does not provide `settings.baseCurrency`, that is a current-slice contract failure.

### Failure Handling Rules

- HTTP `401 Unauthorized` or `403 Forbidden` during user retrieval maps to `unsuccessful server response`.
- HTTP `404 Not Found` maps to `incompatible server contract`.
- Other non-2xx responses map to `unsuccessful server response` unless the response itself proves contract incompatibility.
- Unsupported content type, malformed JSON, or missing `settings` maps to `incompatible server contract`.

## Activities Retrieval Contract

### Request

`GET /api/v1/activities`

Headers:

- `Authorization: Bearer <jwt>`

Required query parameters:

- `skip`
- `take`
- `sortColumn=date`
- `sortDirection=asc`

The client may vary `skip` and `take`, but it must preserve full-history correctness.

### Successful Response Shape

HTTP `200 OK`

```json
{
  "activities": [
    {
      "id": "activity-id",
      "date": "2026-01-31T10:00:00+01:00",
      "type": "BUY",
      "quantity": 1.25,
      "currency": "CHF",
      "unitPrice": 50000,
      "value": 62500,
      "fee": 25,
      "unitPriceInAssetProfileCurrency": 52000,
      "feeInAssetProfileCurrency": 26,
      "valueInBaseCurrency": 62500,
      "feeInBaseCurrency": 25,
      "comment": "optional",
      "SymbolProfile": {
        "symbol": "BTC",
        "name": "Bitcoin",
        "currency": "EUR"
      },
      "account": {
        "id": "optional-account-id",
        "name": "optional-account-name"
      }
    }
  ],
  "count": 1
}
```

The upstream Ghostfolio schema is larger than this example. The client depends only on the fields needed for reporting-ready normalization in this slice.

The sync base-currency context is not sourced from a required field on each activity row. The client derives that context from authenticated `GET /api/v1/user` response data and applies it to base-valued activity amounts during mapping and validation.

### Activity Field Nullability And Presence Rules

Verified from Ghostfolio `main` as of 2026-05-17:

- Required and non-null in the verified shared `Activity` interface, or non-null in the upstream persisted `Order` model where noted:
  - `id`
  - `date`
  - `type`
  - `quantity`
  - `fee`
  - `unitPrice`
  - `value`
  - `feeInAssetProfileCurrency`
  - `feeInBaseCurrency`
  - `unitPriceInAssetProfileCurrency`
  - `valueInBaseCurrency`
  - `SymbolProfile`
- Non-null in the upstream persisted `Order` model, but not separately required by this slice as an independent JSON field when the nested `SymbolProfile` object is present:
  - `symbolProfileId`
- Present in the public `Activity` interface but nullable in the upstream persisted order model or optional in the shared interface:
  - `currency`: nullable because `Order.currency` is `String?`
  - `comment`: nullable because `Order.comment` is `String?`
  - `accountId`: nullable because `Order.accountId` is `String?`
  - `accountUserId`: nullable because `Order.accountUserId` is `String?`
  - `account`: optional on `Activity`
  - `error`: optional on `Activity`
  - `tagIds`: optional on `Activity`
  - `tags`: optional on `Activity`
  - `updateAccountBalance`: optional on `Activity`
- Nested `SymbolProfile` fields relevant to this slice:
  - `symbol`: required in `EnhancedSymbolProfile`
  - `currency`: optional in `EnhancedSymbolProfile`, so the public response contract does not guarantee it even though the Prisma `SymbolProfile.currency` column is non-null
  - `name`: optional in `EnhancedSymbolProfile`
- Fields not guaranteed per activity row in the verified public contract:
  - no per-activity `baseCurrency` field exists in `Activity`
  - base-currency context comes from authenticated `GET /api/v1/user` via `User.settings.baseCurrency`

Current-slice contract interpretation:

- `currency = null` is an allowed upstream value and must be preserved as returned.
- `comment = null` is an allowed upstream value except where this slice separately requires an explanatory comment for a zero-priced `SELL`.
- `account` and tag-related fields are optional preserved inputs, not current-slice required inputs.
- `SymbolProfile.currency` must be treated as not guaranteed by the public response contract even though current source wiring usually populates it.
- Current-slice verification rerun on 2026-05-18 covers `currency = null`, missing `SymbolProfile.currency`, and omitted authenticated-user `settings.baseCurrency`; valid rows with one uninformed tier still succeed when other tiers remain informed.

### Minimum Required Inputs Per Retrieved Activity

The sync must fail when any activity required for holdings reconstruction is missing these normalized inputs:

- source identifier
- timestamp
- activity type
- asset identity
- quantity
- unit price or gross value information sufficient to derive normalized basis inputs
- fee information when present in source data
- explanatory comment for any zero-priced `SELL`

The current slice treats these activity fields as allowed to be absent, optional, or nullable without failing sync on their own:

- `currency` when Ghostfolio returns `null`
- `comment` except for a zero-priced `SELL`
- `account`
- `tagIds`
- `tags`
- `error`
- `updateAccountBalance`
- `SymbolProfile.name`
- `SymbolProfile.currency`
- `settings.baseCurrency` in authenticated-user

Optional preserved inputs:

- account or other source-scope grouping data
- asset display metadata
- opaque source-system identity
- Ghostfolio `quantity`, `unitPrice`, and `value` remain preserved as independent source fields. The current slice does not assume that `value == quantity * unitPrice` exactly for every activity row, because production rows can contain precision or truncation differences while still providing sufficient basis inputs.

### Pagination Rules

- The client must continue paging until the number of retrieved activity items is greater than or equal to `count`.
- A partial first page is never a complete sync.
- Pagination must be monotonic. Contradictory `count`, duplicate page boundaries that cannot be normalized as exact duplicates, or other inconsistent paging behavior fail the sync.
- A valid empty history is a successful retrieval when `count == 0` and `activities` is empty.
- The upstream activities service currently appends `id` as a server-side tie-break after the requested sort order. The client still preserves the original `occurred_at` value, but for same-asset same-day local ordering it ignores Ghostfolio time-of-day precision and uses source calendar date, then `activity_type` with `BUY` before `SELL`, then `source_id`.

### Failure Handling Rules

- HTTP `401 Unauthorized` or `403 Forbidden` during activities retrieval maps to `unsuccessful server response`.
- HTTP `400 Bad Request` maps to `incompatible server contract`.
- Other non-2xx responses map to `unsuccessful server response` unless the response itself proves contract incompatibility.
- Unsupported content type, malformed JSON, missing `activities`, invalid `count`, or contradictory pagination semantics map to `incompatible server contract`.

## Normalization And Validation Rules

- Normalize the complete retrieved history before persistence.
- Preserve timestamps in RFC3339 form with the source offset intact.
- Sort normalized history chronologically.
- For same-asset events that share the same source calendar date, ignore Ghostfolio time-of-day precision and break ties by `activity_type` with `BUY` before `SELL`, then `source_id` ascending.
- Reject the sync if the source calendar date cannot be read from `occurred_at` or if stable deterministic same-asset ordering still cannot be established after exact-duplicate removal.
- Remove exact duplicates only after canonical normalization.
- Supported normalized activity types are only:

```text
BUY
SELL
```

- Any other activity type maps to `unsupported activity history` and fails the full sync.
- A normalized `BUY` with `unit_price = 0` maps to `unsupported activity history` and fails the full sync.
- A normalized `SELL` with `unit_price = 0` is valid only when an explanatory comment is present. It is stored as a non-taxable holding reduction for future reporting use and does not enable reporting in this slice.
- A normalized zero-priced `SELL` without an explanatory comment maps to `unsupported activity history` and fails the full sync.
- Currency-context validation fails only when Ghostfolio omits currency context in all the three tracked currency tiers described below remains available for the preserved monetary data:
  - `currency` in the activity record that is obtained through `Order.currency`
  - `SymbolProfile.currency` for asset-profile-valued fields in the activity record
  - authenticated-user `settings.baseCurrency` for base-valued fields
- all three currency tiers and their related fields are independent and don't need to maintain consistency with each other
- Remaining gaps or contradictions that make future basis calculation non-defensible map to `unsupported activity history` and fail the full sync.
- An exact mismatch between preserved source `value` and a recomputed `quantity * unitPrice` does not by itself make a row unsupported or non-defensible in this slice.
- Missing or unreliable source-scope data does not fail sync by itself. The normalized cache records scope reliability for future reporting.
- `available_report_years` are derived from each normalized timestamp's own offset and calendar date, not from machine-local time or forced UTC year boundaries.

## Protected Snapshot Contract

### Filesystem Layout

- Bootstrap file: `ghostfolio-cryptogains/setup.json`
- Protected snapshots: `ghostfolio-cryptogains/snapshots/<opaque-id>.snapshot`

The bootstrap file remains the only startup-readable local state.

### Cleartext Header Rules

The snapshot header may expose only the minimum non-secret metadata needed before decrypt:

- envelope magic
- `format_version`
- `server_discovery_key`
- KDF algorithm and parameters
- random salt
- AEAD nonce

`server_discovery_key` is derived from the canonical selected Ghostfolio origin and is used only to limit unlock attempts to selected-server candidates.

The header must not expose:

- Ghostfolio token
- token hash
- reusable token verifier
- activity history
- available years
- user-readable profile data

### Encrypted Payload Rules

The payload contains:

- stored-data version markers
- protected setup profile with the stored server reference
- registered local user metadata
- normalized activity cache, including available report years and sync metadata

### Write Rules

- Successful sync writes one complete replacement payload.
- Use temp-file write, `fsync`, and atomic rename.
- The previous readable snapshot remains untouched until the replacement write succeeds.
- A failed, cancelled, or incompatible replacement must not leave a partially readable replacement snapshot behind.

## Unlock, Isolation, And Compatibility Rules

- Before any token attempt, enumerate snapshot headers.
- Only snapshot headers whose `server_discovery_key` matches the current bootstrap `server_origin` are eligible unlock candidates.
- The supplied Ghostfolio security token is tried only against that selected-server candidate set.
- Wrong token or corrupted snapshot data must not expose whether the failure was wrong-token or corruption. The user receives a generic unlock failure rather than any protected detail.
- Unsupported `format_version` maps to `unsupported stored-data version` before decrypt.
- Unsupported payload stored-data version maps to `unsupported stored-data version` after decrypt.
- The application must not auto-migrate, auto-overwrite, or partially load an unsupported snapshot.
- If no selected-server snapshot unlocks but Ghostfolio auth succeeds, the workflow treats the token as a new isolated local-user context and creates a new protected snapshot only after full sync success.
- If no selected-server snapshot unlocks and Ghostfolio rejects the token, the workflow ends with `rejected token` and changes no local data.

## Server Replacement Rules

- Server mismatch confirmation is driven by the active readable snapshot in memory for the current run.
- If the active readable snapshot's protected `server_origin` differs from the current bootstrap `server_origin`, the application must show a confirmation before retrieval starts.
- The confirmation must say that continuing will replace the current protected data tied to that token and server.
- Declining the confirmation maps to `server replacement cancelled` and leaves protected data unchanged.
- Accepting the confirmation starts a replacement sync, but the old snapshot remains active until the new sync completes successfully and the atomic replacement write succeeds.

## Incompatible New Data Rules

- If a new sync retrieves activity data that the current application version cannot safely normalize or persist within its supported stored-data model, the workflow ends with `incompatible new sync data`.
- The newly retrieved in-memory data is discarded.
- Any existing readable protected snapshot remains active and unchanged.
- The application must tell the user that the new data was not stored and that the previously readable protected data remains available.

## Success Requirements For This Slice

The application may report sync success only when all of these are true:

- bootstrap setup is complete
- auth succeeds
- full activity retrieval succeeds
- normalization and validation succeed
- the protected snapshot write succeeds atomically

The success result must also tell the user:

- that activity data was stored for future use
- that no report generation or preview was run
- that cached-data browsing is not part of this slice

## Security Rules

- Never persist the Ghostfolio security token.
- Never persist the Ghostfolio JWT.
- Never persist raw unprotected Ghostfolio payloads.
- Never include the Ghostfolio token, JWT, request body, or raw unprotected payloads in user-visible messages, logs, traces, crash text, or diagnostics produced by project-owned code.
- If wrapped or dependency-generated errors would otherwise surface secret or unprotected payload content, project-owned code must redact or suppress that content before display or persistence.

## Explicitly Deferred Behavior

- cost basis selection
- gains-or-losses calculation
- report generation
- report preview
- cached-activity browsing or export
- recovery or bypass for a lost Ghostfolio security token
