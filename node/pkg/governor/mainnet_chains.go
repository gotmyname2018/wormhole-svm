// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		{emitterChainID: vaa.ChainIDSolana, dailyLimit: 25_000_000, bigTransactionSize: 2_500_000},
	}
}
