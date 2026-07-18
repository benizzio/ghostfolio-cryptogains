# Contract: Final Report Rendering

## Scope

This contract refines the externally visible Markdown and PDF report contract
for final release presentation. It supersedes the blanket canonical visible
decimal rule in `specs/008-report-pdf-annex/contracts/report-rendering.md` only
for currency-denominated amounts and unit prices. Existing section order,
calculation rules, quantity formatting, exchange-rate precision, output bundle
shape, PDF layout requirements, and security rules remain in force.

In this contract, inherited provider-published rate precision means the
canonical normalized exact-decimal rate value, not the provider response's
lexical scale.

The output transaction remains governed by
`specs/008-report-pdf-annex/contracts/report-output.md`. This feature does not
change the bundle shape, exclusive reservation, matched Markdown suffix,
write/sync/close completion point, failed-attempt cleanup, collision retention,
or post-save opener-warning behavior. Every reserved report path is requested
with owner-read and owner-write permissions only, mode `0600`, before writing.

## Explicit User Export Contract

- Markdown and PDF output follows a direct user request and is generated and
  saved locally to the user-controlled Documents directory.
- Every successful result, including opener-warning success, identifies the
  files as cleartext financial-data exports, reports every saved path, and tells
  the user to delete all listed files to remove the exported data.
- No additional cleartext copy, temporary report file, report history, durable
  output-path state, reopen catalog, remote transfer, or automatic re-ingestion
  is permitted.
- Failed-attempt cleanup removes only current-attempt paths. Successful files
  become user-managed after the optional operating-system open handoff.

## Main Report Warning

The main report must contain this exact sentence once:

```text
The data in this report does not follow any legally required rules for any country's tax returns and is for reference only.
```

Rules:

- It appears immediately after the initial `Report Calculation Currency` field.
- It appears immediately before `Gains-And-Losses Summary`.
- It is one standalone logical paragraph. No other metadata, heading, table
  content, or prose shares or interrupts that paragraph.
- Markdown encodes that paragraph on one source line. PDF may wrap it into
  multiple physical lines or text runs because of available width; the ordered
  fragments remain one occurrence and every fragment remains bold.
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

Its complete visible grammar is fixed-point ASCII
`^-?[0-9]+\.[0-9]{2}$`. The decimal separator is the literal `.`, the only
permitted sign is one leading `-`, and grouping, leading `+`, whitespace,
alternate separators, and exponent notation are forbidden.

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

- Accepted sources are finite decimals whose adjusted exponent, source exponent
  plus coefficient digit count minus one, is from `-100000` through `100000` and
  whose correctly rounded scale-2 result, including a carry, also has adjusted
  exponent no greater than `100000`. Scale-2 quantization precision, including
  coefficient expansion and one carry digit, must not exceed `4294967295`.
  Successful formatting rules apply only inside that domain. Values outside it,
  including an upper-bound carry to adjusted exponent `100001`, fail under the
  Failure Contract before visible output.
- Use exact-decimal HALF UP rounding at two places.
- HALF UP applies symmetrically, so an exact negative tie rounds away from zero.
- Round only while deriving final visible strings.
- Do not mutate calculated decimals or use visible strings in later calculation,
  conversion, comparison, omission, storage, or totals.
- Retain the existing explicit currency identity or currency column for each
  value.
- A present whole or one-place value receives trailing zeros.
- An accepted present exact zero renders `0.00`.
- An accepted negative value that rounds to zero renders `0.00`, not `-0.00`.
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

Quantity values use the FR-009 canonical fixed-point representation computed
from the exact pre-presentation model value: no exponent or grouping separator,
no leading `+`, no rounding or fractional padding, and no fractional trailing
zeros. A decimal separator appears only when a fractional part remains, and
numeric zero renders as `0`.

Examples:

| Exact quantity | Visible quantity |
|----------------|------------------|
| `2` | `2` |
| `0.1` | `0.1` |
| `0.00000001` | `0.00000001` |

Disclosed `Rate Value` ratios use the canonical normalized fixed-point
representation of the exact validated rate evidence and never pass through
two-decimal formatting. Provider lexical trailing zeros are not retained:
`0.86010`, `16.9140`, `1.094600`, and `2.00` render as `0.8601`, `16.914`,
`1.0946`, and `2`. No significant digit may be rounded or discarded, and the
source currency, report base currency, rate date, and quote direction come from
the same pre-presentation evidence.

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

`IsZeroPricedHoldingReduction` is authoritative input inherited from the
explained zero-priced `SELL` rule in Feature 003 FR-017 and its reporting
treatment in Feature 005 FR-029/FR-029a. Feature 009 does not recompute,
redefine, broaden, or validate that classification and does not change sync
admission.

For a Detailed Per-Asset Audit Report row carrying that exact pre-format
classification:

- The calculated audit model retains its existing pre-format `ActivityCurrency`
  value and the exact zero-priced-reduction classification before presentation;
  the feature does not invent source provenance when no source currency context
  was selected.
- `Original Activity Currency` is blank.
- `Calculation Currency` remains populated.
- Activity type, quantity, quantity after activity, basis after activity,
  liquidation evidence, gain or loss, conversion status, and note remain
  unchanged and visible when otherwise applicable.

For every row where `IsZeroPricedHoldingReduction` is false, `Original Activity
Currency` retains its existing value, including when a non-zero price displays
as `0.00`.

No renderer may infer the classification from a two-decimal `Unit Price` string.

## Converted Amounts Contract

Each included conversion component uses this logical entry syntax:

```text
<field label>: <original amount> -> <converted amount>
```

Rules:

- Presentation acceptance fixtures cover exactly the eight order-preserving
  subsequences of `unit_price`, `gross_value`, and `fee_amount`: empty, each of
  the three singletons, `[unit_price, gross_value]`, `[unit_price, fee_amount]`,
  `[gross_value, fee_amount]`, and the complete three-entry sequence. This set
  does not constrain inherited calculator output or define list validity.
- Presentation treats the received supported-kind sequence as read-only and
  does not sort, deduplicate, synthesize, or reorder it. This feature adds no
  duplicate-kind or list-order validation failure, including for a supported
  received sequence outside the canonical fixture set.
- There is exactly one ordinary space after `:`.
- There is exactly one ordinary space on each side of `->`.
- Original and converted amounts follow the two-decimal financial contract.
- A component is omitted only when its exact original and converted amounts are
  both zero.
- Every included entry has one distinct logical start in the existing
  `Converted Amounts` cell. The first starts at the cell origin, and each later
  entry starts after the renderer-controlled boundary following the prior
  semicolon. Width-driven physical wraps inside an entry create no additional
  logical start.
- Every non-final entry ends with `;` followed by a format-appropriate visible
  line break.
- The final entry has no trailing semicolon.
- A single entry starts at the cell origin and has no semicolon or controlled
  line boundary. It may occupy multiple physical lines through width wrapping.
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

For this contract, the fresh-page row area is the vertical space between the
first valid data-row origin after mandatory page context, table header, and
spacing and the preserved bottom printable margin. If a measured row does not
fit the current page but fits that fresh-page row area, the renderer advances
before drawing any part of the row and draws it whole exactly once. If it cannot
fit that fresh-page row area, rendering fails without row splitting, clipping,
overlap, repeated empty continuation pages, PDF finalization, or output success.

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
- Complete-row preflight follows FR-026 and FR-027 for both remaining-page and
  fresh-page capacity. A logical row is never split across pages.
- Existing arbitrary report text stays single-line sanitized. Only controlled
  Converted Amounts cell boundaries are preserved as newlines.

## Accessibility And Readability Contract

The inherited searchable/selectable PDF-text and human-readable layout
requirements remain normative for the warning and multiline Converted Amounts
content. The warning's meaning remains explicit in its complete text rather than
depending on bold style alone. Searchable/selectable means emitted report text
remains searchable and selectable in a supporting PDF reader; it does not imply
tagged-PDF semantics, PDF/UA conformance, semantic table associations,
guaranteed assistive reading order, or screen-reader interoperability. Markdown
`<br>` and PDF newline boundaries are visible layout encodings and create no
additional assistive-technology conformance claim.

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

## Confidentiality Contract

SEC-001 applies to successful and failed rendering across generated documents,
result and error text including wrapped causes, diagnostics, documentation and
test examples, screenshots, and committed or generated fixtures. A contextual
error may identify the failing stage, field, or row, but project-owned boundaries
must redact or suppress real credentials, bearer or JWT values, reusable
authentication or decryption material, token-derived verifiers or keys, raw
encrypted or decrypted protected-payload serialization or reversible encodings,
and real user financial data outside contracted export fields and separately
reviewed diagnostic modes. Contracted cleartext financial fields do not authorize
the same values in errors, logs, screenshots, examples, fixtures, or unrelated
document content. Examples, fixtures, and redaction sentinels must be clearly
synthetic and non-reusable.

## Failure Contract

- A non-finite decimal, finite but unrepresentable value, formatting precision
  or exponent limit, unexpected decimal condition, or PDF measurement,
  wrapping, page-fit, drawing, or byte-finalization failure returns a contextual
  non-secret render error identifying the failed stage and applicable semantic
  field or table-row context, wraps only a redacted cause, leaves the application
  available for retry, and must not create a successful output result.
- PDF byte finalization means serializing the completed in-memory PDF layout
  into the byte payload used to construct the combined PDF report document. It
  occurs before output-path reservation and must use an error-returning path.
- A render or finalization failure discards partial bytes, returns no successful
  report document, leaves the application process running, and invokes neither
  the output writer nor the opener.
- Selected-format output success occurs only after every required file has been
  exclusively reserved, fully written, synced, closed, validated, and recorded.
  Candidate collisions use the inherited suffix retry and are not terminal
  failures. Any unrecoverable reservation or later pre-success writer or bundle
  failure removes every path reserved or created by that attempt and reports no
  saved path.
- Failed-attempt cleanup must not overwrite, truncate, remove, or otherwise
  change a pre-existing colliding file or an earlier successful report bundle.
- An opener failure after output success is a success-with-warning. All saved
  paths remain present and are reported.
- Every successful and failed channel follows the Confidentiality Contract.
- Capability insufficiency must not trigger a remote renderer, browser, external
  binary, platform service, dependency change, or reduced report contract at
  runtime. It is a DEP-001 planning blocker.

## Automated Evidence

Automated acceptance must use the closed manifest and semantic occurrence keys
defined by the specification. A failed listed format attempt remains in its
applicable populations. Evidence reports numerator and denominator counts for
`A`, `W`, `V`, `R`, `M`, `Q`, `B`, `Z`, `N`, `C`, `P`, and `E`; none may be
empty. `A` is generated only from the closed case-ID schemas, tables, and literal
sets in the specification, while `R` is enumerated from the pinned baseline test
tree and is not treated as a renderer-attempt population.

Automated tests must cover both formats and prove:

- exact warning text, one occurrence, and placement
- generated-PDF text runs covering the complete warning, including the final
  period, all use the embedded bold font
- the complete Financial Presentation Acceptance Matrix cross-product at each
  specified Markdown and PDF semantic field boundary, not only a shared
  formatter or aggregate extracted-text sample
- inclusive FR-004a adjusted-exponent and precision arithmetic at the formatter
  boundary including an accepted upper-bound non-carry case, plus Markdown and
  PDF failures immediately outside either exponent bound, on upper-bound carry,
  and above the precision limit with no visible or saved output
- unchanged quantities, exchange rates, calculated values, and exact inclusion
  decisions
- canonical rate cases prove that significant digits remain while provider
  lexical trailing zeros are removed identically in both formats
- every quantity occurrence equals the FR-009 canonical value computed from its
  exact pre-presentation decimal without using prior output as a baseline
- pre-render and post-render model equality for every AUD-001 exact decimal,
  currency, quantity, rate and metadata field, inclusion or omission state, and
  classification
- both boolean labels
- the existing calculated audit `ActivityCurrency` value remains unchanged
  before a zero-priced presentation cell becomes blank
- classified blank and unclassified retained visible activity currencies,
  including an unclassified positive `0.004` control that displays as `0.00`
- presentation fixtures receive existing true and false classifications without
  exercising upstream classification predicates or sync admission
- zero, one, two, and three included converted-entry cases covering each of the
  eight FR-019 canonical subsequences
- received converted-entry order is preserved without new duplicate-kind or
  supported-kind order validation
- exact colon, arrow, semicolon, order, and line-boundary behavior
- generated-PDF coordinates show each converted entry starts on a later visible
  line
- PDF measured and drawn line counts, row height, and bottom-margin preflight
  agree for explicit newlines and long space-wrapped content
- table-start relocation before the first row emits no continuation label, while
  actual continuation pages retain inherited context and repeated headers
- normal success and opener-warning result copy identify cleartext financial
  exports, report every saved path, provide deletion guidance, and retain no
  report or path history after result-flow exit
- a row that fits only the fresh-page row area advances before drawing and is
  drawn whole once, while a row exceeding that area returns a layout error and
  does not finalize or save a PDF
- non-finite, unrepresentable, layout, drawing, and byte-finalization failures
  return normally with no successful document, output path, or opener request
- Markdown second-file failure and PDF save failure remove only current-attempt
  paths, preserve sentinel contents in colliding prior files, and report no
  partial success
- opener failure after complete output success retains and reports every file
- SEC-001 sentinels are absent from successful documents, returned and wrapped
  errors, diagnostics, examples, and generated fixtures
- warning and converted-entry content remains complete searchable/selectable
  text with preserved order and non-overlapping layout, without claiming the
  assistive capabilities excluded by ACC-002
- deterministic package, contract, and integration tests prove that the named
  10,000-activity fixture produces exactly 6,666 three-entry conversion rows in
  both formats; the PDF audit spans multiple pages with every row and entry
  present, searchable, inside printable bounds, and accompanied by repeated
  header and continuation context where applicable
- the isolated performance suite proves the exact local workload composition,
  separate Markdown and PDF duration records under the specified CI environment
  and timing boundary, successful non-empty outputs, opener invocation, and
  performance-coverage isolation. It does not inspect document content or
  duplicate deterministic warning, rate, conversion-entry, cardinality,
  heading, or pagination assertions.
