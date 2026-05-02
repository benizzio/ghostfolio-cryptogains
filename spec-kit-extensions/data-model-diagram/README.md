# Data Model Diagram Extension

Generate a raw Mermaid `erDiagram` file from the active Spec Kit `data-model.md` after `/speckit.plan` finishes.

## Behavior

- Provides the `speckit.data-model-diagram.generate` command
- Registers a mandatory `after_plan` hook
- Reads the active feature `data-model.md`
- Asks the agent to infer and overwrite `data-model-diagram.mmd`
- Preserves the data model's declared documentary field types instead of collapsing them to generic primitives
- Includes scalar fields and relationship-bearing typed attributes inside Mermaid entity blocks when the model supports them
- Preserves optionality and nullability using Mermaid-safe documentary types such as `string_nullable` or `timestamp_nullable`
- Keeps Mermaid relationship lines in addition to those typed attributes

## Development Install

```bash
specify extension add --dev spec-kit-extensions/data-model-diagram
```

## Output

The generated file contains raw Mermaid source only:

```text
erDiagram
    EntityA {
        UUID_string id
        decimal_string amount
        RelatedEntity_nullable related_entity
        ChildEntity_array child_entities
    }
    EntityA ||--o{ EntityB : contains
```
