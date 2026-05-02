# Contract: Ghostfolio Sync Boundary

## Scope

This contract defines the minimum external HTTP behavior the Go TUI relies on when synchronizing activity history from Ghostfolio. It is intentionally conservative and documents only the baseline surface required for this feature.

Observed upstream reference:

- Ghostfolio release `3.1.0` dated 2026-04-29.
- Upstream `main` branch source observed on 2026-05-01.

## Compatibility Rules

- The configured Ghostfolio base origin defaults to `https://ghostfol.io`, is stored in encrypted local setup data, and may be replaced by the user with a self-hosted origin.
- The client targets the observed `api/v1` base path.
- The client must validate connectivity and endpoint compatibility at runtime instead of assuming a permanent public contract.
- HTTPS is required for all production-like origins. Only explicitly permitted local-development origins may use HTTP, and production-like HTTP attempts must fail with a blocking error.

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

### Success Response

HTTP `200 OK`

```json
{
  "authToken": "<jwt>"
}
```

### Failure Response

- HTTP `403 Forbidden` for an invalid or rejected security token.
- Other non-2xx responses are treated as sync failures.

### Client Rules

- Each sync session is initiated for one selected registered local user context.
- The Ghostfolio security token is entered by the user for each sync session.
- The security token is used only for this request and is never persisted.
- The returned JWT is stored only in memory for the active sync workflow.
- The client must not log request bodies or authorization headers.

## Optional Health Probe Contract

### Request

`GET /api/v1/health`

### Client Rules

- The health endpoint may be used for a fast connectivity check before prompting for a token or before a longer sync operation.
- Failure of the health probe is treated as a connectivity problem, not as proof that the server is incompatible.

## Activities Retrieval Contract

### Request

`GET /api/v1/activities`

Headers:

- `Authorization: Bearer <jwt>`

Query parameters used by the client:

- `skip`
- `take`
- `sortColumn=date`
- `sortDirection=asc`

The client may add filtering parameters only if they preserve full-history correctness.

### Success Response

HTTP `200 OK`

```json
{
  "activities": [
    {
      "id": "activity-id",
      "date": "2026-01-31T10:00:00.000Z",
      "type": "BUY",
      "quantity": 1.25,
      "value": 62500,
      "valueInBaseCurrency": 62500,
      "feeInBaseCurrency": 25,
      "unitPriceInAssetProfileCurrency": 50000,
      "comment": "optional",
      "SymbolProfile": {
        "symbol": "BTC"
      },
      "account": {
        "id": "optional-account-id"
      }
    }
  ],
  "count": 1
}
```

The full upstream activity schema is larger than this example. The baseline client depends only on the minimum fields required for defensible holdings and basis reconstruction.

### Minimum Required Fields Per Activity

The client must reject sync when any activity required for holdings reconstruction is missing these normalized inputs:

- source identifier
- timestamp
- activity type
- asset symbol or equivalent asset identity
- quantity
- gross value or unit price information sufficient to derive basis and proceeds
- fee information when present in source data
- explanatory comment when zero-priced movement semantics depend on it

Optional fields used when available:

- account scope data used as wallet-equivalent input for wallet-scoped matching
- asset name and display metadata
- opaque data-source metadata

### Pagination Rules

- The client must continue paging until the number of retrieved activities is greater than or equal to `count`.
- A partial first page must never be treated as a complete history.
- If pagination becomes inconsistent or non-monotonic, the client must fail the sync.

### Failure Responses And Handling

- HTTP `401 Unauthorized`: treat as invalid or expired session JWT, end the sync, and clear in-memory credentials.
- HTTP `403 Forbidden`: treat as permission or feature-gating failure and end the sync without persisting any new profile data.
- HTTP `400 Bad Request`: treat as client/server contract mismatch and surface a non-secret actionable error.
- Malformed JSON, redacted numeric values, unsupported holding-affecting event types, or missing required fields: reject the entire sync.

## Normalization And Validation Rules

- Sort the complete retrieved activity history chronologically before persistence.
- Remove exact duplicates after canonical normalization.
- Reject the sync if unsupported event types that affect holdings are present.
- Reject the sync if remaining gaps or contradictions make basis calculation non-defensible.
- If account-derived wallet scope is missing or unreliable, downgrade wallet-scoped unit matching to asset-level FIFO exactly as defined in the feature specification.

## Security Rules

- Never send the Ghostfolio security token to any origin other than the currently selected and canonicalized Ghostfolio origin.
- Persist only the encrypted local snapshot produced by the application itself.
- Do not persist the Ghostfolio JWT or any server-provided bearer credential.
- Do not include secrets or raw unprotected activity payloads in logs or persisted error messages.
