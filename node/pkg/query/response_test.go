package query

import (
	"fmt"
	"testing"
	"time"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

///////////// Solana Account Query tests /////////////////////////////////

func createSolanaAccountQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	perChainResponses := []*PerChainQueryResponse{}
	for idx, pcr := range queryRequest.PerChainQueries {
		switch req := pcr.Query.(type) {
		case *SolanaAccountQueryRequest:
			results := []SolanaAccountResult{}
			for idx := range req.Accounts {
				results = append(results, SolanaAccountResult{
					Lamports:   uint64(2000 + idx),
					RentEpoch:  uint64(3000 + idx),
					Executable: (idx%2 == 0),
					Owner:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Data:       []byte([]byte(fmt.Sprintf("Result %d", idx))),
				})
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &SolanaAccountQueryResponse{
					SlotNumber: uint64(1000 + idx),
					BlockTime:  timeForTest(t, time.Now()),
					BlockHash:  ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
					Results:    results,
				},
			})
		default:
			panic("invalid query type!")
		}

	}

	return &QueryResponsePublication{
		Request:           signedQueryRequest,
		PerChainResponses: perChainResponses,
	}
}

func TestSolanaAccountQueryResponseMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaAccountQueryRequestForTesting(t)
	respPub := createSolanaAccountQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

///////////// Solana PDA Query tests /////////////////////////////////

func createSolanaPdaQueryResponseFromRequest(t *testing.T, queryRequest *QueryRequest) *QueryResponsePublication {
	queryRequestBytes, err := queryRequest.Marshal()
	require.NoError(t, err)

	sig := [65]byte{}
	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig[:],
	}

	perChainResponses := []*PerChainQueryResponse{}
	for idx, pcr := range queryRequest.PerChainQueries {
		switch req := pcr.Query.(type) {
		case *SolanaPdaQueryRequest:
			results := []SolanaPdaResult{}
			for idx := range req.PDAs {
				results = append(results, SolanaPdaResult{
					Account:    ethCommon.HexToHash("4fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773e"),
					Bump:       uint8(255 - idx),
					Lamports:   uint64(2000 + idx),
					RentEpoch:  uint64(3000 + idx),
					Executable: (idx%2 == 0),
					Owner:      ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"),
					Data:       []byte([]byte(fmt.Sprintf("Result %d", idx))),
				})
			}
			perChainResponses = append(perChainResponses, &PerChainQueryResponse{
				ChainId: pcr.ChainId,
				Response: &SolanaPdaQueryResponse{
					SlotNumber: uint64(1000 + idx),
					BlockTime:  timeForTest(t, time.Now()),
					BlockHash:  ethCommon.HexToHash("0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e3"),
					Results:    results,
				},
			})
		default:
			panic("invalid query type!")
		}

	}

	return &QueryResponsePublication{
		Request:           signedQueryRequest,
		PerChainResponses: perChainResponses,
	}
}

func TestSolanaPdaQueryResponseMarshalUnmarshal(t *testing.T) {
	queryRequest := createSolanaPdaQueryRequestForTesting(t)
	respPub := createSolanaPdaQueryResponseFromRequest(t, queryRequest)

	respPubBytes, err := respPub.Marshal()
	require.NoError(t, err)

	var respPub2 QueryResponsePublication
	err = respPub2.Unmarshal(respPubBytes)
	require.NoError(t, err)
	require.NotNil(t, respPub2)

	assert.True(t, respPub.Equal(&respPub2))
}

///////////// End of Solana PDA Query tests ///////////////////////////
