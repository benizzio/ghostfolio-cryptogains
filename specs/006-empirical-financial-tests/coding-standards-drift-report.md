# Coding Standards Drift Report: Empirical Solidified Financial Tests

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: 2026-06-14
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.coding-standards-drift-control.remediation-plan`.

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.
- The reviewed implementation slice is `internal/support/math/`, `tests/empirical/`, `tools/empiricaloracle/`, `testdata/empirical/`, `third_party/rotki/`, `Makefile`, and the active feature artifacts under `specs/006-empirical-financial-tests/`.

## Standards Baseline

- `AGENTS.md:94-99`: shared support packages centralize exact-decimal formatting, math implementations, and reusable support behavior.
- `AGENTS.md:106-124`: tests and tools have defined repository locations, and empirical validation suites stay under the test boundary.
- `AGENTS.md:134-158`: code must follow Clean Code, DDD, Clean Architecture, SOLID, SRP, DRY, descriptive naming, cohesion, and separation between domain, infrastructure, and reusable layers.
- `AGENTS.md:164-186`: AI-generated code must include package, type, and function or method documentation with authoring information; public API code must include detailed usage instructions.
- `.specify/memory/constitution.md:65-107`: financial precision rules must remain explicit, deterministic, auditable, and reproducible.
- `.specify/memory/constitution.md:109-129`: project-owned code must be automatically tested and coverage instrumentation must support the project verification model.
- `.specify/memory/constitution.md:133-147`: dependency and external integration decisions must document supported versions, authentication model, failure modes, and security implications.
- `.specify/memory/constitution.md:149-160`: code must follow Clean Architecture and domain clarity, use descriptive names, minimize duplication, and separate domain rules from IO and infrastructure concerns.

## Findings

### CODE-STAND-DRIFT-001: Shared Math Helper Reads Process Environment

**Severity**: High
**Diverges from**:

- `AGENTS.md:94-99`
- `AGENTS.md:150-158`
- `.specify/memory/constitution.md:149-160`

**Evidence**:

- `internal/support/math/decimal_policy.go:3-8`
- `internal/support/math/decimal_policy.go:40-51`
- `internal/support/math/decimal_ops.go:138-180`

**Description**:

`DivideFiniteRoundHalfUp` is a reusable arithmetic helper in `internal/support/math`, but it now selects its rounding policy by calling `selectedDecimalPolicy`, which reads `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` through `os.LookupEnv`. This mixes process configuration and IO-style global state into the shared math layer. Financial calculation behavior becomes implicit process state instead of an explicit input at the boundary that chooses a policy.

### CODE-STAND-DRIFT-002: Empirical Fixture Duplicates Runtime Scope-Reliability Rules

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:134-141`
- `AGENTS.md:150-158`
- `.specify/memory/constitution.md:149-160`

**Evidence**:

- `tests/empirical/fixture/project_translation.go:210-296`
- `internal/sync/normalize/activity_history.go:215-297`

**Description**:

The empirical fixture package defines `deriveProjectScopeReliability`, `deriveProjectTimelineScopeReliability`, and `usableProjectSourceScope` with the same structure as runtime normalization helpers. The local comment states that the code mirrors runtime normalization. This duplicates a domain rule across the runtime sync normalizer and the empirical fixture translator, increasing the risk that one path changes while the other silently drifts.

### CODE-STAND-DRIFT-003: Oracle Command `run` Mixes Multiple Responsibilities

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:134-158`
- `.specify/memory/constitution.md:149-160`

**Evidence**:

- `tools/empiricaloracle/main.go:38-185`

**Description**:

`run` handles CLI flag setup and parsing, repository path resolution, dataset loading, dataset validation, golden-fixture discovery, rotki source runtime setup, oracle generation routing, output marshaling, artifact writing, and user-facing reporting. The function is doing command parsing, application orchestration, IO, and generation control in one block, which weakens SRP and makes the oracle command harder to change safely.

### CODE-STAND-DRIFT-004: Rotki Adapter Boundary Uses Generic Payloads And Mixed Logic

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:134-158`
- `.specify/memory/constitution.md:149-160`

**Evidence**:

- `tools/empiricaloracle/rotki_adapter.py:410-527`

**Description**:

`execute_rotki_boundary` interprets untyped payload dictionaries, mutates the rotki cost-basis manager, calculates aggregate values, builds match evidence, tracks closing state, and renders the response dictionary. The generic `dict[str, object]` shape hides the domain concepts that are modeled explicitly in the Go fixture layer, and the function combines adapter input parsing, oracle execution, normalization, and response construction.

### CODE-STAND-DRIFT-005: Coverage Target Omits The Empirical Fixture Subpackage

**Severity**: Medium
**Diverges from**:

- `AGENTS.md:106-124`
- `.specify/memory/constitution.md:109-129`

**Evidence**:

- `Makefile:26-30`
- `tests/empirical/fixture/model_test.go:1-16`

**Description**:

`make coverage` runs `go test` for `./tests/empirical` but not `./tests/empirical/...`. The empirical helper code and its package-local tests live in the `tests/empirical/fixture` subpackage, so the coverage command does not execute that package during coverage generation. This creates an inconsistent verification boundary for project-owned empirical helper code.

### CODE-STAND-DRIFT-006: External Source Boundary Documentation Omits Required Integration Details

**Severity**: Medium
**Diverges from**:

- `.specify/memory/constitution.md:133-147`
- `.specify/memory/constitution.md:175-176`

**Evidence**:

- `tools/empiricaloracle/rotki_source.go:291-324`
- `tools/empiricaloracle/rotki_source.go:380-406`
- `third_party/rotki/README.md:26-57`

**Description**:

The rotki source boundary downloads a GitHub archive and verifies the remote release tag through `git ls-remote`. The adjacent provenance documentation records pinning, checksums, platform scope, and cache policy, but does not document the authentication model, expected external failure modes, and security implications for these external calls. The constitution requires those details before external API integrations are implemented.

### CODE-STAND-DRIFT-007: AI-Authored Feature Packages Lack Package-Level Documentation

**Severity**: Low
**Diverges from**:

- `AGENTS.md:164-186`

**Evidence**:

- `tools/empiricaloracle/main.go:1`
- `tests/empirical/dataset_validation_test.go:1`
- `tests/empirical/fixture/model.go:1`

**Description**:

The feature's AI-authored Go packages start directly with package declarations and have no package-level documentation or package-level authoring block. Function and type comments exist in many files, but the baseline also requires component or module documentation with authoring information for AI-generated code.

### CODE-STAND-DRIFT-008: Dataset Validation Test Documentation Is Stale

**Severity**: Low
**Diverges from**:

- `AGENTS.md:20-21`
- `AGENTS.md:164-186`

**Evidence**:

- `tests/empirical/dataset_validation_test.go:23-39`
- `tests/empirical/dataset_validation_test.go:64-70`

**Description**:

The comments still describe parser and validator hooks as future or missing implementation, but the hook variable is wired to `fixture.LoadEmpiricalDataset` and `fixture.ValidateEmpiricalDataset`. This is stale AI-authored documentation and conflicts with the repository rule to keep claims empirically grounded.

### CODE-STAND-DRIFT-009: Active Oracle Code Retains Journal Terminology

**Severity**: Low
**Diverges from**:

- `AGENTS.md:134-141`
- `.specify/memory/constitution.md:149-160`

**Evidence**:

- `tools/empiricaloracle/oracle_helpers.go:31-44`
- `tools/empiricaloracle/oracle_helpers.go:79-82`
- `tools/empiricaloracle/unsupported.go:168-173`
- `specs/006-empirical-financial-tests/contracts/dataset-format.md:108-120`

**Description**:

The active rotki/composite oracle code still uses `journalLotMode` and `compareJournalActivities`, and unsupported-reason text refers to `lot mode` through that journal helper. The active dataset contract also names copied upstream `hledger` rows in the synthetic-only rule without strikethrough or historical context. These names preserve superseded hledger/journal terminology in active implementation and contract text, weakening domain clarity after the feature moved to rotki-backed and composite-oracle boundaries.

### CODE-STAND-DRIFT-010: Exported Fixture Helper Lacks Public Usage Documentation

**Severity**: Low
**Diverges from**:

- `AGENTS.md:164-186`

**Evidence**:

- `tests/empirical/fixture/project_output.go:37-48`

**Description**:

`NormalizeProjectCalculationOutputForCase` is exported from `tests/empirical/fixture`, but its comment only gives a short purpose statement and no usage example. The baseline requires detailed usage instructions for public API code usable from other packages.

### CODE-STAND-DRIFT-011: Dead Helper Code Remains In Empirical Test Slice

**Severity**: Low
**Diverges from**:

- `AGENTS.md:134-141`
- `.specify/memory/constitution.md:155-156`

**Evidence**:

- `tests/empirical/fixture/dataset_coverage_test.go:201-211`
- `tools/empiricaloracle/rotki_adapter.py:546-556`

**Description**:

`allRequiredCoverageTags` and `calculate_open_state` are defined but are not referenced by the active feature slice. The unused helpers add maintenance surface and duplicate concepts that are already represented elsewhere in the empirical coverage and adapter code.

## Notes

- No existing `coding-standards-drift-report.md` was present, so IDs were assigned starting at `CODE-STAND-DRIFT-001`.
- `.specify/templates/tasks-template.md` and `tasks.md` were loaded to interpret task state. No unchecked tasks were found before writing this report.
- `AGENTS.md` was the only proprietary agent-instruction file discovered in the repository root during this run.
