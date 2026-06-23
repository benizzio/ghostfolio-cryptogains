## [X] 1 - Rate Source Summary repeating itself:

The `Rate Source Summary` section is repeating multiple times in a unwanted behavior. Here is a production extraction:

```markdown
## Rate Source Summary

- Report Base Currency: EUR
- Authority: European Central Bank
- Provider: ECB Data Portal `EXR`
- Rate Kind: daily euro foreign exchange reference rate
- Unavailable-Date Rule: most recent previous available ECB observation
- Quote Direction: source_per_base
- Rate Value: 1.1189
- Authority: European Central Bank
- Provider: ECB Data Portal `EXR`
- Rate Kind: daily euro foreign exchange reference rate
- Unavailable-Date Rule: most recent previous available ECB observation
- Quote Direction: source_per_base
- Rate Value: 1.1336
- Authority: European Central Bank
- Provider: ECB Data Portal `EXR`
```

Since the report has only one base currency, this segment cannot repeat. Also, the `Quote Direction` `Rate Value` fields are unnecessary for this segment and should be removed

## [X] 2 - Currency Conversion Audit cleanup and fixes:

The current `Currency Conversion Audit` table is too large with unnecessary information. We can remove the following columns:

- `Rate Authority`
- `Rate Kind`

The order of the columns is also not intuitive. We need to change for the following order:

```markdown
| Date | Source ID | Asset | Amount Kind | Rate Date | Source Currency | Original Amount  | Report Base Currency | Converted Amount | Quote Direction | Rate Value |
```

## [ ] 3 - Currency Conversion Audit bloat

According to the audit multiple unnecessary conversions are happening. Example extracted from production data:

```markdown
| Date | Source ID | Asset | Amount Kind | Rate Date | Source Currency | Original Amount | Report Base Currency | Converted Amount | Quote Direction | Rate Value |
|------|-----------|-------|-------------|-----------|-----------------|-----------------|----------------------|------------------|-----------------|------------|
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | unit_price | 2019-12-30 | USD | 8334.372169710421 | EUR | 7448.7194295383153097 | source_per_base | 1.1189 |
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | gross_value | 2019-12-30 | USD | 185.7031469366197 | EUR | 165.9693868412009116 | source_per_base | 1.1189 |
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | fee_amount | 2019-12-30 | USD | 0 | EUR | 0 | source_per_base | 1.1189 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | unit_price | 2019-12-30 | USD | 4910.542616239068 | EUR | 4388.7234035562320136 | source_per_base | 1.1189 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | gross_value | 2019-12-30 | USD | 1064.5861443464435 | EUR | 951.4578106590790062 | source_per_base | 1.1189 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | fee_amount | 2019-12-30 | USD | 0 | EUR | 0 | source_per_base | 1.1189 |
| 2020-03-11 | 8b2288ad-0420-41b5-98a6-7c0c28cb3ba1 | BTCUSD | unit_price | 2020-03-11 | USD | 5401.138393784537 | EUR | 4764.5892676292669372 | source_per_base | 1.1336 |
| 2020-03-11 | 8b2288ad-0420-41b5-98a6-7c0c28cb3ba1 | BTCUSD | gross_value | 2020-03-11 | USD | 51.776122813577636 | EUR | 45.6740674078842943 | source_per_base | 1.1336 |
| 2020-03-11 | 8b2288ad-0420-41b5-98a6-7c0c28cb3ba1 | BTCUSD | fee_amount | 2020-03-11 | USD | 0 | EUR | 0 | source_per_base | 1.1336 |
```

Problems with the example:

- in the case of the fee ammount it is audiding a conversion from 0 to 0. This is unnecessary bloat. Only real conversions should be audited
- showing one individual line for each ammount kind with the same `Source ID` is confusing and bloats the table. We should have one line per source, so we need to desing a clear way to show all relevant "kinds" of the same source compacted in the same row
