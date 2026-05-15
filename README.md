# ghostfolio-cryptogains

Terminal UI for the current Ghostfolio full-history sync slice. The application keeps bootstrap setup in `setup.json`, retrieves supported Ghostfolio activity history on demand, normalizes and validates it, and stores successful results only in token-locked local snapshots. Report generation and cached-data browsing remain out of scope.

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
make test
make coverage
GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1 go test ./tests/integration -run TestSyncPerformanceFlowLargeHistoryFixture -count=1 -v
```

`make coverage` writes `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` using the maintained coverage gate configuration in `.cov.json`.
The coverage run instruments project-owned packages from `cmd/` and `internal/` so execution driven by contract and integration tests counts toward the repository coverage gate.
The explicit performance command runs the deterministic `SC-006` verification path for a 10,000-activity protected snapshot refresh.

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

Protection notes:

- Unix-like platforms create the config directory with `0700` permissions and the setup file with `0600` permissions where the platform exposes those permission bits.
- Windows uses the current user's application-data directory and does not rely on Unix permission bits.
- Protected snapshots use token-derived encryption and store normalized activity history, not the Ghostfolio token or JWT.
- Eligible synced-data failures can write structured local diagnostic reports. Outside explicit development mode those reports redact financial-value fields.
- The application does not persist the Ghostfolio security token, Ghostfolio JWT, or raw unprotected Ghostfolio payloads in this slice.

## Removing Local Data

Delete the bootstrap setup file to force the next launch back to first-run setup.

- Linux: `rm "$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json"` or `rm ~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `rm "$HOME/Library/Application Support/ghostfolio-cryptogains/setup.json"`
- Windows PowerShell: `Remove-Item "$env:AppData\ghostfolio-cryptogains\setup.json"`

- Delete the `snapshots/` directory under the same application root to remove protected synced activity history.
- Delete the `diagnostics/` directory under the same application root to remove local synced-data diagnostic reports.
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
- the main menu exposes only `Sync Data`
- `Sync Data` prompts for the Ghostfolio security token only when that workflow starts
- sync calls `POST /api/v1/auth/anonymous` and then pages `GET /api/v1/activities?skip=<n>&take=<n>&sortColumn=date&sortDirection=asc` until the full reported history is retrieved
- successful sync normalizes and validates supported `BUY` and `SELL` activity history and stores it as a protected local snapshot for future use
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

Not in scope yet:

- capital-gains calculations
- report generation
- report preview
- cached-data browsing
