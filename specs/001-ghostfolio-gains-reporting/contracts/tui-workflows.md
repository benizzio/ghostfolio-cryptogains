# Contract: TUI Workflows

## Scope

This contract defines the user-visible workflow states and guardrails for the installed Go TUI application.

## Global UX Rules

- The application is terminal-native and keyboard-driven.
- Ghostfolio tokens are always entered through masked input fields.
- Transient workflow errors are shown only in the active workflow and are not persisted for later display.
- Destructive replacement of encrypted local data always requires an explicit confirmation step.
- Reporting actions remain disabled until setup is complete and at least one successful sync exists for the selected server.

## Screen Contract

### Launch And Unlock Screen

Entry conditions:

- Application start.

Visible actions:

- `Unlock Existing Data`
- `Start First-Time Setup`
- `Quit`

Rules:

- If encrypted snapshot files exist, the user may enter a Ghostfolio token to unlock one of them.
- Unlock failure is generic and must not reveal whether the token was wrong or the snapshot was corrupted.
- If no profile unlocks successfully, the user may continue to first-time setup.

Success transition:

- unlocked profile -> `Sync And Reporting Home`
- first-time setup selection -> `Setup Screen`

### Setup Screen

Entry conditions:

- No unlocked setup exists, or the user chose to update setup after unlocking.

Visible inputs:

- server selection: `Ghostfolio Cloud` or `Custom Server`
- custom origin input when applicable
- insecure HTTP override only when required for local development

Rules:

- The user cannot continue to report generation from this screen.
- Custom origins must be canonicalized and validated before the next step.
- Changing the selected origin on an existing profile does not immediately delete protected data.

Success transition:

- valid setup -> `Sync Screen`

### Sync Screen

Entry conditions:

- Setup is present.

Visible actions:

- `Start Sync`
- `Back`

Runtime fields:

- masked token input
- progress indicator
- non-secret workflow status text

Rules:

- The application prompts for a Ghostfolio token every time a sync starts.
- A sync session always runs for the currently selected unlocked local profile.
- The token is exchanged for a session JWT and then used only for the active sync workflow.
- The application does not create or retain a new local profile when auth or retrieval fails.
- On successful retrieval, the history is normalized and validated before any encrypted write occurs.

Success transition:

- successful sync -> `Sync And Reporting Home`

Failure transition:

- sync failure -> stay on `Sync Screen` with an in-workflow error message only

### Server Mismatch Confirmation Screen

Entry conditions:

- The unlocked profile contains protected data for one server origin and the current setup points to another origin.

Visible actions:

- `Continue And Replace`
- `Cancel`

Required message content:

- The message must state that continuing will clean the current protected data tied to the user and Ghostfolio security token and replace it with data from the newly selected server.

Rules:

- `Cancel` leaves the existing encrypted snapshot unchanged and aborts the sync.
- `Continue And Replace` starts a replacement sync, but the old snapshot is kept until the new sync succeeds completely.

Success transition:

- continue -> `Sync Screen`
- cancel -> `Sync And Reporting Home`

### Sync And Reporting Home

Entry conditions:

- An unlocked profile exists.

Visible sections:

- sync status summary
- selected Ghostfolio server summary
- available report years
- available cost basis methods

Rules:

- Report generation controls are disabled when no successful sync exists.
- The selected year list contains only years present in cached activity history.
- Changing the cost basis method updates an informational message that explains the matching rule and fallback behavior in jurisdiction-neutral language.

Success transition:

- generate report -> `Report Generation Screen`

### Report Generation Screen

Entry conditions:

- A year and cost basis method are selected.

Visible actions:

- `Generate PDF`
- `Back`

Rules:

- Calculation uses only activity up to the end of the selected year while still using prior history for basis.
- Activity after the selected year is ignored for the current report run.
- Failure messages must not expose the token or raw unprotected cached data.

Success transition:

- successful render -> `Report Result Screen`
- failure -> remain on `Report Generation Screen`

### Report Result Screen

Entry conditions:

- PDF generation completed successfully.

Visible information:

- generated file path
- selected year
- selected cost basis method

Rules:

- The gains and losses summary is the first section in the PDF.
- Detailed sections follow and are grouped by asset.
- Assets liquidated before the selected year and not reopened are listed only in the reference section.

Success transition:

- `Back` -> `Sync And Reporting Home`

## Workflow State Summary

```text
Launch And Unlock
├── Start First-Time Setup -> Setup Screen -> Sync Screen
└── Unlock Existing Data -> Sync And Reporting Home

Sync Screen
├── Server mismatch detected -> Server Mismatch Confirmation Screen
├── Sync success -> Sync And Reporting Home
└── Sync failure -> Sync Screen

Sync And Reporting Home
└── Generate report -> Report Generation Screen -> Report Result Screen
```
