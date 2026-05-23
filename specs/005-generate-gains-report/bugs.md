# Bugs found to report

- [X] clarified the explained zero-priced holding-reduction docs so preserved explicit zero `unit_price`, `gross_value`, and `fee_amount` remain distinct from missing values, while `activity_currency` stays blank because no activity monetary context is required
- [X] specs/005-generate-gains-report/spec.md:135 mentions the field is named `symbolProfileId` but it's actually just `id`, so the full path from activity is `SymbolProfile.id`
- [X] in the `Sync Data` UI screen, the Ghostfolio Security Token field is still being show. As it is now only needed in the unlock screen and stays in the context, it should not appear internally anymore
- [X] inside the `Sync and Reports` UI screen, when `Generate Capital Gains Report` is disabled, and we are selecting an option with the arrow keys, tha cursor still "navigates" through the disabled option, which is a bit confusing. It should skip the disabled option and only navigate through the enabled options
- [X] when trying to unlock local protected synced data with a new Security Token that can't decrypt any existing local data, the application presents the normal `Sync and Reports` options (for a new datastore). The requirement from previous slice of only allowing further actions if the Ghostfolio server authenticates the security token successfully should be kept. If a token can't authenticate on the Ghostfolio server we should show an error message in the unlock screen already that informs "access denied" and clear the field to allow informing another token. We should allowin a sync of a new protected cache of a new user only if the user is successfully authenticated in the ghostfolio server
- [X] report generation error should also generate the failure diagnostics report file, that should print information about the report generation error and, if the error is related to a specific activity, original activity data should be included to trace the issue to the original datasource
  - when printing activity data in the diagnostics reports, always use the original persisted data as it is the closest to the original Ghostfolio source and makes it simple to understand input that generated the error
  - make sure to print null values as well as it helps understand the errors. The current sync diagnostic report is not printing null values, so it should be changed
- [X] we currently have the following error report with production data:
  ```json
    {
      "schema_version": 1,
      "generated_at": "2026-05-23T13:02:11.027298886Z",
      "failure_category": "unsupported report calculation",
      "server_origin": "https://ghostfol.io",
      "explicit_development_mode": true,
      "financial_values_redacted": false,
      "attempt": {
        "attempt_id": "attempt-1779541331026220964",
        "status": "failed",
        "started_at": "2026-05-23T13:02:11.026220434Z",
        "completed_at": "2026-05-23T13:02:11.027257866Z",
        "server_mismatch_confirmed": false
      },
      "failure_detail": "activity \"ae8c8ca0-0feb-4303-bc66-c9e7296e02e6\" order currency context is incomplete; provide gross value and fee from that tier only (asset \"BTCUSD\", source \"ae8c8ca0-0feb-4303-bc66-c9e7296e02e6\")",
      "offending_activity_record": {
        "source_id": "ae8c8ca0-0feb-4303-bc66-c9e7296e02e6",
        "occurred_at": "2019-12-30T23:00:00.000Z",
        "activity_type": "BUY",
        "asset_identity_key": "0714ee32-4438-4734-ac99-659e5bb092e8",
        "asset_symbol": "BTCUSD",
        "asset_name": "Bitcoin USD",
        "quantity": "0.0222816",
        "order_currency": null,
        "order_unit_price": "8334.372169710421",
        "order_gross_value": "185.7031469366197",
        "order_fee_amount": "0",
        "asset_profile_currency": "USD",
        "asset_profile_unit_price": "8334.372169710421",
        "asset_profile_fee_amount": "0",
        "base_currency": "EUR",
        "base_gross_value": "165.7771992703204",
        "base_fee_amount": "0",
        "comment": null,
        "data_source": null,
        "source_scope": {
          "id": "24f2c5ed-c7c8-4802-aa25-f18395640308",
          "name": "Cryptofolio",
          "kind": "account",
          "reliability": "reliable"
        },
        "raw_hash": "157e22a7a82b9c2e25b39bee41e026194098cfecde940225542c7f77f71961d6"
      }
    }
  ```
  - this means there is a misconception of the spec: picking a "single-activity currency context in priority order `order -> asset -> base`" should mean that if the currency information is missing for the first tier in the priority order, we should skip it try the next one, and only fail if currency information is missing from all the tiers. This also means ONLY using the financial information of the tier that correctly provides the currency
  - failure should only occur when a tier that properly informed is currency did not provide the needed financial values that are required to do calculations and generate the report, ONLY if the values can't be derived (e.g. obtaining gross value from quantity and unit price, unit price from gross value and quantity, etc.) 
- [ ] the application is currently keeping the token in memory in cleartext and a memory dump of the application process would reveal the token. We need to add a security layer to make sure this is not possible while still being able to keep and use the token for later requests (research the best method to do it)