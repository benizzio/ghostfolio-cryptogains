# Coding Standards Drift Report: Final Report Adjustments

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-07-18
**Updated**: 2026-07-18
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-control.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- The reviewed implementation slice was derived from `spec.md`, `plan.md`, `tasks.md`, the supporting feature artifacts, and the Feature 009 diff against `origin/main`.

## Standards Baseline

- `AGENTS.md:69-163`: package ownership, report-layer boundaries, test-suite responsibilities, shared test-support placement, and performance-test isolation.
- `AGENTS.md:171-199`: descriptive naming, SOLID and SRP, cohesive decomposition, DRY, consistency, layered architecture, and the production cognitive-complexity threshold.
- `AGENTS.md:201-225`: required documentation and author or co-author information for AI-generated and agent-touched declarations, including detailed public API usage documentation.
- `AGENTS.md:229-238`: repository preference for `var` declarations and the limited `:=` reuse exception.
- `.specify/memory/constitution.md:149-182`: test responsibility, non-duplicative evidence, coverage, and quality-gate principles.
- `.specify/memory/constitution.md:200-213`: Clean Architecture and domain clarity, descriptive modeling, cohesion, minimized duplication, SOLID boundaries, and separation of domain rules from infrastructure.

## Findings

### CODE-STAND-DRIFT-001: Integration Scenarios Bypass Shared Test Support

**Status**: Pending
**Severity**: High
**Diverges from**:

- `AGENTS.md:137,158`: shared runtime-backed scenario and artifact support belongs in `tests/testutil/runtimeflow`, without duplication in scenario packages.
- `AGENTS.md:173-180,189-197`: DRY, SRP, cohesion, and layered architecture.
- `.specify/memory/constitution.md:200-211`: cohesive modules, minimized duplication, explicit concepts, and consistent architecture boundaries.

**Evidence**:

- `tests/integration/report_audit_presentation_flow_test.go:183-485`
- `tests/integration/report_converted_amounts_flow_test.go:210-531`
- `tests/integration/report_generation_flow_test.go:190-211`
- `tests/integration/report_generation_flow_test.go:773-815`
- `tests/integration/report_generation_flow_test.go:899-960`
- `tests/integration/report_value_presentation_flow_test.go:30-48`

**Description**:

🚩 [DIVERGENT] The audit and converted-amount scenarios each implement a separate PDF artifact-processing stack for locating source identifiers, grouping text runs into rows, interpreting coordinates, and selecting semantic cells. The audit scenario additionally mirrors renderer table geometry and column widths. New scenarios also consume fixture, request, path, and output-discovery helpers declared inside the unrelated `report_generation_flow_test.go` scenario file. This creates duplicated maintenance-sensitive infrastructure and hidden package-global coupling instead of using the repository's designated shared runtime-flow and artifact-support boundary.

**Remediation plan**:

Move the integration-neutral mixed-currency fixture, report-request, output-discovery, and PDF row/cell inspection helpers into cohesive files under `tests/testutil/runtimeflow`, and make the integration scenarios consume those shared APIs. Consolidate the duplicated audit and converted-amount text-run grouping there while keeping scenario-specific Markdown parsing, semantic expectations, and AUD-001 assertions in `tests/integration`; preserve fixture values, row geometry, coordinate tolerances, text-run order, and blank-cell handling exactly. Validate the shared package and all affected integration scenarios with `go test ./tests/testutil/runtimeflow ./tests/integration -count=1`.

### CODE-STAND-DRIFT-002: Converted Amounts Use a Stringly Typed Renderer Boundary

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:173-180,182-197`: descriptive domain modeling, DRY, cohesion, and layered architecture.
- `.specify/memory/constitution.md:203-209`: domain concepts must be explicit rather than hidden behind vague helpers or infrastructure-centric representations.

**Evidence**:

- `internal/report/presentation/converted_amounts.go:9-40`
- `internal/report/presentation/rows.go:68-83`
- `internal/report/markdown/renderer_conversion.go:78-107`

**Description**:

🚩 [DIVERGENT] The format-neutral presentation API describes converted amounts as delimiter-free logical entries but returns strings that already contain the `: ` and ` -> ` syntax. The Markdown renderer then parses each string with `LastIndex` and reconstructs the same syntax after escaping its components. The logical entry's label and values are therefore hidden in a string protocol, and delimiter knowledge is duplicated across presentation and rendering instead of being represented explicitly at the package boundary.

**Remediation plan**:

Introduce an explicit `ConvertedAmountEntry` presentation type with separate label, original-amount, and converted-amount fields; return and store that type through `ConvertedAmounts` and `ConversionAuditRow`, then let Markdown and PDF assemble their existing fixed delimiters directly. Remove the Markdown string parser while preserving exact zero omission, received order, financial formatting, contextual errors, Markdown component escaping, whole-entry PDF sanitization, PDF reverse iteration, and generated output bytes. Validate presentation, Markdown, PDF, converted-amount contracts, and converted-amount integration flows.

### CODE-STAND-DRIFT-003: Successful Report Copy Has Competing Runtime and TUI Owners

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:76-79,123-127`: runtime owns orchestration while TUI components and screens own workflow copy and result rendering.
- `AGENTS.md:179-180`: avoid duplication and remain consistent.
- `.specify/memory/constitution.md:206-210`: modules must remain cohesive, minimize duplication, and use consistent ownership boundaries.

**Evidence**:

- `internal/app/runtime/report_output_outcome.go:149-170`
- `internal/tui/component/workflow_copy.go:100-112`
- `internal/tui/screen/report_screen.go:267-326`

**Description**:

🚩 [DIVERGENT] Runtime success and opener-warning messages contain saved-path and cleartext deletion guidance, while the report-result screen separately owns cleartext disclosure, path labels, and deletion guidance. The screen appends the runtime message between its own disclosure and deletion strings, so normal runtime outcomes repeat deletion guidance. User-visible success copy consequently has competing owners across orchestration and presentation layers.

**Remediation plan**:

Reduce runtime success and opener-warning messages to operational save/open status and redacted opener detail only, removing path, cleartext, and deletion text from `internal/app/runtime`. Keep cleartext disclosure, role-specific path rendering, and deletion guidance owned by the TUI component/screen, including the legacy single-output fallback; preserve success classification, `OpenError`, saved files, cleanup, and retry behavior. Add focused runtime, screen, workflow-contract, and integration assertions that every path plus disclosure and deletion guidance appears exactly once for normal and opener-warning success.

### CODE-STAND-DRIFT-004: Report PDF Rendering Exceeds the Complexity Threshold

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:177-180,185-188`: production cognitive complexity should remain below 15 and functions should be decomposed by responsibility when it exceeds that threshold.
- `.specify/memory/constitution.md:206-207`: functions must remain cohesive and respect SOLID boundaries.

**Evidence**:

- `internal/report/pdf/main_report.go:104-141`
- `.golangci.yml:31-37`

**Description**:

🚩 [DIVERGENT] `gocognit v1.2.0 -over 15 internal/report/pdf/main_report.go` reports cognitive complexity 16 for `renderRateSourceSection`. The same function on `origin/main` reports 14; the Feature 009 empty-state branch moved it above the written repository threshold. The configured golangci-lint gate does not report this exact score because its `min-complexity: 16` comparison is exclusive, but the implementation still diverges from the explicit `AGENTS.md` rule.

**Remediation plan**:

Extract only the empty-rate-source paragraph branch into a private helper and return that helper directly from `renderRateSourceSection`, reducing the parent function to cognitive complexity 14 without changing non-empty provider traversal. Change `.golangci.yml` from `min-complexity: 16` to `min-complexity: 14` because the adapter reports only scores greater than the configured value, thereby rejecting production complexity 15 and above while retaining the existing test-file exclusion. Preserve operation order, paragraph and error text, duplicate suppression, and source order; validate the lint configuration, existing main-report tests, the standalone `gocognit v1.2.0 -test=false -over 14 internal/report/pdf/main_report.go` result, and the changed-source quality gate.

### CODE-STAND-DRIFT-005: Acceptance Fixture Combines Independent Responsibilities

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:176-180,189-197`: SRP applies to Go files as well as functions, and independently changing responsibilities should remain cohesive.
- `.specify/memory/constitution.md:206-211`: modules must remain cohesive, minimize duplication, and document deliberate consistency deviations.

**Evidence**:

- `tests/testutil/report_presentation_fixtures.go:6-197`
- `tests/testutil/report_presentation_fixtures.go:199-451`
- `tests/testutil/report_presentation_fixtures.go:454-928`
- `tests/testutil/report_presentation_fixtures.go:930-1088`

**Description**:

🚩 [DIVERGENT] One test-support file owns the exported acceptance schema and taxonomy, case-catalog construction, semantic occurrence generation and accounting, and the financial matrices and numeric vectors. These sections change for different reasons and serve separate consumers. Their colocation broadens every review and couples catalog, accounting, and model evolution beyond a cohesive fixture responsibility.

**Remediation plan**:

Split `report_presentation_fixtures.go` within package `testutil` into cohesive schema/taxonomy, catalog construction, occurrence accounting, financial matrix/vector, and scalar-case files after the related key and counter APIs are simplified. Move declarations without changing visibility, names, case ordering, occurrence ordering, vector values, fresh allocation behavior, or type/method/builder cohesion. Validate the testutil package and the closed-manifest, acceptance-accounting, Annex, and converted-amount contracts.

### CODE-STAND-DRIFT-006: PDF Inspection Support Spans Multiple Parser Layers

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:176-180,189-197`: SRP, cohesive decomposition, DRY, and separation of layered responsibilities.
- `.specify/memory/constitution.md:206-209`: modules must remain cohesive and separate infrastructure concerns where that improves change safety.

**Evidence**:

- `tests/testutil/pdf_inspection.go:128-298`
- `tests/testutil/pdf_inspection.go:300-497`
- `tests/testutil/pdf_inspection.go:499-699`
- `tests/testutil/pdf_inspection.go:701-861`

**Description**:

🚩 [DIVERGENT] Feature 009 expanded the same test-support file across PDF object and page resolution, text-state and operator parsing, CMap and TrueType decoding, stream decompression, literal decoding, and search normalization. Ordered text-run extraction now changes in the same declaration unit as lower-level font and object parsing, creating multiple independent parser responsibilities in one file.

**Remediation plan**:

Split `pdf_inspection.go` within package `testutil` into the public inspection/search API, object and page resolution, text-run/operator parsing, literal decoding, CMap decoding, TrueType decoding, and stream decompression files. Keep each stateful type with its methods and move code without changing exported APIs, object/page/event ordering, coordinate interpretation, decoder fallback order, stream close behavior, normalization, or error strings. Validate all PDF inspector tests plus contract and integration consumers.

### CODE-STAND-DRIFT-007: Acceptance Keys Use Error-Prone Positional String APIs

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:175,178-180`: descriptive and unambiguous names, cohesive functions, and DRY.
- `.specify/memory/constitution.md:203-207,210-211`: explicit domain concepts, cohesive functions, minimized duplication, and consistency.

**Evidence**:

- `tests/testutil/report_presentation_fixtures.go:646-732`
- `tests/testutil/report_presentation_fixtures.go:735-847`
- `tests/testutil/report_presentation_fixtures.go:851-894`

**Description**:

🚩 [DIVERGENT] `newPresentationCase` accepts nine positional arguments and `formatOccurrenceKeys` accepts ten, including several adjacent strings that are commonly supplied as repeated empty placeholders. Argument transposition is not type-checked, and call sites are difficult to interpret without repeatedly consulting the declaration. `formatOccurrenceKeys` also has two branches that construct the same one-element result, adding duplication without a distinct abstraction.

**Remediation plan**:

Change `newPresentationCase` to accept a named `ReportPresentationAcceptanceCase` value and retain only its cloning, attempt initialization, and base-key behavior. Delete `formatOccurrenceKeys` and append named `ReportPresentationOccurrenceKey` literals at its callers because both existing branches return one identical-shape key; preserve all empty identity fields, amount ordinals, append order, slice cloning, and occurrence cardinality. Validate the closed manifest and aggregate population contracts before the fixture files are split.

### CODE-STAND-DRIFT-008: Population Counters Duplicate Their Domain Mapping

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:173-180`: descriptive naming, DRY, and consistency.
- `.specify/memory/constitution.md:203-207,210-211`: explicit concepts, minimized duplication, and mandatory consistency.

**Evidence**:

- `tests/testutil/report_presentation_fixtures.go:74-108`
- `tests/testutil/report_presentation_fixtures.go:173-189`
- `tests/testutil/report_presentation_fixtures.go:896-927`
- `tests/contract/report_rendering_values_contract_test.go:334-360`

**Description**:

🚩 [DIVERGENT] Descriptive population constants are reduced to eleven single-letter counter fields, then mapped once while counting and mapped independently in a contract helper. The reverse mapping's default silently returns zero. Adding or renaming a population therefore requires synchronized switch changes that the type system cannot enforce, despite the descriptive population type already representing the domain concept.

**Remediation plan**:

Replace the letter-named counter fields with `CaseCount` plus a freshly allocated `map[ReportPresentationPopulation]int`, count occurrences directly by their typed population, and remove the duplicate contract switch. Require explicit comma-ok lookups where a population must exist and exact map equality where extra populations must fail; preserve every population's current string value and numerator/denominator. Validate acceptance accounting and the closed-manifest, Annex, and converted-amount population contracts.

### CODE-STAND-DRIFT-009: Performance Tests Duplicate Deterministic Document Contracts

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:66,134,156-160`: performance tests provide resource-sensitive evidence only and remain isolated from deterministic test and coverage responsibilities.
- `AGENTS.md:179-180`: avoid duplicated behavior checks and remain consistent.
- `.specify/memory/constitution.md:153-166,206-211`: testing responsibilities should not substantially duplicate behavior, and modules must remain cohesive.

**Evidence**:

- `tests/performance/report_performance_flow_test.go:90-120`
- `tests/performance/report_performance_flow_test.go:210-259`
- `tests/performance/report_performance_flow_test.go:261-326`

**Description**:

🚩 [DIVERGENT] The isolated performance scenario now verifies exact warning and currency text, converted-entry labels and delimiters, exact rate occurrences, output cardinality, and repeated table headings. These are deterministic document-contract checks already owned by contract and integration suites rather than timing, responsiveness, bounded lookup, or resource evidence. The performance suite consequently has a second behavioral-contract responsibility.

**Remediation plan**:

Remove exact document text, delimiter, row-count, heading, rate-occurrence, and bundle-cardinality assertions from the performance scenario while retaining the exact 10,000-activity workload, local deterministic inputs, separate Markdown/PDF generation timers, strict duration limits, successful non-empty output, opener invocation, environment logging, and performance-coverage isolation. Update the feature spec, plan, research, rendering contract, quickstart, and existing performance-task wording so deterministic content evidence remains assigned to package, contract, and integration tests. Validate those deterministic owners and run `make test-performance` separately.

### CODE-STAND-DRIFT-010: Agent-Touched Declarations Have Incomplete Documentation and Attribution

**Status**: Pending
**Severity**: Medium
**Diverges from**:

- `AGENTS.md:203-223`: every AI-generated and agent-touched function, method, component, type, and package requires purpose documentation and author or co-author information.
- `AGENTS.md:224-225`: public cross-package APIs require detailed usage instructions.
- `.specify/memory/constitution.md:210-211`: consistency is mandatory unless a deviation is documented and justified.

**Evidence**:

- `internal/report/presentation/rows.go:1-3`
- `internal/report/presentation/rows.go:86-89`
- `internal/report/presentation/rows.go:147-150`
- `internal/report/presentation/rows.go:171-174`
- `internal/report/presentation/rows.go:226-229`
- `internal/report/pdf/layout_contract.go:39-47`
- `internal/report/pdf/renderer_internal_test.go:1961-1964`
- `internal/report/pdf/renderer_internal_test.go:1990`
- `internal/tui/component/workflow_copy.go:104-112`
- `tests/testutil/report_presentation_fixtures.go:111-197`
- `tests/performance/helpers_test.go:39-69`

**Description**:

🚩 [DIVERGENT] The presentation package description still covers only table values although Feature 009 added financial formatting, converted-entry processing, and legal-warning policy. Branch-touched exported row builders, workflow-copy constants, and public test-support models have summaries but not the required detailed cross-package usage and invariant documentation. The feature-touched PDF `Bytes` seam declarations and `largeReportFixture` function omit declaration-level purpose or author/co-author information entirely. The drift is systematic across production and test support rather than one isolated missing comment.

**Remediation plan**:

After the structural and ownership remediations settle declaration locations, update the presentation package comment and the cited exported row builders, workflow-copy constants, and test-support models with accurate responsibilities, invariants, and public usage examples where required. Add purpose documentation and OpenCode author/co-author attribution to the cited PDF finalization seams, test doubles, and performance fixture helper, preserving existing authorship and changing no signatures, strings, or behavior. Run `gofmt` and focused tests for presentation, PDF, TUI components, testutil, and the performance package.

### CODE-STAND-DRIFT-011: PDF Renderer Tests Mix Unrelated Production Responsibilities

**Status**: Pending
**Severity**: Low
**Diverges from**:

- `AGENTS.md:177-180,189-194`: SRP and cohesive decomposition apply to Go files; file length alone is not the criterion.
- `.specify/memory/constitution.md:206-207`: modules must remain cohesive and respect SOLID boundaries.

**Evidence**:

- `internal/report/pdf/renderer_internal_test.go:269-636`
- `internal/report/pdf/renderer_internal_test.go:730-970`
- `internal/report/pdf/renderer_internal_test.go:1014-1075`
- `internal/report/pdf/renderer_internal_test.go:1478-1495`
- `internal/report/pdf/renderer_internal_test.go:1689-1818`

**Description**:

🚩 [DIVERGENT] Feature 009 added separate test groups for table measurement and pagination, warning and financial presentation, byte finalization, main-report failures, and bold-paragraph layout to the same package-local file. These groups follow distinct production responsibilities owned by different PDF source files. The finding concerns the responsibility spread introduced by the feature, not a numeric file-length limit.

**Remediation plan**:

Move the cited test functions, unchanged and still in package `pdf`, into responsibility-specific files for table layout/pagination, main-report composition, PDF presentation, byte finalization, and bold-paragraph layout. Keep shared package-local recorders, fixtures, and test doubles single-sourced unless moving them is required for type/method cohesion; preserve test names, seam restoration, ordering, and package-private access, and do not add parallel execution. Compare discovered tests before and after the move and run the full `internal/report/pdf` suite.

### CODE-STAND-DRIFT-012: Report Tests Duplicate Shared Setup and Fixed Presentation Text

**Status**: Pending
**Severity**: Low
**Diverges from**:

- `AGENTS.md:137,158,179-180`: shared report test support should remain cohesive in `tests/testutil`, and duplication should be avoided.
- `.specify/memory/constitution.md:206-210`: modules must minimize duplication and remain consistent.

**Evidence**:

- `tests/testutil/report_presentation_fixtures.go:10-20`
- `tests/contract/report_annex_contract_test.go:84-101`
- `tests/contract/report_converted_amounts_contract_test.go:79-94`
- `tests/contract/report_rendering_confidentiality_test.go:71-85`
- `tests/contract/report_rendering_confidentiality_test.go:112-127`
- `tests/contract/report_rendering_confidentiality_test.go:215-229`
- `tests/integration/report_value_presentation_flow_test.go:20-23`
- `tests/unit/report_markdown_test.go:131-159`
- `tests/performance/report_performance_flow_test.go:210-220`

**Description**:

🚩 [DIVERGENT] Contract files repeatedly construct the same font-backed PDF renderer, render a fixture, and inspect the resulting bytes rather than using one shared helper. The exact legal-warning sentence is also redeclared or embedded in integration, unit, and performance suites even though the feature already exposes a shared test fixture constant. These duplicated setup and literal copies can drift independently as renderer construction or presentation text changes.

**Remediation plan**:

Add a dedicated `tests/testutil/reportpdf` helper package that constructs the standard font-backed renderer and optionally renders then inspects a successful report, avoiding an import cycle with package-local PDF tests. Replace duplicated contract setup where renderer lifetime and failure assertions permit, retain renderer reuse in the 148-case acceptance test, and replace surviving test-owned warning copies with the independent `testutil.ReportPresentationLegalWarningText` fixture constant; do not source expected text from production. Validate contract, integration, unit, Markdown, and performance compilation paths affected by the shared support.

### CODE-STAND-DRIFT-013: Names and Comments Lag the Current Behavior

**Status**: Pending
**Severity**: Low
**Diverges from**:

- `AGENTS.md:175,180,203-208`: names and documentation must be descriptive, unambiguous, accurate, and consistent.
- `.specify/memory/constitution.md:203-211`: domain concepts and documentation must remain explicit and consistent.

**Evidence**:

- `internal/tui/screen/report_screen.go:267-308`
- `internal/report/pdf/renderer_internal_test.go:1954-1955`
- `internal/report/pdf/renderer_internal_test.go:1986-1987`

**Description**:

🚩 [DIVERGENT] `reportOutputBundleSummary` can receive the legacy single `OutputFile` fallback returned by `reportOutputFiles`, so its name asserts bundle-only semantics that its accepted input no longer guarantees. Two Feature 009 test-double comments describe `AddBoldParagraph` as a “future” warning seam even though the seam exists and is used in the same implementation revision.

**Remediation plan**:

Rename `reportOutputBundleSummary` to `reportSuccessfulOutputSummary` and update its documentation to cover both bundle files and the legacy single `OutputFile` fallback, without changing bundle precedence, path labels, disclosure, deletion guidance, or no-file fallback behavior. Correct the two `AddBoldParagraph` test-double comments to describe their current legal-warning operation and injected-failure roles, coordinating with the PDF test-file split. Validate TUI fallback/success rendering and the affected PDF seam tests.

### CODE-STAND-DRIFT-014: Feature-Added Declarations Bypass the `var` Preference

**Status**: Pending
**Severity**: Low
**Diverges from**:

- `AGENTS.md:229-236`: prefer `var` over `:=` except for multiple declarations with subsequent reuse.

**Evidence**:

- `internal/report/presentation/converted_amounts_test.go:64,85,108,134,171,195`
- `internal/report/presentation/financial_test.go:45,53,65,84,102,120,152,232,286,452`
- `internal/app/runtime/report_service_internal_test.go:416`
- `internal/tui/screen/report_screen.go:322`
- `tests/integration/report_converted_amounts_flow_test.go:128-136`
- `tests/integration/report_converted_amounts_flow_test.go:171-181`

**Description**:

🚩 [DIVERGENT] Branch-added tests and one production screen path repeatedly use standalone short declarations or `if` initializers where no later redeclaration or reuse satisfies the documented exception. The pattern is limited in structural impact but is inconsistent with the repository's explicit declaration-style preference.

**Remediation plan**:

After the overlapping structural, typed-entry, and TUI changes, convert the report-cited standalone short declarations and `if` initializers to explicit `var` declarations while retaining one evaluation, scope-safe descriptive names, subtest capture, assertion order, and existing error handling. Do not rewrite range clauses or legitimate multi-assignment/reuse sites covered by the documented exception. Run `gofmt` and focused presentation, runtime, screen, and converted-amount integration tests.

## Notes

- No earlier `coding-standards-drift-report.md` existed, so all identifiers were allocated sequentially as new pending findings.
- All normal Feature 009 checklist tasks `T001` through `T049` were checked before this review. References to “unchecked tasks” in `tasks.md` are orchestration instructions, not open tasks.
- No additional proprietary agent-instruction files were discovered in repository or feature scope beyond `AGENTS.md`; `.specify/memory/constitution.md` was also loaded as required.
- Test code was not evaluated against the production cognitive-complexity threshold because `AGENTS.md:188` explicitly exempts it.
- Focused remediation plans and Phase 7 tasks T050-T065 were added after the normal implementation completed; all finding statuses remain `Pending` until the final successful-remediation update in T065.
