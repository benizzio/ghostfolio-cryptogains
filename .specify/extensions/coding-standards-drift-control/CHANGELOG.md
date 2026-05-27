# Changelog

## 0.3.0

- Rename the extension ID and command namespace from `coding-standards-drift-analysis` to `coding-standards-drift-control`
- Prepare the extension for standalone publishing in `benizzio/spec-kit-coding-standards-drift-control`
- Replace `DRIFT-###` finding identifiers with `CODE-STAND-DRIFT-###` and migrate legacy IDs by preserving numeric suffixes
- Keep beta versioning and omit backward-compatible aliases for the previous namespace

## 0.2.0

- Rename the checklist command to `remediation-plan`
- Append drift remediation tasks to the active feature `tasks.md` instead of creating a separate checklist
- Block report generation and remediation planning while the active feature has open or pending tasks
- Remove the dedicated `remediate` command because remediation is executed through `/speckit.implement`

## 0.1.0

- Add the initial beta Spec Kit extension
- Generate or refresh `coding-standards-drift-report.md` for the active feature implementation
- Generate additive remediation checklists from drift findings
- Execute unresolved remediation items and update the checklist as work completes
- Register an optional `after_implement` hook for report generation
