# Feature Specification: Capital Gains Report PDF And Audit Annex

**Feature Branch**: `[008-report-pdf-annex]`

**Created**: 2026-07-02

**Status**: Draft

**Input**: User description: "Specify GitHub issue #39, Capital Gains And Losses Report - new format and fixes, including GitHub issue #34, Audit report."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Choose Report Output Format (Priority: P1)

As a user generating a capital gains and losses report, I want to choose whether the report is produced as PDF or Markdown before generation so that I can use the same report data in the format required by my review or filing workflow.

**Why this priority**: PDF output is the primary new capability requested, while the existing Markdown output must remain available.

**Independent Test**: Can be fully tested by generating the same report once as PDF and once as Markdown, then confirming that both outputs are available and contain the same required report data, text blocks, and table content except for format-specific pagination and file-splitting behavior.

**Acceptance Scenarios**:

1. **Given** the user has completed report setup and selected valid report inputs, **When** the user selects PDF before generation, **Then** the report is generated as an A4 PDF file.
2. **Given** the user has completed report setup and selected valid report inputs, **When** the user selects Markdown before generation, **Then** the report is generated in the existing Markdown format.
3. **Given** the same report inputs are used for PDF and Markdown, **When** both reports are reviewed, **Then** shared report sections contain the same output data, explanatory text, and table content, with only format-specific page breaks, page titles, and annex placement differing.

---

### User Story 2 - Read A Clearer Main Report (Priority: P2)

As a user reviewing the main capital gains and losses report, I want labels and sections to be easier to scan and unnecessary zero-value information to be removed so that the report is shorter and clearer without losing relevant financial evidence.

**Why this priority**: The requested fixes reduce report bloat and make existing report information easier to interpret.

**Independent Test**: Can be tested by generating reports that include zero net gain or loss summary rows, rate-source disclosures, assets with and without report-year activity, zero-priced sell activities, and currency conversion statuses, then verifying the rendered report content.

**Acceptance Scenarios**:

1. **Given** the main report has an initial details block with values such as Year and Cost Basis Method, **When** the report is rendered, **Then** each information classifier label in that block is bold.
2. **Given** the Gains-And-Losses Summary includes rows whose Net Gain Or Loss is zero, **When** the report is rendered, **Then** those zero net gain or loss rows are omitted and non-zero rows remain.
3. **Given** the report includes a Rate Source Summary, **When** the section is rendered, **Then** each information classifier label in that section is bold.
4. **Given** the Reference Section includes the full-liquidation count disclosure, **When** the report is rendered, **Then** the header reads `Historical Full Liquidation Count`.
5. **Given** an asset has no activity registered during the report year, **When** the Asset Detail section is rendered, **Then** the report shows the closing-position information under the title `Historical Position` and omits the separate `Opening Position`, `In-Year Activity`, and `Closing Position` subsections for that asset.
6. **Given** an In-Year Activity row includes a conversion status, **When** the row is rendered, **Then** the status is shown as a user-friendly label and does not expose code-style or snake_case values.
7. **Given** an In-Year Activity row represents a zero-priced SELL activity, **When** the row is rendered, **Then** the Type value is `BLOCKCHAIN OP` instead of `SELL`.
8. **Given** a Currency Conversion Audit row includes quote direction, **When** the row is rendered, **Then** quote direction is shown as a user-friendly label and does not expose code-style or snake_case values.

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

### Edge Cases

- If every Gains-And-Losses Summary row has zero Net Gain Or Loss, the section must still render a clear empty-state message instead of an empty table.
- If an asset has historical holdings but no report-year activity, its Asset Detail must show only `Historical Position` with the same closing-position facts that would otherwise be shown in Closing Position.
- If an asset has both report-year activity and historical activity before the report year, its main Asset Detail must keep the normal report-year structure, while Annex 1 includes all activity for that asset on or before the report year end.
- If an asset has activity after the report year end, that later activity must not appear in Annex 1 for the selected report year.
- If a report has no currency conversions, Annex 1 must still include the Currency Conversion Audit section with a clear statement that no converted activity was present.
- If a label mapping is unavailable for conversion status or quote direction, the report must fail before final output rather than exposing internal code labels to the user.
- If PDF generation cannot complete, no incomplete or misleading final PDF report should be presented as successfully generated.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow the user to choose PDF or Markdown as the report output format before report generation begins.
- **FR-002**: The system MUST keep Markdown output available as a report output option.
- **FR-003**: The system MUST generate PDF reports on A4-sized pages.
- **FR-004**: The system MUST keep the PDF and Markdown main report layouts as close as practical while preserving the same output data, explanatory text blocks, table content, and section intent.
- **FR-005**: The system MUST allow PDF-specific page breaks and page titles when needed for readable multi-page reports.
- **FR-006**: The system MUST render information classifier labels in the initial report details block in bold.
- **FR-007**: The system MUST omit Gains-And-Losses Summary rows whose Net Gain Or Loss is zero.
- **FR-008**: The system MUST render a clear empty-state message when all Gains-And-Losses Summary rows are omitted because their Net Gain Or Loss is zero.
- **FR-009**: The system MUST render information classifier labels in the Rate Source Summary in bold.
- **FR-010**: The system MUST rename the Reference Section header `Full Liquidation Count Through Year End` to `Historical Full Liquidation Count`.
- **FR-011**: The system MUST render `Historical Position` for assets with no activity registered during the report year.
- **FR-012**: For assets with no activity registered during the report year, the system MUST omit the separate `Opening Position`, `In-Year Activity`, and `Closing Position` subsections and show the same facts from Closing Position under `Historical Position`.
- **FR-013**: The system MUST render In-Year Activity conversion statuses as user-friendly labels and MUST NOT expose code-style or snake_case values.
- **FR-014**: The system MUST render zero-priced SELL activities in the In-Year Activity Type column as `BLOCKCHAIN OP`.
- **FR-015**: The system MUST render Currency Conversion Audit quote directions as user-friendly labels and MUST NOT expose code-style or snake_case values.
- **FR-016**: The system MUST add Annex 1 with the title `Annex 1 - Audit` to every successful capital gains and losses report output.
- **FR-017**: Annex 1 MUST contain the detailed per-asset audit report as its first section.
- **FR-018**: The detailed per-asset audit report MUST list all activity for each reported asset from the beginning of available history through the end of the selected report year, including report-year activity and excluding activity after the report year end.
- **FR-019**: Each per-asset audit activity row MUST disclose activity details, held quantity after the activity, cost-basis effects after the activity, any full liquidation event, and gains or losses from that activity.
- **FR-020**: Annex 1 MUST contain Currency Conversion Audit as its second section.
- **FR-021**: When Markdown output is selected, the system MUST write Annex 1 as a separate Markdown file whose name preserves the main report filename pattern and inserts `-annex-1-` immediately before the date segment.
- **FR-022**: When PDF output is selected, the system MUST include Annex 1 in the same PDF file after a page break.
- **FR-023**: If report generation fails before final output completion, the system MUST not present a partial report file as successfully generated.
- **FR-024**: Successful report generation MUST communicate all generated output files to the user, including the separate Markdown annex file when Markdown is selected.
- **FR-025**: Generated reports and report-generation failure messages MUST NOT include Ghostfolio tokens, security tokens, bearer tokens, reusable authentication material, or other secrets.
- **FR-026**: The system MUST NOT send report data, financial data, tokens, or generated report files to any remote storage, telemetry destination, or external document-generation service as part of this feature.

### Financial Calculation Evidence *(include when feature affects financial calculations)*

- **Numeric Representation**: This feature changes report output format, presentation, and audit disclosure. It must preserve the existing exact-decimal financial values and explicit currency identities already used by the report.
- **Conversion And Rounding**: This feature does not authorize new conversion sources, conversion boundaries, or rounding rules. Currency conversion evidence must continue to follow the active report base-currency conversion specification.
- **Empirical Solidified Financial Tests**: Existing empirical financial tests remain applicable as regression evidence for capital gains and losses calculations. This feature adds presentation and audit traceability coverage around those results.
- **Empirical External Dataset Changes**: The empirical external dataset remains read-only for this feature.

### Security, Persistence, And Integration Evidence

- **Persistence Impact**: This feature creates user-requested report output files only. It does not add synced-data persistence, protected snapshot persistence, remote persistence, or background storage of generated reports.
- **Token Handling Impact**: Ghostfolio tokens and security tokens remain runtime-only secrets and must not appear in generated reports, annexes, errors, diagnostics, examples, or fixtures.
- **External Integration Impact**: This feature does not require a new external data provider or external document-generation service. Any optional dependency choice for producing PDF files must be justified during planning before implementation.
- **Security Review Scope**: The feature must be reviewed for secret disclosure in cleartext reports, local report-file handling, path or filename safety, output-generation failure handling, dependency risk if any PDF dependency is proposed, and injection risk in rendered report content.

### Quality Gate Evidence *(mandatory)*

- **Changed Source Inputs**: Source changes are expected for report output selection, report rendering, audit annex generation, and related tests. No dependency-file changes are assumed by this specification.
- **Quality Gate Command**: `make quality QUALITY_BASE_REF=<base-ref>` must pass locally or through the `Quality` GitHub Actions check.
- **No-Source-Change Behavior**: Not expected to apply because source inputs are expected to change; if planning later removes all source changes, the quality gate must still pass with explicit skip messages.

### Key Entities *(include if feature involves data)*

- **Report Output Format Selection**: The user's selected output format for a report generation run, either PDF or Markdown.
- **Main Capital Gains And Losses Report**: The primary report containing selected inputs, summaries, reference sections, asset details, and other non-annex report content.
- **PDF Report Output**: The A4 paged report file that contains the main report and Annex 1 in one file.
- **Markdown Report Output**: The Markdown main report file generated for the selected report inputs.
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
- **SC-006**: In generated reports, 100% of visible conversion status and quote direction values use user-friendly labels with no snake_case or internal code-style values.
- **SC-007**: In generated reports, 100% of zero-priced SELL activities shown in In-Year Activity use `BLOCKCHAIN OP` as the Type value.
- **SC-008**: Annex 1 allows a reviewer to trace 100% of each reported asset's activities on or before the report year end to post-activity held quantity, cost-basis effect, full liquidation status, and gain or loss effect, while excluding activities after the report year end.

## Assumptions

- The target users are the same users who currently generate capital gains and losses reports from synced Ghostfolio activity data.
- The Markdown main report remains the existing default-compatible report format, with only the requested presentation and annex changes.
- Annex 1 is generated for every successful report so the output contract is predictable across report formats.
- If an annex section has no rows, it shows an explicit empty-state message instead of being omitted.
- PDF pagination may differ from Markdown structure, but report data and section meaning remain aligned.
- This feature does not change which activities are included in the report or how gains and losses are calculated.
- Existing currency conversion rules from the active base-currency conversion feature remain authoritative.
- Existing empirical external datasets remain unchanged and read-only.
