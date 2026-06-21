# Contract: Report Base Currency TUI Workflows

## Scope

This contract extends the existing `Sync and Reports` TUI report-generation workflow with required report base-currency selection.

## Global UX Rules

- The application remains terminal-native, full-screen, and keyboard-driven.
- Report base-currency selection is transient and applies only to the active report run.
- Report content is not previewed in the TUI before final save.
- Provider lookup and report calculation run asynchronously and keep the Bubble Tea event loop responsive.
- User-visible errors must be actionable and must not expose Ghostfolio tokens, JWTs, raw protected payloads, or unredacted production diagnostic financial values.

## Report Selection Screen

Entry conditions:

- User selected available `Generate Capital Gains Report` inside an unlocked `Sync and Reports` context.
- A protected activity cache exists with at least one reportable year.

Visible content:

- selected year list containing only years present in the protected activity cache
- selected cost basis method list containing exactly the supported methods
- report base-currency list containing exactly `USD` and `EUR`
- plain-language explanation for the highlighted or selected method
- short explanation that all monetary report calculations and totals will use the selected base currency
- primary action menu

Supported base currencies:

- `USD`
- `EUR`

Primary menu items:

- `Generate Report`
- `Back`

Rules:

- Year selection must be constrained to `available_report_years`.
- Method selection must be constrained to the supported method list.
- Base-currency selection must be constrained to `USD` and `EUR`.
- Report generation cannot start until year, method, and base currency are all selected.
- Changing the selected base currency changes only the pending report request, not synced data and not any persisted setup.
- `Back` returns to the unlocked context without clearing the token.

Success transitions:

- `Generate Report` -> `Report Generation Busy Screen`
- `Back` -> `Sync and Reports Menu Screen`

## Report Generation Busy Screen

Entry conditions:

- User confirmed year, method, and base currency.

Visible content:

- non-secret busy message
- selected year
- selected cost basis method
- selected report base currency

Rules:

- Calculation uses the currently unlocked protected cache and does not run a new sync.
- Provider lookup uses only the fixed provider selected by report base currency.
- The UI must not show cleartext report content as a preview.
- On conversion, calculation, render, or save failure before final save, the workflow reports an actionable non-secret error and removes any partial cleartext output created by the attempt.
- On save success and automatic-open failure, the workflow treats the save as successful and reports the open warning.

Success transitions:

- success or failure -> `Report Result Screen`

## Report Result Screen

Entry conditions:

- Report generation completed, failed, or saved with an automatic-open warning.

Visible content:

- success, failure, or success-with-warning status
- selected year
- selected cost basis method
- selected report base currency
- saved Markdown path on successful save
- actionable failure message on failure
- diagnostic availability message when existing diagnostic policy makes one available
- primary action menu

Rules:

- Successful outcomes show the selected report base currency used for report calculations.
- Conversion failures identify source currency, report base currency, and activity date when known.
- Failure messages must not expose Ghostfolio token material or raw protected payload data.
- Returning to report selection preserves the unlocked context but creates a new report request.
- Returning to the main menu clears the token through the existing Sync and Reports context rules.

Success transitions:

- `Generate Another Report` -> `Report Selection Screen`
- `Back To Sync and Reports` -> `Sync and Reports Menu Screen`
- `Back To Main Menu` -> `Main Menu Screen`
