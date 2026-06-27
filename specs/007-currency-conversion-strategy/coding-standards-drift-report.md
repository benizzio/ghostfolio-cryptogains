# Coding Standards Drift Report: Report Base Currency Conversion

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-06-27
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-control.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Standards Baseline

- `AGENTS.md:64-75`: application orchestration belongs in `internal/app/runtime/`; shared support helpers belong in `internal/support/`; domain-neutral helpers should be reused or extended there without moving package-specific domain rules into support.
- `AGENTS.md:95-101`: report request, calculated report models, report documents, cost-basis, calculation, Markdown rendering, and output concerns are separated under `internal/report/`; report-specific financial rules must stay out of runtime, TUI, and generic support helpers.
- `AGENTS.md:141-171`: implementation must follow Clean Code, SOLID, DRY, cohesion, consistency, SRP, and layered architecture; Go files should contain code related to a single responsibility, domain/application code should not be mixed with infrastructure, utility, or reusable code, and Go functions over cognitive complexity 15 require SRP/decomposition analysis with `github.com/uudashr/gocognit`.
- `AGENTS.md:175-199`: AI-generated public API code must contain language-standard documentation, authoring information, and detailed usage instructions when usable by other packages.
- `.specify/memory/constitution.md:149-162`: code must follow Clean Code, Domain-Driven Design, and Clean Architecture; names and domain concepts must be explicit; modules and functions must remain cohesive, minimize duplication, respect SOLID boundaries, and separate domain rules from IO and infrastructure concerns.
- `.specify/memory/constitution.md:164-190`: operational constraints prohibit unsupported practices including undocumented API assumptions, floating-point ledger math, unreviewed dependencies, and persistence or sensitive-data handling outside the documented boundaries.

## Findings

### CODE-STAND-DRIFT-001: Report Models Depend On Integration Provider Types

**Severity**: High
**Diverges from**:

- `AGENTS.md:95-101`: report models own report-domain request, report, document, and validation models; provider integration details should not be embedded into report-domain model ownership.
- `.specify/memory/constitution.md:149-162`: Clean Architecture requires domain rules to stay separated from IO and infrastructure concerns, with cohesive modules and explicit domain concepts.

**Evidence**:

- `internal/report/model/conversion_audit.go:11-59`
- `internal/report/model/conversion_audit.go:92-137`
- `internal/report/model/conversion_audit.go:172-223`
- `internal/report/markdown/renderer.go:10-14`
- `internal/report/markdown/renderer.go:385-404`

**Description**:

The report model package imports `internal/integration/currency` and stores integration-layer provider identifiers, authorities, and quote-direction types directly in report-domain structs and validators. The Markdown renderer also imports the integration package to render provider-facing labels and unavailable-date rules. That makes report-domain validation and rendering depend on the integration package's provider type set instead of on report-owned evidence concepts. The dependency direction increases coupling: changes to provider identity, label helpers, or integration-level enum ownership can require report model and renderer changes even when the report contract only needs stable audit evidence.

### CODE-STAND-DRIFT-002: Currency Service File Mixes Contract, Orchestration, Transport, Classification, And Parsing

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:147-171`: Clean Code and SRP require cohesive files and functions, decomposition by responsibility, and layered separation.
- `.specify/memory/constitution.md:149-162`: modules and functions must remain cohesive and separate domain rules from IO and infrastructure concerns.

**Evidence**:

- `internal/integration/currency/service.go:35-85`
- `internal/integration/currency/service.go:175-215`
- `internal/integration/currency/service.go:239-300`
- `internal/integration/currency/service.go:315-342`
- `internal/integration/currency/service.go:385-429`

**Description**:

`service.go` owns the public lookup request and service contract, provider injection, cache-aware lookup orchestration, conversion failure classification, provider base-currency validation, raw HTTP payload fetching, and provider-rate decimal parsing. These concerns are all related to currency integration, but they are not one file-level responsibility. The current structure weakens locality and makes unrelated changes, such as changing transport behavior or failure classification, touch the same file as public service contract changes.

### CODE-STAND-DRIFT-003: Markdown Renderer Accumulates Multiple Rendering Responsibilities In One File

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:147-171`: Clean Code, SRP, decomposition, and file-level cohesion require code to be split when a file owns multiple responsibilities.
- `.specify/memory/constitution.md:149-162`: modules and functions must remain cohesive, minimize duplication, and respect SOLID boundaries where they improve clarity and change safety.

**Evidence**:

- `internal/report/markdown/renderer.go:19-28`
- `internal/report/markdown/renderer.go:76-107`
- `internal/report/markdown/renderer.go:119-231`
- `internal/report/markdown/renderer.go:233-369`
- `internal/report/markdown/renderer.go:406-534`

**Description**:

`renderer.go` now owns the top-level render pipeline, test seams for each major section, report header rendering, summary rendering, reference rendering, detail activity rendering, conversion audit rendering, provider source-summary formatting, liquidation rendering, decimal formatting, currency-label fallback, display-label fallback, activity-currency classification, and inline sanitization. The conversion feature added provider summary and conversion audit behavior into the same file instead of keeping rendering responsibilities in narrower files. The test seam block at the top shows that the file already has independent rendering units, but they are still colocated in one broad module.

### CODE-STAND-DRIFT-004: Production Functions Exceed Cognitive Complexity Threshold

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:147-162`: Clean Code and SRP require decomposition when Go cognitive complexity exceeds 15, measured with `github.com/uudashr/gocognit`.
- `.specify/memory/constitution.md:149-162`: modules and functions must remain cohesive and respect SOLID boundaries where those boundaries improve clarity and change safety.

**Evidence**:

- `internal/integration/currency/ecb_mapper.go:19-64`
- `internal/report/model/conversion_audit.go:175-223`

**Description**:

Running `go run github.com/uudashr/gocognit/cmd/gocognit@latest -test=false -over 15 internal` reports `MapECBEXRCSVToEvidence` at cognitive complexity 17 and `ConversionAuditEntry.Validate` at cognitive complexity 16. Both are production functions above the repository threshold. The drift is not the numeric value alone; the measured result indicates these functions need SRP and decomposition review under the local standard before the feature slice is considered coding-standards clean.

## Notes

- No existing `coding-standards-drift-report.md` was present, so identifiers start at `CODE-STAND-DRIFT-001`.
- The local prerequisite check returned `FEATURE_DIR=/home/benizzio/src-workspace-opencode/ghostfolio-cryptogains/specs/007-currency-conversion-strategy`.
- All implementation task checkboxes in `tasks.md` are checked. Checked `Reopened` annotations were treated as historical bugfix markers rather than open task state.
- No additional proprietary agent-instruction files were discovered in scope beyond `AGENTS.md`; `.specify/memory/constitution.md` was loaded as the governing constitution baseline.
- `gocognit` was invoked through Go packaging with `go run github.com/uudashr/gocognit/cmd/gocognit@latest -test=false -over 15 internal`; the command exits nonzero when findings exceed the threshold, which happened for `CODE-STAND-DRIFT-004`.
