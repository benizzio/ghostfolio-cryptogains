# Research: Final Report Adjustments

## Research Inputs

- Feature specification: `specs/009-final-report-adjustments/spec.md`
- Constitution: `.specify/memory/constitution.md`
- Existing report and PDF design: `specs/008-report-pdf-annex/plan.md`,
  `specs/008-report-pdf-annex/research.md`, and
  `specs/008-report-pdf-annex/contracts/report-rendering.md`
- Existing implementation: `internal/report/calculate/`,
  `internal/report/model/`, `internal/report/presentation/`,
  `internal/report/markdown/`, `internal/report/pdf/`, and
  `internal/report/output/`
- Existing automated evidence: package-local report tests and the suites under
  `tests/unit/`, `tests/contract/`, `tests/integration/`, `tests/empirical/`, and
  `tests/performance/`
- Current `apd` documentation for `Context.Quantize`, rounding modes, context
  precision, conditions, and traps
- Current `gopdf v0.36.1` source for `MultiCellWithOption`,
  `SplitTextWithOption`, and `IsFitMultiCellWithNewline`
- OWASP Top 10:2025 release at `https://owasp.org/Top10/2025/`

All technical-context questions are resolved by the decisions below.

## Shared Presentation Boundary

Decision: Keep exact calculations in `internal/report/calculate/`, place
report-specific monetary formatting and format-neutral semantic values in
`internal/report/presentation/`, and retain syntax/layout in the Markdown and
PDF packages.

Rationale: Runtime currently calculates one exact `CapitalGainsReport` and then
forks directly to the Markdown or PDF renderer. PDF does not consume Markdown.
Activity, liquidation, Annex activity, and conversion rows already pass through
the presentation package, while summary and position values are direct-renderer
exceptions. A shared report presentation helper gives both formats identical
visible values without introducing a document AST or moving Markdown syntax into
PDF. Scale 2 is a report-domain display policy, so it does not belong in the
domain-neutral canonical decimal support API.

Alternatives considered: Change `decimal.CanonicalString`, but quantities,
exchange rates, persistence, and calculation diagnostics depend on canonical
un-padded output. Render PDF from Markdown, but the established PDF contract
requires direct structured rendering. Duplicate financial formatting in both
renderers, but that risks format drift.

## Two-Decimal Financial Formatting

Decision: Format each present currency-denominated amount or unit price from a
defensive `apd.Decimal` copy by quantizing to exponent `-2` with
`apd.RoundHalfUp`, then emit fixed-point text with two places. Start from a copy
of `apd.BaseContext` so package exponent limits and default traps remain active.
Set precision from checked source metadata as the source digit count plus any
coefficient expansion needed to reach exponent `-2`, plus one possible carry
digit. Reject precision arithmetic that cannot fit `uint32`. Accept no operation
conditions except the expected `Rounded` and `Inexact` flags. Normalize a
quantized zero to non-negative before rendering.

Rationale: `Context.Quantize` is the `apd` operation for a requested exponent and
supports `RoundHalfUp`. The destination exponent preserves trailing zeros when
rendered in fixed-point form. A separate destination protects the calculated
input. Dynamic result precision avoids rejecting large whole values merely
because adding two fractional places expands their coefficient. Checking the
precision arithmetic, operation error, and condition mask prevents non-finite,
overflowed, or unrepresentable data from becoming misleading report text.
Inheriting `BaseContext` avoids the active zero-valued exponent bounds of a
literal `apd.Context`. Normalizing zero satisfies the explicit `0.00`, not
`-0.00`, contract.

Alternatives considered: String slicing or floating-point formatting, but both
would bypass exact decimal semantics. Reuse the calculation context and its
scale, but report scale 2 is fixed and independent. Round calculated models in
place, but that would violate the no-feedback calculation boundary. Use
`CanonicalString` followed by padding, but padding alone cannot perform HALF UP
rounding.

## Exact Decisions Before Visible Rounding

Decision: Continue making inclusion and omission decisions from exact model
values, and consume existing classifications for currency applicability, before
formatting any display string.

Rationale: A non-zero value such as `0.004` may display as `0.00` but remains a
non-zero calculated value. Summary omission must therefore use
`Decimal.Sign() == 0`, converted components must continue to be omitted only when
both exact amounts are zero, and zero-priced holding reductions must use the
existing `IsZeroPricedHoldingReduction` classification inherited from Feature
003 FR-017 and Feature 005 FR-029/FR-029a. Feature 009 does not recompute or test
that classification and does not change sync admission. This preserves report
membership and audit meaning while changing only visible precision.

Alternatives considered: Make decisions from the two-place string, but that
would incorrectly omit small non-zero values and could misclassify activity.
Infer a zero-priced reduction from the rendered unit price, but a small non-zero
unit price can also render as `0.00`.

## Warning Placement And Emphasis

Decision: Define the exact warning once as shared report presentation text.
Markdown emits it as one standalone `**...**` paragraph after the initial
metadata. PDF adds a dedicated fully bold wrapped-paragraph layout operation at
the same semantic position before the summary heading. Standalone describes one
logical paragraph, not one physical PDF line; width-driven lines and text runs
remain ordered bold fragments of the same occurrence.

Rationale: A shared literal prevents wording drift, while each renderer must own
its emphasis mechanism. The existing PDF `AddParagraph` is regular and
`AddKeyValue` bolds only a label, so neither can prove that the complete sentence
including the final period is bold. A dedicated operation also gives tests an
explicit style and ordering seam.

Alternatives considered: Store the warning in `CapitalGainsReport`, but it is
static presentation policy rather than calculated data. Use a PDF heading, but
that gives the warning incorrect document semantics. Render Markdown bold
markers into PDF, but visible Markdown syntax violates the PDF contract.

## Structured Boolean Labels

Decision: Build the existing Annex `FullLiquidationEvent` presentation value as
`Yes` or `No` in the shared Annex row and render that string directly in both
formats.

Rationale: This is the only structured boolean currently exposed in report
content. Mapping it at the shared row boundary gives Markdown and PDF the same
label and prevents Markdown from converting the string back to `%t`.

Alternatives considered: Replace every occurrence of `true` and `false` in final
documents, but arbitrary notes may contain those words and must not be changed.
Map independently in each renderer, but that duplicates a semantic rule.

## Zero-Priced Audit Currency Applicability

Decision: Add the existing `IsZeroPricedHoldingReduction` classification to the
transient `AuditActivityEntry` report model while retaining its existing
pre-format `ActivityCurrency` value. `BuildAnnexActivityRow` leaves only the
visible activity currency empty when that classification is true. Keep
`CalculationCurrency` and every other audit field unchanged. Treat the
classification inherited from Feature 003 FR-017 and Feature 005 FR-029/FR-029a
as authoritative input. Feature 009 neither recomputes nor broadens it, and every
unclassified row retains its existing visible activity currency.

Rationale: At construction time the exact domain classification is available.
Zero-priced reductions may have no selected currency context, and the
currency boundary currently supplies report calculation currency as an audit
fallback. Retaining that existing field value and adding only the classification
preserves pre-feature model behavior without inventing source provenance. The
shared presentation row gives both formats the required blank cell without
renderer-side financial inference. The added boolean is transient report
evidence copied from an existing input; it does not alter calculation, storage,
currency selection, inclusion, upstream classification, or sync admission.

Alternatives considered: Infer applicability in renderers from activity labels
and visible price, but that duplicates logic and can confuse a rounded non-zero
price with exact zero. Clear `ActivityCurrency` in the calculated audit model,
but that would change its pre-format value. Re-evaluate source monetary fields in
Feature 009, but that would duplicate and potentially broaden the inherited
classification contract.

## Converted Amount Entry Representation

Decision: Change the shared conversion presentation from one delimiter-bearing
string to an ordered list of single-line logical entries. Treat the inherited
`ConversionAuditEntry.Amounts` sequence as read-only, preserve the relative order
after exact zero-to-zero omission, and add no duplicate-kind or supported-kind
order validation. Generated reports retain the calculator's existing
`unit_price`, `gross_value`, `fee_amount` subsequences. Each renderer sanitizes
entries separately and joins them with its own visible separator:
`;<br>` for Markdown table cells and `;\n` for PDF table cells. Markdown escapes
HTML-sensitive and table-delimiter characters in the dynamic label and amount
components before assembling the fixed literal `: ` and ` -> ` syntax and
inserting the controlled `<br>`. The last entry receives no semicolon.

Rationale: A raw newline would terminate a Markdown pipe-table row, while a
controlled `<br>` provides an inline visible break. PDF table cells accept
explicit newlines through `MultiCellWithOption`. A logical list prevents either
format's delimiter from leaking into the other and makes one-, two-, and
three-entry contracts plus the empty case and all eight canonical subsequences
directly testable. Preserving the received order avoids a
new report-generation failure that would conflict with this feature's
presentation-only scope. Escaping before joining ensures only the renderer can
add the HTML break used by this cell.

Alternatives considered: Keep `; `, but it fails the visible-line requirement.
Put literal newlines in Markdown cells, but that breaks table structure. Put
`<br>` in shared presentation data, but PDF must not expose Markdown/HTML syntax.
Create separate converted-entry model copies per renderer, but format-neutral
entry strings are sufficient. Strengthen list-level duplicate or order
validation, but that would change accepted model behavior without a calculation
or audit-integrity requirement and is outside this feature.

## PDF Newline Measurement And Sanitization

Decision: Preserve renderer-inserted line boundaries in a dedicated PDF table
cell path, sanitize every logical line with the existing single-line sanitizer,
and measure table cells with the same break policy used by drawing. First call
`SplitTextWithOption` with the table layout's pinned
`BreakModeIndicatorSensitive` and space separator. Join the resulting lines with
newlines and pass those exact lines to `IsFitMultiCellWithNewline` for height and
fit. Keep generic arbitrary report text single-line.

Rationale: `gopdf` table drawing uses `MultiCellWithOption`, whose text splitting
uses indicator-sensitive word wrapping and honors explicit newline boundaries.
The current project measurement seam uses character-based `IsFitMultiCell`, and
`IsFitMultiCellWithNewline` delegates to the same character-based function for
each line. Calling that helper directly can still underestimate a word-wrapped
cell. Pre-splitting with the drawing option and measuring the resulting lines
makes predicted line count and height match drawing. Splitting, sanitizing, and
rejoining only controlled table content retains redaction while allowing the
required layout. Existing complete-row preflight then applies to the expanded
height. If the row fits only the fresh-page row area, preflight advances before
drawing and keeps the row whole. If it cannot fit that area, rendering returns a
layout error rather than splitting the row, clipping content, or looping through
empty continuation pages. Row splitting was rejected because it would conflict
with the inherited complete-row contract and broaden the presentation model.

Alternatives considered: Raise every conversion row to a hard-coded height, but
wrapped long entries still need measurement. Preserve newlines in the generic
sanitizer, but arbitrary report fields are intentionally flattened. Draw each
entry as a separate table row, but that would alter table row identity and
section evidence.

## Constitution Prerequisite: PDF Finalization

Decision: Isolate one prerequisite correction in the existing PDF adapter:
change its byte-finalization seam to return `([]byte, error)`, use
`gopdf.GetBytesPdfReturnErr`, and propagate the error through `Renderer.Render`
before output reservation or writing, as required by FR-022 and FR-023.

Rationale: The current `GetBytesPdf` convenience method calls `log.Fatalf` when
finalization fails. That terminates the process and bypasses report failure
handling, which is incompatible with the constitution's exceptional-condition
gate and OWASP A10:2025 review. The pinned dependency already exposes the needed
error-returning API. The correction changes no successful report content and is
part of the normative render-failure contract without changing successful
visible content.

Alternatives considered: Leave the existing path unchanged because it predates
this feature, but the post-design constitution gate cannot pass with a known
fatal failure path in the PDF adapter being modified. Recover from `log.Fatalf`,
but process exit is not recoverable.

Render failures occur before reservation and therefore create no report output
path. Writer failures occur after exclusive reservation and remain governed by
the inherited output transaction: every path reserved by the current attempt is
removed, while colliding pre-existing files and earlier successful bundles are
retained. Opener failure after complete bundle success remains a warning and
does not trigger cleanup.

## Testing Evidence Strategy

Decision: Combine the closed acceptance manifest and field/vector/output matrix,
pure shared-presentation tests, exact Markdown contracts, PDF layout recorder
tests, extended concrete generated-PDF inspection, runtime parity and AUD-001
model-equality tests, confidentiality sentinels, unchanged empirical calculation
tests, and the isolated named performance scenario.

Rationale: The current project PDF inspector proves landscape page geometry and
searchable normalized text, but its normalization removes punctuation,
whitespace, signs, line breaks, and font information. Extend its project-specific
inspection model with ordered text runs carrying page, font resource, and text
coordinates. Contract tests can then prove that every emitted warning fragment,
including the final period, uses the embedded bold font and that each converted
entry starts at a later vertical coordinate. Package-local recorder tests still
prove operation order, exact strings, matching measurement/drawing line counts,
complete-row preflight, and safe finalization error propagation. Runtime and
empirical tests protect the boundary before formatting. Searchable/selectable
text and readable non-overlapping layout remain the accessibility boundary;
tagged-PDF, PDF/UA, semantic reading-order, and screen-reader claims are excluded
and cannot be inferred from text-run inspection. Performance remains outside
deterministic coverage.

Failure and recovery evidence also forces unrenderable decimal, PDF layout,
finalization, Markdown second-file, PDF save, and post-save opener failures.
Sentinel colliding files prove that cleanup is limited to current-attempt paths.
The performance fixture keeps its exact 10,000-activity, two-asset,
three-currency composition and 6,666 three-entry conversion rows. Separate
Markdown and PDF timers include generation, multiline PDF pagination and
finalization, save, bundle validation, and opener invocation. Fixture setup and
post-generation row, entry, page, header, and continuation inspection stay
outside the measured intervals.

Alternatives considered: Depend only on normalized extracted text, but it cannot
prove the new requirements. Add a second PDF parser, but the existing
project-owned inspector already parses the pinned generator's object and content
streams and can be extended narrowly. Use only manual inspection, but the
specification requires automated acceptance evidence; manual inspection remains
supplemental for reader-specific visual behavior.

## Dependencies And Integrations

Decision: Add no dependency or external integration. Reuse `apd v3.2.3` and
`gopdf v0.36.1` through their existing pinned versions.

Rationale: `apd` already provides exact HALF UP quantization and a base context
with package exponent limits and default traps. `gopdf` already provides the
word-wrap splitting, newline-aware fit, and explicit-newline drawing APIs
required by the design. The feature does not require provider access, browser
conversion, an external binary, remote fonts, or a document service. Keeping
`go.mod` and `go.sum` unchanged minimizes supply-chain scope.

Alternatives considered: Add a money-formatting library or PDF extraction
library, but the existing APIs and test seams are sufficient. Use an HTML/PDF
service, but that would expose financial data and violate the local-only design.

Capability contingency: This decision remains valid only if the pinned APIs and
project-owned inspection seams produce every normative rendering and acceptance
result. Contrary evidence is a DEP-001 planning failure, not authorization for a
fallback. Work stops while compliant local-only alternatives and dependency
risk are documented and the specification, plan, constitution checks, security
review, tasks, and acceptance evidence are revised. If no compliant plan exists,
the feature remains incomplete.

## Security Review

Decision: Treat the feature as a controlled presentation change inside the
explicit user-requested export boundary defined by constitution v3.0.0 and
review it against OWASP Top 10:2025.

Rationale: The relevant risks are local output access (A01), renderer
configuration (A02), dependency integrity (A03), token disclosure (A04/A07),
injection through Markdown or PDF delimiters (A05), misleading rounded or
missing values (A06/A08), sensitive failure logging (A09), and render/layout or
write failures (A10). The existing writer continues to request mode `0600` and
use its reserve/write/cleanup sequence. Final report files qualify as explicit
user-requested exports because processing is local, every path and the cleartext
financial-data status are disclosed, deletion guidance is shown, and no
additional cleartext copy, report history, durable path state, reopen catalog, or
automatic re-ingestion is retained. SEC-001 applies to successful documents,
returned and wrapped errors, diagnostics, screenshots, examples, and fixtures
and excludes real credentials, reusable authentication or decryption material,
raw protected-payload serialization, and real user financial data outside
contracted export fields and separately reviewed diagnostic modes. Only clearly
synthetic non-reusable redaction sentinels are permitted. Generated
Converted Amounts dynamic components are escaped and sanitized before fixed
entry syntax and controlled `<br>` or newline delimiters are inserted. The
isolated PDF finalization prerequisite replaces process exit with normal error
propagation. No network, authentication, remote storage, or cryptographic
boundary changes.

Alternatives considered: Treat generated reports as protected application
snapshots, but that would make ordinary Markdown and PDF exports unusable and is
unnecessary under the narrow constitution v3.0.0 boundary. Retain cleartext
temporary files or report history, but those remain application-managed
persistence and are prohibited. Add remote rendering for visual verification,
but that would create an unnecessary data disclosure surface.
