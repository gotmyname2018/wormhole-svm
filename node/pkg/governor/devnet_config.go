// This file contains the token and chain config to be used in the devnet environment.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (gov *ChainGovernor) initDevnetConfig() ([]tokenConfigEntry, []chainConfigEntry) {
	gov.logger.Info("setting up devnet config")

	gov.dayLengthInMinutes = 5

	tokens := []tokenConfigEntry{
		tokenConfigEntry{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 8, price: 34.94}, // Addr: So11111111111111111111111111111111111111112, Notional: 4145006
	}

	chains := []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDSolana, dailyLimit: 100, bigTransactionSize: 75},
	}

	return tokens, chains
}
