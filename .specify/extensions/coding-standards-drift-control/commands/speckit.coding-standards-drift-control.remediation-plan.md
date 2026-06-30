---
description: "Append Spec Kit remediation tasks to tasks.md from the drift report"
---

# Generate Coding Standards Drift Remediation Plan

Append a drift remediation phase to the active feature `tasks.md` from the current coding-standards drift report.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may ask to focus on specific drift IDs or narrow task generation to a subset of severities.

## Purpose

- Add actionable remediation tasks to the active feature's `tasks.md` so `/speckit.implement` can execute them.
- Run only after the normal implementation task list is complete.
- Keep generated tasks tied to their source pending `CODE-STAND-DRIFT-###` report topics for context.
- Plan remediation only for findings whose report `Status` is `Pending`.
- Reason through each selected pending finding against the report evidence and loaded coding-standards baseline before generating tasks.
- Add or update a concise `**Remediation plan**:` section in each selected pending report finding, describing the surgical solution that should fix the drift without changing production or test behavior unnecessarily.
- Add the final task that updates successfully remediated report findings to `Status: Resolved`; this generated task is the only workflow point that may set findings to `Resolved`.
- Do not create `coding-standards-drift-remediation.md` or any other separate remediation checklist.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `FEATURE_DIR/coding-standards-drift-report.md` and `FEATURE_DIR/tasks.md` exist. If the report is missing, stop and instruct the user to run `/speckit.coding-standards-drift-control.report` first.
4. Load the current Spec Kit task format references before editing `tasks.md`:
   - `.specify/templates/tasks-template.md` when present
   - the existing `FEATURE_DIR/tasks.md`
5. Using the task state syntax from the current local Spec Kit installation and the existing task file, verify there are no open, unchecked, pending, or reopened tasks in `FEATURE_DIR/tasks.md`. If any are present, stop without editing `coding-standards-drift-report.md` or `tasks.md` and instruct the user to finish implementation with `/speckit.implement` before planning drift remediation.

## Outline

1. Read `FEATURE_DIR/coding-standards-drift-report.md`.
2. Extract every `CODE-STAND-DRIFT-###` finding, status, severity, title, policy references, evidence paths, description, existing remediation plan, and report section anchor from the report. If the report still contains legacy `DRIFT-###` identifiers, normalize them to `CODE-STAND-DRIFT-###` while preserving the numeric suffix. Treat findings without an explicit status as `Pending` for older reports. The finding title is the target topic that each generated task must reference.
3. Extract the report's `## Standards Baseline` references and load the referenced files that are present in the repository or feature scope, such as `AGENTS.md`, known agent-instruction files, and `.specify/memory/constitution.md`. Use the report's quoted policy references when a referenced baseline file is not available.
4. Select only findings whose status is `Pending`. If the report contains no pending findings, do not change `tasks.md`; report that no remediation tasks were generated.
5. For each selected pending finding, reason from the finding description, evidence paths, policy references, and loaded coding-standards baseline to identify the smallest safe remediation. The plan must preserve production and test behavior, identify any behavior-sensitive areas that require special care, and avoid too broad rewrites unrelated to the cited drift.
6. Add or update a concise `**Remediation plan**:` section in each selected pending finding in `FEATURE_DIR/coding-standards-drift-report.md`. Place it after `**Description**` and before `**Resolution**` when a resolution section is present. The section must describe the proposed fix, why it satisfies the cited standards, and any focused validation that should accompany the change. Do not set any finding to `Resolved` while adding remediation plans.
7. Write `FEATURE_DIR/coding-standards-drift-report.md` with the remediation plans before editing `tasks.md`.
8. Read `FEATURE_DIR/tasks.md` and identify the current task numbering, phase heading style, separator style, path-reference style, and task checkbox syntax from the file itself and the loaded Spec Kit references.
9. Remove from the task-generation set any selected pending finding that already has a matching `CODE-STAND-DRIFT-###` task, or a legacy `DRIFT-###` task with the same numeric suffix, in `tasks.md`. If no selected pending findings remain for task generation, do not change `tasks.md`; report that remediation plans were recorded but no new remediation tasks were generated.
10. Append a new final phase dedicated to coding-standards drift remediation, following the phase structure used by the current `tasks.md` rather than a hard-coded template.
11. Create one unchecked Spec Kit task per remaining selected pending drift finding:
    - continue task IDs from the highest existing task ID in `tasks.md`
    - use the task checkbox and task-line conventions from the current local Spec Kit installation
    - include the `CODE-STAND-DRIFT-###` ID, severity, and finding title
    - reference `coding-standards-drift-report.md` and the finding's report topic or anchor
    - include the evidence file paths from the report when they are available
    - base the requested work on the finding's recorded `**Remediation plan**:` section
    - phrase the work as implementation-oriented remediation that `/speckit.implement` can execute
12. Add verification tasks only when they are required by the current Spec Kit task conventions or by the remaining selected pending report findings. Use existing project validation commands from the feature plan or repository files.
13. Add one final unchecked Spec Kit task after all remediation and verification tasks. This task MUST be the last task in the appended phase and MUST instruct `/speckit.implement` to update `coding-standards-drift-report.md` by changing the `Status` of only the findings successfully remediated by the preceding tasks to `Resolved` and adding or updating their `Resolution` notes.
14. Write `FEATURE_DIR/tasks.md`.
15. Report which drift findings received remediation plans, which drift remediation tasks were appended, and remind the user to run `/speckit.implement`.

## Rules

- Do not edit `coding-standards-drift-report.md` or `tasks.md` while any existing task remains open, unchecked, pending, or reopened.
- Do not create remediation checklist files.
- Do not generate tasks for findings whose report `Status` is `Resolved`.
- Do not duplicate remediation tasks for a `CODE-STAND-DRIFT-###` already present in `tasks.md`; treat a legacy `DRIFT-###` reference with the same numeric suffix as a duplicate during migration.
- Before appending remediation tasks, every selected pending finding MUST have a `**Remediation plan**:` section in `coding-standards-drift-report.md`.
- Each `**Remediation plan**:` section MUST be grounded in the report evidence, cited policy references, and loaded coding-standards baseline files, and MUST describe a focused change that avoids unnecessary production or test code behavior changes.
- Generated remediation tasks MUST be based on the corresponding report `**Remediation plan**:` section instead of generic task text.
- The final generated task MUST be the only task or command instruction that sets report finding status to `Resolved`, and it MUST NOT mark a finding resolved unless its matching remediation work has completed successfully.
- Do not hard-code a task phase or task item format. Derive the format from the current local Spec Kit installation and the existing task file.
- Keep task text actionable, implementation-oriented, and scoped to coding-standards remediation.
- Every generated task MUST reference its source `CODE-STAND-DRIFT-###` topic in `coding-standards-drift-report.md`.
