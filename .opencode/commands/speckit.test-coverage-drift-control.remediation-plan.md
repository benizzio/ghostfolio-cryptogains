---
description: Record remediation plans and append Spec Kit coverage remediation tasks
  from the drift report
---


<!-- Extension: test-coverage-drift-control -->
<!-- Config: .specify/extensions/test-coverage-drift-control/ -->
# Generate Test Coverage Drift Remediation Plan

Record a focused plan for each selected pending finding, then append a coverage drift remediation phase to the active feature `tasks.md`.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may ask to focus on specific coverage drift IDs or narrow task generation to a subset of severities.

## Purpose

- Add actionable remediation tasks to the active feature's `tasks.md` so `/speckit.implement` can execute them.
- Run only after the normal implementation task list is complete.
- Keep generated tasks tied to their source pending `COV-DRIFT-###` report topics for context.
- Plan remediation only for findings whose report `Status` is `Pending`.
- Reason through each selected pending finding against the report evidence and loaded coverage baseline before generating tasks.
- Add or update a concise `**Remediation plan**:` section in each selected pending report finding, describing the smallest safe solution that corrects the coverage drift without unnecessarily changing production or existing test behavior.
- Add a final task that updates successfully remediated report findings to `Status: Resolved`; this generated task is the only workflow point that may set findings to `Resolved`.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `FEATURE_DIR/test-coverage-drift-report.md` and `FEATURE_DIR/tasks.md` exist. If the report is missing, stop and instruct the user to run `/speckit.test-coverage-drift-control.report` first.
4. Load the current Spec Kit task format references before editing `tasks.md`:
   - `.specify/templates/tasks-template.md` when present
   - the existing `FEATURE_DIR/tasks.md`
5. Using the task state syntax from the current local Spec Kit installation and the existing task file, verify there are no open, unchecked, pending, or reopened tasks in `FEATURE_DIR/tasks.md`. If any are present, stop without editing `test-coverage-drift-report.md` or `tasks.md` and instruct the user to finish implementation with `/speckit.implement` before planning coverage drift remediation.

## Outline

1. Read `FEATURE_DIR/test-coverage-drift-report.md`.
2. Extract every `COV-DRIFT-###` finding, status, severity, title, coverage-policy references, evidence paths, description, existing remediation plan, and report section anchor from the report. Treat findings without an explicit status as `Pending` for older reports. The finding title is the target topic that each generated task must reference.
3. Extract the report's `## Coverage Baseline` references and load the referenced files that are present in the repository or feature scope, including coverage configuration, maintained test commands, CI workflows, and coverage-gate tooling when cited. Re-read the current implementation and test files referenced by each finding's evidence. Use the report's quoted policy references when a referenced baseline file is not available.
4. Select only findings whose status is `Pending` and that match any user-requested ID or severity scope. If the report contains no matching pending findings, do not change `test-coverage-drift-report.md` or `tasks.md`; report that no remediation plans or tasks were generated.
5. For each selected pending finding, verify the cited drift against the current implementation and then reason from its description, current evidence, coverage-policy references, and loaded baseline to identify the smallest safe remediation. Prefer focused assertions or tests, coverage instrumentation corrections, and coverage-gate wiring. Preserve intended production behavior and existing test intent; do not weaken coverage targets, exclusions, instrumentation scope, required test layers, or assertions merely to make coverage pass. If the current implementation already satisfies the baseline, plan focused validation only instead of unnecessary code changes.
6. Add an explicit `**Status**: Pending` to any selected older finding that lacks status, then add or update a concise `**Remediation plan**:` section in each selected pending finding in `FEATURE_DIR/test-coverage-drift-report.md`. Place the plan after `**Description**` and before `**Resolution**` when a resolution section is present. The plan must describe the proposed correction, why it satisfies the cited coverage baseline, any behavior-sensitive areas to preserve, and focused validation that should accompany the change. Do not set any finding to `Resolved` while adding remediation plans.
7. Write `FEATURE_DIR/test-coverage-drift-report.md` with the remediation plans before editing `tasks.md`.
8. Read `FEATURE_DIR/tasks.md` and identify the current task numbering, phase heading style, separator style, path-reference style, and task checkbox syntax from the file itself and the loaded Spec Kit references.
9. Treat completed matching `COV-DRIFT-###` tasks in existing phases as historical attempts, not duplicates: a finding that is still or again `Pending` requires a new remediation or validation attempt. Ensure each selected pending finding appears only once in the new phase.
10. Append a new final phase dedicated to test-coverage drift remediation, following the phase structure used by the current `tasks.md` rather than a hard-coded template.
11. Create one unchecked Spec Kit task per selected pending coverage drift finding:
    - continue task IDs by finding existing IDs that match `T` followed by digits, incrementing the highest numeric suffix, and preserving the existing numeric width
    - ignore mixed-prefix IDs and non-numeric suffix variants such as `T012a` when deriving the next numeric task ID
    - use the task checkbox and task-line conventions from the current local Spec Kit installation
    - include the `COV-DRIFT-###` ID, severity, and finding title
    - reference `test-coverage-drift-report.md` and the finding's report topic or anchor
    - include the evidence file paths from the report when they are available
    - base the requested work on the finding's recorded `**Remediation plan**:` section
    - phrase the work as implementation-oriented remediation that `/speckit.implement` can execute
    - make the task explicitly address the coverage target, coverage gate instrumentation, required test type, or test-structure drift identified by the finding
    - include the plan's focused validation in the task or identify the matching verification task that must pass before the finding can be resolved
12. Add verification tasks when they are required by the current Spec Kit task conventions, by the selected pending report findings, or by the coverage baseline. Use existing project validation commands from the feature plan, repository files, or coverage definition reference files.
13. Add one final unchecked Spec Kit task after all remediation and verification tasks. This task MUST be the last task in the appended phase and MUST enumerate the `COV-DRIFT-###` IDs planned in that phase and instruct `/speckit.implement` to update `test-coverage-drift-report.md` by changing the `Status` of only enumerated findings whose remediation and focused validation completed successfully to `Resolved`, adding or updating their `Resolution` notes, and leaving unsuccessful or unvalidated findings `Pending`. It must preserve all other findings, remediation plans, historical resolution context, and report content.
14. Write `FEATURE_DIR/tasks.md`.
15. Report which coverage drift findings received remediation plans, which remediation tasks were appended, and remind the user to run `/speckit.implement`.

## Rules

- Do not edit `test-coverage-drift-report.md` or `tasks.md` while any existing task remains open, unchecked, pending, or reopened.
- Do not create remediation checklist files.
- Do not generate tasks for findings whose report `Status` is `Resolved`.
- Do not generate more than one remediation task for the same `COV-DRIFT-###` in a single appended phase. A completed task from an earlier phase does not disqualify a finding that remains or becomes `Pending` from receiving a new attempt.
- Before appending remediation tasks, every selected pending finding MUST have a `**Remediation plan**:` section in `test-coverage-drift-report.md`.
- Each `**Remediation plan**:` section MUST be grounded in the report evidence, cited coverage-policy references, and loaded coverage baseline files, and MUST describe a focused change that avoids unnecessary production or existing test behavior changes.
- Generated remediation tasks MUST be based on the corresponding report `**Remediation plan**:` section instead of generic task text.
- The final generated task MUST be the only task or command instruction that sets report finding status to `Resolved`. It MUST be limited to the finding IDs planned in the current phase, MUST preserve all unrelated report content, and MUST NOT mark a finding resolved unless its matching remediation and focused validation completed successfully.
- Do not weaken coverage targets, exclusions, instrumentation scope, required test layers, assertions, or CI gates as remediation unless the cited coverage baseline explicitly requires that change.
- Do not hard-code a task phase or task item format. Derive the format from the current local Spec Kit installation and the existing task file.
- Keep task text actionable, implementation-oriented, and scoped to coverage remediation.
- Every generated task MUST reference its source `COV-DRIFT-###` topic in `test-coverage-drift-report.md`.
- Generated tasks MUST be suitable for `/speckit.implement` and MUST not ask for manual-only review unless the coverage baseline requires manual evidence.