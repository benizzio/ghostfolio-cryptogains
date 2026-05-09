# Feature Specification: Sync Data Validation

**Feature Branch**: `[002-sync-data-validation]`  
**Created**: 2026-05-09  
**Status**: Ready  
**Input**: User description: "We created before the specs on @specs/001-ghostfolio-gains-reporting/ but their scope is too big. They will now become only source of knowledge and we will create smaller specs to tackle the features with higher granularity. Let's create the first one that will tackle base boilerplate creation of the application and setup, including the selection of the sync data feature and a validation that it works, and ONLY THAT. In this initial spec, when sync data is selected by the user to be executed, the only thing the application will do is receive the result of the call to obtain data from ghostfolio, validate that the received data and request result is ok and give a message to the user that communication is ok, and the actual persistence and report generation will be available in future versions (will be added in future specs)"

## Clarifications

### Session 2026-05-09

- Q: Should setup state be remembered between application runs in this slice? → A: Persist setup between runs immediately.
- Q: What counts as a valid data-retrieval result for communication validation in this slice? → A: HTTP success plus minimal payload shape.
- Q: What self-hosted server origins are allowed in this slice? → A: Dev HTTP, prod HTTPS only.
- Q: How should remembered setup state be protected in this slice? → A: Machine-local protected config.

## Plain-Language Summary

- The application saves only the minimum local setup needed to remember which Ghostfolio server the user chose.
- The only business workflow in this slice is checking whether the application can talk to that server with the user's Ghostfolio security token.
- A successful check proves communication only. It does not store synced data or create any report.

## Terms Used In This Spec

- **Saved setup**: Startup-readable machine-local settings for this slice. They remember only setup completion and server-selection information.
- **Machine-local setup storage**: The current computer user's application settings location. It is not shared across computer user accounts.
- **Compatible Ghostfolio server**: A selected server that supports the limited authentication and one-page activity check required by this slice.
- **Incompatible server**: A reachable server that responds but does not support that limited contract or returns a response that breaks it.
- **First actionable screen**: The first full-screen view that shows an enabled primary action or a required labeled input the user can use immediately.
- **Explicit development mode**: A deliberate startup opt-in intentionally supplied for the current run. Saved setup, server behavior, or ambient environment assumptions can never turn it on by themselves.
- **Temporary session credential**: The short-lived server-issued credential returned after a successful token check for the active validation attempt.
- **Timeout**: A validation attempt that fails because no usable server response was received within 30 seconds after the active validation request started.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Complete Initial Setup (Priority: P1)

A user starting the application for the first time can complete the minimum setup needed to choose a Ghostfolio server and reach the main feature selection flow.

**Why this priority**: The application cannot validate Ghostfolio communication until the user can start the program and define which Ghostfolio server it should contact.

**Independent Test**: On a fresh run, the user can open the application, choose the default Ghostfolio cloud server or provide an allowed self-hosted server origin, complete setup, restart the application, and reach the point where feature execution can be selected without redoing setup or entering the Ghostfolio security token at startup.

**Acceptance Scenarios**:

1. **Given** the application is launched for the first time, **When** no setup exists yet, **Then** the application requires the user to complete setup before any feature execution starts.
2. **Given** the user is in setup, **When** the user selects the hosted Ghostfolio service, **Then** the application records that choice and advances to the main feature selection flow.
3. **Given** the user is in setup, **When** the user provides a self-hosted Ghostfolio server origin that uses `https`, or uses `http` while the application is explicitly running in development mode, **Then** the application records that choice and advances to the main feature selection flow.
4. **Given** the application is not explicitly running in development mode, **When** the user provides a self-hosted `http` server origin during setup, **Then** the application rejects that origin and requires the user to provide an allowed server origin.
5. **Given** setup was completed in a previous run, **When** the user launches the application again, **Then** the application reuses the remembered setup and selected Ghostfolio server without requiring setup to be repeated or the Ghostfolio security token to be entered at startup.
6. **Given** remembered setup exists, **When** startup reload determines that the stored server address is malformed, cannot be normalized into a valid allowed form, or no longer satisfies the current transport-safety rule, **Then** the application makes no Ghostfolio network request, explains that the saved server selection is no longer valid, and returns the user to setup.
7. **Given** remembered setup exists, **When** the user edits setup and saves a different valid server choice, **Then** the application atomically replaces the previous remembered server selection and uses the new selection for later validation attempts and the next launch.
8. **Given** remembered setup exists, **When** the user changes setup inputs but leaves setup before saving, **Then** the application keeps the previous remembered setup unchanged.
9. **Given** a valid remembered setup was loaded for the current run, **When** the bootstrap setup file is removed before the next persisted read, **Then** the current run continues using the already loaded in-memory setup and the next launch returns to first-run setup unless the user saves new valid setup first.
10. **Given** setup is incomplete, **When** the user attempts to execute a feature, **Then** the application blocks the action and returns the user to finish setup.
11. **Given** the user needs to locate or reset remembered setup, **When** the user reads the project documentation for this slice, **Then** the documentation names the saved-setup location for Linux, macOS, and Windows, explains the expected local protection approach on each platform, and explains the safe removal procedure that resets setup on the next launch.

---

### User Story 2 - Select Sync Data Feature (Priority: P1)

After setup is complete, a user can select the sync data feature as the only available business workflow in this release.

**Why this priority**: The requested scope explicitly includes feature selection and limits this first slice to the sync-validation workflow only.

**Independent Test**: After setup completes, the user can reach a feature-selection step, choose sync data, and start the validation workflow without accessing data storage or reporting behavior.

**Acceptance Scenarios**:

1. **Given** setup is complete, **When** the user reaches the main workflow selection step, **Then** the application shows sync data as an executable feature.
2. **Given** the user selects sync data, **When** the workflow starts, **Then** the application prompts only for the Ghostfolio security token needed to validate communication with the selected Ghostfolio server.
3. **Given** the user is using this release, **When** the user finishes setup and enters the main workflow selection step, **Then** the application does not offer persistence or report-generation workflows as executable outcomes.

---

### User Story 3 - Validate Ghostfolio Communication (Priority: P1)

When the user runs sync data, the application authenticates and probes activity data using this slice's supported Ghostfolio communication contract, validates that the request succeeded and that the response includes the minimal payload shape required by that contract, and informs the user whether communication is working.

**Why this priority**: This is the sole business outcome of the requested slice and confirms that the application can communicate correctly before later specs add storage or reporting.

**Independent Test**: With a reachable compatible Ghostfolio server and a valid Ghostfolio security token, the user can run sync data and receive a success message. With a rejected Ghostfolio security token, timeout, unreachable server, unsuccessful response, or incompatible retrieval result, the user receives a failure message in the correct failure category and no later-stage behavior occurs.

**Acceptance Scenarios**:

1. **Given** setup is complete and the selected Ghostfolio server is reachable and compatible, **When** the user starts sync data and provides a valid Ghostfolio security token, **Then** the application authenticates with Ghostfolio, requests activity data, validates that the response is successful and includes the minimal payload shape required by this slice's supported communication contract, and shows a success message confirming communication is working.
2. **Given** the selected Ghostfolio server rejects the provided Ghostfolio security token, **When** the user starts sync data, **Then** the application shows a failure result in the `rejected token` category.
3. **Given** the validation request exceeds the slice's allowed wait time, **When** the user starts sync data, **Then** the application shows a failure result in the `timeout` category.
4. **Given** the selected Ghostfolio server is unreachable or the network connection cannot be completed, **When** the user starts sync data, **Then** the application shows a failure result in the `connectivity problem` category.
5. **Given** the selected Ghostfolio server responds with a non-2xx HTTP result that does not mean rejected token and does not prove contract incompatibility, **When** the user starts sync data, **Then** the application shows a failure result in the `unsuccessful server response` category.
6. **Given** the selected Ghostfolio server responds but does not support this slice's contract, including unsupported response format, malformed structured data, contradictory activity-count information, missing required fields, or an unreadable first-activity date, **When** the application validates the result, **Then** the application shows a failure result in the `incompatible server contract` category.
7. **Given** sync data completes successfully, **When** the application shows the workflow result, **Then** the application confirms that communication is working, explicitly states that data has not been stored or prepared for reporting, does not persist retrieved data, and does not start any report-generation flow.
8. **Given** a previous sync data attempt failed, **When** the user starts sync data again, **Then** the application allows a new validation attempt without requiring setup to be repeated.
9. **Given** a previous sync data attempt succeeded, **When** the user chooses to validate again or starts sync data again later in the same run, **Then** the application allows a new validation attempt without requiring setup to be repeated.
10. **Given** the user exits the application or the application terminates during an in-flight validation attempt, **When** the user launches the application again, **Then** the previous attempt is not resumed and no earlier validation result is shown.

---

### Edge Cases

- The user provides a malformed self-hosted server origin, a non-`https` origin outside development mode, or any other origin not allowed by the setup rules.
- Remembered setup becomes invalid at startup because the stored server address is malformed, cannot be normalized into a valid allowed form, or an `http` origin is reloaded without explicit development mode.
- The selected Ghostfolio server responds but returns an unexpected or unsupported response format, malformed structured data, contradictory `count` and `activities` values, or omits the minimal payload shape required by the validated Ghostfolio communication contract.
- The selected Ghostfolio server accepts authentication and returns no data entries; the application treats this as successful communication if the retrieval result is otherwise valid.
- The network request times out after the user starts sync data, and the user-visible outcome must identify timeout rather than a generic failure bucket.
- The user leaves setup before saving; partial setup must not replace previous remembered setup or create a new remembered setup.
- The user exits during an in-flight validation attempt; the attempt must be abandoned, secrets cleared from transient state, and no result resumed after restart.
- The bootstrap setup file is removed after launch; the current run continues from in-memory setup only, and the next launch returns to first-run setup unless the user saves valid setup again first.
- The application encounters a crash, trace, or diagnostic event during the workflow; the Ghostfolio security token must not appear in any application-produced logs, dumps, traces, crash text, or persisted diagnostics.
- The user completes setup successfully, but exits before selecting a feature.
- The user expects data to be stored after a successful communication check; the application must clearly state that persistence is not part of this release.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, precision,
testing, dependency, and external integration impacts when the feature touches
those areas.

### Functional Requirements

- **FR-001**: The system MUST provide a runnable base application that guides the user through first-run setup before any business workflow can be executed.
- **FR-002**: The system MUST allow the user during setup to choose either the hosted Ghostfolio service or a self-hosted Ghostfolio server origin.
- **FR-003**: The system MUST accept a self-hosted server origin only when it uses `https`, except that `http` origins MAY be accepted only while the application is started in explicit development mode through a deliberate startup opt-in.
- **FR-004**: The system MUST prevent entry into any business workflow until a valid saved setup has been loaded or created through setup. In this slice, a valid saved setup contains only the startup-readable fields listed for the saved setup record, has setup marked complete, stores one normalized allowed server address, and contains no Ghostfolio token, temporary session credential, synced data, or local user profile data.
- **FR-005**: The system MUST persist only these startup-readable saved-setup fields between runs: format version, setup-complete flag, selected server mode, normalized server address, development-mode HTTP allowance flag, and last-updated timestamp. The system MUST store them only in machine-local setup storage scoped to the current computer user account, MUST rewrite the full document atomically, and MUST use the strongest user-scoped filesystem protection available on the current platform.
- **FR-006**: The system MUST present sync data as the executable feature in this release after setup is complete.
- **FR-007**: The system MUST require only the Ghostfolio security token from the user to validate communication with the selected Ghostfolio server when the user chooses sync data.
- **FR-008**: The system MUST use the user-supplied Ghostfolio security token for that active validation attempt to authenticate with the selected Ghostfolio server and request activity data through the validated Ghostfolio communication contract for this slice.
- **FR-009**: The system MUST validate both the request result and the returned payload before declaring communication successful. Validation MUST reject malformed or unsupported success responses, missing required fields, contradictory activity-count information, and semantically invalid minimum fields such as an unreadable first-activity date.
- **FR-010**: The system MUST treat communication validation as successful only when the selected Ghostfolio server accepts the request, returns a successful activity-retrieval response, and includes the minimal payload shape required by the validated Ghostfolio communication contract, even if the returned activity list is empty. A one-page probe is invalid when `count > 0` but no activity is returned, when `count == 0` but activities are returned, or when more than one activity is returned for the probe.
- **FR-011**: The system MUST map a successful communication-validation outcome to a user-visible success result that states that communication with the selected server is working, states that no data was stored locally, states that no report-generation workflow ran or became available, and offers the user a next-step path to validate again or return to the main menu.
- **FR-012**: The system MUST map a failed communication-validation outcome to a user-visible failure result. The failure result MUST identify exactly one failure category for the finished attempt from this slice's supported set: rejected Ghostfolio security token, timeout, connectivity problem, unsuccessful server response, or incompatible server contract. The failure result MUST explain that communication validation did not succeed, MUST avoid secrets and raw unprotected payload data, and MUST offer a retry path without requiring setup to be repeated.
- **FR-013**: The system MUST end the workflow after showing the communication-validation result without persisting the retrieved data or entering any report-related workflow.
- **FR-014**: This release MUST not expose report-generation as an executable action or produce report output.
- **FR-015**: The success result in the sync data workflow MUST inform the user that successful communication validation does not yet mean data has been stored, normalized, or prepared for reporting.
- **FR-016**: The system MUST allow the user to re-run sync data after any completed validation attempt, including both success and failure, without requiring setup to be repeated.
- **FR-017**: The system MUST document, in user-facing project documentation, where the local bootstrap setup file is stored on Linux, macOS, and Windows, which protections are expected on each platform, and a safe removal procedure that resets setup on the next launch; the runtime workflow does not need to display this path.
- **FR-018**: The system MUST treat remembered setup as invalid at startup when the stored server address is malformed, cannot be normalized into a valid allowed form, or no longer satisfies the current transport-safety rule. In that case the system MUST make no Ghostfolio network request, MUST explain that setup must be completed again because the saved server selection is no longer valid, and MUST return the user to setup.
- **FR-019**: When the user edits remembered setup and saves a new valid server selection, the system MUST atomically replace the previous remembered setup with the new selection and updated metadata. Until that save succeeds, the previous valid remembered setup remains in effect.
- **FR-020**: If the user leaves setup before saving, the system MUST not persist partial setup state. If a previous valid remembered setup exists, it MUST remain unchanged; otherwise the next launch MUST begin in first-run setup.
- **FR-021**: If the remembered setup file is removed after the application has already loaded a valid setup into memory, the current run MAY continue using that in-memory setup. A later launch without a new successful setup save MUST return to first-run setup.
- **FR-022**: The system MUST treat user exit or application termination during an in-flight validation request as an abandoned attempt: no success or failure result is remembered for later display, the token and any temporary session credential are cleared from transient state, and the next launch returns only to setup or the main menu according to remembered setup.

### Acceptance Coverage By Requirement

- `FR-001` to `FR-005`, `FR-018`, `FR-019`, `FR-020`, and `FR-021` are accepted by User Story 1 scenarios 1 through 10.
- `FR-006`, `FR-007`, and `FR-014` are accepted by User Story 2 scenarios 1 through 3.
- `FR-008` to `FR-016` and `FR-022` are accepted by User Story 3 scenarios 1 through 10.
- `FR-017` is accepted when user-facing project documentation names the saved-setup location for Linux, macOS, and Windows, explains the expected protection approach on each platform, and explains the safe removal procedure that resets setup on the next launch.

### Security, Precision, and Integration Constraints

- **SEC-001**: The Ghostfolio security token MUST be the only user-entered secret required for the sync data workflow in this slice.
- **SEC-002**: The Ghostfolio security token MUST remain only in transient application memory for the active validation attempt, MUST be cleared when the attempt ends or the application exits, and MUST not be written or exposed through user-facing messages, logs, crash text, diagnostics, or persisted artifacts produced by project-owned code. If dependency-generated or wrapped error text would otherwise surface the token, temporary session credential, request body, or raw unprotected payload, project-owned code MUST redact or suppress that content before display or persistence. This slice does not claim control over external tooling that operates outside the application process.
- **SEC-003**: The Ghostfolio security token MUST remain the basis for Ghostfolio communication in this slice, but persisted setup state that must be readable before token entry MUST use local device protection rather than Ghostfolio-token-derived protection.
- **SEC-004**: This feature slice MUST not persist the Ghostfolio security token, Ghostfolio-returned payloads, or any derived sync data locally.
- **SEC-005**: The origin transport-security rule MUST be enforced both when setup data is entered and when persisted setup is reloaded for runtime use: production usage rejects self-hosted `http` server origins and allows only `https`, while `http` is permitted only when the application was started with the same explicit development-mode opt-in that allowed it to be saved. Reloading a remembered `http` origin without that opt-in MUST invalidate remembered setup and return the user to setup.
- **FIN-001**: Financial calculation rules are out of scope for this slice; any numeric values received during validation are used only to confirm that the minimal expected response structure was returned and not to derive balances, gains, losses, or reports.
- **QUAL-001**: Automated validation MUST cover first-run setup gating, setup completion, setup persistence between runs, invalid remembered setup at startup, local device protection of persisted setup state through restrictive permissions where the operating system exposes them and protected app-config placement otherwise, self-hosted origin acceptance rules for development and production modes, remembered `http`-origin invalidation without explicit development mode, sync data selection, Ghostfolio security token-only input, successful communication validation, rejected-token handling, timeout handling, connectivity failure, unsuccessful response handling, incompatible-server handling, contradictory activity-page handling, invalid response payload handling, token non-persistence, token non-exposure in application-produced diagnostics, retry after both failure and success, abandoned in-flight attempt behavior, and confirmation that no data persistence or report flow occurs.
- **QUAL-002**: Automated validation MUST cover startup rendering of the first actionable setup screen from a clean config directory and of the first actionable main menu from a remembered setup directory. In this slice, `first actionable` means the first full-screen view that offers an enabled primary action or required labeled input; passive splash or bootstrap status views do not satisfy this requirement. Validation MUST confirm that these startup screens use only local bootstrap state and require no Ghostfolio network requests or token entry before the user starts sync data.
- **QUAL-003**: Automated validation MUST cover Bubble Tea busy-state progress updates and terminal resize handling while Ghostfolio authentication and activities requests are in flight so the event loop remains responsive during communication validation.
- **QUAL-004**: When truecolor is unavailable, the full-screen TUI MUST still preserve menu-selection state, primary-action emphasis, and failure-state distinction through readable ANSI color choices or, if the terminal is effectively monochrome, through visible non-color cues such as labels, prefixes, or emphasis.
- **INT-001**: The feature depends on a compatible Ghostfolio server that accepts the Ghostfolio security token through this slice's supported authentication and one-page activity-probe contract and returns the required minimal payload shape. A reachable server that responds with contract drift, unsupported authentication behavior, unsupported response format, contradictory activity-page semantics, unreadable minimum field values, or missing supported endpoints is treated as an incompatible server rather than as a generic connectivity failure. The integration MUST validate that compatibility at runtime rather than assume a permanently stable remote contract.

### Key Entities *(include if feature involves data)*

This slice reuses the earlier reporting reference model and includes only the subset needed for setup and communication validation.

- **Saved Setup Record (`AppSetupConfig`)**: The protected local bootstrap configuration on the user's machine that identifies the selected Ghostfolio server and whether setup is complete. In this slice, it contains only six persisted fields: format version, setup-complete flag, selected server mode, normalized server address, development-mode HTTP allowance flag, and last-updated timestamp. A saved setup is valid only when all six fields are present, setup is marked complete, the address can be normalized into a valid allowed form, the address still satisfies the current transport-safety rule, and no Ghostfolio token, temporary session credential, synced data, or local user identity is present.
- **Ghostfolio Session (`GhostfolioSession`)**: The transient authenticated runtime state for one application run. In this slice, it includes the active server address, the in-memory Ghostfolio security token supplied by the user, and any temporary session credential returned by Ghostfolio during the active validation flow only.
- **Sync Validation Attempt (`SyncValidationAttempt`)**: The transient workflow state for one sync execution. In this slice, it covers starting the validation request, receiving the result, validating structural success or failure, classifying any failure into exactly one supported user-visible category, and ending with a transient success or failure outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In automated validation runs from both a clean config directory and a remembered setup directory, 100% of launches reach an actionable setup or main-menu screen before any Ghostfolio network request can occur and without prompting for the Ghostfolio security token at startup.
- **SC-002**: In controlled validation runs with a reachable and compatible Ghostfolio server, 100% of valid communication attempts end with a success message confirming communication is working.
- **SC-003**: In controlled validation runs with a rejected Ghostfolio security token, timeout, connectivity problem, unsuccessful server response, or incompatible server contract, 100% of attempts end with the correct failure category and do not proceed to any later-stage workflow.
- **SC-004**: 100% of successful communication-validation runs end without storing returned Ghostfolio data locally.
- **SC-005**: 100% of user-visible outcomes in this slice are limited to setup completion, sync data selection, and communication-validation messaging, with no report-generation outcome exposed.
- **SC-006**: In controlled validation runs with delayed Ghostfolio responses, 100% of sync-validation attempts continue to render busy-state updates and process terminal resize events until the request finishes.

### Measurement Notes

- For **SC-001**, the `first actionable` screen is the first full-screen view that offers an enabled primary action or a required labeled input the user can use immediately. Passive splash or bootstrap status views do not count.
- For **SC-001** through **SC-006**, `100% of launches` and `100% of attempts` refer to the full automated fixture set for this slice: clean setup, remembered setup, invalid remembered setup, valid token, rejected token, timeout, connectivity failure, unsuccessful response, incompatible server response, contradictory activities page, unreadable timestamp, delayed response, repeated success, repeated failure, abandoned in-flight attempt, and setup-file removal after startup load.
- Controlled validation runs use deterministic local test fixtures or equivalent fixed-response stubs so the audited scope is stable and repeatable.
- **SC-001**, **SC-002**, **SC-003**, **SC-004**, and **SC-006** are verified by automated tests. **SC-005** is verified by automated screen assertions that only setup, sync-data selection, and communication-validation outcomes are visible in this slice.
- A validation attempt counts as successful for **SC-002** only when the visible success result also satisfies **FR-011** and **FR-015**. A failed attempt counts as correct for **SC-003** only when the visible failure result satisfies **FR-012**.
- A responsiveness result counts as correct for **SC-006** only when, during a delayed-response fixture, the interface visibly advances at least one busy-state update after the request starts and still applies at least one terminal-resize event before the request finishes.

## Assumptions

- This slice intentionally narrows a previously broader feature definition into a first step focused only on setup, sync data selection, and communication validation.
- The first release slice needs only enough response validation to confirm communication is functioning, limited to successful authentication, a successful activity-retrieval response, and the minimal payload shape required by the validated Ghostfolio sync contract; domain-level activity normalization, persistence of retrieved sync data, and reporting are deferred to later specs.
- This slice does not require keeping any Ghostfolio-returned data after the communication result is shown. Setup state is remembered between runs and stored using local device protection so the application can determine setup completion before Ghostfolio token entry; token-protected persisted user data remains deferred to later specs.
- The machine-local bootstrap setup file remains user-removable, and deleting it resets the application to the first-run setup flow on the next launch.
- The Ghostfolio security token is the only user-entered secret required to exercise the successful communication path.
- Hosted Ghostfolio and self-hosted Ghostfolio are both in scope for setup, but only one selected server is validated per sync attempt.
- This slice reuses the validated Ghostfolio sync contract from the earlier reporting reference work to define the supported authentication and activity-retrieval boundary.
- Development mode is enabled only by an explicit startup opt-in, is distinct from installed production usage, and is the only mode in which the `FR-003` and `SEC-005` self-hosted `http` exception applies.
- Development mode must not be inferred from remembered setup content, server responses, or any ambient environment assumption that the user did not deliberately enable for that application start.

## Explicitly Deferred In This Slice

- Full-history pagination beyond the one-page communication probe.
- Validation of the full Ghostfolio activity schema beyond the minimum fields needed to prove communication works.
- Domain-level enforcement of activity-type support across the full retrieved history.
- Zero-priced `BUY` or `SELL` business rules.
- Chronological normalization, duplicate removal, or any other sync-data cleanup step.
- Local persistence or cache reuse of synced Ghostfolio payloads.
- Financial calculations, cost-basis selection, gains or losses derivation, or report preparation.
- Report generation, report output formats, and any filing-oriented workflow.
- Multi-user local profiles or token-protected persisted user data.
