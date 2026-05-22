# Feature Specification: Generate Yearly Gains And Losses Report

**Feature Branch**: `[005-generate-gains-report]`  
**Created**: 2026-05-19  
**Status**: Draft  
**Input**: User description: "Use previously synced activity data to add yearly capital gains and losses report generation inside a token-unlocked `Sync and Reports` workflow, show the last successful sync time beside the sync option, require year selection and cost basis method selection before generation, save the report as a timestamped Markdown file in the user's Documents folder, open it in the operating system's default application, keep no report history, protect report contents until the final file is saved, and for this slice choose one single-activity currency context in priority order `order -> asset -> base` while treating all selected activity values as equal-value inputs without conversion once they enter cross-activity calculations, and showing `NOT APPLICABLE` as this slice's shared report calculation currency for calculated row values"

## Clarifications

### Session 2026-05-20

- Q: What should each per-asset detail section contain beyond opening/in-year/closing blocks? → A: Show every in-year activity row, including acquisitions and non-taxable holding reductions, plus liquidation calculations.
- Q: How should yearly totals be shown in the report? → A: Include one overall yearly net total row at the end of `Gains-And-Losses Summary`.
- Q: How should assets be grouped for calculations and report sections? → A: Group by the stored Ghostfolio asset identity key, with display label used only for rendering.
- Q: How should numeric values be rendered in the Markdown report for this slice? → A: Render canonical exact-decimal values with no rounding in this slice, trimming only non-significant formatting.
- Q: How should rendered cross-activity monetary outputs handle currency in this slice? → A: A successful report uses one explicit shared report calculation currency that will be defined as `NOT APPLICABLE` for this slice while considering all currencies used in the calculation as equal.

**Bugfix**: 2026-05-22 — [BUG-001] Clarified that explained zero-priced holding reductions may preserve explicit zero-valued source fields without making monetary context required for calculation.

**Bugfix**: 2026-05-22 — [BUG-002] Corrected the upstream Ghostfolio asset identity dependency from the superseded `symbolProfileId` assumption to `Activity.SymbolProfile.id` for report grouping and refresh assumptions.

**Bugfix**: 2026-05-22 — [BUG-003] Clarified that in-context `Sync Data` must reuse the unlocked runtime token without showing or accepting token input again.

## Terms Used In This Spec

- **Sync and Reports context**: The token-unlocked workflow reached from the main menu. While this context remains active, the user can run `Sync Data` and `Generate Capital Gains Report` without informing the token again.
- **Single-activity currency context**: The one currency tier selected for one activity before any cross-activity basis or gains-and-losses calculation begins.
- **Report calculation currency**: The explicit currency code used for rendered cross-activity monetary outputs in one successful report run. Will be defined as `NOT APPLICABLE` for this slice while considering all currencies used in the calculation as equal.
- **Source calendar year**: The calendar year read from an activity's preserved `occurred_at` timestamp using the offset embedded in that timestamp, without converting it to machine-local time or forced UTC.
- **Reportable year**: A source calendar year present in the protected cache's `available_report_years`. A year remains reportable even when its in-year activity contains only acquisitions or only explained zero-priced holding reductions.
- **Inside the selected year**: An activity is inside the selected year when its source calendar year equals the chosen report year. Activity with a smaller source calendar year is earlier activity. Activity with a larger source calendar year is later activity.
- **Full liquidation**: A point where the quantity for an asset, or for that asset inside the applicable scope when a scope-local method is active, reaches zero.
- **Full liquidation count**: The number of full liquidations completed for an asset on or before the end of the selected year. For the scope-local hybrid method, one asset's count is the sum of applicable-scope transitions to zero through that cutoff.
- **Reference report template**: The section structure established by the earlier reporting specification and reused here in Markdown form.
- **Asset identity key**: The stable stored Ghostfolio asset identity preserved in synced data and used to group activity, holdings, liquidations, and report sections for one asset. Display labels do not change this grouping key.
- **First acquisition**: The earliest `BUY` activity for one asset identity key in deterministic synced-history order. `SELL` activity, including explained zero-priced holding reductions, does not change this definition, later reopenings do not redefine it, and no transfer-specific activity type exists in this slice because synced history supports only `BUY` and `SELL`.
- **Applicable scope**: For the scope-local hybrid method, either one reliable wallet or centralized trading platform account scope for an asset or the asset as a whole when scope data cannot be narrowed reliably.
- **Reliable scope data**: Scope information already classified by synced data as `reliable` and represented by a stable non-empty wallet or account identifier and scope kind without contradiction for the activities used by the report.
- **Defensible exact identification**: A liquidation or holding reduction in one applicable scope can be matched to one unique set of still-open acquisition fragments in that same applicable scope using deterministic synced-history order and no cross-scope inference.
- **Reopened on or before year end**: After a full liquidation event for one asset or one applicable scope, a later `BUY` for that same asset or applicable scope appears no later than the selected-year cutoff in deterministic synced-history order and causes quantity to become positive again by the end of the selected year.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enter The Sync And Reports Context (Priority: P1)

After setup is complete, the user can open `Sync and Reports`, unlock the token-scoped working context once, and then choose between `Sync Data` and `Generate Capital Gains Report` while seeing whether synced data is ready for reporting.

**Why this priority**: This keeps the existing application flow closer to earlier slices while still introducing reporting without forcing the user to re-enter the token between related actions.

**Independent Test**: Start from the main menu with completed setup, open `Sync and Reports`, provide the token once, verify that the contextual menu shows `Sync Data`, `Generate Capital Gains Report`, and the correct synced-data readiness state, then enter `Sync Data` and verify that the screen explains token reuse without rendering token input.

**Acceptance Scenarios**:

1. **Given** setup is complete and the user is on the main menu, **When** the user selects `Sync and Reports`, **Then** the system requires the Ghostfolio security token before exposing sync or reporting actions.
2. **Given** the token unlocks the active context and no synced activity data exists for that token and selected server, **When** the `Sync and Reports` menu is shown, **Then** both `Sync Data` and `Generate Capital Gains Report` are visible, `Sync Data` is available, and report generation is unavailable with a clear reason.
3. **Given** the token unlocks the active context and synced activity data exists, **When** the `Sync and Reports` menu is shown, **Then** `Sync Data` shows the last successful sync date and time and `Generate Capital Gains Report` is available.
4. **Given** the user completes a sync or report-generation workflow and remains inside the active unlocked context, **When** that workflow ends, **Then** the system returns to the `Sync and Reports` menu without requiring the token again.
5. **Given** the user selects `Sync Data` inside the active unlocked context, **When** the `Sync Data` screen is shown, **Then** the system does not ask for, display, focus, or allow editing of the Ghostfolio security token and shows only the action prompt needed to start sync.

---

### User Story 2 - Generate A Yearly Gains And Losses Markdown Report (Priority: P1)

With synced data available in the active unlocked context, the user can choose a year and cost basis method, generate a yearly gains-and-losses report, save it to the user's Documents folder, ask the operating system to open it, and return to the `Sync and Reports` menu.

**Why this priority**: Producing the yearly gains-and-losses report from already synced data is the core user outcome of this slice.

**Independent Test**: Using a deterministic multi-year synced dataset, select an available year and a supported cost basis method, generate the report, verify the output file contents and location, and confirm that the workflow returns to `Sync and Reports` without asking for the token again.

**Acceptance Scenarios**:

1. **Given** synced data contains at least one reportable year, **When** the user selects a year and cost basis method and confirms generation, **Then** the system creates a yearly capital gains and losses report as a Markdown file in the user's Documents folder, requests the operating system to open it, and returns the user to the `Sync and Reports` menu.
2. **Given** the selected year has activity before, within, and after it, **When** the report is calculated, **Then** earlier activity is used to establish holdings and basis, only liquidations inside the selected year contribute gains and losses, and later activity is ignored.
3. **Given** an asset's first acquisition occurs after the selected year, **When** the report is generated, **Then** that asset is ignored completely for that report run.
4. **Given** an asset has an open position at the end of the selected year or is fully liquidated during the selected year, **When** the report is generated, **Then** that asset appears in the main report sections.
5. **Given** an asset was fully liquidated before the selected year and was not reopened on or before the end of that selected year, **When** the report is generated, **Then** that asset is excluded from the main sections and shown only in the reference section.
6. **Given** an asset is fully liquidated before or within the selected year and a new position in that asset is opened before or within that same selected year, **When** the report is generated, **Then** only liquidations inside the selected year contribute gains and losses and the reference section shows the full-liquidation count reached by the end of the selected year for that asset.
7. **Given** an included asset has a zero net result for the selected year, **When** the report is generated, **Then** that asset still appears in the gains-and-losses summary with a zero result.
8. **Given** an included asset or a report total results in a loss, **When** the report is generated, **Then** that loss is shown with a negative sign.
9. **Given** the report file is saved successfully but the operating system cannot open it automatically, **When** the workflow completes, **Then** the saved file remains in the Documents folder, the user is told where it was saved and that automatic opening failed, and the application returns to the `Sync and Reports` menu.
10. **Given** a selected year remains reportable but no asset qualifies for the main report sections after all inclusion and exclusion rules are applied, **When** the report is generated, **Then** the report still succeeds, the gains-and-losses summary shows the documented empty state and an overall yearly net total of `0`, the report calculation currency label is `NOT APPLICABLE`, the reference section follows its own rules, and no per-asset detail sections are rendered.
11. **Given** the selected year and method require one priced activity whose monetary values cannot be supplied completely by any one activity currency context, **When** the user starts generation, **Then** the attempt fails before any file is saved, the error is actionable and non-secret, and the user remains inside the unlocked `Sync and Reports` context.
12. **Given** the selected year and method would render cross-activity monetary outputs from priced activities whose selected activity currency codes are not all the same, **When** the user starts generation, **Then** the calculations consider all currencies used in the calculation as equal and defines the shared report calculation currency as `NOT APPLICABLE` for this slice.
13. **Given** synced data preserves explicit zero-valued `unit_price`, `gross_value`, or `fee_amount` on an explained zero-priced holding reduction, **When** the report is generated, **Then** those preserved values remain available to render as `0` in that detail row, they do not create proceeds or gain-or-loss behavior, and they do not make an activity-currency context required for that row.

---

### User Story 3 - Choose And Understand A Cost Basis Method (Priority: P2)

Before generating the report, the user can review the available cost basis methods, read a short explanation of each one, and choose the method that should govern that report run.

**Why this priority**: Different cost basis methods can materially change reported gains and losses, so the user needs an understandable and deliberate selection step.

**Independent Test**: Open the report-generation workflow with synced multi-year data, move through each method choice, verify the explanatory text, and compare method-specific outcomes against controlled expected ledgers.

**Acceptance Scenarios**:

1. **Given** the user is on the cost basis selection step, **When** the highlighted method changes, **Then** a short explanation describes how acquisitions and liquidations are matched or pooled and whether scope-specific fallback rules apply.
2. **Given** any supported cost basis method is selected, **When** the yearly report is generated, **Then** that one method is applied consistently to every included liquidation in that report run.
3. **Given** the scope-local hybrid method is selected and reliable wallet or centralized trading platform account scope information is unavailable for an asset, **When** the report is calculated, **Then** the method broadens to the whole asset instead of failing the report solely because scope detail is missing.
4. **Given** the scope-local hybrid method is selected and one applicable scope first loses defensible exact identification mid-position, **When** the report is calculated, **Then** scope-local average-cost fallback activates for that scope only and remains active for later disposals in that same scope until that scope reaches zero.
5. **Given** the scope-local hybrid method is selected and one applicable scope reaches zero and is later reacquired, **When** the report is calculated, **Then** a later reacquisition in that same scope starts a new scope-local state whose exact-identification eligibility is evaluated again from that reacquisition forward, while other scopes for the same asset keep their own independent state.

---

### Edge Cases

- Synced activity data exists but contains no reportable year, so report generation remains unavailable with a clear reason.
- Two reports are generated within the same second, so filenames must stay unique without losing alphabetical date ordering.
- A selected year contains acquisitions and holding reductions but no liquidations, producing a valid report with zero realized gain or loss.
- A selected year contains only zero-priced disposal records that reduce holdings and basis without creating gain or loss, including rows that preserve explicit zero-valued `unit_price`, `gross_value`, or `fee_amount` and show those values as `0` rather than blank when rendered.
- A selected year remains reportable but no asset qualifies for the main report sections after all inclusion and exclusion rules, so the report renders the documented empty states and yearly total `0`.
- Same-asset activity rows share one source calendar date or cross a year boundary with different offsets, so classification and reopening decisions reuse deterministic synced-history order and source calendar years rather than Ghostfolio time-of-day.
- A priced `BUY` or priced `SELL` row from an older compatible snapshot has quantity `0` or less, so report generation fails safely instead of calculating from a non-defensible quantity.
- A priced activity has an explicit fee value of `0`, which remains valid, while an absent fee value remains incomplete and causes context selection to continue or the report attempt to fail.
- Scope-local exact matching falls back mid-scope, remains in fallback until that scope reaches zero, and later reacquisition in another scope does not reset the already-open scope.
- The user's Documents location is unavailable or not writable at generation time.
- The synced dataset contains mixed currency labels across activities. For this slice, each activity must still use one single-activity currency context, and report generation must consider all currencies used in the calculation as equal and define the shared report calculation currency as `NOT APPLICABLE` for this slice.
- The user generates a report and immediately starts another one inside the same unlocked context; the application shows no report history or previously generated report list.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, financial
precision and currency-handling, testing, dependency, and external integration
impacts when the feature touches those areas.

### Functional Requirements

- **FR-001**: The system MUST present `Sync and Reports` as the main-menu workflow entry for token-unlocked sync and reporting actions.
- **FR-002**: Selecting `Sync and Reports` MUST require the user to inform the Ghostfolio security token before the system exposes `Sync Data` or `Generate Capital Gains Report`.
- **FR-003**: While the active `Sync and Reports` context remains unlocked, the system MUST present `Sync Data` and `Generate Capital Gains Report` as separate actions and MUST allow the user to move between them without informing the token again. Inside that active context, the `Sync Data` workflow MUST use the stored runtime token and MUST NOT ask for, display, focus, or allow editing of the token again.
- **FR-004**: The system MUST show the last successful sync date and time beside `Sync Data` within the `Sync and Reports` context when synced data exists for the active token and selected server, and MUST indicate when no synced data is available.
- **FR-005**: The system MUST keep `Generate Capital Gains Report` unavailable until synced data exists and at least one reportable year can be derived from that data for the active `Sync and Reports` context.
- **FR-006**: The system MUST allow the user to choose only from years present in the synced activity data.
- **FR-006a**: The system MUST treat a `reportable year` as any source calendar year present in `available_report_years`, derived from each activity timestamp using that timestamp's own stored offset, and MUST keep that year selectable even when its in-year activity contains only acquisitions or only explained zero-priced holding reductions.
- **FR-007**: The system MUST allow the user to choose one cost basis method from this set for each report run: FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order.
- **FR-008**: The system MUST show a short plain-language explanation for the highlighted or selected cost basis method before the report is generated.
- **FR-009**: The system MUST calculate the report from the currently synced dataset in the active `Sync and Reports` context and MUST not require a new sync to begin report generation.
- **FR-010**: The system MUST generate the report according to the section definitions in `Report Structure Definitions`.
- **FR-011**: The system MUST show gains as positive values, losses with a negative sign, and zero results as zero in the gains-and-losses summary and any report totals.
- **FR-012**: The system MUST include in the main report sections every asset that has an open position at the end of the selected year or at least one full liquidation during the selected year.
- **FR-012a**: The system MUST group activity, holdings, liquidations, reopening checks, and report sections by the stored Ghostfolio asset identity key, and MUST use any asset symbol or name text only as a rendering label. The upstream dependency that supplies that stored key is defined in `FR-012b`.
- **FR-012b**: For this slice, the stored Ghostfolio asset identity key MUST be the non-empty normalized string preserved from Ghostfolio symbol-profile identity, specifically ~~`symbolProfileId`~~ `SymbolProfile.id` from the upstream activity contract already reviewed in `specs/003-store-activity-data/`. `symbolProfileId` is superseded by [BUG-002] because the reviewed Ghostfolio activity shape exposes the stable reporting identity on the nested `SymbolProfile` object. If a stored activity record required for reporting lacks that non-empty key, report generation MUST fail safely or require a refresh that rebuilds the protected cache with that key.
- **FR-013**: The system MUST calculate yearly gains and losses using only liquidations that occur inside the selected year.
- **FR-013a**: The system MUST classify activity as earlier than, inside, or later than the selected year by comparing the selected year with the activity's source calendar year from the preserved `occurred_at` timestamp and that timestamp's own stored offset.
- **FR-014**: The system MUST ignore an asset completely when that asset's first acquisition occurs after the selected year.
- **FR-014a**: For `FR-014`, `first acquisition` means the earliest stored `BUY` for one asset identity key in deterministic synced-history order. `SELL` rows, including explained zero-priced holding reductions, do not change an asset's first acquisition, a reopened position does not redefine it, and no other activity type can qualify as an acquisition in this slice because synced history supports only `BUY` and `SELL`.
- **FR-015**: The system MUST use activity before and within the selected year to establish holdings and basis for that year and MUST ignore activity after the selected year.
- **FR-016**: The system MUST exclude from the main report sections any asset fully liquidated before the selected year and not reopened on or before the end of the selected year, and MUST show that asset only in the reference section.
- **FR-016a**: `Reopened on or before the end of the selected year` MUST be determined by replaying the deterministic same-asset history order already established by synced data through the selected-year cutoff. Because that order ignores Ghostfolio time-of-day and sorts same-source-calendar-date activity as `BUY` before `SELL`, a `BUY` and `SELL` on the same source calendar date do not create a reopen-after-liquidation event within that date.
- **FR-017**: The system MUST, for an asset fully liquidated before or within the selected year and reopened before or within that same selected year, include only the selected year's liquidations in gains-and-losses results and MUST show that asset's full-liquidation count through the end of the selected year in the reference section.
- **FR-017a**: The reference section MUST show one row per asset identity key. For FIFO, LIFO, HIFO, and Average Cost Basis, `full-liquidation count through the end of the selected year` counts asset-level transitions to zero. For the scope-local hybrid method, it counts the sum of applicable-scope transitions to zero across that asset through the same cutoff.
- **FR-018**: The system MUST show each included asset detail section as the opening position at the start of the selected year together with the cost basis carried into that moment under the selected method, every in-year activity row including acquisitions, liquidations, and explained zero-priced holding reductions together with the cost basis after that row is applied, the liquidation calculations for each in-year liquidation, and the closing position at the end of the selected year together with the cost basis at that closing moment, without including later activity.
- **FR-018a**: If an included asset has no in-year activity inside the selected year, its detail section MUST still show the opening position and closing position and MUST show an explicit no-in-year-activity message instead of in-year tables.
- **FR-018b**: If no asset qualifies for the main report sections after applying `FR-012` through `FR-017`, the system MUST still generate a valid report for the selected year, MUST render an explicit empty-state in `Gains-And-Losses Summary`, MUST render `Overall Yearly Net Total` as `0`, and MUST omit per-asset detail sections.
- **FR-019**: The system MUST generate the report only as a plain Markdown document in this slice.
- **FR-019a**: The system MUST, for this slice, render report quantities and monetary values as canonical exact-decimal strings with no report-boundary rounding, trimming only non-significant formatting.
- **FR-020**: The system MUST name the output file with a human-readable local timestamp in `YYYY-MM-DD_HH-MM-SS` order so filenames sort alphabetically by creation time, MUST save the file into the current user's personal Documents folder for the operating system in use, and MUST avoid overwriting an existing report file by adding a disambiguating suffix when needed.
- **FR-021**: The system MUST request the operating system to open the saved Markdown file in the default application associated with that file type, MUST return the user to the unlocked `Sync and Reports` context after report generation completes, MUST show a transient result message that confirms the saved file path or explains why automatic opening failed, MUST tell the user the saved file path so the user can later remove that cleartext report by deleting the file from the Documents folder, and MUST NOT keep an in-application history, catalog, or reopen list of previously generated reports or store the final report content back into protected synced-data storage after the file is saved.
- **FR-022**: The system MUST define FIFO by the mathematical rules in `Cost Basis Method Definitions`.
- **FR-023**: The system MUST define LIFO by the mathematical rules in `Cost Basis Method Definitions`.
- **FR-024**: The system MUST define HIFO by the mathematical rules in `Cost Basis Method Definitions`.
- **FR-025**: The system MUST define Average Cost Basis by the mathematical rules in `Cost Basis Method Definitions`.
- **FR-026**: The system MUST define Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order by the mathematical rules in `Cost Basis Method Definitions`.
- **FR-026a**: For the scope-local hybrid method, the system MUST narrow to a wallet or centralized trading platform account scope only when synced data already classified that scope as `reliable` and every activity used in the report for that narrowed scope carries a stable non-empty scope identifier and scope kind without contradiction. Otherwise the applicable scope is the asset as a whole.
- **FR-026b**: For the scope-local hybrid method, `defensible` exact identification exists only when one liquidation or explained zero-priced holding reduction can be matched to one unique set of still-open acquisition fragments in the same applicable scope using deterministic synced-history order and no cross-scope inference.
- **FR-027**: The system MUST apply the selected cost basis method consistently to every included liquidation in a single report run.
- **FR-027a**: For the Scope-Local Exact Unit Matching method, the system MUST apply the selected exact-identification or fallback treatment consistently per `(asset, applicable_scope)` until that asset's quantity in that applicable scope reaches zero.
- **FR-027b**: In the scope-local hybrid method, the first liquidation or holding reduction in one applicable scope that lacks defensible exact identification MUST activate scope-local average-cost fallback for that scope only, MUST keep that fallback active for later disposals in that same scope until that scope reaches zero, and MUST NOT change the state of other scopes for the same asset.
- **FR-027c**: After an applicable scope reaches zero under the scope-local hybrid method, a later reacquisition in that same scope MUST start a new open-scope state whose exact-identification eligibility is evaluated again from that reacquisition forward. Reacquisition in a different applicable scope MUST start or continue only that different scope's own state.
- **FR-028**: The system MUST determine one report calculation currency for each successful report run. When priced activities contribute to rendered cross-activity monetary outputs, the calculations will consider all currencies used in the calculation as equal and defines the shared report calculation currency as `NOT APPLICABLE` for this slice.
- **FR-029**: The system MUST treat zero-priced disposal records that were admitted into synced data by the explained zero-priced `SELL` rule from `specs/003-store-activity-data/spec.md` as holding reductions that remove quantity and basis under the selected cost basis method without creating gain or loss entries in the report.
- **FR-029a**: For an explained zero-priced holding reduction, the absence of required monetary calculation input MUST NOT be interpreted as forbidding preserved explicit zero-valued source fields. When synced data already carries explicit zero-valued `unit_price`, `gross_value`, or `fee_amount`, the system MUST preserve those `0` values in report inputs and render them as `0` rather than blank when the corresponding detail-row fields are shown. Their presence MUST NOT create proceeds, gain, loss, priced-liquidation treatment, or an activity-currency-context requirement.
- **FR-030**: The system MUST apply `Single-Activity Currency Context Definitions` whenever monetary amounts are needed for calculations from one activity.
- **FR-031**: When monetary amounts are needed for calculations within one activity, the system MUST evaluate currency contexts in this priority order: transaction order currency, asset currency, then portfolio base currency, and select the first context that provides the complete required monetary value set for that activity.
- **FR-031a**: The complete required monetary value set is activity-type specific: a `BUY` requires gross acquisition value and fee in one chosen context, a priced `SELL` requires gross liquidation value and fee in one chosen context, and an explained zero-priced holding reduction requires no gross value, fee, or activity-currency context because it contributes only quantity and method-derived basis removal. Optional preserved explicit zero-valued source fields such as `unit_price`, `gross_value`, or `fee_amount` do not change that requirement and do not make the row a priced `SELL`.
- **FR-031b**: An explicit fee value of `0` satisfies the fee requirement for `FR-031a`. A missing fee value does not count as zero. A priced `BUY` or priced `SELL` with quantity less than or equal to `0` cannot support report calculation and MUST fail the report attempt under `FR-035`.
- **FR-031c**: When a complete chosen context for a priced activity lacks an explicit unit price but contains gross value, fee, and positive quantity, the system MAY derive unit price only when that division terminates exactly under `FIN-001`; otherwise the report attempt MUST fail.
- **FR-032**: After one activity's currency context is chosen, the system MUST use that one chosen context consistently for every monetary value needed from that activity and MUST NOT mix currency tiers within the same activity calculation input.
- **FR-033**: After monetary values leave the single-activity currency context and enter cost basis and gains-and-losses calculations, the system MUST carry forward the selected activity currency code with each priced activity input, MUST NOT perform currency conversion or exchange-rate lookup, and MUST use the report calculation currency defined by `FR-028` for rendered cross-activity monetary outputs.
- **FR-034**: If no context in the priority order can supply the complete set of monetary values needed for a calculation, the system MUST fail the report generation attempt with an actionable error instead of mixing currency tiers within that activity.
- **FR-034a**: If an incomplete monetary context is discovered after the user already selected year and method, the failed attempt MUST leave the user inside the unlocked `Sync and Reports` context, MUST produce no output file, and MUST identify the offending activity only by non-secret reference such as display label and source ID.
- **FR-035**: If report generation cannot complete because the output location is unavailable, not writable, or the synced data cannot support the selected calculation, the system MUST show an actionable error without exposing secrets or unprotected financial details and MUST NOT leave partial cleartext report artifacts behind.
- **FR-035a**: Report generation MUST revalidate report-slice calculation preconditions that earlier sync may not have guaranteed, including positive quantity for priced activity rows, exact-division termination where required by the selected method, chosen-context completeness, shared report-calculation-currency availability where rendered cross-activity monetary outputs are required, and the selected method's ability to allocate basis through the selected-year cutoff. If any such precondition fails, report generation MUST fail safely under `FR-035`.

### Report Structure Definitions

#### Report Header

This content appears before the first section heading.

It MUST show:

- the title `Ghostfolio Capital Gains And Losses Report`
- the selected year
- the selected cost basis method label
- the local generated-at timestamp
- the report calculation currency from `FR-028`

#### Gains-And-Losses Summary

This is the first section of the report.

It contains one entry for each asset that is included in the main report sections for the selected year.

It ends with one overall yearly net total row for the selected year.

If no asset qualifies for the main report sections, this section shows an explicit empty-state sentence before the overall yearly net total row, and that total row renders as `0`.

Each entry MUST show:

- the rendering label for the grouped asset identity key used in the report
- the asset's net gain, net loss, or zero result for the selected year only
- gains as positive values
- losses with a negative sign
- canonical exact-decimal rendering with no rounding in this slice, trimming only non-significant formatting
- the report calculation currency from `FR-028`

The overall yearly net total row MUST show:

- the net sum of the included assets' yearly gains and losses
- gains as positive values
- losses with a negative sign
- zero as `0`
- canonical exact-decimal rendering with no rounding in this slice, trimming only non-significant formatting
- the report calculation currency from `FR-028`

#### Reference Section

This is the second section of the report.

It provides a reference-only view of full-liquidation history through the end of the selected year.

For each asset that reaches zero quantity at least once on or before the end of the selected year, this section MUST show the full-liquidation count reached by that cutoff.

For the scope-local hybrid method, that asset-row count is the sum of applicable-scope transitions to zero for the asset through the cutoff.

Assets that were fully liquidated before the selected year and were not reopened on or before the end of the selected year appear only in this section and not in the main report sections.

#### Per-Asset Detail Sections

These sections appear after the reference section.

There is one detail section for each asset included in the main report sections.

Each detail section is grouped by one stored Ghostfolio asset identity key and may render a user-facing symbol or name label for that grouped asset.

Each detail section MUST show:

- the opening position at the start of the selected year together with the cost basis carried into that moment under the selected method
- every activity row that occurs within the selected year, including acquisitions, liquidations, and explained zero-priced holding reductions, together with the cost basis after that row is applied
- for each in-year liquidation, the disposed quantity, allocated basis, net liquidation proceeds, and gain or loss
- the closing position at the end of the selected year together with the cost basis at that closing moment

For priced in-year rows, the detail section may show that row's selected activity currency for gross value, fee, and net liquidation proceeds. The report calculation currency from `FR-028` remains the explicit currency for cross-activity calculation outputs such as carried basis, allocated basis, gain, and loss.

For explained zero-priced holding-reduction rows, activity currency remains optional. When synced data preserves explicit zero-valued `unit_price`, `gross_value`, or `fee_amount`, the corresponding rendered row shows those values as `0` rather than blank if that field is part of the rendered layout.

If an included asset has no in-year activity, the section shows an explicit empty-state sentence instead of in-year activity and liquidation tables.

If no asset qualifies for the main report sections, no per-asset detail sections are rendered.

All rendered quantities and monetary values in these sections use canonical exact-decimal rendering with no rounding in this slice, trimming only non-significant formatting.

Activity after the selected year is excluded from these sections.

### Cost Basis Method Definitions

#### Deterministic Ordering Reused From Synced Data

Whenever report generation compares acquisition age, resolves same-source-calendar-date ties, determines whether a liquidation occurred inside the selected year, or decides whether an asset or applicable scope was reopened, it reuses the deterministic same-asset order already established by `specs/003-store-activity-data/spec.md`:

```text
source calendar date from occurred_at
then activity_type with BUY before SELL
then source_id
```

Ghostfolio time-of-day precision is not reinterpreted during report generation. This same preserved order is also used for HIFO tie-breaks after unit-cost comparison.

#### Shared Calculation Rules

These formulas apply to all supported methods unless a method definition narrows them.

Acquisition:

```text
acquisition_basis = gross_acquisition_value + acquisition_fee
unit_cost = acquisition_basis / acquired_quantity
```

Liquidation:

```text
net_liquidation_proceeds = gross_liquidation_value - liquidation_fee
gain_or_loss = net_liquidation_proceeds - allocated_basis
```

If one liquidation draws from multiple matched acquisition fragments, net liquidation proceeds are allocated proportionally by matched quantity:

```text
proceeds_per_unit = net_liquidation_proceeds / liquidated_quantity
matched_proceeds_i = proceeds_per_unit * matched_quantity_i
matched_gain_or_loss_i = matched_proceeds_i - matched_basis_i
```

#### FIFO

FIFO allocates liquidation basis from the oldest still-open acquisitions first.

Partial-acquisition consumption follows this rule:

```text
lot_unit_cost = lot_basis / lot_quantity
matched_basis = lot_unit_cost * matched_quantity
remaining_quantity' = remaining_quantity - matched_quantity
remaining_basis' = remaining_basis - matched_basis
```

#### LIFO

LIFO allocates liquidation basis from the newest still-open acquisitions first.

It uses the same partial-acquisition formulas as FIFO after the newest acquisition is selected.

#### HIFO

HIFO allocates liquidation basis from the still-open acquisitions with the highest unit cost first.

If two still-open acquisitions have the same unit cost, the older acquisition takes precedence. If a stable tie-break is still needed after that comparison, the deterministic order already established in synced history is used.

It uses the same partial-acquisition formulas as FIFO after the highest-cost acquisition is selected.

#### Average Cost Basis

Average Cost Basis maintains one moving weighted-average pool per asset using all activity for that asset.

Pool state:

```text
pool_quantity
pool_basis
average_unit_cost = pool_basis / pool_quantity
```

On acquisition:

```text
pool_quantity' = pool_quantity + acquired_quantity
pool_basis' = pool_basis + acquisition_basis
average_unit_cost' = pool_basis' / pool_quantity'
```

On liquidation:

```text
allocated_basis = liquidated_quantity * average_unit_cost
pool_quantity' = pool_quantity - liquidated_quantity
pool_basis' = pool_basis - allocated_basis
gain_or_loss = net_liquidation_proceeds - allocated_basis
```

When pool quantity reaches zero, the next acquisition starts a new pool for that asset.

#### Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order

This method first chooses the most reliable available scope for an asset:

```text
applicable_scope = reliable wallet or centralized trading platform account scope when available and defensible
otherwise the applicable_scope = the asset as a whole
```

Within that applicable scope:

1. If the outgoing units can be defensibly identified exactly, allocate basis from those exact units.
2. If exact identification is not possible, deem the oldest-acquired units of the same asset in that same scope to have been disposed first.
3. For valuation under the fallback, use average cost within that same scope.
4. If exact identification remains possible for every liquidation in an open scope, continue exact-unit matching in that scope.
5. After the first average-cost fallback in an open scope, continue using wallet-local average cost for that asset in that scope until all units of that asset in that scope have been fully liquidated, even if later records contain more source detail.
6. After full liquidation and later reacquisition in that same scope, exact-identification eligibility is evaluated again from that reacquisition forward within the current report run.
7. Different applicable scopes for the same asset are tracked independently while they remain open.
8. Reacquisition in a different applicable scope does not reset or continue another scope's open exact-match or fallback state.

Exact-unit basis:

```text
allocated_basis = sum(matched_quantity_i * matched_unit_cost_i)
matched_unit_cost_i = matched_basis_i / matched_quantity_i
```

Scope-local average-cost fallback:

```text
deemed_disposal_order = remaining quantities ordered by acquired_at ascending, then deterministic synced-history order
scope_quantity = sum(remaining_quantity_i)
scope_basis = sum(remaining_basis_i)
average_unit_cost = scope_basis / scope_quantity
allocated_basis = liquidated_quantity * average_unit_cost
scope_quantity' = scope_quantity - liquidated_quantity
scope_basis' = scope_basis - allocated_basis
gain_or_loss = net_liquidation_proceeds - allocated_basis
```

#### Zero-Priced Holding Reduction Rule Carried Forward From Synced Data

This report slice reuses the earlier synced-data rule from `specs/003-store-activity-data/spec.md` that allows an explained zero-priced `SELL` record to exist in stored history.

When such a record is encountered during report generation:

```text
allocated_basis = method_specific_basis_for_removed_quantity
remaining_quantity' = remaining_quantity - removed_quantity
remaining_basis' = remaining_basis - allocated_basis
gain_or_loss = 0
```

The quantity and basis are reduced according to the active cost basis method, but the event produces no gain or loss entry in the report.

### Single-Activity Currency Context Definitions

Within one activity, the system chooses exactly one currency context before using that activity's monetary values in calculations.

The selection priority is:

```text
transaction order currency
then asset currency
then portfolio base currency
```

Once one context is chosen for that activity:

- every monetary value needed from that activity, such as unit price, fee, gross acquisition value, gross liquidation value, and activity totals, must come from that one chosen context
- the system must not combine different currency tiers inside that same activity
- if the chosen context cannot provide every monetary value needed from that activity, the report-generation attempt fails instead of mixing tiers

The complete required monetary value set is:

```text
BUY: gross_acquisition_value and acquisition_fee from one context
priced SELL: gross_liquidation_value and liquidation_fee from one context
explained zero-priced holding reduction: no activity monetary values are required from the row itself for calculation, but preserved explicit zero-valued source fields such as unit_price, gross_value, and fee_amount may still remain present
```

An explicit fee value of `0` is valid. A missing fee value is not equivalent to zero. For priced `BUY` and priced `SELL` activity, quantity must be greater than zero.

After values from multiple activities enter cost basis and gains-and-losses calculations, the report continues to carry forward the selected activity currency code from each priced activity. A successful report may render cross-activity monetary outputs with currency under `FR-028`. The system performs no conversion.

### Security, Precision, and Integration Constraints

- **SEC-001**: The Ghostfolio security token MUST be informed explicitly by the user to enter `Sync and Reports`, kept only for the active unlocked context, reusable for sync and reporting actions while that context remains open, cleared when the user leaves that context or the application ends, excluded/unreadable from logs, output, diagnostics, and persisted artifacts, and not re-rendered or exposed as editable input by the in-context `Sync Data` workflow after unlock.
- **SEC-002**: Before the final Markdown file is saved to the user's Documents folder, report inputs, intermediate calculations, rendered content, and any temporary storage MUST remain inside the same security boundary and protection level used for synced financial and user-linked data, and they MUST be kept out of unprotected temporary locations.
- **SEC-003**: The final Markdown file saved in the Documents folder is intentionally outside the application's protected-storage boundary. After that file is saved and handed to the operating system, the application MUST retain no additional cleartext copy of the report and MUST leave no recoverable cleartext temporary residue under application-managed storage. User removal of that persisted cleartext output is by deleting the saved Markdown file from the Documents folder because this slice keeps no report catalog or additional retained copy.
- **FIN-001**: All quantities, proceeds, fees, allocated basis, gains, and losses MUST use exact decimal arithmetic, preserve source precision until final rendering, include fees in acquisition basis or liquidation proceeds as applicable, treat explained zero-priced holding reductions as zero-gain and zero-loss events, show losses with a negative sign in final report output, and, for this slice, render report values as canonical exact-decimal strings with no report-boundary rounding and only non-significant formatting trimmed.
- **FIN-002**: Within one activity, calculations MUST use the single-activity currency context defined in `Single-Activity Currency Context Definitions`, including the priority `order -> asset -> base`, the rule against mixing currency tiers inside one activity, and the requirement to fail instead of mixing tiers when one chosen context is incomplete. After values enter cross-activity cost basis and gains-and-losses calculations, the system MUST preserve the selected activity currency code for priced activity inputs, MUST require one shared report calculation currency under `FR-028` for rendered cross-activity monetary outputs, and MUST perform no currency conversion in this slice.
- **QUAL-001**: Automated validation MUST cover main-menu entry into `Sync and Reports`, token gating and token reuse within the active unlocked context, in-context `Sync Data` token reuse without any visible or editable token input, last-sync timestamp display beside `Sync Data`, available-year selection, each supported cost basis method, yearly gains-and-losses calculations that count only liquidations inside the selected year, ignoring assets first acquired after the selected year, reference-section liquidation counts for reopened assets, report section ordering and definitions, negative-sign rendering for losses, zero-result assets, zero-priced holding reductions carried forward from the synced-data rules, preserved explicit zero-valued explained zero-priced holding-reduction fields distinguished from missing values, single-activity currency-context selection, no cross-tier mixing, report-calculation-currency determination, Documents-folder save behavior, filename uniqueness and alphabetical sortability, automatic-open request success and failure behavior, absence of report history, protected handling of pre-save report content, and the repository's full coverage gates using integration-first tests and only targeted unit tests for complex calculation rules.
- **INT-001**: The feature depends on the existing synced activity dataset from earlier slices, including available report years, scope reliability, and the explained zero-priced `SELL` records admitted by `specs/003-store-activity-data/spec.md`, which may preserve explicit zero-valued source fields even though report calculation does not require them, and on operating-system services that provide a writable personal Documents location and a default Markdown-file association. This slice introduces no external currency-conversion dependency.
- **INT-001a**: Report generation MAY trust from the earlier synced-data slice only these invariants: supported activity types already limited to `BUY` and `SELL`, stable stored asset identity keys preserved from upstream `SymbolProfile.id`, `available_report_years` already derived from source timestamps, deterministic same-asset order already established from source calendar date then `BUY` before `SELL` then `source_id`, explained zero-priced `SELL` rows already carrying their required explanation and possibly preserved explicit zero-valued source fields such as `unit_price`, `gross_value`, and `fee_amount`, running quantity already validated not to drop below zero during synced-data replay, and scope reliability already classified.
- **INT-001b**: Report generation MUST still validate report-run preconditions that are method-specific or output-specific, including selected-year membership, positive priced-activity quantity, chosen-context completeness, exact-division termination, shared report-calculation-currency availability where rendered cross-activity monetary outputs are required, method-specific basis-allocation ability, and output-path writability.

### Key Entities *(include if feature involves data)*

- **Protected Activity Cache**: The existing protected synced dataset reused by this slice. It contains the normalized activity history, available report years, scope reliability, and last successful sync metadata that gate and drive yearly report generation.
- **Sync and Reports Context**: The active token-unlocked workflow state for one selected server and one token-scoped protected dataset, exposing `Sync Data` and `Generate Capital Gains Report` without repeated token entry while the context stays open.
- **Report Request**: The user's selected report year and cost basis method for one report-generation run.
- **Activity Record**: One normalized `BUY` or `SELL` event from the protected activity cache, carrying the quantity, monetary tiers, explanatory comment, and source-scope details needed for basis allocation and yearly gains-and-losses calculations.
- **Asset Identity Key**: The stable stored Ghostfolio asset identity used to group activity records into one asset timeline and one report section set, independent of the display label shown to the user.
- **Single-Activity Currency Context**: The chosen monetary context for one activity, selected in tiers `order -> asset -> base` priority and used consistently for every monetary value needed from that activity.
- **Report Calculation Currency**: The explicit shared currency code rendered for cross-activity monetary outputs in one successful report run, or `NOT APPLICABLE` in this slice.
- **Source Scope**: The optional preserved account, wallet, or equivalent grouping attached to an activity record. It determines whether the scope-local hybrid method can remain narrow or must broaden to the asset as a whole.
- **Asset Position Timeline**: The derived per-asset chronological view used to establish opening position, in-year activity, closing position, report-section inclusion, and full-liquidation counting through the end of the selected year.
- **Capital Gains Report**: The generated yearly report that contains the gains-and-losses summary, the reference section, the per-asset detail sections, the report calculation currency from `FR-028`, and the generated Markdown output path.
- **Report Output File**: The clear Markdown file saved in the user's Documents folder after generation, with a timestamped name and no continuing application ownership after save.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of active `Sync and Reports` menu states show both `Sync Data` and `Generate Capital Gains Report`, and 100% of states without reportable synced data keep report generation unavailable with a clear reason.
- **SC-002**: 100% of users with synced data can reach the year-and-method selection step from `Sync and Reports` in one action and can return to `Sync and Reports` after report generation without re-entering the token while the unlocked context remains active, and 100% of in-context `Sync Data` entries explain token reuse without showing or accepting token input.
- **SC-003**: For controlled multi-year ledgers with known expected outcomes, 100% of yearly per-asset results and yearly totals match the expected values across all five supported cost basis methods, count only liquidations inside the selected year, ignore assets first acquired after the selected year, show losses with a negative sign, and include scope-local hybrid scenarios covering exact matching that remains exact, first fallback activation in one applicable scope, continued fallback until that scope reaches zero, and same-scope post-liquidation reset after later reacquisition.
- **SC-004**: 100% of generated reports follow the defined header and the three defined section types in the required order, display the explicit report calculation currency from `FR-028`, and preserve explicit zero-valued explained zero-priced holding-reduction detail fields as `0` when present.
- **SC-005**: 100% of successful report runs create one Markdown file in the user's Documents folder with a timestamped name that sorts chronologically when listed alphabetically, and no successful run overwrites an earlier report file.
- **SC-006**: 100% of successful report runs request one operating-system default-app open attempt after the final save, preserve the saved file when that open attempt fails, and return the user to the `Sync and Reports` context without additional manual navigation.
- **SC-007**: In the opt-in local verification path driven by `GHOSTFOLIO_CRYPTOGAINS_RUN_PERFORMANCE=1` with a deterministic 10,000-activity fixture spanning at least 5 calendar years and a stub opener, one yearly report run completes request validation, calculation, Markdown rendering, final save, and opener invocation in under 2 minutes.
- **SC-008**: 100% of inspected application-managed storage and temporary artifacts created during report generation keep cleartext report content out of persistent application storage before final save and leave no leftover cleartext temporary files after success or failure.

## Assumptions

- The main menu uses `Sync and Reports` as the entry point for both sync and report actions in this slice, and those actions run inside one active token-unlocked context.
- Reports remain yearly only in this slice. Combined multi-year, quarter-based, and custom date-range reports are out of scope.
- The reference report structure from the earlier reporting specification remains the required content template for this slice, even though the output format is Markdown instead of PDF.
- Markdown is the only report document format in scope for this slice. PDF and other document types are deferred to later slices.
- The final file saved in the user's Documents folder is intentionally cleartext and outside the application's protected-storage responsibility after save. The user removes that cleartext output later by deleting the saved Markdown file from Documents.
- The earlier synced-data slice already supplies the protected history, available report years, scope reliability, and explained zero-priced `SELL` records needed by this slice.
- Report generation trusts only the earlier-slice invariants listed in `INT-001a` and revalidates report-run preconditions listed in `INT-001b`.
- If one chosen single-activity currency context cannot provide every monetary value needed from an activity, report generation fails instead of mixing currency tiers inside that activity.
- Until a later slice defines real conversion rules, report generation carries forward each priced activity's selected currency code, requires one shared report calculation currency for rendered cross-activity monetary outputs using `NOT APPLICABLE` for this slice and considers all currencies equal during the calculations.
