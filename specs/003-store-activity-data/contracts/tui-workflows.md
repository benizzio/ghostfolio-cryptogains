# Contract: TUI Workflows

## Scope

This contract defines the user-visible workflow states and keyboard rules for the `Store Activity Data` slice. It keeps the full-screen Bubble Tea interaction model from `specs/002-sync-data-validation/` and extends the `Sync Data` workflow from communication validation into full-history retrieval and protected storage.

## Global UX Rules

- The application launches in the terminal alternate screen and keeps the persistent ASCII-safe project header visible on every full-screen view.
- The TUI keeps the Ghostfolio-inspired visual hierarchy from `002`: clear primary actions, restrained panels, visible help text, and readable failure states.
- The next main workflow steps are always shown as a vertical menu navigable with `Up`, `Down`, and `Enter`.
- Optional side actions remain visible and preferably use modifier-based hotkeys.
- Labeled text inputs keep the focus and paste-safe behavior defined in `002`.
- Ghostfolio token entry is always masked.
- Busy states are asynchronous and must keep the Bubble Tea event loop responsive while auth, pagination, normalization, or protected-write work is in flight.
- Before token entry, the application may read only bootstrap setup. Protected snapshot contents remain inaccessible until the user starts `Sync Data` and supplies a token.
- Reporting, report preview, gains-or-losses output, and cached-activity browsing must not appear as executable workflows in this slice.

## Screen Contract

### Setup Screen

Entry conditions:

- Application launch with no valid bootstrap setup.
- Application launch with bootstrap setup that is now invalid.
- User chooses to edit the selected server.

Visible content:

- setup explanation panel
- primary action menu
- labeled custom-origin input when `Custom Server` is selected
- visible note about `https` requirement outside explicit development mode
- visible explanation when remembered bootstrap setup became invalid

Primary menu items:

- `Use Ghostfolio Cloud`
- `Use Custom Server`
- `Save And Continue`

Rules:

- Setup saves only bootstrap fields from `002`.
- This screen must not prompt for the Ghostfolio token.
- Changing setup does not directly decrypt, delete, or overwrite protected snapshots.
- Leaving setup before save keeps the previous valid bootstrap setup unchanged.

Success transition:

- valid save -> `Main Menu Screen`

### Main Menu Screen

Entry conditions:

- Valid bootstrap setup loaded on startup.
- Setup just completed successfully.
- User returns from a sync result screen.

Visible content:

- selected server summary
- bootstrap status summary
- primary action menu
- optional side-action help

Primary menu items:

- `Sync Data`

Optional side actions:

- `Edit Setup`
- `Quit`

Rules:

- `Sync Data` is the only business workflow in this slice.
- The screen must not offer report generation, report preview, or cached-data browsing.
- If a readable protected snapshot is already active for the current run, the screen may summarize that protected data exists, but it must not expose activity details or years.

Success transition:

- `Sync Data` -> `Sync Data Screen`
- `Edit Setup` -> `Setup Screen`

### Sync Data Screen

Entry conditions:

- The user selected `Sync Data` from the main menu.

Visible content:

- workflow explanation panel stating that the application will authenticate, retrieve activity history, validate it, and store it securely for future use only
- labeled masked token input
- primary action menu
- transient status panel or busy panel

Primary menu items:

- `Start Sync`
- `Back`

Rules:

- The token input is the only secret input in this slice.
- Selecting `Start Sync` with an empty token is blocked with an in-workflow validation message.
- After `Start Sync`, the workflow may:
  - discover selected-server snapshot candidates
  - attempt token unlock against only that candidate set
  - authenticate with Ghostfolio
  - page through the full activity history
  - normalize and validate the full dataset
  - write or refresh the protected snapshot atomically
- Busy-state messaging must remain non-secret and must not display raw payload content.
- If the application exits during an in-flight sync, the attempt is abandoned and must not resume automatically after restart.

Success transition:

- completed sync -> `Sync Result Screen`

Failure transition:

- failed sync -> `Sync Result Screen`

### Server Mismatch Confirmation Screen

Entry conditions:

- A readable protected snapshot is already active in memory for the current run.
- The bootstrap `server_origin` now differs from that snapshot's protected `server_origin`.

Visible content:

- warning panel
- impact explanation
- primary action menu

Primary menu items:

- `Continue And Replace`
- `Cancel`

Rules:

- The warning must state that continuing will replace the current protected data tied to that token and server only after the replacement sync completes successfully.
- `Cancel` leaves the active readable snapshot unchanged and aborts the new sync before retrieval begins.
- `Continue And Replace` returns to the sync busy state and starts the replacement workflow.

Success transition:

- `Continue And Replace` -> `Sync Data Screen` busy state
- `Cancel` -> `Sync Result Screen`

### Sync Result Screen

Entry conditions:

- A `Sync Data` attempt finished.

Visible content:

- success or failure result panel
- explanatory note about what the result does and does not mean
- primary action menu
- optional side-action help

Primary menu items:

- `Sync Again`
- `Back To Main Menu`

Rules:

- On success, the screen must state that activity data was stored securely for future use and that no report-generation, report-preview, or cached-data browsing workflow is available in this slice.
- On failure, the screen must show exactly one final outcome category from the supported set:
  - `rejected token`
  - `timeout`
  - `connectivity problem`
  - `unsuccessful server response`
  - `incompatible server contract`
  - `unsupported activity history`
  - `unsupported stored-data version`
  - `incompatible new sync data`
  - `server replacement cancelled`
- Failure results must explain the next step without exposing the token, JWT, or raw unprotected payloads.
- `Sync Again` starts a new sync attempt without requiring setup to be repeated.
- Result messages are transient and are not persisted for later display after restart.

Success transition:

- `Sync Again` -> `Sync Data Screen`
- `Back To Main Menu` -> `Main Menu Screen`

## Workflow State Summary

```text
Application Launch
├── no valid bootstrap setup -> Setup Screen -> Main Menu Screen
└── valid bootstrap setup -> Main Menu Screen

Main Menu Screen
├── Sync Data -> Sync Data Screen
└── Edit Setup -> Setup Screen

Sync Data Screen
├── active readable snapshot server differs -> Server Mismatch Confirmation Screen
├── sync success -> Sync Result Screen
└── sync failure -> Sync Result Screen

Sync Result Screen
├── Sync Again -> Sync Data Screen
└── Back To Main Menu -> Main Menu Screen
```
