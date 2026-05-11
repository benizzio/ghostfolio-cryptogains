---
description: Generate or extend code-standard-drift-remediation.md from the drift report
---

<!-- Extension: coding-standards-drift-analysis -->
<!-- Config: .specify/extensions/coding-standards-drift-analysis/ -->
# Generate Code Standard Drift Remediation Checklist

Generate or extend the remediation checklist for the active feature code-standard drift report.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may request a full refresh, ask to focus on specific drift IDs, or narrow checklist generation to a subset of severities.

## Purpose

- Create actionable remediation items only for drift findings that do not already have checklist entries.
- Preserve existing checkbox state and any manually tracked progress.
- Keep the checklist additive and idempotent across reruns.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `FEATURE_DIR/code-standard-drift-report.md` exists. If it does not, stop and instruct the user to run `/speckit.coding-standards-drift-analysis.report` first.

## Outline

1. Read `FEATURE_DIR/code-standard-drift-report.md`.
2. Read `FEATURE_DIR/checklists/code-standard-drift-remediation.md` if it exists.
3. Extract every `DRIFT-###` finding and its severity from the report.
4. For each finding:
   - If the checklist already contains an item referencing the same `DRIFT-###`, preserve that item and its current `[x]` or `[ ]` state.
   - If the checklist does not contain that drift ID, add a new unchecked item in the severity section that matches the report.
5. Keep severity sections in descending order for the severities that are present: `Critical`, `High`, `Medium`, `Low`.
6. Add or preserve a `## Closure Criteria` section with generic completion checks.
7. Write `FEATURE_DIR/checklists/code-standard-drift-remediation.md`, creating the `checklists/` directory if necessary.
8. Report which checklist items were added and which existing items were preserved.

## Output Format

Write a Markdown checklist with this structure:

```markdown
# Checklist: Code Standard Drift Remediation

**Purpose**: Track correction of the coding-standard drift recorded in [`../code-standard-drift-report.md`](../code-standard-drift-report.md).
**Created**: [YYYY-MM-DD]
**Feature**: [spec.md](../spec.md)

## High Priority

- [ ] DRIFT-001 [Actionable remediation task]

## Medium Priority

- [ ] DRIFT-002 [Actionable remediation task]

## Low Priority

- [ ] DRIFT-003 [Actionable remediation task]

## Closure Criteria

- [ ] Re-run the coding-standards review after remediation and confirm that every drift item in `../code-standard-drift-report.md` is either resolved or intentionally reclassified.
- [ ] Confirm the remediation changes preserve project-owned automated coverage expectations.
- [ ] Confirm any updated public API comments or author-attribution notes remain accurate after the code changes.
```

## Rules

- Add only missing checklist items. Do not duplicate items that already reference the same drift ID.
- Preserve existing `[x]` and `[ ]` states for matching items.
- Keep checklist text actionable and implementation-oriented.
- Do not silently remove historical items from an existing checklist. If a finding disappeared from the latest report, leave the checklist untouched and let the user decide whether to archive it.
- If the report contains no findings, still create or preserve the checklist and note that no new remediation items were generated.
