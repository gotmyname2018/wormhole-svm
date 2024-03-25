package adminrpc

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/holiman/uint256"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/exp/slices"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/certusone/wormhole/node/pkg/common"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	vaaInjectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_vaa_injections_total",
			Help: "Total number of injected VAA queued for broadcast",
		})
)

type nodePrivilegedService struct {
	nodev1.UnimplementedNodePrivilegedServiceServer
	db              *db.Database
	injectC         chan<- *common.MessagePublication
	obsvReqSendC    chan<- *gossipv1.ObservationRequest
	logger          *zap.Logger
	signedInC       chan<- *gossipv1.SignedVAAWithQuorum
	governor        *governor.ChainGovernor
	gsCache         sync.Map
	gk              *ecdsa.PrivateKey
	guardianAddress ethcommon.Address
	rpcMap          map[string]string
}

func NewPrivService(
	db *db.Database,
	injectC chan<- *common.MessagePublication,
	obsvReqSendC chan<- *gossipv1.ObservationRequest,
	logger *zap.Logger,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	governor *governor.ChainGovernor,
	gk *ecdsa.PrivateKey,
	guardianAddress ethcommon.Address,
	rpcMap map[string]string,

) *nodePrivilegedService {
	return &nodePrivilegedService{
		db:              db,
		injectC:         injectC,
		obsvReqSendC:    obsvReqSendC,
		logger:          logger,
		signedInC:       signedInC,
		governor:        governor,
		gk:              gk,
		guardianAddress: guardianAddress,
		rpcMap:          rpcMap,
	}
}

// adminGuardianSetUpdateToVAA converts a nodev1.GuardianSetUpdate message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminGuardianSetUpdateToVAA(req *nodev1.GuardianSetUpdate, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if len(req.Guardians) == 0 {
		return nil, errors.New("empty guardian set specified")
	}

	if len(req.Guardians) > common.MaxGuardianCount {
		return nil, fmt.Errorf("too many guardians - %d, maximum is %d", len(req.Guardians), common.MaxGuardianCount)
	}

	addrs := make([]ethcommon.Address, len(req.Guardians))
	for i, g := range req.Guardians {
		if !ethcommon.IsHexAddress(g.Pubkey) {
			return nil, fmt.Errorf("invalid pubkey format at index %d (%s)", i, g.Name)
		}

		ethAddr := ethcommon.HexToAddress(g.Pubkey)
		for j, pk := range addrs {
			if pk == ethAddr {
				return nil, fmt.Errorf("duplicate pubkey at index %d (duplicate of %d): %s", i, j, g.Name)
			}
		}

		addrs[i] = ethAddr
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyGuardianSetUpdate{
			Keys:     addrs,
			NewIndex: guardianSetIndex + 1,
		}.Serialize())

	return v, nil
}

// adminContractUpgradeToVAA converts a nodev1.ContractUpgrade message to its canonical VAA representation.
// Returns an error if the data is invalid.
func adminContractUpgradeToVAA(req *nodev1.ContractUpgrade, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	b, err := hex.DecodeString(req.NewContract)
	if err != nil {
		return nil, errors.New("invalid new contract address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new_contract address")
	}

	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid chain_id")
	}

	newContractAddress := vaa.Address{}
	copy(newContractAddress[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyContractUpgrade{
			ChainID:     vaa.ChainID(req.ChainId),
			NewContract: newContractAddress,
		}.Serialize())

	return v, nil
}

// tokenBridgeRegisterChain converts a nodev1.TokenBridgeRegisterChain message to its canonical VAA representation.
// Returns an error if the data is invalid.
func tokenBridgeRegisterChain(req *nodev1.BridgeRegisterChain, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid chain_id")
	}

	b, err := hex.DecodeString(req.EmitterAddress)
	if err != nil {
		return nil, errors.New("invalid emitter address encoding (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid emitter address (expected 32 bytes)")
	}

	emitterAddress := vaa.Address{}
	copy(emitterAddress[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyTokenBridgeRegisterChain{
			Module:         req.Module,
			ChainID:        vaa.ChainID(req.ChainId),
			EmitterAddress: emitterAddress,
		}.Serialize())

	return v, nil
}

// recoverChainId converts a nodev1.RecoverChainId message to its canonical VAA representation.
// Returns an error if the data is invalid.
func recoverChainId(req *nodev1.RecoverChainId, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	evm_chain_id_big := big.NewInt(0) //TBDel
	evm_chain_id_big, ok := evm_chain_id_big.SetString(req.EvmChainId, 10)
	if !ok {
		return nil, errors.New("invalid evm_chain_id")
	}

	// uint256 has Bytes32 method for easier serialization
	evm_chain_id, overflow := uint256.FromBig(evm_chain_id_big)
	if overflow {
		return nil, errors.New("evm_chain_id overflow")
	}

	if req.NewChainId > math.MaxUint16 {
		return nil, errors.New("invalid new_chain_id")
	}

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyRecoverChainId{
			Module:     req.Module,
			EvmChainID: evm_chain_id,
			NewChainID: vaa.ChainID(req.NewChainId),
		}.Serialize())

	return v, nil
}

// tokenBridgeUpgradeContract converts a nodev1.TokenBridgeRegisterChain message to its canonical VAA representation.
// Returns an error if the data is invalid.
func tokenBridgeUpgradeContract(req *nodev1.BridgeUpgradeContract, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.TargetChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}

	b, err := hex.DecodeString(req.NewContract)
	if err != nil {
		return nil, errors.New("invalid new contract address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new contract address (expected 32 bytes)")
	}

	newContract := vaa.Address{}
	copy(newContract[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyTokenBridgeUpgradeContract{
			Module:        req.Module,
			TargetChainID: vaa.ChainID(req.TargetChainId),
			NewContract:   newContract,
		}.Serialize())

	return v, nil
}

// wormholeRelayerSetDefaultDeliveryProvider converts a nodev1.WormholeRelayerSetDefaultDeliveryProvider message to its canonical VAA representation.
// Returns an error if the data is invalid.
func wormholeRelayerSetDefaultDeliveryProvider(req *nodev1.WormholeRelayerSetDefaultDeliveryProvider, timestamp time.Time, guardianSetIndex uint32, nonce uint32, sequence uint64) (*vaa.VAA, error) {
	if req.ChainId > math.MaxUint16 {
		return nil, errors.New("invalid target_chain_id")
	}

	b, err := hex.DecodeString(req.NewDefaultDeliveryProviderAddress)
	if err != nil {
		return nil, errors.New("invalid new default delivery provider address (expected hex)")
	}

	if len(b) != 32 {
		return nil, errors.New("invalid new default delivery provider address (expected 32 bytes)")
	}

	NewDefaultDeliveryProviderAddress := vaa.Address{}
	copy(NewDefaultDeliveryProviderAddress[:], b)

	v := vaa.CreateGovernanceVAA(timestamp, nonce, sequence, guardianSetIndex,
		vaa.BodyWormholeRelayerSetDefaultDeliveryProvider{
			ChainID:                           vaa.ChainID(req.ChainId),
			NewDefaultDeliveryProviderAddress: NewDefaultDeliveryProviderAddress,
		}.Serialize())

	return v, nil
}

func GovMsgToVaa(message *nodev1.GovernanceMessage, currentSetIndex uint32, timestamp time.Time) (*vaa.VAA, error) {
	var (
		v   *vaa.VAA
		err error
	)

	switch payload := message.Payload.(type) {
	case *nodev1.GovernanceMessage_GuardianSet:
		v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_ContractUpgrade:
		v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_BridgeRegisterChain:
		v, err = tokenBridgeRegisterChain(payload.BridgeRegisterChain, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_BridgeContractUpgrade:
		v, err = tokenBridgeUpgradeContract(payload.BridgeContractUpgrade, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_RecoverChainId:
		v, err = recoverChainId(payload.RecoverChainId, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	case *nodev1.GovernanceMessage_WormholeRelayerSetDefaultDeliveryProvider:
		v, err = wormholeRelayerSetDefaultDeliveryProvider(payload.WormholeRelayerSetDefaultDeliveryProvider, timestamp, currentSetIndex, message.Nonce, message.Sequence)
	default:
		panic(fmt.Sprintf("unsupported VAA type: %T", payload))
	}

	return v, err
}

func (s *nodePrivilegedService) InjectGovernanceVAA(ctx context.Context, req *nodev1.InjectGovernanceVAARequest) (*nodev1.InjectGovernanceVAAResponse, error) {
	s.logger.Info("governance VAA injected via admin socket", zap.String("request", req.String()))

	var (
		v   *vaa.VAA
		err error
	)

	timestamp := time.Unix(int64(req.Timestamp), 0)

	digests := make([][]byte, len(req.Messages))

	for i, message := range req.Messages {
		v, err = GovMsgToVaa(message, req.CurrentSetIndex, timestamp)

		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		// Generate digest of the unsigned VAA.
		digest := v.SigningDigest()

		s.logger.Info("governance VAA constructed",
			zap.Any("vaa", v),
			zap.String("digest", digest.String()),
		)

		vaaInjectionsTotal.Inc()

		s.injectC <- &common.MessagePublication{
			TxHash:           ethcommon.Hash{},
			Timestamp:        v.Timestamp,
			Nonce:            v.Nonce,
			Sequence:         v.Sequence,
			ConsistencyLevel: v.ConsistencyLevel,
			EmitterChain:     v.EmitterChain,
			EmitterAddress:   v.EmitterAddress,
			Payload:          v.Payload,
			Unreliable:       false,
		}

		digests[i] = digest.Bytes()
	}

	return &nodev1.InjectGovernanceVAAResponse{Digests: digests}, nil
}

// fetchMissing attempts to backfill a gap by fetching and storing missing signed VAAs from the network.
// Returns true if the gap was filled, false otherwise.
func (s *nodePrivilegedService) fetchMissing(
	ctx context.Context,
	nodes []string,
	c *http.Client,
	chain vaa.ChainID,
	addr string,
	seq uint64) (bool, error) {

	// shuffle the list of public RPC endpoints
	rand.Shuffle(len(nodes), func(i, j int) {
		nodes[i], nodes[j] = nodes[j], nodes[i]
	})

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	for _, node := range nodes {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(
			"%s/v1/signed_vaa/%d/%s/%d", node, chain, addr, seq), nil)
		if err != nil {
			return false, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.Do(req)
		if err != nil {
			s.logger.Warn("failed to fetch missing VAA",
				zap.String("node", node),
				zap.String("chain", chain.String()),
				zap.String("address", addr),
				zap.Uint64("sequence", seq),
				zap.Error(err),
			)
			continue
		}

		switch resp.StatusCode {
		case http.StatusNotFound:
			resp.Body.Close()
			continue
		case http.StatusOK:
			type getVaaResp struct {
				VaaBytes string `json:"vaaBytes"`
			}
			var respBody getVaaResp
			if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
				resp.Body.Close()
				s.logger.Warn("failed to decode VAA response",
					zap.String("node", node),
					zap.String("chain", chain.String()),
					zap.String("address", addr),
					zap.Uint64("sequence", seq),
					zap.Error(err),
				)
				continue
			}

			// base64 decode the VAA bytes
			vaaBytes, err := base64.StdEncoding.DecodeString(respBody.VaaBytes)
			if err != nil {
				resp.Body.Close()
				s.logger.Warn("failed to decode VAA body",
					zap.String("node", node),
					zap.String("chain", chain.String()),
					zap.String("address", addr),
					zap.Uint64("sequence", seq),
					zap.Error(err),
				)
				continue
			}

			s.logger.Info("backfilled VAA",
				zap.Uint16("chain", uint16(chain)),
				zap.String("address", addr),
				zap.Uint64("sequence", seq),
				zap.Int("numBytes", len(vaaBytes)),
			)

			// Inject into the gossip signed VAA receive path.
			// This has the same effect as if the VAA was received from the network
			// (verifying signature, storing in local DB...).
			s.signedInC <- &gossipv1.SignedVAAWithQuorum{
				Vaa: vaaBytes,
			}

			resp.Body.Close()
			return true, nil
		default:
			resp.Body.Close()
			return false, fmt.Errorf("unexpected response status: %d", resp.StatusCode)
		}
	}

	return false, nil
}

func (s *nodePrivilegedService) FindMissingMessages(ctx context.Context, req *nodev1.FindMissingMessagesRequest) (*nodev1.FindMissingMessagesResponse, error) {
	b, err := hex.DecodeString(req.EmitterAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid emitter address encoding: %v", err)
	}
	emitterAddress := vaa.Address{}
	copy(emitterAddress[:], b)

	ids, first, last, err := s.db.FindEmitterSequenceGap(db.VAAID{
		EmitterChain:   vaa.ChainID(req.EmitterChain),
		EmitterAddress: emitterAddress,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database operation failed: %v", err)
	}

	if req.RpcBackfill {
		c := &http.Client{}
		unfilled := make([]uint64, 0, len(ids))
		for _, id := range ids {
			if ok, err := s.fetchMissing(ctx, req.BackfillNodes, c, vaa.ChainID(req.EmitterChain), emitterAddress.String(), id); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to backfill VAA: %v", err)
			} else if ok {
				continue
			}
			unfilled = append(unfilled, id)
		}
		ids = unfilled
	}

	resp := make([]string, len(ids))
	for i, v := range ids {
		resp[i] = fmt.Sprintf("%d/%s/%d", req.EmitterChain, emitterAddress, v)
	}
	return &nodev1.FindMissingMessagesResponse{
		MissingMessages: resp,
		FirstSequence:   first,
		LastSequence:    last,
	}, nil
}

func (s *nodePrivilegedService) SendObservationRequest(ctx context.Context, req *nodev1.SendObservationRequestRequest) (*nodev1.SendObservationRequestResponse, error) {
	if err := common.PostObservationRequest(s.obsvReqSendC, req.ObservationRequest); err != nil {
		return nil, err
	}

	s.logger.Info("sent observation request", zap.Any("request", req.ObservationRequest))
	return &nodev1.SendObservationRequestResponse{}, nil
}

func (s *nodePrivilegedService) ChainGovernorStatus(ctx context.Context, req *nodev1.ChainGovernorStatusRequest) (*nodev1.ChainGovernorStatusResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	return &nodev1.ChainGovernorStatusResponse{
		Response: s.governor.Status(),
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorReload(ctx context.Context, req *nodev1.ChainGovernorReloadRequest) (*nodev1.ChainGovernorReloadResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	resp, err := s.governor.Reload()
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorReloadResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorDropPendingVAA(ctx context.Context, req *nodev1.ChainGovernorDropPendingVAARequest) (*nodev1.ChainGovernorDropPendingVAAResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	resp, err := s.governor.DropPendingVAA(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorDropPendingVAAResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorReleasePendingVAA(ctx context.Context, req *nodev1.ChainGovernorReleasePendingVAARequest) (*nodev1.ChainGovernorReleasePendingVAAResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	resp, err := s.governor.ReleasePendingVAA(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorReleasePendingVAAResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) ChainGovernorResetReleaseTimer(ctx context.Context, req *nodev1.ChainGovernorResetReleaseTimerRequest) (*nodev1.ChainGovernorResetReleaseTimerResponse, error) {
	if s.governor == nil {
		return nil, fmt.Errorf("chain governor is not enabled")
	}

	if len(req.VaaId) == 0 {
		return nil, fmt.Errorf("the VAA id must be specified as \"chainId/emitterAddress/seqNum\"")
	}

	resp, err := s.governor.ResetReleaseTimer(req.VaaId)
	if err != nil {
		return nil, err
	}

	return &nodev1.ChainGovernorResetReleaseTimerResponse{
		Response: resp,
	}, nil
}

func (s *nodePrivilegedService) SignExistingVAA(ctx context.Context, req *nodev1.SignExistingVAARequest) (*nodev1.SignExistingVAAResponse, error) {
	v, err := vaa.Unmarshal(req.Vaa)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal VAA: %w", err)
	}

	if req.NewGuardianSetIndex <= v.GuardianSetIndex {
		return nil, errors.New("new guardian set index must be higher than provided VAA")
	}

	var gs *common.GuardianSet
	if cachedGs, exists := s.gsCache.Load(v.GuardianSetIndex); exists {
		var ok bool
		gs, ok = cachedGs.(*common.GuardianSet)
		if !ok {
			return nil, fmt.Errorf("internal error")
		}
	} else {
		/* TBDel
		evmGs, err := s.evmConnector.GetGuardianSet(ctx, v.GuardianSetIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to load guardian set [%d]: %w", v.GuardianSetIndex, err)
		}
		gs = &common.GuardianSet{
			Keys:  evmGs.Keys,
			Index: v.GuardianSetIndex,
		}
		s.gsCache.Store(v.GuardianSetIndex, gs)
		*/
	}

	if slices.Index(gs.Keys, s.guardianAddress) != -1 {
		return nil, fmt.Errorf("local guardian is already on the old set")
	}

	// Verify VAA
	err = v.Verify(gs.Keys)
	if err != nil {
		return nil, fmt.Errorf("failed to verify existing VAA: %w", err)
	}

	if len(req.NewGuardianAddrs) > 255 {
		return nil, errors.New("new guardian set has too many guardians")
	}
	newGS := make([]ethcommon.Address, len(req.NewGuardianAddrs))
	for i, guardianString := range req.NewGuardianAddrs {
		guardianAddress := ethcommon.HexToAddress(guardianString)
		newGS[i] = guardianAddress
	}

	// Make sure there are no duplicates. Compact needs to take a sorted slice to remove all duplicates.
	newGSSorted := slices.Clone(newGS)
	slices.SortFunc(newGSSorted, func(a, b ethcommon.Address) int {
		return bytes.Compare(a[:], b[:])
	})
	newGsLen := len(newGSSorted)
	if len(slices.Compact(newGSSorted)) != newGsLen {
		return nil, fmt.Errorf("duplicate guardians in the guardian set")
	}

	localGuardianIndex := slices.Index(newGS, s.guardianAddress)
	if localGuardianIndex == -1 {
		return nil, fmt.Errorf("local guardian is not a member of the new guardian set")
	}

	newVAA := &vaa.VAA{
		Version: v.Version,
		// Set the new guardian set index
		GuardianSetIndex: req.NewGuardianSetIndex,
		// Signatures will be repopulated
		Signatures:       nil,
		Timestamp:        v.Timestamp,
		Nonce:            v.Nonce,
		Sequence:         v.Sequence,
		ConsistencyLevel: v.ConsistencyLevel,
		EmitterChain:     v.EmitterChain,
		EmitterAddress:   v.EmitterAddress,
		Payload:          v.Payload,
	}

	// Copy original VAA signatures
	for _, sig := range v.Signatures {
		signerAddress := gs.Keys[sig.Index]
		newIndex := slices.Index(newGS, signerAddress)
		// Guardian is not part of the new set
		if newIndex == -1 {
			continue
		}
		newVAA.Signatures = append(newVAA.Signatures, &vaa.Signature{
			Index:     uint8(newIndex),
			Signature: sig.Signature,
		})
	}

	// Add our own signature only if the new guardian set would reach quorum
	if vaa.CalculateQuorum(len(newGS)) > len(newVAA.Signatures)+1 {
		return nil, errors.New("cannot reach quorum on new guardian set with the local signature")
	}

	// Add local signature
	newVAA.AddSignature(s.gk, uint8(localGuardianIndex))

	// Sort VAA signatures by guardian ID
	slices.SortFunc(newVAA.Signatures, func(a, b *vaa.Signature) int {
		if a.Index < b.Index {
			return -1
		} else if a.Index > b.Index {
			return 1
		}
		return 0
	})

	newVAABytes, err := newVAA.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal new VAA: %w", err)
	}

	return &nodev1.SignExistingVAAResponse{Vaa: newVAABytes}, nil
}

func (s *nodePrivilegedService) DumpRPCs(ctx context.Context, req *nodev1.DumpRPCsRequest) (*nodev1.DumpRPCsResponse, error) {
	return &nodev1.DumpRPCsResponse{
		Response: s.rpcMap,
	}, nil
}

func (s *nodePrivilegedService) GetAndObserveMissingVAAs(ctx context.Context, req *nodev1.GetAndObserveMissingVAAsRequest) (*nodev1.GetAndObserveMissingVAAsResponse, error) {
	// Get URL and API key from the command line
	url := req.GetUrl()
	apiKey := req.GetApiKey()

	// Create the body of the request
	jsonBody := []byte(`{"apiKey": "` + apiKey + `"}`)
	jsonBodyReader := bytes.NewReader(jsonBody)

	// Create the actual request
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, jsonBodyReader)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not create request: %s\n", err)
		return nil, err
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	// Call the cloud function to get the missing VAAs
	results, err := client.Do(httpRequest)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: error making http request: %s\n", err)
		return nil, err
	}

	// Collect the results
	resBody, err := io.ReadAll(results.Body)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not read response body: %s\n", err)
		return nil, err
	}
	fmt.Printf("client: response body: %s\n", resBody)
	type MissingVAA struct {
		Chain  int    `json:"chain"`
		VaaKey string `json:"vaaKey"`
		Txhash string `json:"txhash"`
	}
	var missingVAAs []MissingVAA
	err = json.Unmarshal(resBody, &missingVAAs)
	if err != nil {
		fmt.Printf("GetAndObserveMissingVAAs: could not unmarshal response body: %s\n", err)
		return nil, err
	}

	MAX_VAAS_TO_PROCESS := 25
	// Only do a max of 25 at a time so as to not overload the node
	numVaas := len(missingVAAs)
	processingLen := numVaas
	if processingLen > MAX_VAAS_TO_PROCESS {
		processingLen = MAX_VAAS_TO_PROCESS
	}

	// Start injecting the VAAs
	obsCounter := 0
	errCounter := 0
	errMsgs := "Messages: "
	for i := 0; i < processingLen; i++ {
		missingVAA := missingVAAs[i]
		// First check to see if this VAA has already been signed
		// Convert vaaKey to VAAID
		splits := strings.Split(missingVAA.VaaKey, "/")
		chainID, err := strconv.Atoi(splits[0])
		if err != nil {
			errMsgs += fmt.Sprintf("\nerror converting chainID [%s] to int", missingVAA.VaaKey)
			errCounter++
			continue
		}
		sequence, err := strconv.ParseUint(splits[2], 10, 64)
		if err != nil {
			errMsgs += fmt.Sprintf("\nerror converting sequence %s to uint64", splits[2])
			errCounter++
			continue
		}
		vaaKey := db.VAAID{EmitterChain: vaa.ChainID(chainID), EmitterAddress: vaa.Address([]byte(splits[1])), Sequence: sequence}
		hasVaa, err := s.db.HasVAA(vaaKey)
		if err != nil || hasVaa {
			errMsgs += fmt.Sprintf("\nerror checking for VAA %s", missingVAA.VaaKey)
			errCounter++
			continue
		}
		var obsvReq gossipv1.ObservationRequest
		obsvReq.ChainId = uint32(missingVAA.Chain)
		obsvReq.TxHash, err = hex.DecodeString(strings.TrimPrefix(missingVAA.Txhash, "0x"))
		if err != nil {
			obsvReq.TxHash, err = base58.Decode(missingVAA.Txhash)
			if err != nil {
				errMsgs += "Invalid transaction hash (neither hex nor base58)"
				errCounter++
				continue
			}
		}
		errMsgs += fmt.Sprintf("\nAttempting to observe %s", missingVAA.Txhash)
		// Call the following function to send the observation request
		if err := common.PostObservationRequest(s.obsvReqSendC, &obsvReq); err != nil {
			errMsgs += fmt.Sprintf("\nPostObservationRequest error %s", err.Error())
			errCounter++
			continue
		}
		obsCounter++
	}
	response := "There were no missing VAAs to recover."
	if processingLen > 0 {
		response = fmt.Sprintf("Successfully injected %d of %d VAAs. %d errors were encountered.", obsCounter, processingLen, errCounter)
		if numVaas > MAX_VAAS_TO_PROCESS {
			response += fmt.Sprintf("\nOnly %d of the %d missing VAAs were processed.  Run the command again to process more.", MAX_VAAS_TO_PROCESS, numVaas)
		}
	}
	response += "\n" + errMsgs
	return &nodev1.GetAndObserveMissingVAAsResponse{
		Response: response,
	}, nil
}
