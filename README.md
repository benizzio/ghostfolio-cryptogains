# ghostfolio-cryptogains

Terminal UI for the first runnable Ghostfolio integration slice. The current release validates communication with a selected Ghostfolio server but does not sync data to local storage and does not generate reports.

## Running

```bash
make run
```

Use `make run-dev` only when intentionally testing a custom `http://` origin.

Supported runtime flags:

- `--config-dir <path>` overrides the base config directory used for the bootstrap setup file.
- `--dev-mode` allows custom `http://` origins for local development only.
- `--request-timeout <duration>` sets the validation timeout. The default is `30s`.

## Verification

```bash
make test
make coverage
```

`make coverage` writes `dist/coverage/coverage.out` and `dist/coverage/coverage.xml` using the maintained coverage gate configuration in `.cov.json`.
The coverage run instruments project-owned packages from `cmd/` and `internal/` so execution driven by contract and integration tests counts toward the repository coverage gate.

## Local Setup Storage

The current slice persists only bootstrap setup state in a single machine-local JSON file named `setup.json`.

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

Protection notes:

- Unix-like platforms create the config directory with `0700` permissions and the setup file with `0600` permissions where the platform exposes those permission bits.
- Windows uses the current user's application-data directory and does not rely on Unix permission bits.
- The application does not persist the Ghostfolio security token, Ghostfolio JWT, or Ghostfolio activities payload in this slice.

## Removing Local Setup

Delete the bootstrap setup file to force the next launch back to first-run setup.

- Linux: `rm "$XDG_CONFIG_HOME/ghostfolio-cryptogains/setup.json"` or `rm ~/.config/ghostfolio-cryptogains/setup.json`
- macOS: `rm "$HOME/Library/Application Support/ghostfolio-cryptogains/setup.json"`
- Windows PowerShell: `Remove-Item "$env:AppData\ghostfolio-cryptogains\setup.json"`

If the file is removed after startup, the current run keeps its in-memory server selection until the application exits.

## Development Mode

Start the app with `make run-dev` to allow custom `http://` origins during setup.

- Without `--dev-mode`, custom origins must use `https://`.
- The default Ghostfolio Cloud origin remains `https://ghostfol.io`.
- Remembered setup is revalidated on every launch. A remembered `http://` origin becomes invalid when the app is started again without `--dev-mode`, and the user is sent back to setup before any Ghostfolio request runs.

## Current Slice Scope

Current behavior:

- the application opens in a full-screen Bubble Tea interface
- first-run setup lets the user choose Ghostfolio Cloud or a canonical custom origin
- the main menu exposes only `Sync Data`
- `Sync Data` prompts for the Ghostfolio security token only when that workflow starts
- validation calls `POST /api/v1/auth/anonymous` and then `GET /api/v1/activities?skip=0&take=1&sortColumn=date&sortDirection=asc`
- success confirms that communication works but that no data was stored locally
- failure is categorized as `rejected token`, `timeout`, `connectivity problem`, `unsuccessful server response`, or `incompatible server contract`

Not in scope yet:

- persisting synced Ghostfolio data
- capital-gains calculations
- report generation
