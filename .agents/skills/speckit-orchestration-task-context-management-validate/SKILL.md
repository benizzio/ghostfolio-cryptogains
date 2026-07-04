---
name: speckit-orchestration-task-context-management-validate
description: Validate that tasks.md has a current work-unit orchestration ledger
compatibility: Requires spec-kit project structure with .specify/ directory
metadata:
  author: github-spec-kit
  source: orchestration-task-context-management:commands/speckit.orchestration-task-context-management.validate.md
---

# Validate Orchestration Work Units

Validate that the active feature `tasks.md` contains a complete, current context-orchestration work-unit ledger before `/speckit.implement` starts.

This command is read-only. It must not modify any file.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding. The user may provide a feature directory or `tasks.md` path to validate.

## Prerequisites

1. Verify the repository is a Spec Kit project by checking for `.specify/`.
2. Locate the target `tasks.md`:
   - If `$ARGUMENTS` names a file, use that file only when it ends with `tasks.md` and is under `specs/`.
   - If `$ARGUMENTS` names a feature directory, use `<feature-dir>/tasks.md`.
   - Otherwise run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from the repository root and parse the absolute `FEATURE_DIR`.
3. Read `FEATURE_DIR/tasks.md`.
4. Read `.specify/templates/tasks-template.md` only as a task-format reference when present. Do not edit it.
5. When collecting checklist task IDs, ignore the generated orchestration section, ledger table rows, subagent examples, dependency prose, and fenced code blocks.

## Validation Checklist

Validate all of the following:

1. `## Context Orchestration Process` appears exactly once.
2. The orchestration section contains these subsections exactly once:
   - `### Parent Orchestrator Rules`
   - `### Required Subagent Handoff Packet`
   - `### Parent Verification Gate`
   - `### Context Compaction Recovery`
   - `### Work Unit Ledger`
3. The work-unit ledger table has exactly these columns:

   ```markdown
   | Unit | Phase | Tasks | Atomic scope and touched paths | Prerequisites | Required handoff sources | Parent verification |
   ```

4. Every checklist task ID matching `T` followed by digits appears in exactly one ledger row.
5. Every task ID referenced by the ledger exists in the checklist.
6. No ledger row references duplicate task IDs.
7. Work-unit IDs are sequential as `WU01`, `WU02`, and so on.
8. Every prerequisite is `None` or references only earlier `WU##` work units.
9. No prerequisite references a later or missing work unit.
10. Every ledger row has non-empty phase, tasks, atomic scope, required handoff sources, and parent verification cells.
11. Test-classified units appear before implementation-classified units for the same user story or behavior.
12. The parent rules state that task checkboxes remain authoritative and are marked only after parent verification.
13. The handoff packet requires exact task IDs, exact task descriptions, exact paths, allowed edit paths, validation commands, and final response fields.
14. The parent verification gate requires diff inspection and targeted verification.
15. The context compaction recovery procedure explains how a new parent resumes from checked tasks, current diffs, and the first incomplete work unit.
16. If `### Parallel Opportunities` exists, it states that work-unit ordering controls orchestration and task-level `[P]` markers are local hints.
17. `tasks.md` does not still contain obvious template placeholders such as `[FEATURE NAME]`, `[Title]`, `TXXX`, or sample paths from the template.

## Freshness Checks

Report the ledger as stale if any of these are true:

- a task ID exists in the checklist but not in the ledger
- a ledger task ID no longer exists in the checklist
- a task ID appears in more than one work unit
- a work unit has an empty parent verification cell
- a prerequisite is not `None` and contains anything other than earlier `WU##` references
- the file contains more than one orchestration process or ledger table

## Report

Return one of these outcomes:

```markdown
# Orchestration Validation Passed

| Field | Value |
|-------|-------|
| Feature directory | `<FEATURE_DIR>` |
| Tasks covered | `<covered>/<total>` |
| Work units | `<count>` |

Ready for `/speckit.implement`.
```

or:

```markdown
# Orchestration Validation Failed

| Field | Value |
|-------|-------|
| Feature directory | `<FEATURE_DIR>` |
| Tasks covered | `<covered>/<total>` |
| Work units | `<count or unknown>` |

## Findings

- `<specific failure>`

Run `/speckit.orchestration-task-context-management.prepare` to refresh `tasks.md`, then rerun this validation.
```

## Rules

- Read-only command. Do not edit files.
- Do not start implementation if validation fails.
- Do not accept a ledger that omits tasks, references missing tasks, or has invalid prerequisites.
- Do not require changes to `.specify/templates/`; this extension validates only the generated active feature `tasks.md`.