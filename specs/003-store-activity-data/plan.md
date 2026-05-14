# Implementation Plan: Store Activity Data

**Branch**: `[003-store-activity-data]` | **Date**: 2026-05-14 | **Spec**: `/specs/003-store-activity-data/spec.md`
**Input**: Feature specification from `/specs/003-store-activity-data/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend the existing Go Bubble Tea sync-validation slice from `specs/002-sync-data-validation/` into a full-history secure-storage slice. The application keeps `setup.json` as the only startup-readable bootstrap state, replaces the one-page activities probe with full authenticated pagination, normalizes and validates reporting-ready `BUY` and `SELL` activity data using the technical rules established in `specs/001-ghostfolio-gains-reporting/`, and persists successful results only as token-derived encrypted local snapshots. Successful sync now ends after protected storage is updated and confirmed. Reporting, cached-data browsing, and gains-or-losses calculation remain out of scope.

## Technical Context

**Language/Version**: Go 1.26.3  
**Primary Dependencies**: `charm.land/bubbletea/v2`, selected `charm.land/bubbles/v2` components, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, Go standard library (`net/http`, `encoding/json`, `crypto/aes`, `crypto/cipher`, `crypto/rand`, `crypto/sha256`, `os`, `path/filepath`)  
**Storage**: Local-only bootstrap `setup.json` plus local-only encrypted snapshot files under the OS config or app-data directory in `ghostfolio-cryptogains/snapshots/`; Argon2id key derivation from the runtime Ghostfolio security token; AES-256-GCM protected payload; minimal authenticated cleartext header with envelope version and server discovery key; atomic rewrite with `fsync` and rename  
**Testing**: `make test` and `make coverage`; integration-first `go test` suites with `httptest.Server`, temp directories, deterministic Ghostfolio fixtures, and branch/file coverage via `github.com/Fabianexe/gocoverageplus`  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access  
**Project Type**: Single-module Go TUI application  
**Performance Goals**: Complete full-history retrieval, normalization, and protected replacement for up to 10,000 activities spanning at least 5 years in under 2 minutes; selected-server snapshot discovery and unlock responsiveness stays under 2 seconds on supported hardware; Bubble Tea busy-state updates remain responsive while sync work is in flight  
**Constraints**: Bootstrap setup remains the only state readable before token entry; Ghostfolio token and JWT are runtime-only; no raw Ghostfolio payload persistence; support only `BUY` and `SELL`; normalized `BUY` records require non-zero unit price; zero-priced `SELL` records require an explanatory comment and are stored as non-taxable holding reductions; available report years are derived from each source timestamp's own offset and calendar date; deterministic ordering is mandatory; unlock attempts are limited to selected-server snapshot candidates; failed or incompatible replacements must preserve the existing readable snapshot; no reporting, preview, or cached-data browsing is exposed in this slice  
**Scale/Scope**: One bootstrap setup profile per local OS user profile; multiple protected snapshots per machine for different valid Ghostfolio tokens; one executable business workflow (`Sync Data`) plus setup edit path; full Ghostfolio `api/v1` auth and paginated activity retrieval; up to 10,000 normalized activities per protected snapshot

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Persistence is justified because future reporting depends on local reuse of full activity history. Bootstrap setup remains local-only and plaintext-readable before token entry, while all financial and user-linked sync data moves into token-derived encrypted snapshots. The plan documents OWASP Cryptographic Storage Cheat Sheet alignment through local-only storage, Argon2id key derivation, established AEAD encryption, integrity protection for cleartext header metadata via AEAD additional authenticated data, fresh salt and nonce generation on every rewrite, atomic replacement, no stored token or token verifier, and removal guidance through deletion of snapshot files. The OWASP Top 10 review scope for this slice covers cryptographic failures, identification and authentication failures, insecure design, security misconfiguration, software and data integrity failures, and logging or diagnostic leakage.
- [x] Precision: Domain parsing and persistence use exact decimal values only. Ghostfolio numeric inputs are parsed as JSON numbers and converted immediately into canonical decimal strings and `apd.Decimal` values. No floating-point values are used in normalized activity storage or validation logic, and this slice still performs no currency conversion or report-boundary rounding.
- [x] Testing: Integration-first automated tests cover full pagination, empty-history success, first protected snapshot creation, same-token refresh, wrong-token denial, different-valid-token isolation, atomic replacement, server-mismatch confirmation, unsupported activity rejection, zero-priced `BUY` rejection, zero-priced `SELL` acceptance with explanation, deterministic ordering, duplicate removal, unsupported stored-data versions, incompatible newly synced data retention, token non-persistence, and confirmation that no reporting path is exposed. Unit tests are limited to complex isolated concerns such as envelope encoding, decimal parsing, duplicate hashing, and tie-break ordering. Statement and branch/file coverage remain explicit release gates through `make coverage`.
- [x] Dependencies: The only new runtime dependencies planned beyond the existing TUI stack are `github.com/cockroachdb/apd/v3` for exact decimals and `golang.org/x/crypto/argon2` for token-derived key derivation. `research.md` records need, maintenance, freshness, activity, and risk for those dependencies and keeps SQL, PDF, and other unnecessary libraries out of this slice.
- [x] External APIs: Ghostfolio integration remains necessary. The plan documents the observed `POST /api/v1/auth/anonymous` and paginated `GET /api/v1/activities` contract, bearer-JWT model, pagination rules, minimum required fields, contract-drift failure modes, and the rule that the client validates compatibility at runtime instead of assuming a permanently stable upstream API.
- [x] Architecture: The design keeps bootstrap configuration, Ghostfolio transport DTOs, sync normalization and validation, protected snapshot storage, and TUI workflow state in separate packages so business rules stay testable without terminal or filesystem coupling.

## Project Structure

### Documentation (this feature)

```text
specs/003-store-activity-data/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── ghostfolio-sync.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── ghostfolio-cryptogains/
    └── main.go

internal/
├── app/
│   ├── bootstrap/
│   └── runtime/
├── config/
│   ├── model/
│   └── store/
├── ghostfolio/
│   ├── client/
│   ├── dto/
│   ├── mapper/
│   └── validator/
├── snapshot/
│   ├── envelope/
│   ├── model/
│   └── store/
├── sync/
│   ├── model/
│   ├── normalize/
│   └── validate/
├── tui/
│   ├── component/
│   ├── flow/
│   └── screen/
└── support/
    ├── decimal/
    └── redact/

tests/
├── contract/
├── integration/
└── unit/
```

**Structure Decision**: Keep the repository as one Go module rooted at the project root. `internal/config` remains bootstrap-only from `002`, `internal/ghostfolio` grows from the validation probe into the full transport and mapping boundary, `internal/sync` owns normalized activity rules derived from `001`, `internal/snapshot` owns token-derived protected storage, and `internal/tui` continues to own full-screen workflow rendering without gaining direct responsibility for crypto or normalization logic.

## Bootstrap And Protected Storage Rules

- Keep `setup.json` as the only startup-readable file. Its schema remains limited to bootstrap setup fields from `002` and must not absorb synced activity data, available years, user-linked metadata, or version markers created by this slice.
- Add a sibling protected-snapshot directory under the same application data root: `ghostfolio-cryptogains/snapshots/`.
- Store each isolated protected snapshot in one opaque file named by a random identifier rather than by server origin, token, or user-readable profile name.
- The on-disk cleartext header contains only the fields needed before decrypt: envelope magic, `format_version`, KDF parameters, random salt, AEAD nonce, and a `server_discovery_key` derived from the canonical selected server origin.
- Define `server_discovery_key = SHA-256(canonical_server_origin)` so snapshot discovery can stay server-scoped without writing the plaintext origin or any token-derived verifier into the snapshot header.
- Authenticate the serialized cleartext header as AEAD additional authenticated data so header tampering fails decryption even though the header itself remains readable.
- Keep a versioned encrypted payload that contains stored-data version markers, protected setup profile, registered local user metadata, normalized activity cache, available report years, and sync metadata.
- Write every successful protected update through a temp file, `fsync`, and atomic rename. A failed write, validation failure, or canceled replacement must leave the previously readable snapshot untouched.
- Document removal paths clearly: deleting `setup.json` resets bootstrap setup, while deleting protected snapshot files removes token-locked synced data. Neither operation persists or reveals the Ghostfolio security token.

## Full-History Sync Rules

- Reuse the existing `POST /api/v1/auth/anonymous` boundary from `002`, but replace the validation-only `take=1` probe with full authenticated pagination against `GET /api/v1/activities`.
- Continue paging with `skip`, `take`, `sortColumn=date`, and `sortDirection=asc` until the number of retrieved records is greater than or equal to the reported `count`. Treat non-monotonic or contradictory pagination as sync failure.
- Replace `ActivitiesProbeResponse` and `ActivityProbeEntry` with full DTOs and mappers that preserve the fields needed by `001`: source identifier, timestamp, activity type, asset identity, quantity, unit price, gross value, fee amount, explanatory comment, and available source-scope data.
- Parse numeric transport fields through `json.Decoder.UseNumber` and convert them immediately into canonical decimal strings backed by `apd.Decimal` values. Raw `float64` values are not allowed in the normalized model.
- Normalize the complete retrieved history before any persistence step:
  - map DTOs into stored `ActivityRecord` values
  - canonicalize exact-decimal fields
  - sort chronologically
  - remove exact duplicates by a hash of normalized source fields
  - validate supported activity rules and defensibility
  - derive `available_report_years` from each timestamp's own offset and calendar date
- Use deterministic ordering for same-asset events that share the same instant: `occurred_at` ascending, then `source_id` ascending. Reject the sync if stable ordering cannot be established.
- Support only `BUY` and `SELL`. Any other activity type fails the sync. Normalized `BUY` records must have `unit_price > 0`. Normalized `SELL` records may have `unit_price = 0` only when an explanatory comment is present, in which case they are stored as non-taxable holding reductions for future reporting.
- Preserve source holding scope when present, but record scope reliability rather than making any reporting decision in this slice. Missing or unreliable scope data does not fail sync by itself.
- Treat a valid empty history as a successful sync that still creates or refreshes the protected snapshot state for the selected server and token.
- Do not calculate gains or losses, select cost basis methods, expose report years to the UI, or show any cached-activity browsing in this slice.

## Compatibility And Replacement Rules

- Treat snapshot compatibility as two layers:
  - cleartext `format_version` for envelope and header compatibility checks before decrypt
  - protected stored-data version markers for payload-schema and normalized-activity-model compatibility after decrypt
- Unsupported `format_version` fails immediately with a compatibility error and no unlock attempt.
- Unsupported payload stored-data version fails after decrypt with a compatibility error. The application must not partially load, auto-migrate, or overwrite that snapshot.
- When attempting to unlock existing protected data, enumerate snapshot headers but attempt the user-supplied token only against headers whose `server_discovery_key` matches the currently selected bootstrap server.
- When a readable protected snapshot is already active in memory for the current run and the bootstrap server is changed, compare the active snapshot's protected `server_origin` with the bootstrap `server_origin` before the next sync. If they differ, show the required replacement confirmation before starting retrieval.
- Declining server replacement leaves the active snapshot unchanged and aborts the new sync before Ghostfolio retrieval begins.
- Confirmed server replacement still preserves the old protected snapshot until the new sync has fully authenticated, retrieved, normalized, validated, and been written atomically.
- If a supplied token does not unlock any selected-server snapshot candidate but does authenticate successfully with Ghostfolio, treat that token as a new isolated local-user context and create a new protected snapshot only after the full sync succeeds.
- If a supplied token does not unlock any selected-server snapshot candidate and Ghostfolio rejects the token, leave all local protected data unchanged.
- If newly retrieved data cannot be normalized or persisted within the current supported stored-data model, discard the new in-memory data, keep any existing readable snapshot active and unchanged, and show a user-visible incompatibility result instead of overwriting the old snapshot.

## Complexity Tracking

No constitution violations require justification for this plan.
