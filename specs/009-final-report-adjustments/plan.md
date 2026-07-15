# Implementation Plan: Final Report Adjustments

**Branch**: `[009-final-report-adjustments]` | **Date**: 2026-07-15 | **Spec**: `/specs/009-final-report-adjustments/spec.md`

**Input**: Feature specification from `/specs/009-final-report-adjustments/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Apply the release-ready report presentation rules to the existing Markdown and
PDF renderers without changing report calculation. A report-specific shared
presentation helper will format every currency-denominated amount and unit price
to two decimal places with HALF UP rounding while leaving quantities and
provider-published exchange rates canonical. Shared presentation rows will also
produce `Yes`/`No` boolean labels and ordered converted-amount entries. Markdown
will encode controlled table-cell breaks as `<br>`, while PDF will encode them
as explicit newlines measured with the same word-wrap policy used for drawing.
The exact legal-use warning will be inserted once in each main report, and
zero-priced holding reductions will retain their existing pre-format audit
currency value while omitting that non-applicable value from the visible Annex 1
cell.

## Technical Context

**Language/Version**: Go 1.26.5

**Primary Dependencies**: Existing `github.com/cockroachdb/apd/v3 v3.2.3`
for exact decimals and HALF UP quantization,
`github.com/signintech/gopdf v0.36.1` for local structured PDF rendering,
`golang.org/x/image v0.43.0` for application-supplied PDF fonts, and the Go
standard library. No dependency or version change is planned.

**Storage**: Existing user-requested cleartext Markdown and PDF exports in the
resolved local Documents directory. Exports retain the existing writer's
requested `0600` mode and remain outside the established application-managed
persistence boundary. No new cache, snapshot field, report history, telemetry,
remote storage, or automatic report re-ingestion is added.

**Testing**: Go `testing` across package-local report tests, `tests/unit`,
`tests/contract`, `tests/integration`, existing `tests/empirical`, and the
isolated build-tagged `tests/performance` suite. Package-local renderer tests
will verify exact formatting, PDF style intent, operation order, controlled line
breaks, matching PDF measurement/drawing wrap behavior, and row height. Contract
tests will inspect generated PDF text runs, fonts, and coordinates to prove
complete bold warning output and separate converted-entry lines. Integration
tests will verify generated Markdown/PDF parity and unchanged calculation
evidence. Final gates are `make test`, `make coverage`,
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

**Performance Goals**: Preserve independent generation of one Markdown bundle
and one PDF document from 10,000 cached activities in under two minutes per
selected format.

**Constraints**: Presentation-only rounding at scale 2 with `apd.RoundHalfUp`;
no floating-point financial values; no mutation or reuse of rounded values in
calculation; exact-zero decisions occur before display formatting; quantities
and disclosed exchange rates retain established precision; nil optional amounts
remain blank; rounded negative zero becomes `0.00`; no new dependency, network,
authentication, persistence, report section, table column, or output format;
existing landscape A4 PDF pagination, searchable text, embedded fonts, complete
row preflight, output reservation/cleanup sequence, and secret redaction remain
intact; the constitution-required PDF finalization seam must return errors instead
of terminating the process; production statement, line, branch, and per-file
coverage remains 100% as enforced by the repository gate.

**Scale/Scope**: One existing calculated report, one Markdown main document and
separate Annex 1 document, one combined PDF document, all main-report and Annex
financial fields, one structured boolean field currently exposed by Annex 1,
one transient audit classification copied into the calculated report model, and
one converted-amount cell containing up to the validated ordered component set
`unit_price`, `gross_value`, and `fee_amount`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS
Post-design gate status: PASS

- [x] Security: Under the repository's established boundary, reports remain
  explicit cleartext user-owned exports written with requested mode `0600`, not
  application-managed persistence. Token handling, protected snapshots, output
  reservation/cleanup, and redaction are unchanged.
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
  `Rounded` and `Inexact` conditions. Quantities and rates continue through
  canonical formatting. Each existing pre-format activity currency value remains
  unchanged before rendering. No conversion boundary, rate source, cost-basis
  rule, calculation scale, or stored value changes.
- [x] Testing: Integration and contract tests cover the same deterministic
  report rendered as Markdown and PDF, including warning placement, all monetary
  field classes, quantity/rate preservation, boolean labels, audit currency,
  and one-to-three converted entries. Targeted package tests are justified for
  the pure decimal display boundary and PDF font/layout seams. `make coverage`
  remains the canonical 100% coverage gate.
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
  introduced.
- [x] Architecture: Report-specific financial display policy and format-neutral
  rows stay in `internal/report/presentation/`; Markdown syntax stays in
  `internal/report/markdown/`; PDF styling, sanitization, measurement, and
  pagination stay in `internal/report/pdf/`; the existing zero-priced audit
  classification is copied by `internal/report/calculate/` into the transient
  report model under `internal/report/model/`, and only presentation suppresses
  the visible non-applicable currency. Runtime and output writing remain
  unchanged.

## Project Structure

### Documentation (this feature)

```text
specs/009-final-report-adjustments/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── report-rendering.md
└── tasks.md             # Created later by /speckit.tasks
```

### Source Code (repository root)
```text
internal/
├── report/
│   ├── model/           # Retain pre-format currency, copy zero-priced classification, validate converted kind order
│   ├── calculate/       # Populate the transient audit classification without changing financial results
│   ├── presentation/    # Shared financial display strings, boolean labels, and logical converted entries
│   ├── markdown/        # Warning emphasis, direct summary/position values, and controlled <br> table breaks
│   └── pdf/             # Bold warning, matched wrap measurement, row preflight, and safe finalization prerequisite
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

1. `internal/report/calculate/` continues to produce exact `CapitalGainsReport`
   values. While constructing an Annex audit row, it retains the existing
   `ActivityCurrency` value and copies the existing
   `IsZeroPricedHoldingReduction` classification into the transient audit model.
   A zero-priced reduction may have no selected source currency, so this feature
   does not invent provenance. No financial value, currency identity, or
   activity inclusion changes before presentation.
2. `internal/report/presentation/` formats monetary values from defensive
   decimal copies at two places with HALF UP, canonicalizes quantities and
   rates through existing helpers, maps the report boolean to `Yes` or `No`, and
   returns converted amount entries as an ordered logical list after exact
   zero-to-zero omission. It blanks only the visible original currency for an
   audit row carrying the exact zero-priced-reduction classification. Report
   model validation rejects duplicate or non-canonical converted amount kinds.
3. `internal/report/markdown/` inserts the exact fully bold warning after report
   metadata, uses the shared financial formatter for summary and position values
   that bypass row builders, escapes and sanitizes each dynamic converted-entry
   component before assembling the fixed `: ` and ` -> ` syntax, and joins
   entries with `;<br>` so a pipe-table row remains structurally valid and only
   the renderer can introduce that HTML delimiter.
4. `internal/report/pdf/` inserts the same warning through a dedicated bold
   wrapped-paragraph operation, uses the same shared financial formatter for
   direct values, sanitizes each converted entry, and joins entries with `;\n`.
   Table-cell measurement first applies `SplitTextWithOption` using the pinned
   table layout's indicator-sensitive space break option, then measures those
   exact lines with `IsFitMultiCellWithNewline`, so preflight and drawing use the
   same wrap result.
5. `internal/app/runtime/` and `internal/report/output/` continue selecting one
   renderer and saving the same one-file PDF or two-file Markdown bundle. No
   rounded string crosses back into calculation, persistence, or conversion.

## Testing Strategy

- Pure presentation tests cover positive, negative, zero, whole, high-precision,
  exact-half, very small, very large, nil, non-finite, and negative-zero cases,
  and assert source decimals are unchanged.
- Shared-row tests cover every monetary field, canonical quantities and rates,
  `Yes`/`No`, exact-zero converted-entry omission, and inclusion of non-zero
  values that display as `0.00`. Model tests reject duplicate or out-of-order
  converted amount kinds and prove the existing audit currency value remains
  unchanged before the presentation row is built.
- Markdown tests assert the exact bold warning once and in order, exact two-place
  values, valid pipe-table rows, `<br>` converted-entry boundaries, and no
  changed quantity or rate text.
- PDF package tests record ordered layout operations to prove metadata, fully
  bold warning, then summary; assert exact cells rather than punctuation-stripped
  searchable text; compare measured and drawn line counts for long spaced and
  explicit-newline content; prove expanded rows remain above the printable
  bottom margin; and prove PDF finalization failures return through
  `Renderer.Render` without terminating the process.
- Extend the project-owned generated-PDF inspector with ordered text runs that
  expose page, font resource, and coordinates. Concrete PDF contract tests use
  those runs to prove every warning fragment including the final period uses the
  embedded bold font and each converted entry begins at a later vertical
  coordinate. Landscape A4 and searchable text assertions remain.
- Runtime integration tests render both formats from the same protected cache
  and retain exact calculated-model assertions while updating only rendered
  monetary expectations and asserting the pre-format audit currency remains
  unchanged before presentation.
- Existing empirical fixtures remain unchanged, and the existing performance
  scenario independently times both selected formats at 10,000 activities.

## Constitution Prerequisite

The current PDF adapter finalizes through `gopdf.GetBytesPdf`, whose failure path
calls `log.Fatalf`. Before feature completion, change the internal PDF document
seam from `Bytes() []byte` to `Bytes() ([]byte, error)`, finalize through
`GetBytesPdfReturnErr`, and propagate the error through `Renderer.Render` before
output writing starts. Add package-local coverage for the error path.

This is isolated from the external report rendering contract: it changes no
successful document content, format, section, field, or file bundle. It is a
minimal prerequisite discovered by the post-design OWASP A10 and constitution
review, because a report render failure must not terminate the process.

## Complexity Tracking

No constitution violations are planned.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
