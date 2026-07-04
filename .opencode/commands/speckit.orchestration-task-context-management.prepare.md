---
description: Insert or refresh subagent work-unit orchestration in tasks.md
---


<!-- Extension: orchestration-task-context-management -->
<!-- Config: .specify/extensions/orchestration-task-context-management/ -->
# Prepare Orchestration Work Units

Analyze the active feature `tasks.md` and surgically insert or refresh the context-orchestration work-unit control plane used by a parent `/speckit.implement` agent to delegate bounded work units to clean-context subagents.

This command edits only the active feature `tasks.md`. It must not edit `.specify/templates/`, spec sources, implementation code, or any other artifact.

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding. The user may provide a feature directory, a `tasks.md` path, or refresh instructions such as `refresh`, `strict`, `max-unit-tasks=6`, or `no-examples`.

## Purpose

- Keep the generated Spec Kit task checklist intact while adding an implementation control plane for context-bounded subagent delegation.
- Convert existing task IDs into ordered atomic work units that a parent orchestrator can hand off, verify, and mark complete only after inspection.
- Stay independent of future Spec Kit template changes by deriving format from the current local `tasks.md` and using `.specify/templates/tasks-template.md` only as a read-only reference.

## Prerequisites

1. Verify the repository is a Spec Kit project by checking for `.specify/`.
2. Locate the target `tasks.md`:
   - If `$ARGUMENTS` names a file, use that file only when it ends with `tasks.md` and is under `specs/`.
   - If `$ARGUMENTS` names a feature directory, use `<feature-dir>/tasks.md`.
   - Otherwise run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from the repository root and parse the absolute `FEATURE_DIR`.
3. Verify `FEATURE_DIR/tasks.md` exists and is writable.
4. Load these read-only references when present:
   - `.specify/templates/tasks-template.md`
   - `FEATURE_DIR/spec.md`
   - `FEATURE_DIR/plan.md`
   - `FEATURE_DIR/research.md`
   - `FEATURE_DIR/data-model.md`
   - `FEATURE_DIR/quickstart.md`
   - files under `FEATURE_DIR/contracts/`
5. Refuse to continue if `tasks.md` still contains obvious template placeholders such as `[FEATURE NAME]`, `[Title]`, `TXXX`, or sample paths from the template.
6. Parse all checklist tasks in `tasks.md`, excluding the generated orchestration section, ledger table rows, subagent examples, dependency prose, and code fences. Refuse to continue if no task IDs are found, if any task ID is duplicated in the checklist, or if any executable task cannot be assigned to exactly one work unit.
7. If `## Context Orchestration Process` or `### Work Unit Ledger` already exists, treat it as generated content and refresh it instead of appending a duplicate section.

## Reverse-Engineered Pattern

The target shape comes from the experimented `tasks.md` pattern:

1. The normal Spec Kit task checklist remains authoritative.
2. A `## Context Orchestration Process` section is inserted before the first phase heading.
3. The orchestration section contains parent rules, a required handoff packet, a parent verification gate, context compaction recovery, and a work-unit ledger.
4. The ledger maps existing task IDs to `WU##` units with phase, scope, prerequisites, handoff sources, and parent verification instructions.
5. Task-level `[P]` markers become local hints only; the ledger controls cross-unit execution order and parent verification.
6. Handoff examples and implementation strategy notes explain how the parent agent delegates to clean-context subagents without giving up final verification.

## Analysis Procedure

1. Parse `tasks.md` into sections:
   - preamble before the first `## Phase`
   - phase headings and phase bodies
   - dependencies, parallel opportunities, implementation strategy, and notes sections when present
2. Parse each checklist task line into:
   - checkbox state
   - task ID, preserving numeric width
   - `[P]` marker if present
   - story label such as `[US1]` when present
   - raw description
   - exact repository paths enclosed in backticks
   - explicit dependency text such as `depends on T012`
3. Classify each task using phase heading, subsection heading, and task text:
   - setup
   - foundation
   - fail-first tests
   - implementation
   - documentation
   - final validation
   - drift remediation or bugfix follow-up
4. Derive project-specific constraints from the loaded artifacts and current tasks:
   - required test-before-implementation rules
   - architectural boundaries from `plan.md`
   - domain invariants from `spec.md` and `data-model.md`
   - provider, security, persistence, fixture, or generated-file restrictions from `research.md`, `contracts/`, and `quickstart.md`
   - quality gates and validation commands from `plan.md`, `quickstart.md`, Makefiles, CI docs, or existing task text
5. Build ordered work units:
   - Keep every unit within one phase unless it is an explicit final validation or cross-cutting remediation unit.
   - Do not mix fail-first test tasks with implementation tasks.
   - Do not mix unrelated user stories in one unit.
   - Prefer 1 to 6 tasks per unit by default, or the numeric limit from `max-unit-tasks=<N>` when provided; use a smaller unit when files, packages, or risks are unrelated.
   - Keep tightly coupled tasks together when they touch the same package, public boundary, or verification target.
   - Split tasks that touch different architectural boundaries even if `[P]` is present.
   - Put parent-owned final validation, coverage, release, or empirical-dataset checks in a final gate unit rather than a normal subagent unit.
6. Assign `WU01`, `WU02`, and so on in execution order.
7. Derive prerequisites:
   - setup before foundation
   - foundation before story work
   - fail-first tests before implementation for the same story or behavior
   - implementation before dependent integration, rendering, runtime, or documentation units
   - explicit task dependency text before implicit heuristics
   - final validation after all implementation and remediation units
8. Derive parent verification for each unit:
   - targeted tests from task descriptions and `quickstart.md`
   - closest package compile/test command when tasks only add models or scaffolding
   - diff inspection against allowed paths
   - boundary re-read for public types, contracts, rendering output, persistence, security, or diagnostics changes

## Required Edit Shape

Update `tasks.md` with these changes only:

1. If the preamble has an `**Organization**:` line, append or refresh one sentence stating that the context-orchestration work-unit ledger is the execution control plane for parent agents and clean-context subagents. Preserve the original organization meaning.
2. Insert or replace `## Context Orchestration Process` immediately before the first `## Phase` heading.
3. The generated `## Context Orchestration Process` section MUST include these subsections in this order:
   - `### Parent Orchestrator Rules`
   - `### Required Subagent Handoff Packet`
   - `### Parent Verification Gate`
   - `### Context Compaction Recovery`
   - `### Work Unit Ledger`
4. Generate the work-unit ledger table with exactly these columns:

   ```markdown
   | Unit | Phase | Tasks | Atomic scope and touched paths | Prerequisites | Required handoff sources | Parent verification |
   |------|-------|-------|--------------------------------|---------------|--------------------------|---------------------|
   ```

5. After the dependencies or parallel opportunities section, insert or refresh `## Subagent Handoff Examples` unless `$ARGUMENTS` contains `no-examples`. Include one test-oriented example and one implementation-oriented example when those unit types exist.
6. If `### Parallel Opportunities` exists, refresh it so it states that work-unit ordering controls orchestration and `[P]` markers are local hints only. Preserve any existing concrete parallel opportunities that remain valid, but express them in terms of work-unit prerequisites and parent verification.
7. If `## Implementation Strategy` exists, add or refresh a `### Context-Orchestrated Subagent Strategy` subsection. If it does not exist, append a concise `## Implementation Strategy` section before `## Notes` or at the end of the file.
8. If `## Notes` exists, add concise orchestration notes only when they are not already present. Do not duplicate existing notes.

## Required Orchestration Content

The generated parent rules MUST state:

- Execute work units in ledger order unless the ledger explicitly marks units as parallel candidates and their prerequisites are verified.
- The task checkboxes remain authoritative; a ledger unit is complete only when every referenced task is checked after parent verification.
- Use a clean subagent session for each delegated work unit.
- Include all required handoff context in the subagent prompt; do not rely on prior conversation state.
- Keep subagents inside the listed scope and require them to stop before editing outside allowed paths.
- Require fail-first behavior for test tasks.
- Parent must inspect diffs, run targeted verification, check for unrelated changes, and fix or re-delegate inconsistencies before starting a dependent unit.
- Parent owns final validation and must rerun final gates even if a subagent helped triage command output.

The required handoff packet MUST include:

- work unit ID, phase, task IDs, exact task descriptions, and exact paths
- relevant spec sources and contract files
- non-negotiable project constraints derived from the loaded artifacts
- current implementation status from previously verified units
- allowed edit paths and forbidden paths
- tests or validation commands to run
- required final response fields: files changed, task IDs completed, tests run with results, expected failures, assumptions, and parent follow-up

The parent verification gate MUST include:

- inspect `git diff -- <unit paths>` plus any extra paths reported by the subagent
- confirm edits are inside unit scope or justified adjacent changes
- run targeted tests or closest compiling package test
- re-read relevant contracts or data-model sections for public boundary changes
- confirm forbidden generated, fixture, empirical, secret, or persistence paths remain unchanged when such restrictions are present
- mark task checkboxes only after the gate passes

The context compaction recovery procedure MUST include:

- read the orchestration process, ledger, and checklist before editing
- run `git status --short` and inspect existing diffs
- resume at the first ledger unit with unchecked referenced tasks unless an earlier partial diff exists
- reconstruct prior state from checked tasks, current diffs, and targeted tests
- finish and verify a partial unit before opening a new subagent

## Ledger Generation Rules

- Every executable task ID in the checklist MUST appear in exactly one work unit.
- No work unit may reference a task ID that is not present in the checklist.
- Preserve task checkbox states and descriptions. Do not mark tasks complete or reopen tasks.
- Use existing phase titles verbatim in the `Phase` column.
- In `Atomic scope and touched paths`, summarize the work and list exact backticked paths from the referenced task descriptions. If a task has no explicit path, write `No explicit path in task; parent must constrain before delegation.` and include that as a validation warning in the final report.
- In `Required handoff sources`, name only existing feature artifacts or contract files where possible. If an expected artifact is missing, say `tasks.md and available spec artifacts` rather than inventing paths.
- In `Parent verification`, use concrete commands from the task text, quickstart, Makefile, package conventions, or nearby existing tasks. If no command can be derived, require a diff inspection plus the closest compiling or linting gate discovered in the repository.
- If `$ARGUMENTS` contains `strict`, stop instead of writing when any task lacks explicit paths or any unit needs an inferred verification command.
- Mark parallel candidates in the `Prerequisites` cell only when units are independent by paths and dependencies and the parent can verify all results before any dependent unit starts.
- Parent-only final validation units may say so explicitly in `Atomic scope and touched paths`.

## Validation Before Writing

Before writing `tasks.md`, validate the generated content:

1. The edit target is exactly `FEATURE_DIR/tasks.md`.
2. `.specify/templates/tasks-template.md` and other templates are unchanged.
3. The orchestration process appears exactly once.
4. The work-unit ledger appears exactly once.
5. Every checklist task ID appears in exactly one ledger row.
6. Every ledger task ID exists in the checklist.
7. Work-unit prerequisites reference only earlier work units or `None`.
8. Test units precede implementation units for the same story or behavior.
9. Every work unit has a non-empty parent verification instruction.
10. Existing task checkboxes and task descriptions are unchanged.
11. Existing bugfix, drift, and reopen annotations are preserved.

If validation fails, do not write the file. Report the reason and the smallest safe manual correction needed.

## Report

After writing `tasks.md`, report:

- feature directory
- number of tasks parsed
- number of work units generated
- any warnings about tasks without explicit paths or inferred verification
- whether an existing orchestration section was inserted or refreshed
- next step: run `/speckit.orchestration-task-context-management.validate`, then `/speckit.implement`

## Rules

- Modify only the active feature `tasks.md`.
- Never edit `.specify/templates/`.
- Never regenerate tasks from scratch.
- Never add, delete, complete, or reopen implementation tasks.
- Preserve the current local Spec Kit task format instead of hard-coding the upstream template.
- Prefer minimal, idempotent edits.
- Stop rather than guessing when task IDs, phase boundaries, or feature directory resolution are ambiguous.