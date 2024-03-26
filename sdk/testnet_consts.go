package sdk

import "github.com/wormhole-foundation/wormhole/sdk/vaa"

// KnownTestnetEmitters is a list of known emitters on the various L1 testnets.
var KnownTestnetEmitters = buildKnownEmitters(knownTestnetTokenbridgeEmitters, knownTestnetNFTBridgeEmitters)

// KnownTestnetTokenbridgeEmitters is a map of known tokenbridge emitters on the various L1 testnets.
var KnownTestnetTokenbridgeEmitters = buildEmitterMap(knownTestnetTokenbridgeEmitters)
var knownTestnetTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana: "3b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98",
}

// KnownTestnetNFTBridgeEmitters is a map  of known NFT emitters on the various L1 testnets.
var KnownTestnetNFTBridgeEmitters = buildEmitterMap(knownTestnetNFTBridgeEmitters)
var knownTestnetNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana: "752a49814e40b96b097207e4b53fdd330544e1e661653fbad4bc159cc28a839e",
}
