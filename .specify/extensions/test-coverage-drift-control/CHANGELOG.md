# Changelog

## [0.2.1] - 2026-05-31

- Remove source command metadata markers so installed opencode command wrappers do not duplicate extension/config comments
- Document the `COV-DRIFT-###` remediation identifier convention for release consistency

## [0.2.0] - 2026-05-27

- Extract the extension into the standalone `spec-kit-test-coverage-drift-control` repository
- Rename the extension ID and command namespace from `test-coverage-drift-analysis` to `test-coverage-drift-control`
- Update the manifest metadata, repository links, and README for standalone publishing
- Preserve the existing report, remediation, and optional `after_implement` workflow
- Use stable `COV-DRIFT-###` identifiers, with the `COV-DRIFT-` prefix plus a numeric sequence, for test-coverage-drift-control reporting and remediation workflows

## [0.1.0]

- Add the initial beta Spec Kit extension
- Generate or refresh `test-coverage-drift-report.md` for the active feature implementation
- Compare implemented test coverage against repository coverage targets and test-structure definitions
- Append coverage remediation tasks to the active feature `tasks.md`
- Register an optional `after_implement` hook for report generation
