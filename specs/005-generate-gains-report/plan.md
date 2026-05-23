# Implementation Plan: Generate Yearly Gains And Losses Report

**Branch**: `[005-generate-gains-report]` | **Date**: 2026-05-20 | **Spec**: `/specs/005-generate-gains-report/spec.md`
**Input**: Feature specification from `/specs/005-generate-gains-report/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

**Bugfix**: 2026-05-22 — [BUG-001] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-002] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-003] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-004] Updated from bugfix patch

**Bugfix**: 2026-05-22 — [BUG-005] Updated from bugfix patch

**Bugfix**: 2026-05-23 — [BUG-006] Updated from bugfix patch

**Bugfix**: 2026-05-23 — [BUG-007] Updated from bugfix patch

## Summary

Add yearly Markdown capital gains and losses report generation on top of the protected activity cache created by `specs/003-store-activity-data/`. The main business workflow becomes `Sync and Reports`, unlocks one token-scoped context once, shows `Sync Data` with last successful sync metadata, reuses the unlocked runtime token for in-context sync without showing token input again, gates `Generate Capital Gains Report` until reportable protected data exists, collects one report year and one cost-basis method, calculates gains and losses from normalized activity history with exact decimal arithmetic, saves one timestamped Markdown file to the current user's Documents folder, asks the operating system to open it, and returns to the unlocked context without retaining report history. Entering `Sync and Reports` activates that context only after either a selected-server protected snapshot unlocks or Ghostfolio authenticates the token as a new isolated local-user context. If no selected-server snapshot unlocks and Ghostfolio rejects the token, the workflow remains on the unlock screen with `access denied`, `Unlock` disabled, and token-field clearing deferred until the user leaves with `Back` and later returns.

The implementation keeps HTTP, protected snapshot storage, TUI state, report calculation, Markdown rendering, and filesystem output in separate boundaries. No new third-party dependencies are planned.

## Technical Context

**Language/Version**: Go 1.26.3  
**Primary Dependencies**: Existing `charm.land/bubbletea/v2`, selected `charm.land/bubbles/v2` components, `charm.land/lipgloss/v2`, `github.com/cockroachdb/apd/v3`, `golang.org/x/crypto/argon2`, `github.com/Fabianexe/gocoverageplus`, and Go standard library packages including `bytes`, `context`, `crypto`, `encoding/json`, `errors`, `fmt`, `net/http`, `os`, `os/exec`, `path/filepath`, `runtime`, `sort`, `strings`, and `time`. No new runtime dependency is planned.  
**Storage**: Existing bootstrap `setup.json` remains bootstrap-only. Existing local-only token-derived encrypted snapshots remain the only persisted synced-data store. This slice may require a protected activity-model version increment to persist a distinct `asset_identity_key` derived from Ghostfolio ~~`symbolProfileId`~~ `SymbolProfile.id`, because the reviewed upstream activity contract exposes the stable reporting identity on the nested `SymbolProfile` object (BUG-002). Existing snapshots, fixtures, and compatibility handling created under the superseded field-name assumption need reevaluation during implementation. The final generated Markdown file is intentionally cleartext in the user's Documents folder after save. Report content, report ledgers, output paths, and report history are not persisted back to setup or snapshots. No cleartext report temp file is written under application-managed storage or OS temp locations. User removal of that persisted cleartext output is by deleting the saved Markdown file from Documents. Eligible report-generation failures also write a separate local JSON diagnostics artifact under the existing application-owned diagnostics directory, using the same production prompt and explicit-development-mode automatic-generation policy introduced in `003`. The shared diagnostics serialization must preserve absent source fields as explicit `null` rather than omitting them so both report-failure and synced-data diagnostics remain source-faithful.  
**Testing**: `make test`, `make coverage`, integration-first Go `testing` suites under `tests/integration`, contract suites under `tests/contract`, targeted unit tests under `tests/unit` and package-local `_internal_test.go` files for complex basis and rendering rules, plus a gated large-history performance path.  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access, a writable user Documents folder, and an optional OS Markdown-file association.  
**Project Type**: Single-module Go TUI application.  
**Performance Goals**: In the opt-in local verification path driven by `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1`, one yearly report run over a deterministic 10,000-activity fixture spanning at least 5 calendar years, with worst-case supported lot fragmentation for HIFO and scope-local fallback and a stub opener, completes request validation, calculation, Markdown rendering, final save, and opener invocation in under 2 minutes; unlocked menu actions remain responsive while report generation is in flight.  
**Constraints**: Ghostfolio security token is runtime-only and scoped to the active `Sync and Reports` context; token and JWT are never persisted or output; the in-context `Sync Data` workflow must not display, focus, or allow editing of that token after unlock; snapshot miss alone must not activate the context; if no selected-server snapshot unlocks, Ghostfolio authentication must succeed before a new isolated local-user context is exposed, and a server-rejected token must keep the user on the unlock screen with `access denied`, `Unlock` disabled, only `Back`, and token-field clearing deferred until the user leaves and later re-enters; final report is Markdown only; no in-app report history; filenames use local `YYYY-MM-DD_HH-MM-SS` ordering and avoid overwrite with numeric suffixes; report generation uses years derived from synced activity data only; report values use exact decimal arithmetic with no report-boundary rounding; non-terminating required divisions fail the report attempt because no rounding policy exists in this slice; single-activity monetary input selection uses `order -> asset -> base` and fails instead of mixing tiers; explicit fee `0` is valid while missing fee is not equivalent to zero; rendered cross-activity monetary outputs require one shared explicit report calculation currency across all contributing priced activities, that will be `NOT APPLICABLE` for this slice; no conversion or exchange-rate lookup occurs; zero-priced explained `SELL` records reduce holdings and basis without realized gain or loss; preserved explicit zero-valued `unit_price`, `gross_value`, or `fee_amount` on those rows remain optional source details rather than required calculation inputs and must not be coerced to blank solely because the row is zero-priced; output-location or calculation failure removes partial cleartext artifacts created by the failed attempt. Eligible report-generation failures reuse the shared diagnostics artifact flow: a successfully saved report with only an opener warning is not diagnostic-eligible, while calculation, validation, rendering, and output-preparation failures are. Diagnostics artifacts must exclude token/JWT data, respect production redaction versus explicit-development-mode detail, disclose their local path when written, and serialize absent source fields as explicit `null`.  
**Scale/Scope**: One active token-unlocked context per application run; one selected Ghostfolio server; multiple protected snapshots can still exist per local OS user; one yearly report per generation action; five supported cost-basis methods; asset grouping, holdings, liquidations, reopening checks, and report sections use a stable stored Ghostfolio asset identity key, while symbol and name are rendering labels only.

**BUG-007 Note**: Single-activity currency selection requires an explicit tier currency before completeness validation. Tiers that carry order, asset-profile, or base financial values but no currency label are ineligible and must be skipped. Within the selected tier, exact same-tier derivation is allowed, and multiplication from quantity and unit price to gross value takes precedence over division-based unit-price derivation.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: Token handling remains runtime-only. The token is entered to unlock `Sync and Reports`, reused only while that context stays active, cleared when the user leaves that context or the application exits, excluded from logs, diagnostics, generated reports, setup, snapshots, and result messages, and not shown again as editable in-context sync input after unlock. Snapshot miss alone does not unlock the context. If no selected-server snapshot unlocks and Ghostfolio rejects the token, the workflow must remain on the unlock screen with `access denied`, `Unlock` disabled, only `Back`, and the retained field value cleared only after the user leaves with `Back` and later returns. Existing synced financial data remains in token-derived encrypted snapshots following the storage design from `003`. Report inputs, intermediate calculations, rendered Markdown, and output path state stay in memory until the final Documents file is written. The final Markdown file is intentionally cleartext and outside protected storage after save, and user removal is by deleting that saved file from Documents. Eligible report-generation failures can additionally write a separate local diagnostics artifact under the same application-owned diagnostics directory used by sync failures. That artifact must exclude token/JWT material, never become a retained report copy, respect production redaction versus explicit-development-mode detail, and preserve absent source fields as explicit `null` in shared diagnostic JSON. No additional app-managed report history or cleartext temp storage is introduced. OWASP Top 10 review scope for implementation must cover cryptographic failures, identification and authentication failures, insecure design, security misconfiguration, software and data integrity failures, vulnerable or outdated components, and logging or diagnostic leakage.
- [x] Precision: Financial domain logic uses `apd.Decimal` and exact decimal helper functions only. No floating-point value is allowed in report calculation, rendering, or assertions. Currency-denominated values remain tied to explicit order, asset-profile, or base currency until the documented single-activity currency-context boundary selects one complete input context. No cross-activity currency conversion occurs. After that boundary, report , values are intentionally treated as equal-value report inputs and calculations without conversion can be used for rendered cross-activity monetary outputs. No report-boundary rounding is defined or applied in this slice.
- [x] BUG-007 currency-selection correctness: A monetary tier is eligible only when it includes an explicit currency code. Verification must prove that higher-priority currencyless tiers are skipped, lower-priority explicit-currency tiers can still supply the activity input, same-tier multiplication derives gross value before division-based fallbacks, and terminal failure occurs only after all explicit-currency tiers are exhausted.
- [x] Testing: Integration-first tests must cover main-menu entry, token gating, selected-server snapshot unlock, authenticated new-context unlock after snapshot miss, rejected-token refusal that keeps the unlock screen in an `access denied` back-only state with `Unlock` blocked and token-field clearing deferred until exit and re-entry, token reuse, sync and report actions inside one context, last-sync timestamp display, report availability gating, year and method selection, generated Markdown structure, Documents save, OS-open success and failure, no report history, artifact leakage checks, preserved explicit zero-valued explained zero-priced holding-reduction fields versus missing values, report-failure diagnostics eligibility for calculation, validation, rendering, and output-preparation failures, production prompt and explicit-development-mode automatic generation, diagnostics-path disclosure, activity-specific original persisted-record context, explicit `null` field rendering in both report and synced-data diagnostics, opener-only warning distinction after a successful save, and the full coverage gate. Targeted unit tests are justified for cost-basis calculators, exact decimal edge cases, single-activity currency selection, filename uniqueness, Markdown rendering, output cleanup, shared diagnostic JSON serialization, diagnostics redaction, original-persisted-record diagnostic capture, and rejected-token unlock-screen rendering because those units have high calculation or IO or state branching risk.
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
- Reuse the shared runtime diagnostic-report writer from the synced-data slice for eligible report-generation failures. Extend it with report-failure categories and user-visible outcome state, but keep the diagnostics artifact separate from successful Markdown output and keep opener-only warnings out of that eligibility set.
- Add or confirm a stable `asset_identity_key` in stored normalized activity data. For this slice, that key is the non-empty normalized string preserved from Ghostfolio ~~`symbolProfileId`~~ `SymbolProfile.id`, because the reviewed upstream activity contract exposes the stable reporting identity on the nested `SymbolProfile` object (BUG-002). The report groups activity, basis state, inclusion decisions, liquidation counts, and sections by that key. Symbol and name remain rendering labels. If the stored model changes, bump `activity_model_version`; unsupported older snapshots or records lacking that key must fail safely or require a successful refresh before reporting.
- Build one in-memory asset timeline per `asset_identity_key` using only activity whose source-calendar year is less than or equal to the selected year. Activity after the selected year is ignored for that report run.
- Include an asset in main report sections when it has an open position at the selected year end or at least one full liquidation during the selected year. Assets fully liquidated before the selected year and not reopened on or before year end appear only in the reference section when they have at least one full liquidation before the cutoff.
- Maintain full-liquidation counts through the selected year end. A full liquidation is counted whenever the asset or applicable scope reaches zero quantity after applying an event. The reference section still renders one row per asset identity key; for the scope-local hybrid method the displayed count is the sum of per-scope transitions to zero for that asset.
- Resolve monetary inputs per activity using a strict `order -> asset -> base` context selection. A tier is eligible only when it has an explicit currency code. Currencyless tiers are skipped even if they carry gross value, unit price, or fee fields. The first eligible tier that can supply or exactly derive the complete monetary value set for that activity becomes the selected context. The calculator must not fill missing gross value, fee, or unit price from a different tier.
- When a selected tier lacks gross value but has positive quantity and unit price, derive gross value by same-tier multiplication before any division-based unit-price fallback. Unit-price derivation by division remains allowed only when the division terminates exactly.
- Compute acquisition basis from `ActivityCalculationInput.gross_value + ActivityCalculationInput.fee_amount`, and compute liquidation proceeds from `ActivityCalculationInput.gross_value - ActivityCalculationInput.fee_amount`, after the single-activity context is selected for that activity. Explained zero-priced holding reductions require no activity monetary context and contribute only quantity and method-derived basis removal, even when synced data preserves explicit zero-valued source fields such as `unit_price`, `gross_value`, or `fee_amount`.
- Require one shared explicit report calculation currency across all priced activities that contribute to rendered cross-activity monetary outputs. For this slice no conversion is done and calculations ignore activity currency. The explicit report calculation currency is shown as `NOT APPLICABLE`.
- Keep all calculation state in `apd.Decimal`. Reuse `internal/support/decimal` for canonical exact-decimal parsing, exact division, and formatting, and add report-local helpers only for operations that are genuinely report-specific. HIFO lot comparison should use cross multiplication where possible to avoid unnecessary division. Required divisions must be checked for exactness; if no finite exact decimal exists, generation fails with an actionable calculation error because this slice defines no rounding policy.
- Treat explained zero-priced `SELL` records as holding reductions. They consume quantity and remove basis according to the selected method, appear in detail rows, and do not contribute proceeds, realized gains, realized losses, or summary totals. When synced data preserves explicit zero-valued `unit_price`, `gross_value`, or `fee_amount`, keep those values distinct from missing values in the report model and render them as `0` rather than blank when the corresponding detail-row field is shown.
- When a report failure is tied to one activity, capture the original persisted `ActivityRecord` from the protected cache as the diagnostics source of truth. Do not substitute selected single-activity inputs, rendered rows, or other derived report values. Shared diagnostic JSON must render absent source fields as explicit `null` for both report-failure and synced-data diagnostics.
- For the scope-local hybrid method, fallback state is tracked independently per `(asset_identity_key, applicable_scope)`, remains active for that open scope until quantity reaches zero, and is re-evaluated only after a later reacquisition in that same scope.
- Render all quantities and monetary values with canonical exact-decimal formatting, trimming only non-significant formatting and rendering zero as `0`.
- If a selected year stays reportable but yields no main-section assets, still generate a valid report with the documented summary empty-state, yearly total `0`, reference section behavior, and no detail sections.

## Workflow Design

- Main menu exposes `Sync and Reports` as the business workflow entry.
- Selecting `Sync and Reports` prompts for the Ghostfolio token once with masked input. Snapshot discovery runs first. If a selected-server snapshot unlocks, the context opens from that snapshot. If no selected-server snapshot unlocks, the context opens only when Ghostfolio authentication succeeds for a new isolated local-user context. If Ghostfolio rejects the token, the workflow stays on the unlock screen with `access denied`, `Unlock` disabled, only `Back`, and the token field preserved until the user leaves and later re-enters.
- The unlocked context menu always shows `Sync Data` and `Generate Capital Gains Report`. `Sync Data` shows the last successful sync local date and time when a protected cache exists; otherwise it states that no synced data is available. `Generate Capital Gains Report` remains visible but unavailable until a cache exists and at least one reportable year can be derived. When report generation is unavailable, Up and Down navigation skip that disabled action and land only on enabled actions.
- Running `Sync Data` inside the context reuses the entered token, does not render token input again, shows only the action prompt needed to start sync, and returns to the unlocked context after success, failure, or server-replacement cancellation.
- Running `Generate Capital Gains Report` collects one available year and one supported method. The highlighted method shows a short explanation before generation.
- A calculation, validation, rendering, or output-preparation failure after year and method selection keeps the user in the unlocked context, produces no output file, and reports only non-secret activity references in the transient UI. The same failure result must support the production prompt and explicit-development-mode automatic report-diagnostics flow, disclose the diagnostics path when one is written, and include original persisted activity data when the failure is activity-specific.
- Successful generation shows the saved Markdown path, makes that path available for later manual deletion by the user, and shows any automatic-open warning, then returns to the unlocked context without another token prompt. A saved report whose later automatic-open request fails remains a successful save with a warning, not a report-failure diagnostics case.
- Leaving `Sync and Reports` clears the token and any in-memory report content. Leaving a rejected-token unlock screen with `Back` also clears the retained failed-token field before the next entry. The next entry requires token input again.

## Output Design

- Use one final Markdown document per successful report run. No PDF, HTML, or secondary export format is in scope.
- Resolve the Documents directory using OS-appropriate user-document conventions first: on Linux, honor XDG user directories when configured and otherwise fall back to `$HOME/Documents`; on macOS, use the per-user Documents directory under the user's home as defined by the platform conventions; on Windows, target the per-user Documents known folder rather than assuming a literal folder name. If the resolved path is unavailable or not writable, fail with an actionable non-secret error and leave no partial report file.
- Name files as `ghostfolio-capital-gains-<year>-<method>-<YYYY-MM-DD_HH-MM-SS>.md`. Keep `YYYY-MM-DD_HH-MM-SS` ordering for deterministic sorting. If the target path exists, append `-2`, `-3`, and so on before `.md`.
- Reserve the final path using exclusive creation where supported. If writing fails after file creation, close and remove the partial file. If saving succeeds but OS-open fails, keep the file and report the open failure.
- Request OS default-app open after save through a platform-specific standard-library command adapter. Exactly one opener request is made after a successful save. Opener failure does not turn the saved report into a failed save.
- Eligible report-generation failures may write a separate `.diagnostic.json` troubleshooting artifact under the existing application-owned diagnostics directory, not in the user's Documents folder. These artifacts reuse the synced-data diagnostics location and path-disclosure flow, apply production redaction unless explicit development mode is active, and serialize absent source fields as explicit `null`.

## Verification Plan

- Contract tests cover visible workflow changes: `Sync and Reports` main-menu entry, token unlock, selected-server snapshot unlock, authenticated new-context unlock after snapshot miss, rejected-token refusal with `access denied`, blocked `Unlock`, back-only unlock navigation, deferred token clearing after leaving and re-entering, token reuse, in-context `Sync Data` token-free screen behavior, context menu actions, last-sync timestamp, report gating, skip-disabled keyboard navigation when report generation is unavailable, year and method choice, result transitions, report-failure diagnostics eligibility, production prompt and explicit-development-mode automatic generation, diagnostics-path disclosure, and opener-only warning distinction.
- Integration tests use deterministic protected snapshots and fake runtime services to verify report generation without real Ghostfolio calls, valid-new-context unlock after selected-server snapshot miss, rejected-token unlock refusal that stays on the unlock screen, Documents-folder behavior through temporary home directories, OS-open success and failure through a stub opener, empty-main-section reports, incomplete monetary-context failure after selection, activity-specific report-failure diagnostics that serialize the original persisted activity record with explicit `null` fields and no derived substitute values, same-calendar-date reopening behavior, in-context sync token immutability, skip-disabled unlocked-context menu navigation when report generation is unavailable, regression coverage that synced-data diagnostics also render explicit `null` fields instead of omitting them, and persisted-artifact leakage checks.
- Integration tests must also cover production-shaped priced `BUY` rows where order financial values exist but `order_currency` is absent, asset or base tier has explicit currency and complete or exactly derivable same-tier values, and report generation proceeds with that later tier instead of failing on the currencyless order tier.
- Unit tests cover basis calculators for FIFO, LIFO, HIFO, Average Cost Basis, scope-local hybrid fallback and post-zero reset, zero-priced holding reductions, production-shaped explained zero-priced `SELL` rows with explicit zero `unit_price`, `gross_value`, and `fee_amount`, full-liquidation counts, single-activity currency selection, rejected-token unlock-screen rendering and field-reset behavior, canonical Markdown rendering, exact-division failure, filename suffixing, partial-file cleanup, shared diagnostic JSON serialization, original-persisted-record diagnostic capture, explicit `null` emission for nullable fields, and report-failure diagnostics redaction rules.
- Unit tests must also cover skipping tiers without explicit currency before completeness validation, exact same-tier gross-value derivation by multiplication, and multiplication preference over division-based fallbacks within one selected explicit-currency tier.
- Coverage verification remains `make coverage`. Performance verification uses a deterministic 10,000-activity fixture spanning at least 5 years with worst-case supported lot fragmentation, a stub opener, and one timed end-to-end run gated behind an opt-in environment variable.

## Complexity Tracking

No constitution violations require justification for this plan.

- BUG-001: Reporting must distinguish absent monetary context from preserved explicit zero-valued source fields on explained zero-priced holding reductions so calculation, report models, and Markdown rendering do not collapse both cases into blank output.
- BUG-002: The upstream Ghostfolio asset identity dependency is `SymbolProfile.id`, not the superseded top-level `symbolProfileId` assumption, so DTO mapping, snapshot compatibility, and stored-fixture refresh handling need reevaluation together.
- BUG-003: The unlocked-context `Sync Data` path must not reuse the token-entry renderer or token-input flow after unlock; regression coverage must assert token-free screen behavior and stored-token-only sync start.
- BUG-004: In the keyboard-driven `Sync and Reports` menu, disabled report actions must stay visible with an unavailable reason while Up and Down navigation skip them instead of letting selection land on the disabled row.
- BUG-005: Unlock activation must distinguish selected-server snapshot unlock, Ghostfolio-authenticated new isolated local-user context, and Ghostfolio-rejected token. Snapshot miss alone must not expose `Sync and Reports`; rejected tokens need an `access denied` back-only unlock state with deferred field clearing until the user exits and re-enters.
- BUG-006: Report-generation failures need a diagnostics path separate from successful Markdown output yet consistent with the earlier synced-data diagnostics policy. The shared diagnostic serializer also currently omits nullable source fields, so this fix must extend both report and existing sync diagnostics to emit explicit `null` without regressing production redaction or opener-warning handling.
- BUG-007: Single-activity currency selection must separate tier eligibility from tier completeness. Higher-priority tiers that lack currency labels remain diagnostic context only and must not cause terminal failure. The implementation also needs exact same-tier derivation with multiplication precedence over division so production-shaped asset-tier inputs do not fail unnecessarily.
