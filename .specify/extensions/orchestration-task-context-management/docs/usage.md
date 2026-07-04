# Usage

## Typical Flow

1. Run `/speckit.tasks` for a feature.
2. Accept the `after_tasks` hook or run `/speckit.orchestration-task-context-management.prepare` manually.
3. Review the generated `## Context Orchestration Process` section in `tasks.md`.
4. Run `/speckit.orchestration-task-context-management.validate`.
5. Run `/speckit.implement`.

## Manual Refresh

Run the prepare command again after any task-list change:

```text
/speckit.orchestration-task-context-management.prepare refresh
```

The command refreshes the generated orchestration sections and keeps task checkbox states unchanged.

## Validation Failure

If validation fails before implementation, refresh the task file:

```text
/speckit.orchestration-task-context-management.prepare
/speckit.orchestration-task-context-management.validate
```

Do not start `/speckit.implement` until validation passes.

## Optional Arguments

- `refresh`: explicitly refresh an existing ledger.
- `strict`: stop on any inferred path or verification command.
- `max-unit-tasks=6`: guide the maximum number of tasks grouped into a normal work unit.
- `no-examples`: skip generation of subagent handoff examples.
