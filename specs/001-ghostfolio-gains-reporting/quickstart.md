# Quickstart: Ghostfolio Gains Reporting

This document defines the verification flow that the implementation for this feature must make possible. The commands below describe the expected developer workflow for the implementation branch, not the current empty scaffold.

## Prerequisites

- Go 1.26.2 installed.
- The default Ghostfolio cloud server `https://ghostfol.io`, a self-hosted Ghostfolio server reachable over HTTPS, or a local fixture server used by integration tests.
- A valid Ghostfolio security token for manual end-to-end verification.
- A writable local filesystem location for encrypted snapshots and generated PDFs.

## Automated Verification Flow

1. Run the full automated test suite.

```bash
go test ./... -covermode=atomic -coverprofile=coverage.cov
```

2. Generate the branch and file coverage report required by the constitution.

```bash
gocoverageplus -i coverage.cov -o coverage.xml
```

3. Confirm the suite covers these baseline journeys:

- first-run setup gating
- successful token-to-JWT sync
- failed auth without local-profile creation
- encrypted snapshot unreadable with the wrong token
- server-mismatch warning and confirmed replacement
- chronological normalization and exact-duplicate removal
- unsupported-event rejection
- non-defensible-history rejection
- all five cost basis methods
- yearly report generation ordering and inclusion rules

## Manual TUI Verification Flow

1. Launch the application.

```bash
go run ./cmd/ghostfolio-cryptogains
```

2. On first run, complete setup by keeping the default Ghostfolio cloud server `https://ghostfol.io` or by entering a custom self-hosted server origin. Any non-HTTPS production-like origin must be rejected with a blocking error.

3. Start a sync session for the selected registered local user and enter the Ghostfolio security token.

Expected result:

- token entry is masked
- no reporting actions are available before sync succeeds
- the application exchanges the token for a session JWT without persisting either secret
- the application writes one encrypted local snapshot only after the sync is normalized and validated

4. Restart the application and unlock the stored snapshot with the same token.

Expected result:

- the cached profile unlocks successfully
- the token is requested again for the new session
- report years are derived from the cached activity history

5. Change the configured Ghostfolio server origin and start another sync.

Expected result:

- the application warns that continuing will clean the current protected data tied to that user and token
- declining the prompt keeps the old encrypted snapshot intact
- accepting the prompt replaces the snapshot only after the new sync succeeds

6. Generate yearly reports for each supported cost basis method.

Expected result:

- only years present in cached data are selectable
- the gains and losses summary is the first section in the PDF
- open positions and in-year liquidations remain in the main sections
- assets liquidated before the selected year and never reopened appear only in the reference list

## Fixture-Driven Integration Expectations

The implementation should provide deterministic fixtures under `tests/fixtures/` for:

- valid multi-year activity history
- out-of-order and duplicate activities that normalize successfully
- unsupported event types
- gaps or inconsistencies that make basis calculation non-defensible
- zero-priced `BUY` and `SELL` activity with explanatory comments
- reliable and unreliable account-scope cases used as wallet-equivalent matching input

These fixtures must allow the full sync and reporting suite to run without requiring a live Ghostfolio server.
