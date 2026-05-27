---
description: "Append Spec Kit coverage remediation tasks to tasks.md from the drift report"
---

<!-- Extension: test-coverage-drift-control -->
<!-- Config: .specify/extensions/test-coverage-drift-control/ -->
# Generate Test Coverage Drift Remediation Plan

Append a coverage drift remediation phase to the active feature `tasks.md` from the current test-coverage drift report.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may ask to focus on specific coverage drift IDs or narrow task generation to a subset of severities.

## Purpose

- Add actionable remediation tasks to the active feature's `tasks.md` so `/speckit.implement` can execute them.
- Run only after the normal implementation task list is complete.
- Keep generated tasks tied to their source `COV-DRIFT-###` report topics for context.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify `FEATURE_DIR/test-coverage-drift-report.md` and `FEATURE_DIR/tasks.md` exist. If the report is missing, stop and instruct the user to run `/speckit.test-coverage-drift-control.report` first.
4. Load the current Spec Kit task format references before editing `tasks.md`:
   - `.specify/templates/tasks-template.md` when present
   - the existing `FEATURE_DIR/tasks.md`
5. Using the task state syntax from the current local Spec Kit installation and the existing task file, verify there are no open, unchecked, pending, or reopened tasks in `FEATURE_DIR/tasks.md`. If any are present, stop without editing `tasks.md` and instruct the user to finish implementation with `/speckit.implement` before planning coverage drift remediation.

## Outline

1. Read `FEATURE_DIR/test-coverage-drift-report.md`.
2. Extract every `COV-DRIFT-###` finding, severity, title, evidence paths, and report section anchor from the report. The finding title is the target topic that each generated task must reference.
3. If the report contains no findings, do not change `tasks.md`; report that no remediation tasks were generated.
4. Read `FEATURE_DIR/tasks.md` and identify the current task numbering, phase heading style, separator style, path-reference style, and task checkbox syntax from the file itself and the loaded Spec Kit references.
5. Append a new final phase dedicated to test-coverage drift remediation, following the phase structure used by the current `tasks.md` rather than a hard-coded template.
6. Create one unchecked Spec Kit task per selected coverage drift finding:
   - continue task IDs by finding existing IDs that match `T` followed by digits, incrementing the highest numeric suffix, and preserving the existing numeric width
   - ignore mixed-prefix IDs and non-numeric suffix variants such as `T012a` when deriving the next numeric task ID
   - use the task checkbox and task-line conventions from the current local Spec Kit installation
   - include the `COV-DRIFT-###` ID, severity, and finding title
   - reference `test-coverage-drift-report.md` and the finding's report topic or anchor
   - include the evidence file paths from the report when they are available
   - phrase the work as implementation-oriented remediation that `/speckit.implement` can execute
   - make the task explicitly address the coverage target, coverage gate instrumentation, required test type, or test-structure drift identified by the finding
7. Add verification tasks when they are required by the current Spec Kit task conventions, by the report findings, or by the coverage baseline. Use existing project validation commands from the feature plan, repository files, or coverage definition reference files.
8. Write `FEATURE_DIR/tasks.md`.
9. Report which coverage drift remediation tasks were appended and remind the user to run `/speckit.implement`.

## Rules

- Do not edit `tasks.md` while any existing task remains open, unchecked, pending, or reopened.
- Do not create remediation checklist files.
- Do not duplicate remediation tasks for a `COV-DRIFT-###` already present in `tasks.md`; report the duplicate and skip it.
- Do not hard-code a task phase or task item format. Derive the format from the current local Spec Kit installation and the existing task file.
- Keep task text actionable, implementation-oriented, and scoped to coverage remediation.
- Every generated task MUST reference its source `COV-DRIFT-###` topic in `test-coverage-drift-report.md`.
- Generated tasks MUST be suitable for `/speckit.implement` and MUST not ask for manual-only review unless the coverage baseline requires manual evidence.
