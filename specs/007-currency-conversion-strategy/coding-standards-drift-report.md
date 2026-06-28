# Coding Standards Drift Report: Report Base Currency Conversion

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-06-28
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-control.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Standards Baseline

- `AGENTS.md:64-75`: application orchestration belongs in `internal/app/runtime/`; shared support helpers belong in `internal/support/`; domain-neutral helpers should be reused or extended there without moving package-specific domain rules into support.
- `AGENTS.md:95-101`: report request, calculated report models, report documents, cost-basis, calculation, Markdown rendering, and output concerns are separated under `internal/report/`; report-specific financial rules must stay out of runtime, TUI, and generic support helpers.
- `AGENTS.md:147-171`: implementation must follow Clean Code, SOLID, DRY, cohesion, consistency, SRP, and layered architecture; Go files should contain code related to a single responsibility, domain/application code should not be mixed with infrastructure, utility, or reusable code, and Go functions over cognitive complexity 15 require SRP/decomposition analysis with `github.com/uudashr/gocognit`.
- `.specify/memory/constitution.md:149-162`: code must follow Clean Code, Domain-Driven Design, and Clean Architecture; names and domain concepts must be explicit; modules and functions must remain cohesive, minimize duplication, respect SOLID boundaries, and separate domain rules from IO and infrastructure concerns.

## Findings

### CODE-STAND-DRIFT-005: Report Construction Bypasses Conversion Artifact Validation

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:95-101`: report-domain models own calculated report models and report-domain validation.
- `.specify/memory/constitution.md:149-162`: domain concepts and functions must remain cohesive, explicit, and separated so business rules remain testable and replaceable.

**Evidence**:

- `internal/report/model/capital_gains_report.go:15-54`
- `internal/report/model/capital_gains_report.go:91-121`
- `internal/report/calculate/calculator.go:128-147`

**Description**:

`NewCapitalGainsReport` constructs a calculated report and calls `Validate` before returning it. `Validate` includes conversion-artifact checks for `RateSources` and `ConversionAuditEntries`. The calculator then assigns `report.ConversionAuditEntries` and `report.RateSources` after the constructor has already returned. That makes the report model's validation boundary non-cohesive: the constructor claims to create a validated report, but two validation-sensitive fields are populated outside that boundary. This weakens the model invariant and forces later callers, such as the Markdown renderer, to become the effective validation backstop.

### CODE-STAND-DRIFT-006: Runtime Parses Conversion Failure Context From Error Text

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:147-171`: Clean Code, Domain-Driven Design, and layered architecture require descriptive explicit concepts and cohesive functions rather than hidden string protocols across layers.
- `.specify/memory/constitution.md:149-162`: names and domain concepts must be explicit, and domain rules must stay separated from IO and infrastructure concerns so business logic remains testable and replaceable.

**Evidence**:

- `internal/integration/currency/errors.go:43-55`
- `internal/integration/currency/errors.go:102-129`
- `internal/report/calculate/errors.go:57-69`
- `internal/app/runtime/report_service.go:238-295`
- `internal/app/runtime/report_failure_context.go:83-144`

**Description**:

The integration layer defines a typed `ConversionFailure` with source currency, report base currency, activity date, provider, and reason fields. Report calculation wraps that typed failure for safe propagation. Runtime then extracts conversion context by parsing `calculationError.Error()` and by matching token fragments such as `reason=`, `provider=`, `source_currency=`, and fallback phrases like `from ... to ... on ...`. This creates a hidden string protocol between calculation and runtime. It makes runtime behavior depend on user-visible message formatting instead of an explicit report-owned failure context, which is a maintainability risk when failure copy changes.

### CODE-STAND-DRIFT-007: Conversion Audit Model File Combines Multiple Domain Entities And Display Helpers

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:147-171`: SRP and file-level cohesion require Go files to contain code related to a single responsibility, and long files with multiple types should be evaluated for type-focused decomposition.
- `.specify/memory/constitution.md:149-162`: modules and functions must remain cohesive, minimize duplication, and model domain concepts explicitly.

**Evidence**:

- `internal/report/model/conversion_audit.go:15-97`
- `internal/report/model/conversion_audit.go:99-128`
- `internal/report/model/conversion_audit.go:130-230`
- `internal/report/model/conversion_audit.go:336-375`
- `specs/007-currency-conversion-strategy/data-model.md:258-368`

**Description**:

`conversion_audit.go` currently owns separate report-domain concepts for converted amount kind, conversion status, rate authority, provider identity, quote direction, exchange-rate evidence, converted activity amounts, conversion audit entries, validation for those entities, and provider display labels. The feature data model names `ExchangeRateEvidence`, `ConvertedActivityAmount`, and `ConversionAuditEntry` as separate entities with separate relationships and validation rules. Keeping those entities and display-label helpers in one 478-line model file weakens file-level cohesion and makes unrelated model changes touch the same file.

## Notes

- The active feature directory was resolved with `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` as `/home/benizzio/src-workspace-opencode/ghostfolio-cryptogains/specs/007-currency-conversion-strategy`.
- `spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`, `quickstart.md`, and `contracts/` were reviewed for implementation-surface scope.
- No unchecked task syntax was found in `tasks.md`. Checked `Reopened` annotations were treated as historical bugfix markers rather than open task state.
- The previous report contained `CODE-STAND-DRIFT-001` through `CODE-STAND-DRIFT-004`. Those findings were not carried forward because the current tree no longer contains the same substantive drift: report model and Markdown packages no longer import `internal/integration/currency`, currency service responsibilities are split across cohesive files, Markdown rendering is split across section files, and `gocognit` reported no production functions over threshold.
- No additional proprietary agent-instruction files were discovered in scope beyond `AGENTS.md`. `.specify/memory/constitution.md` was loaded as the governing constitution baseline.
- `go run github.com/uudashr/gocognit/cmd/gocognit@latest -test=false -over 15 internal` returned no over-threshold production functions.
