# About the user stories:

- in `User Story 1`, the current structure of the usage flow may have too much breaking changes for the implementation. Let's reformulate it:
  - The current application's "Sync Data" menu becomes "Sync and Reports"
    - Inside this menu, we have two options: "Sync Data" (previously "Start Sync") and "Generate Capital Gains Report". Both will depend on a context of unlocking the data with the security token
    - The previous applied rules are moved to this second contextual menu.
- in `User Story 2`:
  - the requirement to return to the home menu is now invalid. We should return to the menu inside the "Sync and Reports" option to keep consistency and avoid the need to re-inform the token
  - let's make sure to mention that the yearly gains/losses reports considers only liquidations INSIDE the selected year
  - we can add the acceptance scenario that if the asset is first bought after the selected year it is simply ignored in the report
  - we can add the acceptance scenario that if the asset is fully liquidated before or in selected year and a new position is opened before or inside the selected year, the report considers the gains/losses only of the liquidations inside the current year and adds a counter of full liquidations until the end of the current year in the reference section
- change and adapt ALL FUNCTIONAL REQUIREMENTS tied to these definitions
- change and adapt ALL SECURITY, PRECISION, AND INTEGRATION CONSTRAINTS tied to these definitions

# About edge cases:

- A selected year contains acquisitions and holding reductions but no taxable disposals, producing a valid report with zero realized gain or loss.
  - the application should make NO DISTINCTION of what is or is not taxable.
- The synced dataset contains mixed currency labels across activities; for this slice the report must still choose the first available currency tier per activity and treat the chosen values as equal without conversion.
  - the report must show that the main final report currency is "NO CURRENCY APPLIES, ALL CONSIDERED EQUAL"

# About Functional requirements:

- from FR-022 to FR-025, we can have the references here, but we need a bigger section that explains each mathematical formula in a more detailed way, without tying to implementation details. The functional requirements can then point to this section.
- We need a new one: local currency selection inside one activity is used to choose all needed values (unit price, fee, total value, gross value, etc) from that single-activity-currency-context and keep consistency. Inside the single-activity-currency-context, this is ALWAYS required. After we exit the single-activity-currency-context and the value enters the cost basis calculation and the gain/loss calculation context, the application simply treats all currencies as equal and considers the currency of the realm as "NO CURRENCY APPLIES, ALL CONSIDERED EQUAL" for this slice
  - FR-031: this requirement needs to be referenced in it, as it is a complement. It's rules are valid inside the "single-activity-currency-context"
  - FR-032: is also related to this, let's reference them
- FR-031: "When monetary amounts are needed for reporting" is not precise. "When monetary amounts are needed for calculations"
- FR-029: The system MUST treat zero-priced disposal records that represent fees or transfers out as holding reductions that remove basis under the selected cost basis method without creating realized gain or loss entries in the report.
  - This is correct, but let's mention that this comes from previous sync rules that consider the zero-priced disposals with comments. Inspect the previous specs to find and reference it.

# About Security, Precision, and Integration Constraints

- we need a new FIN-XXX definition for what I mentioned in previous "single-activity-currency-context" requirements
  - move all related to single-activity-currency-context from FIN-001 to this new one.

# General changes:

- we need to reinforce wherever is appropriate that both GAINS and LOSSES are considered in the report, and losses are shown with a negative sign in the final report
- FR-010: The system MUST follow the reference report template structure with sections in this order: gains-and-losses summary, reference section for previously liquidated assets, then per-asset detail sections.
  - this is correct but we need to find some place in the spec where we define in a general form what is each of these sections. Remove any assumptions and keep the structure well defined