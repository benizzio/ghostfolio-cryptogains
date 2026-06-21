# Modifications in the planning phase I want done:

- regarding source code, adding all related to integration and anticorruption layer in `internal/report/exchange/` will bloat the report module too much. Let's create a new layer on the architecture called `integration`.
  - we will keep all code related to exchange rate integration and cannonicalization in `internal/integration/currency/`. This package will keep the anticorr and client layer, being responsible to fetch API data and return it cannonicalized via a public application service function
  - the `internal/report` package can then implement calling the application service to obtain cannonicalized data and perform the needed calculations
  - we can create a github issue for moving current ghostfolio sync integration implementation to the same archetype (at `internal/integration/ghostfolio`) in a future PR, without the need to an anticorruption layer as the source is unique.

- testing for the integration layer: we will also create a new category of tests - "external integration tests". 
  - These tests will be applied only directly to the HTTP client layer, to verify the exchange rate providers API is working acording to our client expectations. The test must simply choose load one fixed instace of historical data were the exchange rates are known and commited in the project, request the providers for it, assert the return and, with that, verify if the HTTP layer still works as expected. To avoid excessive HTTP load on the providers, only this single record verification for each unique client endpoint is needed.

- in the `rate-provider-integration.md` contract, some improvements are needed:
  - the canonical lookup contract seems to be missing the financial value in the source currency that will be converted
  - foe the "In-Memory Report-Run Cache", we can state that the cache is maintained in memory while the TUI session is executed, between multiple report runs and even with different security tokens
