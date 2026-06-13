package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
	"github.com/cockroachdb/apd/v3"
)

const oracleDecimalPolicy = "scale=16,rounding=half_up"

// hledgerJournalOracleData stores the hledger print and balance views required
// to build one or more oracle fixtures from a rendered case journal.
// Authored by: OpenCode
type hledgerJournalOracleData struct {
	balanceCommandArguments []string
	balanceRows             []hledgerBalanceAccountRow
	printCommandArguments   []string
	printTransactions       []hledgerPrintTransaction
}

// hledgerPrintTransaction stores the hledger JSON print fields required by the
// oracle normalizer.
// Authored by: OpenCode
type hledgerPrintTransaction struct {
	Date        string           `json:"tdate"`
	Description string           `json:"tdescription"`
	Postings    []hledgerPosting `json:"tpostings"`
}

// hledgerPosting stores the hledger JSON posting fields required by the oracle
// normalizer.
// Authored by: OpenCode
type hledgerPosting struct {
	Account string          `json:"paccount"`
	Amounts []hledgerAmount `json:"pamount"`
	Tags    [][]string      `json:"ptags"`
}

// hledgerAmount stores the amount and cost-basis fields required by oracle
// normalization.
// Authored by: OpenCode
type hledgerAmount struct {
	Commodity string               `json:"acommodity"`
	Cost      *hledgerTaggedAmount `json:"acost"`
	CostBasis *hledgerCostBasis    `json:"acostbasis"`
	Quantity  hledgerDecimal       `json:"aquantity"`
}

// hledgerTaggedAmount stores one tagged hledger amount payload such as a unit
// sale price.
// Authored by: OpenCode
type hledgerTaggedAmount struct {
	Contents hledgerRawAmount `json:"contents"`
	Tag      string           `json:"tag"`
}

// hledgerCostBasis stores the hledger JSON acquisition basis metadata attached
// to a lotful amount.
// Authored by: OpenCode
type hledgerCostBasis struct {
	Cost  hledgerRawAmount `json:"cbCost"`
	Label string           `json:"cbLabel"`
}

// hledgerRawAmount stores the raw commodity and decimal fields for nested cost
// amounts.
// Authored by: OpenCode
type hledgerRawAmount struct {
	Commodity string         `json:"acommodity"`
	Quantity  hledgerDecimal `json:"aquantity"`
}

// hledgerDecimal stores the exact mantissa and scale emitted by hledger JSON.
// Authored by: OpenCode
type hledgerDecimal struct {
	Mantissa int64 `json:"decimalMantissa"`
	Places   int   `json:"decimalPlaces"`
}

// hledgerBalanceAccountRow stores the account-name and amount columns emitted by
// hledger JSON balance reports.
// Authored by: OpenCode
type hledgerBalanceAccountRow struct {
	Account string
	Amounts []hledgerAmount
}

// UnmarshalJSON decodes one positional JSON balance row into the account and
// amount fields required by the oracle generator.
// Authored by: OpenCode
func (row *hledgerBalanceAccountRow) UnmarshalJSON(content []byte) error {
	var payload []json.RawMessage
	if err := json.Unmarshal(content, &payload); err != nil {
		return fmt.Errorf("decode hledger balance row: %w", err)
	}
	if len(payload) < 4 {
		return fmt.Errorf("decode hledger balance row: expected at least 4 fields, got %d", len(payload))
	}

	if err := json.Unmarshal(payload[0], &row.Account); err != nil {
		return fmt.Errorf("decode hledger balance row account: %w", err)
	}
	if err := json.Unmarshal(payload[3], &row.Amounts); err != nil {
		return fmt.Errorf("decode hledger balance row amounts: %w", err)
	}

	return nil
}

// collectHledgerJournalOracleData runs the vendored hledger print and balance
// commands needed to derive oracle fixtures for one case journal.
// Authored by: OpenCode
func collectHledgerJournalOracleData(
	ctx context.Context,
	command vendoredHledgerCommand,
	journalRelativePath string,
	year int,
) (hledgerJournalOracleData, error) {
	var printCommandArguments = oraclePrintCommandArguments(journalRelativePath, year)
	var rawPrintOutput, err = runVendoredHledgerCommand(ctx, command, journalRelativePath, oraclePrintSubcommandArguments(year)...)
	if err != nil {
		return hledgerJournalOracleData{}, err
	}

	var printTransactions []hledgerPrintTransaction
	printTransactions, err = parseHledgerPrintTransactions(rawPrintOutput)
	if err != nil {
		return hledgerJournalOracleData{}, err
	}

	var balanceCommandArguments = oracleClosingBalanceCommandArguments(journalRelativePath, year)
	var rawBalanceOutput []byte
	rawBalanceOutput, err = runVendoredHledgerCommand(ctx, command, journalRelativePath, oracleClosingBalanceSubcommandArguments(year)...)
	if err != nil {
		return hledgerJournalOracleData{}, err
	}

	var balanceRows []hledgerBalanceAccountRow
	balanceRows, err = parseHledgerBalanceRows(rawBalanceOutput)
	if err != nil {
		return hledgerJournalOracleData{}, err
	}

	return hledgerJournalOracleData{
		balanceCommandArguments: balanceCommandArguments,
		balanceRows:             balanceRows,
		printCommandArguments:   printCommandArguments,
		printTransactions:       printTransactions,
	}, nil
}

// buildOracleOutputForAsset derives one normalized oracle fixture for one case,
// method, and target asset from already-collected hledger data.
// Authored by: OpenCode
func buildOracleOutputForAsset(
	dataset fixture.EmpiricalDataset,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	assetIdentityKey string,
	hledgerVersion string,
	renderedJournal journal,
	journalRelativePath string,
	oracleData hledgerJournalOracleData,
) (fixture.OracleOutput, error) {
	var values comparableOutputValuesInput
	var matches []oracleMatchEvidenceInput
	var err error
	values, matches, err = buildOracleComparableValues(assetIdentityKey, oracleData)
	if err != nil {
		return fixture.OracleOutput{}, err
	}

	var unsupportedSegments = buildUnsupportedSegments(dataset, empiricalCase, method, assetIdentityKey, renderedJournal.ledger.GenerationNotes, matches)

	return normalizeOracleOutput(oracleOutputNormalizationInput{
		DatasetVersion:      strings.TrimSpace(dataset.DatasetVersion),
		CaseID:              strings.TrimSpace(empiricalCase.CaseID),
		Method:              method,
		Year:                empiricalCase.Year,
		AssetIdentityKey:    strings.TrimSpace(assetIdentityKey),
		Values:              values,
		Matches:             matches,
		UnsupportedSegments: unsupportedSegments,
		Metadata: oracleGenerationMetadataInput{
			RunID:                   "",
			OracleName:              "hledger",
			SourceURL:               "https://github.com/simonmichael/hledger",
			SourceChecksum:          stablePrefixedSHA256Hash([]byte("https://github.com/simonmichael/hledger@" + strings.TrimSpace(hledgerVersion))),
			VersionOrCommit:         strings.TrimSpace(hledgerVersion),
			AdapterArguments:        oracleCommandProvenanceArguments(oracleData),
			AdapterConstraints:      []string{"retained historical hledger boundary"},
			DatasetInputHash:        strings.TrimSpace(renderedJournal.ledger.DatasetInputHash),
			ExternalOracleInputHash: strings.TrimSpace(renderedJournal.ledger.ExternalOracleInputHash),
			DecimalPolicy:           oracleDecimalPolicy,
			FinancialTolerances: map[string]string{
				"realized_gain_or_loss": "0",
				"allocated_basis":       "0",
				"closing_basis":         "0",
			},
			ToleranceNotes: map[string]string{},
			GeneratedAt:    "",
		},
	})
}

// buildOracleComparableValues derives one asset-scoped comparable value block
// and its hledger-backed match evidence.
// Authored by: OpenCode
func buildOracleComparableValues(assetIdentityKey string, oracleData hledgerJournalOracleData) (comparableOutputValuesInput, []oracleMatchEvidenceInput, error) {
	var realizedGainOrLoss = supportmath.Zero()
	var allocatedBasis = supportmath.Zero()
	var matches = make([]oracleMatchEvidenceInput, 0)
	var transactionIndex int

	for transactionIndex = range oracleData.printTransactions {
		var transactionMatches []oracleMatchEvidenceInput
		var transactionBasis apd.Decimal
		var err error
		transactionMatches, transactionBasis, err = transactionAssetMatches(oracleData.printTransactions[transactionIndex], assetIdentityKey)
		if err != nil {
			return comparableOutputValuesInput{}, nil, err
		}
		if len(transactionMatches) == 0 {
			continue
		}

		allocatedBasis, err = supportmath.Add(allocatedBasis, transactionBasis)
		if err != nil {
			return comparableOutputValuesInput{}, nil, fmt.Errorf("sum allocated basis for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
		}

		var transactionGainOrLoss apd.Decimal
		transactionGainOrLoss, err = transactionRealizedGainOrLoss(oracleData.printTransactions[transactionIndex])
		if err != nil {
			return comparableOutputValuesInput{}, nil, err
		}
		realizedGainOrLoss, err = supportmath.Add(realizedGainOrLoss, transactionGainOrLoss)
		if err != nil {
			return comparableOutputValuesInput{}, nil, fmt.Errorf("sum realized gain or loss for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
		}

		matches = append(matches, transactionMatches...)
	}

	var closingQuantity apd.Decimal
	var closingBasis apd.Decimal
	var err error
	closingQuantity, closingBasis, err = closingBalanceForAsset(assetIdentityKey, oracleData.balanceRows)
	if err != nil {
		return comparableOutputValuesInput{}, nil, err
	}

	var realizedGainOrLossText string
	realizedGainOrLossText, err = decimalsupport.CanonicalString(realizedGainOrLoss)
	if err != nil {
		return comparableOutputValuesInput{}, nil, fmt.Errorf("format realized gain or loss for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
	}

	var allocatedBasisText string
	allocatedBasisText, err = decimalsupport.CanonicalString(allocatedBasis)
	if err != nil {
		return comparableOutputValuesInput{}, nil, fmt.Errorf("format allocated basis for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
	}

	var closingQuantityText string
	closingQuantityText, err = decimalsupport.CanonicalString(closingQuantity)
	if err != nil {
		return comparableOutputValuesInput{}, nil, fmt.Errorf("format closing quantity for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
	}

	var closingBasisText string
	closingBasisText, err = decimalsupport.CanonicalString(closingBasis)
	if err != nil {
		return comparableOutputValuesInput{}, nil, fmt.Errorf("format closing basis for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
	}

	return comparableOutputValuesInput{
		RealizedGainOrLoss: realizedGainOrLossText,
		AllocatedBasis:     allocatedBasisText,
		ClosingQuantity:    closingQuantityText,
		ClosingBasis:       closingBasisText,
	}, matches, nil
}

// transactionAssetMatches extracts one hledger-backed match per target-asset
// dispose posting fragment from one printed transaction.
// Authored by: OpenCode
func transactionAssetMatches(transaction hledgerPrintTransaction, assetIdentityKey string) ([]oracleMatchEvidenceInput, apd.Decimal, error) {
	var matches = make([]oracleMatchEvidenceInput, 0)
	var allocatedBasis = supportmath.Zero()
	var postingIndex int

	for postingIndex = range transaction.Postings {
		if postingTagValue(transaction.Postings[postingIndex], "_ptype") != "dispose" {
			continue
		}
		if accountAssetIdentityKey(transaction.Postings[postingIndex].Account) != strings.TrimSpace(assetIdentityKey) {
			continue
		}

		var match oracleMatchEvidenceInput
		var matchBasis apd.Decimal
		var err error
		match, matchBasis, err = oracleMatchFromPosting(transaction, transaction.Postings[postingIndex])
		if err != nil {
			return nil, apd.Decimal{}, err
		}

		allocatedBasis, err = supportmath.Add(allocatedBasis, matchBasis)
		if err != nil {
			return nil, apd.Decimal{}, fmt.Errorf("sum matched basis for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
		}
		matches = append(matches, match)
	}

	return matches, allocatedBasis, nil
}

// oracleMatchFromPosting derives one normalized comparable match from one
// target-asset dispose posting fragment.
// Authored by: OpenCode
func oracleMatchFromPosting(transaction hledgerPrintTransaction, posting hledgerPosting) (oracleMatchEvidenceInput, apd.Decimal, error) {
	var amount, found = assetPostingAmount(posting)
	if !found {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("transaction %q posting %q has no amounts", strings.TrimSpace(transaction.Description), strings.TrimSpace(posting.Account))
	}

	var matchedQuantity, err = absoluteHledgerAmountQuantity(amount)
	if err != nil {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, err
	}

	var unitBasis apd.Decimal
	unitBasis, err = hledgerAmountBasisUnit(amount)
	if err != nil {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, err
	}

	var matchedBasis apd.Decimal
	matchedBasis, err = supportmath.Multiply(matchedQuantity, unitBasis)
	if err != nil {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("derive matched basis for posting %q: %w", strings.TrimSpace(posting.Account), err)
	}

	var matchedProceeds = supportmath.Zero()
	var matchedGainOrLoss = supportmath.Zero()
	var hasExplicitProceeds bool
	if amount.Cost != nil {
		var unitProceeds apd.Decimal
		unitProceeds, err = amount.Cost.Contents.Quantity.toDecimal()
		if err != nil {
			return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("derive matched proceeds for posting %q: %w", strings.TrimSpace(posting.Account), err)
		}

		matchedProceeds, err = supportmath.Multiply(matchedQuantity, unitProceeds)
		if err != nil {
			return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("derive matched proceeds total for posting %q: %w", strings.TrimSpace(posting.Account), err)
		}
		matchedGainOrLoss, err = supportmath.Subtract(matchedProceeds, matchedBasis)
		if err != nil {
			return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("derive matched gain or loss for posting %q: %w", strings.TrimSpace(posting.Account), err)
		}
		hasExplicitProceeds = true
	}

	var disposedSourceID = strings.TrimSpace(postingTagValue(posting, "posting_source_id"))
	if disposedSourceID == "" {
		disposedSourceID = zeroPricedReductionDisposedSourceID(transaction.Description)
	}
	if disposedSourceID == "" {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("transaction %q posting %q is missing posting_source_id evidence", strings.TrimSpace(transaction.Description), strings.TrimSpace(posting.Account))
	}

	var matchedQuantityText string
	matchedQuantityText, err = decimalsupport.CanonicalString(matchedQuantity)
	if err != nil {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("format matched quantity for posting %q: %w", strings.TrimSpace(posting.Account), err)
	}

	var matchedBasisText string
	matchedBasisText, err = decimalsupport.CanonicalString(matchedBasis)
	if err != nil {
		return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("format matched basis for posting %q: %w", strings.TrimSpace(posting.Account), err)
	}

	var matchedProceedsText = "0"
	var matchedGainOrLossText = "0"
	if hasExplicitProceeds {
		matchedProceedsText, err = decimalsupport.CanonicalString(matchedProceeds)
		if err != nil {
			return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("format matched proceeds for posting %q: %w", strings.TrimSpace(posting.Account), err)
		}
		matchedGainOrLossText, err = decimalsupport.CanonicalString(matchedGainOrLoss)
		if err != nil {
			return oracleMatchEvidenceInput{}, apd.Decimal{}, fmt.Errorf("format matched gain or loss for posting %q: %w", strings.TrimSpace(posting.Account), err)
		}
	}

	return oracleMatchEvidenceInput{
		DisposedSourceID:    disposedSourceID,
		AcquisitionSourceID: strings.TrimSpace(hledgerAmountBasisLabel(amount)),
		ScopeID:             recoverScopeIDFromAccount(posting.Account),
		MatchedQuantity:     matchedQuantityText,
		MatchedBasis:        matchedBasisText,
		MatchedProceeds:     matchedProceedsText,
		MatchedGainOrLoss:   matchedGainOrLossText,
		SupportLabel:        fixture.EvidenceSupportLabelRotkiBacked,
	}, matchedBasis, nil
}

// assetPostingAmount returns the first amount on an asset posting that still
// carries asset-side quantity and lot metadata.
// Authored by: OpenCode
func assetPostingAmount(posting hledgerPosting) (hledgerAmount, bool) {
	var amountIndex int
	for amountIndex = range posting.Amounts {
		if posting.Amounts[amountIndex].CostBasis != nil {
			return posting.Amounts[amountIndex], true
		}
	}
	if len(posting.Amounts) == 0 {
		return hledgerAmount{}, false
	}

	return posting.Amounts[0], true
}

// transactionRealizedGainOrLoss sums the matching transaction's rgain postings
// and converts hledger revenue sign convention into report sign convention.
// Authored by: OpenCode
func transactionRealizedGainOrLoss(transaction hledgerPrintTransaction) (apd.Decimal, error) {
	var total = supportmath.Zero()
	var postingIndex int

	for postingIndex = range transaction.Postings {
		if postingTagValue(transaction.Postings[postingIndex], "_ptype") != "rgain" {
			continue
		}

		var amountIndex int
		for amountIndex = range transaction.Postings[postingIndex].Amounts {
			var amount apd.Decimal
			var err error
			amount, err = transaction.Postings[postingIndex].Amounts[amountIndex].Quantity.toDecimal()
			if err != nil {
				return apd.Decimal{}, fmt.Errorf("parse rgain amount in transaction %q: %w", strings.TrimSpace(transaction.Description), err)
			}

			var realizedContribution apd.Decimal
			realizedContribution, err = negateDecimal(amount)
			if err != nil {
				return apd.Decimal{}, fmt.Errorf("invert rgain amount in transaction %q: %w", strings.TrimSpace(transaction.Description), err)
			}

			total, err = supportmath.Add(total, realizedContribution)
			if err != nil {
				return apd.Decimal{}, fmt.Errorf("sum rgain amounts in transaction %q: %w", strings.TrimSpace(transaction.Description), err)
			}
		}
	}

	return total, nil
}

// closingBalanceForAsset derives the target asset's end-of-year closing quantity
// and basis from hledger balance rows.
// Authored by: OpenCode
func closingBalanceForAsset(assetIdentityKey string, rows []hledgerBalanceAccountRow) (apd.Decimal, apd.Decimal, error) {
	var closingQuantity = supportmath.Zero()
	var closingBasis = supportmath.Zero()
	var rowIndex int

	for rowIndex = range rows {
		if accountAssetIdentityKey(rows[rowIndex].Account) != strings.TrimSpace(assetIdentityKey) {
			continue
		}

		var amountIndex int
		for amountIndex = range rows[rowIndex].Amounts {
			var quantity apd.Decimal
			var err error
			quantity, err = rows[rowIndex].Amounts[amountIndex].Quantity.toDecimal()
			if err != nil {
				return apd.Decimal{}, apd.Decimal{}, fmt.Errorf("parse balance quantity for account %q: %w", strings.TrimSpace(rows[rowIndex].Account), err)
			}

			closingQuantity, err = supportmath.Add(closingQuantity, quantity)
			if err != nil {
				return apd.Decimal{}, apd.Decimal{}, fmt.Errorf("sum closing quantity for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
			}

			var unitBasis apd.Decimal
			unitBasis, err = hledgerAmountBasisUnit(rows[rowIndex].Amounts[amountIndex])
			if err != nil {
				return apd.Decimal{}, apd.Decimal{}, err
			}

			var basisContribution apd.Decimal
			basisContribution, err = supportmath.Multiply(quantity, unitBasis)
			if err != nil {
				return apd.Decimal{}, apd.Decimal{}, fmt.Errorf("derive closing basis contribution for account %q: %w", strings.TrimSpace(rows[rowIndex].Account), err)
			}

			closingBasis, err = supportmath.Add(closingBasis, basisContribution)
			if err != nil {
				return apd.Decimal{}, apd.Decimal{}, fmt.Errorf("sum closing basis for asset %q: %w", strings.TrimSpace(assetIdentityKey), err)
			}
		}
	}

	return closingQuantity, closingBasis, nil
}

// parseHledgerPrintTransactions decodes the vendored hledger JSON print output.
// Authored by: OpenCode
func parseHledgerPrintTransactions(content []byte) ([]hledgerPrintTransaction, error) {
	var transactions []hledgerPrintTransaction
	if err := json.Unmarshal(content, &transactions); err != nil {
		return nil, fmt.Errorf("decode hledger print JSON: %w", err)
	}

	return transactions, nil
}

// parseHledgerBalanceRows decodes the vendored hledger JSON balance output into
// the emitted account rows.
// Authored by: OpenCode
func parseHledgerBalanceRows(content []byte) ([]hledgerBalanceAccountRow, error) {
	var payload []json.RawMessage
	if err := json.Unmarshal(content, &payload); err != nil {
		return nil, fmt.Errorf("decode hledger balance JSON: %w", err)
	}
	if len(payload) != 2 {
		return nil, fmt.Errorf("decode hledger balance JSON: expected 2 top-level elements, got %d", len(payload))
	}

	var rows []hledgerBalanceAccountRow
	if err := json.Unmarshal(payload[0], &rows); err != nil {
		return nil, fmt.Errorf("decode hledger balance rows: %w", err)
	}

	return rows, nil
}

// runVendoredHledgerCommand builds and executes one vendored hledger command for
// the supplied journal path and subcommand arguments.
// Authored by: OpenCode
func runVendoredHledgerCommand(
	ctx context.Context,
	command vendoredHledgerCommand,
	journalRelativePath string,
	args ...string,
) ([]byte, error) {
	var cmd, err = command.buildCommand(ctx, journalRelativePath, args...)
	if err != nil {
		return nil, fmt.Errorf("build vendored hledger command for %s: %w", journalRelativePath, err)
	}

	var output, runErr = cmd.CombinedOutput()
	if runErr == nil {
		return output, nil
	}

	var renderedArgs = strings.Join(args, " ")
	var trimmedOutput = strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return nil, fmt.Errorf("run vendored hledger %s: %w", renderedArgs, runErr)
	}

	return nil, fmt.Errorf("run vendored hledger %s: %w: %s", renderedArgs, runErr, trimmedOutput)
}

// oraclePrintCommandArguments returns the persisted repository-relative print
// command arguments recorded in fixture metadata.
// Authored by: OpenCode
func oraclePrintCommandArguments(journalRelativePath string, year int) []string {
	var arguments = []string{"-n", "-f", strings.TrimSpace(journalRelativePath)}
	arguments = append(arguments, oraclePrintSubcommandArguments(year)...)
	return arguments
}

// oracleClosingBalanceCommandArguments returns the persisted repository-relative balance command arguments recorded in fixture metadata.
// Authored by: OpenCode
func oracleClosingBalanceCommandArguments(journalRelativePath string, year int) []string {
	var arguments = []string{"-n", "-f", strings.TrimSpace(journalRelativePath)}
	arguments = append(arguments, oracleClosingBalanceSubcommandArguments(year)...)
	return arguments
}

// oracleCommandProvenanceArguments records both print and balance command inputs in stable metadata order.
// Authored by: OpenCode
func oracleCommandProvenanceArguments(oracleData hledgerJournalOracleData) []string {
	var arguments = make([]string, 0, len(oracleData.printCommandArguments)+len(oracleData.balanceCommandArguments)+1)
	arguments = append(arguments, copyStringSlice(oracleData.printCommandArguments)...)
	if len(oracleData.printCommandArguments) != 0 && len(oracleData.balanceCommandArguments) != 0 {
		arguments = append(arguments, "--next-command--")
	}
	arguments = append(arguments, copyStringSlice(oracleData.balanceCommandArguments)...)
	return arguments
}

// oraclePrintSubcommandArguments returns the wrapper-managed print subcommand
// arguments used to derive year-scoped oracle data.
// Authored by: OpenCode
func oraclePrintSubcommandArguments(year int) []string {
	return []string{
		"print",
		"--lots",
		"--explicit",
		"-O",
		"json",
		"-b",
		fmt.Sprintf("%04d-01-01", year),
		"-e",
		fmt.Sprintf("%04d-01-01", year+1),
	}
}

// oracleClosingBalanceSubcommandArguments returns the historical end-balance
// query used to derive closing quantity and cost-basis state.
// Authored by: OpenCode
func oracleClosingBalanceSubcommandArguments(year int) []string {
	return []string{
		"balance",
		"-H",
		"--lots",
		"-O",
		"json",
		"-e",
		fmt.Sprintf("%04d-01-01", year+1),
		"assets:empirical",
	}
}

// postingTagValue returns the value for one hledger posting tag key.
// Authored by: OpenCode
func postingTagValue(posting hledgerPosting, key string) string {
	var tagIndex int
	for tagIndex = range posting.Tags {
		if len(posting.Tags[tagIndex]) < 2 {
			continue
		}
		if posting.Tags[tagIndex][0] != key {
			continue
		}

		return strings.TrimSpace(posting.Tags[tagIndex][1])
	}

	return ""
}

// accountAssetIdentityKey recovers one synthetic asset identity key from a
// rendered empirical asset account path.
// Authored by: OpenCode
func accountAssetIdentityKey(account string) string {
	var parts = strings.Split(strings.TrimSpace(account), ":")
	if len(parts) < 4 {
		return ""
	}
	if parts[0] != "assets" || parts[1] != "empirical" {
		return ""
	}

	if parts[2] == reportmodel.CostBasisMethodScopeLocalHybrid.FilenameSlug() {
		if len(parts) < 5 {
			return ""
		}

		return strings.TrimSpace(parts[4])
	}

	return strings.TrimSpace(parts[3])
}

// recoverScopeIDFromAccount returns the reliable scope identifier embedded in a
// scope-local-hybrid account path when one is directly recoverable.
// Authored by: OpenCode
func recoverScopeIDFromAccount(account string) string {
	var parts = strings.Split(strings.TrimSpace(account), ":")
	if len(parts) < 5 {
		return ""
	}
	if parts[0] != "assets" || parts[1] != "empirical" || parts[2] != reportmodel.CostBasisMethodScopeLocalHybrid.FilenameSlug() {
		return ""
	}
	if strings.TrimSpace(parts[3]) == "fallback" {
		return ""
	}

	return strings.TrimSpace(parts[3])
}

// absoluteHledgerAmountQuantity returns the absolute quantity represented by one
// hledger amount.
// Authored by: OpenCode
func absoluteHledgerAmountQuantity(amount hledgerAmount) (apd.Decimal, error) {
	var quantity, err = amount.Quantity.toDecimal()
	if err != nil {
		return apd.Decimal{}, err
	}
	if quantity.Sign() >= 0 {
		return supportmath.Clone(quantity), nil
	}

	return negateDecimal(quantity)
}

// hledgerAmountBasisUnit returns the per-unit basis embedded in one hledger lot
// amount.
// Authored by: OpenCode
func hledgerAmountBasisUnit(amount hledgerAmount) (apd.Decimal, error) {
	if amount.CostBasis == nil {
		return apd.Decimal{}, fmt.Errorf("hledger amount is missing cost-basis metadata")
	}

	return amount.CostBasis.Cost.Quantity.toDecimal()
}

// hledgerAmountBasisLabel returns the acquisition source label attached to one
// hledger lot amount when it is present.
// Authored by: OpenCode
func hledgerAmountBasisLabel(amount hledgerAmount) string {
	if amount.CostBasis == nil {
		return ""
	}

	return strings.TrimSpace(amount.CostBasis.Label)
}

// zeroPricedReductionDisposedSourceID falls back to the transaction description
// for native zero-priced reductions, whose generated postings do not yet carry a
// posting_source_id tag.
// Authored by: OpenCode
func zeroPricedReductionDisposedSourceID(description string) string {
	var trimmedDescription = strings.TrimSpace(description)
	if !strings.HasPrefix(trimmedDescription, "zero-priced reduction ") {
		return ""
	}

	var remainder = strings.TrimPrefix(trimmedDescription, "zero-priced reduction ")
	var sourceID, _, found = strings.Cut(remainder, " from ")
	if !found {
		return ""
	}

	return strings.TrimSpace(sourceID)
}

// toDecimal converts one hledger mantissa and decimal-place pair into a finite
// exact decimal.
// Authored by: OpenCode
func (decimalValue hledgerDecimal) toDecimal() (apd.Decimal, error) {
	var sign = ""
	var magnitude = decimalValue.Mantissa
	if magnitude < 0 {
		sign = "-"
		magnitude = -magnitude
	}

	var digits = strconv.FormatInt(magnitude, 10)
	if decimalValue.Places == 0 {
		var parsed, _, err = decimalsupport.ParseString(sign + digits)
		if err != nil {
			return apd.Decimal{}, fmt.Errorf("parse hledger decimal %s%s: %w", sign, digits, err)
		}

		return parsed, nil
	}

	if len(digits) <= decimalValue.Places {
		digits = strings.Repeat("0", decimalValue.Places-len(digits)+1) + digits
	}

	var pointIndex = len(digits) - decimalValue.Places
	var raw = sign + digits[:pointIndex] + "." + digits[pointIndex:]
	var parsed, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("parse hledger decimal %s: %w", raw, err)
	}

	return parsed, nil
}
