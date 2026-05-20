# Research: Generate Yearly Gains And Losses Report

## Decision: Keep Report Calculation In A Dedicated Domain Boundary

Use `internal/report` for report request models, cost-basis calculation, derived report structures, Markdown rendering, and output helpers. Runtime orchestration calls this boundary after the protected activity cache is unlocked. TUI screens collect choices and render progress only.

**Rationale**: Cost-basis rules are domain behavior. Keeping them outside TUI, Ghostfolio transport, and snapshot packages preserves the repository's clean architecture boundary and makes calculation tests independent from terminal and filesystem behavior.

**Alternatives considered**:

- Put calculation in `internal/app/runtime`: rejected because runtime would gain detailed financial rules and become harder to test.
- Put calculation in `internal/sync`: rejected because sync owns normalization and validation of stored history, not report-specific yearly realization rules.
- Put calculation in TUI screens: rejected because presentation code must not own financial behavior.

## Decision: Use Existing `apd.Decimal` Arithmetic Without New Numeric Dependencies

Use `github.com/cockroachdb/apd/v3` for all quantities, gross values, fees, basis, proceeds, gains, losses, and totals. Use existing helpers from `internal/support/decimal` for canonical formatting when possible. Do not use `float64` or a new rational-number dependency.

**Rationale**: The constitution prohibits floating-point financial domain logic. `apd/v3` is already adopted by this repository and is sufficient for finite exact decimal arithmetic. Adding another numeric library would expand dependency surface before a demonstrated need exists.

**Alternatives considered**:

- `float64`: rejected because it violates deterministic financial precision.
- Integer minor units: rejected because crypto quantities and Ghostfolio monetary inputs do not share one fixed minor-unit scale.
- New rational arithmetic library: rejected because this slice requires canonical exact-decimal output, not fractional notation, and the existing dependency set should remain minimal.

## Decision: Fail Required Non-Terminating Decimal Divisions

When a required calculation division cannot be represented as a finite exact decimal, report generation fails with an actionable calculation error. The implementation should minimize division where possible, for example by comparing HIFO unit costs with cross multiplication and allocating basis as `lot_basis * disposed_quantity / lot_quantity`.

**Rationale**: The feature requires no report-boundary rounding and canonical exact-decimal rendering. A repeating decimal cannot be rendered as a finite exact decimal without introducing a rounding policy that the specification explicitly does not define for this slice.

**Alternatives considered**:

- Round to a fixed scale: rejected because no rounding policy exists in the spec.
- Render fractions: rejected because the spec requires canonical exact-decimal strings.
- Silently truncate: rejected because it would create unauditable financial output.

## Decision: Implement Five Cost-Basis Methods As In-Memory Report State

Implement FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order as report-run state. Do not persist method state or generated ledgers.

**Rationale**: The report is derived from the protected activity cache for one year and one method. Persisting derived state would create report history and increase sensitive storage scope without a requirement.

**Alternatives considered**:

- Persist per-method ledgers in the snapshot: rejected because the spec requires no report history and no derived report content in protected storage.
- Recalculate by scanning only in-year activity: rejected because prior activity establishes opening holdings and basis.
- Use one generic average-cost implementation for all methods: rejected because FIFO, LIFO, and HIFO require lot-specific matching and produce different results.

## Decision: Select One Complete Activity Currency Context Before Calculation

For each activity, resolve monetary inputs using this priority: order currency, then asset-profile currency, then base currency. A tier is usable only when it supplies the complete monetary value set needed for that activity. After selection, the calculation treats selected values from all activities as equal-value inputs and uses the report-wide label `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL`.

**Rationale**: This directly follows the feature's single-activity currency context rule and avoids undocumented currency conversion. It also preserves the constitution rule that values stay explicitly tied to currency until the feature-defined boundary.

**Alternatives considered**:

- Mix gross value from one tier and fee from another: rejected because it violates the single-activity context rule.
- Convert all values to a common currency: rejected because this slice has no exchange-rate source, conversion boundary, or rounding rule.
- Prefer base currency globally: rejected because the spec defines the priority as `order -> asset -> base`.

## Decision: Treat Zero-Priced Explained `SELL` Records As Holding Reductions

Use the active cost-basis method to remove quantity and basis for explained zero-priced `SELL` records. Show them in per-asset detail activity rows, but do not create realized proceeds, gains, losses, or summary contributions.

**Rationale**: The synced-data slice admits these records as explained non-taxable holding reductions. Reporting must carry that meaning forward while still keeping remaining basis correct.

**Alternatives considered**:

- Ignore zero-priced `SELL` records during reporting: rejected because holdings and basis would be wrong.
- Treat them as taxable disposals with zero proceeds and a loss: rejected because the spec defines zero gain and zero loss treatment.
- Fail any zero-priced `SELL`: rejected because valid explained records are already supported by the stored activity model.

## Decision: Require A Stable Stored Asset Identity Key For Grouping

Report grouping, holdings, liquidations, reopening checks, and section ordering use a stable stored Ghostfolio asset identity key. Symbol and name are display labels only. If current snapshots do not store this key distinctly, increment `activity_model_version` and require a successful refresh or safe compatibility failure before reporting.

**Rationale**: The feature clarification explicitly rejects grouping by display label. Current repository code appears symbol-centered in `ActivityRecord`, so implementation must not rely on symbol alone unless it is proven to be the stable stored identity.

**Alternatives considered**:

- Group by symbol: rejected because display symbols can change or collide and the spec requires stored identity grouping.
- Group by asset name: rejected because names are rendering labels and can be missing or mutable.
- Derive a key from symbol plus name at report time: rejected because it is not a stored Ghostfolio identity key.

## Decision: Use `Sync and Reports` As One Token-Unlocked Context

The main menu exposes `Sync and Reports`. Entering it prompts once for the Ghostfolio token and then shows `Sync Data` and `Generate Capital Gains Report`. The token is reused only while the context remains active and cleared on exit.

**Rationale**: The feature requires token reuse between related actions inside one active context while still requiring explicit user entry before protected data access.

**Alternatives considered**:

- Keep `Sync Data` directly on the main menu: rejected because `FR-001` requires `Sync and Reports`.
- Prompt separately for sync and report generation: rejected because the unlocked context must allow movement between actions without repeated token entry.
- Unlock protected data at application startup: rejected because bootstrap setup remains the only startup-readable state.

## Decision: Show Report Action While Unavailable When Data Is Missing

Inside the unlocked context, always show both `Sync Data` and `Generate Capital Gains Report`. Report generation is unavailable until a protected cache exists and has at least one reportable year, with a clear reason displayed.

**Rationale**: The spec requires both actions to be visible and report generation to be unavailable when synced data is not ready.

**Alternatives considered**:

- Hide report generation until data exists: rejected because it conflicts with the visible action requirement.
- Allow report generation to start and fail later: rejected because the unavailable reason can be known before entry.

## Decision: Display Last Successful Sync As An Absolute Local Timestamp

Show the last successful sync date and time beside `Sync Data` after the active token/server context unlocks a protected cache. Use an absolute timestamp rather than relative text.

**Rationale**: Absolute timestamps are deterministic for tests and avoid stale relative labels. The timestamp stays protected until after token unlock.

**Alternatives considered**:

- Show relative labels such as `5 minutes ago`: rejected because they are time-sensitive and harder to verify.
- Show snapshot metadata before token unlock: rejected because protected sync metadata should not be exposed before the token boundary.

## Decision: Generate Plain Markdown With Standard Library Rendering

Render one `.md` file using standard-library string or buffer builders. Do not add a Markdown, PDF, HTML, or document-generation dependency.

**Rationale**: The slice requires plain Markdown only. The report sections are deterministic text and tables, so a rendering dependency is unnecessary.

**Alternatives considered**:

- PDF output: rejected because PDF is out of scope for this slice.
- HTML output: rejected because Markdown is the only required output format.
- Markdown rendering library: rejected because deterministic Markdown text is simple enough with standard library code.

## Decision: Save Final Output In The User's Documents Folder

Resolve Documents as the current user's home directory plus `Documents`, using `os.UserHomeDir` and `filepath.Join`. If the folder is unavailable or not writable, fail with an actionable error and remove any partial file created by the attempt.

**Rationale**: The Go standard library has no full cross-platform known-folder API. The dependency-free home-plus-Documents convention satisfies the requirement for supported installations while allowing clear failure when the location is unavailable.

**Alternatives considered**:

- Add a platform known-folder dependency: rejected because this slice can satisfy requirements without a new dependency.
- Write to the current working directory: rejected because the spec requires Documents.
- Fall back silently to app-data or temp directories: rejected because it would violate the output-location requirement and cleartext temp constraints.

## Decision: Use Timestamped Unique Filenames With Numeric Suffixes

Use local `YYYY-MM-DD_HH-MM-SS` timestamp ordering in the filename and append `-2`, `-3`, and so on before `.md` if a file already exists.

**Rationale**: The timestamp keeps alphabetical order aligned with creation time. Numeric suffixes handle multiple reports in the same second without overwriting.

**Alternatives considered**:

- Random or UUID prefixes: rejected because they harm chronological sort order.
- Overwrite existing files: rejected by the spec.
- Include sub-second values in the primary timestamp: rejected because the specified filename order is second-based; suffixes are clearer and still deterministic.

## Decision: Treat OS Open Failure As Non-Fatal After Save

After a successful save, request default-app open through a platform command adapter using `os/exec`. If the open request fails, keep the saved file, tell the user the path and open failure, and return to the unlocked context.

**Rationale**: The acceptance scenarios require the saved file to remain when automatic opening fails. Default-app availability is outside application control.

**Alternatives considered**:

- Treat opener failure as report-generation failure and delete the file: rejected because the file was saved successfully.
- Do not attempt to open: rejected because the spec requires an OS open request.
- Add a cross-platform opener dependency: rejected because standard-library command execution is sufficient.

## Decision: Keep Pre-Save Report Content In Memory Only

Before final save, keep report inputs, calculated structures, and rendered Markdown in process memory within the unlocked context. Do not write cleartext report content to app-managed storage or OS temp directories. Do not persist final report content or metadata after save.

**Rationale**: This satisfies `SEC-002`, `SEC-003`, and the no report history requirement. The final cleartext Markdown file is intentionally outside the protected boundary only after it is successfully saved in Documents.

**Alternatives considered**:

- Write a temp Markdown file and rename it: rejected because it creates cleartext outside the final destination before save.
- Store generated report metadata in the snapshot: rejected because it creates report history.
- Store recent report path in setup: rejected because setup must remain bootstrap-only and the spec forbids an in-app report list.
