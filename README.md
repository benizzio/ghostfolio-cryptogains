# ghostfolio-cryptogains

`ghostfolio-cryptogains` is a terminal application that syncs supported activity
history from Ghostfolio and generates yearly capital gains and losses reports.
Synced activity is stored in token-locked local snapshots. Reports are generated
as Markdown or PDF files in the current user's Documents directory.

> The generated reports are reference material. They do not implement the legally
> required tax rules of any country and are not tax-return output.

## Main Features

- Full-screen terminal workflow for Ghostfolio Cloud and custom Ghostfolio origins.
- Complete paginated sync of supported `BUY` and `SELL` activity history.
- Deterministic normalization, duplicate removal, and validation before data is
  accepted for reporting.
- Token-derived encrypted snapshots for later use without storing the Ghostfolio
  security token or JWT.
- Exact-decimal report calculation with FIFO, LIFO, HIFO, Average Cost Basis, and
  a scope-local hybrid cost-basis method.
- USD and EUR reports with official ECB or Federal Reserve exchange-rate evidence
  when currency conversion is required.
- Timestamped Markdown main-plus-annex bundles or combined searchable PDF reports.
- Local diagnostic reports for eligible sync and report failures.
- No in-application report history, reopen catalog, remote report storage, or
  automatic re-ingestion of exported reports.

## Requirements

- Go 1.26.5.
- An interactive terminal.
- A Ghostfolio Cloud account or a reachable compatible self-hosted Ghostfolio
  server.
- A Ghostfolio security token.
- A pre-existing, writable Documents directory for report exports.
- Network access for initial dependency installation, Ghostfolio sync, and any
  required ECB or Federal Reserve currency conversion.

Linux, macOS, and Windows are target platforms. Maintained CI verification runs on
Ubuntu. On Linux, `xdg-open` is optional and is used only to open a saved report in
the default application.

## Installation

The project does not currently publish tagged releases, prebuilt binaries, or
package-manager distributions. Install the current source with Go:

```bash
go install github.com/benizzio/ghostfolio-cryptogains/cmd/ghostfolio-cryptogains@latest
```

Go installs the executable in `GOBIN`, or in Go's default binary directory when
`GOBIN` is unset. Add that directory to `PATH` to invoke
`ghostfolio-cryptogains` by name.

To build from a checkout instead:

```bash
git clone https://github.com/benizzio/ghostfolio-cryptogains.git
cd ghostfolio-cryptogains
go build -o ghostfolio-cryptogains ./cmd/ghostfolio-cryptogains
./ghostfolio-cryptogains
```

On Windows, use `ghostfolio-cryptogains.exe` as the output and executable name.

## Startup

Start an installed executable:

```bash
ghostfolio-cryptogains
```

Run directly from a source checkout:

```bash
make run
```

Pass runtime flags through the Make target with `ARGS`:

```bash
make run ARGS='--config-dir /path/to/config-base --request-timeout 45s'
```

Use `make run-dev` only when intentionally testing a custom `http://` origin:

```bash
make run-dev ARGS='--config-dir /path/to/config-base'
```

HTTP does not provide transport encryption, and development mode does not limit
custom HTTP origins to the local machine. Do not use development mode with
production credentials or across an untrusted network; the security token, JWT,
account context, and activity data could be exposed in transit.

### Runtime Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `--config-dir <path>` | Operating-system user config directory | Overrides the base config directory. The application still appends `ghostfolio-cryptogains/`. |
| `--dev-mode` | `false` | Allows custom `http://` origins for the current process and automatically writes eligible diagnostics. |
| `--request-timeout <duration>` | `30s` | Sets the Ghostfolio client and sync-attempt timeout. The value must be a positive Go duration. |
| `--window-width <int>` | `100` | Sets the initial test-friendly terminal window width. |
| `--window-height <int>` | `32` | Sets the initial test-friendly terminal window height. |

## Using The Application

### 1. Configure Ghostfolio

On first launch, choose Ghostfolio Cloud or enter a custom server origin.

- Ghostfolio Cloud uses `https://ghostfol.io`.
- A custom origin must contain only an absolute scheme, host, and optional port.
- Custom origins require HTTPS unless the application is running with
  `--dev-mode`.
- Setup saves the selected server, but it does not authenticate or store a
  Ghostfolio security token.
- An HTTP origin remembered in development mode is invalid on a later launch
  without `--dev-mode`; the application returns to setup before making a
  Ghostfolio request.

Use `Ctrl+E` from the main menu to edit remembered setup.

### 2. Unlock Sync And Reports

Select `Sync and Reports` and enter the Ghostfolio security token. The token is
masked in the terminal and retained only in the active application context. It is
used to unlock a matching local snapshot or authenticate a new context against
Ghostfolio.

An existing protected snapshot can be unlocked without contacting Ghostfolio.
Leaving the Sync and Reports context clears its transient token and report state.
The same token is required to recover that snapshot in a later process.

### 3. Sync Activity

Select `Sync Data` to:

1. Authenticate against Ghostfolio.
2. Retrieve the user's base-currency context.
3. Retrieve the complete paginated activity history.
4. Normalize, order, deduplicate, and validate supported activity.
5. Atomically write a protected local snapshot.

Only `BUY` and `SELL` activity is supported. Validation rejects unsupported or
ambiguous history, including non-positive quantities, invalid monetary values,
zero-priced buys, unexplained zero-priced sells, and activity that would produce
negative holdings.

A successful refresh with the same token replaces its selected-server snapshot
only after the new protected write succeeds. Another valid token creates an
isolated snapshot.

### 4. Generate A Report

Select `Generate Capital Gains Report`, then choose:

- a year available in the unlocked snapshot;
- FIFO, LIFO, HIFO, Average Cost Basis, or the scope-local hybrid method;
- USD or EUR as the report currency; and
- Markdown or PDF as the output format.

The calculation replays supported history through the selected year. Cross-currency
activity may require live ECB access for EUR reports or Federal Reserve access for
USD reports. Each provider supports a fixed set of source currencies and searches
up to 30 days back for an available observation. Unsupported currencies or
unavailable observations stop report generation before export. Exchange-rate
evidence is cached only for the current process.

Markdown output contains one main report and one separate Annex 1 file. PDF output
contains the main report and Annex 1 in one landscape A4 file. The report includes
summary, per-asset, rate-source, reference, and detailed activity audit data.

### 5. Manage The Export

Reports are saved to the resolved Documents directory:

| Platform | Report directory |
| --- | --- |
| Linux | Configured XDG Documents directory, otherwise `~/Documents/` |
| macOS | `~/Documents/` |
| Windows | `%USERPROFILE%\Documents\` |

The Documents directory must already exist. The output location is not currently
configurable, and `--config-dir` does not change it.

Report names include the year, cost-basis method, and timestamp. Existing files
are not overwritten; a numeric suffix is added on collision. The application asks
the operating system to open the main Markdown file or combined PDF after saving.
An open failure is reported as a warning and does not remove the saved files.

All exported Markdown and PDF files contain cleartext financial data. The result
screen lists every saved path. Delete every listed file when the export is no
longer needed.

### Generate A Diagnostic

For an eligible failure in a normal run, select `Generate Diagnostic Report` from
the failure result or Sync and Reports context. Development mode writes eligible
diagnostics automatically. The action is available only for failure categories
that provide diagnostic context.

### Keyboard Controls

| Key | Action |
| --- | --- |
| `Up` / `Down` | Move through menu choices. |
| `Enter` | Select the current choice. |
| `Tab` | Change input focus where multiple fields are present. |
| `Esc` | Cancel setup editing. |
| `Ctrl+E` | Edit setup from the main menu. |
| `Page Up` / `Page Down` | Scroll a report result. |
| `Ctrl+C` | Quit. |

## Local Data And Security

The default application root is:

| Platform | Application root |
| --- | --- |
| Linux | `$XDG_CONFIG_HOME/ghostfolio-cryptogains/` or `~/.config/ghostfolio-cryptogains/` |
| macOS | `~/Library/Application Support/ghostfolio-cryptogains/` |
| Windows | `%AppData%\ghostfolio-cryptogains\` |

When `--config-dir <path>` is supplied, the application root becomes
`<path>/ghostfolio-cryptogains/`.

| Path below the application root | Contents |
| --- | --- |
| `setup.json` | Bootstrap schema, selected server mode and origin, development-HTTP state, and update time. |
| `snapshots/*.snapshot` | Normalized activity history protected with a token-derived Argon2id key and AES-GCM. |
| `diagnostics/*.diagnostic.json` | Unencrypted diagnostics generated for eligible failures. |

On platforms that expose Unix permission bits, application directories request
mode `0700`, while setup, snapshot, diagnostic, and report files request mode
`0600`. Windows relies on the current user's filesystem access controls instead.

The Ghostfolio security token, Ghostfolio JWT, and raw unprotected Ghostfolio
payload are not persisted. Protected snapshots cannot be recovered without the
corresponding token.

Diagnostics are not encrypted. Production-mode diagnostics omit quantity and
monetary-value fields but can retain identifiers, timestamps, asset metadata,
currencies, comments, server origin, and failure context. Development-mode
diagnostics can retain financial values. Review a diagnostic before sharing it.

### Removing Local Data

Exit the application before removing local data.

- Delete `setup.json` to return the next launch to first-run setup.
- Delete `snapshots/` to remove protected synced activity history.
- Delete `diagnostics/` to remove local diagnostic reports.
- Delete every generated `ghostfolio-capital-gains-*.md` and
  `ghostfolio-capital-gains-*.pdf` file from Documents to remove cleartext report
  exports.

Deleting setup does not delete snapshots, diagnostics, or report exports.

## Current Limitations

- Activity types other than `BUY` and `SELL` are not supported.
- Report currencies are limited to USD and EUR.
- Reports are not country-specific tax-return output.
- Cross-currency report calculation may require live ECB or Federal Reserve
  access even when an existing protected snapshot is available offline.
- Cross-currency conversion is limited to source currencies and observations
  supported by the selected official provider.
- The Documents directory must exist and cannot be overridden.
- There is no report preview, persistent report history, or reopen catalog.
- There are no published release binaries or declared minimum compatible
  Ghostfolio product versions.

## Development

### Developer Requirements

- Go 1.26.5.
- Git and Make.
- A POSIX-compatible command environment for maintained Make recipes. On Windows,
  use WSL, Git Bash, or an equivalent environment.
- Network access when Go modules, toolchain components, vulnerability data, or
  opt-in live integrations are not cached.
- A fetched `origin/main` ref for the standard changed-source quality gate.

Development tools such as `golangci-lint`, `govulncheck`, `gitleaks`, and
`gocoverageplus` are pinned through `tool` directives in `go.mod`; separate global
installations are not required.

### Project Structure

| Path | Responsibility |
| --- | --- |
| `cmd/ghostfolio-cryptogains/` | CLI parsing, runtime assembly, startup routing, and Bubble Tea process entrypoint. |
| `internal/app/bootstrap/` | Process options and startup decisions. |
| `internal/app/runtime/` | Cross-package orchestration for setup, sync, snapshots, diagnostics, and report generation. |
| `internal/config/` | Bootstrap model, origin normalization, and atomic `setup.json` persistence. |
| `internal/ghostfolio/` | Ghostfolio HTTP client, DTO validation, and mapping into normalized activity. |
| `internal/integration/currency/` | ECB and Federal Reserve clients, rate evidence, conversion policy, and in-process caching. |
| `internal/sync/` | Normalized activity model, ordering, duplicate handling, year derivation, and history validation. |
| `internal/snapshot/` | Protected snapshot model, cryptographic envelope, discovery, compatibility, and atomic storage. |
| `internal/report/model/` | Report requests, calculated models, output metadata, validation, and report-domain errors. |
| `internal/report/basis/` | FIFO, LIFO, HIFO, average-cost, and scope-local hybrid allocation state. |
| `internal/report/calculate/` | Yearly gains-and-losses calculation from protected activity history. |
| `internal/report/presentation/` | Shared report display rules and format-neutral presentation rows. |
| `internal/report/markdown/` | Markdown main-report and Annex 1 rendering. |
| `internal/report/pdf/` | Local searchable PDF rendering, layout, pagination, and embedded fonts. |
| `internal/report/output/` | Documents resolution, collision-safe writes, cleanup, and default-app opening. |
| `internal/tui/` | Root workflow state, screens, reusable components, keyboard interaction, and result copy. |
| `internal/support/` | Shared date, decimal, exact-math, redaction, and text helpers. |
| `tests/` | Unit, contract, integration, empirical, performance, live external-integration, and shared test support. |
| `tools/` | Coverage denominator and gate tools plus explicit empirical-oracle regeneration tooling. |
| `specs/` | Feature requirements, plans, research, contracts, quickstarts, and task records. |

Keep Ghostfolio transport details inside `internal/ghostfolio/`, normalized sync
rules inside `internal/sync/`, report financial rules inside `internal/report/`,
cross-package orchestration inside `internal/app/runtime/`, and rendering or input
state inside `internal/tui/`.

### Verification Commands

| Command | Scope |
| --- | --- |
| `make test-unit` | Deterministic package-local tests under `cmd/`, `internal/`, and `tests/unit`. |
| `make test-contract` | Deterministic externally visible contracts under `tests/contract`. |
| `make test-integration` | Deterministic cross-package runtime flows under `tests/integration`; no live providers. |
| `make test-empirical` | Exact calculations against committed synthetic datasets and golden fixtures. |
| `make test-tools` | Repository-local empirical-oracle tool tests without regeneration. |
| `make test` | Exact deterministic aggregate of the five targets above. |
| `make coverage` | Canonical production coverage profile and 100% statement, line, and branch enforcement. |
| `make test-performance` | Isolated build-tagged resource scenarios; no live provider network or coverage artifact. |
| `make test-external-integration` | Opt-in live ECB and Federal Reserve checks; excluded from pull-request CI. |
| `make quality QUALITY_BASE_REF=origin/main` | Changed Go and module source lint, vulnerability, and secret scans. |

`make coverage` writes `dist/coverage/coverage.out` and
`dist/coverage/coverage.xml`. Diagnostic leaf targets are `coverage-unit`,
`coverage-contract`, `coverage-integration`, `coverage-empirical`,
`coverage-tools`, and `coverage-external-integration`. Performance evidence never
contributes to canonical coverage.

The changed-source quality gate considers `*.go`, `go.mod`, and `go.sum`. It must
also succeed through its explicit skip path for documentation-only changes.

Empirical datasets and generated oracle fixtures under `testdata/empirical/` are
read-only during ordinary development. `make regenerate-empirical-fixtures` is an
explicit maintenance operation that may use Python, Git, network access, and the
pinned rotki cache; do not run it unless the active specification authorizes
fixture maintenance.

### CI Checks

Pull requests use these independent checks on fresh Ubuntu runners:

- `test / run`
- `coverage / run`
- `test-performance / run`
- `quality`

Live external integration is not part of pull-request CI.

### Further Reading

- [Repository operating rules and package ownership](AGENTS.md)
- [Engineering constitution](.specify/memory/constitution.md)
- [Current implementation plan](specs/009-final-report-adjustments/plan.md)
- [Current verification quickstart](specs/009-final-report-adjustments/quickstart.md)
- [Feature specifications and design history](specs/)
- [Empirical dataset boundary](testdata/empirical/README.md)
- [Pinned rotki oracle provenance](third_party/rotki/README.md)
- [Maintained commands](Makefile)

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).
