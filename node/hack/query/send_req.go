// This tool can be used to send various queries to the p2p gossip network.
// It is meant for testing purposes only.

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/tendermint/tendermint/libs/rand"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/gagliardetto/solana-go"
)

// this script has to be run inside kubernetes since it relies on UDP
// https://github.com/kubernetes/kubernetes/issues/47862
// kubectl --namespace=wormhole exec -it spy-0 -- sh -c "cd node/hack/query/ && go run send_req.go"
// one way to iterate inside the container
// kubectl --namespace=wormhole exec -it spy-0 -- bash
// apt update
// apt install nano
// cd node/hack/query
// echo "" > send_req.go
// nano send_req.go
// [paste, ^x, y, enter]
// go run send_req.go

func main() {

	//
	// BEGIN SETUP
	//

	p2pNetworkID := "/wormhole/dev"
	var p2pPort uint = 8998 // don't collide with spy so we can run from the same container in tilt
	p2pBootstrap := "/dns4/guardian-0.guardian/udp/8996/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"
	nodeKeyPath := "./querier.key"

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	signingKeyPath := string("./dev.guardian.key")

	logger.Info("Loading signing key", zap.String("signingKeyPath", signingKeyPath))
	sk, err := common.LoadGuardianKey(signingKeyPath, true)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}
	logger.Info("Signing key loaded", zap.String("publicKey", ethCrypto.PubkeyToAddress(sk.PublicKey).Hex()))

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	// Manual p2p setup
	components := p2p.DefaultComponents()
	components.Port = p2pPort
	bootstrapPeers := p2pBootstrap
	networkID := p2pNetworkID + "/ccq"

	h, err := p2p.NewHost(logger, ctx, networkID, bootstrapPeers, components, priv)
	if err != nil {
		panic(err)
	}

	topic_req := fmt.Sprintf("%s/%s", networkID, "ccq_req")
	topic_resp := fmt.Sprintf("%s/%s", networkID, "ccq_resp")

	logger.Info("Subscribing pubsub topic", zap.String("topic_req", topic_req), zap.String("topic_resp", topic_resp))
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	th_req, err := ps.Join(topic_req)
	if err != nil {
		logger.Panic("failed to join request topic", zap.String("topic_req", topic_req), zap.Error(err))
	}

	th_resp, err := ps.Join(topic_resp)
	if err != nil {
		logger.Panic("failed to join response topic", zap.String("topic_resp", topic_resp), zap.Error(err))
	}

	sub, err := th_resp.Subscribe()
	if err != nil {
		logger.Panic("failed to subscribe to response topic", zap.Error(err))
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))
	// Wait for peers
	for len(th_req.ListPeers()) < 1 {
		time.Sleep(time.Millisecond * 100)
	}

	//
	// END SETUP
	//

	//
	// Solana Tests
	//

	{
		logger.Info("Running Solana account test")

		// Start of query creation...
		account1, err := solana.PublicKeyFromBase58("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
		if err != nil {
			panic("solana account1 is invalid")
		}
		account2, err := solana.PublicKeyFromBase58("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE")
		if err != nil {
			panic("solana account2 is invalid")
		}
		callRequest := &query.SolanaAccountQueryRequest{
			Commitment:      "finalized",
			DataSliceOffset: 0,
			DataSliceLength: 100,
			Accounts:        [][query.SolanaPublicKeyLength]byte{account1, account2},
		}

		queryRequest := &query.QueryRequest{
			Nonce: rand.Uint32(),
			PerChainQueries: []*query.PerChainQueryRequest{
				{
					ChainId: 1,
					Query:   callRequest,
				},
			},
		}
		sendSolanaQueryAndGetRsp(queryRequest, sk, th_req, ctx, logger, sub)
	}

	{
		logger.Info("Running Solana PDA test")

		// Start of query creation...
		callRequest := &query.SolanaPdaQueryRequest{
			Commitment:      "finalized",
			DataSliceOffset: 0,
			DataSliceLength: 100,
			PDAs: []query.SolanaPDAEntry{
				query.SolanaPDAEntry{
					ProgramAddress: ethCommon.HexToHash("0x02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"), // Devnet core bridge
					Seeds: [][]byte{
						[]byte("GuardianSet"),
						make([]byte, 4),
					},
				},
			},
		}

		queryRequest := &query.QueryRequest{
			Nonce: rand.Uint32(),
			PerChainQueries: []*query.PerChainQueryRequest{
				{
					ChainId: 1,
					Query:   callRequest,
				},
			},
		}
		sendSolanaQueryAndGetRsp(queryRequest, sk, th_req, ctx, logger, sub)
	}

	logger.Info("Solana tests complete!")

	// Cleanly shutdown
	// Without this the same host won't properly discover peers until some timeout
	sub.Cancel()
	if err := th_req.Close(); err != nil {
		logger.Fatal("Error closing the request topic", zap.Error(err))
	}
	if err := th_resp.Close(); err != nil {
		logger.Fatal("Error closing the response topic", zap.Error(err))
	}
	if err := h.Close(); err != nil {
		logger.Fatal("Error closing the host", zap.Error(err))
	}

	//
	// END SHUTDOWN
	//

	logger.Info("Success! All tests passed!")
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

func sendSolanaQueryAndGetRsp(queryRequest *query.QueryRequest, sk *ecdsa.PrivateKey, th *pubsub.Topic, ctx context.Context, logger *zap.Logger, sub *pubsub.Subscription) {
	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		panic(err)
	}
	numQueries := len(queryRequest.PerChainQueries)

	// Sign the query request using our private key.
	digest := query.QueryRequestDigest(common.UnsafeDevNet, queryRequestBytes)
	sig, err := ethCrypto.Sign(digest.Bytes(), sk)
	if err != nil {
		panic(err)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig,
	}

	msg := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&msg)
	if err != nil {
		panic(err)
	}

	err = th.Publish(ctx, b)
	if err != nil {
		panic(err)
	}

	logger.Info("Waiting for message...")
	// TODO: max wait time
	// TODO: accumulate signatures to reach quorum
	for {
		envelope, err := sub.Next(ctx)
		if err != nil {
			logger.Panic("failed to receive pubsub message", zap.Error(err))
		}
		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			logger.Info("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
			continue
		}
		var isMatchingResponse bool
		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryResponse:
			logger.Info("query response received", zap.Any("response", m.SignedQueryResponse),
				zap.String("responseBytes", hexutil.Encode(m.SignedQueryResponse.QueryResponse)),
				zap.String("sigBytes", hexutil.Encode(m.SignedQueryResponse.Signature)))
			isMatchingResponse = true

			var response query.QueryResponsePublication
			err := response.Unmarshal(m.SignedQueryResponse.QueryResponse)
			if err != nil {
				logger.Warn("failed to unmarshal response", zap.Error(err))
				break
			}
			if bytes.Equal(response.Request.QueryRequest, queryRequestBytes) && bytes.Equal(response.Request.Signature, sig) {
				// TODO: verify response signature
				isMatchingResponse = true

				if len(response.PerChainResponses) != numQueries {
					logger.Warn("unexpected number of per chain query responses", zap.Int("expectedNum", numQueries), zap.Int("actualNum", len(response.PerChainResponses)))
					break
				}
				// Do double loop over responses
				for index := range response.PerChainResponses {
					switch r := response.PerChainResponses[index].Response.(type) {
					case *query.SolanaAccountQueryResponse:
						logger.Info("solana account query per chain response", zap.Int("index", index), zap.Any("pcr", r))
					case *query.SolanaPdaQueryResponse:
						logger.Info("solana pda query per chain response", zap.Int("index", index), zap.Any("pcr", r))
					default:
						panic(fmt.Sprintf("unsupported query type, should be solana, index: %d", index))
					}
				}
			}
		default:
			continue
		}
		if isMatchingResponse {
			break
		}
	}
}
