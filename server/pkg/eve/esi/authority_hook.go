package esi

import "context"

// OnCorporationAuthorityChanged is wired by jobs to avoid depending on services
// from the ESI package. It runs only after a successful local snapshot write.
var OnCorporationAuthorityChanged func(context.Context, []int64)

func notifyCorporationAuthorityChanged(ctx context.Context, characterIDs []int64) {
	if OnCorporationAuthorityChanged != nil && len(characterIDs) > 0 {
		OnCorporationAuthorityChanged(ctx, characterIDs)
	}
}
