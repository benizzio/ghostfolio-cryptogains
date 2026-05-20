# Quickstart: Generate Yearly Gains And Losses Report

## Goal

Validate the slice that:

- exposes `Sync and Reports` from the main menu
- unlocks sync and report actions with one Ghostfolio token entry per active context
- shows last successful sync time beside `Sync Data`
- gates report generation until synced reportable data exists
- generates one yearly Markdown gains-and-losses report from protected synced activity data
- saves the report to the user's Documents folder
- requests OS default-app open
- keeps no in-application report history
- leaves no cleartext report artifacts outside the final saved file

## Prerequisites

- Go 1.26.3 installed
- a terminal that supports alternate-screen applications
- repository dependencies installed through normal Go module resolution
- writable application config or app-data directory
- writable personal Documents folder for manual output verification
- a supported Ghostfolio target or deterministic test stub implementing the existing `003` sync contract
- a valid Ghostfolio security token for manual success-path verification

## Contributor Verification Commands

Run:

```bash
make test
make coverage
GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestReportPerformanceFlowLargeHistoryFixture -count=1 -v
```

Expected automated coverage scope for this slice includes:

- `Sync and Reports` main-menu entry
- masked token unlock for the active context
- token reuse for sync and report actions while the context remains active
- token clearing when leaving the context
- last successful sync timestamp display beside `Sync Data`
- report-generation unavailable state when no synced data or no reportable years exist
- available-year selection from synced data only
- each supported cost-basis method
- method explanation text
- yearly gains and losses counting only selected-year liquidations
- opening and closing basis derived from prior and in-year activity
- assets first acquired after the selected year ignored
- reference-section full-liquidation counts
- reopened asset behavior
- zero-result included assets
- negative loss rendering
- zero-priced holding reductions
- single-activity currency context priority and no tier mixing
- report-wide label `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL`
- Markdown section order and required detail rows
- Documents-folder save
- filename uniqueness within the same second
- OS-open success and failure handling
- no report history after returning to the context
- persisted-artifact leakage checks

## Launch The Application

Run:

```bash
make run
```

Use `make run-dev` only when intentionally testing a self-hosted `http://` origin or explicit development behavior already supported by earlier slices.

Expected result:

- the application takes over the full terminal screen
- startup goes either to setup or to the main menu based only on bootstrap setup
- the app does not ask for the Ghostfolio token at startup

## Bootstrap Setup Path

1. Start with no valid `setup.json`.
2. Launch the application.
3. Choose `Use Ghostfolio Cloud` or `Use Custom Server`.
4. Save setup.

Expected result:

- bootstrap setup is saved locally
- the workflow advances to the main menu
- no Ghostfolio token prompt appears during setup
- no report-generation state is persisted

## Enter Sync and Reports Without Existing Synced Data

1. From the main menu, select `Sync and Reports`.
2. Enter a Ghostfolio security token.
3. Unlock the context.

Expected result:

- the token input is masked
- the context menu shows `Sync Data`
- the context menu shows `Generate Capital Gains Report`
- `Sync Data` states that no synced data is available when no protected cache exists
- `Generate Capital Gains Report` is unavailable with a clear reason
- the token is not asked for again while staying in the context

## Sync Data Inside The Unlocked Context

1. From `Sync and Reports`, choose `Sync Data`.
2. Start sync if an explicit start confirmation is present.

Expected result:

- the workflow reuses the active context token
- the application authenticates, retrieves, normalizes, validates, and writes protected synced data according to the `003` contract
- successful sync returns to `Sync and Reports`
- `Sync Data` now shows the last successful sync date and time
- `Generate Capital Gains Report` becomes available when at least one reportable year exists
- no second token entry is required

## Generate A Markdown Report

1. From `Sync and Reports`, choose `Generate Capital Gains Report`.
2. Select one available year.
3. Select one cost basis method.
4. Review the method explanation.
5. Choose `Generate Report`.

Expected result:

- year choices are limited to years in synced activity data
- methods are exactly FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order
- report generation does not run a new sync
- one Markdown file is created in the user's Documents folder
- the filename contains a local `YYYY-MM-DD_HH-MM-SS` timestamp and does not overwrite existing files
- the application requests the OS to open the file
- the result shows the saved path
- the workflow returns to `Sync and Reports` without asking for the token again

## Automatic Open Failure Path

1. Configure the test opener to fail, or test on a system without a Markdown default-app association.
2. Generate a report.

Expected result:

- the Markdown file remains saved in Documents
- the result shows the saved path
- the result states that automatic opening failed
- the workflow returns to `Sync and Reports`
- the open failure is not treated as a failed save

## Documents Folder Failure Path

1. Run with a controlled home directory where `Documents` is absent or not writable.
2. Attempt report generation.

Expected result:

- report generation fails with an actionable output-location error
- no partial Markdown file remains
- no app-managed cleartext report artifact is written
- the workflow returns to the report result or selection state without clearing the active token context

## Same-Second Filename Collision Path

1. Generate two reports within the same second using a deterministic clock in tests.

Expected result:

- both files are saved
- the second file uses a numeric suffix such as `-2.md`
- neither file overwrites the other
- alphabetical ordering still groups the files by timestamp

## Calculation Fixture Expectations

Use deterministic fixtures that include:

- multi-year activity before, within, and after the selected year
- an asset first acquired after the selected year
- an asset fully liquidated before the selected year and not reopened
- an asset fully liquidated and reopened before or within the selected year
- at least one zero-result included asset
- at least one realized loss
- an explained zero-priced `SELL` holding reduction
- mixed available monetary tiers for single-activity currency context selection
- unreliable or missing source scope for scope-local fallback

Expected result:

- only selected-year priced liquidations contribute gains and losses
- prior activity establishes opening basis
- later activity is ignored
- assets are grouped by stored Ghostfolio asset identity key
- display labels do not affect grouping
- zero-priced holding reductions reduce holdings and basis but produce no gain or loss
- reference-section liquidation counts are correct through selected year end
- report totals match controlled expected values for all five methods

## Generated Markdown Inspection

Open the generated `.md` file.

Expected sections in order:

1. `Gains-And-Losses Summary`
2. `Reference Section`
3. one `Asset Detail: <label>` section for each included asset

Expected content:

- one summary row per included asset
- one `Overall Yearly Net Total` row
- report calculation currency `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL`
- canonical exact-decimal values with no rounding
- losses shown with negative sign
- opening position, in-year rows, liquidation calculations, and closing position in each detail section
- no activity after the selected year

## No Report History Check

1. Generate a report successfully.
2. Return to `Sync and Reports`.
3. Generate another report or leave and re-enter the context.

Expected result:

- the application does not show a generated-report list
- the application does not show a reopen action for prior reports
- setup and protected snapshots do not store report content or generated-report metadata
- only the final Markdown files in Documents remain

## Local File Layout And Removal

Bootstrap setup file remains from earlier slices:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json` or `~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/setup.json`
- Windows: `%AppData%\ghostfolio-cryptogains\setup.json`

Protected snapshot directory remains from `003`:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/snapshots/` or `~/.config/ghostfolio-cryptogains/snapshots/`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/snapshots/`
- Windows: `%AppData%\ghostfolio-cryptogains\snapshots\`

Report output location for this slice:

- Linux: configured XDG Documents directory when present, otherwise `~/Documents/`
- macOS: `~/Documents/`
- Windows: per-user Documents known folder, with documented default `%USERPROFILE%\Documents\`

Removal behavior:

- deleting `setup.json` resets bootstrap setup on next launch
- deleting `snapshots/` removes protected synced activity data
- deleting generated Markdown files from Documents removes report outputs
- none of these deletion paths reveals or recovers the Ghostfolio security token

## Persisted Artifact Inspection

After one successful report and one failed report-generation attempt, inspect the application-data root and Documents folder.

Expected result:

- `setup.json` contains only bootstrap fields
- `snapshots/` contains encrypted snapshot files only
- app-managed storage contains no Markdown report content
- app-managed storage contains no generated-report catalog or history
- Documents contains only successful final report files
- failed output attempts leave no partial Markdown file
- no persisted artifact stores the Ghostfolio security token or JWT

## Large-History Performance Verification

Run:

```bash
GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestReportPerformanceFlowLargeHistoryFixture -count=1 -v
```

Expected result:

- the command uses a deterministic 10,000-activity fixture spanning at least 5 calendar years
- the timed path includes report request validation, basis calculation, Markdown rendering, final save, and opener stub invocation
- at least 95% of measured report runs complete under 2 minutes on supported hardware
