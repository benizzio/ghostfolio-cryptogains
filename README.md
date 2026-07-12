# ghostfolio-cryptogains

Terminal UI for syncing supported Ghostfolio activity history into token-locked local snapshots and generating yearly Markdown capital gains and losses reports from the unlocked synced dataset. The application keeps bootstrap setup in `setup.json`, stores synced activity history only in protected snapshots, writes successful report output only to the current user's Documents folder, and keeps no in-application report history.

## Running

```bash
make run
```

Use `make run-dev` only when intentionally testing a custom `http://` origin.

Supported runtime flags:

- `--config-dir <path>` overrides the base config directory used for `setup.json`, protected snapshots, and diagnostic reports.
- `--dev-mode` allows custom `http://` origins for local development only and auto-generates eligible synced-data diagnostic reports.
- `--request-timeout <duration>` sets the full sync timeout. The default is `30s`.

## Verification

```bash
# Isolated package behavior
make test-unit
# Externally visible workflows and storage
make test-contract
# Cross-package offline workflows
make test-integration
# Synthetic financial oracle datasets
make test-empirical
# Repository-local tool behavior
make test-tools
# Aggregate deterministic offline verification
make test

# Isolated package diagnostic profile
make coverage-unit
# Workflow and storage diagnostic profile
make coverage-contract
# Cross-package offline diagnostic profile
make coverage-integration
# Financial oracle dataset diagnostic profile
make coverage-empirical
# Repository-local tool diagnostic profile
make coverage-tools
# Canonical production coverage enforcement
make coverage

# Tagged 10,000-activity resource scenarios
make test-performance

# Opt-in live-network integration checks
make test-external-integration
# Live-network integration diagnostic profile
make coverage-external-integration

# Changed Go and module source scanning
make quality QUALITY_BASE_REF=origin/main
```

`make test` is the canonical deterministic offline aggregate. It runs unit, contract, deterministic integration, empirical, and tool suites once. It excludes live external integration and resource-sensitive performance checks.
`make coverage` is the canonical deterministic aggregate. It writes fresh `dist/coverage/coverage.out` and `dist/coverage/coverage.xml`, instruments project-owned packages from `cmd/` and `internal/` so black-box contract and integration execution counts, enforces the repository-wide 100% gate, and separately executes tool coverage without expanding the production denominator.
`coverage-unit`, `coverage-contract`, `coverage-integration`, `coverage-empirical`, `coverage-tools`, and `coverage-external-integration` write partial diagnostic profiles under `dist/coverage/`. They do not run the repository percentage gate.
`tests/performance` contains timed 10,000-activity scenarios. Its build tag and maintained command keep it isolated from `make test`, `make coverage`, and ordinary `go test ./...`; run `make test-performance` only when the dedicated resources are available. Performance scenarios provide timing, responsiveness, and resource evidence. They do not produce a separate coverage profile because canonical deterministic coverage already proves the complete production denominator independently. Pull requests run the performance command on a fresh, independent runner.
`make test-external-integration` and `make coverage-external-integration` are opt-in live-network checks and are excluded from pull-request CI.
Pull-request checks map to `test` (`make test`), `coverage` (`make coverage`), `test-performance` (`make test-performance`), and the separate `quality` check.
`make quality` runs the changed-source quality gate for `*.go`, `go.mod`, and `go.sum` changes using `golangci-lint`, `govulncheck`, and `gitleaks`. It must pass for every feature, including the explicit skip path when no source inputs changed.

## Local Storage

Bootstrap setup stays in a single machine-local JSON file named `setup.json`.

Persisted fields:

- `schema_version`
- `setup_complete`
- `server_mode`
- `server_origin`
- `allow_dev_http`
- `updated_at`

Expected file locations:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json` or `~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/setup.json`
- Windows: `%AppData%\ghostfolio-cryptogains\setup.json`

Protected snapshot directory:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/snapshots/` or `~/.config/ghostfolio-cryptogains/snapshots/`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/snapshots/`
- Windows: `%AppData%\ghostfolio-cryptogains\snapshots\`

Diagnostic report directory:

- Linux: `$XDG_CONFIG_HOME/ghostfolio-cryptogains/diagnostics/` or `~/.config/ghostfolio-cryptogains/diagnostics/`
- macOS: `~/Library/Application Support/ghostfolio-cryptogains/diagnostics/`
- Windows: `%AppData%\ghostfolio-cryptogains\diagnostics\`

Report output directory:

- Linux: configured XDG Documents directory when available, otherwise `~/Documents/`
- macOS: `~/Documents/`
- Windows: `%USERPROFILE%\Documents\`

Protection notes:

- Unix-like platforms create the config directory with `0700` permissions and the setup file with `0600` permissions where the platform exposes those permission bits.
- Windows uses the current user's application-data directory and does not rely on Unix permission bits.
- Protected snapshots use token-derived encryption and store normalized activity history, not the Ghostfolio token or JWT.
- Report generation reads only the unlocked protected snapshot. It does not persist report content, report paths, or report history back into `setup.json`, snapshots, or diagnostics.
- Successful report output is one cleartext Markdown file in the user's Documents folder. The application keeps report content in memory until that final save succeeds.
- Eligible synced-data failures can write structured local diagnostic reports. Outside explicit development mode those reports redact financial-value fields.
- The application does not persist the Ghostfolio security token, Ghostfolio JWT, or raw unprotected Ghostfolio payloads in this slice.

## Removing Local Data

Delete the bootstrap setup file to force the next launch back to first-run setup.

- Linux: `rm "$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json"` or `rm ~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `rm "$HOME/Library/Application Support/ghostfolio-cryptogains/setup.json"`
- Windows PowerShell: `Remove-Item "$env:AppData\ghostfolio-cryptogains\setup.json"`

- Delete the `snapshots/` directory under the same application root to remove protected synced activity history.
- Delete the `diagnostics/` directory under the same application root to remove local synced-data diagnostic reports.
- Delete generated `ghostfolio-capital-gains-*.md` files from the user's Documents folder to remove cleartext report output.
- If `setup.json` is removed after startup, the current run keeps its in-memory server selection until the application exits.

## Development Mode

Start the app with `make run-dev` to allow custom `http://` origins during setup.

- Without `--dev-mode`, custom origins must use `https://`.
- The default Ghostfolio Cloud origin remains `https://ghostfol.io`.
- Remembered setup is revalidated on every launch. A remembered `http://` origin becomes invalid when the app is started again without `--dev-mode`, and the user is sent back to setup before any Ghostfolio request runs.
- Eligible synced-data failures write diagnostic reports automatically only in explicit development mode. Production-like runs require an explicit user choice.

## Current Slice Scope

Current behavior:

- the application opens in a full-screen Bubble Tea interface
- first-run setup lets the user choose Ghostfolio Cloud or a canonical custom origin
- the main menu exposes `Sync and Reports`
- entering `Sync and Reports` prompts once for the Ghostfolio security token and unlocks the active sync-and-report context
- the unlocked context always shows `Sync Data` and `Generate Capital Gains Report`
- sync calls `POST /api/v1/auth/anonymous` and then pages `GET /api/v1/activities?skip=<n>&take=<n>&sortColumn=date&sortDirection=asc` until the full reported history is retrieved
- successful sync normalizes and validates supported `BUY` and `SELL` activity history and stores it as a protected local snapshot for future use
- unlocked synced data shows last successful sync metadata and available report years without forcing a new sync
- report generation uses the unlocked protected snapshot as input, not a fresh Ghostfolio API call
- report generation supports FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order
- successful report generation writes one timestamped Markdown file to Documents, requests one OS default-app open, and shows the saved path with user file-removal guidance
- automatic-open failure leaves the saved report file in place and reports the warning without treating the save as failed
- leaving the result screen or the unlocked context clears transient in-memory report state so the application keeps no report history or reopen list
- same-token refresh replaces the existing selected-server snapshot only after the new protected write succeeds atomically
- a different valid token creates a separate isolated protected snapshot for the same server
- an active readable snapshot can trigger server-replacement confirmation before a new sync starts
- eligible synced-data failures can write local diagnostic reports and disclose the written path

Supported failure categories:

- `rejected token`
- `timeout`
- `connectivity problem`
- `unsuccessful server response`
- `incompatible server contract`
- `unsupported activity history`
- `unsupported stored-data version`
- `incompatible new sync data`
- `server replacement cancelled`
- `no synced data available`
- `no reportable years available`
- `unsupported report calculation`
- `documents folder unavailable`
- `report file write failed`
- `automatic open failed after save`

Not in scope yet:

- report preview before save
- report history or reopen catalog
- cached-data browsing beyond the unlocked readiness summary
