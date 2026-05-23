# Specification Quality Checklist: Generate Yearly Gains Report

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-05-19
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Validation completed on 2026-05-19 against `specs/005-generate-gains-report/spec.md`.
- No clarification markers remain.
- Revalidated after the `spec-fixes.md` updates: the workflow now centers on the unlocked `Sync and Reports` context, the report structure is defined explicitly, and single-activity currency-context rules are separated from exact-decimal rules.
- Revalidated on 2026-05-23 for BUG-006 artifact consistency: `data-model.md`, `contracts/tui-workflows.md`, `quickstart.md`, and this checklist now align on diagnostics outcome state, diagnostics-path disclosure, source-faithful persisted `ActivityRecord` use, and explicit `null` rendering.
- Runtime-backed verification for regenerated coverage and fresh artifact inspection remains tracked separately by `T062` and `T064`; the evidence below distinguishes current documented evidence from those pending reruns.

## Implementation Evidence

### OWASP Top 10 Evidence

- A02 Cryptographic Failures: protected synced activity history remains inside token-derived encrypted snapshots; generated report content stays in memory until the final Documents save and is not written to app-managed temp storage.
- A07 Identification and Authentication Failures: the Ghostfolio token is entered through the masked `Sync and Reports` unlock flow, reused only while that context remains active, and cleared when the user leaves that context.
- A04 Insecure Design: the report workflow keeps the protected-storage boundary separate from Markdown rendering and cleartext Documents output; the application keeps no report history or reopen catalog.
- A05 Security Misconfiguration: report generation fails when the Documents location cannot be resolved or used safely instead of silently falling back to app-data, current-working-directory, or temp locations.
- A08 Software and Data Integrity Failures: successful report output uses one exclusive-create final path with same-second suffix reservation and removes any partial file when the write fails.
- A06 Vulnerable and Outdated Components: the feature was implemented without introducing new runtime third-party dependencies; it reuses the repository's existing Go, Bubble Tea, `apd`, and `x/crypto` dependency set.
- A09 Security Logging and Monitoring Failures: report result and failure messages are constrained to non-secret references and must not expose tokens, JWTs, raw payloads, or cleartext report previews before save.

### Cryptographic-Storage Boundary Evidence

- `setup.json` remains bootstrap-only and does not store synced activity data, report content, generated-report paths, or report history.
- Protected snapshots remain the only persisted store for synced activity history before reporting.
- Report generation reads from the unlocked protected snapshot and writes one final cleartext Markdown file only to the user's Documents folder.
- User removal guidance is explicit: delete the saved Markdown file from Documents to remove cleartext report output.
- The application clears transient in-memory report state on result dismissal and context exit, so saved paths and rendered report content are not retained as application history.

### Cleartext Report Output Evidence

- Cleartext report output is intentional only after a successful final save to Documents.
- Failed output attempts remove the partial file created during that attempt.
- Automatic-open failure after save is non-fatal and keeps the saved file in place.
- App-managed storage is expected to contain no Markdown report content or generated-report catalog. Eligible failed sync or report attempts may still write separate `.diagnostic.json` artifacts under the application-owned diagnostics directory.
- Verified on 2026-05-21 for successful-report leakage paths through runtime-backed integration coverage and artifact inspection: `tests/integration/report_generation_flow_test.go`, `tests/integration/report_failure_flow_test.go`, `tests/integration/report_cost_basis_methods_flow_test.go`, and `tests/integration/report_performance_flow_test.go` all call `assertNoCleartextReportInAppStorage(t, harness.BaseDir)`, which walks plaintext application artifacts under `<baseDir>/ghostfolio-cryptogains/` and fails on any `.md` file or persisted report header marker. Workspace inspection after that verification run also found no `ghostfolio-capital-gains-*.md` files under the repository worktree.
- BUG-006 extends this review to diagnostics artifacts. Fresh runtime-backed verification that generated `.diagnostic.json` files stay outside Documents, disclose only the intended path, and do not reintroduce cleartext Markdown copies remains tracked by `T064`.

### Report-Failure Diagnostics Artifact Review Evidence

- `spec.md` now requires eligible pre-save report failures to support a separate local diagnostics artifact, distinguishes that flow from opener-only warnings, and requires path disclosure under `FR-034b` through `FR-035c`.
- `data-model.md` now models transient diagnostics outcome state, diagnostics path, failure category, and original persisted offending-record context through `ReportFailureDiagnostics` and `ReportGenerationOutcome`.
- `contracts/tui-workflows.md` now states that the failure-result screen offers `Generate Diagnostic Report` outside explicit development mode, generates diagnostics automatically in explicit development mode, and shows the written diagnostics path when generation succeeds.
- `quickstart.md` now includes manual verification steps for production-mode prompting, explicit-development automatic generation, and diagnostics-artifact location checks.

### Source-Faithful Persisted Activity Evidence

- `spec.md` requires activity-specific diagnostics to serialize the original persisted `ActivityRecord`, not selected activity-calculation inputs, rendered report rows, or other derived report values.
- `data-model.md` now defines `offending_activity_record` on `ReportFailureDiagnostics` and constrains it to the original persisted record only.
- `quickstart.md` now requires manual inspection that activity-specific diagnostics use the original persisted `ActivityRecord` shape.

### Explicit `null` Rendering Evidence

- `spec.md` requires absent source fields in report-failure and synced-data diagnostics to render as explicit `null` values.
- `data-model.md` now records that nullable source fields in serialized offending-record output must render as explicit `null` values.
- `quickstart.md` now includes manual inspection guidance for explicit `null` rendering in generated diagnostics artifacts.
- Full rerun evidence for synced-data and report-failure explicit-`null` regression paths remains tracked by `T062`.

### Dependency And API Review Evidence

- No new Ghostfolio API endpoints were added for this slice; report generation operates on the synced protected cache and `Sync Data` continues using the existing anonymous auth and paged activities endpoints.
- No new runtime third-party dependencies were added for report generation, Markdown rendering, Documents resolution, or OS open handling.
- Local OS integration remains limited to standard-library filesystem access and platform opener commands: `xdg-open` on Linux, `open` on macOS, and `cmd /c start "" <path>` on Windows.

### Evidence Sources

- `specs/005-generate-gains-report/spec.md`
- `specs/005-generate-gains-report/plan.md`
- `specs/005-generate-gains-report/research.md`
- `specs/005-generate-gains-report/data-model.md`
- `specs/005-generate-gains-report/contracts/markdown-report.md`
- `specs/005-generate-gains-report/contracts/tui-workflows.md`
- `specs/005-generate-gains-report/quickstart.md`
- `internal/app/runtime/report_service.go`
- `internal/report/output/documents.go`
- `internal/report/output/writer.go`
- `internal/report/output/opener.go`
- `internal/report/markdown/renderer.go`
- `tests/integration/helpers_test.go`
- `tests/integration/persistence_security_flow_test.go`
- `go.mod`
