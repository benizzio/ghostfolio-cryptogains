# Specification Quality Checklist: Report Base Currency Conversion

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-06-20
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

- Validation passed on the first iteration.
- No clarification markers remain.
- The spec deliberately defers official source access details to planning research while fixing the user-visible authority, currency, conversion boundary, audit, and failure rules.

## Requirements Unit-Test Addendum

**Purpose**: Validate the clarity, completeness, consistency, and measurability of report base-currency conversion requirements before implementation review.
**Created**: 2026-06-21
**Feature**: [spec.md](../spec.md)
**Focus Areas**: Conversion correctness and auditability; official provider evidence and safe-failure behavior
**Depth**: Standard
**Actor/Timing**: PR reviewer before implementation review

## Requirement Completeness

- [x] CHK001 Are report base-currency selection requirements complete for selection, supported options, and calculation gating? [Completeness, Spec §FR-001-FR-003]
- [x] CHK002 Are selected activity monetary-context requirements complete enough to prevent mixing order, asset-profile, and base tiers before conversion? [Completeness, Spec §FR-005-FR-006]
- [x] CHK003 Are conversion boundary requirements complete for every monetary value that can affect basis, proceeds, fees, gains, losses, and totals? [Completeness, Spec §FR-008-FR-009]
- [x] CHK004 Are report audit requirements complete for both same-currency and converted activity values? [Completeness, Spec §FR-019-FR-021]

## Requirement Clarity

- [x] CHK005 Is "exactly one report base currency" unambiguous about whether the choice applies per report run, per report year, or per reporting session? [Clarity, Spec §FR-001]
- [x] CHK006 Is "priced activity" defined clearly enough to distinguish rows requiring conversion from zero-priced holding reductions? [Ambiguity, Spec §FR-007-FR-008, Spec §FR-022]
- [x] CHK007 Is "officially trusted or authorized" defined with objective evidence criteria for ECB and Federal Reserve sources? [Clarity, Spec §FR-010-FR-011, Plan §Official Rate Source Decisions]
- [x] CHK008 Are "source-calendar date" and timestamp-offset derivation specified with enough precision to avoid machine-local date ambiguity? [Clarity, Spec §FR-013, Spec §Assumptions]

## Requirement Consistency

- [x] CHK009 Do the spec and plan consistently limit report base currencies to USD and EUR without implying future base currencies are user-selectable now? [Consistency, Spec §FR-002, Plan §Scale/Scope]
- [x] CHK010 Are cache reuse requirements consistent between the clarification, assumptions, and no-persistence storage plan? [Consistency, Spec §Clarifications, Spec §Assumptions, Plan §Technical Context]
- [x] CHK011 Are safe-failure and success criteria requirements consistent about when no partial cleartext report artifact may remain? [Consistency, Spec §FR-027, Spec §SC-004, Plan §Failure Handling]
- [x] CHK012 Are rounding and exact decimal requirements consistent between feature requirements, financial evidence, and the implementation plan? [Consistency, Spec §FR-024-FR-026, Plan §Conversion Boundary And Rounding]

## Acceptance Criteria Quality

- [x] CHK013 Do success criteria define measurable coverage for mixed-currency reports, converted activity audit entries, failures, and regression cases? [Acceptance Criteria, Spec §SC-001-SC-007]
- [x] CHK014 Is the "under 30 seconds" outcome scoped well enough to avoid dependence on unspecified dataset size, terminal environment, or provider availability? [Measurability, Spec §SC-005]
- [x] CHK015 Does the deterministic dataset success criterion define enough dataset composition to measure source-currency, report-year, and rounding coverage? [Acceptance Criteria, Spec §SC-002]
- [x] CHK016 Are "100%" success criteria tied to clearly identified scenario sets so the requirement is objectively auditable? [Measurability, Spec §SC-001-SC-004, Spec §SC-006-SC-007]

## Scenario Coverage

- [x] CHK017 Are primary scenarios complete for choosing USD and EUR report base currencies and producing separate report outcomes? [Coverage, Spec §User Story 1, Spec §Edge Cases]
- [x] CHK018 Are alternate scenarios defined for same-currency rows that bypass conversion while preserving prior single-currency results? [Coverage, Spec §FR-007, Spec §SC-006]
- [x] CHK019 Are exception scenarios complete for unavailable providers, unsupported source currencies, malformed currency identity, and missing authoritative rates? [Coverage, Spec §User Story 3, Spec §Edge Cases]
- [x] CHK020 Are recovery-context requirements specified for keeping the user inside the unlocked reporting context after conversion failure? [Coverage, Spec §FR-027]

## Edge Case Coverage

- [x] CHK021 Are requirements defined for activities with fees and gross values that must share one selected currency context before conversion? [Edge Case, Spec §Edge Cases]
- [x] CHK022 Are requirements defined for explicit zero-valued monetary fields so zero remains valid without creating monetary effects? [Edge Case, Spec §FR-023]
- [x] CHK023 Are non-publication-day requirements clear for weekends, public holidays, and other missing-rate dates without assuming every previous business day has data? [Edge Case, Spec §FR-015, Spec §Edge Cases]
- [x] CHK024 Are deterministic-history failure requirements defined for a late conversion failure after earlier conversion evidence has already been collected? [Edge Case, Spec §Edge Cases]

## Non-Functional Requirements

- [x] CHK025 Are precision requirements specific enough to prohibit floating-point financial decisions across amounts, rates, converted values, and assertions? [Non-Functional, Spec §FR-024]
- [x] CHK026 Are security requirements complete for token exclusion, financial-value redaction, and cleartext report audit disclosure boundaries? [Security, Spec §FR-028-FR-029, Spec §Assumptions]
- [x] CHK027 Are performance requirements complete for responsive provider lookup and calculation across the stated 10,000-activity scale target? [Performance, Plan §Performance Goals]
- [x] CHK028 Are persistence requirements complete for no exchange-rate disk cache and in-memory session cache lifecycle boundaries? [Persistence, Spec §Assumptions, Plan §Technical Context]

## Dependencies & Assumptions

- [x] CHK029 Are assumptions about preserved activity currency identity documented and tied to existing sync data contracts? [Assumption, Spec §Assumptions, Spec §FR-005]
- [x] CHK030 Are provider authority relationships documented enough to distinguish primary official sources from merely unofficial compatible datasets? [Dependency, Spec §FR-010-FR-011, Plan §Official Rate Source Decisions]
- [x] CHK031 Are assumptions about empirical dataset immutability and new project-owned conversion coverage explicit enough to avoid repurposing existing oracle fixtures? [Assumption, Spec §Financial Calculation Evidence, Plan §Testing Strategy]
- [x] CHK032 Are dependencies on outbound HTTPS access to fixed official-provider hosts documented as environmental prerequisites for manual or live report generation? [Dependency, Plan §Target Platform, Plan §Constraints]

## Ambiguities & Conflicts

- [x] CHK033 Is there any unresolved ambiguity between "daily reference or closing rate" and provider-specific rate kinds such as ECB reference rates and Federal Reserve H.10 noon buying rates? [Ambiguity, Spec §FR-014, Plan §Official Rate Source Decisions]
- [x] CHK034 Is the supported source-currency set defined or intentionally delegated to provider research outputs? [Gap, Spec §FR-016, Spec §FR-030]
- [x] CHK035 Is the report audit detail granularity clear when one activity has multiple converted monetary values such as gross amount and fee? [Ambiguity, Spec §FR-020]
- [x] CHK036 Is the boundary between report-domain requirements and provider-integration implementation details clearly separated in the plan? [Architecture, Plan §Integration Anticorruption Layer]
