# Research: Sync Data Validation

## Go Toolchain And Full-Screen TUI Stack

Decision: Keep the implementation as a single-module Go 1.26.2 application using `charm.land/bubbletea/v2` for the event loop and only the `charm.land/bubbles/v2` components needed for menus, labeled inputs, help text, and loading states.

Rationale: The broader feature research in `specs/001-ghostfolio-gains-reporting/research.md` already established that Go 1.26.2 satisfies the current Bubble Tea and Bubbles minimum toolchain requirements and that both libraries are actively maintained. This smaller slice needs only the TUI subset of that stack. Bubble Tea also cleanly supports a root model launched with `tea.WithAltScreen()`, which satisfies the requirement that the TUI take over the full terminal screen from the moment it starts.

Alternatives considered: `tview` was rejected because the Bubble Tea state model is easier to align with deterministic workflow tests and future feature growth. `gocui` was rejected because it would require more custom layout and input handling for the same workflow. A partial-screen TUI was rejected because the user explicitly asked for full-terminal presentation.

## Full-Screen Workflow Interaction Model

Decision: Standardize every screen on a full-screen layout with clearly delimited regions and one vertical arrow-key menu for the next main workflow steps. Use visible modifier-based hotkeys only for optional side actions. Use labeled `textinput` fields for custom origin and token entry, and disable conflicting hotkeys while an input has focus.

Rationale: This directly implements the user guidance for a pleasant, clear, and keyboard-driven TUI. A single primary menu per screen keeps the main path obvious. Visible hotkeys in a help footer keep side actions discoverable without forcing them into the main path. Focus-aware key routing prevents the common failure case where plain-character hotkeys interfere with typing inside text fields.

Alternatives considered: Global single-letter hotkeys on every screen were rejected because they conflict with text entry and make state reasoning harder. Placeholder-only inputs were rejected because labels disappear while the user is typing. A table-first interaction model was rejected because this slice is about action selection rather than row comparison.

## Setup Persistence Strategy

Decision: Persist only the bootstrap setup state as a small machine-local JSON document under the operating system's config or app-data directory, with atomic rewrites and restrictive local permissions.

Rationale: The persisted data in this slice is limited to setup completion state and the selected Ghostfolio origin. The application must be able to read that state before the user enters a Ghostfolio token, so a runtime token-derived encrypted file would not satisfy the slice. A local file keeps the runtime small, avoids CGO and desktop-service dependencies, and is the most reliable cross-platform bootstrap design for a terminal application.

Alternatives considered: OS keychain or keyring storage was rejected because it adds runtime variability, desktop-service assumptions, and little practical security value for this non-secret bootstrap data. SQLite was rejected because a single small setup document does not justify a database. A Ghostfolio-token-derived encrypted file was rejected because it conflicts with the slice requirement that setup be readable before token entry.

## Constitution Alignment For Bootstrap Persistence

Decision: Treat the bootstrap setup file as less-sensitive machine-local configuration that still requires explicit justification, local-only storage, and proportionate protection, but not Ghostfolio-token-derived encryption.

Rationale: The amended constitution now reserves the strict token-derived encryption and OWASP Cryptographic Storage Cheat Sheet requirements for persisted data that contains financial information or can be connected to a specific person or user. This slice persists only setup completion state and the selected Ghostfolio origin, stores no Ghostfolio token, JWT, financial payload, or user identity, and therefore fits the narrower bootstrap-configuration case. The plan must still document why the file is persisted, what fields it contains, and what local protection controls apply.

Alternatives considered: Treating the bootstrap file as exempt without documentation was rejected because the constitution still requires traceable persistence justification. Persisting the Ghostfolio token or a token verifier to unlock setup at startup was rejected because token persistence remains prohibited.

## Ghostfolio Communication Validation Contract

Decision: Treat communication as valid only after a successful anonymous-auth request and a successful one-page activities probe against the selected Ghostfolio origin.

Rationale: The reference Ghostfolio contract in `specs/001-ghostfolio-gains-reporting/contracts/ghostfolio-sync.md` already identifies the observed `api/v1` boundary. This smaller slice only needs enough validation to prove that the selected server accepts the provided Ghostfolio token and returns the minimum activities payload shape the later sync flow will depend on. The minimal successful probe is:

- `POST /api/v1/auth/anonymous` returns HTTP `200 OK` and a non-empty string `authToken`.
- `GET /api/v1/activities?skip=0&take=1&sortColumn=date&sortDirection=asc` returns HTTP `200 OK`, a JSON object with `activities` as an array and `count` as a non-negative integer.
- When `count > 0`, the first returned item contains non-empty `id`, `date`, and `type` fields.
- When `count == 0`, an empty `activities` list still counts as success.

Alternatives considered: A health-only probe was rejected because it does not prove that authentication and activity retrieval work. Full-history pagination and domain validation were rejected because this slice must stop after communication validation and explicitly defer persistence and reporting concerns.

## Testing And Coverage Gate

Decision: Use integration-first Go tests with `httptest.Server` for Ghostfolio interactions and screen-flow tests for first-run setup, main-menu selection, sync validation, and retry behavior. Keep `github.com/Fabianexe/gocoverageplus` as the branch and file coverage gate helper used in the broader feature research.

Rationale: This slice is primarily workflow and contract behavior, which is best tested end to end. Unit tests still add value for pure pieces such as origin canonicalization, minimal response validators, and focus-aware key routing. The constitution still requires 100 percent coverage for project-owned code, so the existing development-time coverage helper remains the smallest practical choice until the repository replaces it with an in-repo verifier.

Alternatives considered: Unit-only coverage was rejected because it would miss screen transitions and network contract behavior. Statement-only coverage was rejected because it does not satisfy the constitution's branch and file coverage requirement.

## Dependency Due Diligence Summary

| Dependency | Purpose In This Slice | Evidence Source | Acceptance And Risk Summary |
|------------|-----------------------|-----------------|-----------------------------|
| `bubbletea` | Full-screen event loop and screen routing | `specs/001-ghostfolio-gains-reporting/research.md` | Active and appropriate for a terminal workflow application; presentation-layer concern only |
| `bubbles` | Standard menu, input, help, and spinner widgets | `specs/001-ghostfolio-gains-reporting/research.md` | Use selectively to keep the dependency surface small |
| `gocoverageplus` | Branch and file coverage gate in local verification | `specs/001-ghostfolio-gains-reporting/research.md` | Development-only helper with limited ecosystem adoption, acceptable while monitored |

Dependencies intentionally deferred from the broader feature are `apd/v3`, `argon2`, and `gopdf` because this slice does not yet perform financial calculations, token-derived encrypted storage, or PDF generation.
