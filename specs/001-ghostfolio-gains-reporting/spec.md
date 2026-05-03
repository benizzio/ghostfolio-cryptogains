# Feature Specification: Ghostfolio Gains Reporting

**Feature Branch**: `[001-ghostfolio-gains-reporting]`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "Create an installed application that securely connects to a selected Ghostfolio server, stores protected activity history locally, and generates yearly PDF capital gains and losses reports from that history."

## Clarifications

### Session 2026-05-02

- Q: How should the baseline release handle unsupported Ghostfolio activity event types that affect holdings? → A: Refuse sync entirely and show an error stating that the events are unsupported.
- Q: How should the application handle a user changing the selected Ghostfolio server after a protected cache already exists? → A: Detect the mismatch using the stored server reference for that user/token, warn explicitly that continuing will clean the current protected data tied to that user/token, and replace the protected profile/cache only after user confirmation.
- Q: How should the baseline release handle activity history that arrives out of order, contains exact duplicates, or has gaps that make basis calculations non-defensible? → A: Sort chronologically, remove exact duplicates, and reject the sync if gaps or inconsistencies still prevent a defensible calculation.
- Q: How should the spec resolve the conflict between a moving-average method and a scope-local hybrid method? → A: Keep `Average Cost Basis` as one moving weighted-average cost pool per asset using all activity up to each disposal, also support `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order`, and show an informational message when the user selects a method.
- Q: If `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order` is selected but the synced history does not provide reliable wallet/account scope data, what should the application do? → A: Use asset-level scope for that asset while still applying the same method.

### Session 2026-05-03

- Q: Which Ghostfolio activity record types are supported by the baseline sync? → A: Only `BUY` and `SELL`; any other activity type fails the sync.
- Q: How should a `SELL` record with unit price `0` be interpreted? → A: Treat it as a blockchain fee or transfer-out movement that reduces holdings without realizing gain or loss, and require an explanatory comment.
- Q: How should a `BUY` record with unit price `0` be handled? → A: Reject the sync because acquisitions require a defensible non-zero unit price; transfer-in acquisitions must carry their intended basis directly in the `BUY` record, with comments kept only as explanation.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Secure Setup And Sync (Priority: P1)

After installing the application, the user can complete mandatory setup, choose which Ghostfolio server to use, start a short-lived authenticated session, and gather activity data into a protected local cache before any reporting starts.

**Why this priority**: Without installation, setup, and a secure data refresh, the application cannot produce any report or establish user trust.

**Independent Test**: On a fresh installation, the user can complete first-run setup, restart the application with setup retained, enter a valid session token, gather activity data, and receive confirmation that reporting can proceed.

**Acceptance Scenarios**:

1. **Given** the application is opened for the first time, **When** no setup exists, **Then** the application requires setup before allowing access to reporting features.
2. **Given** setup is incomplete, **When** the user tries to start reporting, **Then** the application blocks the action and sends the user to complete setup.
3. **Given** setup exists for a registered local user, **When** the user starts a Ghostfolio interaction session for that local user and provides a valid token, **Then** the application gathers activity data, stores it in protected local storage, confirms the data is ready, and offers report generation.
4. **Given** setup exists, **When** the user updates the selected Ghostfolio server, **Then** the new setup is securely persisted for later sessions.
5. **Given** setup, authentication, connectivity, or protected-storage handling fails during setup or sync, **When** the application informs the user, **Then** the message appears during that workflow, explains the problem and next step, does not reveal the token or unprotected activity data, and is not persisted for later display.
6. **Given** Ghostfolio access or activity retrieval fails for a token, **When** the failed workflow ends, **Then** the application does not create or retain a local registered user entry or protected local data for that failed attempt.
7. **Given** retrieved activity history contains any activity record whose type is not `BUY` or `SELL`, **When** the sync workflow detects it, **Then** the application refuses the sync, does not create or update protected local data for that attempt, and shows an error stating that only `BUY` and `SELL` are supported.
8. **Given** protected local data exists for a registered local user and the currently selected Ghostfolio server does not match the stored server reference for that user, **When** the user starts a sync, **Then** the application shows an explicit warning that continuing will clean the current protected data tied to that user and security token and replace it with data from the newly selected server.
9. **Given** a server mismatch warning is shown, **When** the user declines the replacement, **Then** the application leaves the existing protected data unchanged and does not start the sync.
10. **Given** retrieved activity history arrives out of order or contains exact duplicate records, **When** the sync workflow processes the history, **Then** the application normalizes it into chronological order, removes exact duplicates, and only then evaluates it for persistence.
11. **Given** retrieved activity history still contains gaps or inconsistencies that prevent a defensible basis calculation after normalization, **When** the sync workflow ends, **Then** the application refuses the sync, does not create or update protected local data for that attempt, and shows an actionable error.
12. **Given** retrieved activity history contains a `BUY` record whose normalized unit price is `0`, **When** the sync workflow validates the history, **Then** the application refuses the sync, does not create or update protected local data for that attempt, and shows an actionable error stating that `BUY` records require a non-zero unit price.

---

### User Story 2 - Generate A Yearly Gains Report (Priority: P1)

With activity history available, the user can choose a cost basis method, choose a year present in the stored data, and generate a capital gains and losses report as a PDF for that year only, including the correct asset inclusion and exclusion behavior for that year.

**Why this priority**: This is the primary business outcome of the product and the reason for retrieving and protecting the activity history.

**Independent Test**: With a known multi-year activity history, the user can select each supported cost basis method and an available year, generate a report, and verify the yearly results, asset inclusion rules, reference list behavior, and report-generation error handling.

**Acceptance Scenarios**:

1. **Given** protected activity history is available, **When** the user selects a supported cost basis method and a year present in the data, **Then** the application generates a yearly capital gains and losses report as a PDF.
2. **Given** the selected year has prior history, **When** the report is calculated, **Then** earlier activity is used to establish basis and later activity after the selected year is ignored.
3. **Given** an included asset has zero net gain or loss for the selected year, **When** the report is generated, **Then** that asset still appears in the gains and losses section with a zero result.
4. **Given** an asset has an open position at the end of the selected year, **When** the report is generated, **Then** the asset appears in the main results and detailed sections.
5. **Given** an asset is fully liquidated during the selected year, **When** the report is generated, **Then** the asset appears in the main results and detailed sections.
6. **Given** an asset was fully liquidated before the selected year and has no later reopened position, **When** the report is generated, **Then** the asset is excluded from the main asset sections and shown only in the reference list of previously liquidated assets.
7. **Given** report calculation or report generation fails, **When** the application informs the user, **Then** the message appears during that workflow, explains the problem and next step, does not reveal the token or unprotected activity data, and is not persisted for later display.
8. **Given** activity history includes a `SELL` record with unit price `0` and an explanatory comment, **When** the report is generated, **Then** the application treats that record as a blockchain fee or transfer-out movement that reduces holdings and allocated basis without realizing gain or loss.
9. **Given** the user selects `Average Cost Basis`, **When** a disposal is calculated, **Then** the application uses one moving weighted-average cost pool for that asset using all activity up to that disposal.
10. **Given** the user selects `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order`, **When** a disposal is calculated, **Then** the application matches disposed units within the applicable scope when exact identification is possible and otherwise uses scope-local average-cost valuation with oldest-acquired deemed-disposal ordering in that same scope.
11. **Given** the user selects a cost basis method, **When** the selection changes, **Then** the application displays a message that explains the method's matching rule and any scope-local or fallback behavior using neutral mathematical and data-model terms.
12. **Given** the user selects `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order` and the synced activity history does not provide reliable wallet or account scope data, **When** a disposal is calculated, **Then** the application uses asset-level scope for that asset and still applies the same method.
13. **Given** activity history includes a `BUY` record with a non-zero unit price and an explanatory comment indicating a blockchain transfer reception, **When** the report is generated, **Then** the application uses the provided non-zero unit price to establish acquisition basis for the receiving holdings and does not infer basis linkage from the free-text comment.

---

### Edge Cases

- The user loses the Ghostfolio token after creating a protected local cache and later wants to reopen the cached data.
- The selected year contains acquisitions but no disposals for one or more assets, producing a zero gain or loss while holdings remain open.
- The selected year contains only final liquidating disposals for an asset that had been opened in earlier years.
- Activity history arrives out of order, contains exact duplicates, or contains gaps that prevent a defensible basis calculation; the application must normalize ordering and remove exact duplicates, then refuse the sync if the remaining history is still non-defensible.
- The selected Ghostfolio server is unreachable, responds slowly, or rejects the provided token.
- The user switches setup from one Ghostfolio server to another after a cache already exists; the application must detect the mismatch using the stored server reference, warn that continuing will clean the current protected data tied to that user and security token, and proceed only after confirmation.
- Activity history includes any activity type other than `BUY` or `SELL`; the application must refuse sync, avoid creating or updating protected local data for that attempt, and inform the user that only `BUY` and `SELL` are supported.
- A workflow fails, the user restarts the application, and no stale transient error message is shown unless the failure happens again.
- A token for one registered user is supplied while protected local data exists for a different registered user on the same computer.
- A `SELL` record with unit price `0` and an explanatory comment must reduce holdings and allocated basis without realizing gain or loss.
- A `BUY` record with unit price `0` must be rejected during sync because acquisition basis cannot be derived defensibly.
- `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order` is selected for a dataset that lacks reliable wallet or account scope information, so the asset as a whole becomes `applicable_scope`.
- Exact unit identification is not defensible within a reliable scope, so the scope-local hybrid method must fall back to scope-local average cost and remain pooled for that partition until quantity reaches zero.
- Multiple same-asset events share the same timestamp; deterministic ordering must resolve them by `source_id` or the sync must be rejected as non-defensible.
- A blockchain transfer is represented by a zero-priced `SELL` plus a priced `BUY`; the receiving `BUY` must carry the intended acquisition price directly, and comments are explanatory only.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, precision,
testing, dependency, and external integration impacts when the feature touches
those areas.

### Functional Requirements

- **FR-001**: The system MUST provide an installation process that leaves the user with a usable local application.
- **FR-002**: The system MUST require setup on first interaction before allowing any reporting workflow.
- **FR-003**: The system MUST allow the user to choose the default Ghostfolio cloud server or provide a self-hosted Ghostfolio server address during setup.
- **FR-004**: The system MUST persist each registered user's setup data between sessions in cryptographically protected local storage and MUST allow that user to update the setup after unlocking with the informed Ghostfolio security token.
- **FR-005**: The system MUST require the user to start a Ghostfolio interaction session for a specific registered local user and provide a Ghostfolio security token before each data-gathering workflow.
- **FR-006**: The system MUST use the Ghostfolio security token only for the active session and MUST not leave a recoverable token trace after the application interaction or process ends.
- **FR-007**: The system MUST gather asset activity history needed for capital gains and losses reporting from the selected Ghostfolio server after a valid session begins.
- **FR-008**: The system MUST maintain an updatable local cache of gathered activity history for each successfully registered user for reuse across sessions.
- **FR-009**: The system MUST cryptographically protect all locally persisted user-related data, including registered-user metadata, setup data, and gathered activity history, using protection unlocked only by the same Ghostfolio security token so the local data cannot be recovered if the token is lost.
- **FR-010**: The system MUST confirm that data gathering completed successfully before offering report generation.
- **FR-011**: The system MUST block report generation until setup is complete and a successful activity sync exists for the selected Ghostfolio server.
- **FR-012**: The system MUST offer only one report type in the baseline release: a capital gains and losses report.
- **FR-013**: The system MUST let the user choose one baseline cost basis method before report generation from this method set: FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order.
- **FR-014**: The system MUST apply the selected cost basis method consistently to all included disposals in the generated report.
- **FR-015**: The system MUST let the user choose only a year that is present in the stored activity history.
- **FR-016**: The system MUST calculate gains and losses for the selected year using activity before and within that year to establish basis and MUST ignore activity after that year.
- **FR-017**: The system MUST generate the baseline report only as a PDF.
- **FR-018**: The system MUST include in the main report sections every asset that has an open position at the end of the selected year or is fully liquidated during the selected year.
- **FR-019**: The system MUST exclude from the main report sections any asset fully liquidated before the selected year that has no later reopened position and MUST list those assets separately as a reference.
- **FR-020**: The system MUST present a first report section that lists each included asset and its gain, loss, or zero result for the selected year.
- **FR-021**: The system MUST present detailed report sections after the gains and losses section, grouped by asset and ordered as opening position before the selected year and activity within the selected year, while ignoring position states after the selected year.
- **FR-022**: The system MUST provide secure user-facing error messages during setup, session, sync, storage, and report workflows without exposing secrets or unprotected cached data.
- **FR-023**: The system MUST not persist transient failure messages or secret-bearing diagnostic details for later display after the failing workflow ends.
- **FR-024**: The system MUST create or retain a local registered user only after successful Ghostfolio access and successful activity retrieval, and MUST not maintain a local user profile when Ghostfolio access or retrieval fails.
- **FR-025**: The system MUST support only Ghostfolio `BUY` and `SELL` activity records when consuming Ghostfolio data.
- **FR-026**: The system MUST refuse a data-gathering workflow when retrieved activity history contains any activity record whose type is not `BUY` or `SELL`, MUST not create or update protected local data for that failed attempt, and MUST show a user-facing error stating that only `BUY` and `SELL` are supported.
- **FR-027**: The system MUST store with each registered local user the Ghostfolio server reference used for that user's current protected data and MUST compare it on later sync attempts to detect server mismatches after configuration changes.
- **FR-028**: The system MUST show an explicit confirmation when the selected Ghostfolio server does not match the stored server reference for the current registered local user, and that confirmation MUST state that continuing will clean the current protected data tied to that user and security token.
- **FR-029**: The system MUST replace the current protected setup/profile and activity cache for a registered local user with data from the newly selected Ghostfolio server only after the user confirms the server-mismatch warning.
- **FR-030**: The system MUST normalize retrieved activity history into chronological order and remove exact duplicate records before deciding whether the history can be persisted for reporting use.
- **FR-031**: The system MUST refuse a data-gathering workflow when normalized activity history still contains gaps or inconsistencies that prevent a defensible basis calculation, MUST not create or update protected local data for that failed attempt, and MUST show an actionable user-facing error.
- **FR-032**: The system MUST define Average Cost Basis as one moving weighted-average cost pool per asset using all activity up to each disposal, where average_unit_cost = pool_basis / pool_quantity, allocated_basis = disposed_quantity * average_unit_cost, pool_quantity' = pool_quantity - disposed_quantity, and pool_basis' = pool_basis - allocated_basis.
- **FR-033**: The system MUST define Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order as follows: for each disposal, determine the applicable scope as the reliable wallet or account scope for that asset when available, otherwise the asset as a whole; partition holdings by (asset, applicable_scope); if the disposed units are exactly identifiable within that partition, allocate basis from those exact units using allocated_basis = sum(matched_quantity_i * matched_unit_cost_i); otherwise compute average_unit_cost = partition_basis / partition_quantity immediately before the disposal, compute allocated_basis = disposed_quantity * average_unit_cost, reduce the partition basis and quantity by that amount, and assign deemed-disposal order to the oldest acquired remaining quantities first, ordered by occurred_at ascending and then source_id ascending.
- **FR-034**: The system MUST show an informational message when the user selects a cost basis method, and that message MUST explain the method's matching rule and any scope-local or fallback behavior in neutral mathematical and data-model terms.
- **FR-035**: The system MUST apply Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order using asset-level scope when the synced activity history does not provide reliable wallet or account scope data.
- **FR-036**: The system MUST treat a `SELL` activity record with unit price `0` and an explanatory comment as a blockchain fee or transfer-out movement that reduces holdings and allocated basis without realizing gain or loss.
- **FR-037**: The system MUST require every normalized `BUY` activity record to have a non-zero unit price, MUST refuse sync when a `BUY` record normalizes to unit price `0`, and MUST use the provided non-zero unit price from transfer-in `BUY` records as the acquisition basis input instead of inferring basis from free-text comments.

### Security, Precision, and Integration Constraints

- **SEC-001**: Secret input MUST be entered explicitly by the user for each Ghostfolio interaction session for a specific registered local user, kept only for the duration of the active session, excluded from logs and report output, and cleared when the session ends or fails.
- **SEC-002**: Persisted user-related data MUST remain local to the user's computer and MUST be protected in accordance with the OWASP Cryptographic Storage Cheat Sheet best practices, including minimising stored sensitive data, using established cryptography with integrity protection, secure random generation, and separation of protected data from keying material where feasible. All locally persisted user-related data MUST be unlockable only with the informed Ghostfolio security token.
- **FIN-001**: Every quantity, price, fee, proceeds value, and gain or loss calculation MUST use exact decimal arithmetic, preserve source precision, include transaction fees in basis or disposal proceeds when present in source data, treat a `SELL` record with unit price `0` and an explanatory comment as a non-taxable holding reduction with zero realized gain or loss, reject normalized `BUY` records with unit price `0`, and apply a single documented rounding policy only at user-visible output boundaries.
- **QUAL-001**: Automated validation MUST cover installation gating, setup persistence, successful and failed local-user registration behavior, session-token non-persistence, cache unreadability without the token, yearly boundary handling, each supported cost basis method, `BUY`/`SELL`-only sync validation, zero-priced `SELL` non-taxable holding reduction, zero-priced `BUY` rejection, transfer-in `BUY` records using their explicit non-zero unit price instead of comment-based linkage, server-mismatch warning and confirmed replacement behavior, chronological normalization and exact-duplicate removal during sync, rejection of non-defensible normalized histories, exact-unit identification possible within a reliable scope, exact-unit identification impossible within a reliable scope triggering scope-local average fallback, unreliable scope triggering asset-level scope under the same method, pooled-until-zero behavior after the first average-cost fallback in an open partition, asset inclusion and exclusion rules, and final report section ordering by using deterministic sample ledgers and controlled source-system responses.
- **INT-001**: The feature depends on the selected Ghostfolio server exposing authenticated activity history as `BUY` and `SELL` records with enough asset identity, timestamps, quantities, non-zero acquisition pricing for `BUY`, values, fee information, explanatory comments for zero-priced `SELL`, and any available source holding-scope identity to support reproducible yearly capital gains and losses calculations, including source scope data that can narrow `applicable_scope` when reliable and broaden it to asset-level scope when it is not.

### Key Entities *(include if feature involves data)*

- **Registered Local User**: A local user record created only after successful Ghostfolio access and successful activity retrieval, bound to one Ghostfolio security token, owning the protected setup and cache for that user, and storing the Ghostfolio server reference associated with that protected data.
- **Setup Profile**: The protected per-registered-user configuration that identifies which Ghostfolio server the user has chosen and whether setup is complete.
- **Ghostfolio Session**: The temporary authenticated interaction context created when the user provides a Ghostfolio token for the current application run for a specific registered local user.
- **Activity Record**: A timestamped Ghostfolio `BUY` or `SELL` event, including the data needed to determine holdings, basis, proceeds, fees, zero-priced `SELL` explanations, and any available source holding scope used to derive `applicable_scope`.
- **Source Holding Scope**: The wallet, account, or equivalent source grouping associated with activity records when a selected cost basis method can narrow `applicable_scope` to that reliable scope.
- **Protected Activity Cache**: The local persisted collection of activity records for one registered user that can be refreshed from the selected Ghostfolio server and can only be read after token-based unlock.
- **Asset Position Timeline**: The per-asset chronological view derived from activity history that shows opening holdings before the selected year, in-year changes, liquidations, and the resulting position at the end of the selected year.
- **Report Request**: The user's selected cost basis method, report year, and output choice for a single report run.
- **Capital Gains Report**: The yearly user-facing document that contains the gains and losses summary, the reference list of previously liquidated assets, and detailed per-asset opening-position and in-year transaction sections.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of first-time users can complete installation and mandatory setup and reach the data-gathering step in under 5 minutes using only application prompts.
- **SC-002**: For controlled validation ledgers with known expected outcomes, including zero-priced non-taxable `SELL` cases, 100% of per-asset results and yearly totals match the expected gains and losses for each supported cost basis method.
- **SC-003**: 100% of generated reports place the gains and losses summary first and the detailed per-asset sections in the required opening-position and in-year order without including position states after the selected year.
- **SC-004**: 100% of assets with open positions at year end or liquidations during the selected year are included in the main report sections, and 100% of assets liquidated before the selected year with no reopened position are excluded from those sections and shown only in the reference list.
- **SC-005**: 100% of failed token, connectivity, storage, and calculation scenarios produce a user-visible error message during the failing workflow that explains the failure and recommended next action without exposing secret values or unprotected activity history, and those messages are not shown again after restart unless the failure recurs.
- **SC-006**: For a stored history of up to 10,000 activity records spanning at least 5 calendar years, users can generate a yearly PDF report in under 2 minutes on a supported installation.
- **SC-007**: 100% of controlled invalid ledgers containing an activity type other than `BUY` or `SELL` or a zero-priced `BUY` are rejected before persistence with a user-visible sync error.

## Assumptions

- The application may maintain multiple locally registered users on the same computer, but each registered user is created only after successful Ghostfolio access and successful activity retrieval and can be unlocked only with that user's Ghostfolio security token.
- The Ghostfolio activity history consumed by this release is limited to `BUY` and `SELL` records; any other activity type causes sync failure.
- Ghostfolio may include `SELL` records with unit price `0` and explanatory comments for blockchain fees or transfer-out movements; these records reduce holdings without realizing gain or loss.
- Transfer-in or similar receiving acquisitions arrive as `BUY` records with a non-zero unit price that already expresses the intended acquisition basis; comments are explanatory only and are not used to infer basis linkage between records.
- The baseline cost basis scope is limited to FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order. Additional cost-basis methods are out of scope for this release.
- The selected Ghostfolio server provides enough history, value, fee, and explanatory comment information to derive one consistent yearly reporting view for the user's stored data.
- PDF output in this release is intended for review and record-keeping, not as a filing form.
- Losing the Ghostfolio token permanently prevents recovery of the protected activity cache by design.
