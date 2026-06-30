# Changelog

## Unreleased

## 0.4.0

- Make report reruns incremental by preserving prior findings with `Pending`/`Resolved` status
- Limit remediation task planning to pending drift findings
- Add a final generated remediation task that marks successfully remediated findings as `Resolved`
- Add per-finding remediation planning before task generation so each pending drift item records a surgical implementation plan grounded in report evidence and coding standards

## 0.3.1

- Fix review feedback from the source extraction PR by correcting README wording
- Align the optional hook prompt with the drift-control naming

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
