Adaptations to the plan for the gains reporting feature needed:

- currency conversion must at the moment be ignored as it will be added in a future spec. for now calculations should be done as the base currencies have all the same price.
- concept of wallet: as the current version og Ghostfolio will not provide the concept of wallet, we will use the account domain concept to consider a wallet if the method of capital gains selected requires it
- attempts to not use HTTPS in production should be blocked with an error
- The go version in the plan is behind. The research must include the latest go version and relate to the used libraries. If the current latest version cannot be used, the research must have a clear justification.
- chriptographic storage is a key concern for security, so let's add a reference to it in an amendment to the constitution, making sure we reference OWASPs directive that must be followed from (https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html).
- for the selection of ghostfolio server to get activity data, default must be cloud currently available at https://ghostfol.io, self-hosted server can be entered to replace it.
  - make sure to that the plan research guarantees that the cloud server of ghostfolio has the public API available and functional according to ghostfolio source.

Remove this file after the plan and constitution updates.