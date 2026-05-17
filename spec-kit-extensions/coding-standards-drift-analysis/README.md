# Coding Standards Drift Analysis Extension

Beta Spec Kit extension for reviewing an implemented feature against the repository coding-standards baseline and then adding remediation tasks to the feature task list.

## Behavior

- Provides the `speckit.coding-standards-drift-analysis.report` command
- Provides the `speckit.coding-standards-drift-analysis.remediation-plan` command
- Registers an optional `after_implement` hook for report generation
- Uses `AGENTS.md`, `.specify/memory/constitution.md`, and any other known proprietary agent-instruction files present in repository or feature scope as the repository-policy baseline
- Discovers known proprietary agent-instruction files such as `CLAUDE.md`, `GEMINI.md`, `copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, and `.clinerules` when they are present
- Focuses on coding standards and engineering practices, not feature-domain correctness
- Blocks report generation and remediation planning while the active feature has open or pending tasks
- Preserves existing `DRIFT-###` identifiers when the same findings remain on rerun
- Appends remediation tasks to `tasks.md` using the current local Spec Kit task format so `/speckit.implement` can execute them

## Workflow

1. Run `/speckit.coding-standards-drift-analysis.report` after implementation to generate or refresh `code-standard-drift-report.md`.
2. Run `/speckit.coding-standards-drift-analysis.remediation-plan` to append a drift remediation phase to the active feature `tasks.md`.
3. Run `/speckit.implement` to execute the generated remediation tasks and mark them complete as fixes land.

## Development Install

```bash
specify extension add --dev spec-kit-extensions/coding-standards-drift-analysis
```

## Output

- `specs/{feature}/code-standard-drift-report.md`
- `specs/{feature}/tasks.md` with an appended drift remediation phase
