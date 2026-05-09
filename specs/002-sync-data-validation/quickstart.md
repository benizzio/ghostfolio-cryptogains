# Quickstart: Sync Data Validation

## Goal

Validate the first runnable application slice that covers:

- first-run setup
- remembered Ghostfolio server selection
- full-screen TUI workflow selection
- Ghostfolio communication validation
- explicit confirmation that no data was stored and no reporting flow is available yet

## Prerequisites

- Go 1.26.2 installed
- a terminal that supports alternate-screen applications
- one reachable Ghostfolio target:
  - the hosted default at `https://ghostfol.io`, or
  - a self-hosted Ghostfolio origin, or
  - a local test stub that implements the contract in `contracts/ghostfolio-sync-validation.md`
- an explicit development-mode startup flag only when testing a self-hosted `http` origin

## Launch The Application

Run:

```bash
go run ./cmd/ghostfolio-cryptogains
```

Expected result:

- the application opens in the full terminal screen
- a clearly delimited setup or main-menu screen is shown
- the footer displays the currently available hotkeys

## First-Run Setup Path

1. Start with no remembered setup file.
2. Launch the application.
3. Use the arrow-key menu to choose one of these primary actions:
   - `Use Ghostfolio Cloud`
   - `Use Custom Server`
4. If `Use Custom Server` is selected, move focus to the labeled origin input and enter a canonical origin.
5. Activate `Save And Continue`.

Expected result:

- invalid origins are rejected in-place
- production-like `http` origins are rejected unless the app is running in explicit development mode
- the remembered setup is saved locally
- the app advances to the main menu without prompting for the Ghostfolio token

## Remembered Setup Path

1. Complete setup once.
2. Exit the application.
3. Launch the application again.

Expected result:

- the app skips setup
- the remembered Ghostfolio origin is shown on the main menu
- the user does not need to enter the Ghostfolio token at startup

## Sync Validation Success Path

1. From the main menu, select `Sync Data`.
2. Enter a valid Ghostfolio security token in the labeled masked input field.
3. Select `Validate Communication`.

Expected result:

- the UI switches to a busy state during the request
- auth succeeds through `POST /api/v1/auth/anonymous`
- the app requests `GET /api/v1/activities?skip=0&take=1&sortColumn=date&sortDirection=asc`
- a success result screen appears when the response shape matches the contract
- the result explicitly states that communication works, that no data was stored, and that reporting is not available yet

## Sync Validation Failure Paths

Validate each of these cases separately:

1. Invalid Ghostfolio token
2. Unreachable server or timeout
3. Non-2xx auth response
4. Non-2xx activities response
5. Malformed JSON response
6. Missing `authToken`
7. Missing `activities` or invalid `count`
8. `count > 0` with no first activity item
9. First activity item missing `id`, `date`, or `type`

Expected result for each case:

- a failure result screen appears
- the message explains that communication validation did not succeed
- the message does not expose the token, JWT, or raw unprotected payload
- `Validate Again` is available without repeating setup

## Local Development HTTP Path

1. Start the application in explicit development mode.
2. Enter a self-hosted `http` origin during setup.
3. Save setup and run `Sync Data`.

Expected result:

- the origin is accepted only in development mode
- the rest of the validation workflow is unchanged

## Negative Check: No Persistence Beyond Setup

After both successful and failed sync attempts, verify:

- no Ghostfolio activity payload was written to disk
- no Ghostfolio token or JWT was written to disk
- no report-generation screen or action is exposed
- only the bootstrap setup file remains persisted
