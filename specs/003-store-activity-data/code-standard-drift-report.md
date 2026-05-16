# Code Standard Drift Report: Store Activity Data

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-05-16
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: [checklists/code-standard-drift-remediation.md](./checklists/code-standard-drift-remediation.md)

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- Reviewed feature artifacts: `spec.md`, `plan.md`, `tasks.md`, `research.md`, `data-model.md`, and `quickstart.md`.
- Reviewed implementation slice: `internal/app/runtime`, `internal/ghostfolio/{client,dto,mapper,validator}`, `internal/snapshot/{envelope,model,store}`, `internal/sync/{model,normalize,validate}`, and the directly related `internal/tui/{flow,screen}` sync workflow files.

## Standards Baseline

- `AGENTS.md`: `Coding standards > LiteratureAndIndustryReferences` requires descriptive and unambiguous names, SRP, decomposition into smaller functions tied to a single responsibility, DRY, and consistency.
- `AGENTS.md`: `Coding standards > CustomCodeDocs` requires AI-touched code documentation to describe the real purpose of modules and functions accurately, with authoring information.
- `.specify/memory/constitution.md`: `V. Clean Architecture and Domain Clarity` requires descriptive and unambiguous names, cohesive modules, minimal duplication, SOLID boundaries, and separation of domain rules from IO and infrastructure concerns.
- No additional proprietary agent-instruction files were present in the repository or feature scope at review time.

## Findings

### DRIFT-001: Runtime Sync Service Mixes Multiple Architectural Responsibilities

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: descriptive and unambiguous names, SRP, decomposition into smaller functions tied to a single responsibility
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and separate domain rules from IO and infrastructure concerns

**Evidence**:

- `internal/app/runtime/sync_service.go:171-329`
- `internal/app/runtime/sync_service.go:387-427`
- `internal/app/runtime/sync_service.go:503-579`
- `internal/app/runtime/diagnostic_report.go:62-148`

**Description**:

The runtime sync path centralizes server-scoped snapshot discovery and unlock, Ghostfolio authentication, full-history retrieval, mapping and normalization orchestration, validation routing, protected payload construction, snapshot persistence coordination, diagnostic-report policy, and active readable-snapshot lifecycle handling in the same runtime component. This crosses transport, domain-workflow, storage-model, and local-artifact concerns in one place instead of keeping the slice's boundaries cohesive. The result is a broad change surface where storage, diagnostics, and workflow changes all require edits in the same service layer.

### DRIFT-002: Deterministic Ordering Rules Are Reimplemented In Multiple Layers

**Severity**: High
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: avoid code duplication, be consistent
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: modules and functions must remain cohesive, minimize duplication, and keep domain concepts explicit

**Evidence**:

- `internal/sync/normalize/activity_history.go:104-120`
- `internal/sync/normalize/activity_history.go:133-191`
- `internal/sync/validate/activity_history.go:91-117`
- `internal/sync/validate/activity_history.go:169-216`

**Description**:

The same source-date, asset, activity-type, source-id, and ambiguity-handling rules are implemented twice: once in normalization and again in validation, each with its own ordering key type and comparator helpers. That duplicates a core domain invariant instead of keeping one local source of truth. Any change to the ordering contract now requires coordinated edits in both layers, which increases drift risk and weakens locality of behaviour in a part of the feature that already carries strict deterministic rules.

### DRIFT-003: Stale Validation And Probe Terminology Obscures The Storage Workflow

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` `Coding standards > LiteratureAndIndustryReferences`: choose descriptive and unambiguous names, be consistent
- `AGENTS.md` `Coding standards > CustomCodeDocs`: AI-touched code documentation must describe the purpose of modules and functions accurately
- `.specify/memory/constitution.md` `V. Clean Architecture and Domain Clarity`: names must be descriptive and unambiguous, and consistency is mandatory

**Evidence**:

- `internal/app/runtime/sync_service.go:38-40`
- `internal/app/runtime/sync_service.go:63-75`
- `internal/app/runtime/sync_types.go:16-20`
- `internal/app/runtime/sync_types.go:147-163`
- `internal/ghostfolio/client/client.go:1-3`
- `internal/ghostfolio/client/client.go:90-93`
- `internal/ghostfolio/client/client.go:206-224`
- `internal/tui/flow/sync_flow.go:1-3`
- `internal/tui/flow/sync_flow.go:20-23`
- `internal/ghostfolio/dto/activities_probe_response.go:6-14`

**Description**:

The implementation now performs authenticated full-history retrieval and protected storage, but core runtime models, method comments, DTO aliases, and TUI flow comments still describe the slice as "validation-only" and "probe" based. This leaves the active code with names and documentation that no longer match its real responsibility. The drift is widespread enough across runtime, client, DTO, and TUI layers that intent is harder to read and future storage-slice changes will continue to carry legacy terminology.

## Notes

- No prior `code-standard-drift-report.md` existed for this feature, so this snapshot starts at `DRIFT-001`.
- The correction-tracking link is included for consistency with the expected workflow. The remediation checklist file is not present in the feature directory at review time.
