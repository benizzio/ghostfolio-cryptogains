// Package store defines the protected snapshot persistence boundary.
// Authored by: OpenCode
package store

import (
	"bytes"
	"context"

	snapshotenvelope "github.com/benizzio/ghostfolio-cryptogains/internal/snapshot/envelope"
)

// DiscoverServerCandidates filters discovered snapshot headers to the currently
// selected Ghostfolio server.
//
// Example:
//
//	candidates, err := store.DiscoverServerCandidates(context.Background(), snapshots, "https://ghostfol.io")
//	if err != nil {
//		panic(err)
//	}
//	_ = len(candidates)
//
// Use this helper before token unlock attempts so the runtime only tries
// selected-server snapshot candidates.
// Authored by: OpenCode
func DiscoverServerCandidates(ctx context.Context, snapshots Store, serverOrigin string) ([]Candidate, error) {
	if snapshots == nil {
		return []Candidate{}, nil
	}

	candidates, err := snapshots.Candidates(ctx)
	if err != nil {
		return nil, err
	}

	var selectedServerKey = snapshotenvelope.DeriveServerDiscoveryKey(serverOrigin)
	var filtered []Candidate
	for _, candidate := range candidates {
		if bytes.Equal(candidate.Header.ServerDiscoveryKey, selectedServerKey) {
			filtered = append(filtered, candidate)
		}
	}

	return filtered, nil
}
