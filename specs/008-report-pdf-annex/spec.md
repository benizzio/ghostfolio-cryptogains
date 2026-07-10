# Feature Specification: Capital Gains Report PDF And Audit Annex

**Feature Branch**: `[008-report-pdf-annex]`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "Specify GitHub issue #39, Capital Gains And Losses Report - new format and fixes, including GitHub issue #34, Audit report."

## Clarifications

### Session 2026-07-02

- Q: Where should detailed Currency Conversion Audit evidence appear after introducing Annex 1? → A: Move Currency Conversion Audit to Annex 1 only.
- Q: What report scale should PDF and Annex 1 generation support? → A: Support the existing 10,000 cached-activity report scale.
- Q: What filename contract should PDF output use? → A: Preserve the main report filename pattern and use `.pdf`.
- Q: What text accessibility contract should PDF output satisfy? → A: Generate a text-based searchable PDF with selectable report text.
- Q: Where may PDF generation run? → A: PDF generation must be local-only with no remote document service.

### Session 2026-07-03

- Q: When may PDF page breaks and page titles differ from Markdown? → A: Annex 1 always starts on a new page; additional PDF page breaks are allowed only when the next section, table row, or content block would not fit in the remaining printable page area, and continuation pages must repeat visible section or table context.
- Q: What exact user-facing labels are allowed for conversion status and quote direction? → A: `same_currency` renders as `Same currency`, `converted` renders as `Converted`, `source_per_base` renders as `Source currency per base currency`, and `base_per_source` renders as `Base currency per source currency`.
- Q: Do assets that appear only in the Reference Section belong in Annex 1 per-asset audit evidence? → A: Yes. A reported asset is any asset identity selected by the existing report inclusion or reference-section rules for the selected year, including reference-only assets and assets whose zero net summary rows are hidden.
- Q: What platform and font boundary must PDF output satisfy? → A: PDF generation must work on supported Linux, macOS, and Windows installations without requiring user-installed fonts, platform-specific font paths, a browser, or operating-system print-to-PDF support.

### Session 2026-07-04

- Q: Are generated report files application-managed persistence requiring token-derived encryption at rest? → A: No. Generated report files are explicit user-requested exports outside the application-managed persistence boundary. The application writes them locally only when the user requests generation, does not manage them as a cache or durable application state, does not re-ingest them automatically, and must still use safe filenames, owner-local file handling, failure cleanup, and secret redaction.

### Session 2026-07-05

**Bugfix**: 2026-07-05 — [BUG-001] Clarified Markdown initial report detail label formatting.

**Bugfix**: 2026-07-05 — [BUG-002] Clarified that PDF output must be formatted through the PDF renderer and must not render Markdown source syntax as the PDF body.

### Session 2026-07-07

**Bugfix**: 2026-07-07 — [BUG-003] Clarified that PDF output must use `github.com/signintech/gopdf` layout primitives for human-legible headings, styled labels, and tables instead of plain line dumping.

### Session 2026-07-09

**Bugfix**: 2026-07-09 — [BUG-004] Clarified landscape A4 PDF layout, table fit, non-overlapping section spacing, and section-specific PDF presentation rules.

**Bugfix**: 2026-07-09 — [BUG-005] Clarified full-width balanced PDF tables, readable section spacing, and bottom-margin-safe table continuation.

### Session 2026-07-10

**Bugfix**: 2026-07-10 — [BUG-006] Clarified 24-point PDF subheading spacing and concise continuation labels emitted only for actual table continuations.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose Report Output Format (Priority: P1)

As a user generating a capital gains and losses report, I want to choose whether the report is produced as PDF or Markdown before generation so that I can use the same report data in the format required by my review or filing workflow.

**Why this priority**: PDF output is the primary new capability requested, while the existing Markdown output must remain available.

**Independent Test**: Can be fully tested by generating the same report once as PDF and once as Markdown, then confirming that both outputs are available and contain the same required report data, text blocks, and table content except for format-specific pagination and file-splitting behavior.

**Acceptance Scenarios**:

1. **Given** the user has completed report setup and selected valid report inputs, **When** the user selects PDF before generation, **Then** the report is generated as an A4 PDF file.
2. **Given** the user has completed report setup and selected valid report inputs, **When** the user selects Markdown before generation, **Then** the report is generated in the existing Markdown format.
3. **Given** the same report inputs are used for PDF and Markdown, **When** both reports are reviewed, **Then** shared report sections contain the same output data, explanatory text, and table content, with only format-specific page breaks, page titles, and annex placement differing.
4. **Given** PDF output is selected, **When** the generated report is reviewed, **Then** headings, tables, emphasis, and Annex 1 content are presented as formatted PDF text and do not expose Markdown source syntax as report content.
5. **Given** PDF output is selected, **When** the generated report is reviewed, **Then** the visible report uses a human-legible heading hierarchy, styled classifier labels, table headers, table rows, table columns, wrapped cell content, and continuation context rather than a sequential dump of report lines.
6. **Given** PDF output is selected, **When** the generated report is reviewed, **Then** every page uses landscape A4 layout and wide report tables remain inside the printable area with visible right padding, wrapped cell content, and no clipped columns.
7. **Given** PDF output is selected, **When** a wide report table is rendered, **Then** it uses the available landscape printable width with equal left and right margins while retaining the required padding, wrapping, and no-clipping behavior.

---

### User Story 2 - Read A Clearer Main Report (Priority: P2)

As a user reviewing the main capital gains and losses report, I want labels and sections to be easier to scan and unnecessary zero-value information to be removed so that the report is shorter and clearer without losing relevant financial evidence.

**Why this priority**: The requested fixes reduce report bloat and make existing report information easier to interpret.

**Independent Test**: Can be tested by generating reports that include zero net gain or loss summary rows, rate-source disclosures, assets with and without report-year activity, zero-priced sell activities, and currency conversion statuses, then verifying the rendered main report content.

**Acceptance Scenarios**:

1. **Given** the main report has an initial details block with values such as Year and Cost Basis Method, **When** the report is rendered, **Then** each information classifier label in that block is bold.
2. **Given** the Gains-And-Losses Summary includes rows whose Net Gain Or Loss is zero, **When** the report is rendered, **Then** those zero net gain or loss rows are omitted and non-zero rows remain.
3. **Given** the report includes a Rate Source Summary, **When** the section is rendered, **Then** each information classifier label in that section is bold.
4. **Given** the Reference Section includes the full-liquidation count disclosure, **When** the report is rendered, **Then** the header reads `Historical Full Liquidation Count`.
5. **Given** an asset has no activity registered during the report year, **When** the Asset Detail section is rendered, **Then** the report shows the closing-position information under the title `Historical Position` and omits the separate `Opening Position`, `In-Year Activity`, and `Closing Position` subsections for that asset.
6. **Given** an In-Year Activity row includes a conversion status, **When** the row is rendered, **Then** the status is shown as a user-friendly label and does not expose code-style or snake_case values.
7. **Given** an In-Year Activity row represents a zero-priced SELL activity, **When** the row is rendered, **Then** the Type value is `BLOCKCHAIN OP` instead of `SELL`.
8. **Given** PDF output is selected, **When** the Gains-And-Losses Summary is rendered, **Then** `Overall Yearly Net Total` is the final row or footer inside the Gains-And-Losses Summary table.
9. **Given** PDF output is selected, **When** the Rate Source Summary is rendered, **Then** it uses bold classifier label lines followed by non-bold values and does not render as a `Rate Source Summary Table`.
10. **Given** PDF output is selected, **When** the Reference Section is rendered, **Then** it does not introduce a generated `Reference Table` subheading.
11. **Given** PDF output is selected, **When** adjacent main-report text sections, headings, or subheadings are rendered on the same page, **Then** the `Report Calculation Currency` line, `Gains-And-Losses Summary` subtitle, `Asset Detail` headings, and `In-Year Activity` subheadings have non-overlapping vertical spacing from preceding content.
12. **Given** PDF output is selected, **When** the affected main-report section transitions are rendered on the same page, **Then** they have at least 12 points of vertical separation from preceding content.
13. **Given** PDF output is selected, **When** the `Gains-And-Losses Summary`, `Rate Source Summary`, `Reference Section`, `Asset Detail: <asset symbol>`, or `In-Year Activity` subheading is rendered on the same page as preceding content, **Then** it has at least 24 points of vertical separation from that content.

---

### User Story 3 - Review Annex 1 Audit Evidence (Priority: P3)

As a user auditing the report, I want Annex 1 to contain detailed per-asset activity evidence and currency conversion evidence so that I can trace the main report totals back to every available activity up to and including the end of the report year.

**Why this priority**: The annex resolves the requested audit report and moves detailed audit evidence out of the main report while preserving traceability.

**Independent Test**: Can be tested by generating a report with multiple assets, historical activity before the report year, report-year activity, post-year activity, liquidations, gains or losses, and currency conversions, then verifying Annex 1 content and placement in both output formats.

**Acceptance Scenarios**:

1. **Given** any successful report generation, **When** the output is reviewed, **Then** Annex 1 is titled `Annex 1 - Audit`.
2. **Given** Annex 1 is rendered, **When** its sections are inspected, **Then** the detailed per-asset audit report is the first section and Currency Conversion Audit is the second section.
3. **Given** the per-asset audit section is rendered, **When** asset activity is inspected, **Then** every activity recorded on or before the report year end is included, every activity after the report year end is excluded, and each included activity shows the activity details, post-activity held quantity, cost-basis effects after the activity, full liquidation events, and gains or losses triggered by the activity.
4. **Given** the Markdown format is selected, **When** the report is generated, **Then** Annex 1 is written as a separate Markdown file whose name preserves the main report filename pattern and inserts `-annex-1-` immediately before the date segment.
5. **Given** the PDF format is selected, **When** the report is generated, **Then** Annex 1 appears in the same PDF file after a page break.
6. **Given** a Currency Conversion Audit row includes quote direction, **When** Annex 1 is rendered, **Then** quote direction is shown as a user-friendly label and does not expose code-style or snake_case values.
7. **Given** PDF output is selected, **When** Annex 1 renders multiple per-asset audit sections on the same page, **Then** each `Asset: <asset symbol>` subheading has sufficient top margin and does not touch or overlap the previous section's table.
8. **Given** a PDF table continues onto another page, **When** the next row or its borders would cross the bottom printable margin, **Then** the renderer advances before drawing that row and preserves complete row content, borders, and blank bottom margin.
9. **Given** a PDF table continues onto another page, **When** its continuation context is rendered, **Then** it uses the exact format `<section or table context> (continued)` and no continuation label is rendered when no table has continued.

### Edge Cases

- If every Gains-And-Losses Summary row has zero Net Gain Or Loss, the section must still render a clear empty-state message instead of an empty table.
- If an asset has historical holdings but no report-year activity, its Asset Detail must show only `Historical Position` with the same closing-position facts that would otherwise be shown in Closing Position.
- If an asset has both report-year activity and historical activity before the report year, its main Asset Detail must keep the normal report-year structure, while Annex 1 includes all activity for that asset on or before the report year end.
- If an asset has activity after the report year end, that later activity must not appear in Annex 1 for the selected report year.
- If a report has no currency conversions, Annex 1 must still include the Currency Conversion Audit section with a clear statement that no converted activity was present.
- If a label mapping is unavailable for conversion status or quote direction, the report must fail before final output rather than exposing internal code labels to the user.
- If PDF generation cannot complete, no incomplete or misleading final PDF report should be presented as successfully generated.
- If a PDF renderer can emit selectable text but only as sequential dumped lines without visible heading hierarchy, styled labels, tables, rows, columns, wrapping, and continuation context, the PDF output is not a valid successful report.
- If PDF tables, headings, or subheadings would otherwise exceed the printable area or collide with preceding content, the PDF renderer must wrap, reflow, add vertical space, or advance the page instead of clipping columns or overlapping text.
- If a PDF table row or its borders would cross the bottom printable margin, the renderer must advance before drawing the row and preserve the required continuation context rather than clipping row content or margins.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow the user to choose PDF or Markdown as the report output format before report generation begins.
- **FR-002**: The system MUST keep Markdown output available as a report output option.
- **FR-003**: The system MUST generate PDF reports on A4-sized pages.
- **FR-004**: The system MUST keep PDF and Markdown main report shared content aligned by preserving the same required output data, explanatory text blocks, table content, and section meanings, with differences limited to PDF pagination, PDF page titles, and Markdown annex file separation.
- **FR-005**: The system MUST start Annex 1 on a new PDF page, MUST allow additional PDF page breaks only before a top-level section, per-asset annex section, table row, or content block that would not fit in the remaining printable page area, and MUST repeat visible section or table context on continuation pages.
- **FR-006**: The system MUST render information classifier labels in the initial report details block in bold. For Markdown output, the initial details list items MUST use the exact classifier label shape `- **Year:**`, `- **Cost Basis Method:**`, `- **Generated At:**`, and `- **Report Calculation Currency:**` before their values.
- **FR-007**: The system MUST omit Gains-And-Losses Summary rows whose Net Gain Or Loss is zero.
- **FR-008**: The system MUST render a clear empty-state message when all Gains-And-Losses Summary rows are omitted because their Net Gain Or Loss is zero.
- **FR-009**: The system MUST render information classifier labels in the Rate Source Summary in bold.
- **FR-010**: The system MUST rename the Reference Section header `Full Liquidation Count Through Year End` to `Historical Full Liquidation Count`.
- **FR-011**: The system MUST render `Historical Position` for assets with no activity registered during the report year.
- **FR-012**: For assets with no activity registered during the report year, the system MUST omit the separate `Opening Position`, `In-Year Activity`, and `Closing Position` subsections and show the same facts from Closing Position under `Historical Position`.
- **FR-013**: The system MUST render In-Year Activity conversion statuses using only the allowed user-facing labels `Same currency` for `same_currency` and `Converted` for `converted`, and MUST NOT expose code-style or snake_case values.
- **FR-014**: The system MUST render zero-priced SELL activities in the In-Year Activity Type column as `BLOCKCHAIN OP`.
- **FR-015**: The system MUST render Currency Conversion Audit quote directions using only the allowed user-facing labels `Source currency per base currency` for `source_per_base` and `Base currency per source currency` for `base_per_source`, and MUST NOT expose code-style or snake_case values.
- **FR-016**: The system MUST add Annex 1 with the title `Annex 1 - Audit` to every successful capital gains and losses report output.
- **FR-017**: Annex 1 MUST contain the detailed per-asset audit report as its first section.
- **FR-018**: The detailed per-asset audit report MUST list all activity for each reported asset from the beginning of available history through the end of the selected report year, including report-year activity, including assets that appear only in the Reference Section, including assets whose zero net summary rows are hidden from the visible summary, and excluding activity after the report year end.
- **FR-019**: Each per-asset audit activity row MUST disclose activity date or timestamp, non-secret source activity reference, activity type, quantity, applicable unit price, gross value, fee, original activity currency, calculation currency, held quantity after the activity, open cost basis after the activity, any allocated basis, net liquidation proceeds, full liquidation event status, gains or losses from that activity, conversion status when applicable, and sanitized note text when present.
- **FR-020**: Annex 1 MUST contain Currency Conversion Audit as its second section, and the main report MUST NOT include detailed Currency Conversion Audit rows.
- **FR-021**: When Markdown output is selected, the system MUST write Annex 1 as a separate Markdown file whose name preserves the main report filename pattern and inserts `-annex-1-` immediately before the date segment.
- **FR-022**: When PDF output is selected, the system MUST include Annex 1 in the same PDF file after a page break.
- **FR-023**: When PDF output is selected, the system MUST preserve the main report filename pattern and use the `.pdf` file extension.
- **FR-024**: PDF output MUST be text-based and searchable, with generated report text selectable by PDF readers that support text selection.
- **FR-025**: If report generation fails before final output completion, the system MUST not present a partial report file as successfully generated.
- **FR-026**: Successful report generation MUST communicate all generated output files to the user, including the separate Markdown annex file when Markdown is selected.
- **FR-027**: Generated reports and report-generation failure messages MUST NOT include Ghostfolio tokens, security tokens, bearer tokens, reusable authentication material, or other secrets.
- **FR-028**: PDF generation and report rendering MUST run locally and MUST NOT send report data, financial data, tokens, or generated report files to any remote storage, telemetry destination, or external document-generation service as part of this feature.
- **FR-029**: PDF generation MUST work on supported Linux, macOS, and Windows installations without requiring platform-specific font paths, user-installed fonts, a browser, or operating-system print-to-PDF support; required report text MUST use application-supplied local font data.
- **FR-030**: When PDF output is selected, the system MUST render report-domain content through PDF-specific layout and MUST NOT use Markdown-rendered content or Markdown structural syntax, including heading markers, table pipes or separators, or bold markers, as the PDF body.
- **FR-031**: When PDF output is selected, the system MUST use `github.com/signintech/gopdf` layout primitives for A4 page creation, application-supplied font loading, headings, styled text, table headers, table rows, table columns, wrapped cell content, and continuation context; a plain line-dump renderer is not a valid PDF implementation.
- **FR-032**: When PDF output is selected, the system MUST generate every page using landscape A4 orientation.
- **FR-033**: When PDF output is selected, the system MUST keep table columns inside the printable page area with visible right padding, no right-edge clipping, and wrapped cell content where values exceed the column width.
- **FR-034**: When PDF output is selected, the system MUST maintain non-overlapping vertical spacing between adjacent text blocks, headings, subheadings, and tables, including the `Report Calculation Currency` line, `Gains-And-Losses Summary` subtitle, `Asset Detail` headings, `In-Year Activity` subheadings, and Annex 1 per-asset subheadings.
- **FR-035**: When PDF output is selected, the system MUST render `Overall Yearly Net Total` as the final row or footer inside the Gains-And-Losses Summary table.
- **FR-036**: When PDF output is selected, the system MUST render the Rate Source Summary as bold classifier label lines followed by non-bold values and MUST NOT render it as a table titled `Rate Source Summary Table`.
- **FR-037**: When PDF output is selected, the system MUST NOT introduce generated helper subheadings that are not part of the report presentation contract, including `Reference Table` under the Reference Section.
- **FR-038**: Before drawing a PDF table row or its borders, the system MUST determine whether the complete row fits inside the remaining printable height while preserving the bottom margin; otherwise it MUST advance the row to a continuation page with visible table or section context and MUST NOT clip row text, cells, borders, or the bottom margin.
- **FR-039**: When PDF output is selected, the system MUST size each table to use the available landscape printable width with equal left and right outer margins while retaining the padding, wrapping, and no-clipping requirements of FR-033.
- **FR-040**: When PDF output is selected, the system MUST maintain at least 12 points of vertical separation at the affected transitions covered by FR-034.
- **FR-041**: When PDF output is selected, the system MUST maintain at least 24 points of vertical separation before the `Gains-And-Losses Summary`, `Rate Source Summary`, `Reference Section`, `Asset Detail: <asset symbol>`, and `In-Year Activity` subheadings when they follow preceding content on the same page.
- **FR-042**: When a PDF table continues onto a new page, the system MUST render continuation context only on that continuation page, using the exact format `<section or table context> (continued)` without a `Continued: ` prefix; it MUST NOT render a continuation label when no table has continued.

### Financial Calculation Evidence *(include when feature affects financial calculations)*

- **Numeric Representation**: This feature changes report output format, presentation, and audit disclosure. It must preserve the existing exact-decimal financial values and explicit currency identities already used by the report.
- **Conversion And Rounding**: This feature does not authorize new conversion sources, conversion boundaries, or rounding rules. Currency conversion evidence must continue to follow the active report base-currency conversion specification.
- **Empirical Solidified Financial Tests**: Existing empirical financial tests remain applicable as regression evidence for capital gains and losses calculations. This feature adds presentation and audit traceability coverage around those results.
- **Empirical External Dataset Changes**: The empirical external dataset remains read-only for this feature.

### Security, Persistence, And Integration Evidence

- **Persistence Impact**: This feature creates user-requested report export files only. These exports are outside the application-managed persistence boundary because the application does not store them as cache/state, manage their lifecycle after generation, or re-ingest them automatically. It does not add synced-data persistence, protected snapshot persistence, remote persistence, telemetry, or background storage of generated reports.
- **Token Handling Impact**: Ghostfolio tokens and security tokens remain runtime-only secrets and must not appear in generated reports, annexes, errors, diagnostics, examples, or fixtures.
- **External Integration Impact**: This feature does not require a new external data provider or external document-generation service. PDF generation must run locally; any optional local dependency choice for producing PDF files must be justified during planning before implementation.
- **Security Review Scope**: The feature must be reviewed for secret disclosure in cleartext reports, local report-file handling, path or filename safety, output-generation failure handling, dependency risk if any PDF dependency is proposed, and injection risk in rendered report content.

### Quality Gate Evidence *(mandatory)*

- **Changed Source Inputs**: Source changes are expected for report output selection, report rendering, audit annex generation, related tests, and dependency pinning in `go.mod` and `go.sum` for local PDF/font support.
- **Quality Gate Command**: `make quality QUALITY_BASE_REF=<base-ref>` must pass locally or through the `Quality` GitHub Actions check.
- **No-Source-Change Behavior**: Not expected to apply because source inputs are expected to change; if planning later removes all source changes, the quality gate must still pass with explicit skip messages.

### Key Entities *(include if feature involves data)*

- **Report Output Format Selection**: The user's selected output format for a report generation run, either PDF or Markdown.
- **Main Capital Gains And Losses Report**: The primary report containing selected inputs, summaries, reference sections, asset details, and other non-annex report content.
- **PDF Report Output**: The landscape A4 paged report file that contains the main report and Annex 1 in one file.
- **Markdown Report Output**: The Markdown main report file generated for the selected report inputs.
- **Reported Asset**: An asset identity selected by the existing report inclusion or reference-section rules for the selected year. This includes assets in Asset Detail, assets represented in the gains-and-losses summary before zero-net presentation filtering, and assets that appear only in the Reference Section. It excludes synced assets that are not selected by those rules for the generated report.
- **Audit Annex**: Annex 1 of the report, titled `Annex 1 - Audit`, containing per-asset audit evidence followed by Currency Conversion Audit.
- **Per-Asset Audit Section**: A section of Annex 1 that groups all activity evidence for one asset from the beginning of available history through the end of the selected report year.
- **Audit Activity Entry**: One historical activity record in the annex, including activity details, held quantity after the activity, cost-basis effects, full liquidation events, and gains or losses.
- **Currency Conversion Audit Section**: The annex section that discloses conversion evidence for converted activities.
- **Historical Position Section**: A condensed Asset Detail presentation used when an asset has no activity registered during the report year.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can select PDF or Markdown and start report generation in no more than 30 seconds after reaching the report generation step.
- **SC-002**: For identical report inputs, PDF and Markdown main reports contain 100% of the same required shared report sections, table columns, text blocks, and values, excluding only pagination, page titles, and Markdown annex file separation.
- **SC-003**: For Markdown output, every successful generation produces exactly one main report file and exactly one Annex 1 file; for PDF output, every successful generation produces exactly one PDF file containing both the main report and Annex 1.
- **SC-004**: In test reports containing assets without report-year activity, 100% of those assets render as `Historical Position` and omit `Opening Position`, `In-Year Activity`, and `Closing Position` from the main Asset Detail.
- **SC-005**: In test reports containing zero and non-zero Net Gain Or Loss summary rows, 100% of zero rows are omitted and 100% of non-zero rows remain visible.
- **SC-006**: In generated reports, 100% of visible conversion status values in the main report and quote direction values in Annex 1 use the allowed labels `Same currency`, `Converted`, `Source currency per base currency`, or `Base currency per source currency` with no snake_case or internal code-style values.
- **SC-007**: In generated reports, 100% of zero-priced SELL activities shown in In-Year Activity use `BLOCKCHAIN OP` as the Type value.
- **SC-008**: Annex 1 allows a reviewer to trace 100% of each reported asset's activities on or before the report year end, including reference-only reported assets, to post-activity held quantity, cost-basis effect, full liquidation status, and gain or loss effect, while excluding activities after the report year end.
- **SC-009**: PDF output generation and Annex 1 rendering support the existing 10,000 cached-activity report scale and do not introduce a lower activity-count limit than Markdown output.
- **SC-010**: In generated PDF reports, 100% of required report text is emitted as selectable text rather than rasterized page images.
- **SC-011**: In generated PDF reports, 0 Markdown structural syntax markers are visible as report presentation for headings, tables, emphasis, or Annex 1 sections.
- **SC-012**: PDF layout verification confirms required report samples render with visible heading hierarchy, styled classifier labels, table headers, table rows, table columns, wrapped cell content, and continuation context rather than as sequential dumped lines.
- **SC-013**: PDF layout verification confirms required report samples use landscape A4 pages, keep all table columns within the printable area without right-edge clipping, avoid overlapping adjacent text sections, place `Overall Yearly Net Total` inside the Gains-And-Losses Summary table, render Rate Source Summary as label/value lines, omit the extra `Reference Table` subheading, and preserve top margin before main-report and Annex 1 asset subheadings.
- **SC-014**: PDF layout verification confirms required wide-table samples use the available landscape printable width with equal left and right margins, affected section transitions have at least 12 points of vertical separation, and every continued table row and border remains wholly inside the printable area with a preserved bottom margin and visible continuation context.
- **SC-015**: PDF layout verification confirms the named main-report subheadings have at least 24 points of vertical separation, actual table continuation pages use `<section or table context> (continued)`, and unsplit table samples contain no continuation label.

## Assumptions

- The target users are the same users who currently generate capital gains and losses reports from synced Ghostfolio activity data.
- The Markdown main report remains the existing default-compatible report format, with only the requested presentation and annex changes.
- Annex 1 is generated for every successful report so the output contract is predictable across report formats.
- If an annex section has no rows, it shows an explicit empty-state message instead of being omitted.
- PDF pagination may differ from Markdown structure, but report data and section meaning remain aligned.
- This feature does not change which activities are included in the report or how gains and losses are calculated.
- Existing currency conversion rules from the active base-currency conversion feature remain authoritative.
- Existing empirical external datasets remain unchanged and read-only.
