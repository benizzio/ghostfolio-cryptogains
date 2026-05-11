# Code Standard Drift Analysis Extension

Beta Spec Kit extension for reviewing an implemented feature against the repository coding-standards baseline and then tracking and executing remediation.

## Behavior

- Provides the `speckit.code-standard-drift-analysis.report` command
- Provides the `speckit.code-standard-drift-analysis.checklist` command
- Provides the `speckit.code-standard-drift-analysis.remediate` command
- Registers an optional `after_implement` hook for report generation
- Uses `AGENTS.md` and `.specify/memory/constitution.md` as the repository-policy baseline
- Focuses on coding standards and engineering practices, not feature-domain correctness
- Preserves existing `DRIFT-###` identifiers when the same findings remain on rerun
- Keeps remediation checklist generation additive so reruns only add missing work items

## Workflow

1. Run `/speckit.code-standard-drift-analysis.report` after implementation to generate or refresh `code-standard-drift-report.md`.
2. Run `/speckit.code-standard-drift-analysis.checklist` to create or extend `checklists/code-standard-drift-remediation.md`.
3. Run `/speckit.code-standard-drift-analysis.remediate` to execute unresolved checklist items and mark them complete as fixes land.

## Development Install

```bash
specify extension add --dev spec-kit-extensions/code-standard-drift-analysis
```

## Output

- `specs/{feature}/code-standard-drift-report.md`
- `specs/{feature}/checklists/code-standard-drift-remediation.md`
