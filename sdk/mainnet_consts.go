package sdk

import (
	"encoding/hex"
	"fmt"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// PublicRPCEndpoints is a list of known public RPC endpoints for mainnet, operated by
// Wormhole guardian nodes.
//
// This list is duplicated a couple times across the codebase - make to to update all copies!
var PublicRPCEndpoints = []string{
	"https://wormhole-v2-mainnet-api.certus.one",
	"https://wormhole.inotel.ro",
	"https://wormhole-v2-mainnet-api.mcf.rocks",
	"https://wormhole-v2-mainnet-api.chainlayer.network",
	"https://wormhole-v2-mainnet-api.staking.fund",
	"https://wormhole-v2-mainnet.01node.com",
}

type (
	EmitterType uint8
)

const (
	EmitterTypeUnset   EmitterType = 0
	EmitterCoreBridge  EmitterType = 1
	EmitterTokenBridge EmitterType = 2
	EmitterNFTBridge   EmitterType = 3
)

func (et EmitterType) String() string {
	switch et {
	case EmitterTypeUnset:
		return "unset"
	case EmitterCoreBridge:
		return "Core"
	case EmitterTokenBridge:
		return "TokenBridge"
	case EmitterNFTBridge:
		return "NFTBridge"
	default:
		return fmt.Sprintf("unknown emitter type: %d", et)
	}
}

type EmitterInfo struct {
	ChainID    vaa.ChainID
	Emitter    string
	BridgeType EmitterType
}

// KnownEmitters is a list of well-known mainnet emitters we want to take into account
// when iterating over all emitters - like for finding and repairing missing messages.
//
// Wormhole is not permissioned - anyone can use it. Adding contracts to this list is
// entirely optional and at the core team's discretion.
var KnownEmitters = buildKnownEmitters(knownTokenbridgeEmitters, knownNFTBridgeEmitters)

func buildKnownEmitters(tokenEmitters, nftEmitters map[vaa.ChainID]string) []EmitterInfo {
	out := make([]EmitterInfo, 0, len(knownTokenbridgeEmitters)+len(knownNFTBridgeEmitters))
	for id, emitter := range tokenEmitters {
		out = append(out, EmitterInfo{
			ChainID:    id,
			Emitter:    emitter,
			BridgeType: EmitterTokenBridge,
		})
	}

	for id, emitter := range nftEmitters {
		out = append(out, EmitterInfo{
			ChainID:    id,
			Emitter:    emitter,
			BridgeType: EmitterNFTBridge,
		})
	}

	return out
}

func buildEmitterMap(hexmap map[vaa.ChainID]string) map[vaa.ChainID][]byte {
	out := make(map[vaa.ChainID][]byte)
	for id, emitter := range hexmap {
		e, err := hex.DecodeString(emitter)
		if err != nil {
			panic(fmt.Sprintf("Failed to decode emitter address %v: %v", emitter, err))
		}
		out[id] = e
	}

	return out
}

// KnownTokenbridgeEmitters is a list of well-known mainnet emitters for the tokenbridge.
var KnownTokenbridgeEmitters = buildEmitterMap(knownTokenbridgeEmitters)
var knownTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana: "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5",
}

// KnownNFTBridgeEmitters is a list of well-known mainnet emitters for the NFT bridge.
var KnownNFTBridgeEmitters = buildEmitterMap(knownNFTBridgeEmitters)
var knownNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana: "0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b",
}

func GetEmitterAddressForChain(chainID vaa.ChainID, emitterType EmitterType) (vaa.Address, error) {
	for _, emitter := range KnownEmitters {
		if emitter.ChainID == chainID && emitter.BridgeType == emitterType {
			emitterAddr, err := vaa.StringToAddress(emitter.Emitter)
			if err != nil {
				return vaa.Address{}, err
			}

			return emitterAddr, nil
		}
	}

	return vaa.Address{}, fmt.Errorf("lookup failed")
}
