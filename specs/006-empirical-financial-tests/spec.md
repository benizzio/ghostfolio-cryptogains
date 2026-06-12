# Internal Specification: Empirical Solidified Financial Tests

**Feature Branch**: `[006-empirical-financial-tests]`

**Created**: 2026-06-05

**Status**: Draft

**Input**: User description: "Create an empirical external dataset, an hledger-backed empirical calculation oracle, and empirical solidified financial integration tests for the capital gains and losses calculation layer. This is internal test infrastructure only and does not define user-facing stories."

## Purpose

This specification defines the needs for creating and using an `empirical external dataset` to validate the financial calculation layer against expected outcomes produced by an external plain-text-accounting engine.

This work is not a user-facing product feature. It exists to detect calculation drift in FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Hybrid (`scope_local_hybrid`), defined as scope-local exact-unit matching with scope-local average-cost fallback and oldest-acquired deemed-disposal order.

The specification is kept in `spec.md` for repository tooling compatibility, but it intentionally uses internal validation objectives instead of user stories.

## Clarifications

### Session 2026-06-05

- Q: Should zero-priced holding reductions leave cost basis unchanged? -> A: No. The intended rule is that zero-priced holding reductions reduce holdings and remove basis under the selected method, but create no proceeds, gain, or loss.
- Q: Should this be a traditional feature specification with user stories? -> A: No. This is internal test infrastructure. Use the spec template only as inspiration and document needs without user stories.
- Q: Where should empirical solidified financial tests live? -> A: In a completely isolated Go test package dedicated to empirical financial validation, not in the existing integration test package.
- Q: Should empirical solidified financial tests compare generated report formats? -> A: No. They must compare pure normalized calculation results only and must not assert Markdown, report document, or UI formatting.
- Q: Should empirical solidified financial tests cover currency conversion? -> A: No. They must run entirely with one currency to isolate cost-basis and gains-and-losses calculations. Currency conversion logic must never be added to this empirical suite.
- Q: How should empirical comparisons handle decimal equality versus precision differences? -> A: Quantities must compare by exact decimal equality after normalization. Calculated financial values must first be normalized under one selected decimal policy; if ~~hledger~~ the selected external oracle cannot be configured to match the project's 16-decimal production policy for every valid case, empirical tests must set `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY` to make the project calculation layer use ~~hledger's~~ the selected external oracle's established decimal policy while keeping production default behavior at 16 decimals when the variable is unset. After that alignment, calculated financial values may use documented per-field tolerances for residual ~~hledger/project~~ external-oracle/project deviations, capped at one unit of the selected decimal-policy scale.
- Q: How should empirical tests handle ~~hledger~~ external oracle availability? -> A: ~~hledger must be vendored inside the repository for empirical test use.~~ BUG-001 supersedes this with repository-controlled external oracle provenance, pinned rotki materials, adapter constraints, checksums, and retained hledger materials only when applicable.
- Q: Should oracle outputs be persisted or generated only during tests? -> A: Persist normalized oracle outputs as golden fixtures.
- Q: When may empirical tests run ~~hledger~~ external oracle generation? -> A: Only when the required golden fixture is absent or when maintainers explicitly regenerate fixtures through the documented test-time boundary.
- Q: Is ~~hledger licensing~~ external oracle licensing compatible with repository vendoring? -> A: Compatibility must be documented for the selected oracle materials, including license text, source provenance, pinned version or commit, checksums, and no prohibited binary-only vendoring.

### Session 2026-06-10

**Bugfix**: 2026-06-10 — [BUG-001] Patched empirical oracle requirements after fixture-backed runs skipped most supported comparisons.

- Q: Can hledger remain the sole external oracle for every supported empirical method? -> A: No. FIFO, LIFO, HIFO, and Average Cost aggregate comparisons must use a pinned rotki-based test-time oracle adapter. Scope-Local Hybrid must use a separate composite oracle with rotki-backed arithmetic assertions and documented project-owned composition-rule assertions.
- Q: Are zero-priced holding reductions part of the external-oracle comparison scope? -> A: No. They remain project behavior that must be covered by non-oracle unit, integration, or contract tests, and they must not be counted as supported external-oracle fixture coverage.
- Q: May empirical tests pass when supported fixture groups are skipped before project calculation and oracle comparison? -> A: No. Supported fixtures must execute calculation and comparison, and unexpected supported fixture skips must fail acceptance.
- Q: May BUG-001 remediation depend on a developer-local rotki installation or unstaged prototype outputs? -> A: No. Rotki-backed remediation must first establish a repository-controlled boundary. ~~Acceptable evidence is pinned rotki materials under version control or committed raw rotki outputs with provenance metadata under repository-controlled paths.~~ BUG-002 supersedes this evidence shortcut: acceptable regeneration evidence requires verified untracked pinned rotki source execution through the local adapter boundary.

### Session 2026-06-12

**Bugfix**: 2026-06-12 — [BUG-002] Patched rotki fixture regeneration requirements to require verified untracked pinned rotki source execution and reject committed raw rotki outputs as oracle evidence.

- Q: How must rotki-backed golden fixture regeneration obtain oracle data? -> A: The explicit regeneration command must download or reuse pinned rotki source in an untracked project-local folder, verify the configured provenance and checksums, execute rotki calculation code through a project-owned local adapter boundary, and transform those results into committed normalized golden fixtures and provenance metadata.
- Q: Are committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, or vendored rotki source acceptable oracle evidence? -> A: No. They are not acceptable as the source of regenerated oracle data. Normal fixture-backed tests must continue to read committed golden fixtures without downloading rotki or invoking oracle generation.

## Terms Used In This Spec

- **Empirical external dataset**: A project-owned, simplified, human-readable ledger of activity inputs and expected outcome references used only for empirical financial validation.
- **Empirical calculation oracle**: A deterministic tool or test helper that ~~uses `hledger`~~ uses pinned test-time external oracle adapters to process the empirical external dataset and emit normalized, assertable expected results. BUG-001 supersedes the hledger-only assumption because observed runs skipped most supported fixtures.
- **Empirical solidified financial tests**: Supplemental integration tests that translate the empirical external dataset into this project's calculation inputs, run the project calculation layer, and compare those results with the oracle output.
- **Oracle output**: The normalized assertable golden fixture generated from ~~hledger results~~ external-oracle or composite-oracle results, including gains, losses, cost basis, holdings, and matched-lot evidence where supported.
- **Rotki-backed external oracle**: A pinned test-time adapter that derives FIFO, LIFO, HIFO, and Average Cost aggregate expected values from verified rotki source downloaded or reused in an untracked project-local folder during explicit golden-fixture regeneration, and records source provenance, version or commit identity, adapter constraints, arguments, and hashes.
- **Scope-Local Hybrid**: The canonical short name for method key `scope_local_hybrid`, defined as scope-local exact-unit matching with scope-local average-cost fallback and oldest-acquired deemed-disposal order.
- **Zero-priced holding reduction**: An explained activity that reduces quantity and basis under the selected method without proceeds, realized gain, realized loss, or priced-liquidation treatment.
- **Dataset maintenance spec**: An isolated specification whose explicit purpose includes creating, correcting, or expanding the empirical external dataset. This specification is such a dataset-maintenance spec.

## Internal Validation Objectives

### Objective 1 - Maintain An Empirical External Dataset

Create an empirical external dataset that is broad enough to exercise the calculation situations supported by this project.

**Independent Test**: The dataset can be parsed and validated without running ~~hledger~~ external oracle generation or the project calculation layer, and validation confirms at least 3 source-calendar years, at least 150 activities, required method coverage, required edge-case coverage, and stable deterministic ordering metadata.

**Required Outcomes**:

1. The dataset contains at least 150 activities.
2. The dataset spans at least 3 source-calendar years.
3. The dataset uses a simplified, human-readable, maintainable format.
4. The dataset has deterministic source IDs, source timestamps, asset identity keys, display labels, quantities, monetary values, fees, source scopes, and zero-priced reduction explanations where applicable.
5. The dataset remains read-only after this dataset-maintenance work is complete, except in later isolated dataset-maintenance specifications.

### Objective 2 - Produce An ~~hledger-Backed~~ External Oracle

Create an empirical calculation oracle that ~~uses hledger~~ uses a pinned rotki-based test-time adapter for FIFO, LIFO, HIFO, and Average Cost aggregate expected outcomes, plus a separate Scope-Local Hybrid composite oracle.

**Independent Test**: Running the oracle on the dataset produces deterministic normalized golden fixture output whose hash is stable for the same ~~hledger version~~ external oracle identity, adapter code, composite-rule version, and dataset input. Empirical tests reuse existing golden fixtures and invoke oracle generation only when a required golden fixture is absent.

**Required Outcomes**:

1. The oracle records the ~~hledger version, command arguments~~ external oracle name, source URL, pinned version or commit, adapter or command arguments, dataset input hash, oracle output hash, and any normalization step used before comparison.
2. The oracle emits expected realized gains and losses, allocated basis, closing quantity, closing basis, and method-specific lot or pool evidence where the selected external oracle or composite oracle provides enough data.
3. ~~The oracle supports FIFO, LIFO, HIFO, and Average Cost Basis directly through hledger lot-reduction behaviour.~~ The oracle supports FIFO, LIFO, and HIFO through rotki-backed fixtures, and supports Average Cost Basis through rotki-backed ACB aggregate values.
4. The oracle supports the Scope-Local Hybrid (`scope_local_hybrid`) method by using ~~hledger-backed evidence~~ rotki-backed arithmetic assertions where an external subproblem is valid, plus documented project-owned composition rules for the hybrid lifecycle that no single external method models as one native method.
5. ~~Zero-priced holding reductions are translated to hledger in a way that reduces quantity and basis without creating proceeds, realized gain, or realized loss. The preferred hledger representation is a lotful negative holding movement with no transacted selling price when that produces the required no-gain result. If a dataset case cannot be represented faithfully in hledger syntax, the oracle must mark that case as unsupported for external-oracle comparison instead of silently fabricating expected results.~~ Zero-priced holding reductions are excluded from empirical external-oracle fixture scope and remain covered by non-oracle unit, integration, or contract tests.
6. Explicit rotki-backed fixture regeneration obtains oracle data by executing verified pinned rotki source from an untracked project-local source directory through the project-owned local adapter boundary, not from committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, or vendored rotki source.

### Objective 3 - Add Empirical Solidified Financial Integration Tests

Create an integration test category that runs this project's calculation system from the same empirical external dataset and compares project output with oracle output.

**Independent Test**: A targeted empirical test command runs the dataset translation, project calculation, oracle comparison, and failure reporting from a dedicated isolated Go test package without invoking Ghostfolio, TUI rendering, protected snapshot encryption, Markdown rendering, or any report-format assertion.

**Required Outcomes**:

1. Tests are implemented as normal Go tests in a dedicated isolated empirical package, expected to be `tests/empirical`, while still exercising the calculation layer as integration-style validation.
2. Tests translate the empirical external dataset directly into calculation-layer inputs equivalent to synced protected activity records.
3. Tests cover every supported cost basis method in this project.
4. Tests compare only normalized pure calculation output to oracle output with documented decimal-policy alignment rules.
5. Tests do not compare Markdown, report document structures, TUI text, output filenames, or any other report-format boundary.
6. Test failure output identifies the dataset case, method, asset, year, field, expected value, actual value, and difference without exposing secrets or irrelevant artifacts.
7. Normal fixture-backed empirical test runs read committed normalized golden fixtures and do not download rotki source, require rotki source availability, or invoke oracle generation.

## Requirements

Each requirement applies to internal test infrastructure, not to user-facing application behaviour.

### Functional Requirements

- **FR-001**: The repository MUST contain an `empirical external dataset` dedicated to calculation-layer validation.
- **FR-002**: The empirical external dataset MUST be stored in a clear, simplified, human-readable format that exposes activity inputs and expected outcome references without hidden transformation logic.
- **FR-003**: The empirical external dataset MUST contain at least 150 activities.
- **FR-004**: The empirical external dataset MUST span at least 3 source-calendar years.
- **FR-005**: The empirical external dataset MUST include activity shapes needed to validate FIFO, LIFO, HIFO, Average Cost Basis, and Scope-Local Hybrid (`scope_local_hybrid`).
- **FR-006**: The empirical external dataset MUST include acquisitions, partial liquidations, full liquidations, gains, losses, zero-result liquidations, fees, same-source-calendar-date ordering, pre-year opening positions, in-year activity, after-year ignored activity, full liquidation followed by reacquisition, and assets excluded from selected-year main results.
- **FR-007**: The empirical external dataset MUST include reliable scoped activity and unreliable or unavailable scoped activity so Scope-Local Hybrid (`scope_local_hybrid`) can validate narrowing, broadening, fallback activation, fallback carry-forward until zero, same-scope reset after zero, and independent other-scope state.
- **FR-008**: ~~The empirical external dataset MUST include zero-priced holding reductions with explanations. These rows MUST reduce quantity and basis under the selected method and MUST NOT create proceeds, realized gain, realized loss, or priced-liquidation treatment.~~ Superseded for empirical external-oracle scope by BUG-001: zero-priced holding reduction behavior MUST be covered by non-oracle unit, integration, or contract tests and MUST NOT be counted as supported external-oracle fixture coverage.
- **FR-009**: The empirical external dataset MUST include cases that require non-terminating division or proportional allocation so shared internal precision behaviour is exercised.
- **FR-010**: The empirical external dataset MUST include deterministic source IDs and stable ordering fields sufficient to reproduce project calculation order and ~~hledger~~ external-oracle input order.
- **FR-011**: The empirical calculation oracle MUST use ~~hledger~~ a pinned rotki-based test-time external oracle adapter as the external calculation source for FIFO, LIFO, HIFO, and Average Cost aggregate supported cases.
- **FR-012**: The empirical calculation oracle MUST record the ~~hledger version and command line~~ external oracle name, source URL, pinned version or commit, adapter arguments, and constraints used to produce each oracle output.
- **FR-013**: The empirical calculation oracle MUST transform ~~hledger~~ external-oracle output into an assertable normalized format rather than comparing human-formatted terminal output directly inside integration tests.
- **FR-014**: The empirical calculation oracle MUST include input and output hashes so oracle results are reproducible and drift is visible.
- **FR-015**: The empirical calculation oracle MUST reject or mark unsupported any dataset case that cannot be represented faithfully in ~~hledger~~ the selected external oracle or composite oracle without changing the case's financial meaning.
- **FR-016**: The empirical calculation oracle MUST persist normalized oracle output as repository golden fixtures used by empirical solidified financial tests.
- **FR-017**: The empirical solidified financial tests MUST read existing golden fixtures by default and MUST invoke ~~hledger-backed~~ external-oracle-backed generation only when the required golden fixture is absent.
- **FR-018**: The empirical solidified financial tests MUST be integration tests and MUST NOT replace existing unit, contract, integration, coverage, or performance verification requirements.
- **FR-019**: The empirical solidified financial tests MUST be implemented in a completely isolated Go test package dedicated to empirical financial validation. The expected package path is `tests/empirical`, separate from `tests/integration`.
- **FR-020**: The empirical solidified financial tests MUST run the project calculation layer from translated dataset records.
- **FR-021**: The empirical solidified financial tests MUST compare pure normalized calculation results only and MUST NOT assert Markdown, report document structure, TUI text, saved filenames, output paths, or other report-format boundaries.
- **FR-022**: The empirical solidified financial tests MUST compare per-year and per-method outputs for realized gain or loss, allocated basis, closing quantity, closing basis, full-liquidation effects, and method-specific matching evidence when those fields are comparable under the comparability requirements in this specification.
- **FR-023**: The empirical solidified financial tests MUST report comparison failures with enough context to identify the exact dataset row or calculation segment that drifted.
- **FR-024**: After the dataset is introduced by this dataset-maintenance work, ordinary feature work MUST treat it as read-only and MUST adapt code or project-owned tests around it instead of mutating it.
- **FR-025**: Average Cost empirical comparisons MUST compare aggregate realized gain or loss, allocated basis, closing quantity, and closing basis only until project-compatible pool provenance exists.
- **FR-026**: HIFO empirical fixtures MUST be rotki-backed and include at least one deterministic non-zero-priced tie-break case.
- **FR-027**: Scope-Local Hybrid empirical comparisons MUST use a separate composite oracle that labels each assertion as rotki-backed arithmetic or a documented project-owned composition rule.
- **FR-028**: Empirical tests MUST fail when a supported fixture group is skipped before project calculation and oracle comparison. Unsupported field-level segments MAY be skipped only when fixture metadata records an explicit reason.
- **FR-029**: The explicit golden-fixture regeneration command MUST download or reuse pinned rotki source in an untracked project-local folder, verify configured source URL, pinned version or commit, source checksum, and adapter constraints, and execute rotki calculation code through the project-owned local adapter boundary.
- **FR-030**: Golden fixture regeneration MUST reject committed raw rotki outputs, hand-authored rotki datasets, developer-global rotki installations, and vendored rotki source as the source of regenerated oracle data.
- **FR-031**: Normal fixture-backed empirical test runs MUST read committed normalized golden fixtures without downloading rotki source, requiring rotki source availability, or invoking oracle generation.

### Comparability Requirements

- **CMP-001**: A result field is comparable only when the oracle fixture contains a normalized expected value for the same case, method, year, asset, and source-row segment, and no unsupported segment covers that field.
- **CMP-002**: Full-liquidation effects and method-specific lot or pool evidence are comparable only when the oracle fixture records the source IDs, evidence type, and expected values used for that assertion.
- **CMP-003**: Scope-Local Hybrid assertions MUST label each comparison as ~~`hledger_backed`~~ `rotki_backed` or `project_composition_rule`. A project-owned composition-rule assertion MUST include a stable rule ID and the source-row segment it covers. Assertions without this label are unsupported for external-oracle comparison and MUST be reported as skipped with a reason.
- **CMP-004**: A supported empirical fixture group MUST NOT be skipped before project calculation and oracle comparison. Only field-level unsupported segments may be skipped, and each skipped segment MUST include a fixture-backed reason.

### Precision Requirements

- **FIN-001**: The project calculation side MUST use exact decimal arithmetic and MUST NOT introduce floating-point math in empirical dataset translation, oracle normalization, or comparison.
- **FIN-002**: The preferred empirical comparison precision is this project's current shared internal report-calculation precision: 16 decimal places, round half up for required non-terminating divisions and proportional allocations.
- **FIN-003**: The oracle MUST first attempt to normalize ~~hledger-derived~~ external-oracle-derived values to the same 16-decimal precision before comparison.
- **FIN-004**: If ~~hledger~~ the selected external oracle cannot be configured or normalized to align with the 16-decimal internal precision for every otherwise valid empirical case, the implementation MUST add a test-scoped environment-variable configuration path for the project internal report-calculation decimal policy. The expected variable is `GHOSTFOLIO_CRYPTOGAINS_REPORT_DECIMAL_POLICY`; empirical tests MUST set it to the external-oracle-established decimal policy for those runs, production behavior MUST keep the 16-decimal default when the variable is unset, and the accepted variable values MUST be documented before use.
- **FIN-005**: Quantity fields MUST compare by exact decimal equality after normalization under the selected decimal policy, and quantity tolerance MUST be `0`. Calculated financial value fields MUST first compare under the selected decimal policy. Non-zero financial tolerances MAY be used only for documented residual ~~hledger/project~~ external-oracle/project deviations that remain after decimal-policy alignment, MUST be declared per field in oracle metadata, and MUST NOT exceed one unit at the selected decimal-policy scale (`0.0000000000000001` for the 16-decimal policy). A non-zero tolerance MUST NOT be used for a field unless the fixture documents why exact equality is not achievable for that external-oracle-derived value.
- **FIN-006**: Cross-currency conversion remains permanently out of scope for empirical solidified financial tests. The empirical dataset and all empirical test cases MUST use a single currency only, and currency conversion logic MUST NOT be added to this suite.
- **FIN-007**: If future product calculation work introduces currency conversion, that conversion MUST be covered by separate project-owned tests or a separate dedicated empirical suite. It MUST NOT be folded into the empirical solidified financial tests defined here.

### Security And Persistence Requirements

- **SEC-001**: The empirical external dataset MUST NOT contain real user activity, real tokens, bearer JWTs, personally identifying account names, or proprietary financial records.
- **SEC-002**: The empirical external dataset MUST use synthetic assets, synthetic accounts or wallets, synthetic source identifiers, and synthetic timestamps.
- **SEC-003**: Oracle outputs and empirical test artifacts MUST NOT be written to protected application storage, user Documents folders, or OS-specific application config directories.
- **SEC-004**: Oracle outputs persisted in the repository MUST be synthetic, reproducible from the dataset and documented oracle command, and reviewable as non-secret golden fixtures.
- **SEC-005**: Oracle outputs and empirical fixtures MUST NOT embed upstream ~~hledger~~ external-oracle examples, documentation text, or test fixture content. ~~hledger-generated~~ External-oracle-generated output is acceptable only when normalized to project-owned synthetic dataset results.

### Dependency And External Tool Requirements

- **DEP-001**: ~~hledger is a repository-vendored test-time tool for this empirical validation scope. Runtime application code MUST NOT depend on hledger.~~ Superseded by BUG-001: external oracle tools are test-time only; runtime application code MUST NOT depend on hledger, rotki, oracle adapters, or composite oracle helpers.
- **DEP-002**: ~~hledger vendoring MUST comply with hledger's GPL-3.0-or-later license and this repository's GPLv3 license. Binary-only vendoring is prohibited.~~ External oracle vendoring MUST document applicable license compatibility with this GPLv3 repository. Binary-only vendoring remains prohibited unless the applicable license and source-distribution obligations are satisfied and documented.
- **DEP-003**: ~~The repository MUST include complete corresponding hledger source under `third_party/hledger/source/`, plus upstream license text, copyright notices, source version, source URL, and checksum. Any supported platform executable artifact MUST be committed under `third_party/hledger/bin/<goos>-<goarch>/hledger` with a matching checksum. Empirical tests and oracle generation MUST NOT fetch or build hledger during test execution.~~ Any retained hledger materials and any new rotki materials MUST include applicable license text, source provenance, pinned version or commit, source URL, checksums, supported artifact paths, and platform support notes. ~~Empirical tests and oracle generation MUST NOT fetch external oracle source or artifacts during test execution.~~ Normal fixture-backed empirical tests MUST NOT fetch external oracle source or artifacts during test execution; the explicit golden-fixture regeneration command MAY download pinned rotki source into an untracked project-local folder after verifying configured provenance and checksums.
- **DEP-004**: The empirical oracle MUST invoke ~~hledger as a separate test-time command-line tool~~ the selected external oracle through a separate test-time boundary and MUST NOT link, import, or embed hledger, hledger-lib, rotki, or rotki runtime code into runtime application code.
- **DEP-005**: The ~~vendored hledger executable-and-source model, supported version, version-detection command~~ selected oracle vendoring or adapter model, supported version or commit, version-detection or provenance check, platform support, license compliance notes, failure modes, and reproducibility implications MUST be documented before implementation.
- **DEP-006**: The empirical test command MUST require the repository-vendored ~~hledger executable under `third_party/hledger/bin/<goos>-<goarch>/hledger`~~ external oracle artifact or adapter source only when a required golden fixture is absent and generation is needed. Fixture-backed tests MAY run on platforms without a supported oracle executable. Fixture generation MUST fail with an actionable setup error when the required oracle boundary is missing, not executable on the current platform, reports an unsupported version or commit, cannot download or reuse the configured untracked rotki source path, or fails provenance and checksum verification.
- **DEP-007**: The rotki-based oracle adapter MUST be pinned to a documented source version or commit, MUST record adapter constraints for FIFO, LIFO, HIFO, and Average Cost aggregate comparisons, and MUST exclude zero-priced holding reductions from external-oracle fixture generation.
- **DEP-008**: ~~BUG-001 remediation MUST establish a repository-controlled rotki boundary before rotki-backed fixture normalization is considered complete. Acceptable evidence is either vendored or otherwise pinned rotki materials sufficient for documented execution, or committed raw rotki outputs plus provenance metadata under `testdata/empirical/rotki/`. Developer-local or CI-global rotki installations MUST NOT be the only source of truth.~~ Superseded by BUG-002: rotki-backed fixture normalization is complete only when explicit regeneration downloads or reuses verified pinned rotki source in an untracked project-local folder, executes rotki calculation code through the project-owned local adapter boundary, and commits only project-owned normalized golden fixtures plus provenance metadata. Vendored rotki source, committed raw rotki outputs, hand-authored rotki datasets, and developer-local or CI-global rotki installations MUST NOT be the source of regenerated oracle data.

## Dataset Coverage Requirements

The empirical external dataset MUST include at least these categories:

- Multi-year opening history before the selected report year.
- Selected-year liquidations that span one lot and multiple lots.
- Post-selected-year activity that must be ignored for the selected report year.
- FIFO cases where oldest acquisitions are consumed first.
- LIFO cases where newest acquisitions are consumed first.
- HIFO cases where highest unit-cost acquisitions are consumed first, including deterministic tie-breaking.
- Average Cost Basis cases with multiple acquisitions, partial disposal, full disposal, pool reset after zero, and reacquisition.
- Scope-local reliable cases where matching stays inside one wallet or account scope.
- Scope-local broadened cases where missing, partial, contradictory, or unsupported scope data broadens to asset-level calculation.
- Scope-local fallback cases where exact identification stops being defensible and fallback remains active until the scope reaches zero.
- ~~Zero-priced holding reductions with explicit zero-valued source fields and with missing optional source fields.~~ Zero-priced holding reductions are excluded from empirical external-oracle coverage by BUG-001 and remain covered by non-oracle tests.
- Fees that affect acquisition basis or liquidation proceeds for priced activity.
- Loss cases, gain cases, zero gain-or-loss cases, and negative yearly totals.
- Same-source-calendar-date acquisition and disposal ordering cases.
- Activity requiring rounded internal division or allocation under the selected empirical decimal policy.

## Key Entities

- **EmpiricalDataset**: The complete synthetic ledger and metadata used for empirical financial validation. Key attributes include dataset version, source activity rows, expected supported methods, supported years, coverage tags, and source hash.
- **EmpiricalActivity**: One synthetic input activity. Key attributes include source ID, occurred-at timestamp, activity type, asset identity key, display label, quantity, monetary values, fee amount, currency labels where needed, source scope, and zero-priced reduction explanation.
- **EmpiricalScope**: Synthetic source grouping for scope-local tests. Key attributes include scope ID, scope kind, reliability, and optional display name.
- **OracleInputLedger**: The ~~hledger-compatible~~ external-oracle-compatible representation derived from the empirical dataset for one method or method family.
- **OracleOutput**: The normalized assertable expected result derived from ~~hledger~~ external-oracle or composite-oracle output and oracle normalization.
- **ProjectCalculationOutput**: The normalized result produced by this project's calculation layer from the same empirical dataset.
- **EmpiricalComparisonResult**: The per-field comparison between oracle and project outputs, including selected decimal policy, tolerance, difference, and diagnostic context.

## Out Of Scope

- User-facing UI, TUI workflow, or Markdown report changes.
- Ghostfolio API calls.
- Protected snapshot encryption or token unlock behaviour.
- Real user portfolio data.
- Cross-currency exchange-rate conversion.
- Currency conversion testing inside the empirical solidified financial test suite.
- Replacing the existing unit, contract, integration, coverage, or performance suites.
- Markdown report, report document, TUI, filename, or filesystem-output assertions.
- Copying upstream ~~hledger~~ external-oracle, Ledger, or Beancount fixtures verbatim into this repository without a license review.
- Vendoring ~~hledger~~ external oracle artifacts in binary-only form or without the required license text, provenance, and corresponding source where the applicable license requires it.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Dataset validation confirms at least 150 activities across at least 3 source-calendar years.
- **SC-002**: Dataset validation confirms coverage tags for every supported project cost-basis method and every required dataset coverage category in this specification.
- **SC-003**: The oracle produces deterministic normalized golden fixture output for every supported empirical case when run with the documented ~~hledger version and command set~~ external oracle identity, adapter constraints, and command set, and empirical tests do not run oracle generation while the required golden fixtures are present. For rotki-backed regeneration, the command executes verified pinned rotki source from an untracked project-local folder and records enough metadata to reproduce or detect drift in that source execution path.
- **SC-004**: Empirical solidified financial integration tests compare project calculation output against oracle output for every supported method and every comparable field, report actionable differences when comparison fails, and fail if supported fixture groups are skipped before project calculation and oracle comparison.
- **SC-005**: ~~Zero-priced holding reduction cases prove quantity and basis are reduced while proceeds, realized gain, and realized loss remain zero or absent according to the normalized comparison contract.~~ Zero-priced holding reduction behavior is verified by non-oracle unit, integration, or contract tests and is not counted as empirical external-oracle coverage.
- **SC-006**: Precision-sensitive cases match exactly under either the 16-decimal production default policy or an explicitly documented empirical-test-only decimal-policy configuration that matches ~~hledger's~~ the selected external oracle's established policy.
- **SC-007**: Vendored ~~hledger~~ external-oracle materials include applicable license notices, corresponding source or provenance required by the applicable license, version or commit identity, source URL, checksums for source and any supported executable artifact, and empirical tests use external oracle tooling only as a separate test-time boundary. Rotki source itself is not vendored; explicit regeneration verifies the pinned source download or reuse path, while normal fixture-backed empirical tests do not fetch rotki source.

## Assumptions

- ~~hledger is the primary external oracle because it directly supports FIFO, LIFO, HIFO, and AVERAGE lot reduction with gain postings.~~ Superseded by BUG-001: hledger is not sufficient for acceptance because observed empirical runs skipped most supported fixtures; rotki is the planned external oracle for FIFO, LIFO, HIFO, and Average Cost aggregate comparisons.
- ~~hledger and hledger-lib are published as GPL-3.0-or-later packages, which is compatible with this GPLv3 repository when vendored with license notices and corresponding source.~~ External oracle license compatibility, including rotki source licensing and any retained hledger materials, must be documented with source provenance before fixture regeneration.
- Beancount and Ledger remain useful research references, but they are not the primary oracle for all supported methods.
- Scope-Local Hybrid (`scope_local_hybrid`) has no exact single upstream equivalent, so the oracle may combine ~~hledger-backed~~ rotki-backed arithmetic sub-oracle evidence with project-owned hybrid lifecycle assertions.
- Dataset activities are synthetic and can be designed specifically to avoid upstream license inheritance from copied fixture text.
- The empirical suite relies on repository-controlled external oracle artifacts or pinned adapter source instead of system installations. For rotki-backed regeneration, the repository controls provenance, checksums, adapter constraints, and committed normalized fixtures, while rotki source is downloaded or reused only in an untracked project-local folder during explicit regeneration.
