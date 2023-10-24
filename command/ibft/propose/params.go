package propose

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/helper"
	ibftOp "github.com/0xPolygon/polygon-edge/consensus/ibft/proto"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

const (
	voteFlag          = "vote"
	addressFlag       = "addr"
	blsFlag           = "bls"
	votingStationFlag = "voting-station"
)

const (
	authVote = "auth"
	dropVote = "drop"
)

var (
	errInvalidVoteType      = errors.New("invalid vote type")
	errInvalidAddressFormat = errors.New("invalid address format")
)

var (
	params = &proposeParams{}
)

const (
	JSONRPCHost          = "http://localhost:10002"
	voteAddSCFunction    = "function voteAdd(address newValidator)"
	voteRemoveSCFunction = "function voteDrop(address oldValidator)"
)

type proposeParams struct {
	addressRaw       string
	rawBLSPublicKey  string
	rawVotingStation string

	vote          string
	address       types.Address
	blsPublicKey  []byte
	votingStation types.Address
}

func (p *proposeParams) getRequiredFlags() []string {
	return []string{
		voteFlag,
		addressFlag,
	}
}

func (p *proposeParams) validateFlags() error {
	if !isValidVoteType(p.vote) {
		return errInvalidVoteType
	}

	return nil
}

func (p *proposeParams) initRawParams() error {
	if err := p.initAddress(); err != nil {
		return err
	}

	if err := p.initBLSPublicKey(); err != nil {
		return err
	}

	if err := p.initVotingStation(); err != nil {
		return err
	}

	return nil
}

func (p *proposeParams) initAddress() error {
	p.address = types.Address{}
	if err := p.address.UnmarshalText([]byte(p.addressRaw)); err != nil {
		return errInvalidAddressFormat
	}

	return nil
}

func (p *proposeParams) initVotingStation() error {
	p.votingStation = types.Address{}
	if err := p.votingStation.UnmarshalText([]byte(p.rawVotingStation)); err != nil {
		return errInvalidAddressFormat
	}

	return nil
}

func (p *proposeParams) initBLSPublicKey() error {
	if p.rawBLSPublicKey == "" {
		return nil
	}

	blsPubkeyBytes, err := hex.DecodeString(strings.TrimPrefix(p.rawBLSPublicKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse BLS Public Key: %w", err)
	}

	if _, err := crypto.UnmarshalBLSPublicKey(blsPubkeyBytes); err != nil {
		return err
	}

	p.blsPublicKey = blsPubkeyBytes

	return nil
}

func isValidVoteType(vote string) bool {
	return vote == authVote || vote == dropVote
}

func (p *proposeParams) proposeCandidate(grpcAddress string) error {
	ibftClient, err := helper.GetIBFTOperatorClientConnection(grpcAddress)
	if err != nil {
		return err
	}

	if _, err := ibftClient.Propose(
		context.Background(),
		p.getCandidate(),
	); err != nil {
		return err
	}

	return nil
}

func (p *proposeParams) ibftSetValidators(grpcAddress string) {
	var functionSig string
	var functionName string
	var functionArgs []interface{}

	ibftClient, err := helper.GetIBFTOperatorClientConnection(grpcAddress)
	if err != nil {
		fmt.Printf("Failed to get ibft client conn")
	}

	switch p.vote {
	case "auth":
		functionSig = voteAddSCFunction
		functionName = "voteAdd"
		functionArgs = append(make([]interface{}, 0), types.StringToAddress(p.vote))
	case "drop":
		functionSig = voteRemoveSCFunction
		functionName = "voteDrop"
		functionArgs = append(make([]interface{}, 0), types.StringToAddress(p.vote))
	default:
		fmt.Printf("invalid ibft vote command %s", p.vote)
	}

	abiContract, err := ethgoabi.NewABIFromList([]string{functionSig})

	if err != nil {
		fmt.Println(fmt.Errorf("failed to retrieve ethgo ABI %s function with error %w ", functionName, err))
	}

	client, err := jsonrpc.NewClient(JSONRPCHost)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to initialize new ethgo client"))
	}
	encodedPrivateKey, encodeError := ibftClient.GetValidatorPrivateKey(context.Background(), &empty.Empty{})

	if encodeError != nil {
		fmt.Println(fmt.Errorf("failed to get encoded private key %w ", encodeError))
	}

	decodedValidatorKey, decodedError := crypto.ParseECDSAPrivateKey(encodedPrivateKey.PrivateKey)

	if decodedError != nil {
		fmt.Println(fmt.Errorf("failed to get decoded private key %w", encodeError))
	}

	c := contract.NewContract(
		ethgo.Address(p.votingStation),
		abiContract,
		contract.WithSender(wallet.NewKey(decodedValidatorKey)),
		contract.WithJsonRPC(client.Eth()),
	)

	_, err = c.Txn(
		functionName,
		functionArgs...,
	)

	if err != nil {
		fmt.Printf("failed to call %s on voting-station", p.vote)

	}
}

func (p *proposeParams) getCandidate() *ibftOp.Candidate {
	res := &ibftOp.Candidate{
		Address: p.address.String(),
		Auth:    p.vote == authVote,
	}

	if p.blsPublicKey != nil {
		res.BlsPubkey = p.blsPublicKey
	}

	return res
}

func (p *proposeParams) getResult() command.CommandResult {
	return &IBFTProposeResult{
		Address: p.address.String(),
		Vote:    p.vote,
	}
}
