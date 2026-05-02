# Data Model Diagram Extension

Generate a raw Mermaid `erDiagram` file from the active Spec Kit `data-model.md` after `/speckit.plan` finishes.

## Behavior

- Provides the `speckit.data-model-diagram.generate` command
- Registers a mandatory `after_plan` hook
- Reads the active feature `data-model.md`
- Asks the agent to infer and overwrite `data-model-diagram.mmd`

## Development Install

```bash
specify extension add --dev spec-kit-extensions/data-model-diagram
```

## Output

The generated file contains raw Mermaid source only:

```text
erDiagram
    EntityA ||--o{ EntityB : contains
```
