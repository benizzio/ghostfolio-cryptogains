---
description: "Generate or refresh test-coverage-drift-report.md for the active feature"
---

<!-- Extension: test-coverage-drift-control -->
<!-- Config: .specify/extensions/test-coverage-drift-control/ -->
# Generate Test Coverage Drift Report

Review the active feature implementation for divergences from the repository test-coverage baseline and write a structured drift report.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may narrow the review scope, request a full rerun, or ask to focus on specific files or existing coverage drift IDs.

## Purpose

- Focus on test coverage, coverage gates, and test-structure alignment only.
- Do not review general coding standards, domain correctness, product behavior, or feature completeness unless that context is required to explain a test-coverage drift.
- This command is rerunnable. Overwrite the report with a fresh snapshot, but preserve existing `COV-DRIFT-###` identifiers for substantively unchanged findings when possible.
- Run only after the active feature's normal Spec Kit implementation tasks are complete.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `spec.md`, `plan.md`, and `tasks.md` exist in `FEATURE_DIR`.
4. Load the current Spec Kit task format references before interpreting task state:
   - `.specify/templates/tasks-template.md` when present
   - the existing `FEATURE_DIR/tasks.md`
5. Using the task state syntax from the current local Spec Kit installation and the existing task file, verify there are no open, unchecked, pending, or reopened tasks in `FEATURE_DIR/tasks.md`. If any are present, stop without writing the report and instruct the user to finish implementation with `/speckit.implement` before running coverage drift analysis.
6. Load the coverage definition reference files that define the repository's coverage target and expected test structure:
   - `.specify/memory/constitution.md` when present
   - `AGENTS.md` when present
   - any known proprietary agent-instruction files present in repository or feature scope, such as `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, or `.clinerules`
7. If the loaded coverage definition reference files do not define concrete coverage targets, coverage gates, or test-structure expectations, derive the best-guess baseline conservatively from existing repository coverage commands, CI workflows, tests, and feature planning artifacts. Mark derived baseline items as derived in the report.

## Review Scope

- Treat the active feature spec and its implementation as the default and preferred scope.
- Derive candidate files from `spec.md`, `plan.md`, `tasks.md`, current feature documentation, coverage reports or commands referenced by the feature, and source or test files touched by or directly supporting that feature.
- Inspect repository coverage commands and gate definitions such as `Makefile`, CI workflow files, coverage helper tools, and maintained test commands when they are required to determine whether coverage is being measured against the same baseline.
- Expand beyond the active feature only when a directly related supporting file is required to explain coverage gate instrumentation, test type balance, or missing coverage evidence.
- Avoid repository-wide exploration that would bloat the report beyond the active feature slice.
- Prefer exact file and line evidence over broad repository-wide claims.

## Coverage Baseline

Evaluate the implementation against the repository coverage-policy baseline from the coverage definition reference files:

1. `.specify/memory/constitution.md` when present
2. `AGENTS.md` when present
3. Any other known proprietary agent-instruction files present in repository or feature scope
4. Existing repository coverage commands and CI workflows only when the policy baseline references or relies on them

Prioritize the loaded baseline rules around:

- required coverage percentage targets, including line, statement, branch, or file coverage when the tooling distinguishes them
- coverage instrumentation of project-owned packages from contract, integration, and other black-box test packages
- required coverage commands and CI quality gates
- required test structure, including integration tests as the default for user journeys and external-service-facing workflows
- when unit tests are expected, when they are discouraged, and when duplicated unit tests should be removed
- evidence that coverage-relevant regressions and test suites pass before completion

## Outline

1. Load the existing `test-coverage-drift-report.md` if it exists. Use it only to preserve stable `COV-DRIFT-###` identifiers for equivalent findings and to keep the correction-tracking link consistent.
2. Read the feature artifacts:
   - `spec.md`
   - `plan.md`
   - `tasks.md`
   - `research.md`, `data-model.md`, `quickstart.md`, and `contracts/` when they help define the implementation and test surface
3. Inspect relevant implementation files, test files, coverage commands, and coverage gate definitions.
4. Identify only concrete drift items that have explicit coverage-policy evidence from the loaded coverage definition reference files, or clearly marked derived evidence when the explicit baseline is incomplete.
5. Assign severity:
   - `High`: the feature cannot demonstrate the required coverage target, bypasses coverage instrumentation for project-owned packages, or lacks required tests for critical user journeys or external-service-facing workflows
   - `Medium`: test structure drifts from the required integration-first approach, coverage evidence is incomplete, or unit tests duplicate integration coverage in a maintainability-relevant way
   - `Low`: local coverage documentation, command alignment, or minor test organization drift with limited release risk
6. Reuse existing `COV-DRIFT-###` IDs when the finding is substantively the same. Assign new IDs sequentially after the highest existing coverage drift number only for new findings.
7. Write `FEATURE_DIR/test-coverage-drift-report.md`, overwriting the previous file.

## Output Format

Write a Markdown report with this structure:

```markdown
# Test Coverage Drift Report: [Feature Name]

**Purpose**: Record concrete deviations between the current implementation and the repository test-coverage baseline for the active feature slice.
**Created**: [YYYY-MM-DD]
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: Coverage drift remediation tasks are added to [tasks.md](./tasks.md) by `/speckit.test-coverage-drift-control.remediation-plan`.

## Scope

- This report covers test coverage, coverage gates, and test-structure alignment only.
- This report does not cover general coding standards, domain correctness, product behavior, or unrelated constitution-gate evidence.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Coverage Baseline

[List the coverage definition reference files and the specific target, gate, or test-structure clauses used in the review. Mark any derived baseline item as derived.]

## Findings

### COV-DRIFT-001: [Short Title]

**Severity**: [High | Medium | Low]
**Diverges from**:

- [Specific coverage policy reference]
- [Specific coverage policy reference]

**Evidence**:

- `path/to/file.ext:10-30`
- `path/to/file.ext:44-52`

**Description**:

[Concrete explanation of the drift and why it matters in this repository.]

## Notes

- [Any zero-findings note, scope limitation, or validation note]
```

## Rules

- The report MUST be implementation-specific, not a generic testing lecture.
- Every finding MUST cite exact file references.
- Every finding MUST map to an explicit coverage-policy source unless the baseline is explicitly marked as derived because the reference files are incomplete.
- The `## Coverage Baseline` section MUST list every loaded coverage definition reference file used as review evidence.
- Do not include remediation task checkboxes in this report. Remediation tasks belong in `tasks.md`.
- If no drift is found, still write the report with `## Findings` stating that no test-coverage drift was identified in the reviewed scope.
- Keep wording grounded and empirical.
