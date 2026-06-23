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

## [ ] 2 - Currency Conversion Audit cleanup and fixes:

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
| Date | Source ID | Asset | Source Currency | Report Base Currency | Rate Date | Rate Authority | Rate Kind | Quote Direction | Rate Value | Amount Kind | Original Amount | Converted Amount |
|------|-----------|-------|-----------------|----------------------|-----------|----------------|-----------|-----------------|------------|-------------|-----------------|------------------|
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | unit_price | 8334.372169710421 | 7448.7194295383153097 |
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | gross_value | 185.7031469366197 | 165.9693868412009116 |
| 2019-12-30 | 80698e22-be25-4fe1-869d-094d775854d2 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | fee_amount | 0 | 0 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | unit_price | 4910.542616239068 | 4388.7234035562320136 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | gross_value | 1064.5861443464435 | 951.4578106590790062 |
| 2019-12-30 | f06965d0-9951-4e1e-b99e-af8905dd4817 | BTCUSD | USD | EUR | 2019-12-30 | European Central Bank | daily euro foreign exchange reference rate | source_per_base | 1.1189 | fee_amount | 0 | 0 |
```

Problems with the example:

- in the case of the fee ammount it is audiding a conversion from 0 to 0. This is unnecessary bloat. Only real conversions should be done and audited
- the fact that it is converting bout `unit_price` and `gross_value` seems excessive, as only one of those values is needed to calculate the capital gains and cost basis changes, and the application just needs to convert exactly the values that it will use and audit only those conversions. The rules of what values to use have been established by other specs
