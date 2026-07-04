# Spec Kit Orchestration Task Context Management

Spec Kit extension for agent task execution with bounded context, parent-orchestrator control, and clean-context subagent work units.

## Overview

This extension rewrites only the active feature `tasks.md` after `/speckit.tasks` has generated it. It does not modify `.specify/templates/` or any other Spec Kit template, so it remains independent of future Spec Kit template upgrades.

The generated orchestration section turns a normal task checklist into an implementation control plane for `/speckit.implement`:

- parent orchestrator rules
- required subagent handoff packet
- parent verification gate
- context compaction recovery procedure
- ordered work-unit ledger mapped to the existing task IDs
- optional subagent handoff examples

## Commands

- `/speckit.orchestration-task-context-management.prepare`: Analyze the active feature `tasks.md` and insert or refresh the context-orchestration work-unit ledger.
- `/speckit.orchestration-task-context-management.validate`: Read-only validation that the active feature `tasks.md` has a complete, current orchestration ledger before `/speckit.implement` starts.

## Hooks

The extension declares these hooks:

- `after_tasks`: optionally run `prepare` immediately after task generation, before a normal after-task commit hook.
- `before_implement`: always run `validate` before implementation starts.

## Installation

From a Spec Kit project, install the release archive:

```bash
specify extension add orchestration-task-context-management --from https://github.com/benizzio/spec-kit-orchestration-task-context-management/archive/refs/tags/v0.0.0.zip
```

For local development, install from a checkout:

```bash
specify extension add --dev /path/to/spec-kit-orchestration-task-context-management
```

## Usage

After generating tasks, run:

```text
/speckit.orchestration-task-context-management.prepare
```

Then start implementation normally:

```text
/speckit.implement
```

If the ledger is missing or stale, the pre-implement validation hook stops and tells you to refresh the task file with `prepare`.

See `docs/usage.md` for optional arguments and refresh guidance.

## Configuration

No configuration file is required. The extension derives its behavior from the active feature `tasks.md` and available Spec Kit artifacts in the feature directory.

## Safety Model

- Edits only `specs/<feature>/tasks.md`.
- Never edits `.specify/templates/tasks-template.md`.
- Preserves existing task IDs, descriptions, checkbox states, bugfix notes, and user-story phases.
- Replaces only generated orchestration sections when refreshing.
- Refuses to continue when task IDs are duplicated, placeholders remain, or the ledger cannot cover every task exactly once.

## Troubleshooting

- If validation reports a stale ledger, rerun `/speckit.orchestration-task-context-management.prepare`, then rerun `/speckit.orchestration-task-context-management.validate`.
- If `prepare` refuses to write, fix the reported task ID, placeholder, or ambiguous feature-directory issue first.
- If a subagent needs paths outside a ledger unit, stop and refresh the ledger instead of broadening scope manually.

## Support

File issues at `https://github.com/benizzio/spec-kit-orchestration-task-context-management/issues`.

## Version

Current version: `0.0.0`.
