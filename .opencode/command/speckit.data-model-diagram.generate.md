---
description: "Generate data-model-diagram.mmd from data-model.md via agent inference"
---

# Generate Data Model Diagram

Generate a Mermaid ER diagram for the active Spec Kit feature directory.

## Execution

1. Run `.specify/scripts/bash/check-prerequisites.sh --json --paths-only` from the repository root and parse the absolute `FEATURE_DIR`.
2. Read `${FEATURE_DIR}/data-model.md`. If it does not exist, stop and surface the error.
3. Infer the Mermaid `erDiagram` structure from the data model contents. Use the document's entities, fields, and relationships as the source of truth. Resolve gaps conservatively instead of inventing unsupported details.
4. Write raw Mermaid source only to `${FEATURE_DIR}/data-model-diagram.mmd`, overwriting any existing file.
5. Report the absolute input and output paths.
