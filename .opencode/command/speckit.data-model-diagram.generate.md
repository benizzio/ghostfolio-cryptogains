---
description: Generate data-model-diagram.mmd from data-model.md
---


<!-- Extension: data-model-diagram -->
<!-- Config: .specify/extensions/data-model-diagram/ -->
# Generate Data Model Diagram

Generate a Mermaid ER diagram for the active Spec Kit feature directory.

## Execution

1. Run `bash .specify/extensions/data-model-diagram/scripts/bash/generate-data-model-diagram.sh` from the repository root.
2. If the script exits with a non-zero status, stop and surface the error.
3. On success, report the resolved `data-model.md` input path and `data-model-diagram.mmd` output path.
