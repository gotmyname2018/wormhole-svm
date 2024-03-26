package vaa

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

var addr = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
var dummyBytes = [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}

func TestCoreModule(t *testing.T) {
	hexifiedCoreModule := "00000000000000000000000000000000000000000000000000000000436f7265"
	assert.Equal(t, hex.EncodeToString(CoreModule), hexifiedCoreModule)
}

func TestBodyContractUpgrade(t *testing.T) {
	test := BodyContractUpgrade{ChainID: 1, NewContract: addr}
	assert.Equal(t, test.ChainID, ChainID(1))
	assert.Equal(t, test.NewContract, addr)
}

func TestBodyGuardianSetUpdate(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
	}
	test := BodyGuardianSetUpdate{Keys: keys, NewIndex: uint32(1)}
	assert.Equal(t, test.Keys, keys)
	assert.Equal(t, test.NewIndex, uint32(1))
}

func TestBodyTokenBridgeRegisterChain(t *testing.T) {
	module := "test"
	test := BodyTokenBridgeRegisterChain{Module: module, ChainID: 1, EmitterAddress: addr}
	assert.Equal(t, test.Module, module)
	assert.Equal(t, test.ChainID, ChainID(1))
	assert.Equal(t, test.EmitterAddress, addr)
}

func TestBodyTokenBridgeUpgradeContract(t *testing.T) {
	module := "test"
	test := BodyTokenBridgeUpgradeContract{Module: module, TargetChainID: 1, NewContract: addr}
	assert.Equal(t, test.Module, module)
	assert.Equal(t, test.TargetChainID, ChainID(1))
	assert.Equal(t, test.NewContract, addr)
}

func TestBodyContractUpgradeSerialize(t *testing.T) {
	bodyContractUpgrade := BodyContractUpgrade{ChainID: 1, NewContract: addr}
	expected := "00000000000000000000000000000000000000000000000000000000436f72650100010000000000000000000000000000000000000000000000000000000000000004"
	serializedBodyContractUpgrade := bodyContractUpgrade.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyContractUpgrade))
}

func TestBodyGuardianSetUpdateSerialize(t *testing.T) {
	keys := []common.Address{
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed"),
		common.HexToAddress("0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"),
	}
	bodyGuardianSetUpdate := BodyGuardianSetUpdate{Keys: keys, NewIndex: uint32(1)}
	expected := "00000000000000000000000000000000000000000000000000000000436f726502000000000001025aaeb6053f3e94c9b9a09f33669435e7ef1beaed5aaeb6053f3e94c9b9a09f33669435e7ef1beaee"
	serializedBodyGuardianSetUpdate := bodyGuardianSetUpdate.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyGuardianSetUpdate))
}

func TestBodyTokenBridgeRegisterChainSerialize(t *testing.T) {
	module := "test"
	tests := []struct {
		name     string
		expected string
		object   BodyTokenBridgeRegisterChain
		panic    bool
	}{
		{
			name:     "working_as_expected",
			panic:    false,
			object:   BodyTokenBridgeRegisterChain{Module: module, ChainID: 1, EmitterAddress: addr},
			expected: "000000000000000000000000000000000000000000000000000000007465737401000000010000000000000000000000000000000000000000000000000000000000000004",
		},
		{
			name:     "panic_at_the_disco!",
			panic:    true,
			object:   BodyTokenBridgeRegisterChain{Module: "123456789012345678901234567890123", ChainID: 1, EmitterAddress: addr},
			expected: "payload longer than 32 bytes",
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.panic {
				assert.PanicsWithValue(t, testCase.expected, func() { testCase.object.Serialize() })
			} else {
				assert.Equal(t, testCase.expected, hex.EncodeToString(testCase.object.Serialize()))
			}
		})
	}
}

func TestBodyTokenBridgeUpgradeContractSerialize(t *testing.T) {
	module := "test"
	bodyTokenBridgeUpgradeContract := BodyTokenBridgeUpgradeContract{Module: module, TargetChainID: 1, NewContract: addr}
	expected := "00000000000000000000000000000000000000000000000000000000746573740200010000000000000000000000000000000000000000000000000000000000000004"
	serializedBodyTokenBridgeUpgradeContract := bodyTokenBridgeUpgradeContract.Serialize()
	assert.Equal(t, expected, hex.EncodeToString(serializedBodyTokenBridgeUpgradeContract))
}

func TestLeftPadBytes(t *testing.T) {
	payload := "AAAA"
	paddedPayload := LeftPadBytes(payload, int(8))

	buf := &bytes.Buffer{}
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.WriteByte(0x00)
	buf.Write([]byte(payload))

	assert.Equal(t, paddedPayload, buf)
}

func FuzzLeftPadBytes(f *testing.F) {
	// Add examples to our fuzz corpus
	f.Add("FOO", 8)
	f.Add("123", 8)

	f.Fuzz(func(t *testing.T, payload string, length int) {
		// We know length could be negative, but we panic if it is in the implementation
		if length < 0 {
			t.Skip()
		}

		// We know we cannot left pad something shorter than the payload being provided, but we panic if it is
		if len(payload) > length {
			t.Skip()
		}

		paddedPayload := LeftPadBytes(payload, length)

		// paddedPayload must always be equal to length
		assert.Equal(t, paddedPayload.Len(), length)
	})
}

func TestBodyWormholeRelayerSetDefaultDeliveryProviderSerialize(t *testing.T) {
	expected := "0000000000000000000000000000000000576f726d686f6c6552656c617965720300040000000000000000000000000000000000000000000000000000000000000004"
	bodyWormholeRelayerSetDefaultDeliveryProvider := BodyWormholeRelayerSetDefaultDeliveryProvider{
		ChainID:                           4,
		NewDefaultDeliveryProviderAddress: addr,
	}
	assert.Equal(t, expected, hex.EncodeToString(bodyWormholeRelayerSetDefaultDeliveryProvider.Serialize()))
}

func TestBodyCoreRecoverChainIdSerialize(t *testing.T) {
	expected := "00000000000000000000000000000000000000000000000000000000436f72650500000000000000000000000000000000000000000000000000000000000000010fa0"
	BodyRecoverChainId := BodyRecoverChainId{
		Module:     "Core",
		EvmChainID: uint256.NewInt(1),
		NewChainID: 4000,
	}
	assert.Equal(t, expected, hex.EncodeToString(BodyRecoverChainId.Serialize()))
}

func TestBodyTokenBridgeRecoverChainIdSerialize(t *testing.T) {
	expected := "000000000000000000000000000000000000000000546f6b656e4272696467650300000000000000000000000000000000000000000000000000000000000000010fa0"
	BodyRecoverChainId := BodyRecoverChainId{
		Module:     "TokenBridge",
		EvmChainID: uint256.NewInt(1),
		NewChainID: 4000,
	}
	assert.Equal(t, expected, hex.EncodeToString(BodyRecoverChainId.Serialize()))
}
