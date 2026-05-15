# Quickstart: Store Activity Data

## Goal

Validate the slice that:

- keeps bootstrap setup from `002`
- retrieves the full supported Ghostfolio activity history
- normalizes and validates that history for future reporting use
- stores successful results only inside token-locked protected snapshots
- still exposes no reporting or cached-data browsing workflow

## Prerequisites

- Go 1.26.3 installed
- a terminal that supports alternate-screen applications
- one reachable Ghostfolio target:
  - the hosted default at `https://ghostfol.io`, or
  - a self-hosted Ghostfolio origin, or
  - a local test stub that implements `contracts/ghostfolio-sync.md`
- a valid Ghostfolio security token for manual success-path verification
- a writable per-user config or app-data directory

## Contributor Verification Commands

Run:

```bash
make test
make coverage
```

Expected automated coverage scope for this slice includes:

- full-history pagination
- empty-history success
- first protected snapshot creation only after full success
- same-token refresh replacing only after full success
- wrong-token denial
- different-valid-token isolated snapshot creation
- unsupported activity rejection
- duplicate removal and deterministic ordering
- server-mismatch confirmation and cancel path
- unsupported stored-data version failure
- incompatible newly synced data discarded while existing readable snapshot remains active
- confirmation that no reporting workflow is available

## Launch The Application

Run:

```bash
make run
```

Use `make run-dev` only when intentionally testing a self-hosted `http://` origin.

Expected result:

- the application takes over the full terminal screen
- the persistent application header is visible
- startup goes either to setup or to the main menu based only on bootstrap setup
- the app does not ask for the Ghostfolio token at startup

## First-Run Bootstrap Setup Path

1. Start with no existing `setup.json` and no snapshot files.
2. Launch the application.
3. Choose `Use Ghostfolio Cloud` or `Use Custom Server`.
4. If `Use Custom Server` is selected, enter a canonical origin.
5. Select `Save And Continue`.

Expected result:

- bootstrap setup is saved locally
- the workflow advances to the main menu
- no Ghostfolio token prompt appears during setup
- no protected snapshot is created yet

## First Successful Sync Path

1. From the main menu, select `Sync Data`.
2. Enter a valid Ghostfolio security token.
3. Select `Start Sync`.

Expected result:

- the UI enters a busy state
- auth succeeds through `POST /api/v1/auth/anonymous`
- the application pages through `GET /api/v1/activities` until the full reported history is retrieved
- the history is normalized and validated before any protected write occurs
- a protected snapshot file is created only after the full sync succeeds
- the result screen states that activity data was stored for future use and that no reporting workflow is available yet

## Empty-History Success Path

1. Run `Sync Data` against a fixture or server response where `count == 0` and `activities` is empty.

Expected result:

- sync still succeeds
- a protected snapshot is created or refreshed
- the stored cache contains zero activities and zero available years
- the result screen still states that reporting is not part of this slice

## Same-Token Refresh Path

1. Complete one successful sync.
2. Run `Sync Data` again with the same server and same token.

Expected result:

- the existing selected-server snapshot can be unlocked
- the previous snapshot remains active until the new full sync succeeds
- the old snapshot is replaced only after the new protected write succeeds atomically

## Different Valid Token Path

1. Complete one successful sync for a selected server.
2. Run `Sync Data` again for the same selected server with a different valid Ghostfolio token.

Expected result:

- the new token does not unlock the existing snapshot
- Ghostfolio auth succeeds
- a new isolated protected snapshot is created only after the full sync succeeds
- the previously existing protected snapshot remains unchanged

## Invalid Token Path

1. Run `Sync Data` with a token rejected by Ghostfolio.

Expected result:

- the final result category is `rejected token`
- no protected snapshot is created or modified
- bootstrap setup remains unchanged

## Unsupported History Path

Validate at least these failure cases with deterministic fixtures:

- an activity type other than `BUY` or `SELL`
- a normalized `BUY` with `unit_price = 0`
- a zero-priced `SELL` without an explanatory comment
- same-asset same-timestamp events that still cannot be ordered deterministically
- gaps or contradictions that make the normalized history non-defensible

Expected result for each case:

- sync fails before any protected snapshot write
- the final result category is `unsupported activity history`
- any previously readable snapshot remains unchanged

## Server Replacement Path

1. Complete a successful sync.
2. While the readable snapshot is still active in the current run, edit setup and change the selected Ghostfolio server.
3. Start `Sync Data` again.

Expected result:

- the application shows the server replacement confirmation before retrieval starts
- declining leaves the active readable snapshot unchanged and ends with `server replacement cancelled`
- accepting starts the replacement sync, but the old snapshot remains active until the new protected write succeeds

## Unsupported Stored-Data Version Path

1. Prepare a protected snapshot fixture with an unsupported envelope version or unsupported payload stored-data version.
2. Start `Sync Data` for the matching selected server and provide the token.

Expected result:

- the application reports `unsupported stored-data version`
- the snapshot is not partially loaded or overwritten
- no protected data is exposed

## Incompatible New Sync Data Path

1. Start from a readable protected snapshot that the current application version supports.
2. Run a new sync where retrieval succeeds but the newly returned data cannot be normalized or persisted within the current stored-data model.

Expected result:

- the final result category is `incompatible new sync data`
- the newly retrieved data is discarded
- the previously readable snapshot remains active and unchanged

## Local File Layout And Removal

Bootstrap setup file location:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json` or `~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/setup.json`
- Windows: `%AppData%\ghostfolio-cryptogains\setup.json`

Protected snapshot directory location:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/snapshots/` or `~/.config/ghostfolio-cryptogains/snapshots/`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/snapshots/`
- Windows: `%AppData%\ghostfolio-cryptogains\snapshots\`

Removal behavior:

- deleting `setup.json` resets bootstrap setup on the next launch
- deleting the `snapshots/` directory removes protected synced activity data
- neither deletion path reveals or recovers the Ghostfolio security token

## Negative Checks

After both successful and failed sync attempts, verify:

- the Ghostfolio security token was not written to disk
- the Ghostfolio JWT was not written to disk
- raw unprotected Ghostfolio payloads were not written to disk
- only bootstrap setup and protected snapshot files remain persisted
- no report-generation, report-preview, or cached-data browsing action is exposed in the UI
