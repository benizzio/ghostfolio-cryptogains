# Contract: Sync And Reports TUI Workflows

## Scope

This contract defines user-visible workflow behavior for the `Generate Yearly Gains And Losses Report` slice. It replaces the prior direct `Sync Data` main-menu business entry with a token-unlocked `Sync and Reports` context that exposes both syncing and report generation.

## Global UX Rules

- The application remains terminal-native, full-screen, and keyboard-driven.
- The persistent application header remains visible on full-screen views.
- Ghostfolio tokens are always entered through masked inputs.
- Protected sync metadata, report years, and last-sync timestamps are visible only after token unlock.
- Busy states are asynchronous and keep the Bubble Tea event loop responsive.
- User-visible errors must be actionable and must not expose tokens, JWTs, raw protected payloads, or unprotected financial details outside the intentionally saved final report.
- Report content is not previewed as cleartext in the TUI before final save.
- The application keeps no report history, generated-report catalog, or reopen list.

## Screen Contract

### Main Menu Screen

Entry conditions:

- Valid bootstrap setup loaded on startup.
- Setup just completed successfully.
- User returns from the unlocked `Sync and Reports` context.

Visible content:

- selected server summary
- bootstrap setup status summary
- primary action menu
- optional side-action help

Primary menu items:

- `Sync and Reports`

Optional side actions:

- `Edit Setup`
- `Quit`

Rules:

- The main menu must not show protected activity details, report years, last-sync timestamp, generated report paths, or report history.
- Selecting `Sync and Reports` starts token entry for the active context.
- Editing setup does not directly decrypt, delete, or overwrite protected snapshots.

Success transitions:

- `Sync and Reports` -> `Sync and Reports Unlock Screen`
- `Edit Setup` -> existing setup screen

### Sync and Reports Unlock Screen

Entry conditions:

- User selected `Sync and Reports` from the main menu.

Visible content:

- explanation that the token unlocks sync and report actions for this active context
- masked Ghostfolio token input
- primary action menu

Primary menu items:

- `Unlock`
- `Back`

Rules:

- Empty token input is blocked with an in-workflow validation message.
- The token is not persisted.
- Snapshot discovery and unlock use only selected-server candidates.
- If no selected-server snapshot unlocks, the context can still open so the user can run `Sync Data`, but report generation remains unavailable until sync data exists.
- Auth or unlock failures must not reveal token material.

Success transitions:

- successful context unlock or new-context token acceptance -> `Sync and Reports Menu Screen`
- `Back` -> `Main Menu Screen`

### Sync and Reports Menu Screen

Entry conditions:

- A `Sync and Reports` context is active and has a runtime token.
- A sync attempt completed and returned to the active context.
- A report-generation attempt completed and returned to the active context.

Visible content:

- selected server summary
- protected data readiness summary
- `Sync Data` action with last successful sync local date and time when synced data exists
- `Sync Data` action with `no synced data available` when no cache exists
- `Generate Capital Gains Report` action
- unavailable reason beside report generation when reporting is blocked
- optional side-action help

Primary menu items:

- `Sync Data`
- `Generate Capital Gains Report`

Optional side actions:

- `Back To Main Menu`
- `Quit`

Rules:

- Both primary actions are visible in every active context state.
- `Sync Data` is always available while the context is active.
- `Generate Capital Gains Report` is unavailable until a protected activity cache exists and at least one reportable year is present.
- The report action cannot be entered when unavailable.
- Returning to the main menu clears the token and in-memory report content.
- No generated-report path from a prior dismissed result is shown as history.

Success transitions:

- `Sync Data` -> `Sync Data Screen`
- available `Generate Capital Gains Report` -> `Report Selection Screen`
- `Back To Main Menu` -> `Main Menu Screen`

### Sync Data Screen

Entry conditions:

- User selected `Sync Data` inside an active `Sync and Reports` context.

Visible content:

- explanation that the existing context token will be used for sync
- busy status while auth, retrieval, normalization, validation, and protected write run
- non-secret result or failure category after the attempt finishes if an intermediate result view is used

Primary menu items before start, if confirmation is required:

- `Start Sync`
- `Back`

Rules:

- The workflow must not ask for the token again while the active context remains unlocked.
- Server mismatch confirmation still appears before retrieval when the active readable snapshot server differs from bootstrap setup.
- Successful sync refreshes the protected cache visible to the context and updates the last successful sync timestamp.
- Failed sync leaves any previously readable protected cache unchanged.
- After completion, the user returns to `Sync and Reports Menu Screen` without another token prompt.

Success transitions:

- completed sync -> `Sync and Reports Menu Screen`
- server mismatch detected -> `Server Replacement Confirmation Screen`

### Server Replacement Confirmation Screen

Entry conditions:

- A readable protected snapshot is active in the context.
- Bootstrap `server_origin` differs from that snapshot's protected `server_origin`.
- User started `Sync Data`.

Visible content:

- warning panel
- explanation that continuing replaces protected data only after replacement sync succeeds
- primary action menu

Primary menu items:

- `Continue And Replace`
- `Cancel`

Rules:

- `Cancel` leaves the active readable snapshot unchanged and returns to the unlocked context.
- `Continue And Replace` starts sync using the active context token.
- The old snapshot remains active until the replacement sync succeeds completely.

Success transitions:

- `Continue And Replace` -> `Sync Data Screen` busy state
- `Cancel` -> `Sync and Reports Menu Screen`

### Report Selection Screen

Entry conditions:

- User selected available `Generate Capital Gains Report`.
- A protected activity cache exists with at least one reportable year.

Visible content:

- report-generation explanation
- selected year list containing only years present in the protected activity cache
- selected cost basis method list containing exactly the supported methods
- plain-language explanation for the highlighted or selected method
- primary action menu

Supported cost basis methods:

- `FIFO`
- `LIFO`
- `HIFO`
- `Average Cost Basis`
- `Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order`

Primary menu items:

- `Generate Report`
- `Back`

Rules:

- Year selection must be constrained to `available_report_years`.
- Method selection must be constrained to the supported method list.
- Changing the highlighted method updates the explanation before generation.
- `Generate Report` starts asynchronous calculation, render, save, and OS-open work.
- `Back` returns to the unlocked context without clearing the token.

Success transitions:

- `Generate Report` -> `Report Generation Busy Screen`
- `Back` -> `Sync and Reports Menu Screen`

### Report Generation Busy Screen

Entry conditions:

- User confirmed year and method.

Visible content:

- non-secret busy message
- selected year
- selected cost basis method

Rules:

- The UI must not show cleartext report content as a preview.
- Calculation uses the currently unlocked protected cache and does not run a new sync.
- Activity after the selected year is ignored.
- On calculation or save failure, the workflow reports an actionable non-secret error and removes any partial cleartext output created by the attempt.
- On save success and automatic-open failure, the workflow treats the save as successful and reports the open warning.

Success transitions:

- success or failure -> `Report Result Screen`

### Report Result Screen

Entry conditions:

- Report generation attempt completed.

Visible content on success:

- saved Markdown file path
- selected year
- selected cost basis method
- automatic-open status, including a warning if open failed
- primary action menu

Visible content on failure:

- actionable non-secret failure message
- selected year
- selected cost basis method when available
- primary action menu

Primary menu items:

- `Back To Sync and Reports`
- `Generate Another Report` when protected reportable data still exists

Rules:

- The saved path is shown only as a transient result message.
- Dismissing the result must not create report history or a reopen list.
- Returning to `Sync and Reports` does not ask for the token again.
- Leaving the context after this screen clears the token and any in-memory report content.

Success transitions:

- `Back To Sync and Reports` -> `Sync and Reports Menu Screen`
- `Generate Another Report` -> `Report Selection Screen`

## Workflow State Summary

```text
Application Launch
├── no valid bootstrap setup -> Setup Screen -> Main Menu Screen
└── valid bootstrap setup -> Main Menu Screen

Main Menu Screen
├── Sync and Reports -> Sync and Reports Unlock Screen
└── Edit Setup -> Setup Screen

Sync and Reports Unlock Screen
├── Unlock -> Sync and Reports Menu Screen
└── Back -> Main Menu Screen

Sync and Reports Menu Screen
├── Sync Data -> Sync Data Screen -> Sync and Reports Menu Screen
├── Generate Capital Gains Report -> Report Selection Screen
└── Back To Main Menu -> Main Menu Screen

Report Selection Screen
├── Generate Report -> Report Generation Busy Screen -> Report Result Screen
└── Back -> Sync and Reports Menu Screen

Report Result Screen
├── Back To Sync and Reports -> Sync and Reports Menu Screen
└── Generate Another Report -> Report Selection Screen
```

## Supported User-Visible Report Failure Categories

- `no synced data available`
- `no reportable years available`
- `unsupported stored-data version`
- `unsupported report calculation`
- `documents folder unavailable`
- `report file write failed`
- `automatic open failed after save`

Each failed report-generation attempt shows one primary actionable reason. The automatic-open failure category is a warning after a successful save, not a failed save.
