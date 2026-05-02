# Research: Ghostfolio Gains Reporting

## Go And TUI Stack

Decision: Implement the baseline as a single-module Go 1.24 application with `github.com/charmbracelet/bubbletea/v2` as the TUI runtime and only the `github.com/charmbracelet/bubbles` components that are actually needed for token entry, selections, progress, and confirmations.

Rationale: The user explicitly required Go and a TUI. `bubbletea` has a current `v2.0.6` release dated 2026-04-16 and commits on 2026-04-23. `bubbles` has a current `v2.1.0` release dated 2026-03-26 and commits on 2026-04-22. This stack is active, pure Go, and its event-loop model is easy to drive in automated tests. It also keeps the presentation layer separate from the tax domain and storage code.

Alternatives considered: `tview` would reduce custom component work for form-heavy screens but gives a less explicit state model for deterministic workflow tests. `gocui` is too low-level for the number of flow states in this product. Desktop GUI frameworks were rejected because the requirement is specifically a TUI.

## Financial Arithmetic And Rounding

Decision: Use `github.com/cockroachdb/apd/v3` for all quantities, prices, fees, proceeds, basis values, and gains or losses. Parse Ghostfolio JSON numbers with `encoding/json.Decoder.UseNumber`, convert them immediately into canonical decimal strings and `apd.Decimal` values, persist canonical decimal strings, and round only at report-output boundaries.

Rationale: The constitution prohibits floating-point domain logic. `apd` has a current `v3.2.3` release dated 2026-03-23 with commits on 2026-03-23 and 2026-03-13. It is maintained by CockroachDB, supports arbitrary precision, and exposes explicit error-returning arithmetic contexts that fit audit-sensitive financial calculations better than more convenience-oriented decimal libraries. The baseline report policy is:

- Internal calculations retain full source precision.
- Persisted numeric values keep their canonical decimal string form.
- Monetary values rendered in the PDF round to the report currency minor unit using round half up; if minor-unit metadata is unavailable, the fallback is 2 decimal places.
- Quantities render with preserved source scale after trimming insignificant trailing zeros.

Alternatives considered: `shopspring/decimal` is simpler but less strict and less attractive for accounting-grade controls. `math/big.Rat` is exact but awkward for decimal presentation and rounding policy. `govalues/decimal` has a fixed precision ceiling that is harder to justify for crypto ledgers.

## Persistence Strategy

Decision: Store each registered local user in one encrypted snapshot file located under the OS application data directory. Each file uses an opaque random filename, a small authenticated cleartext header, and a versioned JSON payload encrypted with AES-256-GCM. The encryption key is derived only at runtime from the user-entered Ghostfolio token via Argon2id.

Rationale: This is the best fit for a security-first local TUI with modest dataset size and no need for ad hoc SQL queries. It keeps the token out of persistent storage, protects all user-related metadata and activity history, keeps distribution simple, and makes server-mismatch replacement an atomic single-file swap instead of a database migration problem. The baseline envelope is:

- Header fields: magic, outer-format version, KDF algorithm, KDF parameters, random salt, AEAD nonce.
- Header handling: pass the serialized header as AEAD additional authenticated data so metadata tampering is detected.
- Payload fields: schema version, registered-user metadata, setup profile, sync metadata, normalized activity cache, and available report years.
- File permissions: create with `0600` where the platform supports POSIX-style modes.
- Update strategy: write a temp file, `fsync`, then atomic rename over the old snapshot only after a successful sync and validation.

Selected crypto details:

- KDF: `golang.org/x/crypto/argon2` with Argon2id.
- Initial baseline parameters: memory `19 MiB`, iterations `2`, parallelism `1`, key length `32` bytes.
- Salt: fresh random 16-byte salt on every full rewrite.
- Nonce: fresh random 12-byte GCM nonce on every encryption.
- Failure behavior: wrong token and file corruption both surface as the same generic unlock failure; the application does not attempt partial salvage.

Profile discovery decision:

- Do not persist a plaintext profile index because registered-user metadata and setup data must remain protected.
- On unlock, enumerate opaque snapshot files and attempt authenticated decrypt until one succeeds for the supplied token.
- This is acceptable because the expected number of local profiles per machine is small.

Alternatives considered: SQLite with app-layer encryption adds WAL and temp-file complexity without solving key management. SQLCipher is strong technically but introduces CGO and distribution complexity that is unnecessary for 10,000-record snapshots. `bbolt` adds little value if the payload is already encrypted as one blob. OS keychains were rejected for the primary path because the token must not be persisted and the cache must remain unreadable without re-entering that token.

## Ghostfolio HTTP Integration

Decision: Use Go's standard library HTTP client against Ghostfolio's currently observed `api/v1` endpoints, authenticate by posting the user-entered Ghostfolio security token for the selected registered local user to `POST /api/v1/auth/anonymous`, and use the returned bearer JWT only for the active sync session. Retrieve activity history via paged `GET /api/v1/activities` requests.

Rationale: `net/http`, `context`, and `encoding/json` are sufficient and keep the dependency surface small. Ghostfolio has a current `3.1.0` release dated 2026-04-29 and commits on 2026-05-01, so the upstream is active. Current upstream source shows:

- `POST /api/v1/auth/anonymous` accepts `{ "accessToken": "..." }` and returns `{ "authToken": "..." }`.
- Invalid token during anonymous auth returns `403 Forbidden`.
- Authenticated activity endpoints use bearer JWT auth.
- `GET /api/v1/activities` returns `{ activities, count }` and supports `skip` and `take` pagination.
- `GET /api/v1/health` exists for optional preflight checks.

Integration decisions:

- Require HTTPS by default except explicitly allowed local-development origins such as `localhost`.
- Canonicalize and pin the selected Ghostfolio origin in the encrypted setup profile.
- Probe `GET /api/v1/health` or `POST /api/v1/auth/anonymous` early to surface connectivity and version problems before a long sync.
- Treat missing, redacted, unsupported, or internally inconsistent activity payload fields as hard sync failures.

Alternatives considered: `resty` or other HTTP wrappers were rejected because the stdlib already covers the required features. Ghostfolio API keys were rejected for the baseline because the observed history endpoints are JWT-guarded and the specification is written around the Ghostfolio security token. The public portfolio endpoint was rejected because it is not sufficient for defensible activity reconstruction.

⚠️ [UNCERTAINTY] Ghostfolio's authenticated activity endpoints are visible in current source but are not fully documented as a stable public contract. The client must validate compatibility at runtime and fail safely when required fields are missing or the server behavior changes.

## PDF Generation

Decision: Render the yearly report through `github.com/signintech/gopdf`, wrapped behind an internal report-PDF interface so the library can be replaced later without rewriting the report domain.

Rationale: Go's standard library does not provide PDF authoring. `gopdf` is pure Go, has current tags up to `v0.36.0`, and shows activity on 2026-03-12. It is sufficient for deterministic text and table layouts without external binaries or CGO. The wrapper isolates the weakest part of the dependency stack and keeps report layout logic out of the domain layer.

Alternatives considered: `pdfcpu` is strong for PDF processing but is a weaker fit for initial document authoring. `unidoc` was rejected because of licensing and commercial dependency risk. Archived `gofpdf` derivatives were rejected on maintenance grounds.

## Testing And Coverage Gate

Decision: Use integration-first Go tests with `httptest.Server`, temp directories, and deterministic sample ledgers. Use `go test -coverprofile` for statement coverage and `github.com/Fabianexe/gocoverageplus` as the initial branch/file coverage gate helper.

Rationale: The constitution requires 100% automated coverage and explicitly calls out branch coverage when tooling distinguishes it. Go's native tooling does not provide branch coverage. `gocoverageplus` has a current `v1.2.0` release dated 2026-03-22, recent commits on 2026-03-22, a tagged stable module page on `pkg.go.dev`, and the project explicitly documents branch-coverage reporting on top of Go coverage profiles. Because it is a development-only tool and not shipped with the product, its security exposure is limited.

Risk assessment:

- Maintenance: active in the last 3 months.
- Community acceptance: low, with very limited ecosystem usage signals.
- Security posture: development-only report processor, not runtime-linked into the product.
- Release freshness: recent tagged release and recent commit activity.

Alternatives considered: relying only on statement coverage violates the constitution. Building a custom branch analyzer inside this repository is possible but adds work before any product behavior can ship. The implementation should replace `gocoverageplus` only if it proves inaccurate or operationally brittle.

## Dependency Due Diligence Summary

| Dependency | Purpose | Need vs stdlib | Freshness / activity evidence | Acceptance / risk summary |
|------------|---------|----------------|-------------------------------|---------------------------|
| `bubbletea` | TUI runtime | Stdlib has no TUI framework | Release `v2.0.6` on 2026-04-16, commits on 2026-04-23 | Strong activity, low runtime risk, presentation-only concern |
| `bubbles` | Focused TUI widgets | Avoids custom text inputs and selectors for standard controls | Release `v2.1.0` on 2026-03-26, commits on 2026-04-22 | Use selectively to keep dependency surface small |
| `apd/v3` | Exact decimal arithmetic | Stdlib lacks decimal financial arithmetic | Release `v3.2.3` on 2026-03-23, commits on 2026-03-23 | Mature maintainer, good fit for financial correctness |
| `golang.org/x/crypto/argon2` | Token-based KDF | Stdlib lacks Argon2id | Commits on 2026-05-01 and 2026-04-23 in official Go subrepository | Official Go-maintained crypto extension, low concern |
| `gopdf` | PDF writer | Stdlib lacks PDF authoring | Tags through `v0.36.0`, commits on 2026-03-12 | Adequate activity, weaker ecosystem signal, keep behind adapter |
| `gocoverageplus` | Branch/file coverage reporting | Stdlib lacks branch coverage measurement | Release `v1.2.0` on 2026-03-22, commits on 2026-03-22 | Dev-only tool with low adoption; acceptable with monitoring |
