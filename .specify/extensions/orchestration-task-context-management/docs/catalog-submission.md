# Catalog Submission Metadata

Use these values for the Spec Kit Extension Submission issue after publishing GitHub release `v0.0.0`.

| Field | Value |
|-------|-------|
| Extension ID | `orchestration-task-context-management` |
| Name | `Orchestration Task Context Management` |
| Version | `0.0.0` |
| Description | `Adds subagent work-unit orchestration to generated Spec Kit task files` |
| Author | `Igor Benicio de Mesquita` |
| License | `MIT` |
| Repository | `https://github.com/benizzio/spec-kit-orchestration-task-context-management` |
| Homepage | `https://github.com/benizzio/spec-kit-orchestration-task-context-management` |
| Documentation | `https://github.com/benizzio/spec-kit-orchestration-task-context-management/blob/main/README.md` |
| Changelog | `https://github.com/benizzio/spec-kit-orchestration-task-context-management/blob/main/CHANGELOG.md` |
| Download URL | `https://github.com/benizzio/spec-kit-orchestration-task-context-management/archive/refs/tags/v0.0.0.zip` |
| Required Spec Kit version | `>=0.7.2` |
| Commands | `2` |
| Hooks | `2` |
| Tags | `agent`, `orchestration`, `tasks`, `context` |

## Key Features

- Inserts or refreshes a work-unit orchestration ledger in generated feature `tasks.md` files.
- Adds parent-orchestrator rules, subagent handoff requirements, verification gates, and context compaction recovery guidance.
- Provides read-only validation before `/speckit.implement` starts.
- Keeps Spec Kit templates untouched and edits only the active feature task file.

## Testing Confirmation

- Manifest and command file paths were statically validated.
- Local development install was tested with `specify extension add --dev` in a disposable Spec Kit project.
- Installed extension listing confirmed 2 commands and 2 hooks.
