# Implementation Plan: Generate Yearly Gains And Losses Report

**Branch**: `[005-generate-gains-report]` | **Date**: 2026-05-20 | **Spec**: `/specs/005-generate-gains-report/spec.md`
**Input**: Feature specification from `/specs/005-generate-gains-report/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add yearly Markdown capital gains and losses report generation on top of the protected activity cache created by `specs/003-store-activity-data/`. The main business workflow becomes `Sync and Reports`, unlocks one token-scoped context once, shows `Sync Data` with last successful sync metadata, gates `Generate Capital Gains Report` until reportable protected data exists, collects one report year and one cost-basis method, calculates gains and losses from normalized activity history with exact decimal arithmetic, saves one timestamped Markdown file to the current user's Documents folder, asks the operating system to open it, and returns to the unlocked context without retaining report history.

The implementation keeps HTTP, protected snapshot storage, TUI state, report calculation, Markdown rendering, and filesystem output in separate boundaries. No new third-party dependencies are planned.

## Technical Context

**Language/Version**: Go 1.26.3  
**Primary Dependencies**: Existing `charm.land/bubbletea/v2`, selected `charm.land/bubbles/v2` components, `charm.land/lipgloss/v2`, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/Fabianexe/gocoverageplus`, and Go standard library packages including `bytes`, `context`, `crypto`, `encoding/json`, `errors`, `fmt`, `net/http`, `os`, `os/exec`, `path/filepath`, `runtime`, `sort`, `strings`, and `time`. No new runtime dependency is planned.  
**Storage**: Existing bootstrap `setup.json` remains bootstrap-only. Existing local-only token-derived encrypted snapshots remain the only persisted synced-data store. This slice may require a protected activity-model version increment to persist a distinct `asset_identity_key`. The final generated Markdown file is intentionally cleartext in the user's Documents folder after save. Report content, report ledgers, output paths, and report history are not persisted back to setup or snapshots. No cleartext report temp file is written under application-managed storage or OS temp locations.  
**Testing**: `make test`, `make coverage`, integration-first Go `testing` suites under `tests/integration`, contract suites under `tests/contract`, targeted unit tests under `tests/unit` and package-local `_internal_test.go` files for complex basis and rendering rules, plus a gated large-history performance path.  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access, a writable user Documents folder, and an optional OS Markdown-file association.  
**Project Type**: Single-module Go TUI application.  
**Performance Goals**: For a synced history of up to 10,000 activities spanning at least 5 calendar years, at least 95% of yearly report runs complete calculation, Markdown rendering, and final save in under 2 minutes on supported hardware; unlocked menu actions remain responsive while report generation is in flight.  
**Constraints**: Ghostfolio security token is runtime-only and scoped to the active `Sync and Reports` context; token and JWT are never persisted or output; final report is Markdown only; no in-app report history; filenames use local `YYYY-MM-DD_HH-MM-SS` ordering and avoid overwrite with numeric suffixes; report generation uses years derived from synced activity data only; report values use exact decimal arithmetic with no report-boundary rounding; non-terminating required divisions fail the report attempt because no rounding policy exists in this slice; single-activity monetary input selection uses `order -> asset -> base` and fails instead of mixing tiers; after values leave that selected activity context the report uses `NO CURRENCY APPLIES, ALL CONSIDERED EQUAL` and performs no conversion or exchange-rate lookup; zero-priced explained `SELL` records reduce holdings and basis without realized gain or loss; output-location or calculation failure removes partial cleartext artifacts created by the failed attempt.  
**Scale/Scope**: One active token-unlocked context per application run; one selected Ghostfolio server; multiple protected snapshots can still exist per local OS user; one yearly report per generation action; five supported cost-basis methods; asset grouping, holdings, liquidations, reopening checks, and report sections use a stable stored Ghostfolio asset identity key, while symbol and name are rendering labels only.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Token handling remains runtime-only. The token is entered to unlock `Sync and Reports`, reused only while that context stays active, cleared when the user leaves that context or the application exits, and excluded from logs, diagnostics, generated reports, setup, snapshots, and result messages. Existing synced financial data remains in token-derived encrypted snapshots following the storage design from `003`. Report inputs, intermediate calculations, rendered Markdown, and output path state stay in memory until the final Documents file is written. The final Markdown file is intentionally cleartext and outside protected storage after save. No additional app-managed report history or cleartext temp storage is introduced. OWASP Top 10 review scope for implementation must cover cryptographic failures, identification and authentication failures, insecure design, security misconfiguration, software and data integrity failures, vulnerable or outdated components, and logging or diagnostic leakage.
- [x] Precision: Financial domain logic uses `apd.Decimal` and exact decimal helper functions only. No floating-point value is allowed in report calculation, rendering, or assertions. Currency-denominated values remain tied to explicit order, asset-profile, or base currency until the documented single-activity currency-context boundary selects one complete input context. No cross-activity currency conversion occurs. After that feature-defined boundary, values are intentionally treated as equal-value report inputs and rendered with the exact report-wide label required by the spec. No report-boundary rounding is defined or applied in this slice.
- [x] Testing: Integration-first tests must cover main-menu entry, token gating, token reuse, sync and report actions inside one context, last-sync timestamp display, report availability gating, year and method selection, generated Markdown structure, Documents save, OS-open success and failure, no report history, artifact leakage checks, and the full coverage gate. Targeted unit tests are justified for cost-basis calculators, exact decimal edge cases, single-activity currency selection, filename uniqueness, Markdown rendering, and output cleanup because those units have high calculation or IO branching risk.
- [x] Dependencies: No new third-party dependency is planned. Markdown rendering, Documents path handling, filename reservation, and OS default-app opening use the Go standard library. Existing dependencies remain justified by earlier slices.
- [x] External APIs: No new Ghostfolio endpoint is required for report generation. `Sync Data` inside the context reuses the existing Ghostfolio `api/v1` auth, user, and activities contracts from `003`. The only new integration is with local operating-system services for resolving a user Documents path through home-directory conventions and requesting default-app open through platform commands; opener failure is non-fatal and reported without deleting a successfully saved report.
- [x] Architecture: The design keeps report-domain calculation separate from TUI state, Ghostfolio transport, protected snapshot storage, and filesystem output. Runtime orchestration coordinates context unlock, sync, snapshot access, report generation, final save, and OS-open requests. Markdown rendering receives an already calculated report model and has no authority over calculation rules.

## Project Structure

### Documentation (this feature)

```text
specs/005-generate-gains-report/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── markdown-report.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── ghostfolio-cryptogains/
    └── main.go

internal/
├── app/
│   ├── bootstrap/
│   └── runtime/
├── config/
│   ├── model/
│   └── store/
├── ghostfolio/
│   ├── client/
│   ├── dto/
│   ├── mapper/
│   └── validator/
├── report/
│   ├── basis/
│   ├── calculate/
│   ├── markdown/
│   ├── model/
│   └── output/
├── snapshot/
│   ├── envelope/
│   ├── model/
│   └── store/
├── sync/
│   ├── model/
│   ├── normalize/
│   └── validate/
├── tui/
│   ├── component/
│   ├── flow/
│   └── screen/
└── support/
    ├── decimal/
    └── redact/

tests/
├── contract/
├── integration/
├── testutil/
└── unit/
```

**Structure Decision**: Keep the repository as one Go module rooted at the project root. Add `internal/report` for calculation, method-specific basis state, derived report models, Markdown rendering, and local output helpers. Keep protected snapshot reads and writes in `internal/snapshot`, sync-refresh orchestration in `internal/app/runtime`, and screen routing in `internal/tui`. Update `internal/sync/model` only for persisted normalized activity fields that reporting requires, such as a distinct stable asset identity key.

## Reporting Domain Design

- Treat `ProtectedActivityCache` as the only report input source after the token unlocks the active context. Report generation does not call Ghostfolio and does not force a new sync.
- Add or confirm a stable `asset_identity_key` in stored normalized activity data. The report groups activity, basis state, inclusion decisions, liquidation counts, and sections by that key. Symbol and name remain rendering labels. If the stored model changes, bump `activity_model_version`; unsupported older snapshots must fail safely or require a successful refresh before reporting.
- Build one in-memory asset timeline per `asset_identity_key` using only activity whose source-calendar year is less than or equal to the selected year. Activity after the selected year is ignored for that report run.
- Include an asset in main report sections when it has an open position at the selected year end or at least one full liquidation during the selected year. Assets fully liquidated before the selected year and not reopened on or before year end appear only in the reference section when they have at least one full liquidation before the cutoff.
- Maintain full-liquidation counts through the selected year end. A full liquidation is counted whenever the asset or applicable scope reaches zero quantity after applying an event.
- Resolve monetary inputs per activity using a strict `order -> asset -> base` context selection. One selected context must supply the complete monetary value set needed for that activity. The calculator must not fill missing gross value, fee, or unit price from a different tier.
- Compute acquisition basis as `gross_acquisition_value + acquisition_fee` and liquidation proceeds as `gross_liquidation_value - liquidation_fee` after the single-activity context is selected.
- Keep all calculation state in `apd.Decimal`. HIFO lot comparison should use cross multiplication where possible to avoid unnecessary division. Required divisions must be checked for exactness; if no finite exact decimal exists, generation fails with an actionable calculation error because this slice defines no rounding policy.
- Treat explained zero-priced `SELL` records as holding reductions. They consume quantity and remove basis according to the selected method, appear in detail rows, and do not contribute proceeds, realized gains, realized losses, or summary totals.
- Render all quantities and monetary values with canonical exact-decimal formatting, trimming only non-significant formatting and rendering zero as `0`.

## Workflow Design

- Main menu exposes `Sync and Reports` as the business workflow entry.
- Selecting `Sync and Reports` prompts for the Ghostfolio token once with masked input. Snapshot discovery and unlock happen before the context menu exposes protected sync metadata or report years.
- The unlocked context menu always shows `Sync Data` and `Generate Capital Gains Report`. `Sync Data` shows the last successful sync local date and time when a protected cache exists; otherwise it states that no synced data is available. `Generate Capital Gains Report` remains visible but unavailable until a cache exists and at least one reportable year can be derived.
- Running `Sync Data` inside the context reuses the entered token and returns to the unlocked context after success, failure, or server-replacement cancellation.
- Running `Generate Capital Gains Report` collects one available year and one supported method. The highlighted method shows a short explanation before generation.
- Successful generation shows the saved Markdown path and any automatic-open warning, then returns to the unlocked context without another token prompt.
- Leaving `Sync and Reports` clears the token and any in-memory report content. The next entry requires token input again.

## Output Design

- Use one final Markdown document per successful report run. No PDF, HTML, or secondary export format is in scope.
- Resolve the Documents directory from the current OS user's home directory plus `Documents`. If that path is unavailable or not writable, fail with an actionable non-secret error and leave no partial report file.
- Name files with a local timestamp prefix in `YYYY-MM-DD_HH-MM-SS` order and a stable report descriptor. If the target path exists, append `-2`, `-3`, and so on before `.md`.
- Reserve the final path using exclusive creation where supported. If writing fails after file creation, close and remove the partial file. If saving succeeds but OS-open fails, keep the file and report the open failure.
- Request OS default-app open after save through a platform-specific standard-library command adapter. Opener failure does not turn the saved report into a failed save.

## Verification Plan

- Contract tests cover visible workflow changes: `Sync and Reports` main-menu entry, token unlock, token reuse, context menu actions, last-sync timestamp, report gating, year and method choice, and result transitions.
- Integration tests use deterministic protected snapshots and fake runtime services to verify report generation without real Ghostfolio calls, Documents-folder behavior through temporary home directories, OS-open success and failure through a stub opener, and persisted-artifact leakage checks.
- Unit tests cover basis calculators for FIFO, LIFO, HIFO, Average Cost Basis, scope-local hybrid fallback, zero-priced holding reductions, full-liquidation counts, single-activity currency selection, canonical Markdown rendering, exact-division failure, filename suffixing, and partial-file cleanup.
- Coverage verification remains `make coverage`. Performance verification uses a deterministic 10,000-activity fixture spanning at least 5 years and is gated behind an opt-in environment variable.

## Complexity Tracking

No constitution violations require justification for this plan.
