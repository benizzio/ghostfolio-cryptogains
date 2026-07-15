# Test Coverage Drift Control

Spec Kit extension for reviewing implemented features against a repository test-coverage baseline and turning concrete drift findings into executable remediation tasks.
It helps keep feature-level test coverage aligned with repository policy while adding the smallest practical amount of extra Spec Kit process.

## Features

- Provides the `speckit.test-coverage-drift-control.report` command
- Provides the `speckit.test-coverage-drift-control.remediation-plan` command
- Registers an optional `after_implement` hook for report generation
- Uses `.specify/memory/constitution.md`, `AGENTS.md`, and other known agent-instruction files present in repository or feature scope as the baseline when available
- Falls back to derived coverage expectations from repository coverage commands, CI workflows, tests, and feature artifacts when explicit baseline rules are incomplete
- Blocks report generation and remediation planning while the active feature still has open implementation tasks
- Preserves historical `COV-DRIFT-###` findings with stable identifiers and `Pending`/`Resolved` status across incremental reruns
- Records evidence-based remediation plans in pending report findings before appending tasks to `tasks.md`
- Adds a final remediation task that marks only successfully remediated and validated findings as `Resolved`

## Requirements

- Spec Kit `>=0.2.0`
- An initialized Spec Kit project with `.specify/`
- Feature artifacts generated through the normal Spec Kit flow, including `spec.md`, `plan.md`, and `tasks.md`

## Installation

### From a release archive

```bash
specify extension add test-coverage-drift-control --from https://github.com/benizzio/spec-kit-test-coverage-drift-control/archive/refs/tags/v0.3.0.zip
```

### Development install

```bash
specify extension add --dev /path/to/spec-kit-test-coverage-drift-control
```

## Configuration

This extension does not require a custom config file.

At runtime it reads the active feature artifacts plus repository coverage-policy sources such as `.specify/memory/constitution.md`, `AGENTS.md`, and other recognized agent-instruction files when they exist.

## Commands

### `/speckit.test-coverage-drift-control.report`

Generate or incrementally refresh `specs/{feature}/test-coverage-drift-report.md` for the active feature after the normal implementation tasks are complete.

### `/speckit.test-coverage-drift-control.remediation-plan`

Add or update evidence-based remediation plans for pending findings in `specs/{feature}/test-coverage-drift-report.md`, then append a final coverage drift remediation phase to `specs/{feature}/tasks.md`.

## Example Usage

```text
/speckit.implement
/speckit.test-coverage-drift-control.report
/speckit.test-coverage-drift-control.remediation-plan
/speckit.implement
```

## Hook

The extension registers an optional `after_implement` hook that runs `speckit.test-coverage-drift-control.report`.

## Workflow

1. Finish the feature's normal `/speckit.implement` tasks.
2. Run `/speckit.test-coverage-drift-control.report` to generate or incrementally refresh `test-coverage-drift-report.md`.
3. Run `/speckit.test-coverage-drift-control.remediation-plan` to record focused plans for pending findings and append a remediation phase to `tasks.md`.
4. Execute the generated remediation, validation, and final status-update tasks with `/speckit.implement`.
5. Re-run the report after later implementation changes to append new findings or reopen recurring drift without losing history.

## Outputs

- `specs/{feature}/test-coverage-drift-report.md` with finding status, optional remediation plans, and resolution notes
- `specs/{feature}/tasks.md` with an appended coverage drift remediation phase

## Troubleshooting

- If report generation stops immediately, finish all existing implementation tasks in the active feature before rerunning the command.
- If remediation planning stops, ensure `test-coverage-drift-report.md` already exists for the active feature.
- If no remediation tasks are added, the report has no pending findings matching the requested ID or severity scope.
- Findings from older reports without an explicit status are treated as `Pending`; report reruns never infer that an unobserved finding is resolved.
- If the extension cannot determine the active feature, verify the project was initialized with Spec Kit and that `.specify/scripts/bash/check-prerequisites.sh` is available.

## Testing

- Verified with local development installation using `specify extension add --dev`
- Verified with Spec Kit `0.8.14`
- Verified that the installed extension registers `2` commands and `1` optional hook

## Publishing Checklist

- Keep `extension.yml` versioned with semantic versions
- Tag releases as `vX.Y.Z`
- Verify install from the GitHub archive URL before submitting to the Spec Kit community catalog
- Update `CHANGELOG.md` for every release

## Release Process

```bash
git tag v0.3.0
git push origin v0.3.0
```

Then create the GitHub release and verify install from:

```text
https://github.com/benizzio/spec-kit-test-coverage-drift-control/archive/refs/tags/v0.3.0.zip
```

## Support And Contributing

- Use GitHub issues in this repository for bugs or feature requests
- Keep changes aligned with the extension manifest, command namespace, and release version history in `CHANGELOG.md`

## License

MIT
