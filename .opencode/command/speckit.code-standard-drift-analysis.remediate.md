---
description: Execute unresolved remediation tasks and correct flagged code-standard drift
---

<!-- Extension: code-standard-drift-analysis -->
<!-- Config: .specify/extensions/code-standard-drift-analysis/ -->
# Remediate Code Standard Drift

Execute unresolved remediation tasks for the active feature and update the remediation checklist as work completes.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The user may specify one or more `DRIFT-###` identifiers, a severity band, or `all open`. If no scope is provided, default to all unchecked checklist items.

## Prerequisites

1. Verify a Spec Kit project exists by checking for `.specify/`.
2. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse the absolute `FEATURE_DIR`.
3. Verify both of these files exist:
   - `FEATURE_DIR/code-standard-drift-report.md`
   - `FEATURE_DIR/checklists/code-standard-drift-remediation.md`
4. If either file is missing, stop and instruct the user to run the report and checklist commands first.

## Execution Rules

- Treat the checklist as the execution queue.
- Default target set: all unchecked checklist items.
- If the user specifies one or more `DRIFT-###` identifiers, work only those items.
- Make the smallest code changes that remove the drift.
- Preserve unrelated worktree changes.
- Update tests, documentation, or author-attribution notes when required by the remediation.
- Do not mark a checklist item complete until the code change is in place and relevant verification has been run.

## Outline

1. Load `AGENTS.md`, `.specify/memory/constitution.md` when present, the drift report, the remediation checklist, and the relevant feature artifacts.
2. Determine the target drift items from user input or from all unchecked checklist entries.
3. For each target item:
   - read the cited evidence files and enough surrounding code to understand the drift
   - implement the minimal remediation in code and supporting docs or tests
   - run the smallest relevant validation commands for touched areas; run broader project checks too when the plan or repo policy requires them and when feasible
   - if the drift is resolved and verified, mark the checklist item `[x]`
   - if the drift is blocked or only partially addressed, leave it `[ ]` and explain the blocker in the final report
4. Update the `## Closure Criteria` checkboxes only for criteria that were empirically satisfied during this run.
5. Report:
   - touched files
   - validations run
   - checklist items completed
   - remaining open or blocked items
6. Recommend rerunning `/speckit.code-standard-drift-analysis.report` and `/speckit.code-standard-drift-analysis.checklist` when the user wants a refreshed post-remediation snapshot.

## Rules

- Make minimal, targeted changes.
- Respect the repository coding standards while remediating the drift report.
- Do not modify unrelated checklist items.
- If no unchecked items remain, report that nothing needs remediation and do not make file changes.
