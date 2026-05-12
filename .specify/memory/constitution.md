<!--
Sync Impact Report
Version change: 2.0.0 -> 2.1.0
Modified principles:
- III. Testability with Full Coverage
Modified sections:
- III. Testability with Full Coverage
- Delivery Workflow & Quality Gates
Added sections:
- None
Removed sections:
- None
Templates requiring updates:
- ✅ .specify/templates/spec-template.md
- ✅ .specify/templates/plan-template.md
- ✅ .specify/templates/tasks-template.md
Follow-up TODOs:
- None
-->
# ghostfolio-cryptogains Constitution

## Core Principles

### I. Security-First Financial Data Handling
- Security MUST be the first decision filter for design, implementation, testing,
  and review because the application processes financial data.
- Data MUST NOT be persisted unless the feature specification or implementation
  plan explicitly justifies why persistence is necessary.
- Persisted data MUST remain local to the user's machine. Remote storage,
  synchronization, telemetry export, or third-party persistence of financial data
  is prohibited unless this constitution is amended.
- Persisted data that contains financial information or can be connected to a
  specific person or user MUST be encrypted at rest with key material derived
  from the active Ghostfolio security token so that the stored data is
  unreadable without re-supplying a valid token in a later session.
- Persisted data that contains neither financial information nor person- or
  user-linkable data MAY use proportionate machine-local protection instead of
  Ghostfolio-token-derived encryption, but the feature specification or
  implementation plan MUST explicitly justify the stored fields, show that they
  do not cross that sensitivity threshold, and document the local protection
  approach.
- When persisted data contains financial information or can be connected to a
  specific person or user, its cryptographic storage design MUST follow the
  [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html),
  including minimizing stored sensitive data, using established algorithms,
  providing integrity protection, generating salts/nonces/IVs from a
  cryptographically secure random source, and separating persisted ciphertext
  from keying material where feasible.
- The Ghostfolio security token MUST be requested for every usage session,
  cached only in memory for the minimum required duration, and MUST NEVER be
  persisted.
- The Ghostfolio security token MUST NEVER be output, logged, embedded in test
  fixtures, written to files, or otherwise exposed in any channel other than the
  authenticated request path to the Ghostfolio API.
- Every feature MUST document a review of the most recent published OWASP Top 10
  relevant to the attack surface before merge.
- Every feature that persists financial information or person- or user-linkable
  data MUST document how its storage design follows the OWASP Cryptographic
  Storage Cheat Sheet before merge.
Rationale: Financial data and authentication secrets create direct privacy,
fraud, and account access risk.

### II. Deterministic Financial Precision
- Financial values MUST NOT use floating-point types in domain logic,
  persistence, calculations, or assertions except for immediate parsing into a
  safer representation.
- Monetary amounts, quantities, cost basis, exchange rates, taxes, and gains or
  losses MUST use fixed-point decimals or integer minor units with explicit
  scale.
- Rounding and conversion rules MUST be defined where values cross currencies,
  units, or reporting boundaries.
- Calculations MUST remain auditable and reproducible from the stored inputs and
  documented rounding rules.
Rationale: Floating-point behavior is not acceptable for tax and portfolio
reporting.

### III. Testability with Full Coverage
- Project-owned code MUST maintain 100% automated test coverage. When the
  selected tooling distinguishes line and branch coverage, both MUST remain at
  100%.
- Integration tests are the default and MUST verify user journeys and
  Ghostfolio-facing workflows, with outside services mocked or stubbed.
- Coverage commands and CI workflows MUST instrument project-owned packages in a
  way that counts execution driven from black-box contract and integration test
  packages. Coverage gates that only count same-package tests are not
  sufficient.
- Unit tests MUST be added only when coverage is not realistically fulfillable
  through integration tests or when a function, type, or module has enough
  complexity that isolated verification materially lowers risk.
- Unit tests that substantially duplicate the same behavior already verified by
  integration tests MUST be removed.
- A feature is incomplete until required tests, coverage gates, and relevant
  regressions pass in CI or in the local verification path when CI is
  unavailable.
Rationale: High-confidence financial software requires full behavioral evidence,
not partial sampling.

### IV. Minimal Dependencies and External Integrations
- Third-party libraries MUST NOT be added unless the standard library or a core
  language implementation would create disproportionate code size or maintenance
  cost.
- Every new dependency MUST include recorded research covering community
  acceptance, maintenance status, release freshness, security posture, and
  evidence of at least basic activity within the previous 3 months.
- Stale, weakly maintained, or unnecessary dependencies are prohibited.
- External APIs MUST NOT be integrated unless they are necessary for the product
  goal. The Ghostfolio public API is the only standing exception.
- Any external API integration or version upgrade MUST document the latest
  supported version, authentication model, failure modes, and security
  implications before implementation.
Rationale: Dependencies and integrations expand the attack surface and long-term
maintenance cost.

### V. Clean Architecture and Domain Clarity
- Code MUST follow the project baseline from Clean Code, Domain-Driven Design,
  and Clean Architecture.
- Names MUST be descriptive and unambiguous, and domain concepts MUST be modeled
  explicitly instead of hidden behind vague helpers or infrastructure-centric
  abstractions.
- Modules and functions MUST remain cohesive, minimize duplication, and respect
  SOLID boundaries where those boundaries improve clarity and change safety.
- Domain rules MUST be separated from IO and infrastructure concerns so the
  business logic remains testable and replaceable.
- Consistency is mandatory. Any deliberate deviation MUST be documented and
  justified in the relevant plan or review.
Rationale: Clear domain boundaries reduce defects, simplify testing, and keep
the codebase maintainable.

## Operational Constraints

- The default persistence policy is no persistence. When persistence is needed,
  the specification or plan MUST record what is stored, why it is stored, how it
  is protected, and how the user can remove it. When the persisted data contains
  financial information or person- or user-linkable data, this documentation
  MUST also describe token-derived encryption and the applicable OWASP
  Cryptographic Storage Cheat Sheet controls.
- Secret handling is runtime-only. Tokens, credentials, and sensitive financial
  data MUST be redacted from logs, screenshots, examples, fixtures, and
  documentation.
- Dependency and external API research MUST be recorded in `research.md`,
  `plan.md`, or equivalent review evidence before implementation starts.
- Features that affect calculations MUST define numeric representation, scale,
  rounding, and reporting assumptions in the specification.
- Unsupported practices include plaintext local caches, cloud persistence of
  sensitive data, floating-point ledger math, unreviewed dependency additions,
  and undocumented API version assumptions.

## Delivery Workflow & Quality Gates

- Every feature specification MUST capture the feature's impacts on persistence,
  token handling, financial precision, testing strategy, dependency choices, and
  external integrations when applicable.
- Every feature or change that persists financial information or person- or
  user-linkable data MUST record its OWASP Cryptographic Storage Cheat Sheet
  compliance evidence in `spec.md`, `plan.md`, `tasks.md`, or equivalent review
  notes.
- Every implementation plan MUST pass a constitution check before research and
  again before implementation.
- Every task list MUST include work for automated integration testing, coverage
  verification, security review, and any required dependency or API research.
- Pull requests MUST run the repository test workflow automatically on each push
  while the change is under review.
- Code review MUST block changes that violate a core principle or omit the
  evidence required to prove compliance.
- If the tooling cannot measure a required gate yet, adding that measurement is a
  prerequisite for completing the feature.
- A change that cannot satisfy a core principle MUST be rejected until the
  constitution itself is amended.

## Governance

- This constitution is the source of truth for project engineering policy.
  `AGENTS.md`, SpecKit templates, and future process documents MUST align with
  it.
- Compliance MUST be reviewed during specification, planning, task generation,
  implementation review, and before merge or release.
- Amendments MUST be made in the same change that updates this file, refreshes
  the Sync Impact Report comment at the top of this document, and updates every
  affected template or guidance file.
- Principle violations are not waived by TODOs, follow-up issues, or reviewer
  intent. If a rule needs to change, the constitution MUST be amended first or
  in the same change set.
- Versioning policy: MAJOR for incompatible principle removals or redefinitions.
  MINOR for new principles, new mandatory sections, or materially expanded
  governance. PATCH for clarifications, wording improvements, or typo fixes.
- Compliance evidence MUST be traceable in `spec.md`, `plan.md`, `tasks.md`,
  review notes, or equivalent artifacts. Missing evidence counts as
  non-compliance.

**Version**: 2.1.0 | **Ratified**: 2026-05-01 | **Last Amended**: 2026-05-12
