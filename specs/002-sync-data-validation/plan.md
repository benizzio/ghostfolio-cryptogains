# Implementation Plan: Sync Data Validation

**Branch**: `[002-sync-data-validation]` | **Date**: 2026-05-09 | **Spec**: `/specs/002-sync-data-validation/spec.md`
**Input**: Feature specification from `/specs/002-sync-data-validation/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build the first runnable Go terminal application slice for this repository. The application launches into a full-screen Bubble Tea interface, guides first-run setup for Ghostfolio server selection, persists only startup-readable machine-local setup state, presents `Sync Data` as the only business workflow, prompts for a Ghostfolio security token only when that workflow starts, validates Ghostfolio communication through the minimal authenticated activities contract, and reports success or failure without persisting returned Ghostfolio data or exposing any reporting features. The slice also treats its probe-specific validation models as transitional scaffolding that must be removed or evolved when later specs introduce real sync normalization and protected persistence.

## Technical Context

**Language/Version**: Go 1.26.2  
**Primary Dependencies**: `charm.land/bubbletea/v2`, selected `charm.land/bubbles/v2` components (`list`, `textinput`, `help`, `key`, `spinner`), Go standard library (`net/http`, `encoding/json`, `context`, `net/url`, `os`, `path/filepath`)  
**Storage**: Local-only machine-scoped JSON setup file in the OS config or app-data directory, written atomically with restrictive filesystem permissions; no Ghostfolio token, JWT, or activity payload persistence in this slice  
**Testing**: `go test` with `httptest.Server` integration suites for first-run setup, no-pre-sync-network startup, delayed-request busy states, terminal resize responsiveness, and Ghostfolio validation flows; table-driven unit tests for validators, setup-file protection, and focus routing; `go test -coverprofile`, branch and file coverage gate via `github.com/Fabianexe/gocoverageplus`  
**Target Platform**: Installed terminal application for Linux, macOS, and Windows terminals with local filesystem access  
**Project Type**: Single-module Go TUI application  
**Performance Goals**: Render the first actionable setup or main-menu screen using only local bootstrap state before any Ghostfolio network request can occur, continue to process busy-state updates and terminal resize messages while auth or activities requests are in flight, and complete the one-page communication probe without blocking the Bubble Tea event loop  
**Constraints**: Ghostfolio token is runtime-only and cleared after each attempt; no Ghostfolio payload persistence; production-like custom origins require HTTPS and only explicit development mode may allow HTTP; development mode is entered only through an explicit startup flag; the TUI always owns the full terminal screen; primary next-step actions are presented as arrow-key menus; optional side actions are exposed only as clearly labeled hotkeys; labeled text inputs suppress conflicting hotkeys while they are focused; report generation, PDF output, financial calculations, and multi-user token-derived storage are out of scope  
**Scale/Scope**: One application-level setup profile per local OS user profile, one selected Ghostfolio origin, one executable business workflow (`Sync Data`), one anonymous-auth request plus one `take=1` activities probe per validation attempt, and successful empty activity lists are accepted when the response contract is otherwise valid

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Pre-research gate status: PASS  
Post-design gate status: PASS

- [x] Security: This slice persists only startup-readable bootstrap configuration that contains neither financial information nor person- or user-linkable data. It remains local-only, stores no Ghostfolio token, JWT, activity payload, or user identity, revalidates the configured origin on every read, and documents proportionate machine-local protection instead of Ghostfolio-token-derived encryption. The OWASP Top 10 review scope covers identification and authentication failures, security misconfiguration, software and data integrity failures, and logging leakage. Token-derived encryption and OWASP Cryptographic Storage Cheat Sheet evidence remain deferred until the product starts persisting financial or person-linked data.
- [x] Precision: Financial calculations are out of scope in this slice. Numeric values received from Ghostfolio are used only to validate JSON structure and are not used to derive balances, gains, losses, or reports.
- [x] Testing: Integration-first automated tests cover first-run setup gating, setup persistence, origin validation, full-screen workflow transitions, successful and failed communication validation, retry-after-failure behavior, token non-persistence, token non-exposure, and confirmation that no data persistence or report flow occurs. Unit tests isolate origin canonicalization, response-shape validation, and focus-aware key routing. Statement and branch or file coverage remain explicit release gates.
- [x] Dependencies: Only Bubble Tea, selected Bubbles widgets, and the existing development-time coverage helper are planned. `research.md` records the due diligence and keeps cryptographic storage, decimal math, and PDF dependencies out of this slice.
- [x] External APIs: Ghostfolio `api/v1` anonymous auth and activities endpoints are necessary for the product goal. The plan documents the minimal auth and activities probe contract, runtime compatibility validation, failure modes, and origin security rules, and it explicitly limits success to communication validation rather than later reporting compatibility.
- [x] Architecture: The design keeps setup storage, Ghostfolio transport, workflow state, and TUI rendering separate so that UI behavior is testable without network calls and sync-validation rules remain independent from terminal code.

## Project Structure

### Documentation (this feature)

```text
specs/002-sync-data-validation/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── ghostfolio-sync-validation.md
│   └── tui-workflows.md
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
└── ghostfolio-cryptogains/
    └── main.go

internal/
├── app/
│   ├── bootstrap/
│   └── runtime/
├── config/
│   ├── model/
│   └── store/
├── ghostfolio/
│   ├── client/
│   ├── dto/
│   └── validator/
├── tui/
│   ├── component/
│   ├── flow/
│   └── screen/
└── support/
    └── redact/

tests/
├── contract/
├── integration/
└── unit/
```

**Structure Decision**: Use a single Go module rooted at the repository root. `internal/config` owns startup-readable setup persistence, `internal/ghostfolio` owns auth and activities contract validation, `internal/tui` owns full-screen Bubble Tea screens and key routing, and `internal/app` assembles the runtime so the UI flow remains separate from transport and storage concerns.

## Full-Screen TUI Rules

- Launch the root Bubble Tea program with `tea.WithAltScreen()` so the application owns the entire terminal immediately.
- Every screen uses a stable full-screen layout with clearly delimited regions for title and explanation, main workflow content, transient status, and visible hotkeys.
- Use Ghostfolio's general visual identity as the TUI design reference: Inter-style clean sans typography, primary teal accents around `#36cfcc`, secondary blue accents around `#3686cf`, warning and error red around `#dc3545`, and restrained light or dark neutral surfaces rather than saturated backgrounds.
- Adapt that palette to terminal capabilities by using truecolor when available and the nearest readable ANSI fallbacks when it is not. Preserve the palette hierarchy even when exact hex values are unavailable.
- The next main workflow steps are always shown as a vertical arrow-key menu. `Up` and `Down` move selection, and `Enter` activates the selected primary action.
- Optional side steps remain available through visible hotkeys shown in the footer or help region. Prefer modifier-based hotkeys such as `Ctrl+` combinations so they do not collide with text entry.
- Labeled text inputs are rendered outside placeholder text and must take focus explicitly. While an input is focused, plain-character hotkeys are disabled so typing never triggers application actions.
- Token entry is always masked. Origin entry remains unmasked but uses the same focus rules.
- Busy states replace the primary menu with a progress view and non-secret status text. Navigation is limited to actions that are actually safe during the active request.
- Busy states must be driven by asynchronous Bubble Tea commands so spinner ticks, safe status updates, and terminal resize messages continue while Ghostfolio requests are in flight.
- Panels, menus, and help regions should follow Ghostfolio's clean product tone: subtle separators, compact spacing, high-contrast headings, and minimal ornamentation so workflow guidance stays dominant.

## Setup Persistence Rules

- Persist only the bootstrap setup data needed before token entry: setup completion state, selected server mode, canonical Ghostfolio origin, development-mode HTTP allowance, and last-updated timestamp.
- Store the setup file under `os.UserConfigDir()` or the platform-equivalent application data directory in a `ghostfolio-cryptogains` folder.
- Write updates by serializing the full document to a temporary file, syncing it, and renaming it atomically over the previous file.
- Use restrictive local permissions where the operating system supports them.
- Canonicalize and validate the stored origin on every read. Reject malformed or now-disallowed origins before any network request.
- Never persist the Ghostfolio security token, Ghostfolio JWT, raw response payloads, retry diagnostics, or any report-related data in this slice.
- Document the resolved bootstrap file path and the user-reset procedure: deleting the bootstrap setup file forces the application back to first-run setup on the next launch.

## Ghostfolio Communication Validation Rules

- Require completed setup before the user can enter the sync workflow.
- Use the configured canonical origin plus `/api/v1` as the runtime API base.
- Authenticate with `POST /api/v1/auth/anonymous` using `{ "accessToken": "<token>" }` and require HTTP `200 OK` with a non-empty string `authToken`.
- After successful auth, request exactly one activities page with `GET /api/v1/activities?skip=0&take=1&sortColumn=date&sortDirection=asc`.
- Treat communication as successful only when the activities probe returns HTTP `200 OK`, a JSON object with an `activities` array and non-negative integer `count`, and the first returned item contains non-empty `id`, `date`, and `type` fields when `count > 0`.
- Treat `{ "activities": [], "count": 0 }` as success because an empty history is still a valid communication result.
- Any transport error, non-2xx response, malformed JSON, missing `authToken`, missing `activities`, invalid `count`, or missing minimal activity fields ends the workflow with a user-facing failure result.
- A successful validation attempt must explicitly tell the user that communication works, but that no data was stored and no report flow is available yet.

## Slice Evolution Rules

- `ActivitiesProbeResponse` and `ActivityProbeEntry` are validation-only probe models for this slice and must be removed once later specs introduce full-history Ghostfolio retrieval and normalized `ActivityRecord` modeling.
- `SyncValidationAttempt` and `ValidationOutcome` are temporary workflow abstractions and must be evolved or merged into the broader real-sync lifecycle when later specs add retrieval, normalization, persistence, and report readiness states.
- `GhostfolioSession` is expected to remain, but later specs must expand it from a validation-only session into the authenticated runtime context for full sync execution.
- `AppSetupConfig` may remain as bootstrap-only machine-local configuration, but later specs must keep it free of financial data and person-linked identifiers or move such fields into the future token-protected user and setup models.

## Complexity Tracking

No constitution violations require justification for this plan.
