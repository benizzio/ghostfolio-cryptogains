# Contract: TUI Workflows

## Scope

This contract defines the user-visible workflow states and keyboard interaction rules for the full-screen terminal application in the sync-validation slice.

## Global UX Rules

- The application always launches into the terminal alternate screen and occupies the full visible terminal.
- The TUI visual identity follows Ghostfolio's general product style: clean sans-serif typography, teal primary emphasis, blue secondary emphasis, red warning or failure states, and restrained neutral backgrounds and panels.
- When the terminal supports truecolor, the UI should approximate Ghostfolio's live palette with teal near `#36cfcc`, blue near `#3686cf`, and red near `#dc3545`.
- When truecolor is unavailable, the UI must fall back to readable ANSI styling that still distinguishes:
  - the currently selected menu item
  - the primary action region
  - failure or warning states
- If terminal capabilities are effectively monochrome, the UI must add non-color cues such as explicit labels, prefixes, or emphasis so those distinctions remain visible.
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
- Panels and menus should favor subtle separators, compact spacing, and high-contrast headings over decorative borders so the workflow reads like a focused Ghostfolio product surface rather than a generic terminal form.

## Screen Contract

### Setup Screen

Entry conditions:

- Application launch with no valid remembered setup.
- Application launch with remembered setup that is now invalid because the stored origin is malformed, cannot be canonicalized, or no longer satisfies the current transport-security rule.
- User chooses to edit the selected server after setup already exists.

Visible content:

- setup explanation panel
- primary action menu
- labeled custom-origin input field when `Custom Server` is selected
- visible note describing that production-like origins require `https`
- visible invalid-setup explanation when startup rejected remembered setup
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
- Leaving setup before save must not overwrite an earlier valid remembered setup.
- If startup rejected remembered setup, the screen must explain that the saved server selection is no longer valid and that setup must be completed again before sync validation can run.

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
- The screen counts as the first actionable main menu only when `Sync Data` is visible as an enabled primary action.

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
- If the user leaves the application during an active request, the attempt is abandoned and must not resume on the next launch.

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

- On success, the screen must state that communication works, that no Ghostfolio data was stored locally, and that reporting is not available in this slice.
- On failure, the screen must explain exactly one failure category without exposing secrets or raw unprotected payloads.
- The supported user-visible failure categories are `rejected token`, `timeout`, `connectivity problem`, `unsuccessful server response`, and `incompatible server contract`.
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
