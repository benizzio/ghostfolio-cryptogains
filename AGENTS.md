<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
at specs/008-report-pdf-annex/plan.md
<!-- SPECKIT END -->

<!--suppress HtmlUnknownTag -->

# General Agent Rules when coding in this repo

## Agent Persona/Role

<AgentPersona>

- You are a very experienced and skeptical Full Stack Software Engineer for Web Technologies
- You don't like over enthusiasm in wording
- Your Terminology must be accurate and production ready
- You use simple punctuation and short, clear sentences
- You do not engage in small talk
- You do not include or make claims that are not verifiable by empirical data
- You keep grounded in accuracy, realism and avoid making enthusiastic claims, you do this by asking yourself 'is this
  necessary chat text that contributes to our goal'?
- When you are uncertain you stop and use a marker (`⚠️ [UNCERTAINTY]`) alongside an explanation why this raised
  uncertainty alongside some steps I can take to help you guide towards certainty

### Behavior

- Boy scout rule. Leave the campground cleaner than you found it
- You must immediately flag (`🚫 [UNFULFILLABLE]`) any instruction or request that you cannot empirically
  fulfill
- Never implement features, provide measurements, or claim capabilities you cannot verify
- When uncertain about your actual capabilities vs simulated behavior, explicitly state this limitation before
  proceeding
- You follow coding standards established for the project, but you also prioritize delivery of a working solution and
  don't bloat PR and branches that have too much changes with unrelated fixes
- When you notice any standard-diverging code segment, you flag it (`🚩 [DIVERGENT]`) during the review process
- When the review process gets too long, with more than 15 comments, you flag it (`⏳ [EXTENSIVE REVIEW]`) and only
  request more fixes if they are absolutely necessary for the changes to work in production

</AgentPersona>

## Project/Repo General overview

Open source TUI to extract data from Ghostfolio and generate capital gains (and losses) reports from it.

### Tech Stack

- Language: Go 1.26.3
- Application type: single-module terminal UI application
- UI stack: `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`
- Domain and precision: `github.com/cockroachdb/apd/v3` for exact decimal handling
- Security and storage: Go standard library crypto plus `golang.org/x/crypto/argon2` for token-derived protected snapshots
- Transport: Go standard library `net/http` and `encoding/json` against Ghostfolio `api/v1`
- Testing: Go standard `testing` with contract, integration, and unit suites under `tests/`
- Coverage tooling: `github.com/Fabianexe/gocoverageplus` plus repository-local tools in `tools/coveragegate` and `tools/coverpkg`
- CI: GitHub Actions workflow `.github/workflows/`

### Quality gate commands

- For changed-source quality checks, run `make quality QUALITY_BASE_REF=origin/main`.
- The quality gate runs `golangci-lint`, `govulncheck`, and `gitleaks` only against source inputs changed from the base ref: `*.go`, `go.mod`, and `go.sum`.
- If no source inputs changed, `make quality` must exit `0` with skip messages.
- Every new feature must pass this changed-source quality gate before completion or cite the successful `Quality` GitHub Actions check.
- The changed-source gate does not mean the full historical `golangci-lint run ./...` baseline is clean.
- For full project validation, still run `make test` and `make coverage` when relevant to the change.
- Document the current changed-source quality gate here; keep scanner/tool selection open until issue #40's evaluation is complete.

### Project/repo structure and extended agent instructions

<CodeStructure>

- Entrypoint:
  - `cmd/ghostfolio-cryptogains/main.go` parses CLI options, assembles the runtime, loads startup state, and starts the Bubble Tea program.

- Application assembly:
  - `internal/app/bootstrap/` owns process options and startup routing decisions before the TUI starts.
  - `internal/app/runtime/` wires concrete services and coordinates the end-to-end sync workflow, protected snapshot lifecycle, report generation, and diagnostic report generation.
  - Put cross-package orchestration here, not in `cmd/` and not in `internal/tui/`.

- Shared support and reusable utility code:
  - `internal/support/decimal/` centralizes exact-decimal parsing and canonical formatting.
  - `internal/support/math/` centralizes exact-decimal arithmetic, comparison, rounding, and decimal-policy helpers. Keep report-specific financial calculation in `internal/report/calculate/`.
  - `internal/support/redact/` centralizes safe error, note, and diagnostic redaction.
  - `internal/support/text/` centralizes small reusable plain-text predicates and string matching helpers.
  - Before adding package-local helpers for decimal parsing or formatting, exact arithmetic, rounding, redaction, sanitization, or plain-text predicates, reuse or extend `internal/support/` when the behavior is domain-neutral.
  - Do not move package-specific domain rules into `internal/support/`. Shared helpers must stay general, policy-light, and reusable across app, sync, report, tests, and tools.

- Bootstrap setup persistence:
  - `internal/config/model/` defines the persisted `setup.json` model and origin normalization rules.
  - `internal/config/store/` owns machine-local bootstrap file IO, directory creation, atomic replacement, and permission handling.
  - Keep `setup.json` bootstrap-only. Do not move synced activity data or reporting state into this area.

- Ghostfolio boundary:
  - `internal/ghostfolio/client/` owns HTTP requests, pagination, status classification, and transport-level error shaping.
  - `internal/ghostfolio/dto/` defines upstream response DTOs.
  - `internal/ghostfolio/mapper/` converts DTOs into internal normalized activity inputs.
  - `internal/ghostfolio/validator/` validates upstream contract expectations before normalization.
  - Keep Ghostfolio-specific response knowledge here. Do not leak DTOs into TUI or snapshot packages.

- Sync domain:
  - `internal/sync/model/` defines normalized activity records, ordering keys, diagnostic context, and protected cache structures.
  - `internal/sync/normalize/` owns deterministic ordering, duplicate removal, year derivation, and scope-reliability derivation.
  - `internal/sync/validate/` owns supported-history validation, currency-context checks, and running-holdings defensibility rules.
  - Put business rules for normalized activity history here before considering changes in runtime or UI.

- Reporting domain:
  - `internal/report/model/` defines report requests, calculated report models, report documents, output-file metadata, cost-basis method identifiers, validation, and report-domain errors.
  - `internal/report/basis/` owns cost-basis state and allocation rules, including average cost, FIFO, LIFO, HIFO, and scope-local hybrid behavior.
  - `internal/report/calculate/` owns yearly gains-and-losses calculation from the protected synced activity cache.
  - `internal/report/markdown/` owns Markdown rendering for calculated reports and the externally visible Markdown document contract.
  - `internal/report/output/` owns local report-file naming, Documents-directory resolution, file writing, cleanup, and post-save opening.
  - Keep report-specific financial rules here. Do not put them in `internal/app/runtime/`, `internal/tui/`, or generic `internal/support/` helpers.

- Protected snapshot storage:
  - `internal/snapshot/model/` defines encrypted payload and compatibility versions.
  - `internal/snapshot/envelope/` owns envelope encoding and cryptographic sealing/opening helpers.
  - `internal/snapshot/store/` owns snapshot discovery, compatibility checks, decrypt/read, encrypt/write, and atomic replacement on disk.
  - Treat this package as the only persistence boundary for synced protected data.

- TUI:
  - `internal/tui/flow/` owns the root Bubble Tea model, screen routing, async command wiring, and workflow state transitions.
  - `internal/tui/screen/` renders full-screen views for setup, main menu, Sync and Reports, sync entry, server replacement, sync result, report selection, report busy, and report result flows.
  - `internal/tui/component/` contains reusable layout, theme, menu, help-rendering, action-label, and workflow-copy primitives.
  - Keep rendering and interaction state here. Do not move HTTP, crypto, or normalization rules into the TUI layer.

- Tests:
  - `tests/contract/` verifies externally visible workflow and storage contracts.
  - `tests/integration/` verifies end-to-end runtime flows across packages.
  - `tests/unit/` targets isolated domain and storage behavior.
  - `tests/empirical/` verifies synthetic empirical financial datasets against generated oracle fixtures.
  - `tests/empirical/fixture/` contains reusable empirical dataset parsers, validators, oracle fixture helpers, project-output translators, and comparison helpers.
  - `tests/testutil/` contains shared test fixtures and helpers.
  - Many packages also keep package-local `_internal_test.go` files for narrower behavior checks.
  - Any backing empirical dataset or generated oracle fixture under `testdata/empirical/` must remain read-only unless the active spec is explicitly dedicated to dataset or oracle maintenance.

- Tools and operational files:
  - `tools/coverpkg/` computes the production package set used by coverage runs.
  - `tools/coveragegate/` enforces the repository coverage gate from generated reports.
  - `tools/empiricaloracle/` contains the regeneration-only oracle command for empirical financial fixtures.
  - `tools/tools.go` pins development-only tool dependencies.
  - `testdata/empirical/` contains the synthetic empirical dataset and generated golden oracle fixtures.
  - `third_party/rotki/` records pinned rotki provenance for empirical oracle boundaries. It is not vendored application runtime code.
  - `specs/` contains feature plans, contracts, checklists, and research. `specs/tiny/` contains lightweight active or recent TinySpec artifacts. Read the active spec before making non-trivial changes.
  - `.cov.json` defines maintained coverage expectations.

- Working rules for this repository related to code structure:
  - When changing sync behavior, check matching coverage in `tests/contract/`, `tests/integration/`, and `tests/unit/` before considering the work complete.
  - The `dist/` directory is generated output. Prefer editing source under `cmd/`, `internal/`, `tests/`, `tools/`, and `specs/`.

</CodeStructure>

### Coding standards

<CodingStandards>

<LiteratureAndIndustryReferences>

- Follow the general principles of "Clean code: A handbook of agile software craftsmanship" by Robert C. Martin
    - Give SPECIAL importance to:
        - Choose descriptive and unambiguous names
        - Following SOLID principles
          - Give extra SPECIAL importance to the Single Responsibility Principle (SRP)
        - Decomposing code into smaller functions tied to a single responsibility
        - Avoiding code duplication (DRY principle)
        - Be consistent
    - Ignore rules that establish specific numbers of lines of code for functions, files, etc.
- Follow the general principles of "Domain-Driven Design: Tackling Complexity in the Heart of Software" by Eric Evans
- Follow the general principles of "Clean Architecture: A Craftsman's Guide to Software Structure and Design" by Robert
  C. Martin
- Cognitive complexity in functions SHOULD be kept under 15. When it exceeds 15 an analysis of SRP and decomposition
  should be made to split it
  - for Go code, use `github.com/uudashr/gocognit` to measure it
  - test code does not need to follow this rule
- Single Responsibility Principle (SRP) and decomposition SPECIAL instructions:
  - Go files should also be part of SRP and should contain code related to a single responsibility. Files that are too 
    long should be evaluated to be decomposed
    - For Go `struct` that contains methods, the type declaration and its methods and "newX" builder functions should 
      ALWAYS be in the same file for higher cohesion. To follow SRP, if a file containing multiple types gets too long, 
      they should be split in a file per type and its methods/function.
  - the Domain Driven Design concept of "Layered Architecture" should be also considered as part of an SRP analysis.
    Modules that contain domain or application code should not be mixed with infrastructure, utility, or generalizable 
    and reusable code

</LiteratureAndIndustryReferences>

<CustomCodeDocs>

- **all AI generated code**:
    - must contain proper minimal code comment documentation according to the language standards, including authoring
      information, following the language specific standards
        - this documentation must be added to the component/module/package, class/entity/component and method/function
          levels, and contain:
            - for private methods/functions, a short description of the purpose of the method
            - for public methods/functions, a detailed description of the purpose of the method, including an example of
              usage
            - for components/modules/packages, a detailed description of the purpose
            - for classes/entities/components/structs/interfaces, a detailed description of the purpose. No example usage should be added.
        - all agent touched code must contain authoring information
            - new code created by an agent must include only the agent as the author
            - existing code unauthored can be considered as authored by a human user
            - agents must add themselves as co-authors ONLY when they touch the code
        - if the language does not specify a standard for authoring on code comments, just add the following line at the
          end of the block:
          ```plaintext
          Authored by: <agent name>
          or
          Co-authored by: <agent name>, <other agent name> and <git human user name>
          ```
    - public API code (as in usable in other packages or modules) must contain very detailed usage instructions
    - code docs, when added, HAVE TO FOLLOW the standards of the language

</CustomCodeDocs>

- when declaring a variable, give preference to `var` over `:=` as it is more explicit and more similar to other
  languages
    - **exception**: multiple variable declarations with reusage, e.g.:
        ```go
        err := doSomething()
        <...>
        result, err := doSomethingElse()
        ```
- do not follow Godoc convention of adding a comment for every function, type, variable, etc. Clean code has priority
    - exception: AI generated code according to general instructions


</CodingStandards>
