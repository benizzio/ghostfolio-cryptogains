# Quickstart: Sync Data Validation

## Goal

Validate the first runnable application slice that covers:

- first-run setup
- remembered Ghostfolio server selection
- full-screen TUI workflow selection
- Ghostfolio communication validation
- explicit confirmation that no data was stored and no reporting flow is available yet

## Prerequisites

- Go 1.26.3 installed
- a terminal that supports alternate-screen applications
- one reachable Ghostfolio target:
  - the hosted default at `https://ghostfol.io`, or
  - a self-hosted Ghostfolio origin, or
  - a local test stub that implements the contract in `contracts/ghostfolio-sync-validation.md`
- an explicit development-mode startup flag only when testing a self-hosted `http` origin

## Launch The Application

Run:

```bash
make run
```

Use `make run ARGS="--dev-mode"` only when intentionally testing a custom `http://` origin.

Verification commands:

```bash
make test
make coverage
```

Expected result:

- the application takes over the full terminal screen immediately
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
- `Ctrl+E` returns to setup to replace the remembered server selection

## Invalid Remembered Setup Path

Validate at least one invalid remembered setup case:

1. Save a custom `http://` origin while the app is running with `--dev-mode`, then relaunch without `--dev-mode`.
2. Or manually corrupt the saved `setup.json` file so the stored origin or mode is no longer valid.

Expected result:

- the app starts in setup instead of the main menu
- the screen explains that the saved server selection is no longer valid
- no Ghostfolio network request is made before setup is completed again

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

Validate each failure category separately:

1. `rejected token`: auth returns `403 Forbidden`
2. `timeout`: auth or activities does not finish within the configured timeout
3. `connectivity problem`: the selected origin cannot be reached
4. `unsuccessful server response`: auth returns a non-2xx response other than `403`, or activities returns `401`, `403`, or another non-2xx response other than `400`
5. `incompatible server contract`: auth or activities returns unsupported content type, malformed JSON, missing required fields, contradictory `count` and `activities` values, more than one returned activity for the probe, or an unreadable required activity timestamp

Expected result for each case:

- a failure result screen appears
- exactly one supported failure category is shown
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

## Bootstrap Setup File

The bootstrap setup file is stored as `ghostfolio-cryptogains/setup.json` under the current user's config directory.

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json` or `~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/setup.json`
- Windows: `%AppData%\ghostfolio-cryptogains\setup.json`

Delete that file to force the next launch back to first-run setup.

## Negative Check: No Persistence Beyond Setup

After both successful and failed sync attempts, verify:

- no Ghostfolio activity payload was written to disk
- no Ghostfolio token or JWT was written to disk
- no report-generation screen or action is exposed
- only the bootstrap setup file remains persisted
- the config directory contains only `setup.json`
