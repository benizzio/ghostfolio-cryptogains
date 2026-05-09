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
- A server is compatible for this slice only when it supports both of these boundaries during one validation attempt:
  - anonymous auth at the expected auth endpoint with the expected request and a structured success response containing a non-empty session credential
  - a one-page activities probe at the expected activities endpoint with the expected query semantics and a structured success response containing a valid `count` and `activities` shape
- A reachable server that responds with contract drift, unsupported authentication behavior, unsupported content type for a response expected to be JSON, contradictory activities-page semantics, unreadable minimum field values, or missing supported endpoints is treated as an incompatible server rather than as a generic connectivity failure.

## Failure Categories Used In This Slice

- `rejected token`: The selected server explicitly rejects the user-supplied Ghostfolio security token.
- `timeout`: The validation attempt exceeds the slice's allowed wait time before a usable response is received.
- `connectivity problem`: The selected server cannot be reached or the connection cannot be completed.
- `unsuccessful server response`: The selected server responds, but the finished HTTP result does not satisfy success and does not establish incompatible-server contract drift.
- `incompatible server contract`: The selected server responds, but the auth or activities response does not match the supported contract for this slice.

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

- The successful response must declare a JSON-compatible content type for a body that this slice expects to parse as JSON.
- The response body must parse as a JSON object.
- `authToken` must exist.
- `authToken` must be a non-empty string.

### Failure Handling Rules

- Transport timeout before a usable auth response is received is mapped to the `timeout` category.
- DNS failure, refused connection, interrupted handshake, or other inability to reach the selected server is mapped to the `connectivity problem` category.
- HTTP `403 Forbidden` is mapped to the `rejected token` category.
- Any other non-2xx response from the auth endpoint is mapped to the `unsuccessful server response` category unless the response itself proves contract incompatibility.
- Unsupported response content type, malformed JSON, missing `authToken`, or empty `authToken` is mapped to the `incompatible server contract` category.

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

- The successful response must declare a JSON-compatible content type for a body that this slice expects to parse as JSON.
- The response body must parse as a JSON object.
- `activities` must exist and be an array.
- `count` must exist and be a non-negative integer.
- For `take=1`, the returned `activities` length must be `0` or `1`.
- If `count == 0`, an empty `activities` array still counts as successful communication.
- If `count == 0`, any returned activity item makes the response invalid for this slice.
- If `count > 0`, the first returned activity must contain:
  - non-empty string `id`
  - non-empty string `date` that parses as a timestamp
  - non-empty string `type`
- If `count > 0`, at least one activity item must be returned.
- Unknown extra fields are ignored.

### Failure Handling Rules

- Transport timeout before a usable probe response is received is mapped to the `timeout` category.
- DNS failure, refused connection, interrupted handshake, or other inability to reach the selected server is mapped to the `connectivity problem` category.
- HTTP `401 Unauthorized` is mapped to the `unsuccessful server response` category.
- HTTP `403 Forbidden` is mapped to the `unsuccessful server response` category.
- HTTP `400 Bad Request` is mapped to the `incompatible server contract` category.
- Any other non-2xx response is mapped to the `unsuccessful server response` category unless the response itself proves contract incompatibility.
- Unsupported response content type, malformed JSON, missing `activities`, invalid `count`, contradictory `count` and `activities` semantics, more than one returned activity for the one-page probe, unreadable minimum timestamp values, or missing minimum activity fields is mapped to the `incompatible server contract` category.

## Success Criteria For This Slice

The application may tell the user that Ghostfolio communication is working only when all of the following are true:

- setup is complete
- auth succeeds under the contract above
- the activities probe succeeds under the contract above
- the application can produce a success result without persisting the returned payload

The success result must also tell the user that data was not stored and is not yet ready for reporting.

## User-Visible Result Requirements

- Success results must say that communication with the selected server is working.
- Success results must say that no Ghostfolio data was stored locally.
- Success results must say that reporting is still unavailable in this slice.
- Failure results must show exactly one supported failure category from this contract.
- Failure results must explain that communication validation did not succeed without exposing secrets or raw unprotected payload data.
- Both success and failure results must offer the user a path to validate again or return to the main menu.

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
- Never include the Ghostfolio token, JWT, request body, or raw unprotected payload in user-visible messages, logs, traces, crash text, or diagnostics produced by project-owned code.
- If dependency-generated or wrapped error text would otherwise surface the Ghostfolio token, JWT, request body, or raw unprotected payload, project-owned code must redact or suppress that content before display or persistence.
