# Coding Standards Drift Report: Generate Yearly Gains And Losses Report

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-05-24
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-analysis.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Standards Baseline

- `AGENTS.md:64-67`: `internal/app/runtime/` owns cross-package orchestration and that orchestration should not move into `cmd/` or `internal/tui/`.
- `AGENTS.md:74-85`: Ghostfolio boundary packages own upstream response knowledge, `internal/sync/model/` owns normalized activity records, and `internal/sync/validate/` owns currency-context and supported-history validation rules.
- `AGENTS.md:93-97`: shared support packages centralize exact-decimal parsing, canonical formatting, math implementations, and reusable calculation code.
- `AGENTS.md:99-103`: TUI packages own rendering, interaction state, screen routing, async command wiring, and workflow state transitions, but must not own HTTP, crypto, or normalization rules.
- `AGENTS.md:130-151`: code should use descriptive names, SOLID/SRP, decomposition, DRY, consistency, and Go production functions should keep cognitive complexity under 15 or be analyzed for decomposition.
- `AGENTS.md:157-179`: AI-generated code must include minimal language-standard documentation and authoring information at package, type, and function levels; public APIs require detailed usage instructions including an example.
- `.specify/memory/constitution.md:128-141`: code must follow Clean Code, Domain-Driven Design, and Clean Architecture; names must model domain concepts explicitly; modules and functions must be cohesive, minimize duplication, and separate domain rules from IO and infrastructure concerns.

## Findings

### DRIFT-001: Sync Model Owns Validation-Like Amount Resolution

**Severity**: High
**Diverges from**:

- `AGENTS.md:81-85`
- `.specify/memory/constitution.md:128-141`

**Evidence**:

- `internal/sync/model/activity_amount_resolution.go:43-225`

**Description**:

`internal/sync/model` is documented as the normalized data-model package, while validation and currency-context checks belong under the validation boundary. `ResolveActivityAmounts` and its helpers implement current-slice currency-tier selection, gross-value derivation, fee resolution, and validation-shaped error messages in the model package. This mixes normalized record definitions with evolving calculation and validation policy, increasing coupling for later reporting and sync-rule changes.

### DRIFT-002: Round-Half-Up Decimal Policy Is Duplicated Across Packages

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:93-97`
- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:134-135`

**Evidence**:

- `internal/report/decimal/policy.go:16-106`
- `internal/ghostfolio/validator/response_validator.go:279-358`
- `internal/sync/model/activity_amount_resolution.go:283-360`

**Description**:

The fixed-scale round-half-up quotient algorithm, finite decimal fraction conversion, and power-of-ten helper are implemented independently in report decimal policy, Ghostfolio validation, and sync amount resolution. This is general reusable decimal policy for the feature slice, not package-specific response or model state. The duplication creates separate maintenance points for financial arithmetic that should remain centralized.

### DRIFT-003: TUI Handlers Exceed The Cognitive Complexity Threshold

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:142-145`
- `AGENTS.md:146-151`
- `.specify/memory/constitution.md:134-141`

**Evidence**:

- `internal/tui/flow/report_flow.go:57-119`
- `internal/tui/flow/navigation.go:87-137`

**Description**:

`gocognit` reports `(*Model).handleReportSelectionKey` at 35 and `(*Model).updateSyncResult` at 23, above the repository's production-code threshold of 15. These functions combine navigation, focus movement, action dispatch, report start or retry routing, diagnostics branching, and context-aware state transitions. The combined branching weakens locality and makes future workflow changes more error-prone.

### DRIFT-004: Asset Replay Calculation Exceeds The Cognitive Complexity Threshold

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:142-145`
- `AGENTS.md:146-151`
- `.specify/memory/constitution.md:134-141`

**Evidence**:

- `internal/report/calculate/calculator.go:329-415`

**Description**:

`gocognit` reports `calculateAssetGroup` at 21. The function creates basis state, resolves scoped inputs, replays every input, mutates replay state, accumulates yearly result, captures opening and closing positions, classifies liquidation state, and shapes calculation errors. The function has multiple responsibilities inside the report domain and should be split around replay, accumulation, and final-result construction boundaries.

### DRIFT-005: Lot Disposal Calculation Exceeds The Cognitive Complexity Threshold

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:142-145`
- `AGENTS.md:146-151`
- `.specify/memory/constitution.md:134-141`

**Evidence**:

- `internal/report/basis/lot_methods.go:119-178`

**Description**:

`gocognit` reports `(*LotMethodState).Dispose` at 16, just above the threshold. The function validates input, orders lots, calculates proportional basis, mutates lot state, accumulates basis, appends matches, and checks exhaustion in one loop. This is a localized decomposition drift in a financial calculation path.

### DRIFT-006: XDG Documents Parser Exceeds The Cognitive Complexity Threshold

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:142-145`
- `AGENTS.md:146-151`
- `.specify/memory/constitution.md:134-141`

**Evidence**:

- `internal/report/output/documents.go:102-140`

**Description**:

`gocognit` reports `parseXDGDocumentsDirectory` at 19. The function scans lines, filters comments, parses the key, validates quote syntax, unescapes, expands `$HOME`, validates absolute paths, and formats errors in one block. Splitting value extraction from path resolution would better match the repository's decomposition standard.

### DRIFT-007: Runtime Classifies Output Failures By Message Text

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:128-141`

**Evidence**:

- `internal/app/runtime/report_service.go:380-396`

**Description**:

`reportWriteFailureReason` maps output failures into runtime failure reasons by matching lowercased error-message substrings such as `"linux xdg documents entry"` and `"documents path"`. Runtime orchestration is therefore coupled to output-package wording instead of a typed error or explicit failure category. That weakens domain clarity and makes future wording changes capable of changing runtime behavior.

### DRIFT-008: Report Output Package Exposes Test-Only Hooks In Production Code

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:134-137`

**Evidence**:

- `internal/report/output/seams.go:14-60`
- `internal/report/output/seams.go:63-106`

**Description**:

The production output package defines multiple variables described as test seams and exports `InstallWriteFailureAfterCreateForTesting`. This makes test-only mutation hooks part of production package code and surface area. The package responsibility is report filesystem output and opener behavior, while deterministic failure injection belongs in test-only code or a smaller injected dependency boundary.

### DRIFT-009: Menu Actions Depend On Raw Index Positions

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:131-139`

**Evidence**:

- `internal/tui/flow/model.go:575-595`
- `internal/tui/flow/report_flow.go:182-219`
- `internal/tui/flow/navigation.go:155-183`
- `internal/tui/flow/model.go:925-934`

**Description**:

The TUI builds report and sync-report menus as ordered slices, then later interprets actions through raw positions such as `0`, `1`, `2`, and `3`. Action semantics are split between menu construction and handlers instead of being modeled with named action identifiers. This creates implicit coupling between menu order and behavior, which is inconsistent with the repository's descriptive naming and explicit domain-concept guidance.

### DRIFT-010: AI-Generated Documentation Is Incomplete Or Stale

**Severity**: Low
**Diverges from**:

- `AGENTS.md:157-179`
- `.specify/memory/constitution.md:131-139`

**Evidence**:

- `internal/ghostfolio/mapper/activity_mapper.go:17-28`
- `internal/app/runtime/report_types.go:112-120`
- `internal/tui/flow/report_flow.go:1-3`

**Description**:

`activityMoneyContext` is an AI-authored struct without type-level documentation or authoring information. The public `ReportService.Generate` API has a short method comment but no detailed usage instructions or example. `report_flow.go` also carries a stale package comment saying the package owns the `sync-and-storage slice` while the file now implements report-generation workflow routing. These are documentation and attribution drift items under the repository's AI-generated-code rules.

### DRIFT-011: UI Copy Centralization Is Partial And Inconsistent

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:99-103`
- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:134-139`

**Evidence**:

- `internal/tui/component/sync_entry_copy.go:5-46`
- `internal/tui/screen/sync_entry_screen.go:57-63`
- `internal/tui/flow/model.go:317-338`
- `internal/tui/screen/sync_reports_screen.go:47-118`
- `internal/tui/screen/report_screen.go:68-154`
- `internal/tui/screen/sync_result_screen.go:80-132`
- `internal/tui/screen/main_menu_screen.go:43-58`

**Description**:

The feature introduced `SyncEntryCopy` as a reusable copy structure and `SyncEntryScreenView` uses it for `Sync Data` defaults. The same feature slice still hardcodes unlock-screen copy in `model.go` and embeds substantial user-visible text directly in the Sync and Reports, report selection/result, sync result, and main-menu renderers. This leaves two competing patterns for UI copy ownership inside the TUI layer and weakens consistency and DRY behavior for user-facing wording changes.

### DRIFT-012: Generic Decimal Math Helpers Bypass The Shared Support Boundary

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:93-97`
- `AGENTS.md:130-151`
- `.specify/memory/constitution.md:134-137`

**Evidence**:

- `internal/report/basis/lot_methods.go:362-448`
- `internal/report/calculate/decimal_math.go:10-71`
- `internal/report/calculate/calculator.go:20-38`
- `internal/report/calculate/calculator.go:1006-1039`
- `internal/report/decimal/policy.go:16-106`

**Description**:

The repository baseline assigns reusable math implementations and calculation code to `internal/support/math`, but the current feature keeps generic decimal operations inside report-domain packages. `lot_methods.go` defines reusable add, subtract, multiply, min, zero, clone, and proportional arithmetic helpers; `calculate/decimal_math.go` defines another multiplication, comparison, zero-check, and finite-validation layer; `calculator.go` adds another add/subtract wrapper and test seams; and `report/decimal/policy.go` owns fixed-scale quotient mechanics. These are not all report-domain concepts. Keeping generic arithmetic spread across report subpackages increases duplication and makes package boundaries less clear.

## Notes

- No existing `coding-standards-drift-report.md` was present, so all drift IDs are new.
- `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` resolved this feature directory, and `tasks.md` contained no open `- [ ]` tasks.
- No additional proprietary instruction files were present for this scope: `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, and `.clinerules` were not found.
- Cognitive-complexity findings use `go run github.com/uudashr/gocognit/cmd/gocognit@latest -over 15 ...`. Test functions reported by that command were excluded because `AGENTS.md:145` exempts test code from the cognitive-complexity rule.
