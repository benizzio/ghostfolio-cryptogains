# Code Standard Drift Report: Sync Data Validation

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the `002-sync-data-validation` slice.
**Created**: 2026-05-10
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: [checklists/code-standard-drift-remediation.md](./checklists/code-standard-drift-remediation.md)

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current Go source tree.

## Standards Baseline

The findings below diverge from standards defined in these files:

1. `AGENTS.md`
2. `.specify/memory/constitution.md`

The specific standards used for this review are:

- `AGENTS.md` -> `Coding standards` -> `LiteratureAndIndustryReferences`
  - descriptive and unambiguous names
  - SOLID boundaries with special emphasis on SRP
  - decomposition into smaller functions tied to a single responsibility
  - DRY
  - consistency
- `AGENTS.md` -> `Coding standards` -> `CustomCodeDocs`
  - AI-generated or AI-touched code must contain proper minimal language-standard documentation
  - AI-touched code must contain authoring information
  - public API code must contain very detailed usage instructions
- `AGENTS.md` -> `Coding standards`
  - prefer `var` over `:=`, except for multiple declarations with reuse
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity`
  - code must follow Clean Code, DDD, and Clean Architecture
  - modules and functions must remain cohesive and minimize duplication
  - domain rules must be separated from IO and infrastructure concerns
- `.specify/memory/constitution.md` -> `Governance`
  - `AGENTS.md` must align with the constitution and is part of the project engineering policy baseline

## Findings

### DRIFT-001: The TUI layer owns setup persistence and validation orchestration

**Severity**: High
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> Clean Code, DDD, Clean Architecture, SRP, and consistency rules
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity`

**Evidence**:

- `internal/tui/flow/setup_flow.go:187-205`
- `internal/tui/flow/model.go:93-104`
- `internal/tui/flow/model.go:429-446`

**Description**:

The Bubble Tea flow layer currently validates setup input, constructs `configmodel.AppSetupConfig`, writes setup through `ConfigStore`, and starts sync validation through `SyncService`. This mixes presentation responsibilities with application orchestration and infrastructure interaction.

That drift weakens the intended separation between UI state and use-case execution. It also increases the amount of business and IO behavior that must be understood and tested through the TUI model itself.

### DRIFT-002: The runtime sync service leaks transport concerns and presentation copy

**Severity**: High
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> Clean Architecture, SRP, and explicit domain modeling rules
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity`

**Evidence**:

- `internal/app/runtime/sync_service.go:75`
- `internal/app/runtime/sync_service.go:89-95`
- `internal/app/runtime/sync_service.go:178-226`

**Description**:

`SyncValidationAttempt` and `ValidationOutcome` expose `ghostfolioclient.FailureCategory`, and the runtime service builds final user-facing strings such as `SummaryMessage` and `FollowUpNote`. This makes the application layer depend on a transport-layer type and on presentation wording at the same time.

The result is a service contract that is less cohesive and harder to evolve. A Clean Architecture boundary would normally keep the application layer centered on structured outcome semantics, while the UI layer would own the final wording.

### DRIFT-003: Bootstrap startup state carries user-facing messaging

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> Clean Architecture and SRP guidance
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity`

**Evidence**:

- `internal/app/bootstrap/startup.go:15`
- `internal/app/bootstrap/startup.go:29-33`
- `internal/app/bootstrap/startup.go:56-57`

**Description**:

`StartupState` includes `InvalidSetupMessage`, and bootstrap logic fills it with a final UI sentence. This embeds presentation wording in bootstrap logic instead of returning structured state that the TUI can map to visible text.

This is the same architectural drift as the runtime service, but at startup. It keeps user-facing copy coupled to a non-UI layer.

### DRIFT-004: The Ghostfolio HTTP client duplicates the request pipeline

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> DRY and consistency rules
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity` requirement to minimize duplication

**Evidence**:

- `internal/ghostfolio/client/client.go:132-167`
- `internal/ghostfolio/client/client.go:181-217`

**Description**:

`Authenticate` and `FetchActivitiesProbe` repeat the same request construction, execution, status validation, content-type validation, and JSON decoding structure. The response-specific branches differ, but the surrounding boundary workflow is nearly identical.

This duplication increases the maintenance surface for the HTTP boundary and makes future changes to request handling more error-prone.

### DRIFT-005: `JSONStore.Save` has multiple responsibilities in one method

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> SRP and decomposition rules
- `.specify/memory/constitution.md` -> `V. Clean Architecture and Domain Clarity` requirement for cohesive functions

**Evidence**:

- `internal/config/store/json_store.go:127-166`

**Description**:

`Save` currently prepares the parent directory, encodes JSON, creates and manages a temporary file, applies permissions, writes and syncs content, performs the atomic rename, reapplies file permissions, and coordinates cleanup.

That amount of behavior in one method makes the persistence path harder to reason about and harder to change safely. The current implementation works, but it has visible SRP drift against the repository standard.

### DRIFT-006: Public API documentation is thinner than the repo requires

**Severity**: Medium
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> `CustomCodeDocs`

**Evidence**:

- `internal/config/store/store.go:15-27`
- `internal/app/runtime/sync_service.go:98-136`
- `internal/app/bootstrap/options.go:54-66`
- `internal/tui/component/help.go:26-34`

**Description**:

The repository standard requires detailed usage instructions for public API code, and public methods/functions are expected to include example usage in their documentation blocks. Current public API comments are present in several places, but they remain minimal and omit important contract detail.

Examples:

- `Store` does not document key contract behavior such as `ErrNotFound` handling and persistence expectations.
- `SyncService`, `NewSyncService`, and `Validate` do not document their boundary expectations in detail.
- `ParseOptions` does not describe supported flags, defaults, or error behavior in a detailed usage-oriented way.
- `ShortHelp` and `FullHelp` are public methods without example usage blocks.

### DRIFT-007: AI-touched code has incomplete documentation and author attribution

**Severity**: Low
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> `CustomCodeDocs`

**Evidence**:

- `cmd/ghostfolio-cryptogains/main.go:19`
- `cmd/ghostfolio-cryptogains/main.go:31-41`
- `internal/tui/flow/model.go:25`
- `internal/tui/flow/model.go:34`
- `internal/tui/flow/model.go:39`
- `internal/tui/flow/model.go:60`
- `internal/tui/flow/model.go:69`
- `internal/tui/flow/model.go:80`
- `internal/app/runtime/sync_service.go:111`
- `internal/config/store/json_store.go:24`
- `internal/config/store/json_store.go:188-207`

**Description**:

The repo standard requires AI-generated or AI-touched code to include minimal language-standard documentation and authoring information at the component, type, and function levels. The current codebase includes many `Authored by: OpenCode` comments, but that coverage is inconsistent.

Some AI-touched types have no comment block at all, and several functions have descriptive comments without the required author-attribution line.

### DRIFT-008: The preferred `var` declaration style is not followed consistently

**Severity**: Low
**Diverges from**:

- `AGENTS.md` -> `Coding standards` -> variable declaration preference for `var` over `:=`

**Evidence**:

- `cmd/ghostfolio-cryptogains/main.go:34`
- `internal/app/bootstrap/options.go:77`
- `internal/app/bootstrap/startup.go:56`
- `internal/app/runtime/sync_service.go:156`
- `internal/app/runtime/sync_service.go:170`
- `internal/ghostfolio/client/client.go:157`
- `internal/ghostfolio/client/client.go:207`

**Description**:

The repo explicitly prefers `var` over `:=` except where multiple declarations reuse an existing variable. The current source follows that convention in many places, but these single-name short declarations still diverge from the local style baseline.

## Notes

- No `float32` or `float64` usage was found in the current Go code.
- No `TODO`, `FIXME`, or `XXX` markers were found in the current Go code.
- This report is intended to support remediation planning and tracking, not to replace implementation tasks or code review.
