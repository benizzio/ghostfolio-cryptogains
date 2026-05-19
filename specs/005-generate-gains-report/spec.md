# Feature Specification: Generate Yearly Gains Report

**Feature Branch**: `[005-generate-gains-report]`  
**Created**: 2026-05-19  
**Status**: Draft  
**Input**: User description: "Use previously synced activity data to add yearly capital gains report generation to the main menu, show the last successful sync time beside the sync option, require year selection and cost basis method selection before generation, save the report as a timestamped Markdown file in the user's Documents folder, open it in the operating system's default application, keep no report history, protect report contents until the final file is saved, and for this slice choose the first available currency tier in order `order -> asset -> base` while treating all chosen currencies as equal-value inputs without conversion."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See Report Readiness From The Home Menu (Priority: P1)

After unlocking the application, the user can tell from the home menu whether synced data is ready for reporting, when it was last refreshed, and whether report generation is currently available.

**Why this priority**: The user must be able to judge report readiness before starting a financial calculation workflow.

**Independent Test**: Open the home menu once with no synced data and once with synced data, then verify that the last-sync information and report-generation availability are shown correctly in each state.

**Acceptance Scenarios**:

1. **Given** no synced activity data exists, **When** the home menu is shown, **Then** both `Sync Data` and `Generate Capital Gains Report` are visible, and report generation is unavailable with a clear reason.
2. **Given** synced activity data exists, **When** the home menu is shown, **Then** the `Sync Data` option shows the last successful sync date and time and `Generate Capital Gains Report` is available.
3. **Given** a sync has just completed successfully, **When** the user returns to the home menu, **Then** the displayed last-sync timestamp reflects that completed sync.

---

### User Story 2 - Obtain Year For Synced Data Markdown Report (Priority: P1)

With synced data available, the user can start report generation, choose a year and cost basis method, generate a yearly capital gains report, have it saved to the user's Documents folder, and return to the home menu after the file is opened or an opening failure is explained.

**Why this priority**: Producing the yearly capital gains report from already synced data is the core user outcome of this slice.

**Independent Test**: Using a deterministic multi-year synced dataset, select an available year and a supported cost basis method, generate the report, verify the output file contents and location, and confirm that the workflow returns to the home menu.

**Acceptance Scenarios**:

1. **Given** synced data contains at least one reportable year, **When** the user selects a year and cost basis method and confirms generation, **Then** the system creates a yearly capital gains report as a Markdown file in the user's Documents folder, requests the operating system to open it, and returns the user to the home menu.
2. **Given** the selected year has activity before and after it, **When** the report is calculated, **Then** earlier activity is used to establish holdings and basis and later activity is ignored.
3. **Given** an asset has an open position at the end of the selected year or is fully liquidated during the selected year, **When** the report is generated, **Then** that asset appears in the main report sections.
4. **Given** an asset was fully liquidated before the selected year and was not reopened later, **When** the report is generated, **Then** that asset is excluded from the main sections and shown only in the reference section.
5. **Given** an included asset has a zero net result for the selected year, **When** the report is generated, **Then** that asset still appears in the gains-and-losses summary with a zero result.
6. **Given** the report file is saved successfully but the operating system cannot open it automatically, **When** the workflow completes, **Then** the saved file remains in the Documents folder, the user is told where it was saved and that automatic opening failed, and the application returns to the home menu.

---

### User Story 3 - Choose And Understand A Cost Basis Method (Priority: P2)

Before generating the report, the user can review the available cost basis methods, read a short explanation of each one, and choose the method that should govern that report run.

**Why this priority**: Different cost basis methods can materially change reported gains or losses, so the user needs an understandable and deliberate selection step.

**Independent Test**: Open the report-generation workflow with synced multi-year data, move through each method choice, verify the explanatory text, and compare method-specific outcomes against controlled expected ledgers.

**Acceptance Scenarios**:

1. **Given** the user is on the cost basis selection step, **When** the highlighted method changes, **Then** a short explanation describes how disposals are matched or pooled and whether scope-specific fallback rules apply.
2. **Given** any supported cost basis method is selected, **When** the yearly report is generated, **Then** that one method is applied consistently to every included disposal in that report run.
3. **Given** the scope-local hybrid method is selected and reliable scope information is unavailable for an asset, **When** the report is calculated, **Then** the method broadens to the whole asset instead of failing the report solely because scope detail is missing.

---

### Edge Cases

- Synced activity data exists but contains no reportable year, so report generation remains unavailable with a clear reason.
- Two reports are generated within the same second, so filenames must stay unique without losing alphabetical date ordering.
- A selected year contains acquisitions and holding reductions but no taxable disposals, producing a valid report with zero realized gain or loss.
- A selected year contains only zero-priced disposal records that reduce holdings without realizing gain or loss.
- The user's Documents location is unavailable or not writable at generation time.
- The synced dataset contains mixed currency labels across activities; for this slice the report must still choose the first available currency tier per activity and treat the chosen values as equal without conversion.
- The user generates a report and immediately starts another one; the application shows no report history or previously generated report list.

## Requirements *(mandatory)*

Each feature specification MUST capture security, persistence, financial
precision and currency-handling, testing, dependency, and external integration
impacts when the feature touches those areas.

### Functional Requirements

- **FR-001**: The system MUST present `Sync Data` and `Generate Capital Gains Report` as separate primary actions on the home menu.
- **FR-002**: The system MUST show the last successful sync date and time alongside `Sync Data` when synced activity data exists, and MUST indicate when no synced data is available.
- **FR-003**: The system MUST keep `Generate Capital Gains Report` unavailable until synced data exists and at least one reportable year can be derived from that data.
- **FR-004**: The system MUST open a dedicated report-generation workflow when the user selects `Generate Capital Gains Report`.
- **FR-005**: The system MUST allow the user to choose only from years present in the synced activity data.
- **FR-006**: The system MUST allow the user to choose one cost basis method from this set for each report run: FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order.
- **FR-007**: The system MUST show a short plain-language explanation for the highlighted or selected cost basis method before the report is generated.
- **FR-008**: The system MUST calculate the report from the currently synced dataset and MUST not require a new sync to begin report generation.
- **FR-009**: The system MUST use activity before and within the selected year to establish holdings and basis for that year and MUST ignore activity after that year.
- **FR-010**: The system MUST follow the reference report template structure with sections in this order: gains-and-losses summary, reference section for previously liquidated assets, then per-asset detail sections.
- **FR-011**: The system MUST include in the main report sections every asset that has an open position at the end of the selected year or is fully liquidated during the selected year.
- **FR-012**: The system MUST keep an included asset in the gains-and-losses summary even when that asset's net result for the selected year is zero.
- **FR-013**: The system MUST exclude from the main report sections any asset fully liquidated before the selected year and not reopened later, and MUST list that asset only in the reference section.
- **FR-014**: The system MUST show each asset detail section as the opening position carried into the selected year followed by the activity that occurs within the selected year, without including later activity.
- **FR-015**: The system MUST generate the report only as a plain Markdown document in this slice.
- **FR-016**: The system MUST name the output file with a human-readable local timestamp in `YYYY-MM-DD_HH-MM-SS` order so filenames sort alphabetically by creation time.
- **FR-017**: The system MUST avoid overwriting an existing report file by preserving the timestamped name and adding a disambiguating suffix when needed.
- **FR-018**: The system MUST save the generated report into the current user's personal Documents folder for the operating system in use.
- **FR-019**: The system MUST request the operating system to open the saved Markdown file in the default application associated with that file type.
- **FR-020**: The system MUST return the user to the home menu after report generation completes and MUST show a transient result message that confirms the saved file path or explains why automatic opening failed.
- **FR-021**: The system MUST NOT keep an in-application history, catalog, or reopen list of previously generated reports, and MUST NOT store the final report content back into protected synced-data storage after the file is saved.
- **FR-022**: The system MUST define FIFO as allocating disposal basis from the oldest still-open acquisitions first.
- **FR-023**: The system MUST define LIFO as allocating disposal basis from the newest still-open acquisitions first.
- **FR-024**: The system MUST define HIFO as allocating disposal basis from the still-open acquisitions with the highest unit cost first, and when two candidate acquisitions have the same unit cost, the older acquisition MUST take precedence.
- **FR-025**: The system MUST define Average Cost Basis as one moving weighted-average cost pool per asset across all activity for that asset, where each disposal uses the average unit cost immediately before that disposal and a new pool begins only after the asset quantity returns to zero.
- **FR-026**: The system MUST define Scope-Local Exact Unit Matching, otherwise Scope-Local Average Cost with Oldest-Acquired Deemed-Disposal Order as first choosing the most reliable available scope for an asset, then using exact unit matching within that scope when outgoing units can be defensibly identified, otherwise using average cost within that same scope and deeming the oldest remaining acquisitions in that scope as disposed first for provenance.
- **FR-027**: The system MUST broaden the scope-local hybrid method to the whole asset when reliable scope information is unavailable, instead of failing the report solely because scope detail is missing.
- **FR-028**: The system MUST continue using the pooled average-cost state within an open scope after the scope-local hybrid method first falls back to average cost there, until the quantity for that scope returns to zero.
- **FR-029**: The system MUST treat zero-priced disposal records that represent fees or transfers out as holding reductions that remove basis under the selected cost basis method without creating realized gain or loss entries in the report.
- **FR-030**: The system MUST apply the one selected cost basis method consistently to every included disposal in a single report run.
- **FR-031**: When monetary amounts are needed for reporting, the system MUST choose the first available currency context for each activity in this priority order: transaction order currency, asset currency, then portfolio base currency.
- **FR-032**: For this slice, the system MUST treat amounts chosen from different currency contexts as equal-value inputs and MUST NOT perform currency conversion or exchange-rate lookup.
- **FR-033**: If report generation cannot complete because the output location is unavailable, not writable, or the synced data cannot support the selected calculation, the system MUST show an actionable error without exposing secrets or unprotected financial details and MUST NOT leave partial cleartext report artifacts behind.

### Security, Precision, and Integration Constraints

- **SEC-001**: Before the final Markdown file is saved to the user's Documents folder, report inputs, intermediate calculations, rendered content, and any temporary storage MUST remain inside the same security boundary and protection level used for synced financial and user-linked data, and they MUST be kept out of logs, diagnostics, and unprotected temporary locations.
- **SEC-002**: The final Markdown file saved in the Documents folder is intentionally outside the application's protected-storage boundary. After that file is saved and handed to the operating system, the application MUST retain no additional cleartext copy of the report and MUST leave no recoverable cleartext temporary residue under application-managed storage.
- **FIN-001**: All quantities, proceeds, fees, allocated basis, and gains or losses MUST use exact decimal arithmetic, preserve source precision until final rendering, include fees in acquisition basis or disposal proceeds as applicable, honor zero-priced holding reductions as zero-gain and zero-loss events, choose currency context per activity using the `order -> asset -> base` priority, and perform no currency conversion in this slice.
- **QUAL-001**: Automated validation MUST cover home-menu gating with and without synced data, last-sync timestamp display, available-year selection, each supported cost basis method, yearly boundary handling, report section ordering and inclusion rules, zero-result assets, zero-priced holding reductions, Documents-folder save behavior, filename uniqueness and alphabetical sortability, automatic-open success and failure behavior, absence of report history, protected handling of pre-save report content, and the repository's full coverage gates using integration-first tests and only targeted unit tests for complex calculation rules.
- **INT-001**: The feature depends on an existing synced activity dataset from earlier slices and on operating-system services that provide a writable personal Documents location and a default Markdown-file association. This slice introduces no new external currency-conversion dependency.

### Key Entities *(include if feature involves data)*

- **Synced Activity Snapshot**: The protected local activity history and sync metadata used as the sole reporting input, including available report years and the last successful sync moment.
- **Report Request**: The user's selection of one report year and one cost basis method for a single generation run.
- **Cost Basis Method**: The domain rule set that decides how disposal basis is allocated across still-open holdings.
- **Yearly Capital Gains Report**: The generated yearly statement containing the gains-and-losses summary, the reference section for previously liquidated assets, and per-asset detail sections.
- **Report Output File**: The clear Markdown file saved in the user's Documents folder after generation, with a timestamped name and no continuing application ownership after save.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of home-menu states show both primary actions, and 100% of states without reportable synced data keep report generation unavailable with a clear reason.
- **SC-002**: 100% of users with synced data can reach the year-and-method selection step from the home menu in one action.
- **SC-003**: For controlled multi-year ledgers with known expected outcomes, 100% of yearly per-asset results and yearly totals match the expected values across all five supported cost basis methods, including zero-priced holding reductions and zero-result assets.
- **SC-004**: 100% of successful report runs create one Markdown file in the user's Documents folder with a timestamped name that sorts chronologically when listed alphabetically, and no successful run overwrites an earlier report file.
- **SC-005**: When a default Markdown-file association exists, at least 95% of successful report runs open the saved file automatically and return the user to the home menu without additional manual navigation.
- **SC-006**: For synced histories of up to 10,000 activities spanning at least 5 calendar years, at least 95% of yearly report runs complete and save the report in under 2 minutes on a supported installation.
- **SC-007**: 100% of inspected application-managed storage and temporary artifacts created during report generation keep cleartext report content out of persistent application storage before final save and leave no leftover cleartext temporary files after success or failure.

## Assumptions

- This slice reuses the protected synced activity data and sync metadata produced by earlier slices; extending data-sync rules beyond the home-menu status label is out of scope here.
- Reports remain yearly only in this slice; combined multi-year, quarter-based, and custom date-range reports are out of scope.
- The reference report structure from the earlier reporting specification remains the required content template for this slice, even though the output format is Markdown instead of PDF.
- Markdown is the only report document format in scope for this slice. PDF and other document types are deferred to later slices.
- The final file saved in the user's Documents folder is intentionally cleartext and outside the application's protected-storage responsibility after save.
- When multiple currency contexts are available for one activity, the report chooses the first available tier in this order: transaction order currency, asset currency, then portfolio base currency.
- Until a later slice defines real conversion rules, the report treats chosen monetary values from different currency contexts as equal-value inputs in calculations.
- The synced dataset already carries the scope detail and reliability information needed to decide whether a scope-local calculation can remain narrow or must broaden to the whole asset.
