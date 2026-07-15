# Contract: Final Report Rendering

## Scope

This contract refines the externally visible Markdown and PDF report contract
for final release presentation. It supersedes the blanket canonical visible
decimal rule in `specs/008-report-pdf-annex/contracts/report-rendering.md` only
for currency-denominated amounts and unit prices. Existing section order,
calculation rules, quantity formatting, exchange-rate precision, output bundle
shape, PDF layout requirements, and security rules remain in force.

## Main Report Warning

The main report must contain this exact sentence once:

```text
The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.
```

Rules:

- It appears immediately after the initial `Report Calculation Currency` field.
- It appears immediately before `Gains-And-Losses Summary`.
- It is a standalone paragraph.
- The complete sentence, including its final period, is bold.
- The Markdown main document uses exactly:

  ```markdown
  **The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.**
  ```

- The combined PDF uses a fully bold text operation without visible Markdown
  markers.
- The separate Markdown Annex does not repeat the warning.

## Financial Display Contract

Every present currency-denominated amount and unit price has exactly two digits
after the decimal separator in both formats.

Included value classes:

| Report area | Financial values |
|-------------|------------------|
| Gains-And-Losses Summary | Per-asset net gain or loss; overall yearly net total |
| Position blocks | Opening, closing, and historical cost basis |
| In-Year Activity | Unit price, gross value, fee, basis after row |
| Liquidation Calculations | Allocated basis, net proceeds, gain or loss |
| Detailed Per-Asset Audit Report | Unit price, gross value, fee, basis after activity, allocated basis, net proceeds, gain or loss |
| Currency Conversion Audit | Original and converted unit price, gross value, and fee amount entries |

Rules:

- Use exact-decimal HALF UP rounding at two places.
- HALF UP applies symmetrically, so an exact negative tie rounds away from zero.
- Round only while deriving final visible strings.
- Do not mutate calculated decimals or use visible strings in later calculation,
  conversion, comparison, omission, storage, or totals.
- Retain the existing explicit currency identity or currency column for each
  value.
- A present whole or one-place value receives trailing zeros.
- A present exact zero renders `0.00`.
- A negative value that rounds to zero renders `0.00`, not `-0.00`.
- A missing optional value remains blank.
- A small exact non-zero value that renders `0.00` remains included wherever the
  exact value was included before this feature.

Required examples:

| Exact value | Visible value |
|-------------|---------------|
| `1` | `1.00` |
| `1.004` | `1.00` |
| `1.005` | `1.01` |
| `-1.005` | `-1.01` |
| `-0.004` | `0.00` |

## Values Excluded From Two-Decimal Formatting

Quantity values retain their established exact canonical representation.

Examples:

| Exact quantity | Visible quantity |
|----------------|------------------|
| `2` | `2` |
| `0.1` | `0.1` |
| `0.00000001` | `0.00000001` |

Disclosed `Rate Value` exchange-rate ratios also retain established
provider-published precision. For example, `1.0946` remains `1.0946`, not
`1.09`.

The following decisions continue to use exact values, not displayed values:

- non-zero summary row inclusion
- zero-to-zero converted component omission
- zero-priced holding-reduction classification
- cost-basis allocation, conversion, gains, losses, and totals

## Structured Boolean Contract

Every structured boolean exposed as a report value renders as:

| Source value | Visible label |
|--------------|---------------|
| `true` | `Yes` |
| `false` | `No` |

The current `Full Liquidation Event` Annex field must follow this mapping in
both formats. Renderers must not expose lowercase `true` or `false` for that
field. Arbitrary notes and explanatory text are not boolean fields and are not
rewritten.

## Original Activity Currency Contract

For a Detailed Per-Asset Audit Report row classified from exact source data as a
zero-priced holding reduction:

- The calculated audit model retains its existing pre-format `ActivityCurrency`
  value and the exact zero-priced-reduction classification before presentation;
  the feature does not invent source provenance when no source currency context
  was selected.
- `Original Activity Currency` is blank.
- `Calculation Currency` remains populated.
- Activity type, quantity, quantity after activity, basis after activity,
  liquidation evidence, gain or loss, conversion status, and note remain
  unchanged and visible when otherwise applicable.

For an activity with an applicable non-zero source price,
`Original Activity Currency` retains the selected source activity currency.

No renderer may infer this rule from a two-decimal `Unit Price` string.

## Converted Amounts Contract

Each included conversion component uses this logical entry syntax:

```text
<field label>: <original amount> -> <converted amount>
```

Rules:

- Labels and order remain `unit_price`, `gross_value`, `fee_amount`.
- A valid conversion audit entry contains no duplicate amount kind and uses only
  the canonical subsequence of that order.
- There is exactly one ordinary space after `:`.
- There is exactly one ordinary space on each side of `->`.
- Original and converted amounts follow the two-decimal financial contract.
- A component is omitted only when its exact original and converted amounts are
  both zero.
- Every included entry begins on a separate visible line in the existing
  `Converted Amounts` cell.
- Every non-final entry ends with `;` followed by a format-appropriate visible
  line break.
- The final entry has no trailing semicolon.
- A single entry occupies one line and has no semicolon.
- No included entries preserves the existing empty-cell behavior.

Required three-entry visible content:

```text
unit_price: 30754.70 -> 28673.04;
gross_value: 254.76 -> 237.52;
fee_amount: 1.79 -> 1.67
```

Markdown encoding inside the pipe-table cell is:

```markdown
unit_price: 30754.70 -> 28673.04;<br>gross_value: 254.76 -> 237.52;<br>fee_amount: 1.79 -> 1.67
```

PDF encoding uses explicit newlines in the table-cell value. PDF measurement
must apply the same indicator-sensitive word-wrap option as table drawing and
must account for those newlines before complete-row page preflight. Drawing must
not clip or overlap subsequent rows, borders, or the bottom printable margin.

## Markdown Contract

- Main and Annex files remain separate.
- The warning appears only in the main file.
- Financial values in both files follow this contract.
- Within the generated Converted Amounts cell, each dynamic label and amount
  component is sanitized and HTML-sensitive characters are escaped before the
  renderer assembles the fixed literal `: ` and ` -> ` syntax and adds
  controlled `<br>` line breaks. No dynamic value can introduce that delimiter.
- A Converted Amounts line break must not terminate or add a Markdown table row.
- Existing headings, columns, empty states, and section order remain unchanged.

## PDF Contract

- Main report and Annex 1 remain in one combined PDF.
- The warning is fully bold and contains no Markdown syntax.
- Financial values in the main report and Annex follow this contract.
- Converted Amounts entries use explicit cell line boundaries with
  drawing-equivalent word-wrap and newline measurement.
- Existing landscape A4 sizing, application-supplied embedded fonts,
  searchable/selectable text, printable-width tables, heading spacing, Annex
  page break, repeated continuation context, and complete-row preflight remain
  unchanged.
- Existing arbitrary report text stays single-line sanitized. Only controlled
  Converted Amounts cell boundaries are preserved as newlines.

## Cross-Format Parity

For identical report inputs, Markdown and PDF must agree on:

- warning text and semantic placement
- every displayed financial value
- every quantity value
- every disclosed exchange-rate value
- every structured boolean label
- original and calculation currency applicability
- converted amount labels, values, order, and entry boundaries

Permitted differences remain limited to Markdown syntax, PDF styling and
pagination, PDF page titles, and separate Markdown Annex output.

## Failure Contract

- A non-finite or otherwise unrenderable decimal returns a contextual render
  error and must not create a successful output result.
- A PDF measurement or drawing failure returns an error before successful
  output is reported.
- This feature does not change the existing output writer's reservation and
  cleanup sequence.
- Errors and generated documents must not expose tokens, protected payload
  bytes, or reusable authentication material.

## Automated Evidence

Automated tests must cover both formats and prove:

- exact warning text, one occurrence, and placement
- generated-PDF text runs covering the complete warning, including the final
  period, all use the embedded bold font
- whole, one-place, two-place, high-precision, positive and negative exact-half,
  very small, very large, zero, negative-zero, and absent financial cases
- unchanged quantities, exchange rates, calculated values, and exact inclusion
  decisions
- both boolean labels
- the existing calculated audit `ActivityCurrency` value remains unchanged
  before a zero-priced presentation cell becomes blank
- zero-priced blank and non-zero-priced retained visible activity currencies
- zero, one, two, and three included converted-entry cases
- duplicate and out-of-order converted amount kinds fail report validation
- exact colon, arrow, semicolon, order, and line-boundary behavior
- generated-PDF coordinates show each converted entry starts on a later visible
  line
- PDF measured and drawn line counts, row height, and bottom-margin preflight
  agree for explicit newlines and long space-wrapped content
- unchanged 10,000-activity selected-format performance limits
