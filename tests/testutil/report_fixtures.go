package testutil

import (
	"strconv"
	"strings"
	"time"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	decimalsupport "github.com/benizzio/ghostfolio-cryptogains/internal/support/decimal"
	syncmodel "github.com/benizzio/ghostfolio-cryptogains/internal/sync/model"
	"github.com/cockroachdb/apd/v3"
)

// ReportLedgerFixture stores one deterministic protected-activity cache and the
// years that later report tests can use for primary success and incomplete-input
// scenarios.
// Authored by: OpenCode
type ReportLedgerFixture struct {
	ProtectedActivityCache      syncmodel.ProtectedActivityCache
	PrimaryReportYear           int
	IncompleteContextReportYear int
	CurrencylessOrderReportYear int
	ExpectedReports             map[reportmodel.CostBasisMethod]ExpectedReportLedger
}

// ReportPerformanceFixture stores one deterministic large-history protected
// cache and request metadata for the opt-in report performance verification
// path.
// Authored by: OpenCode
type ReportPerformanceFixture struct {
	ProtectedActivityCache syncmodel.ProtectedActivityCache
	ReportYear             int
	ActivityCount          int
	CalendarYearSpan       int
}

// ReportOutputFormatFixture stores one deterministic supported report output
// format from the report model plus contract-facing file expectations.
// Authored by: OpenCode
type ReportOutputFormatFixture struct {
	Format      reportmodel.ReportOutputFormat
	Code        string
	Label       string
	FileCount   int
	Extensions  []string
	Description string
}

// ReportRequestFixture stores one deterministic validated report request for a
// selected output format.
// Authored by: OpenCode
type ReportRequestFixture struct {
	Request            reportmodel.ReportRequest
	Year               int
	CostBasisMethod    reportmodel.CostBasisMethod
	ReportBaseCurrency reportmodel.ReportBaseCurrency
	OutputFormat       reportmodel.ReportOutputFormat
	RequestedAt        time.Time
}

// ReportAnnexFixture stores deterministic Annex 1 expected content with the
// validated model shell required by calculated reports.
// Authored by: OpenCode
type ReportAnnexFixture struct {
	Annex                          reportmodel.AuditAnnex
	Title                          string
	SectionOrder                   []reportmodel.AuditAnnexSection
	PerAssetSections               []ExpectedPerAssetAuditSection
	CurrencyConversionEntries      []reportmodel.ConversionAuditEntry
	CurrencyConversionEmptyMessage string
}

// ExpectedPerAssetAuditSection stores one deterministic expected Annex 1 asset
// section.
// Authored by: OpenCode
type ExpectedPerAssetAuditSection struct {
	AssetIdentityKey string
	DisplayLabel     string
	Entries          []ExpectedAuditActivityEntry
}

// ExpectedAuditActivityEntry stores one deterministic expected Annex 1 activity
// row or subsection.
// Authored by: OpenCode
type ExpectedAuditActivityEntry struct {
	SourceID               string
	OccurredAt             time.Time
	ActivityType           syncmodel.ActivityType
	Quantity               string
	UnitPrice              string
	GrossValue             string
	FeeAmount              string
	ActivityCurrency       string
	CalculationCurrency    string
	QuantityAfterActivity  string
	BasisAfterActivity     string
	FullLiquidationEvent   bool
	AllocatedBasis         string
	NetLiquidationProceeds string
	GainOrLoss             string
	ConversionStatus       reportmodel.ConversionStatus
	Note                   string
}

// ReportConversionFixture stores deterministic report conversion audit evidence
// for renderer, annex, and output contract tests.
// Authored by: OpenCode
type ReportConversionFixture struct {
	RateSource         reportmodel.ExchangeRateEvidence
	ConversionEntry    reportmodel.ConversionAuditEntry
	ConvertedAmount    reportmodel.ConvertedActivityAmount
	SameCurrencyAmount reportmodel.ConvertedActivityAmount
}

// ExpectedReportLedger stores one deterministic expected yearly report outcome
// for one cost-basis method.
// Authored by: OpenCode
type ExpectedReportLedger struct {
	ReportCalculationCurrency string
	YearlyNetTotal            string
	SummaryByAsset            map[string]ExpectedAssetSummary
	ReferenceByAsset          map[string]ExpectedReferenceEntry
	DetailByAsset             map[string]ExpectedAssetDetail
}

// ExpectedAssetSummary stores one expected summary row keyed by asset identity.
// Authored by: OpenCode
type ExpectedAssetSummary struct {
	AssetIdentityKey string
	DisplayLabel     string
	NetGainOrLoss    string
}

// ExpectedReferenceEntry stores one expected reference-section row keyed by
// asset identity.
// Authored by: OpenCode
type ExpectedReferenceEntry struct {
	AssetIdentityKey                   string
	DisplayLabel                       string
	FullLiquidationCountThroughYearEnd int
	MainSectionStatus                  reportmodel.ReferenceSectionStatus
}

// ExpectedAssetDetail stores one expected per-asset detail ledger keyed by
// asset identity.
// Authored by: OpenCode
type ExpectedAssetDetail struct {
	AssetIdentityKey             string
	DisplayLabel                 string
	OpeningQuantity              string
	OpeningCostBasis             string
	ClosingQuantity              string
	ClosingCostBasis             string
	CalculationCurrency          string
	ActivityRows                 []ExpectedAssetActivityRow
	LiquidationSummaries         []ExpectedLiquidationSummary
	ExpectedFullLiquidationCount int
}

// ExpectedAssetActivityRow stores one expected in-year activity ledger row.
// Authored by: OpenCode
type ExpectedAssetActivityRow struct {
	SourceID                    string
	ActivityType                syncmodel.ActivityType
	Quantity                    string
	UnitPrice                   string
	GrossValue                  string
	FeeAmount                   string
	ActivityCurrency            string
	BasisAfterRow               string
	CalculationCurrency         string
	QuantityAfterRow            string
	HoldingReductionExplanation string
}

// ExpectedLiquidationSummary stores one expected priced-liquidation ledger row.
// Authored by: OpenCode
type ExpectedLiquidationSummary struct {
	SourceID               string
	DisposedQuantity       string
	AllocatedBasis         string
	NetLiquidationProceeds string
	GainOrLoss             string
	ActivityCurrency       string
	CalculationCurrency    string
}

// DeterministicReportLedgerFixture returns one reusable multi-year synced-data
// fixture for report calculation, rendering, and runtime tests.
//
// The fixture keeps its coverage intentionally broad but data volume small. It
// includes activity before, within, and after the primary report year, multiple
// asset timelines, one production-shaped explained zero-priced holding
// reduction that preserves explicit zero-valued source fields, one priced row
// with an explicit zero fee, one priced BUY whose order tier is currencyless
// while later tiers remain usable, mixed amount-source tiers, and scope
// variation.
//
// Authored by: OpenCode
func DeterministicReportLedgerFixture() ReportLedgerFixture {
	var activities = []syncmodel.ActivityRecord{
		reportActivity(reportActivityInput{
			SourceID:         "eth-buy-2022-001",
			OccurredAt:       "2022-06-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         "5",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "1000",
			OrderGrossValue:  "5000",
			OrderFeeAmount:   "5",
			SourceScope:      reliableAccountScope("account-long-term", "Long Term Account"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "eth-sell-2023-001",
			OccurredAt:       "2023-03-10T09:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-eth-001",
			AssetSymbol:      "ETH",
			AssetName:        "Ethereum",
			Quantity:         "5",
			OrderCurrency:    "USD",
			OrderGrossValue:  "6500",
			OrderFeeAmount:   "10",
			SourceScope:      reliableAccountScope("account-long-term", "Long Term Account"),
		}),
		reportActivity(reportActivityInput{
			SourceID:              "btc-buy-2023-boundary-001",
			OccurredAt:            "2023-12-31T23:30:00-02:00",
			ActivityType:          syncmodel.ActivityTypeBuy,
			AssetIdentityKey:      "asset-btc-001",
			AssetSymbol:           "BTC",
			AssetName:             "Bitcoin",
			Quantity:              "2",
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: "20000",
			AssetProfileFeeAmount: "18",
			BaseCurrency:          "USD",
			BaseGrossValue:        "44000",
			BaseFeeAmount:         "20",
			SourceScope:           reliableWalletScope("wallet-main", "Main Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ltc-buy-2023-001",
			OccurredAt:       "2023-02-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-ltc-001",
			AssetSymbol:      "LTC",
			AssetName:        "Litecoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "100",
			OrderGrossValue:  "100",
			OrderFeeAmount:   "0",
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ltc-buy-2023-002",
			OccurredAt:       "2023-07-01T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-ltc-001",
			AssetSymbol:      "LTC",
			AssetName:        "Litecoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "300",
			OrderGrossValue:  "300",
			OrderFeeAmount:   "0",
		}),
		reportActivity(reportActivityInput{
			SourceID:         "avax-buy-beta-2023-001",
			OccurredAt:       "2023-01-10T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-avax-001",
			AssetSymbol:      "AVAX",
			AssetName:        "Avalanche",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "500",
			OrderGrossValue:  "500",
			OrderFeeAmount:   "0",
			SourceScope:      reliableWalletScope("wallet-avax-beta", "AVAX Beta Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "avax-buy-alpha-2023-001",
			OccurredAt:       "2023-06-10T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-avax-001",
			AssetSymbol:      "AVAX",
			AssetName:        "Avalanche",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "100",
			OrderGrossValue:  "100",
			OrderFeeAmount:   "0",
			SourceScope:      reliableWalletScope("wallet-avax-alpha", "AVAX Alpha Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "btc-sell-2024-zero-fee-001",
			OccurredAt:       "2024-01-01T00:15:00+01:00",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-btc-001",
			AssetSymbol:      "BTC",
			AssetName:        "Bitcoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "25000",
			OrderFeeAmount:   "0",
			SourceScope:      reliableWalletScope("wallet-main", "Main Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "xrp-buy-2024-001",
			OccurredAt:       "2024-02-01T12:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-xrp-001",
			AssetSymbol:      "XRP",
			AssetName:        "XRP",
			Quantity:         "1000",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0.5",
			OrderGrossValue:  "500",
			OrderFeeAmount:   "1",
			SourceScope:      reliableWalletScope("wallet-alt", "Alt Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "xrp-reduction-2024-001",
			OccurredAt:       "2024-04-01T12:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-xrp-001",
			AssetSymbol:      "XRP",
			AssetName:        "XRP",
			Quantity:         "200",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0",
			OrderGrossValue:  "0",
			OrderFeeAmount:   "0",
			Comment:          "Bridge migration holding reduction",
			SourceScope:      reliableWalletScope("wallet-alt", "Alt Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ada-buy-2024-001",
			OccurredAt:       "2024-06-15T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-ada-001",
			AssetSymbol:      "ADA",
			AssetName:        "Cardano",
			Quantity:         "1000",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0.25",
			OrderGrossValue:  "250",
			OrderFeeAmount:   "2",
			SourceScope:      reliableAccountScope("account-broker", "Broker Account"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ada-buy-2024-002",
			OccurredAt:       "2024-09-01T09:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-ada-001",
			AssetSymbol:      "ADA",
			AssetName:        "Cardano",
			Quantity:         "500",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0.28",
			OrderGrossValue:  "140",
			OrderFeeAmount:   "1",
			SourceScope:      reliableAccountScope("account-broker", "Broker Account"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ada-sell-2024-001",
			OccurredAt:       "2024-09-01T09:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-ada-001",
			AssetSymbol:      "ADA",
			AssetName:        "Cardano",
			Quantity:         "1000",
			OrderCurrency:    "USD",
			OrderGrossValue:  "300",
			OrderFeeAmount:   "3",
			SourceScope:      reliableAccountScope("account-broker", "Broker Account"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "ltc-sell-2024-001",
			OccurredAt:       "2024-07-15T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-ltc-001",
			AssetSymbol:      "LTC",
			AssetName:        "Litecoin",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "250",
			OrderFeeAmount:   "0",
		}),
		reportActivity(reportActivityInput{
			SourceID:         "avax-sell-alpha-2024-001",
			OccurredAt:       "2024-08-15T10:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-avax-001",
			AssetSymbol:      "AVAX",
			AssetName:        "Avalanche",
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  "250",
			OrderFeeAmount:   "0",
			SourceScope:      reliableWalletScope("wallet-avax-alpha", "AVAX Alpha Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:         "btc-sell-2025-base-tier-001",
			OccurredAt:       "2025-05-01T12:00:00Z",
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: "asset-btc-001",
			AssetSymbol:      "BTC",
			AssetName:        "Bitcoin",
			Quantity:         "1",
			BaseCurrency:     "USD",
			BaseGrossValue:   "28000",
			BaseFeeAmount:    "12",
			SourceScope:      nil,
		}),
		reportActivity(reportActivityInput{
			SourceID:         "doge-buy-2025-incomplete-001",
			OccurredAt:       "2025-03-01T12:00:00Z",
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: "asset-doge-001",
			AssetSymbol:      "DOGE",
			AssetName:        "Dogecoin",
			Quantity:         "10000",
			OrderCurrency:    "USD",
			OrderUnitPrice:   "0.1",
			OrderGrossValue:  "1000",
			SourceScope:      reliableWalletScope("wallet-speculative", "Speculative Wallet"),
		}),
		reportActivity(reportActivityInput{
			SourceID:              "sol-buy-2026-asset-tier-001",
			OccurredAt:            "2026-01-10T08:00:00Z",
			ActivityType:          syncmodel.ActivityTypeBuy,
			AssetIdentityKey:      "asset-sol-001",
			AssetSymbol:           "SOL",
			AssetName:             "Solana",
			Quantity:              "50",
			OrderUnitPrice:        "81",
			OrderGrossValue:       "4050",
			OrderFeeAmount:        "1",
			AssetProfileCurrency:  "EUR",
			AssetProfileUnitPrice: "80",
			AssetProfileFeeAmount: "0.5",
			BaseCurrency:          "USD",
			BaseGrossValue:        "4300",
			BaseFeeAmount:         "0.75",
			SourceScope:           reliableWalletScope("wallet-growth", "Growth Wallet"),
		}),
	}

	return ReportLedgerFixture{
		ProtectedActivityCache: syncmodel.ProtectedActivityCache{
			SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
			RetrievedCount:       len(activities),
			ActivityCount:        len(activities),
			AvailableReportYears: []int{2022, 2023, 2024, 2025, 2026},
			ScopeReliability:     syncmodel.ScopeReliabilityPartial,
			Activities:           activities,
		},
		PrimaryReportYear:           2024,
		IncompleteContextReportYear: 2025,
		CurrencylessOrderReportYear: 2026,
		ExpectedReports:             deterministicExpectedReportsByMethod(),
	}
}

// DeterministicLargeReportPerformanceFixture returns one deterministic
// 10,000-activity cache shaped for the opt-in large-history report performance
// verification path.
//
// The fixture spans at least five calendar years, forces heavy open-lot
// fragmentation for HIFO, and broadens the scope-local hybrid method back to
// asset-level scope by mixing reliable and unavailable source scope entries on
// the same asset timeline.
//
// Authored by: OpenCode
func DeterministicLargeReportPerformanceFixture() ReportPerformanceFixture {
	const activityCount = 10000
	const assetActivityCount = activityCount / 2
	const preReportBuyCountPerAsset = assetActivityCount / 2
	const inYearSellCountPerAsset = assetActivityCount / 2
	const startYear = 2020
	const calendarYearSpan = 6
	const reportYear = 2025

	var activities = make([]syncmodel.ActivityRecord, 0, activityCount)
	var years = make([]int, 0, calendarYearSpan)
	for year := startYear; year < startYear+calendarYearSpan; year++ {
		years = append(years, year)
	}

	activities = append(activities, appendPerformanceAssetActivities(performanceAssetInput{
		AssetIdentityKey:      "asset-btc-performance-001",
		AssetSymbol:           "BTC",
		AssetName:             "Bitcoin",
		PreReportBuyCount:     preReportBuyCountPerAsset,
		InYearSellCount:       inYearSellCountPerAsset,
		PreReportValueOffset:  100,
		InYearSellValueOffset: 1200,
		ReliableScope:         reliableWalletScope("wallet-performance-main", "Performance Main Wallet"),
	})...)
	activities = append(activities, appendPerformanceAssetActivities(performanceAssetInput{
		AssetIdentityKey:      "asset-eth-performance-001",
		AssetSymbol:           "ETH",
		AssetName:             "Ethereum",
		PreReportBuyCount:     preReportBuyCountPerAsset,
		InYearSellCount:       inYearSellCountPerAsset,
		PreReportValueOffset:  300,
		InYearSellValueOffset: 900,
		ReliableScope:         reliableWalletScope("wallet-performance-fallback", "Performance Fallback Wallet"),
		ForceFallbackScope:    true,
	})...)

	return ReportPerformanceFixture{
		ProtectedActivityCache: syncmodel.ProtectedActivityCache{
			SyncedAt:             time.Date(2026, time.May, 20, 15, 4, 5, 0, time.UTC),
			RetrievedCount:       len(activities),
			ActivityCount:        len(activities),
			AvailableReportYears: years,
			ScopeReliability:     syncmodel.ScopeReliabilityPartial,
			Activities:           activities,
		},
		ReportYear:       reportYear,
		ActivityCount:    len(activities),
		CalendarYearSpan: calendarYearSpan,
	}
}

// DeterministicReportOutputFormatFixtures returns the user-selectable output
// formats from the report-output contract using the validated report model
// constants.
//
// Authored by: OpenCode
func DeterministicReportOutputFormatFixtures() []ReportOutputFormatFixture {
	return []ReportOutputFormatFixture{
		{
			Format:      reportmodel.ReportOutputFormatMarkdown,
			Code:        string(reportmodel.ReportOutputFormatMarkdown),
			Label:       reportmodel.ReportOutputFormatMarkdown.Label(),
			FileCount:   2,
			Extensions:  []string{".md", ".md"},
			Description: "Main Markdown report plus separate Annex 1 Markdown file",
		},
		{
			Format:      reportmodel.ReportOutputFormatPDF,
			Code:        string(reportmodel.ReportOutputFormatPDF),
			Label:       reportmodel.ReportOutputFormatPDF.Label(),
			FileCount:   1,
			Extensions:  []string{".pdf"},
			Description: "One PDF containing the main report and Annex 1",
		},
	}
}

// DeterministicReportRequestFixture returns one validated report request for the
// provided output format and the shared primary report scenario.
//
// Authored by: OpenCode
func DeterministicReportRequestFixture(outputFormat reportmodel.ReportOutputFormat) ReportRequestFixture {
	var year = 2024
	var method = reportmodel.CostBasisMethodFIFO
	var reportBaseCurrency = reportmodel.ReportBaseCurrencyUSD
	var requestedAt = time.Date(2026, time.May, 21, 12, 34, 56, 0, time.UTC)
	var request, err = reportmodel.NewReportRequest(year, method, reportBaseCurrency, outputFormat, requestedAt)
	if err != nil {
		panic(err)
	}

	return ReportRequestFixture{
		Request:            request,
		Year:               year,
		CostBasisMethod:    method,
		ReportBaseCurrency: reportBaseCurrency,
		OutputFormat:       outputFormat,
		RequestedAt:        requestedAt,
	}
}

// DeterministicReportAnnexFixture returns expected Annex 1 audit data tied to
// DeterministicReportLedgerFixture's primary report year.
//
// Authored by: OpenCode
func DeterministicReportAnnexFixture() ReportAnnexFixture {
	var conversion = DeterministicReportConversionFixture()
	var annex = reportmodel.DefaultAuditAnnex()

	return ReportAnnexFixture{
		Annex:        annex,
		Title:        annex.Title,
		SectionOrder: append([]reportmodel.AuditAnnexSection(nil), annex.SectionOrder...),
		PerAssetSections: []ExpectedPerAssetAuditSection{
			{
				AssetIdentityKey: "asset-btc-001",
				DisplayLabel:     "BTC",
				Entries: []ExpectedAuditActivityEntry{
					{
						SourceID:              "btc-buy-2023-boundary-001",
						OccurredAt:            time.Date(2024, time.January, 1, 1, 30, 0, 0, time.UTC),
						ActivityType:          syncmodel.ActivityTypeBuy,
						Quantity:              "2",
						GrossValue:            "44000",
						FeeAmount:             "20",
						ActivityCurrency:      "USD",
						CalculationCurrency:   "USD",
						QuantityAfterActivity: "2",
						BasisAfterActivity:    "44019.8",
						ConversionStatus:      reportmodel.ConversionStatusConverted,
						Note:                  "Converted asset-profile EUR evidence retained separately from selected USD calculation fields.",
					},
					{
						SourceID:               "btc-sell-2024-zero-fee-001",
						OccurredAt:             time.Date(2023, time.December, 31, 23, 15, 0, 0, time.UTC),
						ActivityType:           syncmodel.ActivityTypeSell,
						Quantity:               "1",
						GrossValue:             "25000",
						FeeAmount:              "0",
						ActivityCurrency:       "USD",
						CalculationCurrency:    "USD",
						QuantityAfterActivity:  "1",
						BasisAfterActivity:     "22009.9",
						AllocatedBasis:         "22009.9",
						NetLiquidationProceeds: "25000",
						GainOrLoss:             "2990.1",
						ConversionStatus:       reportmodel.ConversionStatusSameCurrency,
					},
				},
			},
			{
				AssetIdentityKey: "asset-xrp-001",
				DisplayLabel:     "XRP",
				Entries: []ExpectedAuditActivityEntry{
					{
						SourceID:              "xrp-reduction-2024-001",
						OccurredAt:            time.Date(2024, time.April, 1, 12, 0, 0, 0, time.UTC),
						ActivityType:          syncmodel.ActivityTypeSell,
						Quantity:              "200",
						UnitPrice:             "0",
						GrossValue:            "0",
						FeeAmount:             "0",
						ActivityCurrency:      "USD",
						CalculationCurrency:   "USD",
						QuantityAfterActivity: "800",
						BasisAfterActivity:    "400.8",
						ConversionStatus:      reportmodel.ConversionStatusSameCurrency,
						Note:                  "Bridge migration holding reduction",
					},
				},
			},
		},
		CurrencyConversionEntries:      []reportmodel.ConversionAuditEntry{conversion.ConversionEntry},
		CurrencyConversionEmptyMessage: "No converted activity amounts were used for this report.",
	}
}

// DeterministicReportConversionFixture returns valid same-currency and
// converted amount evidence for tests that need report conversion audit data.
//
// Authored by: OpenCode
func DeterministicReportConversionFixture() ReportConversionFixture {
	var activityDate = time.Date(2024, time.February, 5, 14, 30, 0, 0, time.UTC)
	var rateDate = time.Date(2024, time.February, 5, 0, 0, 0, 0, time.UTC)
	var rateValue = decimalValue("1.08")
	var evidence = reportmodel.ExchangeRateEvidence{
		SourceCurrency:   "EUR",
		BaseCurrency:     reportmodel.ReportBaseCurrencyUSD,
		ActivityDate:     activityDate,
		RateDate:         rateDate,
		Authority:        reportmodel.RateAuthorityFederalReserve,
		ProviderID:       reportmodel.RateProviderIDFederalReserveH10,
		RateKind:         "daily noon buying rate",
		QuoteDirection:   reportmodel.QuoteDirectionBasePerSource,
		RateValue:        rateValue,
		DatasetReference: "Federal Reserve H.10 2024-02-05",
	}
	var convertedAmount = reportmodel.ConvertedActivityAmount{
		SourceID:             "eur-sell-2024-converted-001",
		AmountKind:           reportmodel.ConvertedAmountKindGrossValue,
		OriginalCurrency:     "EUR",
		OriginalAmount:       decimalValue("100"),
		ReportBaseCurrency:   reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:      decimalValue("108"),
		ExchangeRateEvidence: &evidence,
		ConversionStatus:     reportmodel.ConversionStatusConverted,
	}
	var sameCurrencyAmount = reportmodel.ConvertedActivityAmount{
		SourceID:           "usd-sell-2024-same-currency-001",
		AmountKind:         reportmodel.ConvertedAmountKindGrossValue,
		OriginalCurrency:   "USD",
		OriginalAmount:     decimalValue("250"),
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		ConvertedAmount:    decimalValue("250"),
		ConversionStatus:   reportmodel.ConversionStatusSameCurrency,
	}
	var conversionEntry = reportmodel.ConversionAuditEntry{
		SourceID:           convertedAmount.SourceID,
		AssetLabel:         "EUR Asset",
		ActivityDate:       activityDate,
		SourceCurrency:     "EUR",
		ReportBaseCurrency: reportmodel.ReportBaseCurrencyUSD,
		RateDate:           rateDate,
		RateAuthority:      reportmodel.RateAuthorityFederalReserve,
		RateKind:           "daily noon buying rate",
		RateValue:          rateValue,
		QuoteDirection:     reportmodel.QuoteDirectionBasePerSource,
		Amounts:            []reportmodel.ConvertedActivityAmount{convertedAmount},
	}

	return ReportConversionFixture{
		RateSource:         evidence,
		ConversionEntry:    conversionEntry,
		ConvertedAmount:    convertedAmount,
		SameCurrencyAmount: sameCurrencyAmount,
	}
}

// performanceAssetInput collects one large-history asset timeline declaration.
// Authored by: OpenCode
type performanceAssetInput struct {
	AssetIdentityKey      string
	AssetSymbol           string
	AssetName             string
	PreReportBuyCount     int
	InYearSellCount       int
	PreReportValueOffset  int
	InYearSellValueOffset int
	ReliableScope         *syncmodel.SourceScope
	ForceFallbackScope    bool
}

// appendPerformanceAssetActivities builds one deterministic asset timeline for
// the large-history report performance fixture.
// Authored by: OpenCode
func appendPerformanceAssetActivities(input performanceAssetInput) []syncmodel.ActivityRecord {
	var activities = make([]syncmodel.ActivityRecord, 0, input.PreReportBuyCount+input.InYearSellCount)

	for index := 0; index < input.PreReportBuyCount; index++ {
		var year = 2020 + (index % 5)
		var occurredAt = performanceOccurredAt(year, index)
		var grossValue = input.PreReportValueOffset + (index % 900)
		var feeAmount = (index % 5) + 1

		activities = append(activities, reportActivity(reportActivityInput{
			SourceID:         performanceActivitySourceID(input.AssetSymbol, "buy", index+1),
			OccurredAt:       occurredAt,
			ActivityType:     syncmodel.ActivityTypeBuy,
			AssetIdentityKey: input.AssetIdentityKey,
			AssetSymbol:      input.AssetSymbol,
			AssetName:        input.AssetName,
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  decimalStringFromInt(grossValue),
			OrderFeeAmount:   decimalStringFromInt(feeAmount),
			SourceScope:      performanceAssetScope(input, index),
		}))
	}

	for index := 0; index < input.InYearSellCount; index++ {
		var occurredAt = performanceOccurredAt(2025, index+input.PreReportBuyCount)
		var grossValue = input.InYearSellValueOffset + (index % 700)
		var feeAmount = (index % 5) + 1

		activities = append(activities, reportActivity(reportActivityInput{
			SourceID:         performanceActivitySourceID(input.AssetSymbol, "sell", index+1),
			OccurredAt:       occurredAt,
			ActivityType:     syncmodel.ActivityTypeSell,
			AssetIdentityKey: input.AssetIdentityKey,
			AssetSymbol:      input.AssetSymbol,
			AssetName:        input.AssetName,
			Quantity:         "1",
			OrderCurrency:    "USD",
			OrderGrossValue:  decimalStringFromInt(grossValue),
			OrderFeeAmount:   decimalStringFromInt(feeAmount),
			SourceScope:      performanceAssetScope(input, index+input.PreReportBuyCount),
		}))
	}

	return activities
}

// deterministicExpectedReportsByMethod returns one controlled expected report
// ledger for each supported cost-basis method.
// Authored by: OpenCode
func deterministicExpectedReportsByMethod() map[reportmodel.CostBasisMethod]ExpectedReportLedger {
	return map[reportmodel.CostBasisMethod]ExpectedReportLedger{
		reportmodel.CostBasisMethodFIFO:             buildExpectedPrimaryReportLedger("2935.1", expectedADAFIFO(), expectedLTCFIFO(), expectedAVAXFIFO(), expectedHybridReference(false)),
		reportmodel.CostBasisMethodLIFO:             buildExpectedPrimaryReportLedger("3120.1", expectedADALIFO(), expectedLTCLIFO(), expectedAVAXLIFO(), expectedHybridReference(false)),
		reportmodel.CostBasisMethodHIFO:             buildExpectedPrimaryReportLedger("2720.1", expectedADAHIFO(), expectedLTCHIFO(), expectedAVAXHIFO(), expectedHybridReference(false)),
		reportmodel.CostBasisMethodAverageCost:      buildExpectedPrimaryReportLedger("3025.1", expectedADAAverage(), expectedLTCAverage(), expectedAVAXAverage(), expectedHybridReference(false)),
		reportmodel.CostBasisMethodScopeLocalHybrid: buildExpectedPrimaryReportLedger("3225.1", expectedADAHybrid(), expectedLTCHybrid(), expectedAVAXHybrid(), expectedHybridReference(true)),
	}
}

// buildExpectedPrimaryReportLedger assembles one deterministic expected report
// outcome for the primary report year from shared and method-specific rows.
// Authored by: OpenCode
func buildExpectedPrimaryReportLedger(
	yearlyNetTotal string,
	ada ExpectedAssetDetail,
	ltc ExpectedAssetDetail,
	avax ExpectedAssetDetail,
	includeHybridScopeReference bool,
) ExpectedReportLedger {
	var report = ExpectedReportLedger{
		ReportCalculationCurrency: "USD",
		YearlyNetTotal:            yearlyNetTotal,
		SummaryByAsset: map[string]ExpectedAssetSummary{
			"asset-btc-001":  expectedSummary("asset-btc-001", "BTC", "2990.1"),
			"asset-xrp-001":  expectedSummary("asset-xrp-001", "XRP", "0"),
			"asset-ada-001":  expectedSummary("asset-ada-001", "ADA", netGainFromDetail(ada)),
			"asset-ltc-001":  expectedSummary("asset-ltc-001", "LTC", netGainFromDetail(ltc)),
			"asset-avax-001": expectedSummary("asset-avax-001", "AVAX", netGainFromDetail(avax)),
		},
		ReferenceByAsset: map[string]ExpectedReferenceEntry{
			"asset-eth-001": expectedReference("asset-eth-001", "ETH", 1, reportmodel.ReferenceSectionStatusReferenceOnly),
		},
		DetailByAsset: map[string]ExpectedAssetDetail{
			"asset-btc-001":  expectedBTCDetail(),
			"asset-xrp-001":  expectedXRPDetail(),
			"asset-ada-001":  ada,
			"asset-ltc-001":  ltc,
			"asset-avax-001": avax,
		},
	}

	if includeHybridScopeReference {
		report.ReferenceByAsset["asset-avax-001"] = expectedReference("asset-avax-001", "AVAX", 1, reportmodel.ReferenceSectionStatusIncludedInMainSections)
	}

	return report
}

// expectedSummary creates one concise expected summary row.
// Authored by: OpenCode
func expectedSummary(assetIdentityKey string, displayLabel string, netGainOrLoss string) ExpectedAssetSummary {
	return ExpectedAssetSummary{
		AssetIdentityKey: assetIdentityKey,
		DisplayLabel:     displayLabel,
		NetGainOrLoss:    netGainOrLoss,
	}
}

// expectedReference creates one concise expected reference row.
// Authored by: OpenCode
func expectedReference(assetIdentityKey string, displayLabel string, fullLiquidationCount int, status reportmodel.ReferenceSectionStatus) ExpectedReferenceEntry {
	return ExpectedReferenceEntry{
		AssetIdentityKey:                   assetIdentityKey,
		DisplayLabel:                       displayLabel,
		FullLiquidationCountThroughYearEnd: fullLiquidationCount,
		MainSectionStatus:                  status,
	}
}

// expectedBTCDetail returns the shared expected BTC detail ledger.
// Authored by: OpenCode
func expectedBTCDetail() ExpectedAssetDetail {
	return ExpectedAssetDetail{
		AssetIdentityKey:             "asset-btc-001",
		DisplayLabel:                 "BTC",
		OpeningQuantity:              "2",
		OpeningCostBasis:             "44019.8",
		ClosingQuantity:              "1",
		ClosingCostBasis:             "22009.9",
		CalculationCurrency:          "USD",
		ExpectedFullLiquidationCount: 0,
		ActivityRows: []ExpectedAssetActivityRow{
			expectedPricedRow("btc-sell-2024-zero-fee-001", syncmodel.ActivityTypeSell, "1", "25000", "0", "22009.9", "1"),
		},
		LiquidationSummaries: []ExpectedLiquidationSummary{
			expectedLiquidation("btc-sell-2024-zero-fee-001", "1", "22009.9", "25000", "2990.1"),
		},
	}
}

// expectedXRPDetail returns the shared expected XRP detail ledger.
// Authored by: OpenCode
func expectedXRPDetail() ExpectedAssetDetail {
	return ExpectedAssetDetail{
		AssetIdentityKey:             "asset-xrp-001",
		DisplayLabel:                 "XRP",
		OpeningQuantity:              "0",
		OpeningCostBasis:             "0",
		ClosingQuantity:              "800",
		ClosingCostBasis:             "400.8",
		CalculationCurrency:          "USD",
		ExpectedFullLiquidationCount: 0,
		ActivityRows: []ExpectedAssetActivityRow{
			expectedPricedRow("xrp-buy-2024-001", syncmodel.ActivityTypeBuy, "1000", "500", "1", "501", "1000"),
			expectedHoldingReductionRow("xrp-reduction-2024-001", "200", "400.8", "800", "Bridge migration holding reduction"),
		},
	}
}

// expectedADAFIFO returns the FIFO ADA detail ledger.
// Authored by: OpenCode
func expectedADAFIFO() ExpectedAssetDetail {
	return expectedADADetail("141", "252", "45")
}

// expectedADALIFO returns the LIFO ADA detail ledger.
// Authored by: OpenCode
func expectedADALIFO() ExpectedAssetDetail {
	return expectedADADetail("126", "267", "30")
}

// expectedADAHIFO returns the HIFO ADA detail ledger.
// Authored by: OpenCode
func expectedADAHIFO() ExpectedAssetDetail {
	return expectedADADetail("126", "267", "30")
}

// expectedADAAverage returns the Average Cost Basis ADA detail ledger.
// Authored by: OpenCode
func expectedADAAverage() ExpectedAssetDetail {
	return expectedADADetail("131", "262", "35")
}

// expectedADAHybrid returns the scope-local ADA detail ledger.
// Authored by: OpenCode
func expectedADAHybrid() ExpectedAssetDetail {
	return expectedADADetail("131", "262", "35")
}

// expectedADADetail returns one ADA detail ledger with the method-specific sell
// allocation applied.
// Authored by: OpenCode
func expectedADADetail(closingBasis string, allocatedBasis string, gain string) ExpectedAssetDetail {
	return ExpectedAssetDetail{
		AssetIdentityKey:             "asset-ada-001",
		DisplayLabel:                 "ADA",
		OpeningQuantity:              "0",
		OpeningCostBasis:             "0",
		ClosingQuantity:              "500",
		ClosingCostBasis:             closingBasis,
		CalculationCurrency:          "USD",
		ExpectedFullLiquidationCount: 0,
		ActivityRows: []ExpectedAssetActivityRow{
			expectedPricedRow("ada-buy-2024-001", syncmodel.ActivityTypeBuy, "1000", "250", "2", "252", "1000"),
			expectedPricedRow("ada-buy-2024-002", syncmodel.ActivityTypeBuy, "500", "140", "1", "393", "1500"),
			expectedPricedRow("ada-sell-2024-001", syncmodel.ActivityTypeSell, "1000", "300", "3", closingBasis, "500"),
		},
		LiquidationSummaries: []ExpectedLiquidationSummary{
			expectedLiquidation("ada-sell-2024-001", "1000", allocatedBasis, "297", gain),
		},
	}
}

// expectedLTCFIFO returns the FIFO LTC detail ledger.
// Authored by: OpenCode
func expectedLTCFIFO() ExpectedAssetDetail {
	return expectedLTCDetail("300", "100", "150")
}

// expectedLTCLIFO returns the LIFO LTC detail ledger.
// Authored by: OpenCode
func expectedLTCLIFO() ExpectedAssetDetail {
	return expectedLTCDetail("100", "300", "-50")
}

// expectedLTCHIFO returns the HIFO LTC detail ledger.
// Authored by: OpenCode
func expectedLTCHIFO() ExpectedAssetDetail {
	return expectedLTCDetail("100", "300", "-50")
}

// expectedLTCAverage returns the Average Cost Basis LTC detail ledger.
// Authored by: OpenCode
func expectedLTCAverage() ExpectedAssetDetail {
	return expectedLTCDetail("200", "200", "50")
}

// expectedLTCHybrid returns the scope-local LTC detail ledger.
// Authored by: OpenCode
func expectedLTCHybrid() ExpectedAssetDetail {
	return expectedLTCDetail("200", "200", "50")
}

// expectedLTCDetail returns one LTC detail ledger with the method-specific sell
// allocation applied.
// Authored by: OpenCode
func expectedLTCDetail(closingBasis string, allocatedBasis string, gain string) ExpectedAssetDetail {
	return ExpectedAssetDetail{
		AssetIdentityKey:             "asset-ltc-001",
		DisplayLabel:                 "LTC",
		OpeningQuantity:              "2",
		OpeningCostBasis:             "400",
		ClosingQuantity:              "1",
		ClosingCostBasis:             closingBasis,
		CalculationCurrency:          "USD",
		ExpectedFullLiquidationCount: 0,
		ActivityRows: []ExpectedAssetActivityRow{
			expectedPricedRow("ltc-sell-2024-001", syncmodel.ActivityTypeSell, "1", "250", "0", closingBasis, "1"),
		},
		LiquidationSummaries: []ExpectedLiquidationSummary{
			expectedLiquidation("ltc-sell-2024-001", "1", allocatedBasis, "250", gain),
		},
	}
}

// expectedAVAXFIFO returns the FIFO AVAX detail ledger.
// Authored by: OpenCode
func expectedAVAXFIFO() ExpectedAssetDetail {
	return expectedAVAXDetail("100", "500", "-250", 0)
}

// expectedAVAXLIFO returns the LIFO AVAX detail ledger.
// Authored by: OpenCode
func expectedAVAXLIFO() ExpectedAssetDetail {
	return expectedAVAXDetail("500", "100", "150", 0)
}

// expectedAVAXHIFO returns the HIFO AVAX detail ledger.
// Authored by: OpenCode
func expectedAVAXHIFO() ExpectedAssetDetail {
	return expectedAVAXDetail("100", "500", "-250", 0)
}

// expectedAVAXAverage returns the Average Cost Basis AVAX detail ledger.
// Authored by: OpenCode
func expectedAVAXAverage() ExpectedAssetDetail {
	return expectedAVAXDetail("300", "300", "-50", 0)
}

// expectedAVAXHybrid returns the scope-local AVAX detail ledger.
// Authored by: OpenCode
func expectedAVAXHybrid() ExpectedAssetDetail {
	return expectedAVAXDetail("500", "100", "150", 1)
}

// expectedAVAXDetail returns one AVAX detail ledger with the method-specific
// sell allocation and scope-local liquidation count.
// Authored by: OpenCode
func expectedAVAXDetail(closingBasis string, allocatedBasis string, gain string, fullLiquidationCount int) ExpectedAssetDetail {
	return ExpectedAssetDetail{
		AssetIdentityKey:             "asset-avax-001",
		DisplayLabel:                 "AVAX",
		OpeningQuantity:              "2",
		OpeningCostBasis:             "600",
		ClosingQuantity:              "1",
		ClosingCostBasis:             closingBasis,
		CalculationCurrency:          "USD",
		ExpectedFullLiquidationCount: fullLiquidationCount,
		ActivityRows: []ExpectedAssetActivityRow{
			expectedPricedRow("avax-sell-alpha-2024-001", syncmodel.ActivityTypeSell, "1", "250", "0", closingBasis, "1"),
		},
		LiquidationSummaries: []ExpectedLiquidationSummary{
			expectedLiquidation("avax-sell-alpha-2024-001", "1", allocatedBasis, "250", gain),
		},
	}
}

// expectedHybridReference returns whether the scope-local hybrid method should
// contribute a scope-local reference row for AVAX.
// Authored by: OpenCode
func expectedHybridReference(enabled bool) bool {
	return enabled
}

// netGainFromDetail returns the expected net result from one detail ledger's
// priced liquidation rows.
// Authored by: OpenCode
func netGainFromDetail(detail ExpectedAssetDetail) string {
	if len(detail.LiquidationSummaries) == 0 {
		return "0"
	}

	return detail.LiquidationSummaries[0].GainOrLoss
}

// expectedPricedRow returns one expected priced activity row.
// Authored by: OpenCode
func expectedPricedRow(sourceID string, activityType syncmodel.ActivityType, quantity string, grossValue string, feeAmount string, basisAfterRow string, quantityAfterRow string) ExpectedAssetActivityRow {
	return ExpectedAssetActivityRow{
		SourceID:            sourceID,
		ActivityType:        activityType,
		Quantity:            quantity,
		UnitPrice:           "",
		GrossValue:          grossValue,
		FeeAmount:           feeAmount,
		ActivityCurrency:    "USD",
		BasisAfterRow:       basisAfterRow,
		CalculationCurrency: "USD",
		QuantityAfterRow:    quantityAfterRow,
	}
}

// expectedHoldingReductionRow returns one expected zero-priced holding
// reduction row.
// Authored by: OpenCode
func expectedHoldingReductionRow(sourceID string, quantity string, basisAfterRow string, quantityAfterRow string, explanation string) ExpectedAssetActivityRow {
	return ExpectedAssetActivityRow{
		SourceID:                    sourceID,
		ActivityType:                syncmodel.ActivityTypeSell,
		Quantity:                    quantity,
		UnitPrice:                   "0",
		GrossValue:                  "0",
		FeeAmount:                   "0",
		BasisAfterRow:               basisAfterRow,
		CalculationCurrency:         "USD",
		QuantityAfterRow:            quantityAfterRow,
		HoldingReductionExplanation: explanation,
	}
}

// expectedLiquidation returns one expected priced-liquidation summary row.
// Authored by: OpenCode
func expectedLiquidation(sourceID string, disposedQuantity string, allocatedBasis string, netLiquidationProceeds string, gainOrLoss string) ExpectedLiquidationSummary {
	return ExpectedLiquidationSummary{
		SourceID:               sourceID,
		DisposedQuantity:       disposedQuantity,
		AllocatedBasis:         allocatedBasis,
		NetLiquidationProceeds: netLiquidationProceeds,
		GainOrLoss:             gainOrLoss,
		ActivityCurrency:       "USD",
		CalculationCurrency:    "USD",
	}
}

// reportActivityInput collects one concise test record declaration before it is
// converted into the synced activity model.
// Authored by: OpenCode
type reportActivityInput struct {
	SourceID              string
	OccurredAt            string
	ActivityType          syncmodel.ActivityType
	AssetIdentityKey      string
	AssetSymbol           string
	AssetName             string
	Quantity              string
	OrderCurrency         string
	OrderUnitPrice        string
	OrderGrossValue       string
	OrderFeeAmount        string
	AssetProfileCurrency  string
	AssetProfileUnitPrice string
	AssetProfileFeeAmount string
	BaseCurrency          string
	BaseGrossValue        string
	BaseFeeAmount         string
	Comment               string
	SourceScope           *syncmodel.SourceScope
}

// reportActivity converts one compact fixture declaration into the normalized
// synced activity model used by snapshots and later report calculations.
// Authored by: OpenCode
func reportActivity(input reportActivityInput) syncmodel.ActivityRecord {
	return syncmodel.ActivityRecord{
		SourceID:              input.SourceID,
		OccurredAt:            input.OccurredAt,
		ActivityType:          input.ActivityType,
		AssetIdentityKey:      input.AssetIdentityKey,
		AssetSymbol:           input.AssetSymbol,
		AssetName:             input.AssetName,
		Quantity:              decimalValue(input.Quantity),
		OrderCurrency:         input.OrderCurrency,
		OrderUnitPrice:        decimalPointer(input.OrderUnitPrice),
		OrderGrossValue:       decimalPointer(input.OrderGrossValue),
		OrderFeeAmount:        decimalPointer(input.OrderFeeAmount),
		AssetProfileCurrency:  input.AssetProfileCurrency,
		AssetProfileUnitPrice: decimalPointer(input.AssetProfileUnitPrice),
		AssetProfileFeeAmount: decimalPointer(input.AssetProfileFeeAmount),
		BaseCurrency:          input.BaseCurrency,
		BaseGrossValue:        decimalPointer(input.BaseGrossValue),
		BaseFeeAmount:         decimalPointer(input.BaseFeeAmount),
		Comment:               input.Comment,
		DataSource:            "report-ledger-fixture",
		SourceScope:           input.SourceScope,
		RawHash:               input.SourceID,
	}
}

// reliableAccountScope returns one deterministic account scope for report
// fixture timelines that should preserve stable source scope data.
// Authored by: OpenCode
func reliableAccountScope(id string, name string) *syncmodel.SourceScope {
	return &syncmodel.SourceScope{
		ID:          id,
		Name:        name,
		Kind:        syncmodel.SourceScopeKindAccount,
		Reliability: syncmodel.ScopeReliabilityReliable,
	}
}

// reliableWalletScope returns one deterministic wallet scope for report fixture
// timelines that should preserve stable source scope data.
// Authored by: OpenCode
func reliableWalletScope(id string, name string) *syncmodel.SourceScope {
	return &syncmodel.SourceScope{
		ID:          id,
		Name:        name,
		Kind:        syncmodel.SourceScopeKindWallet,
		Reliability: syncmodel.ScopeReliabilityReliable,
	}
}

// performanceScopeForIndex returns one deterministic scope pattern that keeps
// scope-local fallback active on the large-history performance asset.
// Authored by: OpenCode
func performanceAssetScope(input performanceAssetInput, index int) *syncmodel.SourceScope {
	if input.ForceFallbackScope && index%4 == 0 {
		return nil
	}

	return input.ReliableScope
}

// performanceActivitySourceID returns one stable large-history activity
// identifier.
// Authored by: OpenCode
func performanceActivitySourceID(symbol string, prefix string, sequence int) string {
	return strings.ToLower(symbol) + "-" + prefix + "-performance-" + decimalStringFromInt(sequence)
}

// performanceOccurredAt returns one deterministic timestamp for the large-
// history report performance fixture.
// Authored by: OpenCode
func performanceOccurredAt(year int, index int) string {
	var month = time.Month((index % 12) + 1)
	var day = (index % 28) + 1
	var hour = index % 24
	var minute = index % 60
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC).Format(time.RFC3339)
}

// decimalStringFromInt formats one positive integer for deterministic fixture
// decimal fields.
// Authored by: OpenCode
func decimalStringFromInt(value int) string {
	return strconv.Itoa(value)
}

// decimalValue parses one fixture decimal string and panics when the fixture is
// malformed.
// Authored by: OpenCode
func decimalValue(raw string) apd.Decimal {
	var value, _, err = decimalsupport.ParseString(raw)
	if err != nil {
		panic(err)
	}

	return value
}

// decimalPointer parses one optional fixture decimal string and returns nil for
// the empty string.
// Authored by: OpenCode
func decimalPointer(raw string) *apd.Decimal {
	if raw == "" {
		return nil
	}

	var value = decimalValue(raw)
	return &value
}
