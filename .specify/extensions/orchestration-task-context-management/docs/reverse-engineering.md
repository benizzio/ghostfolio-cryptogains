# Reverse-Engineered Task Orchestration Pattern

This document records the pattern derived from the experimented task file at `specs/007-currency-conversion-strategy/tasks.md` and the local Spec Kit `tasks-template.md`.

## Original Spec Kit Task Shape

The stock task template produces a feature-local `tasks.md` with:

- a preamble describing inputs, prerequisites, tests, organization, and task format
- phase-based task groups
- user-story test and implementation subsections
- dependencies and execution order
- parallel examples
- implementation strategy
- notes

The template intentionally keeps task execution generic. It does not define a parent orchestrator, clean-context subagent handoffs, or a durable recovery model for context compaction.

## Experimented Orchestration Shape

The experimented file preserved the normal task checklist and added an execution control plane before Phase 1:

- `## Context Orchestration Process`
- `### Parent Orchestrator Rules`
- `### Required Subagent Handoff Packet`
- `### Parent Verification Gate`
- `### Context Compaction Recovery`
- `### Work Unit Ledger`

The ledger table mapped existing task IDs to ordered `WU##` rows. Each row described phase, tasks, atomic scope and paths, prerequisites, required handoff sources, and parent verification.

The experimented file also reframed later sections:

- task-level `[P]` markers became local hints rather than the primary execution plan
- parallelism was allowed only when ledger prerequisites were satisfied and the parent could verify all results before dependent units
- subagent handoff examples showed complete prompts for bounded units
- implementation strategy clarified that the parent owns ordering, review, and final validation

## Generalized Extension Behavior

The extension generalizes that pattern without changing Spec Kit templates:

- locate the active feature `tasks.md`
- parse the existing task checklist and phase structure
- derive work units from current task IDs, paths, stories, and phase semantics
- insert or refresh only the generated orchestration sections
- keep all original tasks, checkbox states, bugfix notes, and user-story phases intact
- validate that every task is covered exactly once by the ledger

## Non-Goals

- It does not generate implementation tasks from specs.
- It does not edit `spec.md`, `plan.md`, `research.md`, `data-model.md`, contracts, or templates.
- It does not mark tasks complete.
- It does not replace `/speckit.implement`; it prepares the task file so the implement command has an explicit orchestration protocol.
