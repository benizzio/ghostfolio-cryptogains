# Research: Store Activity Data

## Go Toolchain And Existing TUI Continuity

Decision: Keep the repository's single-module Go 1.26.3 application structure and the existing Bubble Tea and Bubbles stack established in `specs/002-sync-data-validation/`. This slice adds new sync and snapshot packages without changing the application's full-screen TUI foundation.

Rationale: The current repository already ships the `002` validation slice on Go 1.26.3 with `bubbletea` and selected `bubbles` components. `specs/001-ghostfolio-gains-reporting/research.md` already recorded the broader due diligence for this stack, and nothing in `003` requires a different presentation framework. Reusing the current Bubble Tea flow preserves existing keyboard behavior, test seams, and startup rules while keeping the new work focused on sync, normalization, and protected storage.

Alternatives considered: Replacing the TUI stack with `tview` or another framework was rejected because it would create unrelated migration work and duplicate a decision already validated in `001` and implemented in `002`. Introducing a separate unlock-at-start application mode was rejected because this slice keeps the startup-readable bootstrap pattern from `002` and does not need a new launch surface.

## Exact Decimal Parsing And Persistence

Decision: Introduce `github.com/cockroachdb/apd/v3` for all normalized quantity, unit price, gross value, and fee parsing and validation. Decode Ghostfolio JSON with `encoding/json.Decoder.UseNumber`, convert immediately into canonical decimal strings backed by `apd.Decimal`, and persist canonical decimal strings inside the protected snapshot.

Rationale: The constitution forbids floating-point domain logic, and `FIN-001` in `specs/003-store-activity-data/spec.md` requires preserving exact source precision with no rounding or conversion in this slice. The `001` research already selected `apd/v3` as the exact-decimal library for financial inputs. This slice only needs the storage and validation part of that decision, not the later report-boundary rounding behavior.

Alternatives considered: `float64` was rejected by the constitution and by the spec's exact-precision requirement. `shopspring/decimal` was rejected because `apd/v3` was already chosen in `001` and provides the stricter error-aware arithmetic context preferred for later reporting work. `math/big.Rat` was rejected because it makes canonical decimal persistence and future human-readable output rules more awkward without adding value in this slice.

## Protected Snapshot Storage Layout

Decision: Keep `setup.json` as the only plaintext bootstrap file and add a separate `snapshots/` directory containing one encrypted snapshot file per isolated local protected context. Use Argon2id for token-derived key derivation, AES-256-GCM for payload encryption, and a minimal authenticated cleartext header containing envelope metadata plus a server discovery key derived from the canonical selected server origin.

Rationale: `specs/003-store-activity-data/spec.md` requires bootstrap state to stay readable before token entry while all synced activity data and user-specific sync state become token-locked. A per-snapshot encrypted blob matches the `001` storage direction, keeps the Ghostfolio token out of persistent storage, simplifies atomic replacement, and supports multiple isolated snapshots on the same machine. A small cleartext header is still necessary to reject unsupported envelope versions before decrypt and to limit unlock attempts to snapshots associated with the selected Ghostfolio server.

Selected storage details:

- bootstrap file path stays `ghostfolio-cryptogains/setup.json`
- protected snapshot directory becomes `ghostfolio-cryptogains/snapshots/`
- snapshot filename is an opaque random identifier
- cleartext header fields: magic, `format_version`, `server_discovery_key`, KDF parameters, salt, nonce
- `server_discovery_key = SHA-256(canonical_server_origin)`
- cleartext header bytes are authenticated as AEAD additional authenticated data
- encrypted payload fields: stored-data version markers, protected setup profile, registered local user metadata, normalized activity cache, available report years, and sync metadata
- every rewrite uses fresh salt and nonce and replaces the snapshot atomically through temp file plus rename

Alternatives considered: SQLite or another embedded database was rejected because this slice persists one cohesive protected snapshot per local context and does not need queryable tables. SQLCipher was rejected because it adds distribution complexity and CGO concerns that are unnecessary for a Go TUI. A plaintext profile index was rejected because the spec and constitution allow only minimal cleartext metadata outside the encrypted payload. Storing a token hash or reusable token verifier was rejected explicitly by `SEC-002`.

## Ghostfolio Full-History Sync Contract

Decision: Expand the current Ghostfolio boundary from a one-page communication probe into a full authenticated retrieval flow using `POST /api/v1/auth/anonymous` followed by paginated `GET /api/v1/activities` requests until the full available history has been collected.

Rationale: `specs/003-store-activity-data/spec.md` turns `Sync Data` into full-history retrieval and protected persistence. `specs/001-ghostfolio-gains-reporting/contracts/ghostfolio-sync.md` already documents the observed `api/v1` auth and activities contract, including pagination rules and minimum required fields. The current `002` client and runtime service were intentionally transitional, and the `002` plan explicitly says their probe DTOs must be removed or evolved when real sync is introduced.

Selected retrieval rules:

- keep the existing anonymous-auth request model and runtime-only JWT handling
- request activities in ascending date order using `skip`, `take`, `sortColumn=date`, and `sortDirection=asc`
- continue paging until retrieved record count is greater than or equal to the reported `count`
- treat inconsistent pagination, malformed JSON, unsupported content types, or missing required activity fields as sync failure
- treat a valid empty history as successful retrieval

Alternatives considered: Keeping the `take=1` activities probe from `002` was rejected because it cannot prove full-history correctness. Using only a health probe or auth-only success was rejected because this slice depends on the actual activity-history contract. Persisting raw Ghostfolio payloads and postponing normalization was rejected because the spec requires storage to stop only after normalization and validation succeed.

## Normalization, Deduplication, And Validation Pipeline

Decision: Convert transport DTOs into normalized stored `ActivityRecord` values, sort the full history chronologically, remove exact duplicates after canonical normalization, enforce the `BUY` and `SELL` support boundary from `001`, preserve source timestamps with their original offsets, and derive available report years from those source timestamps before persistence.

Rationale: `003` explicitly inherits the reporting-ready activity-model subset from `001` while still forbidding reporting behavior in this slice. The stored data needs to be defensible for later basis calculations, so the sync pipeline must normalize and validate now rather than postpone correctness checks. The clarification in `specs/003-store-activity-data/spec.md` requires report-year derivation from each timestamp's own offset and calendar date, which means the stored model must not discard source offset information during normalization.

Selected normalization rules:

- map each supported Ghostfolio activity into a normalized `ActivityRecord`
- preserve the timestamp in RFC3339 form with its source offset intact
- establish deterministic order as `occurred_at` ascending and `source_id` ascending for same-asset same-instant ties
- compute a `raw_hash` from normalized source fields and remove exact duplicates by that hash
- reject the full sync if a supported deterministic order cannot be established
- reject any activity type other than `BUY` or `SELL`
- reject normalized `BUY` records with `unit_price = 0`
- accept normalized `SELL` records with `unit_price = 0` only when an explanatory comment is present, storing them as non-taxable holding reductions for future reporting
- preserve available source-scope data and record whether scope is reliable enough for later reporting decisions
- derive `available_report_years` from the stored source timestamps after normalization completes

Alternatives considered: Skipping unsupported activities was rejected because both `001` and `003` require the whole sync to fail on unsupported source activity types. Using machine-local time or forced UTC conversion for year derivation was rejected because it would violate the source-offset rule in `003`. Inferring transfer linkage from free-text comments was rejected because `001` already limits comments to explanatory text only.

## Stored-Data Versioning And Safe Replacement

Decision: Use two compatibility layers: cleartext `format_version` for the snapshot envelope and protected stored-data version markers for the payload schema and normalized activity model. Restrict unlock attempts to selected-server snapshot candidates, drive server-mismatch confirmation from the active readable snapshot for the current run, and discard newly incompatible sync results while preserving any existing readable snapshot.

Rationale: `003` adds explicit compatibility requirements that did not exist in `002` and tightens the earlier `001` storage design. The application must fail safely when it cannot read a snapshot written by another version, must not overwrite unreadable snapshots automatically, and must not brute-force unlock snapshots from other servers. At the same time, confirmed server replacement and incompatible new-data handling require a notion of an active readable snapshot in memory during the current run.

Selected compatibility rules:

- unsupported `format_version` fails before decrypt with a compatibility error
- supported envelope but unsupported payload stored-data version fails after decrypt with a compatibility error and no partial load
- snapshot header discovery is server-scoped by `server_discovery_key` before any token attempt
- an already readable snapshot in memory is treated as the active local context for mismatch warning and incompatible-data retention
- if no selected-server snapshot unlocks but Ghostfolio auth succeeds, a different valid token creates a new isolated snapshot only after full sync success
- if newly retrieved data cannot be normalized or persisted within the current supported stored-data model, the new data is discarded and any existing readable snapshot remains active and unchanged

Alternatives considered: A single version marker was rejected because it cannot distinguish envelope incompatibility from payload-model incompatibility cleanly. Trying the supplied token against every snapshot file was rejected by `FR-036`. Auto-migrating or auto-overwriting unsupported snapshots was rejected by `FR-035` and the constitution's fail-safe posture.

## Testing And Coverage Gate

Decision: Keep the integration-first testing approach from `002`, extend it to cover full-history pagination, protected snapshot discovery and replacement, compatibility errors, and multi-token isolation, and continue to use `make test` and `make coverage` as the contributor entry points.

Rationale: The slice's main risks are workflow correctness, contract drift, and replacement safety. These are best verified through end-to-end application and boundary tests using `httptest.Server`, temp directories, and deterministic fixture histories. The repository already exposes maintained `Makefile` targets and already uses `gocoverageplus` for branch and file coverage, so the smallest correct plan is to extend that verification path rather than invent a new one.

Coverage focus for this slice:

- bootstrap setup remains readable before token entry
- full pagination until `count` is satisfied
- empty history success
- first protected snapshot creation after full success only
- same-token refresh replacing only after full success
- different-valid-token creation of a separate isolated snapshot
- invalid token leaving local data unchanged
- wrong-token denial for an existing snapshot
- duplicate removal and deterministic ordering
- unsupported activity rejection
- zero-priced `BUY` rejection and zero-priced `SELL` acceptance with explanation
- server-mismatch confirmation and cancel path
- unsupported stored-data version compatibility failure
- incompatible newly synced data discarded while existing readable snapshot stays active
- no report generation or cached-data browsing exposed in the UI

Alternatives considered: Statement-only coverage was rejected because the constitution requires branch or file coverage where tooling can distinguish it. Unit-only testing was rejected because it would miss the end-to-end protection and workflow guarantees that dominate this slice.

## Dependency Due Diligence Summary

| Dependency | Purpose In This Slice | Evidence Source | Acceptance And Risk Summary |
|------------|-----------------------|-----------------|-----------------------------|
| `bubbletea` | Existing full-screen TUI runtime | `specs/001-ghostfolio-gains-reporting/research.md`, implemented in `002` | Already adopted in-repo, active, and still the smallest correct UI foundation |
| `bubbles` | Existing focused inputs, menus, help, spinner widgets | `specs/001-ghostfolio-gains-reporting/research.md`, implemented in `002` | Already adopted in-repo; use remains selective |
| `github.com/cockroachdb/apd/v3` | Exact decimal parsing and normalized storage values | `specs/001-ghostfolio-gains-reporting/research.md` | Mature and well-suited for financial precision; new runtime dependency is justified by constitution requirements |
| `golang.org/x/crypto/argon2` | Token-derived key derivation for protected snapshots | `specs/001-ghostfolio-gains-reporting/research.md` | Official Go-maintained crypto extension; justified because stdlib lacks Argon2id |
| `github.com/Fabianexe/gocoverageplus` | Existing branch/file coverage gate | `specs/001-ghostfolio-gains-reporting/research.md`, implemented in `002` | Development-only helper already in use; acceptable while monitored |

Dependencies intentionally deferred in this slice are PDF libraries and any database library. Reporting remains out of scope, and one encrypted snapshot file per local context is simpler than introducing a queryable store.
