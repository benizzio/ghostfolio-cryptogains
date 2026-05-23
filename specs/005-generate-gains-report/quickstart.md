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

Expected result:

- `make test` passes across the maintained contract, integration, and unit suites
- `make coverage` regenerates `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` and passes the repository coverage gate
- the explicit performance command runs the deterministic `SC-007` verification path for one 10,000-activity yearly report generation and completes within the 2-minute threshold

Expected implemented automated coverage scope for this slice includes:

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
- single-activity currency context priority, skipping tiers without explicit currency before completeness validation, multiplication-based same-tier gross-value derivation before division-based fallback, and no tier mixing
- deterministic histories that require repeating-decimal internal division or allocation and still succeed under the shared 16-decimal internal calculation precision without extra report-boundary rounding
- explicit shared report calculation currency `NOT APPLICABLE` for this slice
- empty-main-section report rendering with yearly total `0`
- incomplete activity monetary-context failure after year and method selection
- same-calendar-date reopening and selected-year boundary classification
- scope-local fallback activation, carry-forward until zero, and same-scope reset after reacquisition
- Markdown section order and required detail rows
- Documents-folder save
- filename uniqueness within the same second
- OS-open success and failure handling
- production-mode report-failure diagnostics prompt and explicit-development-mode automatic diagnostics generation
- diagnostics-path disclosure for eligible failures
- original persisted `ActivityRecord` inclusion for activity-specific report failures
- explicit `null` rendering in report-failure and synced-data diagnostics artifacts
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
- the application requests the OS to open the file exactly once after save
- the result shows the saved path
- the saved path is sufficient for the user to delete the cleartext report later from Documents
- the workflow returns to `Sync and Reports` without asking for the token again

## Manual Verification Scenarios

Verify at least these scenarios when checking the implemented workflow manually:

- unlock `Sync and Reports` with no readable snapshot and confirm report generation stays visible but unavailable
- run `Sync Data` inside the unlocked context and confirm last successful sync metadata appears without another token prompt
- generate one report successfully and confirm the saved path points into Documents
- generate another report from the same unlocked context and confirm no previous report path is retained as history
- trigger an automatic-open failure and confirm the saved file remains in Documents with manual deletion guidance
- trigger a Documents-directory failure and confirm no partial Markdown file remains
- trigger an eligible pre-save report-generation failure and confirm the result offers `Generate Diagnostic Report` outside explicit development mode or generates diagnostics automatically in explicit development mode
- inspect one generated diagnostics artifact and confirm it uses the original persisted `ActivityRecord` with explicit `null` values for absent source fields instead of derived report data, and that wrapped failures preserve actionable `failure_detail` plus ordered `failure_cause_chain` entries without leaking secret-bearing nested causes
- use a fixture or server history that yields a reportable year with no main-section assets and confirm the empty-state report still saves
- use a fixture with mixed selected activity currencies and confirm the report still renders `NOT APPLICABLE` for report calculation currency columns
- use a priced `BUY` fixture where order financial values exist but `order_currency` is absent, the asset-profile tier has explicit currency and fee, and gross value must be derived from quantity and unit price; confirm generation skips the currencyless order tier and uses the asset-profile currency instead of failing early
- use a production-shaped explained zero-priced `SELL` row that preserves explicit zero `unit_price`, `gross_value`, and `fee_amount`, and confirm the report keeps those zeros distinct from missing values without treating the row as a priced liquidation

## Reportable Year With No Main-Section Assets

1. Use a deterministic fixture where `available_report_years` includes one year whose in-year activity contains acquisitions or explained zero-priced holding reductions but, after applying inclusion and exclusion rules, no asset qualifies for the main report sections.
2. Generate the report for that year.

Expected result:

- report generation still succeeds
- `Gains-And-Losses Summary` renders a clear empty-state sentence
- `Overall Yearly Net Total` is `0`
- the reference section still follows its own inclusion rules
- no `Asset Detail: <label>` sections are rendered

## Incomplete Monetary Context Failure After Selection

1. Use a deterministic fixture where one priced `BUY` or priced `SELL` inside the selected year still lacks a complete explicit-currency `order`, `asset`, or `base` monetary context for the required gross value and fee pair even after skipping currencyless tiers and allowing same-tier derivation under the shared 16-decimal internal calculation precision where division is required.
2. Select that year and any method.
3. Start report generation.

Expected result:

- generation fails before any file is saved
- generation fails only after every explicit-currency tier for the offending activity is unusable
- the error is actionable and identifies only non-secret references such as asset label and source ID
- the unlocked `Sync and Reports` context remains active
- no partial Markdown file remains
- outside explicit development mode, the failure result offers `Generate Diagnostic Report` before any diagnostics artifact is written
- when diagnostics generation is requested, the workflow discloses the local `.diagnostic.json` path outside Documents
- if the failure is tied to one activity, the diagnostics artifact includes the original persisted `ActivityRecord` with explicit `null` values for absent source fields and without substituting selected calculation inputs or rendered report rows
- if diagnostics are generated from a wrapped failure, the artifact preserves the actionable `failure_detail` plus an outer-to-inner `failure_cause_chain` whose secret-bearing nested causes are redacted or omitted

## Currencyless Higher-Priority Tier Path

1. Use a deterministic fixture where a priced `BUY` has order financial fields but `order_currency = null`, while the asset-profile tier has an explicit currency, fee, and unit price that can derive gross value from `quantity * unit_price`.
2. Select that year and any method.
3. Start report generation.

Expected result:

- generation succeeds or proceeds past activity-input validation using the asset-profile tier
- the order tier is skipped before completeness validation because it lacks an explicit currency code
- the row or liquidation detail that shows `Activity Currency` reflects the asset-profile currency rather than the skipped order tier
- gross value is derived by same-tier multiplication before any division-based fallback is considered
- no values from the order tier or base tier are mixed into the selected activity input

## Rounded Internal Division Path

1. Use a deterministic fixture where same-tier derivation, average-cost updates, partial-lot basis allocation, or proportional proceeds allocation requires a repeating decimal.
2. Select that year and any method.
3. Start report generation.

Expected result:

- generation succeeds
- the internal calculation path uses the shared 16-decimal precision with round-half-up rounding where division or proportional allocation is required
- rendered report values reuse those internal results without an added report-boundary rounding step

## Explicit-Development Automatic Diagnostics Path

1. Start the application in explicit development mode.
2. Trigger an eligible report-generation failure before the final Markdown file is saved.

Expected result:

- the failure result does not wait for a manual diagnostics confirmation step
- the workflow generates a local `.diagnostic.json` artifact automatically
- the result discloses the generated diagnostics path
- the generated artifact still excludes token and JWT material

## Wrapped Failure Cause-Chain Inspection

1. Trigger an eligible report-generation failure whose implementation wraps a lower-level calculation, rendering, output-preparation, or diagnostics-generation error.
2. Generate the diagnostics artifact.
3. Inspect the serialized `failure_detail` and `failure_cause_chain` fields.

Expected result:

- `failure_detail` remains the actionable outer report-failure summary
- `failure_cause_chain[0]` matches or restates the same outer report failure preserved by `failure_detail`
- later `failure_cause_chain` entries continue in deterministic outer-to-inner order toward the deepest eligible non-secret wrapped cause
- nested secret-bearing or production-disallowed financial-value causes are redacted or omitted instead of being written verbatim

## Mixed Selected-Currency

1. Use a deterministic fixture where priced activities that would contribute to rendered cross-activity monetary outputs resolve to more than one selected activity currency code.
2. Select that year and any method.
3. Start report generation.

Expected result:

- generation is successful
- the generated report keeps per-row `Activity Currency` values from the selected activity context for priced rows
- the generated report contains `NOT APPLICABLE` in report calculation currency columns
- no currency conversion or exchange-rate lookup occurs during report generation

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
- the failure may still generate a separate `.diagnostic.json` artifact outside Documents under the application-owned diagnostics directory
- when diagnostics are generated, the result discloses the diagnostics path instead of implying the Markdown output path succeeded
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
- activity exactly at source-calendar year boundaries across differing timestamp offsets
- same-asset same-source-calendar-date `BUY` and `SELL` rows that rely on synced deterministic ordering
- an asset first acquired after the selected year
- an asset fully liquidated before the selected year and not reopened
- an asset fully liquidated and reopened before or within the selected year
- a reportable year that yields no main-section assets
- at least one zero-result included asset
- at least one realized loss
- a production-shaped explained zero-priced `SELL` holding reduction that preserves explicit zero `unit_price`, `gross_value`, and `fee_amount`
- a priced activity with explicit fee `0`
- a priced `BUY` where order financial fields exist but `order_currency` is absent and the asset-profile or base tier can still supply or exactly derive same-tier values
- a priced activity with incomplete monetary context
- mixed available monetary tiers for single-activity currency context selection
- unreliable or missing source scope for scope-local fallback
- scope-local exact matching that falls back mid-scope and later resets after same-scope reacquisition

Expected result:

- only selected-year priced liquidations contribute gains and losses
- source calendar year with preserved timestamp offset decides year membership
- synced deterministic same-asset order decides same-date reopening behavior
- prior activity establishes opening basis
- later activity is ignored
- assets are grouped by stored Ghostfolio asset identity key
- display labels do not affect grouping
- zero-priced holding reductions reduce holdings and basis but produce no gain or loss
- preserved explicit zero-valued `unit_price`, `gross_value`, and `fee_amount` on zero-priced holding reductions remain distinct from missing values during calculation and detail-row preparation
- tiers without explicit currency are skipped before completeness validation
- same-tier gross-value derivation by multiplication is preferred before division-based unit-price derivation
- explicit fee `0` is accepted, while incomplete priced-activity monetary context fails the attempt
- incomplete priced-activity monetary context fails only after all explicit-currency tiers are unusable
- reference-section liquidation counts are correct through selected year end
- scope-local fallback remains active only for the affected open scope until that scope reaches zero
- later reacquisition in the same scope re-evaluates exact matching, while other scopes keep independent state
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
- report calculation currency equal to the shared explicit currency code for the report, or `NOT APPLICABLE` for this slice
- canonical exact-decimal values with no rounding
- losses shown with negative sign
- opening position, in-year rows, liquidation calculations, and closing position in each detail section
- liquidation tables show both `Activity Currency` and `Calculation Currency`
- `Net Liquidation Proceeds` stays in the liquidation row's activity currency, while `Allocated Basis` and `Gain Or Loss` use the shared explicit report calculation currency
- when an explained zero-priced holding-reduction row preserves explicit zero `gross_value` or `fee_amount`, the `In-Year Activity` table renders those cells as `0` rather than blank, while `Activity Currency` remains blank because no selected context exists for that row
- no activity after the selected year

Expected output layout details:

- header block with `Year`, `Cost Basis Method`, `Generated At`, and `Report Calculation Currency`
- `Gains-And-Losses Summary` first
- `Reference Section` second
- each `Asset Detail: <label>` section after the reference section
- each detail section ordered as `Opening Position`, `In-Year Activity`, optional `Liquidation Calculations`, then `Closing Position`

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

Diagnostics artifact location for eligible failed sync or report attempts:

- Linux: application-owned diagnostics directory under `$XDG_CONFIG_HOME/ghostfolio-cryptogains/` or `~/.config/ghostfolio-cryptogains/`
- macOS: application-owned diagnostics directory under `~/Library/Application Support/ghostfolio-cryptogains/`
- Windows: application-owned diagnostics directory under `%AppData%\ghostfolio-cryptogains\`

Removal behavior:

- deleting `setup.json` resets bootstrap setup on next launch
- deleting `snapshots/` removes protected synced activity data
- deleting generated Markdown files from Documents removes report outputs; this is the only user-managed cleartext report removal path
- none of these deletion paths reveals or recovers the Ghostfolio security token

## Persisted Artifact Inspection

After one successful report and one failed report-generation attempt, inspect the application-data root and Documents folder.

Expected result:

- `setup.json` contains only bootstrap fields
- `snapshots/` contains encrypted snapshot files only
- app-managed storage contains no Markdown report content
- app-managed storage contains no generated-report catalog or history
- app-managed storage may contain `.diagnostic.json` files only for eligible failed sync or report attempts
- diagnostics artifacts generated from wrapped failures preserve actionable `failure_detail` and ordered `failure_cause_chain` entries, with nested secret-bearing causes redacted or omitted
- any diagnostics artifact remains outside Documents and excludes token and JWT material
- Documents contains only successful final report files
- failed output attempts leave no partial Markdown file
- no persisted artifact stores the Ghostfolio security token or JWT

## Performance Verification

Run:

```bash
GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestReportPerformanceFlowLargeHistoryFixture -count=1 -v
```

Expected result:

- the run exercises request validation, calculation, Markdown rendering, final save, and opener invocation against the deterministic large-history fixture
- the logged runtime stays under the `SC-007` threshold of 2 minutes
- verified on 2026-05-21 with `SC-007 verification completed in 6.222988008s for 10000 activities across 6 calendar years`
