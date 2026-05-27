# Coding Standards Drift Control

Beta Spec Kit extension for reviewing implemented features against the repository coding-standards baseline and appending remediation tasks for drift findings.
Track and fix code that moves away from technical definitions adding the smallest amout of complexity possible to the Spec Kit artifacts.

## Features

- Provides the `/speckit.coding-standards-drift-control.report` command
- Provides the `/speckit.coding-standards-drift-control.remediation-plan` command
- Registers an optional `after_implement` hook for report generation
- Uses `AGENTS.md`, `.specify/memory/constitution.md`, and other known agent-instruction files as the review baseline when present
- Focuses on coding standards and engineering practices rather than feature-domain correctness
- Blocks report generation and remediation planning while the active feature has open or pending tasks
- Uses `CODE-STAND-DRIFT-###` finding identifiers and migrates legacy `DRIFT-###` findings to the new format on rerun
- Appends remediation tasks to `tasks.md` so `/speckit.implement` can execute them

## Requirements

- Spec Kit `>=0.2.0`
- A Spec Kit project with `.specify/`
- An active feature with `spec.md`, `plan.md`, and `tasks.md`

## Installation

### Release Install

```bash
specify extension add coding-standards-drift-control --from https://github.com/benizzio/spec-kit-coding-standards-drift-control/archive/refs/tags/v0.3.0.zip
```

### Development Install

```bash
specify extension add --dev /path/to/spec-kit-coding-standards-drift-control
```

## Workflow

1. Finish the active feature's normal implementation tasks.
2. Run `/speckit.coding-standards-drift-control.report` to generate or refresh `coding-standards-drift-report.md`.
3. Run `/speckit.coding-standards-drift-control.remediation-plan` to append a drift remediation phase to `tasks.md`.
4. Run `/speckit.implement` to execute the generated remediation tasks.

## Commands

### `/speckit.coding-standards-drift-control.report`

Generates or refreshes `specs/{feature}/coding-standards-drift-report.md` for the active feature after implementation is complete.

Example:

```text
/speckit.coding-standards-drift-control.report focus on parser and validation files
```

### `/speckit.coding-standards-drift-control.remediation-plan`

Appends a final remediation phase to `specs/{feature}/tasks.md` from the current drift report.

Example:

```text
/speckit.coding-standards-drift-control.remediation-plan only high severity drift items
```

## Behavior

- Reviews the active feature implementation against repository policy files and local engineering standards
- Loads `AGENTS.md`, `.specify/memory/constitution.md`, and known agent-instruction files such as `CLAUDE.md`, `GEMINI.md`, `.github/copilot-instructions.md`, `.cursorrules`, `.cursor/rules/**`, `.windsurfrules`, and `.clinerules` when present
- Falls back to a conservative baseline derived from the local codebase when those files do not define concrete standards
- Keeps findings grounded in exact file references and explicit policy evidence
- Skips remediation task generation when the report has no findings or when a matching `CODE-STAND-DRIFT-###` task, or a legacy `DRIFT-###` task for the same finding, already exists in `tasks.md`

## Configuration

No additional extension configuration file is required. The commands inspect the active Spec Kit feature artifacts and repository policy files at runtime.

## Output

- `specs/{feature}/coding-standards-drift-report.md`
- `specs/{feature}/tasks.md` with an appended drift remediation phase

## Troubleshooting

### Report Generation Stops Immediately

Complete the remaining open or pending tasks in the active feature's `tasks.md`, then rerun the report command.

### No Remediation Tasks Were Added

The report may contain no findings, or matching `CODE-STAND-DRIFT-###` tasks, or legacy `DRIFT-###` tasks for the same findings, may already exist in `tasks.md`.

### `specify extension` Commands Are Missing

Install the Spec Kit CLI from the GitHub repository rather than the unrelated PyPI stub package:

```bash
uv tool install specify-cli --force --from git+https://github.com/github/spec-kit.git
```

## Release Notes

- Repository: `https://github.com/benizzio/spec-kit-coding-standards-drift-control`
- Documentation: `README.md`
- Changelog: `CHANGELOG.md`
- License: `MIT`
- Current beta version: `0.3.0`

## Catalog Submission Metadata

- Extension ID: `coding-standards-drift-control`
- Extension Name: `Coding Standards Drift Control`
- Version: `0.3.0`
- Commands: `2`
- Hooks: `1`
- Suggested tags: `analysis`, `standards`, `quality`, `maintenance`

## License

MIT
