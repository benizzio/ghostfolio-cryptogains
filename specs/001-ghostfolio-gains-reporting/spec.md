# Feature Specification: Ghostfolio Gains Reporting

**Feature Branch**: `[001-ghostfolio-gains-reporting]`  
**Created**: 2026-05-02  
**Status**: Draft  
**Input**: User description: "Create an installed application that securely connects to a selected Ghostfolio server, stores protected activity history locally, and generates yearly PDF capital gains and losses reports from that history."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Secure Setup And Sync (Priority: P1)

After installing the application, the user can complete mandatory setup, choose which Ghostfolio server to use, start a short-lived authenticated session, and gather activity data into a protected local cache before any reporting starts.

**Why this priority**: Without installation, setup, and a secure data refresh, the application cannot produce any report or establish user trust.

**Independent Test**: On a fresh installation, the user can complete first-run setup, restart the application with setup retained, enter a valid session token, gather activity data, and receive confirmation that reporting can proceed.

**Acceptance Scenarios**:

1. **Given** the application is opened for the first time, **When** no setup exists, **Then** the application requires setup before allowing access to reporting features.
2. **Given** setup is incomplete, **When** the user tries to start reporting, **Then** the application blocks the action and sends the user to complete setup.
3. **Given** setup exists, **When** the user starts a Ghostfolio interaction session and provides a valid token, **Then** the application gathers activity data, stores it in protected local storage, confirms the data is ready, and offers report generation.
4. **Given** setup exists, **When** the user updates the selected Ghostfolio server, **Then** the new setup is securely persisted for later sessions.
5. **Given** setup, authentication, connectivity, or protected-storage handling fails during setup or sync, **When** the application informs the user, **Then** the message appears during that workflow, explains the problem and next step, does not reveal the token or unprotected activity data, and is not persisted for later display.
6. **Given** Ghostfolio access or activity retrieval fails for a token, **When** the failed workflow ends, **Then** the application does not create or retain a local registered user entry or protected local data for that failed attempt.

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
8. **Given** activity history includes a `BUY` or `SELL` record with unit price `0` and an explanatory comment, **When** the report is generated, **Then** the application interprets that record as a non-fiat asset movement or crypto-denominated fee event that adjusts holdings and calculations according to its direction rather than as a fiat trade with zero value.

---

### Edge Cases

- The user loses the Ghostfolio token after creating a protected local cache and later wants to reopen the cached data.
- The selected year contains acquisitions but no disposals for one or more assets, producing a zero gain or loss while holdings remain open.
- The selected year contains only final liquidating disposals for an asset that had been opened in earlier years.
- Activity history arrives out of order, contains duplicates, or contains gaps that prevent a defensible basis calculation.
- The selected Ghostfolio server is unreachable, responds slowly, or rejects the provided token.
- The user switches setup from one Ghostfolio server to another after a cache already exists.
- Activity history includes unsupported event types that affect holdings but are outside the baseline release scope.
- A workflow fails, the user restarts the application, and no stale transient error message is shown unless the failure happens again.
- A token for one registered user is supplied while protected local data exists for a different registered user on the same computer.
- `BUY` or `SELL` records with unit price `0` and explanatory comments must affect holdings and gains logic correctly without being treated as fiat purchases or fiat sales.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, precision,
testing, dependency, and external integration impacts when the feature touches
those areas.

### Functional Requirements

- **FR-001**: The system MUST provide an installation process that leaves the user with a usable local application.
- **FR-002**: The system MUST require setup on first interaction before allowing any reporting workflow.
- **FR-003**: The system MUST allow the user to choose the default Ghostfolio cloud server or provide a self-hosted Ghostfolio server address during setup.
- **FR-004**: The system MUST persist each registered user's setup data between sessions in cryptographically protected local storage and MUST allow that user to update the setup after unlocking with the informed Ghostfolio security token.
- **FR-005**: The system MUST require the user to start a Ghostfolio interaction session and provide a Ghostfolio security token before each data-gathering workflow.
- **FR-006**: The system MUST use the Ghostfolio security token only for the active session and MUST not leave a recoverable token trace after the application interaction or process ends.
- **FR-007**: The system MUST gather asset activity history needed for capital gains and losses reporting from the selected Ghostfolio server after a valid session begins.
- **FR-008**: The system MUST maintain an updatable local cache of gathered activity history for each successfully registered user for reuse across sessions.
- **FR-009**: The system MUST cryptographically protect all locally persisted user-related data, including registered-user metadata, setup data, and gathered activity history, using protection unlocked only by the same Ghostfolio security token so the local data cannot be recovered if the token is lost.
- **FR-010**: The system MUST confirm that data gathering completed successfully before offering report generation.
- **FR-011**: The system MUST block report generation until setup is complete and a successful activity sync exists for the selected Ghostfolio server.
- **FR-012**: The system MUST offer only one report type in the baseline release: a capital gains and losses report.
- **FR-013**: The system MUST let the user choose one baseline cost basis method before report generation from this method set: FIFO, LIFO, HIFO, and Average Cost Basis.
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
- **FR-025**: The system MUST interpret `BUY` and `SELL` activity records with unit price `0` and explanatory comments as non-fiat asset movements or crypto-denominated fee events that still change holdings and report calculations according to their direction.

### Security, Precision, and Integration Constraints

- **SEC-001**: Secret input MUST be entered explicitly by the user for each Ghostfolio interaction session, kept only for the duration of the active session, excluded from logs and report output, and cleared when the session ends or fails.
- **SEC-002**: Persisted user-related data MUST remain local to the user's computer and MUST be protected in accordance with the OWASP Cryptographic Storage Cheat Sheet best practices, including minimising stored sensitive data, using established cryptography with integrity protection, secure random generation, and separation of protected data from keying material where feasible. All locally persisted user-related data MUST be unlockable only with the informed Ghostfolio security token.
- **FIN-001**: Every quantity, price, fee, proceeds value, and gain or loss calculation MUST use exact decimal arithmetic, preserve source precision, include transaction fees in basis or disposal proceeds when present in source data, interpret `BUY` and `SELL` records with unit price `0` and explanatory comments according to their non-fiat economic effect, and apply a single documented rounding policy only at user-visible output boundaries.
- **QUAL-001**: Automated validation MUST cover installation gating, setup persistence, successful and failed local-user registration behavior, session-token non-persistence, cache unreadability without the token, yearly boundary handling, each supported cost basis method, zero-priced non-fiat movement handling, asset inclusion and exclusion rules, and final report section ordering by using deterministic sample ledgers and controlled source-system responses.
- **INT-001**: The feature depends on the selected Ghostfolio server exposing authenticated activity history with enough asset identity, timestamps, quantities, values, fee information, and explanatory comments to support reproducible yearly capital gains and losses calculations, including zero-priced non-fiat movements.

### Key Entities *(include if feature involves data)*

- **Registered Local User**: A local user record created only after successful Ghostfolio access and successful activity retrieval, bound to one Ghostfolio security token and owning the protected setup and cache for that user.
- **Setup Profile**: The protected per-registered-user configuration that identifies which Ghostfolio server the user has chosen and whether setup is complete.
- **Ghostfolio Session**: The temporary authenticated interaction context created when the user provides a Ghostfolio token for the current application run.
- **Activity Record**: A timestamped asset acquisition or disposal event, including the data needed to determine holdings, basis, proceeds, and fees.
- **Protected Activity Cache**: The local persisted collection of activity records for one registered user that can be refreshed from the selected Ghostfolio server and can only be read after token-based unlock.
- **Asset Position Timeline**: The per-asset chronological view derived from activity history that shows opening holdings before the selected year, in-year changes, liquidations, and the resulting position at the end of the selected year.
- **Report Request**: The user's selected cost basis method, report year, and output choice for a single report run.
- **Capital Gains Report**: The yearly user-facing document that contains the gains and losses summary, the reference list of previously liquidated assets, and detailed per-asset opening-position and in-year transaction sections.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of first-time users can complete installation and mandatory setup and reach the data-gathering step in under 5 minutes using only application prompts.
- **SC-002**: For controlled validation ledgers with known expected outcomes, including zero-priced non-fiat movement cases, 100% of per-asset results and yearly totals match the expected gains and losses for each supported cost basis method.
- **SC-003**: 100% of generated reports place the gains and losses summary first and the detailed per-asset sections in the required opening-position and in-year order without including position states after the selected year.
- **SC-004**: 100% of assets with open positions at year end or liquidations during the selected year are included in the main report sections, and 100% of assets liquidated before the selected year with no reopened position are excluded from those sections and shown only in the reference list.
- **SC-005**: 100% of failed token, connectivity, storage, and calculation scenarios produce a user-visible error message during the failing workflow that explains the failure and recommended next action without exposing secret values or unprotected activity history, and those messages are not shown again after restart unless the failure recurs.
- **SC-006**: For a stored history of up to 10,000 activity records spanning at least 5 calendar years, users can generate a yearly PDF report in under 2 minutes on a supported installation.

## Assumptions

- The application may maintain multiple locally registered users on the same computer, but each registered user is created only after successful Ghostfolio access and successful activity retrieval and can be unlocked only with that user's Ghostfolio security token.
- Ghostfolio activity history may include `BUY` and `SELL` records with unit price `0` and explanatory comments that represent non-fiat movements such as swaps or crypto-denominated network, wallet, or swap fees; these records still affect holdings and calculations according to their direction.
- The baseline cost basis scope is limited to FIFO, LIFO, HIFO, and Average Cost Basis. Jurisdiction-specific matching or anti-avoidance rules such as same-day matching, 30-day matching, superficial loss rules, or whole-portfolio disposal formulas are out of scope for this release.
- The selected Ghostfolio server provides enough history, value, fee, and explanatory comment information to derive one consistent yearly reporting view for the user's stored data.
- PDF output in this release is intended for review and record-keeping, not as a jurisdiction-specific filing form.
- Losing the Ghostfolio token permanently prevents recovery of the protected activity cache by design.
