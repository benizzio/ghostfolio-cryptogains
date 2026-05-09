# Contract: Ghostfolio Sync Validation Boundary

## Scope

This contract defines the minimum Ghostfolio HTTP behavior required for the `Sync Data` communication-validation slice. It intentionally documents only the reduced boundary needed to prove that setup, authentication, and activity retrieval work well enough for later slices.

Reference source of truth:

- `specs/001-ghostfolio-gains-reporting/contracts/ghostfolio-sync.md`
- `specs/002-sync-data-validation/spec.md`

## Compatibility Rules

- The configured Ghostfolio origin defaults to `https://ghostfol.io`.
- The client targets the observed `api/v1` base path on the selected origin.
- The client validates runtime compatibility instead of assuming a permanently stable public API contract.
- Production-like custom origins must use `https`.
- `http` custom origins are allowed only when the application was started in explicit development mode.
- The Ghostfolio security token is sent only to the selected canonical Ghostfolio origin.

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

- The response body must parse as a JSON object.
- `authToken` must exist.
- `authToken` must be a non-empty string.

### Failure Handling Rules

- HTTP `403 Forbidden` is treated as an invalid or rejected Ghostfolio security token.
- Any other non-2xx response is treated as communication validation failure.
- Malformed JSON or a missing or empty `authToken` is treated as an incompatible or unexpected server response.

## Activities Probe Contract

### Request

`GET /api/v1/activities?skip=0&take=1&sortColumn=date&sortDirection=asc`

Headers:

- `Authorization: Bearer <jwt>`

### Successful Response Shapes

HTTP `200 OK`

Empty history remains valid:

```json
{
  "activities": [],
  "count": 0
}
```

Non-empty history remains valid when the first activity contains the minimum required fields:

```json
{
  "activities": [
    {
      "id": "activity-id",
      "date": "2026-01-31T10:00:00.000Z",
      "type": "BUY"
    }
  ],
  "count": 1
}
```

### Runtime Validation Rules

- The response body must parse as a JSON object.
- `activities` must exist and be an array.
- `count` must exist and be a non-negative integer.
- For `take=1`, the returned `activities` length must be `0` or `1`.
- If `count == 0`, an empty `activities` array still counts as successful communication.
- If `count > 0`, the first returned activity must contain:
  - non-empty string `id`
  - non-empty string `date` that parses as a timestamp
  - non-empty string `type`
- Unknown extra fields are ignored.

### Failure Handling Rules

- HTTP `401 Unauthorized` is treated as an invalid or expired in-memory JWT.
- HTTP `403 Forbidden` is treated as a permission or server-policy failure.
- HTTP `400 Bad Request` is treated as client or server contract mismatch.
- Any other non-2xx response, malformed JSON, missing `activities`, invalid `count`, or missing minimum activity fields is treated as communication validation failure.

## Success Criteria For This Slice

The application may tell the user that Ghostfolio communication is working only when all of the following are true:

- setup is complete
- auth succeeds under the contract above
- the activities probe succeeds under the contract above
- the application can produce a success result without persisting the returned payload

The success result must also tell the user that data was not stored and is not yet ready for reporting.

## Explicitly Deferred Behavior

The following are intentionally out of scope for this slice and must not be treated as implied by a successful communication validation result:

- full-history pagination
- full activity schema validation
- `BUY` and `SELL`-only enforcement across the full dataset
- zero-priced `BUY` or `SELL` business rules
- chronological normalization and duplicate removal
- local cache persistence
- cost basis calculations
- report generation

## Security Rules

- Never persist the Ghostfolio security token.
- Never persist the JWT returned by Ghostfolio.
- Never persist the activities probe payload.
- Never include the Ghostfolio token, JWT, request body, or raw unprotected payload in user-visible messages, logs, traces, or diagnostics.
