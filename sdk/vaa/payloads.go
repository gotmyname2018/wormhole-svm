package vaa

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// CoreModule is the identifier of the Core module (which is used for governance messages)
var CoreModule = []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x43, 0x6f, 0x72, 0x65}

// WormholeRelayerModule is the identifier of the Wormhole Relayer module (which is used for governance messages).
// It is the hex representation of "WormholeRelayer" left padded with zeroes.
var WormholeRelayerModule = [32]byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x57, 0x6f, 0x72, 0x6d, 0x68, 0x6f, 0x6c, 0x65, 0x52, 0x65, 0x6c, 0x61, 0x79, 0x65, 0x72,
}
var WormholeRelayerModuleStr = string(WormholeRelayerModule[:])

type GovernanceAction uint8

var (
	// Wormhole core governance actions
	// See e.g. GovernanceStructs.sol for semantic meaning of these
	ActionContractUpgrade    GovernanceAction = 1
	ActionGuardianSetUpdate  GovernanceAction = 2
	ActionCoreSetMessageFee  GovernanceAction = 3
	ActionCoreTransferFees   GovernanceAction = 4
	ActionCoreRecoverChainId GovernanceAction = 5

	// Wormhole tokenbridge governance actions
	ActionRegisterChain             GovernanceAction = 1
	ActionUpgradeTokenBridge        GovernanceAction = 2
	ActionTokenBridgeRecoverChainId GovernanceAction = 3

	// Wormhole relayer governance actions
	WormholeRelayerSetDefaultDeliveryProvider GovernanceAction = 3
)

type (
	// BodyContractUpgrade is a governance message to perform a contract upgrade of the core module
	BodyContractUpgrade struct {
		ChainID     ChainID
		NewContract Address
	}

	// BodyGuardianSetUpdate is a governance message to set a new guardian set
	BodyGuardianSetUpdate struct {
		Keys     []common.Address
		NewIndex uint32
	}

	// BodyTokenBridgeRegisterChain is a governance message to register a chain on the token bridge
	BodyTokenBridgeRegisterChain struct {
		Module         string
		ChainID        ChainID
		EmitterAddress Address
	}

	// BodyTokenBridgeUpgradeContract is a governance message to upgrade the token bridge.
	BodyTokenBridgeUpgradeContract struct {
		Module        string
		TargetChainID ChainID
		NewContract   Address
	}

	// BodyRecoverChainId is a governance message to recover a chain id.
	BodyRecoverChainId struct {
		Module     string
		EvmChainID *uint256.Int
		NewChainID ChainID
	}

	// BodyWormholeRelayerSetDefaultDeliveryProvider is a governance message to set the default relay provider for the Wormhole Relayer.
	BodyWormholeRelayerSetDefaultDeliveryProvider struct {
		ChainID                           ChainID
		NewDefaultDeliveryProviderAddress Address
	}
)

func (b BodyContractUpgrade) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionContractUpgrade)
	// ChainID
	MustWrite(buf, binary.BigEndian, uint16(b.ChainID))

	buf.Write(b.NewContract[:])

	return buf.Bytes()
}

func (b BodyGuardianSetUpdate) Serialize() []byte {
	buf := new(bytes.Buffer)

	// Module
	buf.Write(CoreModule)
	// Action
	MustWrite(buf, binary.BigEndian, ActionGuardianSetUpdate)
	// ChainID - 0 for universal
	MustWrite(buf, binary.BigEndian, uint16(0))

	MustWrite(buf, binary.BigEndian, b.NewIndex)
	MustWrite(buf, binary.BigEndian, uint8(len(b.Keys)))
	for _, k := range b.Keys {
		buf.Write(k[:])
	}

	return buf.Bytes()
}

func (r BodyTokenBridgeRegisterChain) Serialize() []byte {
	payload := &bytes.Buffer{}
	MustWrite(payload, binary.BigEndian, r.ChainID)
	payload.Write(r.EmitterAddress[:])
	// target chain 0 = universal
	return serializeBridgeGovernanceVaa(r.Module, ActionRegisterChain, 0, payload.Bytes())
}

func (r BodyTokenBridgeUpgradeContract) Serialize() []byte {
	return serializeBridgeGovernanceVaa(r.Module, ActionUpgradeTokenBridge, r.TargetChainID, r.NewContract[:])
}

// TBDel
func (r BodyRecoverChainId) Serialize() []byte {
	// Module
	buf := LeftPadBytes(r.Module, 32)
	// Action
	var action GovernanceAction
	if r.Module == "Core" {
		action = ActionCoreRecoverChainId
	} else {
		action = ActionTokenBridgeRecoverChainId
	}
	MustWrite(buf, binary.BigEndian, action)
	// EvmChainID
	MustWrite(buf, binary.BigEndian, r.EvmChainID.Bytes32())
	// NewChainID
	MustWrite(buf, binary.BigEndian, r.NewChainID)
	return buf.Bytes()
}

func (r BodyWormholeRelayerSetDefaultDeliveryProvider) Serialize() []byte {
	payload := &bytes.Buffer{}
	payload.Write(r.NewDefaultDeliveryProviderAddress[:])
	return serializeBridgeGovernanceVaa(WormholeRelayerModuleStr, WormholeRelayerSetDefaultDeliveryProvider, r.ChainID, payload.Bytes())
}

func EmptyPayloadVaa(module string, actionId GovernanceAction, chainId ChainID) []byte {
	return serializeBridgeGovernanceVaa(module, actionId, chainId, []byte{})
}

func serializeBridgeGovernanceVaa(module string, actionId GovernanceAction, chainId ChainID, payload []byte) []byte {
	buf := LeftPadBytes(module, 32)
	// Write action ID
	MustWrite(buf, binary.BigEndian, actionId)
	// Write target chain
	MustWrite(buf, binary.BigEndian, chainId)
	// Write emitter address of chain to be registered
	buf.Write(payload[:])

	return buf.Bytes()
}

// Prepends 0x00 bytes to the payload buffer, up to a size of `length`
func LeftPadBytes(payload string, length int) *bytes.Buffer {
	if length < 0 {
		panic("cannot prepend bytes to a negative length buffer")
	}

	if len(payload) > length {
		panic(fmt.Sprintf("payload longer than %d bytes", length))
	}

	buf := &bytes.Buffer{}

	// Prepend correct number of 0x00 bytes to the payload slice
	for i := 0; i < (length - len(payload)); i++ {
		buf.WriteByte(0x00)
	}

	// add the payload slice
	buf.Write([]byte(payload))

	return buf
}
