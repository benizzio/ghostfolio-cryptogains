package validator

import (
	"testing"

	"github.com/benizzio/ghostfolio-cryptogains/internal/ghostfolio/dto"
)

func TestValidateAuthResponseCoversSuccessAndFailure(t *testing.T) {
	t.Parallel()

	if err := ValidateAuthResponse(dto.AuthResponse{}); err == nil {
		t.Fatalf("expected empty token to fail")
	}
	if err := ValidateAuthResponse(dto.AuthResponse{AuthToken: "jwt"}); err != nil {
		t.Fatalf("expected auth token to pass: %v", err)
	}
}

func TestValidateActivitiesProbeResponseCoversBranches(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		response dto.ActivitiesProbeResponse
		wantErr  bool
	}{
		{name: "negative count", response: dto.ActivitiesProbeResponse{Count: -1}, wantErr: true},
		{name: "more than one activity", response: dto.ActivitiesProbeResponse{Count: 2, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "2026-01-31T10:00:00Z", Type: "BUY"}, {ID: "2", Date: "2026-01-31T10:00:00Z", Type: "SELL"}}}, wantErr: true},
		{name: "count zero with activity", response: dto.ActivitiesProbeResponse{Count: 0, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "2026-01-31T10:00:00Z", Type: "BUY"}}}, wantErr: true},
		{name: "count positive without activity", response: dto.ActivitiesProbeResponse{Count: 1}, wantErr: true},
		{name: "empty history", response: dto.ActivitiesProbeResponse{Count: 0, Activities: []dto.ActivityProbeEntry{}}, wantErr: false},
		{name: "missing id", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{Date: "2026-01-31T10:00:00Z", Type: "BUY"}}}, wantErr: true},
		{name: "missing type", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "2026-01-31T10:00:00Z"}}}, wantErr: true},
		{name: "missing date", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{ID: "1", Type: "BUY"}}}, wantErr: true},
		{name: "invalid date", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "bad", Type: "BUY"}}}, wantErr: true},
		{name: "valid activity with fractional seconds", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "2026-01-31T10:00:00.000Z", Type: "BUY"}}}, wantErr: false},
		{name: "valid activity", response: dto.ActivitiesProbeResponse{Count: 1, Activities: []dto.ActivityProbeEntry{{ID: "1", Date: "2026-01-31T10:00:00Z", Type: "BUY"}}}, wantErr: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateActivitiesProbeResponse(testCase.response)
			if testCase.wantErr && err == nil {
				t.Fatalf("expected validation error")
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("expected validation success, got %v", err)
			}
		})
	}
}
