# Contract: TUI Workflows

## Scope

This contract defines the user-visible workflow states and keyboard interaction rules for the full-screen terminal application in the sync-validation slice.

## Global UX Rules

- The application always launches into the terminal alternate screen and occupies the full visible terminal.
- Every screen must clearly delimit these regions:
  - header with title and one-line workflow explanation
  - main content region with the current primary menu or input form
  - transient status region for non-secret progress or result messaging
  - footer or help region with the currently available hotkeys
- The next main workflow steps are always shown as a vertical menu navigable with `Up`, `Down`, and `Enter`.
- Optional side steps may use hotkeys only when those hotkeys are visible on screen.
- Optional side actions should prefer modifier-based hotkeys such as `Ctrl+` combinations.
- Labeled input fields must use persistent visible labels and must not rely on placeholder text as the only description.
- When a text input is focused, plain-character hotkeys are disabled so typing never triggers application actions.
- Ghostfolio token input is always masked.
- Transient failure or success messages are shown only in the active workflow and are not persisted for later display.

## Screen Contract

### Setup Screen

Entry conditions:

- Application launch with no valid remembered setup.
- User chooses to edit the selected server after setup already exists.

Visible content:

- setup explanation panel
- primary action menu
- labeled custom-origin input field when `Custom Server` is selected
- visible note describing that production-like origins require `https`
- visible hotkeys for optional actions such as cancel or quit

Primary menu items:

- `Use Ghostfolio Cloud`
- `Use Custom Server`
- `Save And Continue`

Rules:

- `Save And Continue` is disabled until the selected origin is valid.
- `Use Ghostfolio Cloud` selects `https://ghostfol.io` immediately.
- `Use Custom Server` moves focus to the labeled origin input.
- The custom-origin input must not interfere with hotkeys while it is focused.
- Setup completion persists only the bootstrap setup contract from this slice.
- The screen must not prompt for a Ghostfolio token.

Success transition:

- valid saved setup -> `Main Menu Screen`

### Main Menu Screen

Entry conditions:

- Remembered setup loaded successfully on startup.
- Setup just completed successfully.
- User returns from the sync result workflow.

Visible content:

- selected server summary
- setup status summary
- primary action menu
- visible hotkeys for optional side actions

Primary menu items:

- `Sync Data`

Optional side actions:

- `Edit Setup`
- `Quit`

Rules:

- `Sync Data` is the only business workflow exposed in this release.
- Persistence and report-generation workflows are not shown as executable options.
- Optional side actions may be hotkey-only, but the help region must describe them clearly.

Success transition:

- `Sync Data` -> `Sync Validation Screen`
- `Edit Setup` -> `Setup Screen`

### Sync Validation Screen

Entry conditions:

- The user selected `Sync Data` from the main menu.

Visible content:

- workflow explanation panel that states communication is being validated only
- labeled masked input field for the Ghostfolio security token
- primary action menu
- transient status or busy panel
- visible hotkeys for optional side actions

Primary menu items:

- `Validate Communication`
- `Back`

Rules:

- The token input is the only secret input in this slice.
- The token field has explicit focus behavior and a persistent label.
- While the token field is focused, typing must not trigger optional hotkeys.
- Selecting `Validate Communication` with an empty token is blocked with an in-workflow validation message.
- During the active request, the main content switches to a busy state with non-secret progress text.
- The workflow must not persist the token, JWT, or returned activity payload.

Success transition:

- validation success -> `Validation Result Screen`

Failure transition:

- validation failure -> `Validation Result Screen`

### Validation Result Screen

Entry conditions:

- A sync validation attempt finished.

Visible content:

- success or failure result panel
- explanatory note about what the result does and does not mean
- primary action menu
- visible hotkeys for optional side actions

Primary menu items:

- `Validate Again`
- `Back To Main Menu`

Rules:

- On success, the screen must state that communication works and that data was not stored or prepared for reporting.
- On failure, the screen must explain the failure category without exposing secrets or raw unprotected payloads.
- `Validate Again` starts a new attempt without requiring setup to be repeated.
- The result message is transient and must not be shown again after restart unless the workflow fails again.

Success transition:

- `Validate Again` -> `Sync Validation Screen`
- `Back To Main Menu` -> `Main Menu Screen`

## Workflow State Summary

```text
Application Launch
├── no remembered setup -> Setup Screen -> Main Menu Screen
└── remembered setup -> Main Menu Screen

Main Menu Screen
├── Sync Data -> Sync Validation Screen -> Validation Result Screen
└── Edit Setup -> Setup Screen

Validation Result Screen
├── Validate Again -> Sync Validation Screen
└── Back To Main Menu -> Main Menu Screen
```
