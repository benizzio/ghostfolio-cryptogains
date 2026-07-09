# Contract: Report Output Format TUI Workflows

## Scope

This contract extends the existing `Sync and Reports` report-generation workflow with output-format selection and result reporting for Markdown and PDF outputs.

## Global UX Rules

- The application remains terminal-native, full-screen, and keyboard-driven.
- Output-format selection is transient and applies only to the active report run.
- Report content is not previewed in the TUI before final save.
- Report generation remains asynchronous and keeps the Bubble Tea event loop responsive.
- User-visible errors must be actionable and must not expose Ghostfolio tokens, JWTs, raw protected payload data, or unredacted production diagnostic financial values.

## Report Selection Screen

Entry conditions:

- User selected available `Generate Capital Gains Report` inside an unlocked `Sync and Reports` context.
- A protected activity cache exists with at least one reportable year.

Visible content:

- selected year list containing only years present in the protected activity cache
- selected cost-basis method list containing exactly the supported methods
- report base-currency list containing exactly `USD` and `EUR`
- output-format list containing exactly `Markdown` and `PDF`
- plain-language explanation for the highlighted or selected method
- short explanation that all monetary report calculations and totals use the selected base currency
- short explanation of the selected output format
- primary action menu

Primary menu items:

- `Generate Report`
- `Back`

Rules:

- Year selection is constrained to `available_report_years`.
- Method selection is constrained to the supported method list.
- Base-currency selection is constrained to `USD` and `EUR`.
- Output-format selection is constrained to `Markdown` and `PDF`.
- Report generation cannot start until year, method, base currency, and output format are all selected.
- Changing output format changes only the pending report request, not synced data and not persisted setup.
- `Back` returns to the unlocked context without clearing the token.

Success transitions:

- `Generate Report` -> `Report Generation Busy Screen`
- `Back` -> `Sync and Reports Menu Screen`

## Report Generation Busy Screen

Entry conditions:

- User confirmed year, method, base currency, and output format.

Visible content:

- non-secret busy message
- selected year
- selected cost-basis method
- selected report base currency
- selected output format

Rules:

- Calculation uses the currently unlocked protected cache and does not run a new sync.
- Markdown output renders a main Markdown document plus an Annex 1 Markdown document.
- PDF output renders one local landscape A4 text PDF containing the main report and Annex 1.
- The UI must not show cleartext report content as a preview.
- On calculation, render, or save failure before final save, the workflow reports an actionable non-secret error and removes any partial cleartext output created by the attempt.
- On save success and automatic-open failure, the workflow treats the save as successful and reports the open warning.

Success transitions:

- success, failure, or success-with-warning -> `Report Result Screen`

## Report Result Screen

Entry conditions:

- Report generation completed, failed, or saved with an automatic-open warning.

Visible content:

- success, failure, or success-with-warning status
- selected year
- selected cost-basis method
- selected report base currency
- selected output format
- all saved output paths on successful save
- actionable failure message on failure
- diagnostic availability message when existing diagnostic policy makes one available
- primary action menu

Rules:

- Successful Markdown outcomes show both the saved main Markdown path and the saved Annex 1 Markdown path.
- Successful PDF outcomes show the saved PDF path.
- Failure messages must not expose Ghostfolio token material or raw protected payload data.
- Returning to report selection preserves the unlocked context but creates a new report request.
- Returning to the main menu clears the token through the existing Sync and Reports context rules.

Success transitions:

- `Generate Another Report` -> `Report Selection Screen`
- `Back To Sync and Reports` -> `Sync and Reports Menu Screen`
- `Back To Main Menu` -> `Main Menu Screen`
