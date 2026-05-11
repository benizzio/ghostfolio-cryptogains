---
description: "Generate or refresh code-standard-drift-report.md for the active feature"
---

<!-- Extension: code-standard-drift-analysis -->
<!-- Config: .specify/extensions/code-standard-drift-analysis/ -->
# Generate Code Standard Drift Report

Review the active feature implementation for divergences from the repository coding-standards baseline and write a structured drift report.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may narrow the review scope, request a full rerun, or ask to focus on specific files or existing drift IDs.

## Purpose

- Focus on coding standards and engineering practices only.
- Do not review domain correctness, product behavior, contract compliance, or feature completeness unless that context is required to explain a coding-standard drift.
- This command is rerunnable. Overwrite the report with a fresh snapshot, but preserve existing `DRIFT-###` identifiers for substantively unchanged findings when possible.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `spec.md`, `plan.md`, and `tasks.md` exist in `FEATURE_DIR`.
4. Load `AGENTS.md`.
5. Load `.specify/memory/constitution.md` if it exists.

## Review Scope

- Use the active feature implementation as the primary scope.
- Derive candidate files from `tasks.md`, `plan.md`, current feature documentation, and the implementation files that support the feature.
- Expand to adjacent files only when needed to explain architectural boundaries, duplication, or cross-cutting drift.
- Prefer exact file and line evidence over broad repository-wide claims.

## Standards Baseline

Evaluate the implementation against the repository engineering policy baseline:

1. `AGENTS.md`
2. `.specify/memory/constitution.md` when present

Prioritize repo-defined rules around:

- descriptive and unambiguous naming
- SRP, cohesion, and architectural boundaries
- DRY and consistency
- separation of domain, application, presentation, and infrastructure concerns where the repo expects it
- documentation and author-attribution requirements for AI-touched code
- any local style rules explicitly stated by the repo

## Outline

1. Load the existing `code-standard-drift-report.md` if it exists. Use it only to preserve stable `DRIFT-###` identifiers for equivalent findings and to keep the correction-tracking link consistent.
2. Read the feature artifacts:
   - `spec.md`
   - `plan.md`
   - `tasks.md`
   - `research.md`, `data-model.md`, `quickstart.md`, and `contracts/` when they help define the implementation surface
3. Inspect the relevant implementation files in the repository.
4. Identify only concrete drift items that have explicit repository-policy evidence. Avoid speculative or generic commentary that is not grounded in `AGENTS.md` or the constitution.
5. Assign severity:
   - `High`: architectural boundary drift, mixed responsibilities across layers, or duplication and cohesion problems with clear maintenance or evolution risk
   - `Medium`: decomposition, documentation, or consistency drift that weakens maintainability but is not immediately architecture-breaking
   - `Low`: local style, attribution, or minor consistency drift with limited structural risk
6. Reuse existing `DRIFT-###` IDs when the finding is substantively the same. Assign new IDs sequentially after the highest existing drift number only for new findings.
7. Write `FEATURE_DIR/code-standard-drift-report.md`, overwriting the previous file.

## Output Format

Write a Markdown report with this structure:

```markdown
# Code Standard Drift Report: [Feature Name]

**Purpose**: Record concrete deviations between the current implementation and the repository coding standards baseline for the active feature slice.
**Created**: [YYYY-MM-DD]
**Feature**: [spec.md](./spec.md)
**Correction Tracking**: [checklists/code-standard-drift-remediation.md](./checklists/code-standard-drift-remediation.md)

## Scope

- This report covers coding standards and engineering practices only.
- This report does not cover feature-scope correctness, contract compliance, constitution-gate evidence, or domain-spec validation.
- Evidence references below are a point-in-time snapshot from the current implementation tree.

## Standards Baseline

[List the baseline files and the specific standard clauses or principles used in the review.]

## Findings

### DRIFT-001: [Short Title]

**Severity**: [High | Medium | Low]
**Diverges from**:

- [Specific policy reference]
- [Specific policy reference]

**Evidence**:

- `path/to/file.ext:10-30`
- `path/to/file.ext:44-52`

**Description**:

[Concrete explanation of the drift and why it matters in this repository.]

## Notes

- [Any zero-findings note, scope limitation, or validation note]
```

## Rules

- The report MUST be implementation-specific, not a generic coding-style lecture.
- Every finding MUST cite exact file references.
- Every finding MUST map to an explicit repo policy source.
- Do not include remediation checkboxes in this report. That belongs in the remediation checklist.
- If no drift is found, still write the report with `## Findings` stating that no coding-standard drift was identified in the reviewed scope.
- Keep wording grounded and empirical.
