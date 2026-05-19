# Test Coverage Drift Report: Store Activity Data

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: 2026-05-18
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-analysis.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Coverage Baseline

- `.specify/memory/constitution.md:91-108`
  Project-owned code must maintain 100% automated coverage, and when tooling distinguishes line and branch coverage, both must remain at 100%. Integration tests are the default, and coverage commands or CI must count execution driven from black-box contract and integration suites.
- `.specify/memory/constitution.md:176-183`
  Task lists must include coverage verification, PR workflows must run the repository test workflow automatically, and missing measurement or gating is itself a prerequisite gap.
- `AGENTS.md:1-4`
  Repo-local agent guidance redirects feature-specific implementation and verification context to `specs/003-store-activity-data/plan.md`. `AGENTS.md` does not define a standalone numeric coverage target beyond that redirect.
- `specs/003-store-activity-data/plan.md:21,37`
  The maintained verification commands for this slice are `make test` and `make coverage`; testing is integration-first; statement and branch/file coverage remain explicit release gates through `make coverage`.
- `specs/003-store-activity-data/tasks.md:11,161-163`
  The feature task baseline requires 100% statement coverage from `go test`, 100% branch and file coverage through the `gocoverageplus` gate, and a final rerun that certifies those gates are met.
- `specs/003-store-activity-data/quickstart.md:25-33,35-50`
  The contributor verification path for this feature includes `make coverage` and expects the listed feature coverage scope to be covered by the maintained suites.
- `specs/003-store-activity-data/research.md:97-121`
  The feature keeps an integration-first test strategy and explicitly rejects statement-only coverage as insufficient because the repository tooling distinguishes richer coverage signals.
- No additional proprietary agent-instruction files were present under `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, or `.clinerules`.

## Findings

### COV-DRIFT-001: Coverage Gate Is Not Enforced In The Maintained Verification Path

**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:91-108`
- `.specify/memory/constitution.md:176-183`
- `specs/003-store-activity-data/plan.md:21,37`
- `specs/003-store-activity-data/tasks.md:11,161-163`

**Evidence**:

- `Makefile:17-20`
- `.github/workflows/test.yml:41-42`
- `dist/coverage/coverage.xml:1`

**Description**:

The repository documents `make coverage` as the release gate for this slice, but the maintained verification path does not enforce the required threshold. `make coverage` completed successfully while the generated report still recorded `line-rate="0.95"` and `branch-rate="0.89"` in `dist/coverage/coverage.xml`. The PR workflow also runs only `make test`, so pull requests do not execute the documented coverage gate at all. This leaves the active feature without an enforced mechanism that blocks merges when the repository's 100% coverage requirement is missed.

### COV-DRIFT-002: Active Feature Code Still Measures Below The Required 100% Target

**Severity**: High
**Diverges from**:

- `.specify/memory/constitution.md:91-108`
- `specs/003-store-activity-data/plan.md:21,37`
- `specs/003-store-activity-data/tasks.md:11,161-163`
- `specs/003-store-activity-data/research.md:97-121`

**Evidence**:

- `dist/coverage/coverage.xml:1`
- `internal/ghostfolio/mapper/activity_mapper.go:294-409`
- `dist/coverage/coverage.xml:2164-2410`
- `internal/sync/model/activity_amount_resolution.go:81-117`
- `internal/sync/model/activity_amount_resolution.go:124-162`
- `internal/sync/model/activity_amount_resolution.go:234-240`
- `dist/coverage/coverage.xml:248-373`
- `internal/app/runtime/snapshot_lifecycle.go:52-61`
- `internal/app/runtime/snapshot_lifecycle.go:66-91`
- `internal/app/runtime/snapshot_lifecycle.go:134-159`
- `dist/coverage/coverage.xml:4522-4602`
- `internal/app/runtime/active_snapshot_state.go:38-65`
- `dist/coverage/coverage.xml:4066-4099`

**Description**:

The current feature verification output does not meet the repository baseline. The generated aggregate report is still below policy at 95% line coverage and 89% branch coverage. The misses are not confined to unrelated legacy areas; they remain in feature-owned code added for this slice. Examples include unexecuted mapper diagnostic derivation paths in `activity_mapper.go`, partially covered currency-resolution branches in `activity_amount_resolution.go`, and untested nil-guard or error branches in the protected-snapshot runtime lifecycle helpers. This is a direct drift from the feature's own requirement to keep 100% statement coverage plus 100% branch and file coverage for project-owned code.

## Notes

- No additional integration-first structure drift was identified beyond the uncovered feature branches and the missing gate enforcement.
- The maintained coverage command does correctly instrument black-box contract and integration test packages through `-coverpkg=$(PRODUCTION_PACKAGES)` in `Makefile:19`; the drift is threshold attainment and enforcement, not instrumentation scope.
