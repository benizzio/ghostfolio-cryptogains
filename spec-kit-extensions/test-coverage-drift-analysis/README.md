# Test Coverage Drift Analysis Extension

Beta Spec Kit extension for reviewing an implemented feature against the repository test-coverage baseline and then adding remediation tasks to the feature task list.

## Behavior

- Provides the `speckit.test-coverage-drift-analysis.report` command
- Provides the `speckit.test-coverage-drift-analysis.remediation-plan` command
- Registers an optional `after_implement` hook for report generation
- Uses `.specify/memory/constitution.md`, `AGENTS.md`, and any other known proprietary agent-instruction files present in repository or feature scope as the coverage-policy baseline
- Scans the same coverage definition reference files as the report baseline: constitution and instruction files
- Focuses on coverage target, coverage gate instrumentation, required test structure, and the expected balance between integration and unit tests
- Blocks report generation and remediation planning while the active feature has open or pending tasks
- Preserves existing `COV-DRIFT-###` identifiers when the same findings remain on rerun
- Appends remediation tasks to `tasks.md` using the current local Spec Kit task format so `/speckit.implement` can execute them

## Workflow

1. Run `/speckit.test-coverage-drift-analysis.report` after implementation to generate or refresh `test-coverage-drift-report.md`.
2. Run `/speckit.test-coverage-drift-analysis.remediation-plan` to append a coverage drift remediation phase to the active feature `tasks.md`.
3. Run `/speckit.implement` to execute the generated remediation tasks and mark them complete as fixes land.

## Development Install

```bash
specify extension add --dev spec-kit-extensions/test-coverage-drift-analysis
```

## Output

- `specs/{feature}/test-coverage-drift-report.md`
- `specs/{feature}/tasks.md` with an appended coverage drift remediation phase
