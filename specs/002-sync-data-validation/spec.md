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
6. **Given** setup is incomplete, **When** the user attempts to execute a feature, **Then** the application blocks the action and returns the user to finish setup.

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

When the user runs sync data, the application authenticates using the selected Ghostfolio server's supported Ghostfolio authentication flow, requests activity data, validates that the request succeeded and that the response includes the minimal payload shape required by the validated Ghostfolio sync contract, and informs the user whether communication is working.

**Why this priority**: This is the sole business outcome of the requested slice and confirms that the application can communicate correctly before later specs add storage or reporting.

**Independent Test**: With a reachable Ghostfolio server and a valid Ghostfolio security token, the user can run sync data and receive a success message. With an invalid Ghostfolio security token, unreachable server, or an invalid retrieval result, the user receives a failure message and no later-stage behavior occurs.

**Acceptance Scenarios**:

1. **Given** setup is complete and the selected Ghostfolio server is reachable, **When** the user starts sync data and provides a valid Ghostfolio security token, **Then** the application authenticates with Ghostfolio, requests activity data, validates that the response is successful and includes the minimal payload shape required by the validated Ghostfolio sync contract, and shows a success message confirming communication is working.
2. **Given** the selected Ghostfolio server rejects the provided Ghostfolio security token, **When** the user starts sync data, **Then** the application shows a failure message explaining that communication validation did not succeed.
3. **Given** the selected Ghostfolio server is unreachable or does not respond successfully, **When** the user starts sync data, **Then** the application shows a failure message explaining that communication validation did not succeed.
4. **Given** the request completes but the returned result is missing the minimal payload shape required by the validated Ghostfolio sync contract, **When** the application validates the result, **Then** the application shows a failure message explaining that communication validation did not succeed.
5. **Given** sync data completes successfully, **When** the application shows the workflow result, **Then** the application confirms that communication is working, explicitly states that data has not been stored or prepared for reporting, does not persist retrieved data, and does not start any report-generation flow.
6. **Given** a previous sync data attempt failed, **When** the user starts sync data again, **Then** the application allows a new validation attempt without requiring setup to be repeated.

---

### Edge Cases

- The user provides a malformed self-hosted server origin, a non-`https` origin outside development mode, or any other origin not allowed by the setup rules.
- The selected Ghostfolio server responds but returns an unexpected payload or omits the minimal payload shape required by the validated Ghostfolio sync contract.
- The selected Ghostfolio server accepts authentication and returns no data entries; the application treats this as successful communication if the retrieval result is otherwise valid.
- The network request times out after the user starts sync data.
- The application encounters a crash, trace, or diagnostic event during the workflow; the Ghostfolio security token must not appear in any application-produced logs, dumps, or persisted artifacts.
- The user completes setup successfully, but exits before selecting a feature.
- The user expects data to be stored after a successful communication check; the application must clearly state that persistence is not part of this release.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, precision,
testing, dependency, and external integration impacts when the feature touches
those areas.

### Functional Requirements

- **FR-001**: The system MUST provide a runnable base application that guides the user through first-run setup before any business workflow can be executed.
- **FR-002**: The system MUST allow the user during setup to choose either the hosted Ghostfolio service or a self-hosted Ghostfolio server origin.
- **FR-003**: The system MUST accept a self-hosted server origin only when it uses `https`, except that `http` origins MAY be accepted when the application is explicitly running in development mode.
- **FR-004**: The system MUST prevent feature execution until setup has been completed.
- **FR-005**: The system MUST persist the completed setup state and selected Ghostfolio server between application runs using protected local setup storage on the user's machine that remains readable without prompting for the Ghostfolio security token at startup.
- **FR-006**: The system MUST present sync data as the executable feature in this release after setup is complete.
- **FR-007**: The system MUST require only the Ghostfolio security token from the user to validate communication with the selected Ghostfolio server when the user chooses sync data.
- **FR-008**: The system MUST use the user-supplied Ghostfolio security token for that active validation attempt to authenticate with the selected Ghostfolio server and request activity data through the validated Ghostfolio sync contract.
- **FR-009**: The system MUST validate both the request result and the returned payload before declaring communication successful.
- **FR-010**: The system MUST treat communication validation as successful only when the selected Ghostfolio server accepts the request, returns a successful activity-retrieval response, and includes the minimal payload shape required by the validated Ghostfolio sync contract, even if the returned activity list is empty.
- **FR-011**: The system MUST show a success message to the user when communication validation succeeds.
- **FR-012**: The system MUST show a user-facing failure message when communication validation fails because of a rejected Ghostfolio security token, connectivity problems, unsuccessful responses, or an invalid retrieval result.
- **FR-013**: The system MUST end the workflow after showing the communication-validation result without persisting the retrieved data.
- **FR-014**: The system MUST not generate reports, prepare report output, or present report-generation as part of this release.
- **FR-015**: The system MUST inform the user in the sync data workflow that successful communication validation does not yet mean data has been stored or prepared for reporting.
- **FR-016**: The system MUST allow the user to re-run sync data after a failed validation attempt.

### Security, Precision, and Integration Constraints

- **SEC-001**: The Ghostfolio security token MUST be the only user-entered secret required for the sync data workflow in this slice.
- **SEC-002**: The Ghostfolio security token MUST remain only in transient application memory for the active validation attempt, MUST be cleared when the attempt ends or the application exits, and MUST not be written or exposed through user-facing messages, logs, dumps, traces, diagnostics, or persisted artifacts.
- **SEC-003**: The Ghostfolio security token MUST remain the basis for Ghostfolio communication in this slice, but persisted setup state that must be readable before token entry MUST use local device protection rather than Ghostfolio-token-derived protection.
- **SEC-004**: This feature slice MUST not persist the Ghostfolio security token, Ghostfolio-returned payloads, or any derived sync data locally.
- **SEC-005**: Production usage MUST reject self-hosted `http` server origins and allow only `https` origins; `http` is permitted only when the application is explicitly running in development mode.
- **FIN-001**: Financial calculation rules are out of scope for this slice; any numeric values received during validation are used only to confirm that the minimal expected response structure was returned and not to derive balances, gains, losses, or reports.
- **QUAL-001**: Automated validation MUST cover first-run setup gating, setup completion, setup persistence between runs, local device protection of persisted setup state, self-hosted origin acceptance rules for development and production modes, sync data selection, Ghostfolio security token-only input, successful communication validation, rejected-token handling, connectivity failure, unsuccessful response handling, invalid response payload handling, token non-persistence, token non-exposure in application-produced diagnostics, retry after failure, and confirmation that no data persistence or report flow occurs.
- **INT-001**: The feature depends on a Ghostfolio server that can accept a Ghostfolio security token through the validated Ghostfolio authentication contract and return activity data that satisfies the minimal payload shape required by the validated Ghostfolio sync contract; the integration must validate compatibility at runtime rather than assume a permanently stable remote contract.

### Key Entities *(include if feature involves data)*

This slice reuses the validated reference model from `specs/001-ghostfolio-gains-reporting/` and includes only the subset needed for setup and communication validation.

- **SetupProfile**: The protected local configuration on the user's machine that identifies the selected Ghostfolio server and whether setup is complete. In this slice, it is limited to the server-selection and setup-completion concerns needed before feature execution and remains readable before Ghostfolio token entry.
- **GhostfolioSession**: The ephemeral authenticated runtime state for one application run. In this slice, it includes the active server origin, the in-memory Ghostfolio security token supplied by the user, and any temporary session credential returned by Ghostfolio during the active validation flow only.
- **SyncAttempt**: The ephemeral workflow state for one sync execution. In this slice, it covers starting the validation request, receiving the result, validating structural success or failure, and ending with a user-visible success or failure outcome.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95% of first-time users can complete setup and reach the feature-selection step in under 3 minutes using only in-application prompts.
- **SC-002**: In controlled validation runs with a reachable and compatible Ghostfolio server, 100% of valid communication attempts end with a success message confirming communication is working.
- **SC-003**: In controlled validation runs with a rejected Ghostfolio security token, unreachable server, unsuccessful responses, or invalid response payloads, 100% of attempts end with a failure message and do not proceed to any later-stage workflow.
- **SC-004**: 100% of successful communication-validation runs end without storing returned Ghostfolio data locally.
- **SC-005**: 100% of user-visible outcomes in this slice are limited to setup completion, sync data selection, and communication-validation messaging, with no report-generation outcome exposed.

## Assumptions

- This slice intentionally narrows a previously broader feature definition into a first step focused only on setup, sync data selection, and communication validation.
- The first release slice needs only enough response validation to confirm communication is functioning, limited to successful authentication, a successful activity-retrieval response, and the minimal payload shape required by the validated Ghostfolio sync contract; domain-level activity normalization, persistence of retrieved sync data, and reporting are deferred to later specs.
- This slice does not require keeping any Ghostfolio-returned data after the communication result is shown. Setup state is remembered between runs and stored using local device protection so the application can determine setup completion before Ghostfolio token entry; token-protected persisted user data remains deferred to later specs.
- The Ghostfolio security token is the only user-entered secret required to exercise the successful communication path.
- Hosted Ghostfolio and self-hosted Ghostfolio are both in scope for setup, but only one selected server is validated per sync attempt.
- This slice reuses the validated Ghostfolio sync contract from the reference feature to define the supported authentication and activity-retrieval boundary.
- Development mode is an explicit application mode distinct from installed production usage, and only that mode may permit self-hosted `http` origins.
