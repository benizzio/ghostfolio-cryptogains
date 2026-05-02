# Data Model Diagram Extension

Generate a raw Mermaid `erDiagram` file from the active Spec Kit `data-model.md` after `/speckit.plan` finishes.

## Behavior

- Provides the `speckit.data-model-diagram.generate` command
- Registers a mandatory `after_plan` hook
- Resolves the active feature directory with Spec Kit's shared shell helpers
- Reads `data-model.md` and overwrites `data-model-diagram.mmd` deterministically

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
