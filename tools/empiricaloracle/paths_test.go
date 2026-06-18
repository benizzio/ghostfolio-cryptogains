package main

import (
	"fmt"
	"strings"
	"testing"

	reportmodel "github.com/benizzio/ghostfolio-cryptogains/internal/report/model"
	"github.com/benizzio/ghostfolio-cryptogains/tests/empirical/fixture"
)

// TestGoldenFixtureRelativePathPreservesMultiAssetFormat verifies safe
// multi-asset cases keep the existing fixture filename shape.
// Authored by: OpenCode
func TestGoldenFixtureRelativePathPreservesMultiAssetFormat(t *testing.T) {
	t.Parallel()

	var empiricalCase = fixture.EmpiricalCase{
		CaseID:            " case-multi-asset-2024 ",
		AssetIdentityKeys: []string{"asset-alpha", "asset-beta"},
	}

	var got, err = goldenFixtureRelativePath("testdata/empirical", empiricalCase, reportmodel.CostBasisMethodScopeLocalHybrid, " asset-beta ")
	if err != nil {
		t.Fatalf("build golden fixture path: %v", err)
	}

	var want = "testdata/empirical/golden/scope-local-hybrid/case-multi-asset-2024--asset-beta.json"
	if got != want {
		t.Fatalf("unexpected golden fixture path: got %q want %q", got, want)
	}
}

// TestGoldenFixtureRelativePathRejectsUnsafeCaseID verifies case identifiers
// cannot alter the golden fixture directory structure.
// Authored by: OpenCode
func TestGoldenFixtureRelativePathRejectsUnsafeCaseID(t *testing.T) {
	t.Parallel()

	var unsafeCaseIDs = []string{"", " ", ".", "..", "../case", "case/alpha", `case\alpha`, "case..alpha"}
	var index int
	for index = range unsafeCaseIDs {
		var rawCaseID = unsafeCaseIDs[index]
		t.Run(fmt.Sprintf("case_%d", index), func(t *testing.T) {
			t.Parallel()

			var empiricalCase = fixture.EmpiricalCase{
				CaseID:            rawCaseID,
				AssetIdentityKeys: []string{"asset-alpha"},
			}

			var _, err = goldenFixtureRelativePath("testdata/empirical", empiricalCase, reportmodel.CostBasisMethodFIFO, "asset-alpha")
			if err == nil {
				t.Fatalf("expected unsafe case_id %q to be rejected", rawCaseID)
			}
			if !strings.Contains(err.Error(), "case_id") {
				t.Fatalf("expected case_id context in error, got %v", err)
			}
		})
	}
}

// TestGoldenFixtureRelativePathRejectsUnsafeAssetIdentityKey verifies asset
// identifiers cannot alter the golden fixture directory structure.
// Authored by: OpenCode
func TestGoldenFixtureRelativePathRejectsUnsafeAssetIdentityKey(t *testing.T) {
	t.Parallel()

	var unsafeAssetIdentityKeys = []string{"", " ", ".", "..", "../asset", "asset/beta", `asset\beta`, "asset..beta"}
	var index int
	for index = range unsafeAssetIdentityKeys {
		var rawAssetIdentityKey = unsafeAssetIdentityKeys[index]
		t.Run(fmt.Sprintf("asset_%d", index), func(t *testing.T) {
			t.Parallel()

			var empiricalCase = fixture.EmpiricalCase{
				CaseID:            "case-multi-asset-2024",
				AssetIdentityKeys: []string{"asset-alpha", rawAssetIdentityKey},
			}

			var _, err = goldenFixtureRelativePath("testdata/empirical", empiricalCase, reportmodel.CostBasisMethodFIFO, rawAssetIdentityKey)
			if err == nil {
				t.Fatalf("expected unsafe asset_identity_key %q to be rejected", rawAssetIdentityKey)
			}
			if !strings.Contains(err.Error(), "asset_identity_key") {
				t.Fatalf("expected asset_identity_key context in error, got %v", err)
			}
		})
	}
}
