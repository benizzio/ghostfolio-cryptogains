# Feature Specification: Store Activity Data

**Feature Branch**: `[003-store-activity-data]`  
**Created**: 2026-05-12  
**Status**: Draft  
**Input**: User description: "We will now specify the second slice of the work derived from a split of @specs/001-ghostfolio-gains-reporting/ (should still be the main reference) that will complement and integrate to the work already implemented on @specs/002-sync-data-validation/. In this next slice we will work on obtaining all the activity data returned from the sync with the Ghostfolio server, store it with all the security requirements so it is only accessible using the Ghostfolio Security token provided by the user. This stored activity data must follow the appropriate model already established in the original spec 001 and relevant to the future reporting, but this spec's slice MUST NOT DO ANY REPORTING, just fetch and store data."

## Plain-Language Summary

- This slice turns `Sync Data` from a communication check into a full-history sync that retrieves and stores reporting-ready Ghostfolio activity data.
- The startup-readable bootstrap setup from the previous slice remains in place, but the stored activity data and user-specific sync state move into separate token-locked protected local storage.
- This slice stops after secure retrieval, normalization, validation, and protected persistence. It does not calculate gains or losses, preview activity data, or generate any report.

## Terms Used In This Spec

- **Bootstrap setup**: The startup-readable machine-local setup persisted by the previous slice. It remembers setup completion and the selected Ghostfolio server before the user enters a Ghostfolio security token.
- **Protected snapshot**: The token-locked local container that stores the registered local user state and the validated activity history.
- **Registered local user**: The local protected profile created only after a full authenticated sync succeeds.
- **Normalized activity history**: Activity data that has been ordered chronologically, stripped of exact duplicates, and validated for future reporting use.
- **Defensible history**: A normalized activity history that still supports reproducible future basis calculations and year-based reporting.
- **Server mismatch**: A later sync attempt where the currently selected Ghostfolio server does not match the server reference stored with the protected snapshot.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Sync And Store Full Activity History (Priority: P1)

After setup is complete, the user can run `Sync Data`, provide a Ghostfolio security token, retrieve the full supported activity history from the selected Ghostfolio server, and store it locally in protected form for future reporting work.

**Why this priority**: Future reporting depends on having the complete reporting-ready activity history available locally, and this slice exists to create that foundation securely.

**Independent Test**: With a completed setup and a valid Ghostfolio security token, the user can start `Sync Data`, retrieve a full multi-page activity history, and finish with a confirmation that protected local data was stored for future use without any reporting behavior being offered.

**Acceptance Scenarios**:

1. **Given** setup is complete and the selected Ghostfolio server is reachable, **When** the user starts `Sync Data` and provides a valid Ghostfolio security token, **Then** the system retrieves the full supported activity history from that server, validates it, stores it in protected local storage, and confirms that the data is ready for future reporting use without generating any report.
2. **Given** the selected Ghostfolio server returns activity history across multiple pages or batches, **When** the sync runs, **Then** the system continues retrieving until the full available supported history has been gathered before deciding whether to persist it.
3. **Given** the selected Ghostfolio server returns a valid empty history, **When** the sync completes successfully, **Then** the system creates or refreshes the protected local state for that server and token, records that the sync succeeded with no stored activity entries, and does not offer reporting.
4. **Given** no protected local user exists yet, **When** the first full sync succeeds, **Then** the system creates the registered local user and associated protected snapshot only after retrieval, normalization, validation, and protected persistence all succeed.
5. **Given** the user reaches the sync result screen after a successful full sync, **When** the outcome is shown, **Then** the message confirms that activity data was stored for future use and explicitly states that report generation is not part of this slice.

---

### User Story 2 - Reuse Token-Locked Stored Data (Priority: P1)

After a successful sync, the user can later refresh the stored activity history, but the application can read or update that protected data only when the same Ghostfolio security token is supplied again.

**Why this priority**: The user asked for stored activity data that remains accessible only through the Ghostfolio security token, so protected reuse is part of the core scope rather than a later enhancement.

**Independent Test**: After one successful sync, restart the application and run `Sync Data` again. With the correct token, the protected snapshot can be refreshed. With an incorrect or unavailable token, the stored activity data remains inaccessible and unchanged.

**Acceptance Scenarios**:

1. **Given** protected local data already exists for a previous successful sync, **When** the user later runs `Sync Data` again and provides the same Ghostfolio security token, **Then** the system can unlock the existing protected snapshot, refresh it from the selected server, and replace it only if the new sync fully succeeds.
2. **Given** protected local data already exists, **When** the user provides a Ghostfolio security token that cannot unlock that protected snapshot, **Then** the system does not reveal the stored activity data and leaves the protected snapshot unchanged.
3. **Given** protected local data already exists, **When** a refresh attempt fails after retrieval starts but before a new protected snapshot is written successfully, **Then** the previously stored protected data remains unchanged.
4. **Given** the application starts after a successful earlier sync, **When** startup completes before the user enters a Ghostfolio security token, **Then** only bootstrap setup is readable and the protected activity data remains inaccessible until the correct token is supplied in the sync workflow.

---

### User Story 3 - Preserve Data Quality And Server Boundaries (Priority: P1)

The user can trust that only future-reporting-ready activity histories are stored, and that invalid data or a server change cannot silently contaminate or replace the protected local snapshot.

**Why this priority**: Protected storage is not useful if unsupported, inconsistent, or wrong-server data can be stored and later mistaken for valid reporting input.

**Independent Test**: Run sync attempts with unsupported activity types, zero-priced `BUY` records, non-defensible normalized histories, and server-change scenarios. The system either stores only normalized valid history or keeps the previous protected snapshot unchanged.

**Acceptance Scenarios**:

1. **Given** the retrieved history contains any activity record whose type is not `BUY` or `SELL`, **When** validation runs, **Then** the system refuses the sync, does not create a new protected snapshot for that attempt, and does not replace any existing protected snapshot.
2. **Given** the retrieved history arrives out of order or contains exact duplicate records, **When** validation runs, **Then** the system normalizes the history into chronological order and removes exact duplicates before deciding whether to persist it.
3. **Given** the normalized history still contains gaps or inconsistencies that prevent defensible future reporting, **When** the sync ends, **Then** the system refuses the sync and leaves any existing protected snapshot unchanged.
4. **Given** the retrieved history contains a `BUY` record whose normalized unit price is `0`, **When** validation runs, **Then** the system refuses the sync and stores no new protected data for that attempt.
5. **Given** the retrieved history contains a `SELL` record whose normalized unit price is `0` and includes an explanatory comment, **When** validation runs, **Then** the system stores that record as a non-taxable holding reduction for future reporting use without calculating or displaying any gain or loss in this slice.
6. **Given** protected local data already exists for one selected Ghostfolio server, **When** the user changes the selected Ghostfolio server and starts a new sync, **Then** the system warns that continuing will replace the current protected data tied to that token and server.
7. **Given** a server-mismatch warning is shown, **When** the user declines replacement, **Then** the existing protected snapshot remains unchanged and the new sync does not start.
8. **Given** a server-mismatch warning is shown and the user confirms replacement, **When** the replacement sync later fails or is abandoned, **Then** the existing protected snapshot remains unchanged.

---

### Edge Cases

- The user loses the Ghostfolio security token after a successful sync and later wants to reopen the protected activity history.
- The selected Ghostfolio server returns an empty but otherwise valid activity history.
- The full activity history spans multiple pages or batches and a later retrieval step fails after earlier pages were already received.
- The retrieved history contains exact duplicates, out-of-order records, unreadable timestamps, or multiple same-asset events that still cannot be ordered deterministically after normalization.
- The retrieved history contains any activity type other than `BUY` or `SELL`.
- The retrieved history contains a `BUY` with unit price `0` or a `SELL` with unit price `0` but no explanatory comment.
- The selected Ghostfolio server changes after protected local data already exists.
- The user supplies a Ghostfolio security token that cannot unlock the existing protected snapshot.
- The application exits or fails while a new protected snapshot is being written; the previous protected snapshot must remain intact and no partially readable replacement may remain.
- The source provides wallet or account scope for some activities but not enough to treat that scope as reliable for all later reporting decisions.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, precision,
testing, dependency, and external integration impacts when the feature touches
those areas.

### Functional Requirements

- **FR-001**: The system MUST keep the startup-readable bootstrap setup from the previous slice as the only local state available before Ghostfolio security-token entry, and MUST require completed setup before a full sync can start.
- **FR-002**: The system MUST allow the user to start a full `Sync Data` workflow after setup is complete.
- **FR-003**: The system MUST require the user to provide a Ghostfolio security token for each sync attempt and MUST use that token both to authenticate the sync attempt and to unlock or protect the stored user-specific sync data created by this slice.
- **FR-004**: The system MUST retrieve the full available supported Ghostfolio activity history needed for future reporting from the selected Ghostfolio server before deciding whether a sync attempt succeeds.
- **FR-005**: The system MUST continue retrieval until the full available supported activity history has been gathered, even when that history is returned in multiple pages or batches.
- **FR-006**: The system MUST create or retain a registered local user only after successful Ghostfolio access, full activity retrieval, normalization, validation, and protected persistence.
- **FR-007**: The system MUST persist successful sync results only in token-locked protected local storage, and the stored activity data MUST not be readable without the same Ghostfolio security token.
- **FR-008**: The system MUST keep Ghostfolio-returned activity data, registered-local-user metadata created by this slice, stored server reference, available report years, and sync metadata inside the protected snapshot rather than in the startup-readable bootstrap setup.
- **FR-009**: The system MUST store with each protected snapshot the Ghostfolio server reference used for that snapshot's current protected data and MUST compare it on later sync attempts to detect server mismatch after setup changes.
- **FR-010**: The system MUST show an explicit confirmation when the selected Ghostfolio server does not match the stored server reference for the protected snapshot, and that confirmation MUST state that continuing will replace the current protected data tied to that token and server.
- **FR-011**: The system MUST replace an existing protected snapshot only after the user confirms a server mismatch and the replacement sync completes successfully.
- **FR-012**: The system MUST leave existing protected data unchanged when a replacement sync is declined, fails, or is abandoned before the new protected snapshot is written successfully.
- **FR-013**: The system MUST normalize retrieved activity history into chronological order and MUST remove exact duplicate records before deciding whether the history can be persisted.
- **FR-014**: The system MUST support only Ghostfolio `BUY` and `SELL` activity records in the stored activity history.
- **FR-015**: The system MUST refuse a sync attempt when any normalized activity record is not `BUY` or `SELL`, and MUST not create or update protected local data for that failed attempt.
- **FR-016**: The system MUST require every normalized `BUY` activity record to have a non-zero unit price and MUST reject histories that contain a normalized `BUY` record with unit price `0`.
- **FR-017**: The system MUST treat a normalized `SELL` activity record with unit price `0` and an explanatory comment as a non-taxable holding reduction to be preserved in stored history for future reporting use.
- **FR-018**: The system MUST preserve, in each stored activity record, the activity data needed for future reporting, including asset identity, event time, quantity, unit price, value, fees, explanatory comments when present, and any available source holding-scope data.
- **FR-019**: The system MUST preserve or derive the set of years present in the stored activity history so future reporting slices can limit year selection to years that actually exist in the cached data.
- **FR-020**: The system MUST evaluate normalized histories for gaps or inconsistencies that would prevent defensible future basis calculations and MUST reject such histories before persistence.
- **FR-021**: The system MUST establish a deterministic order for same-asset activities that share the same timestamp when the source history provides enough stable ordering information, and MUST reject the sync if deterministic ordering cannot be established.
- **FR-022**: The system MUST record whether source holding-scope data is reliable enough for future scope-local reporting decisions or whether future reporting will need to broaden those activities to asset-level scope.
- **FR-023**: The system MUST write successful protected sync results atomically as a complete protected-snapshot replacement rather than as partial record updates.
- **FR-024**: The system MUST allow the user to run sync again after both successful and failed attempts without requiring setup to be repeated.
- **FR-025**: The system MUST show user-facing sync outcomes that either confirm successful protected storage or explain the failure and next step without exposing the Ghostfolio security token or unprotected activity data.
- **FR-026**: The system MUST treat a successful sync with an empty activity history as a valid protected local state for that selected server and token.
- **FR-027**: The system MUST refuse access to an existing protected snapshot when the supplied Ghostfolio security token cannot unlock it, and MUST leave the stored data unchanged.
- **FR-028**: The system MUST not expose report generation, report preview, gains-or-losses calculation, or direct cached-activity browsing in this slice.
- **FR-029**: The system MUST not persist transient failure messages, raw unprotected Ghostfolio payloads, or recoverable Ghostfolio security-token traces for later display, diagnostics, or storage.

### Acceptance Coverage By Requirement

- `FR-001` to `FR-005`, `FR-024`, `FR-025`, and `FR-026` are accepted by User Story 1 scenarios 1 through 5 together with the empty-history edge case.
- `FR-003`, `FR-006` to `FR-012`, `FR-023`, `FR-024`, `FR-025`, `FR-027`, and `FR-029` are accepted by User Story 2 scenarios 1 through 4 and User Story 3 scenarios 6 through 8.
- `FR-013` to `FR-022`, `FR-025`, and `FR-029` are accepted by User Story 3 scenarios 1 through 8 together with the invalid-history and deterministic-ordering edge cases.
- `FR-028` is accepted by User Story 1 scenario 5 and by explicit scope review of all workflow outcomes in this slice.

### Security, Precision, and Integration Constraints

- **SEC-001**: The Ghostfolio security token MUST be entered explicitly by the user for each sync attempt, kept only for the active session, used as the unlock basis for protected sync data, cleared when the attempt ends or the application exits, and excluded from logs, output, and persisted artifacts.
- **SEC-002**: Persisted activity data and user-specific sync state created by this slice MUST remain local to the user's computer and MUST be protected with token-derived encryption aligned with the OWASP Cryptographic Storage Cheat Sheet, including established cryptography, integrity protection, fresh randomness on rewrite, minimal cleartext metadata, and no stored token, token hash, or reusable token verifier.
- **SEC-003**: The startup-readable bootstrap setup carried forward from the previous slice MUST remain proportionately machine-local protected and MUST never include activity history, available report years, registered-local-user identity data created by this slice, or anything that would weaken the rule that stored activity data is accessible only through the Ghostfolio security token.
- **FIN-001**: Every stored quantity, unit price, gross value, and fee value MUST preserve exact source precision in storage without rounding or currency conversion in this slice. Zero-priced `BUY` records are rejected, and zero-priced `SELL` records are stored only as non-taxable holding reductions when accompanied by an explanatory comment.
- **QUAL-001**: Automated validation MUST cover full-history retrieval across multiple pages or batches, successful empty-history sync, first successful protected-profile creation, token-required unlock of existing protected data, wrong-token denial, retention of the previous protected snapshot after failed refresh, server-mismatch confirmation and replacement behavior, unsupported activity-type rejection, zero-priced `BUY` rejection, zero-priced `SELL` acceptance with explanation, chronological normalization, exact-duplicate removal, deterministic same-timestamp ordering or rejection, rejection of non-defensible normalized histories, scope-reliability preservation, available-year derivation, atomic protected-snapshot replacement, and confirmation that no reporting workflow is exposed.
- **INT-001**: The feature depends on the selected Ghostfolio server exposing authenticated full activity history with enough asset identity, timestamps, quantities, prices, values, fees, explanatory comments, and any available source holding-scope information to build a normalized future-reporting-ready local history. Empty history is still a compatible success case. Contract drift or incomplete source data that prevents defensible future reporting is treated as sync failure rather than as partially stored success.

### Key Entities *(include if feature involves data)*

- **Bootstrap Setup Record**: The startup-readable machine-local setup carried forward from the previous slice. It remembers setup completion and the selected Ghostfolio server, but never stores synced activity data or unlockable protected user data.
- **Encrypted Snapshot Envelope**: The opaque local container that stores all persisted sync data created by this slice for one registered local user while exposing only minimal non-secret metadata needed to attempt unlock.
- **Snapshot Payload**: The decrypted protected state for one registered local user after the correct Ghostfolio security token is supplied. It contains the protected setup profile, protected activity cache, and the years available for future reporting.
- **Registered Local User**: The local protected profile created only after a successful full sync and bound to one Ghostfolio security token through unlockability rather than through any stored secret copy.
- **Setup Profile**: The protected server reference and related sync profile data stored together with the protected activity history.
- **Protected Activity Cache**: The normalized, deduplicated, validated full activity history and sync metadata stored for one registered local user.
- **Activity Record**: One stored `BUY` or `SELL` event containing the asset, time, quantity, price, value, fee, comment, and source-scope details needed for future reporting.
- **Source Holding Scope**: Optional account, wallet, or equivalent source grouping preserved from Ghostfolio when present so a future reporting slice can decide whether scope-local treatment is reliable or must broaden to the asset level.
- **Ghostfolio Session**: The temporary authenticated sync context used during one active full-history retrieval and protected-storage attempt.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In controlled valid-history runs, 100% of successful sync attempts retrieve and store the full supported activity history for the selected Ghostfolio server before any reporting behavior becomes available.
- **SC-002**: In controlled empty-history runs, 100% of successful sync attempts create or refresh a valid protected local state that records the successful sync without exposing reporting actions.
- **SC-003**: In controlled invalid-history runs containing unsupported activity types, zero-priced `BUY` records, unresolved same-timestamp ordering, or non-defensible normalized histories, 100% of sync attempts are rejected before any new protected snapshot is created or any existing protected snapshot is replaced.
- **SC-004**: 100% of stored protected snapshots remain unreadable without the correct Ghostfolio security token, and 100% of wrong-token attempts leave existing protected data unchanged.
- **SC-005**: 100% of confirmed server-change syncs replace existing protected data only after explicit confirmation and a fully successful replacement sync, and 100% of declined or failed replacement attempts preserve the prior protected snapshot unchanged.
- **SC-006**: For activity histories of up to 10,000 records spanning at least 5 calendar years, users can complete a successful full sync and protected storage refresh in under 2 minutes on a supported installation.
- **SC-007**: 100% of successful sync outcomes state that activity data was stored for future use and that no reporting, preview, or gains-or-losses output is part of this slice.
- **SC-008**: In controlled precision checks, 100% of stored quantity, price, value, and fee inputs match the source precision exactly, with no rounding or currency conversion applied by this slice.

### Measurement Notes

- Controlled runs use deterministic Ghostfolio-compatible fixtures or equivalent fixed responses so full-history retrieval, invalid-history rejection, and replacement behavior can be audited repeatably.
- A sync counts as successful for **SC-001**, **SC-002**, **SC-006**, **SC-007**, and **SC-008** only when the user-visible outcome also confirms protected local storage and still exposes no reporting workflow.
- Wrong-token and server-replacement outcomes for **SC-004** and **SC-005** are verified both by user-visible workflow results and by confirming that the previously stored protected snapshot remains the active protected local state.

## Assumptions

- The bootstrap setup and basic `Sync Data` workflow introduced in `specs/002-sync-data-validation/` remain the entry point for this slice and do not need to be redesigned here.
- The Ghostfolio activity history stored by this slice is limited to `BUY` and `SELL` records. Any other activity type fails the sync rather than being skipped.
- A valid empty Ghostfolio activity history is a successful sync outcome even though it leaves no stored activity entries.
- Report generation, report preview, cost-basis selection, gains-or-losses calculation, and PDF output remain out of scope until later slices.
- The selected Ghostfolio server provides enough identity, timing, quantity, price, fee, comment, and any available scope information to build a reporting-ready protected cache when the history is valid.
- Losing the Ghostfolio security token permanently prevents recovery of the protected activity cache by design.
- This slice does not introduce a separate reporting or cached-data browsing workflow; it only guarantees that future-reporting-ready data can be fetched, validated, and stored securely.

## Explicitly Deferred In This Slice

- Any gains-or-losses calculation.
- Any cost-basis method selection or explanation.
- Any report generation, preview, export, or filing-oriented workflow.
- Any direct browsing, editing, or exporting of cached activity data.
- Any attempt to recover protected activity data without the correct Ghostfolio security token.
