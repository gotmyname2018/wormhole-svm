package query

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

// A timestamp has nanos, but we only marshal down to micros, so trim our time to micros for testing purposes.
func timeForTest(t *testing.T, ts time.Time) time.Time {
	t.Helper()
	return time.UnixMicro(ts.UnixMicro())
}

///////////// Solana Account Query tests /////////////////////////////////

func createSolanaAccountQueryRequestForTesting(t *testing.T) *QueryRequest {
	t.Helper()

	callRequest1 := &SolanaAccountQueryRequest{
		Commitment: "finalized",
		Accounts: [][SolanaPublicKeyLength]byte{
			ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
			ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
		},
	}

	perChainQuery1 := &PerChainQueryRequest{
		ChainId: vaa.ChainIDSolana,
		Query:   callRequest1,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery1},
	}

	return queryRequest
}

func TestSolanaAccountQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaAccountQueryRequestForTesting(t)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
}

func TestSolanaAccountQueryRequestMarshalUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("0000000966696e616c697a656400000000000000000000000000000000000000000000000002165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7")
	require.NoError(t, err)

	var solAccountReq SolanaAccountQueryRequest
	err = solAccountReq.Unmarshal(serialized)
	require.NoError(t, err)
}

func TestSolanaQueryMarshalUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("010000002a01000104000000660000000966696e616c697a65640000000000000000000000000000000000000000000000000202c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa95f83a27e90c622a98c037353f271fd8f5f57b4dc18ebf5ff75a934724bd0491")
	require.NoError(t, err)

	var solQuery QueryRequest
	err = solQuery.Unmarshal(serialized)
	require.NoError(t, err)
}

func TestSolanaPublicKeyLengthIsAsExpected(t *testing.T) {
	// It will break the spec if this ever changes!
	require.Equal(t, 32, SolanaPublicKeyLength)
}

///////////// Solana PDA Query tests /////////////////////////////////

func TestSolanaSeedConstsAreAsExpected(t *testing.T) {
	// It might break the spec if these ever changes!
	require.Equal(t, 16, SolanaMaxSeeds)
	require.Equal(t, 32, SolanaMaxSeedLen)
}

func createSolanaPdaQueryRequestForTesting(t *testing.T) *QueryRequest {
	t.Helper()

	callRequest1 := &SolanaPdaQueryRequest{
		Commitment: "finalized",
		PDAs: []SolanaPDAEntry{
			SolanaPDAEntry{
				ProgramAddress: ethCommon.HexToHash("0x02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"), // Devnet core bridge
				Seeds: [][]byte{
					[]byte("GuardianSet"),
					make([]byte, 4),
				},
			},
		},
	}

	perChainQuery1 := &PerChainQueryRequest{
		ChainId: vaa.ChainIDSolana,
		Query:   callRequest1,
	}

	queryRequest := &QueryRequest{
		Nonce:           1,
		PerChainQueries: []*PerChainQueryRequest{perChainQuery1},
	}

	return queryRequest
}

func TestSolanaPdaQueryRequestMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaPdaQueryRequestForTesting(t)
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	var queryRequest2 QueryRequest
	err = queryRequest2.Unmarshal(queryRequestBytes)
	require.NoError(t, err)

	assert.True(t, queryRequest.Equal(&queryRequest2))
}

func TestSolanaPdaQueryUnmarshalFromSDK(t *testing.T) {
	serialized, err := hex.DecodeString("010000002b010001050000005e0000000966696e616c697a656400000000000008ff000000000000000c00000000000000140102c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e5365740000000400000000")
	require.NoError(t, err)

	var solQuery QueryRequest
	err = solQuery.Unmarshal(serialized)
	require.NoError(t, err)
}

///////////// End of Solana PDA Query tests ///////////////////////////
