# Checklist Validation Remediation: Report Base Currency Conversion

**Date**: 2026-06-21
**Checklist**: `checklists/requirements.md`
**Scope**: Unchecked items from the requirements unit-test addendum were validated against `spec.md`, `plan.md`, `research.md`, `data-model.md`, `quickstart.md`, and the feature contracts.

## Summary

All 36 addendum items are fulfilled by the current spec package and have been marked complete in the checklist. The following 6 items were remediated because the previous artifacts did not make the requirement objectively auditable or internally precise enough.

## Remediated Items

### CHK014 - Acceptance Criterion Timing Scope

**Checklist item**: Is the "under 30 seconds" outcome scoped well enough to avoid dependence on unspecified dataset size, terminal environment, or provider availability? [Measurability, Spec §SC-005]

**Problem**: `SC-005` scopes the outcome to already synced reportable data and generation confirmation, which avoids provider availability. It still depends on unspecified human interaction speed, terminal environment, and dataset shape. This makes the 30-second target difficult to verify objectively.

**Recommended solution**: Rewrite `SC-005` as a system-observable TUI responsiveness requirement, or define a concrete validation fixture. If the time target remains, specify the dataset size, provider-call boundary, input path, and validation environment. A safer criterion would assert that base-currency selection and request confirmation do not perform provider lookup and remain responsive with the 10,000-activity scale fixture.

**Resolution**: `spec.md` replaces the time-based `SC-005` with the named 10,000-Activity Responsiveness Fixture, and `plan.md` adds performance validation for asynchronous behavior, delayed provider responses, and bounded lookup count.

### CHK015 - Deterministic Dataset Composition

**Checklist item**: Does the deterministic dataset success criterion define enough dataset composition to measure source-currency, report-year, and rounding coverage? [Acceptance Criteria, Spec §SC-002]

**Problem**: `SC-002` defines at least 50 priced activities, at least 3 source currencies, and 2 report years. It does not require data that exercises the rounding policy, such as division results requiring the 16-decimal round-half-up boundary. A dataset could satisfy the stated composition while not testing the most important rounding behavior.

**Recommended solution**: Amend `SC-002` or add a fixture requirement requiring at least one conversion that uses division and triggers bounded internal decimal rounding, at least one multiplication quote-direction conversion, and expected converted values that prove the documented rounding policy.

**Resolution**: `spec.md` defines the Deterministic Conversion Fixture with division, multiplication, previous-available-date, and 16-decimal round-half-up coverage; `quickstart.md` adds matching fixture expectations.

### CHK016 - `100%` Scenario-Set Traceability

**Checklist item**: Are "100%" success criteria tied to clearly identified scenario sets so the requirement is objectively auditable? [Measurability, Spec §SC-001-SC-004, Spec §SC-006-SC-007]

**Problem**: The success criteria use `100%` against broad groups such as acceptance testing, regression cases, and production-mode diagnostics. The current artifacts contain useful scenario detail, but the success criteria do not consistently tie each `100%` claim to a named scenario matrix, fixture suite, or regression set.

**Recommended solution**: Update the success criteria to reference named scenario sets. Suggested sets are a mixed-currency acceptance matrix, deterministic conversion fixture, conversion failure matrix, existing single-currency regression suite, and production diagnostic redaction fixture set.

**Resolution**: `spec.md` adds named validation scenario sets and rewrites `SC-001` through `SC-007` to reference those sets.

### CHK027 - Performance Validation Completeness

**Checklist item**: Are performance requirements complete for responsive provider lookup and calculation across the stated 10,000-activity scale target? [Performance, Plan §Performance Goals]

**Problem**: The plan states that the UI must remain responsive, provider lookups should be cached by `(base currency, source currency, activity source-calendar date)`, and the report scale target is up to 10,000 cached activities. It does not define a measurable performance validation condition for provider lookup or calculation at that scale.

**Recommended solution**: Add a performance validation requirement or plan test for the 10,000-activity fixture. The validation should assert asynchronous TUI behavior, no per-monetary-field network requests, bounded lookups by unique rate key, and successful calculation without blocking the event loop. If a time budget is added, state the fixture, disabled-live-provider setup, and acceptable measurement method.

**Resolution**: `plan.md` adds a Performance Validation section, and `quickstart.md` adds automated verification checks for the 10,000-activity responsiveness fixture and unique-rate-key lookup bound.

### CHK029 - Sync Currency Identity Traceability

**Checklist item**: Are assumptions about preserved activity currency identity documented and tied to existing sync data contracts? [Assumption, Spec §Assumptions, Spec §FR-005]

**Problem**: The spec states that activity currency identity is expected to use explicit currency codes already preserved by synced activity data. It does not identify the existing sync contract, activity model, or field-level source that preserves the order, asset-profile, and base currency identities consumed by selected activity monetary-context rules.

**Recommended solution**: Add a traceability note in the spec, data model, or plan identifying the existing synced activity fields or contract that preserve currency identity for each monetary tier. State whether the feature requires no sync persistence migration or whether any sync contract update is needed.

**Resolution**: `spec.md`, `plan.md`, and `data-model.md` now trace currency identity to the existing sync contracts and `ActivityRecord` tier fields, and state that no sync persistence migration is required.

### CHK033 - Provider Rate-Kind Terminology

**Checklist item**: Is there any unresolved ambiguity between "daily reference or closing rate" and provider-specific rate kinds such as ECB reference rates and Federal Reserve H.10 noon buying rates? [Ambiguity, Spec §FR-014, Plan §Official Rate Source Decisions]

**Problem**: `FR-014` and the related acceptance scenario use the phrase "daily reference or closing rate". The plan selects ECB daily reference rates for EUR and Federal Reserve H.10 noon buying rates for USD. H.10 noon buying rates are provider-specific daily rates, but they are not clearly covered by the spec wording.

**Recommended solution**: Reword `FR-014` to refer to the authority provider's single published daily rate for the selected activity date. Include examples such as ECB daily euro foreign exchange reference rate and Federal Reserve H.10 noon buying rate. Keep the requirement that the generated report identifies the provider-specific rate kind.

**Resolution**: `spec.md` rewrites `FR-014`, the related acceptance scenario, and financial calculation evidence to use provider-specific daily rate terminology with ECB and Federal Reserve examples.
