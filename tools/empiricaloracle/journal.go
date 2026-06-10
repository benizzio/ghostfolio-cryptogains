package main

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	reportbasis "github.com/benizzio/ghostfolio-cryptogains/internal/report/basis"
	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	supportmath "github.com/benizzio/ghostfolio-cryptogains/internal/support/math"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
	"github.com/cockroachdb/apd/v3"
)

const reliableHybridFIFOCaseID = "case-scope-local-reliable-epsilon-2024"

// journal stores one generated case-scoped hledger journal and its persisted
// ledger metadata.
// Authored by: OpenCode
type journal struct {
	ledger  fixture.OracleInputLedger
	content string
}

// journalLotTracker tracks open acquisition lots for native FIFO, LIFO, and
// HIFO zero-priced reduction rendering.
// Authored by: OpenCode
type journalLotTracker struct {
	method       reportbasis.LotMethod
	states       map[string]*reportbasis.LotMethodState
	acquisitions map[string]journalAcquisition
}

// journalAcquisition stores the rendered acquisition metadata needed to rebuild
// lot selectors for zero-priced reduction sink transfers.
// Authored by: OpenCode
type journalAcquisition struct {
	sourceID           string
	date               string
	occurredAt         time.Time
	deterministicOrder int
	unitBasis          apd.Decimal
}

// journalLotMatch stores one matched lot fragment selected for a zero-priced
// reduction sink transfer.
// Authored by: OpenCode
type journalLotMatch struct {
	acquisition journalAcquisition
	quantity    apd.Decimal
}

// renderJournals converts every dataset case and requested method into one
// deterministic case-scoped hledger journal.
// Authored by: OpenCode
func renderJournals(dataset fixture.EmpiricalDataset, rawDatasetContent string) ([]journal, error) {
	var outputs = make([]journal, 0)
	var caseIndex int

	for caseIndex = range dataset.Cases {
		if dataset.Cases[caseIndex].OracleSupport == fixture.OracleSupportUnsupported {
			continue
		}
		var methodIndex int
		for methodIndex = range dataset.Cases[caseIndex].Methods {
			var output, err = renderJournal(dataset, rawDatasetContent, dataset.Cases[caseIndex], dataset.Cases[caseIndex].Methods[methodIndex])
			if err != nil {
				return nil, err
			}

			outputs = append(outputs, output)
		}
	}

	sort.Slice(outputs, func(left int, right int) bool {
		return outputs[left].ledger.ExternalOracleInputPath < outputs[right].ledger.ExternalOracleInputPath
	})

	return outputs, nil
}

// renderJournal converts one dataset case and method into one deterministic
// case-scoped hledger journal.
// Authored by: OpenCode
func renderJournal(
	dataset fixture.EmpiricalDataset,
	rawDatasetContent string,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
) (journal, error) {
	if !caseHasMethod(empiricalCase, method) {
		return journal{}, fmt.Errorf("render journal %q %q: method is not declared on the case", strings.TrimSpace(empiricalCase.CaseID), strings.TrimSpace(string(method)))
	}

	var activities, err = selectJournalActivities(dataset, empiricalCase)
	if err != nil {
		return journal{}, err
	}

	var content string
	var generationNotes []string
	content, generationNotes, err = renderJournalContent(activities, empiricalCase, method)
	if err != nil {
		return journal{}, err
	}

	var journalPath = journalRelativePath(method, empiricalCase.CaseID)
	var copiedNotes = make([]string, len(generationNotes))
	copy(copiedNotes, generationNotes)

	return journal{
		ledger: fixture.OracleInputLedger{
			LedgerID:                journalLedgerID(method, empiricalCase.CaseID),
			Method:                  strings.TrimSpace(string(method)),
			CaseIDs:                 []string{strings.TrimSpace(empiricalCase.CaseID)},
			ExternalOracleInputPath: journalPath,
			DatasetInputHash:        stablePrefixedSHA256Hash([]byte(rawDatasetContent)),
			ExternalOracleInputHash: stablePrefixedSHA256Hash([]byte(content)),
			GenerationNotes:         copiedNotes,
		},
		content: content,
	}, nil
}

// caseHasMethod reports whether one case explicitly requests the provided
// cost-basis method.
// Authored by: OpenCode
func caseHasMethod(empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) bool {
	var methodIndex int

	for methodIndex = range empiricalCase.Methods {
		if empiricalCase.Methods[methodIndex] == method {
			return true
		}
	}

	return false
}

// selectJournalActivities chooses the historical case slice required by the
// phase-4 journal contract and returns it in deterministic journal order.
// Authored by: OpenCode
func selectJournalActivities(dataset fixture.EmpiricalDataset, empiricalCase fixture.EmpiricalCase) ([]fixture.EmpiricalActivity, error) {
	var activitiesBySourceID = make(map[string]fixture.EmpiricalActivity, len(dataset.Activities))
	var activityIndex int

	for activityIndex = range dataset.Activities {
		activitiesBySourceID[strings.TrimSpace(dataset.Activities[activityIndex].SourceID)] = dataset.Activities[activityIndex]
	}

	var caseAssets = make(map[string]struct{}, len(empiricalCase.AssetIdentityKeys))
	var assetIndex int
	for assetIndex = range empiricalCase.AssetIdentityKeys {
		caseAssets[strings.TrimSpace(empiricalCase.AssetIdentityKeys[assetIndex])] = struct{}{}
	}

	var latestReferencedByAsset = make(map[string]fixture.EmpiricalActivity, len(caseAssets))
	var sourceIndex int
	for sourceIndex = range empiricalCase.ActivitySourceIDs {
		var sourceID = strings.TrimSpace(empiricalCase.ActivitySourceIDs[sourceIndex])
		var activity, found = activitiesBySourceID[sourceID]
		if !found {
			return nil, fmt.Errorf("select journal activities %q: referenced activity %q is missing from the dataset", strings.TrimSpace(empiricalCase.CaseID), sourceID)
		}

		var assetKey = strings.TrimSpace(activity.AssetIdentityKey)
		if _, allowed := caseAssets[assetKey]; !allowed {
			return nil, fmt.Errorf(
				"select journal activities %q: referenced activity %q uses asset %q outside the case asset list",
				strings.TrimSpace(empiricalCase.CaseID),
				sourceID,
				assetKey,
			)
		}

		var latest, exists = latestReferencedByAsset[assetKey]
		if !exists || compareJournalActivities(activity, latest) > 0 {
			latestReferencedByAsset[assetKey] = activity
		}
	}

	for assetKey := range caseAssets {
		if _, found := latestReferencedByAsset[assetKey]; !found {
			return nil, fmt.Errorf("select journal activities %q: asset %q has no referenced activity", strings.TrimSpace(empiricalCase.CaseID), assetKey)
		}
	}

	var selected = make([]fixture.EmpiricalActivity, 0)
	for activityIndex = range dataset.Activities {
		var activity = dataset.Activities[activityIndex]
		var latest, found = latestReferencedByAsset[strings.TrimSpace(activity.AssetIdentityKey)]
		if !found {
			continue
		}

		if compareJournalActivities(activity, latest) <= 0 {
			selected = append(selected, activity)
		}
	}

	sort.Slice(selected, func(left int, right int) bool {
		return compareJournalActivities(selected[left], selected[right]) < 0
	})

	if len(selected) == 0 {
		return nil, fmt.Errorf("select journal activities %q: no journal activities were selected", strings.TrimSpace(empiricalCase.CaseID))
	}

	return selected, nil
}

// compareJournalActivities returns the deterministic ordering required for
// phase-4 journal rendering.
// Authored by: OpenCode
func compareJournalActivities(left fixture.EmpiricalActivity, right fixture.EmpiricalActivity) int {
	var leftOccurredAt = strings.TrimSpace(left.OccurredAt)
	var rightOccurredAt = strings.TrimSpace(right.OccurredAt)
	if leftOccurredAt < rightOccurredAt {
		return -1
	}
	if leftOccurredAt > rightOccurredAt {
		return 1
	}

	var leftAssetKey = strings.TrimSpace(left.AssetIdentityKey)
	var rightAssetKey = strings.TrimSpace(right.AssetIdentityKey)
	if leftAssetKey < rightAssetKey {
		return -1
	}
	if leftAssetKey > rightAssetKey {
		return 1
	}

	if left.DeterministicOrder < right.DeterministicOrder {
		return -1
	}
	if left.DeterministicOrder > right.DeterministicOrder {
		return 1
	}

	var leftSourceID = strings.TrimSpace(left.SourceID)
	var rightSourceID = strings.TrimSpace(right.SourceID)
	if leftSourceID < rightSourceID {
		return -1
	}
	if leftSourceID > rightSourceID {
		return 1
	}

	return 0
}

// renderJournalContent renders one case-scoped journal body and its generation
// notes without writing anything to disk.
// Authored by: OpenCode
func renderJournalContent(activities []fixture.EmpiricalActivity, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) (string, []string, error) {
	var transactions = make([]string, 0)
	var generationNotes = make([]string, 0)
	var tracker *journalLotTracker
	var err error

	var nativeMethod, native = nativeJournalLotMethod(method, empiricalCase.CaseID)
	if native {
		tracker, err = newJournalLotTracker(nativeMethod)
		if err != nil {
			return "", nil, fmt.Errorf("render journal content %q %q: %w", strings.TrimSpace(empiricalCase.CaseID), strings.TrimSpace(string(method)), err)
		}
	}

	var activityIndex int
	for activityIndex = range activities {
		var renderedTransactions []string
		var renderedNotes []string
		renderedTransactions, renderedNotes, err = renderJournalActivity(activities[activityIndex], empiricalCase, method, tracker)
		if err != nil {
			return "", nil, err
		}

		transactions = append(transactions, renderedTransactions...)
		generationNotes = append(generationNotes, renderedNotes...)
	}

	var sections = make([]string, 0, 2)
	var directives = renderCommodityDirectives(activities, journalLotMode(method, empiricalCase.CaseID))
	if directives != "" {
		sections = append(sections, directives)
	}
	if len(transactions) > 0 {
		sections = append(sections, strings.Join(transactions, "\n\n"))
	}

	var content = strings.Join(sections, "\n\n")
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return content, generationNotes, nil
}

// renderCommodityDirectives emits one deterministic commodity directive per
// journal commodity using the case-selected lot mode.
// Authored by: OpenCode
func renderCommodityDirectives(activities []fixture.EmpiricalActivity, lotMode string) string {
	var symbols = make(map[string]struct{}, len(activities))
	var activityIndex int

	for activityIndex = range activities {
		var symbol = strings.TrimSpace(activities[activityIndex].AssetSymbol)
		if symbol == "" {
			continue
		}

		symbols[symbol] = struct{}{}
	}

	var orderedSymbols = make([]string, 0, len(symbols))
	for symbol := range symbols {
		orderedSymbols = append(orderedSymbols, symbol)
	}
	sort.Strings(orderedSymbols)

	var lines = make([]string, 0, len(orderedSymbols))
	var symbolIndex int
	for symbolIndex = range orderedSymbols {
		lines = append(lines, "commodity "+orderedSymbols[symbolIndex]+"  ; lots: "+lotMode)
	}

	return strings.Join(lines, "\n")
}

// renderJournalActivity renders one activity into zero or more journal
// transactions and optional generation notes.
// Authored by: OpenCode
func renderJournalActivity(
	activity fixture.EmpiricalActivity,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	tracker *journalLotTracker,
) ([]string, []string, error) {
	if isZeroPricedReduction(activity) {
		return renderZeroPricedReductionActivity(activity, empiricalCase, method, tracker)
	}

	switch activity.ActivityType {
	case syncmodel.ActivityTypeBuy:
		var renderedBuy, err = renderPricedBuyActivity(activity, empiricalCase, method, tracker)
		if err != nil {
			return nil, nil, err
		}

		return []string{renderedBuy}, nil, nil
	case syncmodel.ActivityTypeSell:
		var renderedSell, err = renderPricedSellActivity(activity, empiricalCase, method, tracker)
		if err != nil {
			return nil, nil, err
		}

		return []string{renderedSell}, nil, nil
	default:
		return nil, nil, fmt.Errorf("render journal activity %q: unsupported activity type %q", strings.TrimSpace(activity.SourceID), strings.TrimSpace(string(activity.ActivityType)))
	}
}

// renderPricedBuyActivity renders one priced BUY row and updates any native lot
// tracker state used for later zero-priced reductions.
// Authored by: OpenCode
func renderPricedBuyActivity(
	activity fixture.EmpiricalActivity,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	tracker *journalLotTracker,
) (string, error) {
	var quantity, err = activityQuantity(activity)
	if err != nil {
		return "", err
	}

	var basisTotal apd.Decimal
	basisTotal, err = activityBasisTotal(activity)
	if err != nil {
		return "", err
	}

	var occurredAt time.Time
	var date string
	occurredAt, date, err = activityTiming(activity)
	if err != nil {
		return "", err
	}

	var unitBasis apd.Decimal
	unitBasis, err = supportmath.DivideFiniteRoundHalfUp(basisTotal, quantity)
	if err != nil {
		return "", fmt.Errorf("render priced BUY activity %q: divide unit basis: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var account = journalAssetAccount(method, empiricalCase.CaseID, activity)
	if tracker != nil {
		err = tracker.addAcquisition(
			account,
			journalAcquisition{
				sourceID:           strings.TrimSpace(activity.SourceID),
				date:               date,
				occurredAt:         occurredAt,
				deterministicOrder: activity.DeterministicOrder,
				unitBasis:          supportmath.Clone(unitBasis),
			},
			quantity,
			basisTotal,
		)
		if err != nil {
			return "", err
		}
	}

	var selector string
	selector, err = journalLotSelector(date, activity.SourceID, unitBasis)
	if err != nil {
		return "", err
	}

	var assetAmount string
	assetAmount, err = journalCommodityAmount(quantity, activity.AssetSymbol)
	if err != nil {
		return "", err
	}

	var negativeBasis apd.Decimal
	negativeBasis, err = negateDecimal(basisTotal)
	if err != nil {
		return "", fmt.Errorf("render priced BUY activity %q: negate basis total: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var cashAmount string
	cashAmount, err = journalUSDValue(negativeBasis)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		date + " buy " + strings.TrimSpace(activity.SourceID),
		"    " + account + "  " + assetAmount + " " + selector,
		"    assets:cash:USD  " + cashAmount,
	}, "\n"), nil
}

// renderPricedSellActivity renders one priced SELL row and updates any native
// lot tracker state used for later zero-priced reductions.
// Authored by: OpenCode
func renderPricedSellActivity(
	activity fixture.EmpiricalActivity,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	tracker *journalLotTracker,
) (string, error) {
	var quantity, err = activityQuantity(activity)
	if err != nil {
		return "", err
	}

	var account = journalAssetAccount(method, empiricalCase.CaseID, activity)
	if tracker != nil {
		_, err = tracker.dispose(account, quantity)
		if err != nil {
			return "", fmt.Errorf("render priced SELL activity %q: %w", strings.TrimSpace(activity.SourceID), err)
		}
	}

	var netProceeds apd.Decimal
	netProceeds, err = activityNetProceedsTotal(activity)
	if err != nil {
		return "", err
	}

	var unitNetProceeds apd.Decimal
	unitNetProceeds, err = supportmath.DivideFiniteRoundHalfUp(netProceeds, quantity)
	if err != nil {
		return "", fmt.Errorf("render priced SELL activity %q: divide unit net proceeds: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var negativeQuantity apd.Decimal
	negativeQuantity, err = negateDecimal(quantity)
	if err != nil {
		return "", fmt.Errorf("render priced SELL activity %q: negate quantity: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var date string
	_, date, err = activityTiming(activity)
	if err != nil {
		return "", err
	}

	var assetAmount string
	assetAmount, err = journalCommodityAmount(negativeQuantity, activity.AssetSymbol)
	if err != nil {
		return "", err
	}

	var unitPrice string
	unitPrice, err = journalUSDPrice(unitNetProceeds)
	if err != nil {
		return "", err
	}

	var cashAmount string
	cashAmount, err = journalUSDValue(netProceeds)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		date + " sell " + strings.TrimSpace(activity.SourceID),
		"    " + account + "  " + assetAmount + " {} @ " + unitPrice + "  ; posting_source_id: " + strings.TrimSpace(activity.SourceID),
		"    assets:cash:USD  " + cashAmount,
	}, "\n"), nil
}

// renderZeroPricedReductionActivity renders native zero-priced reductions as
// one sink transfer per matched lot fragment, or records an omission note when
// the journal lot mode has no native support.
// Authored by: OpenCode
func renderZeroPricedReductionActivity(
	activity fixture.EmpiricalActivity,
	empiricalCase fixture.EmpiricalCase,
	method reportmodel.CostBasisMethod,
	tracker *journalLotTracker,
) ([]string, []string, error) {
	if tracker == nil {
		return nil, []string{zeroPricedReductionOmissionNote(activity, empiricalCase, method)}, nil
	}

	var quantity, err = activityQuantity(activity)
	if err != nil {
		return nil, nil, err
	}

	var account = journalAssetAccount(method, empiricalCase.CaseID, activity)
	var matches []journalLotMatch
	matches, err = tracker.dispose(account, quantity)
	if err != nil {
		return nil, nil, fmt.Errorf("render zero-priced reduction %q: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var date string
	_, date, err = activityTiming(activity)
	if err != nil {
		return nil, nil, err
	}

	var transactions = make([]string, 0, len(matches))
	var matchIndex int
	for matchIndex = range matches {
		var transaction string
		transaction, err = renderZeroPricedReductionSegment(date, activity, account, matches[matchIndex])
		if err != nil {
			return nil, nil, err
		}

		transactions = append(transactions, transaction)
	}

	return transactions, nil, nil
}

// renderZeroPricedReductionSegment renders one matched lot fragment as a sink
// transfer into the zero-priced reduction equity account.
// Authored by: OpenCode
func renderZeroPricedReductionSegment(date string, activity fixture.EmpiricalActivity, account string, match journalLotMatch) (string, error) {
	var selector, err = journalLotSelector(match.acquisition.date, match.acquisition.sourceID, match.acquisition.unitBasis)
	if err != nil {
		return "", err
	}

	var negativeQuantity apd.Decimal
	negativeQuantity, err = negateDecimal(match.quantity)
	if err != nil {
		return "", fmt.Errorf("render zero-priced reduction segment %q: negate quantity: %w", strings.TrimSpace(activity.SourceID), err)
	}

	var fromAmount string
	fromAmount, err = journalCommodityAmount(negativeQuantity, activity.AssetSymbol)
	if err != nil {
		return "", err
	}

	var toAmount string
	toAmount, err = journalCommodityAmount(match.quantity, activity.AssetSymbol)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		date + " zero-priced reduction " + strings.TrimSpace(activity.SourceID) + " from " + match.acquisition.sourceID,
		"    " + account + "  " + fromAmount + " " + selector,
		"    equity:zero-priced-reduction  " + toAmount,
	}, "\n"), nil
}

// zeroPricedReductionOmissionNote records a deterministic generation note when a
// zero-priced reduction is omitted for non-native lot handling.
// Authored by: OpenCode
func zeroPricedReductionOmissionNote(activity fixture.EmpiricalActivity, empiricalCase fixture.EmpiricalCase, method reportmodel.CostBasisMethod) string {
	return "omitted zero-priced reduction " + strings.TrimSpace(activity.SourceID) +
		" from " + strings.TrimSpace(empiricalCase.CaseID) +
		" because lot mode " + journalLotMode(method, empiricalCase.CaseID) +
		" does not support native zero-priced handling"
}

// newJournalLotTracker creates one native-lot tracker for zero-priced
// reduction rendering.
// Authored by: OpenCode
func newJournalLotTracker(method reportbasis.LotMethod) (*journalLotTracker, error) {
	_, err := reportbasis.NewLotMethodState(method)
	if err != nil {
		return nil, fmt.Errorf("new journal lot tracker: %w", err)
	}

	return &journalLotTracker{
		method:       method,
		states:       make(map[string]*reportbasis.LotMethodState),
		acquisitions: make(map[string]journalAcquisition),
	}, nil
}

// addAcquisition adds one priced acquisition into the tracked native-lot state.
// Authored by: OpenCode
func (tracker *journalLotTracker) addAcquisition(account string, acquisition journalAcquisition, quantity apd.Decimal, basis apd.Decimal) error {
	if tracker == nil {
		return fmt.Errorf("journal lot tracker is required")
	}

	var state, err = tracker.state(account)
	if err != nil {
		return err
	}

	err = state.AddAcquisition(reportbasis.LotAcquisition{
		SourceID:           acquisition.sourceID,
		AcquiredAt:         acquisition.occurredAt,
		DeterministicOrder: acquisition.deterministicOrder,
		RemainingQuantity:  supportmath.Clone(quantity),
		RemainingBasis:     supportmath.Clone(basis),
	})
	if err != nil {
		return fmt.Errorf("track acquisition %q in %q: %w", acquisition.sourceID, strings.TrimSpace(account), err)
	}

	tracker.acquisitions[acquisition.sourceID] = acquisition
	return nil
}

// dispose removes one quantity from the tracked native-lot state and returns
// the selected matched lot fragments.
// Authored by: OpenCode
func (tracker *journalLotTracker) dispose(account string, quantity apd.Decimal) ([]journalLotMatch, error) {
	if tracker == nil {
		return nil, fmt.Errorf("journal lot tracker is required")
	}

	var state, err = tracker.state(account)
	if err != nil {
		return nil, err
	}

	var result reportbasis.LotDisposalResult
	result, err = state.Dispose(quantity)
	if err != nil {
		return nil, fmt.Errorf("dispose quantity from %q: %w", strings.TrimSpace(account), err)
	}

	var matches = make([]journalLotMatch, 0, len(result.Matches))
	var matchIndex int
	for matchIndex = range result.Matches {
		var sourceID = strings.TrimSpace(result.Matches[matchIndex].AcquisitionSourceID)
		var acquisition, found = tracker.acquisitions[sourceID]
		if !found {
			return nil, fmt.Errorf("dispose quantity from %q: acquisition %q is missing from the native-lot tracker", strings.TrimSpace(account), sourceID)
		}

		matches = append(matches, journalLotMatch{
			acquisition: acquisition,
			quantity:    supportmath.Clone(result.Matches[matchIndex].MatchedQuantity),
		})
	}

	return matches, nil
}

// state returns the tracked native-lot state for one asset account path.
// Authored by: OpenCode
func (tracker *journalLotTracker) state(account string) (*reportbasis.LotMethodState, error) {
	if tracker == nil {
		return nil, fmt.Errorf("journal lot tracker is required")
	}

	var trimmedAccount = strings.TrimSpace(account)
	var existing, found = tracker.states[trimmedAccount]
	if found {
		return existing, nil
	}

	var state, err = reportbasis.NewLotMethodState(tracker.method)
	if err != nil {
		return nil, fmt.Errorf("build native-lot state for %q: %w", trimmedAccount, err)
	}

	tracker.states[trimmedAccount] = state
	return state, nil
}

// journalLotMode returns the hledger lot-reduction mode used by one rendered
// case-scoped journal.
// Authored by: OpenCode
func journalLotMode(method reportmodel.CostBasisMethod, caseID string) string {
	switch method {
	case reportmodel.CostBasisMethodFIFO:
		return "FIFO"
	case reportmodel.CostBasisMethodLIFO:
		return "LIFO"
	case reportmodel.CostBasisMethodHIFO:
		return "HIFO"
	case reportmodel.CostBasisMethodAverageCost:
		return "AVERAGE"
	case reportmodel.CostBasisMethodScopeLocalHybrid:
		if strings.TrimSpace(caseID) == reliableHybridFIFOCaseID {
			return "FIFO"
		}

		return "AVERAGE"
	default:
		return strings.ToUpper(strings.TrimSpace(string(method)))
	}
}

// nativeJournalLotMethod returns the native exact-lot method required for
// zero-priced reduction pre-splitting when one exists.
// Authored by: OpenCode
func nativeJournalLotMethod(method reportmodel.CostBasisMethod, caseID string) (reportbasis.LotMethod, bool) {
	switch journalLotMode(method, caseID) {
	case "FIFO":
		return reportbasis.LotMethodFIFO, true
	case "LIFO":
		return reportbasis.LotMethodLIFO, true
	case "HIFO":
		return reportbasis.LotMethodHIFO, true
	default:
		return "", false
	}
}

// journalAssetAccount returns the case-scoped asset account path for one
// rendered journal row.
// Authored by: OpenCode
func journalAssetAccount(method reportmodel.CostBasisMethod, caseID string, activity fixture.EmpiricalActivity) string {
	var assetKey = strings.TrimSpace(activity.AssetIdentityKey)
	if method != reportmodel.CostBasisMethodScopeLocalHybrid {
		return strings.Join([]string{"assets", "empirical", method.FilenameSlug(), assetKey}, ":")
	}

	if strings.TrimSpace(caseID) == reliableHybridFIFOCaseID {
		var scopeID, reliable = reliableScopeID(activity)
		if reliable {
			return strings.Join([]string{"assets", "empirical", "scope-local-hybrid", scopeID, assetKey}, ":")
		}
	}

	return strings.Join([]string{"assets", "empirical", "scope-local-hybrid", "fallback", assetKey}, ":")
}

// reliableScopeID returns the scoped account suffix used by the reliable hybrid
// FIFO case when a row preserves reliable source-scope evidence.
// Authored by: OpenCode
func reliableScopeID(activity fixture.EmpiricalActivity) (string, bool) {
	if activity.SourceScope == nil {
		return "", false
	}

	var scopeID = strings.TrimSpace(activity.SourceScope.ScopeID)
	if scopeID == "" {
		return "", false
	}
	if activity.SourceScope.Reliability != syncmodel.ScopeReliabilityReliable {
		return "", false
	}

	return scopeID, true
}

// journalRelativePath returns the repository-relative path for one generated
// case-scoped journal file.
// Authored by: OpenCode
func journalRelativePath(method reportmodel.CostBasisMethod, caseID string) string {
	return path.Join("testdata", "empirical", "hledger", method.FilenameSlug(), strings.TrimSpace(caseID)+".journal")
}

// journalLedgerID returns the stable logical identity for one generated
// case-scoped journal.
// Authored by: OpenCode
func journalLedgerID(method reportmodel.CostBasisMethod, caseID string) string {
	return strings.Join([]string{"empirical-journal", method.FilenameSlug(), strings.TrimSpace(caseID)}, ":")
}

// isZeroPricedReduction reports whether one SELL row represents a basis-only
// reduction with no priced proceeds.
// Authored by: OpenCode
func isZeroPricedReduction(activity fixture.EmpiricalActivity) bool {
	if strings.TrimSpace(activity.ZeroPricedReductionExplanation) != "" {
		return true
	}

	return activity.ActivityType == syncmodel.ActivityTypeSell &&
		strings.TrimSpace(activity.GrossValue) == "" &&
		strings.TrimSpace(activity.UnitPrice) == "" &&
		strings.TrimSpace(activity.Currency) == ""
}

// activityTiming parses one empirical activity timestamp and returns both the
// exact timestamp and its journal date.
// Authored by: OpenCode
func activityTiming(activity fixture.EmpiricalActivity) (time.Time, string, error) {
	var occurredAt, err = time.Parse(time.RFC3339, strings.TrimSpace(activity.OccurredAt))
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse occurred_at for %q: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return occurredAt, occurredAt.Format("2006-01-02"), nil
}

// activityQuantity parses and validates one activity quantity.
// Authored by: OpenCode
func activityQuantity(activity fixture.EmpiricalActivity) (apd.Decimal, error) {
	var quantity, err = parseActivityDecimal(activity.SourceID, "quantity", activity.Quantity)
	if err != nil {
		return apd.Decimal{}, err
	}
	if err = supportmath.RequirePositive(quantity); err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q quantity: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return quantity, nil
}

// activityBasisTotal parses one priced BUY row and returns its rendered basis as
// gross plus fee.
// Authored by: OpenCode
func activityBasisTotal(activity fixture.EmpiricalActivity) (apd.Decimal, error) {
	var grossValue, err = pricedGrossValue(activity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var feeAmount apd.Decimal
	feeAmount, err = activityFeeAmount(activity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var total apd.Decimal
	total, err = supportmath.Add(grossValue, feeAmount)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q basis total: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return total, nil
}

// activityNetProceedsTotal parses one priced SELL row and returns its rendered
// proceeds as gross minus fee.
// Authored by: OpenCode
func activityNetProceedsTotal(activity fixture.EmpiricalActivity) (apd.Decimal, error) {
	var grossValue, err = pricedGrossValue(activity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var feeAmount apd.Decimal
	feeAmount, err = activityFeeAmount(activity)
	if err != nil {
		return apd.Decimal{}, err
	}

	var total apd.Decimal
	total, err = supportmath.Subtract(grossValue, feeAmount)
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q net proceeds: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return total, nil
}

// pricedGrossValue parses and validates the gross value required for one priced
// journal row.
// Authored by: OpenCode
func pricedGrossValue(activity fixture.EmpiricalActivity) (apd.Decimal, error) {
	var grossValue, err = parseActivityDecimal(activity.SourceID, "gross_value", activity.GrossValue)
	if err != nil {
		return apd.Decimal{}, err
	}
	if err = supportmath.RequireNonNegative(grossValue); err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q gross_value: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return grossValue, nil
}

// activityFeeAmount parses one optional fee amount and falls back to zero when
// the source row omits a fee.
// Authored by: OpenCode
func activityFeeAmount(activity fixture.EmpiricalActivity) (apd.Decimal, error) {
	if strings.TrimSpace(activity.FeeAmount) == "" {
		return supportmath.Zero(), nil
	}

	var feeAmount, err = parseActivityDecimal(activity.SourceID, "fee_amount", activity.FeeAmount)
	if err != nil {
		return apd.Decimal{}, err
	}
	if err = supportmath.RequireNonNegative(feeAmount); err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q fee_amount: %w", strings.TrimSpace(activity.SourceID), err)
	}

	return feeAmount, nil
}

// parseActivityDecimal parses one decimal field from an activity with an
// activity-specific error prefix.
// Authored by: OpenCode
func parseActivityDecimal(sourceID string, field string, raw string) (apd.Decimal, error) {
	var value, _, err = decimalsupport.ParseString(strings.TrimSpace(raw))
	if err != nil {
		return apd.Decimal{}, fmt.Errorf("activity %q %s: %w", strings.TrimSpace(sourceID), strings.TrimSpace(field), err)
	}

	return value, nil
}

// negateDecimal returns the exact additive inverse of one finite decimal.
// Authored by: OpenCode
func negateDecimal(value apd.Decimal) (apd.Decimal, error) {
	return supportmath.Subtract(supportmath.Zero(), value)
}

// journalLotSelector builds the fixed lot-selector annotation required by the
// journal slice for acquisitions and native zero-priced reductions.
// Authored by: OpenCode
func journalLotSelector(date string, sourceID string, unitBasis apd.Decimal) (string, error) {
	var basis, err = journalUSDPrice(unitBasis)
	if err != nil {
		return "", err
	}

	return "{" + strings.TrimSpace(date) + ", " + strconv.Quote(strings.TrimSpace(sourceID)) + ", " + basis + "}", nil
}

// journalCommodityAmount formats one asset commodity amount in canonical
// quantity-first journal form.
// Authored by: OpenCode
func journalCommodityAmount(quantity apd.Decimal, symbol string) (string, error) {
	var trimmedSymbol = strings.TrimSpace(symbol)
	if trimmedSymbol == "" {
		return "", fmt.Errorf("journal commodity amount requires an asset symbol")
	}

	var canonical, err = decimalsupport.CanonicalString(quantity)
	if err != nil {
		return "", fmt.Errorf("format journal commodity amount: %w", err)
	}

	return canonical + " " + trimmedSymbol, nil
}

// journalUSDValue formats one signed USD amount for journal postings.
// Authored by: OpenCode
func journalUSDValue(value apd.Decimal) (string, error) {
	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		return "", fmt.Errorf("format USD journal value: %w", err)
	}

	if strings.HasPrefix(canonical, "-") {
		return "-$" + strings.TrimPrefix(canonical, "-"), nil
	}

	return "$" + canonical, nil
}

// journalUSDPrice formats one non-negative USD unit price for lot selectors and
// transacted-cost annotations.
// Authored by: OpenCode
func journalUSDPrice(value apd.Decimal) (string, error) {
	var canonical, err = decimalsupport.CanonicalString(value)
	if err != nil {
		return "", fmt.Errorf("format USD journal price: %w", err)
	}
	if strings.HasPrefix(canonical, "-") {
		return "", fmt.Errorf("format USD journal price: value must not be negative")
	}

	return "$" + canonical, nil
}
