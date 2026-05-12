# Coding Standards Drift Analysis Extension

Beta Spec Kit extension for reviewing an implemented feature against the repository coding-standards baseline and then tracking and executing remediation.

## Behavior

- Provides the `speckit.coding-standards-drift-analysis.report` command
- Provides the `speckit.coding-standards-drift-analysis.checklist` command
- Provides the `speckit.coding-standards-drift-analysis.remediate` command
- Registers an optional `after_implement` hook for report generation
- Uses `AGENTS.md`, `.specify/memory/constitution.md`, and any other known proprietary agent-instruction files present in repository or feature scope as the repository-policy baseline
- Discovers known proprietary agent-instruction files such as `CLAUDE.md`, `GEMINI.md`, `copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, and `.clinerules` when they are present
- Focuses on coding standards and engineering practices, not feature-domain correctness
- Preserves existing `DRIFT-###` identifiers when the same findings remain on rerun
- Keeps remediation checklist generation additive so reruns only add missing work items

## Workflow

1. Run `/speckit.coding-standards-drift-analysis.report` after implementation to generate or refresh `code-standard-drift-report.md`.
2. Run `/speckit.coding-standards-drift-analysis.checklist` to create or extend `checklists/code-standard-drift-remediation.md`.
3. Run `/speckit.coding-standards-drift-analysis.remediate` to execute unresolved checklist items and mark them complete as fixes land.

## Development Install

```bash
specify extension add --dev spec-kit-extensions/coding-standards-drift-analysis
```

## Output

- `specs/{feature}/code-standard-drift-report.md`
- `specs/{feature}/checklists/code-standard-drift-remediation.md`
