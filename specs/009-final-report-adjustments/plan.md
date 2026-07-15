# Implementation Plan: Final Report Adjustments

**Branch**: `[009-final-report-adjustments]` | **Date**: 2026-07-15 | **Spec**: `/specs/009-final-report-adjustments/spec.md`

**Input**: Feature specification from `/specs/009-final-report-adjustments/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Apply the release-ready report presentation rules to the existing Markdown and
PDF renderers without changing report calculation. A report-specific shared
presentation helper will format every currency-denominated amount and unit price
to two decimal places with HALF UP rounding while leaving quantities and
normalized provider exchange rates canonical. Shared presentation rows will also
produce `Yes`/`No` boolean labels and ordered converted-amount entries. Markdown
will encode controlled table-cell breaks as `<br>`, while PDF will encode them
as explicit newlines measured with the same word-wrap policy used for drawing.
The exact legal-use warning will be inserted once in each main report, and
zero-priced holding reductions will retain their existing pre-format audit
currency value while omitting that non-applicable value from the visible Annex 1
cell. The inherited report-calculation classification continues to accept
all-missing and mixed missing-and-zero monetary shapes without manufacturing
zero values or broadening sync admission. Successful result copy will disclose
cleartext financial exports, every saved path, and deletion guidance.

## Technical Context

**Language/Version**: Go 1.26.5

**Primary Dependencies**: Existing `github.com/cockroachdb/apd/v3 v3.2.3`
for exact decimals and HALF UP quantization,
`github.com/signintech/gopdf v0.36.1` for local structured PDF rendering,
`golang.org/x/image v0.43.0` for application-supplied PDF fonts, and the Go
standard library. No dependency or version change is planned. This no-change
decision is conditional on the capability evidence and stop/replan gate in
DEP-001.

**Storage And User Exports**: Existing direct user-requested cleartext Markdown
and PDF exports in the resolved local user-controlled Documents directory.
Under constitution v3.0.0 these final files remain outside application-managed
persistence because every saved path and the cleartext financial-data status are
disclosed, owner-only mode `0600` is requested where supported, failed-attempt
files are removed, and the application retains no additional cleartext copy,
report history, reopen catalog, durable output-path state, or automatic
re-ingestion. Successful result copy tells the user to delete every listed path
to remove the exported data. No new cache, snapshot field, temporary cleartext
file, telemetry, remote storage, or automatic processing is added. New OWASP
Cryptographic Storage evidence is N/A because no financial or person-linked data
enters application-managed persistence; existing protected snapshots remain
unchanged and token-derived encrypted.

**Testing**: Go `testing` across package-local report tests, `tests/unit`,
`tests/contract`, `tests/integration`, existing `tests/empirical`, and the
isolated build-tagged `tests/performance` suite. Package-local renderer tests
will verify exact formatting, PDF style intent, operation order, controlled line
breaks, matching PDF measurement/drawing wrap behavior, and row height. Contract
tests will enforce the closed acceptance manifest, field/vector/output matrix,
population counts, confidentiality sentinels, and inspect generated PDF text
runs, fonts, and coordinates to prove complete bold warning output and separate
converted-entry lines. Integration tests will verify generated Markdown/PDF
parity and unchanged pre-presentation calculation evidence. Final gates are
`make test`, `make coverage`,
`make test-performance`, and `make quality QUALITY_BASE_REF=origin/main`.

**Empirical Dataset**: Existing synthetic YAML dataset and generated JSON oracle
fixtures under `testdata/empirical/`. They remain read-only. Existing empirical
tests continue to compare exact calculated values before rendering; this
feature does not authorize dataset or oracle regeneration.

**Target Platform**: Existing installed terminal application targets Linux,
macOS, and Windows with local filesystem access. Current maintained CI evidence
is Ubuntu-based; this feature adds no platform-specific rendering or IO path.

**Project Type**: Single-module Go terminal UI application with local report
generation.

**Performance Goals**: Under the specification's named two-asset, three-currency,
10,000-activity fixture and recorded `test-performance / run` Ubuntu environment,
preserve independent generation of one Markdown bundle and one PDF document in
strictly under two minutes per selected format. Each timer surrounds only the
selected generation operation and includes calculation, multiline rendering,
PDF pagination and finalization where applicable, save, bundle validation, and
opener invocation, while fixture/setup and output inspection remain outside.

**Constraints**: Presentation-only rounding at scale 2 with `apd.RoundHalfUp`;
no floating-point financial values; no mutation or reuse of rounded values in
calculation; exact-zero decisions occur before display formatting; quantities
use the FR-009 canonical baseline and disclosed rates use normalized canonical
evidence; nil optional amounts
remain blank; rounded negative zero becomes `0.00`; no new dependency, network,
authentication, persistence, report section, table column, or output format;
existing landscape A4 PDF pagination, searchable text, embedded fonts, complete
row preflight, output reservation/cleanup sequence, and secret redaction remain
intact; the constitution-required PDF finalization seam must return errors instead
of terminating the process; SEC-001 applies to every document, error, diagnostic,
example, and fixture channel, while EXP-001 and FR-028 govern cleartext and path
disclosure plus deletion guidance; searchable/selectable PDF text and readable layout
remain while ACC-002 excludes broader assistive-conformance claims; DEP-001
blocks unplanned capability fallbacks; production statement, line, branch, and
per-file coverage remains 100% as enforced by the repository gate.

**Scale/Scope**: One existing calculated report, one Markdown main document and
separate Annex 1 document, one combined PDF document, all main-report and Annex
financial fields, one structured boolean field currently exposed by Annex 1,
one transient audit classification copied into the calculated report model, and
one converted-amount cell containing the received calculator-produced
`unit_price`, `gross_value`, and `fee_amount` subsequences without new list-level
validation.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS
Post-design gate status: PASS
Post-analysis remediation gate status against constitution v3.0.0: PASS

- [x] Security: Under constitution v3.0.0, reports remain explicit cleartext
  user-requested exports, not application-managed persistence, only because they
  are local and user-controlled, use requested mode `0600` where supported,
  disclose cleartext status and every saved path, include deletion guidance,
  clean current-attempt failures, and add no retained copy, history, durable path
  state, reopen catalog, or automatic re-ingestion. SEC-001 prohibits reusable
  authentication/decryption material, raw protected-payload serialization, and
  real user financial data outside contracted export fields and separately
  reviewed diagnostic modes. Output reservation/cleanup remains unchanged while
  result copy gains the required disclosure and removal guidance.
  The OWASP Top 10:2025 review covers A01 local file access boundaries, A02
  local-only renderer configuration, A03 supply-chain risk with no new modules,
  A04 and A07 token exclusion, A05 sanitization before controlled Markdown/PDF
  delimiters are inserted, A06 exact-value decisions before display rounding,
  A08 report-value integrity and format parity, A09 non-secret failures, and A10
  new formatting/layout error handling without false output success plus the
  prerequisite replacement of fatal PDF finalization with normal error return.
  Renderer-owned Markdown delimiters are inserted only after dynamic entry
  components are escaped and sanitized and the fixed visible syntax is assembled.
- [x] Precision: Existing finite `apd.Decimal` values and explicit currency
  identities remain authoritative. Present monetary amounts and unit prices are
  cloned and quantized only into final strings at exponent `-2` with
  `apd.RoundHalfUp`. Formatting uses a copy of `apd.BaseContext`, checked result
  precision sized for trailing zeros and carry, and accepts only expected
  `Rounded` and `Inexact` conditions. Quantities use the objective FR-009
  canonical representation. Rates preserve every significant digit of the
  normalized evidence while discarding provider lexical scale. Each existing
  pre-format activity currency value remains unchanged before rendering. No
  conversion boundary, rate source, cost-basis rule, calculation scale, or
  stored value changes.
- [x] Testing: Integration and contract tests enforce the closed acceptance
  manifest and denominators, the complete financial field/vector/output matrix,
  warning placement, objective quantity/rate representations, boolean labels,
  audit currency, zero-to-three converted entries, exact pre/post-render model
  equality, and retained searchable/selectable text. Targeted package tests are
  justified for the pure decimal display boundary and PDF font/layout seams.
  `make coverage` remains the canonical 100% coverage gate.
- [x] Quality gate: Report source and test changes in `*.go` are expected;
  `go.mod` and `go.sum` changes are not expected. Verification is
  `make quality QUALITY_BASE_REF=origin/main` or the successful `quality` check,
  together with the exact CI checks `test / run`, `coverage / run`,
  `test-performance / run`, and `quality` where CI evidence is cited.
- [x] Empirical financial validation: `tests/empirical/` continues to verify
  exact calculated results against the read-only YAML and JSON material under
  `testdata/empirical/`. Rendering is downstream of this boundary, so no
  empirical dataset or oracle mutation is required or permitted.
- [x] Dependencies and external integrations: No dependency, version, API, or
  integration change is planned. Existing `apd` quantization and `gopdf`
  split/fit APIs are sufficient when invoked with the table drawing break policy.
  No live provider, browser, external binary, remote font, or report service is
  introduced. If required rendering, finalization, layout, readability,
  sanitization, or inspection evidence proves unavailable, this PASS status is
  invalidated and work stops under DEP-001. Both constitution checks, research,
  planning, security review, tasks, and acceptance evidence must be revised
  before any dependency, integration, or requirement change.
- [x] Architecture: Report-specific financial display policy and format-neutral
  rows stay in `internal/report/presentation/`; Markdown syntax stays in
  `internal/report/markdown/`; PDF styling, sanitization, measurement, and
  pagination stay in `internal/report/pdf/`; the existing zero-priced audit
  classification is copied by `internal/report/calculate/` into the transient
  report model under `internal/report/model/`, and only presentation suppresses
  the visible non-applicable currency. `internal/sync/validate/` remains
  unchanged and is covered only to characterize the inherited sync-admission
  boundary. Runtime and output writing remain unchanged; `internal/tui/screen/`
  owns the successful-result cleartext, path, and deletion copy.

## Project Structure

### Documentation (this feature)

```text
specs/009-final-report-adjustments/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── report-rendering.md
├── checklists/
│   ├── report-contract.md
│   └── requirements.md
└── tasks.md
```

### Source Code (repository root)
```text
internal/
├── sync/
│   └── validate/        # Characterize unchanged sync admission; no source change planned
├── report/
│   ├── model/           # Retain pre-format currency and copy zero-priced classification without new list validation
│   ├── calculate/       # Populate the transient audit classification without changing financial results
│   ├── presentation/    # Shared financial display strings, boolean labels, and logical converted entries
│   ├── markdown/        # Warning emphasis, direct summary/position values, and controlled <br> table breaks
│   └── pdf/             # Bold warning, matched wrap measurement, row preflight, and safe finalization prerequisite
├── tui/
│   └── screen/          # Cleartext export, saved-path, and deletion result guidance
└── support/
    └── decimal/         # Existing canonical quantity/rate formatting remains unchanged

tests/
├── unit/                # Black-box Markdown presentation cases
├── contract/            # Exact Markdown, Annex, output, and concrete PDF contracts
├── integration/         # Runtime generation and pre-format calculation parity
├── empirical/           # Existing exact calculation evidence; fixtures stay read-only
├── performance/         # Existing independent 10,000-activity format timings
└── testutil/            # Extend generated-PDF inspection with text font and coordinate runs
```

**Structure Decision**: Keep the feature inside the existing single Go module
and renderer fork. Add no document AST or second renderer pipeline. Shared
semantic transformations are calculated once as format-neutral strings or
logical entries, after which each renderer owns only syntax and layout. Keep
the fixed two-decimal policy in the report presentation package rather than the
domain-neutral decimal support package because scale 2 is a report contract,
not a general decimal rule.

## Presentation And Rendering Boundary

1. `internal/sync/validate/` keeps its inherited admission rule: newly synced
   activities still require resolvable amount evidence, so this feature does not
   admit an all-monetary-values-missing upstream row. At the downstream report
   compatibility boundary, `internal/report/calculate/` preserves the existing
   zero-priced holding-reduction predicate for explained `SELL` rows whose every
   present source monetary value is exact zero; no monetary value must be
   present, and missing values remain nil. While constructing an Annex audit row,
   calculation retains the existing `ActivityCurrency` value and copies the
   existing `IsZeroPricedHoldingReduction` classification into the transient
   audit model. A classified reduction may have no selected source currency, so
   this feature does not invent provenance. Positive quantity and nonnegative
   running holdings remain enforced by inherited validation and replay. No
   financial value, currency identity, sync admission, or report inclusion
   changes before presentation.
   The inherited priced selector remains separate: it evaluates
   `order -> asset_profile -> base` and selects the first tier with an explicit
   currency, present finite fee including exact zero, finite gross value present
   or derived from same-tier unit price and quantity, and finite unit price
   present or derived from same-tier gross value and quantity. A missing fee or
   other incomplete requirement causes fallback to the next tier; values are
   never mixed across tiers.
2. `internal/report/presentation/` formats monetary values from defensive
   decimal copies at two places with HALF UP, canonicalizes quantities and
   rates through existing helpers, maps the report boolean to `Yes` or `No`, and
   returns converted amount entries as an ordered logical list after exact
   zero-to-zero omission. It blanks only the visible original currency for an
   audit row carrying the exact zero-priced-reduction classification. It treats
   the inherited converted-amount sequence as read-only and adds no duplicate-
   kind or supported-kind order validation.
3. `internal/report/markdown/` inserts the exact fully bold warning after report
   metadata, uses the shared financial formatter for summary and position values
   that bypass row builders, escapes and sanitizes each dynamic converted-entry
   component before assembling the fixed `: ` and ` -> ` syntax, and joins
   entries with `;<br>` so a pipe-table row remains structurally valid and only
   the renderer can introduce that HTML delimiter.
4. `internal/report/pdf/` inserts the same warning through a dedicated bold
   wrapped-paragraph operation. The operation represents one logical standalone
   paragraph even when width creates multiple physical lines or text runs, all
   of which remain bold. PDF uses the same shared financial formatter for
   direct values, sanitizes each converted entry, and joins entries with `;\n`.
   Table-cell measurement first applies `SplitTextWithOption` using the pinned
   table layout's indicator-sensitive space break option, then measures those
   exact lines with `IsFitMultiCellWithNewline`, so preflight and drawing use the
   same wrap result. A row that fits only on a fresh page advances before any
   drawing and remains whole; a row that exceeds fresh-page capacity returns a
   layout error without splitting, finalizing, or saving a PDF.
5. `internal/app/runtime/` and `internal/report/output/` continue selecting one
   renderer and saving the same one-file PDF or two-file Markdown bundle. No
   rounded string crosses back into calculation, persistence, or conversion.
   Rendering and PDF finalization complete before path reservation. The output
   package remains the sole owner of exclusive reservation, complete-bundle
   commit, current-attempt cleanup, prior-file retention, and post-save opener
   warnings under FR-024 and FR-025. `internal/tui/screen/` renders every saved
   path and identifies it as cleartext financial data with instructions to delete
   all listed files; leaving the result flow retains no report state or path list.

## Testing Strategy

- Pure presentation and generated-document tests cover the complete financial
  field/vector/output matrix, including positive, negative, zero, whole,
  high-precision, carry, either-side and exact-half, very small, very large, nil,
  non-finite, and negative-zero cases, and assert source decimals are unchanged.
- Shared-row tests cover every monetary field, FR-009 canonical quantities and
  normalized canonical rates,
  `Yes`/`No`, exact-zero converted-entry omission, and inclusion of non-zero
  values that display as `0.00`. Tests prove the received converted-entry order
  remains unchanged without adding list-level validation and prove the existing
  audit currency value remains unchanged before the presentation row is built.
- Report-calculation input tests cover all-missing, mixed missing-and-zero,
  explicit-zero, and all-zero reduction shapes without manufacturing values.
  Sync validation tests separately characterize unchanged admission: explicit
  resolvable zero evidence with an explanation remains accepted, while an
  all-monetary-values-missing upstream activity remains rejected.
- Markdown tests assert the exact bold warning once and in order, exact two-place
  values, valid pipe-table rows, `<br>` converted-entry boundaries, and no
  changed quantity or rate text.
- PDF package tests record ordered layout operations to prove metadata, fully
  bold warning, then summary; assert exact cells rather than punctuation-stripped
  searchable text; compare measured and drawn line counts for long spaced and
  explicit-newline content; prove expanded rows remain above the printable
  bottom margin; prove fit-on-fresh-page and exceeds-fresh-page outcomes; and
  prove PDF finalization failures return through `Renderer.Render` without
  terminating the process, invoking output writing, or reporting success.
- Output tests force Markdown second-file and PDF write, sync, close, validation,
  and bundle-recording failures. They prove current-attempt cleanup, preservation
  of colliding sentinel files and earlier bundles, no partial saved paths, and
  success-with-warning file retention after opener failure.
- Result-screen and workflow contracts prove normal success and opener-warning
  success identify cleartext financial exports, list every saved path, instruct
  deletion of all listed files, and retain no path or report history after exit.
- Extend the project-owned generated-PDF inspector with ordered text runs that
  expose page, font resource, and coordinates. Concrete PDF contract tests use
  those runs to prove every warning fragment including the final period uses the
  embedded bold font and each converted entry begins at a later vertical
  coordinate. Landscape A4 and searchable text assertions remain.
- Runtime integration tests render both formats from the same protected cache
  and compare all AUD-001 fields before and after each renderer while updating
  only rendered monetary expectations and asserting the pre-format audit
  currency remains unchanged before presentation.
- Confidentiality tests apply clearly synthetic SEC-001 credential,
  protected-payload, and user-financial-data sentinels to successful documents,
  errors and wrapped causes, diagnostics, examples, and generated fixtures.
  Contracted report fields remain allowed only in the user export; other channels
  redact real financial data under their existing policies. Readability evidence
  proves complete searchable/selectable text and non-overlapping layout without
  claiming ACC-002-excluded conformance.
- Existing empirical fixtures remain unchanged. The named performance fixture
  independently times both formats, verifies exactly 6,666 three-entry
  conversion rows, and proves the PDF conversion audit spans continuation pages
  with every row, entry, repeated header, and continuation context retained.
  Inspection occurs outside the measured intervals.

## Constitution Prerequisite

The current PDF adapter finalizes through `gopdf.GetBytesPdf`, whose failure path
calls `log.Fatalf`. To satisfy FR-022, FR-023, and the rendering Failure Contract,
change the internal PDF document seam from `Bytes() []byte` to
`Bytes() ([]byte, error)`, finalize through `GetBytesPdfReturnErr`, and propagate
the error through `Renderer.Render` before output writing starts. Add
package-local and runtime-backed coverage for process survival, no output
reservation, and no opener invocation.

This changes no successful document content, format, section, field, or file
bundle, but it is part of the normative report failure contract. It is a minimal
prerequisite discovered by the post-design OWASP A10 and constitution review,
because a report render failure must not terminate the process.

## Complexity Tracking

No constitution violations are planned.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
