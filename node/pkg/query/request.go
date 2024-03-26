package query

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"

	solana "github.com/gagliardetto/solana-go"
)

// MSG_VERSION is the current version of the CCQ message protocol.
const MSG_VERSION uint8 = 1

// QueryRequest defines a cross chain query request to be submitted to the guardians.
// It is the payload of the SignedQueryRequest gossip message.
type QueryRequest struct {
	Nonce           uint32
	PerChainQueries []*PerChainQueryRequest
}

// PerChainQueryRequest represents a query request for a single chain.
type PerChainQueryRequest struct {
	// ChainId indicates which chain this query is destine for.
	ChainId vaa.ChainID

	// Query is the chain specific query data.
	Query ChainSpecificQuery
}

// ChainSpecificQuery is the interface that must be implemented by a chain specific query.
type ChainSpecificQuery interface {
	Type() ChainSpecificQueryType
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
	UnmarshalFromReader(reader *bytes.Reader) error
	Validate() error
}

// ChainSpecificQueryType is used to interpret the data in a per chain query request.
type ChainSpecificQueryType uint8

////////////////////////////////// Solana Queries ////////////////////////////////////////////////

// SolanaAccountQueryRequestType is the type of a Solana sol_account query request.
const SolanaAccountQueryRequestType ChainSpecificQueryType = 4

// SolanaAccountQueryRequest implements ChainSpecificQuery for a Solana sol_account query request.
type SolanaAccountQueryRequest struct {
	// Commitment identifies the commitment level to be used in the queried. Currently it may only "finalized".
	// Before we can support "confirmed", we need a way to read the account data and the block information atomically.
	// We would also need to deal with the fact that queries are only handled in the finalized watcher and it does not
	// have access to the latest confirmed slot needed for MinContextSlot retries.
	Commitment string

	// The minimum slot that the request can be evaluated at. Zero means unused.
	MinContextSlot uint64

	// The offset of the start of data to be returned. Unused if DataSliceLength is zero.
	DataSliceOffset uint64

	// The length of the data to be returned. Zero means all data is returned.
	DataSliceLength uint64

	// Accounts is an array of accounts to be queried.
	Accounts [][SolanaPublicKeyLength]byte
}

// Solana public keys are fixed length.
const SolanaPublicKeyLength = solana.PublicKeyLength

// According to the Solana spec, the longest comment string is nine characters. Allow a few more, just in case.
// https://pkg.go.dev/github.com/gagliardetto/solana-go/rpc#CommitmentType
const SolanaMaxCommitmentLength = 12

// According to the spec, the query only supports up to 100 accounts.
// https://github.com/solana-labs/solana/blob/9d132441fdc6282a8be4bff0bc77d6a2fefe8b59/rpc-client-api/src/request.rs#L204
const SolanaMaxAccountsPerQuery = 100

func (saq *SolanaAccountQueryRequest) AccountList() [][SolanaPublicKeyLength]byte {
	return saq.Accounts
}

// SolanaPdaQueryRequestType is the type of a Solana sol_pda query request.
const SolanaPdaQueryRequestType ChainSpecificQueryType = 5

// SolanaPdaQueryRequest implements ChainSpecificQuery for a Solana sol_pda query request.
type SolanaPdaQueryRequest struct {
	// Commitment identifies the commitment level to be used in the queried. Currently it may only "finalized".
	// Before we can support "confirmed", we need a way to read the account data and the block information atomically.
	// We would also need to deal with the fact that queries are only handled in the finalized watcher and it does not
	// have access to the latest confirmed slot needed for MinContextSlot retries.
	Commitment string

	// The minimum slot that the request can be evaluated at. Zero means unused.
	MinContextSlot uint64

	// The offset of the start of data to be returned. Unused if DataSliceLength is zero.
	DataSliceOffset uint64

	// The length of the data to be returned. Zero means all data is returned.
	DataSliceLength uint64

	// PDAs is an array of PDAs to be queried.
	PDAs []SolanaPDAEntry
}

// SolanaPDAEntry defines a single Solana Program derived address (PDA).
type SolanaPDAEntry struct {
	ProgramAddress [SolanaPublicKeyLength]byte
	Seeds          [][]byte
}

// According to the spec, there may be at most 16 seeds.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L559
const SolanaMaxSeeds = solana.MaxSeeds

// According to the spec, a seed may be at most 32 bytes.
// https://github.com/gagliardetto/solana-go/blob/6fe3aea02e3660d620433444df033fc3fe6e64c1/keys.go#L557
const SolanaMaxSeedLen = solana.MaxSeedLength

func (spda *SolanaPdaQueryRequest) PDAList() []SolanaPDAEntry {
	return spda.PDAs
}

// PerChainQueryInternal is an internal representation of a query request that is passed to the watcher.
type PerChainQueryInternal struct {
	RequestID  string
	RequestIdx int
	Request    *PerChainQueryRequest
}

func (pcqi *PerChainQueryInternal) ID() string {
	return fmt.Sprintf("%s:%d", pcqi.RequestID, pcqi.RequestIdx)
}

// QueryRequestDigest returns the query signing prefix based on the environment.
func QueryRequestDigest(env common.Environment, b []byte) ethCommon.Hash {
	var queryRequestPrefix []byte
	if env == common.MainNet {
		queryRequestPrefix = []byte("mainnet_query_request_000000000000|")
	} else if env == common.TestNet {
		queryRequestPrefix = []byte("testnet_query_request_000000000000|")
	} else {
		queryRequestPrefix = []byte("devnet_query_request_0000000000000|")
	}

	return ethCrypto.Keccak256Hash(append(queryRequestPrefix, b...))
}

// PostSignedQueryRequest posts a signed query request to the specified channel.
func PostSignedQueryRequest(signedQueryReqSendC chan<- *gossipv1.SignedQueryRequest, req *gossipv1.SignedQueryRequest) error {
	select {
	case signedQueryReqSendC <- req:
		return nil
	default:
		return common.ErrChanFull
	}
}

func SignedQueryRequestEqual(left *gossipv1.SignedQueryRequest, right *gossipv1.SignedQueryRequest) bool {
	if !bytes.Equal(left.QueryRequest, right.QueryRequest) {
		return false
	}
	if !bytes.Equal(left.Signature, right.Signature) {
		return false
	}
	return true
}

//
// Implementation of QueryRequest.
//

// Marshal serializes the binary representation of a query request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (queryRequest *QueryRequest) Marshal() ([]byte, error) {
	if err := queryRequest.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, MSG_VERSION)        // version
	vaa.MustWrite(buf, binary.BigEndian, queryRequest.Nonce) // uint32

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(queryRequest.PerChainQueries)))
	for _, perChainQuery := range queryRequest.PerChainQueries {
		pcqBuf, err := perChainQuery.Marshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal per chain query: %w", err)
		}
		buf.Write(pcqBuf)
	}

	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a query request from a byte array
func (queryRequest *QueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return queryRequest.UnmarshalFromReader(reader)
}

// UnmarshalFromReader deserializes the binary representation of a query request from an existing reader
func (queryRequest *QueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	var version uint8
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return fmt.Errorf("failed to read message version: %w", err)
	}

	if version != MSG_VERSION {
		return fmt.Errorf("unsupported message version: %d", version)
	}

	if err := binary.Read(reader, binary.BigEndian, &queryRequest.Nonce); err != nil {
		return fmt.Errorf("failed to read request nonce: %w", err)
	}

	numPerChainQueries := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPerChainQueries); err != nil {
		return fmt.Errorf("failed to read number of per chain queries: %w", err)
	}

	for count := 0; count < int(numPerChainQueries); count++ {
		perChainQuery := PerChainQueryRequest{}
		err := perChainQuery.UnmarshalFromReader(reader)
		if err != nil {
			return fmt.Errorf("failed to Unmarshal per chain query: %w", err)
		}
		queryRequest.PerChainQueries = append(queryRequest.PerChainQueries, &perChainQuery)
	}

	if reader.Len() != 0 {
		return fmt.Errorf("excess bytes in unmarshal")
	}

	if err := queryRequest.Validate(); err != nil {
		return fmt.Errorf("unmarshaled request failed validation: %w", err)
	}

	return nil
}

// Validate does basic validation on a received query request.
func (queryRequest *QueryRequest) Validate() error {
	// Nothing to validate on the Nonce.
	if len(queryRequest.PerChainQueries) <= 0 {
		return fmt.Errorf("request does not contain any per chain queries")
	}
	if len(queryRequest.PerChainQueries) > math.MaxUint8 {
		return fmt.Errorf("too many per chain queries")
	}
	for idx, perChainQuery := range queryRequest.PerChainQueries {
		if err := perChainQuery.Validate(); err != nil {
			return fmt.Errorf("failed to validate per chain query %d: %w", idx, err)
		}
	}
	return nil
}

// Equal verifies that two query requests are equal.
func (left *QueryRequest) Equal(right *QueryRequest) bool {
	if left.Nonce != right.Nonce {
		return false
	}
	if len(left.PerChainQueries) != len(right.PerChainQueries) {
		return false
	}

	for idx := range left.PerChainQueries {
		if !left.PerChainQueries[idx].Equal(right.PerChainQueries[idx]) {
			return false
		}
	}
	return true
}

//
// Implementation of PerChainQueryRequest.
//

// Marshal serializes the binary representation of a per chain query request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (perChainQuery *PerChainQueryRequest) Marshal() ([]byte, error) {
	if err := perChainQuery.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	vaa.MustWrite(buf, binary.BigEndian, perChainQuery.ChainId)
	vaa.MustWrite(buf, binary.BigEndian, perChainQuery.Query.Type())
	queryBuf, err := perChainQuery.Query.Marshal()
	if err != nil {
		return nil, err
	}

	// Write the length of the query to facilitate on-chain parsing.
	if len(queryBuf) > math.MaxUint32 {
		return nil, fmt.Errorf("query too long")
	}
	vaa.MustWrite(buf, binary.BigEndian, uint32(len(queryBuf)))

	buf.Write(queryBuf)
	return buf.Bytes(), nil
}

// Unmarshal deserializes the binary representation of a per chain query request from a byte array
func (perChainQuery *PerChainQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return perChainQuery.UnmarshalFromReader(reader)
}

// UnmarshalFromReader deserializes the binary representation of a per chain query request from an existing reader
func (perChainQuery *PerChainQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	if err := binary.Read(reader, binary.BigEndian, &perChainQuery.ChainId); err != nil {
		return fmt.Errorf("failed to read request chain: %w", err)
	}

	qt := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &qt); err != nil {
		return fmt.Errorf("failed to read request type: %w", err)
	}
	queryType := ChainSpecificQueryType(qt)

	if err := ValidatePerChainQueryRequestType(queryType); err != nil {
		return err
	}

	// Skip the query length.
	var queryLength uint32
	if err := binary.Read(reader, binary.BigEndian, &queryLength); err != nil {
		return fmt.Errorf("failed to read query length: %w", err)
	}

	switch queryType {
	case SolanaAccountQueryRequestType:
		q := SolanaAccountQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal solana account query request: %w", err)
		}
		perChainQuery.Query = &q
	case SolanaPdaQueryRequestType:
		q := SolanaPdaQueryRequest{}
		if err := q.UnmarshalFromReader(reader); err != nil {
			return fmt.Errorf("failed to unmarshal solana PDA query request: %w", err)
		}
		perChainQuery.Query = &q
	default:
		return fmt.Errorf("unsupported query type: %d", queryType)
	}

	return nil
}

// Validate does basic validation on a per chain query request.
func (perChainQuery *PerChainQueryRequest) Validate() error {
	str := perChainQuery.ChainId.String()
	if _, err := vaa.ChainIDFromString(str); err != nil {
		return fmt.Errorf("invalid chainID: %d", uint16(perChainQuery.ChainId))
	}

	if perChainQuery.Query == nil {
		return fmt.Errorf("query is nil")
	}

	if err := ValidatePerChainQueryRequestType(perChainQuery.Query.Type()); err != nil {
		return err
	}

	if err := perChainQuery.Query.Validate(); err != nil {
		return fmt.Errorf("chain specific query is invalid: %w", err)
	}

	return nil
}

func ValidatePerChainQueryRequestType(qt ChainSpecificQueryType) error {
	if qt != SolanaAccountQueryRequestType && qt != SolanaPdaQueryRequestType {
		return fmt.Errorf("invalid query request type: %d", qt)
	}
	return nil
}

// Equal verifies that two query requests are equal.
func (left *PerChainQueryRequest) Equal(right *PerChainQueryRequest) bool {
	if left.ChainId != right.ChainId {
		return false
	}

	if left.Query == nil && right.Query == nil {
		return true
	}

	if left.Query == nil || right.Query == nil {
		return false
	}

	if left.Query.Type() != right.Query.Type() {
		return false
	}

	switch leftQuery := left.Query.(type) {
	case *SolanaAccountQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *SolanaAccountQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be sol_account")
		}
	case *SolanaPdaQueryRequest:
		switch rightQuery := right.Query.(type) {
		case *SolanaPdaQueryRequest:
			return leftQuery.Equal(rightQuery)
		default:
			panic("unsupported query type on right, must be sol_pda")
		}
	default:
		panic("unsupported query type on left")
	}
}

//
// Implementation of SolanaAccountQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *SolanaAccountQueryRequest) Type() ChainSpecificQueryType {
	return SolanaAccountQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_account request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (saq *SolanaAccountQueryRequest) Marshal() ([]byte, error) {
	if err := saq.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(saq.Commitment)))
	buf.Write([]byte(saq.Commitment))

	vaa.MustWrite(buf, binary.BigEndian, saq.MinContextSlot)
	vaa.MustWrite(buf, binary.BigEndian, saq.DataSliceOffset)
	vaa.MustWrite(buf, binary.BigEndian, saq.DataSliceLength)

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(saq.Accounts)))
	for _, acct := range saq.Accounts {
		buf.Write(acct[:])
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_account query from a byte array
func (saq *SolanaAccountQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return saq.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_account query from a byte array
func (saq *SolanaAccountQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	len := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &len); err != nil {
		return fmt.Errorf("failed to read commitment len: %w", err)
	}

	if len > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment string is too long, may not be more than %d characters", SolanaMaxCommitmentLength)
	}

	commitment := make([]byte, len)
	if n, err := reader.Read(commitment[:]); err != nil || n != int(len) {
		return fmt.Errorf("failed to read commitment [%d]: %w", n, err)
	}
	saq.Commitment = string(commitment)

	if err := binary.Read(reader, binary.BigEndian, &saq.MinContextSlot); err != nil {
		return fmt.Errorf("failed to read min slot: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &saq.DataSliceOffset); err != nil {
		return fmt.Errorf("failed to read data slice offset: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &saq.DataSliceLength); err != nil {
		return fmt.Errorf("failed to read data slice length: %w", err)
	}

	numAccounts := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numAccounts); err != nil {
		return fmt.Errorf("failed to read number of account entries: %w", err)
	}

	for count := 0; count < int(numAccounts); count++ {
		account := [SolanaPublicKeyLength]byte{}
		if n, err := reader.Read(account[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read account [%d]: %w", n, err)
		}
		saq.Accounts = append(saq.Accounts, account)
	}

	return nil
}

// Validate does basic validation on a Solana sol_account query.
func (saq *SolanaAccountQueryRequest) Validate() error {
	if len(saq.Commitment) > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment too long")
	}
	if saq.Commitment != "finalized" {
		return fmt.Errorf(`commitment must be "finalized"`)
	}

	if saq.DataSliceLength == 0 && saq.DataSliceOffset != 0 {
		return fmt.Errorf("data slice offset may not be set if data slice length is zero")
	}

	if len(saq.Accounts) <= 0 {
		return fmt.Errorf("does not contain any account entries")
	}
	if len(saq.Accounts) > SolanaMaxAccountsPerQuery {
		return fmt.Errorf("too many account entries, may not be more than %d", SolanaMaxAccountsPerQuery)
	}
	for _, acct := range saq.Accounts {
		// The account is fixed length, so don't need to check for nil.
		if len(acct) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid account length")
		}
	}

	return nil
}

// Equal verifies that two Solana sol_account queries are equal.
func (left *SolanaAccountQueryRequest) Equal(right *SolanaAccountQueryRequest) bool {
	if left.Commitment != right.Commitment ||
		left.MinContextSlot != right.MinContextSlot ||
		left.DataSliceOffset != right.DataSliceOffset ||
		left.DataSliceLength != right.DataSliceLength {
		return false
	}

	if len(left.Accounts) != len(right.Accounts) {
		return false
	}
	for idx := range left.Accounts {
		if !bytes.Equal(left.Accounts[idx][:], right.Accounts[idx][:]) {
			return false
		}
	}

	return true
}

//
// Implementation of SolanaPdaQueryRequest, which implements the ChainSpecificQuery interface.
//

func (e *SolanaPdaQueryRequest) Type() ChainSpecificQueryType {
	return SolanaPdaQueryRequestType
}

// Marshal serializes the binary representation of a Solana sol_pda request.
// This method calls Validate() and relies on it to range checks lengths, etc.
func (spda *SolanaPdaQueryRequest) Marshal() ([]byte, error) {
	if err := spda.Validate(); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	vaa.MustWrite(buf, binary.BigEndian, uint32(len(spda.Commitment)))
	buf.Write([]byte(spda.Commitment))

	vaa.MustWrite(buf, binary.BigEndian, spda.MinContextSlot)
	vaa.MustWrite(buf, binary.BigEndian, spda.DataSliceOffset)
	vaa.MustWrite(buf, binary.BigEndian, spda.DataSliceLength)

	vaa.MustWrite(buf, binary.BigEndian, uint8(len(spda.PDAs)))
	for _, pda := range spda.PDAs {
		buf.Write(pda.ProgramAddress[:])
		vaa.MustWrite(buf, binary.BigEndian, uint8(len(pda.Seeds)))
		for _, seed := range pda.Seeds {
			vaa.MustWrite(buf, binary.BigEndian, uint32(len(seed)))
			buf.Write(seed)
		}
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes a Solana sol_pda query from a byte array
func (spda *SolanaPdaQueryRequest) Unmarshal(data []byte) error {
	reader := bytes.NewReader(data[:])
	return spda.UnmarshalFromReader(reader)
}

// UnmarshalFromReader  deserializes a Solana sol_pda query from a byte array
func (spda *SolanaPdaQueryRequest) UnmarshalFromReader(reader *bytes.Reader) error {
	len := uint32(0)
	if err := binary.Read(reader, binary.BigEndian, &len); err != nil {
		return fmt.Errorf("failed to read commitment len: %w", err)
	}

	if len > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment string is too long, may not be more than %d characters", SolanaMaxCommitmentLength)
	}

	commitment := make([]byte, len)
	if n, err := reader.Read(commitment[:]); err != nil || n != int(len) {
		return fmt.Errorf("failed to read commitment [%d]: %w", n, err)
	}
	spda.Commitment = string(commitment)

	if err := binary.Read(reader, binary.BigEndian, &spda.MinContextSlot); err != nil {
		return fmt.Errorf("failed to read min slot: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &spda.DataSliceOffset); err != nil {
		return fmt.Errorf("failed to read data slice offset: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &spda.DataSliceLength); err != nil {
		return fmt.Errorf("failed to read data slice length: %w", err)
	}

	numPDAs := uint8(0)
	if err := binary.Read(reader, binary.BigEndian, &numPDAs); err != nil {
		return fmt.Errorf("failed to read number of PDAs: %w", err)
	}

	for count := 0; count < int(numPDAs); count++ {
		programAddress := [SolanaPublicKeyLength]byte{}
		if n, err := reader.Read(programAddress[:]); err != nil || n != SolanaPublicKeyLength {
			return fmt.Errorf("failed to read program address [%d]: %w", n, err)
		}

		pda := SolanaPDAEntry{ProgramAddress: programAddress}
		numSeeds := uint8(0)
		if err := binary.Read(reader, binary.BigEndian, &numSeeds); err != nil {
			return fmt.Errorf("failed to read number of seeds: %w", err)
		}

		for count := 0; count < int(numSeeds); count++ {
			seedLen := uint32(0)
			if err := binary.Read(reader, binary.BigEndian, &seedLen); err != nil {
				return fmt.Errorf("failed to read call Data len: %w", err)
			}
			seed := make([]byte, seedLen)
			if n, err := reader.Read(seed[:]); err != nil || n != int(seedLen) {
				return fmt.Errorf("failed to read seed [%d]: %w", n, err)
			}

			pda.Seeds = append(pda.Seeds, seed)
		}

		spda.PDAs = append(spda.PDAs, pda)
	}

	return nil
}

// Validate does basic validation on a Solana sol_pda query.
func (spda *SolanaPdaQueryRequest) Validate() error {
	if len(spda.Commitment) > SolanaMaxCommitmentLength {
		return fmt.Errorf("commitment too long")
	}
	if spda.Commitment != "finalized" {
		return fmt.Errorf(`commitment must be "finalized"`)
	}

	if spda.DataSliceLength == 0 && spda.DataSliceOffset != 0 {
		return fmt.Errorf("data slice offset may not be set if data slice length is zero")
	}

	if len(spda.PDAs) <= 0 {
		return fmt.Errorf("does not contain any PDAs entries")
	}
	if len(spda.PDAs) > SolanaMaxAccountsPerQuery {
		return fmt.Errorf("too many PDA entries, may not be more than %d", SolanaMaxAccountsPerQuery)
	}
	for _, pda := range spda.PDAs {
		// The program address is fixed length, so don't need to check for nil.
		if len(pda.ProgramAddress) != SolanaPublicKeyLength {
			return fmt.Errorf("invalid program address length")
		}

		if len(pda.Seeds) == 0 {
			return fmt.Errorf("PDA does not contain any seeds")
		}

		if len(pda.Seeds) > SolanaMaxSeeds {
			return fmt.Errorf("PDA contains too many seeds")
		}

		for _, seed := range pda.Seeds {
			if len(seed) == 0 {
				return fmt.Errorf("seed is null")
			}

			if len(seed) > SolanaMaxSeedLen {
				return fmt.Errorf("seed is too long")
			}
		}
	}

	return nil
}

// Equal verifies that two Solana sol_pda queries are equal.
func (left *SolanaPdaQueryRequest) Equal(right *SolanaPdaQueryRequest) bool {
	if left.Commitment != right.Commitment ||
		left.MinContextSlot != right.MinContextSlot ||
		left.DataSliceOffset != right.DataSliceOffset ||
		left.DataSliceLength != right.DataSliceLength {
		return false
	}

	if len(left.PDAs) != len(right.PDAs) {
		return false
	}
	for idx := range left.PDAs {
		if !bytes.Equal(left.PDAs[idx].ProgramAddress[:], right.PDAs[idx].ProgramAddress[:]) {
			return false
		}

		if len(left.PDAs[idx].Seeds) != len(right.PDAs[idx].Seeds) {
			return false
		}

		for idx2 := range left.PDAs[idx].Seeds {
			if !bytes.Equal(left.PDAs[idx].Seeds[idx2][:], right.PDAs[idx].Seeds[idx2][:]) {
				return false
			}
		}
	}

	return true
}
