package staking

import (
	"errors"
	"math/big"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/abi"
)

var (
	// staking contract address
	AddrStakingContract = types.StringToAddress("1001")

	// bytecode indicating BlockRewards
	BlockRewardsInput = []byte("BlockRewards")

	// Gas limit used when querying the validator set
	queryGasLimit uint64 = 100000
)

func DecodeValidators(method *abi.Method, returnValue []byte) ([]types.Address, error) {
	decodedResults, err := method.Outputs.Decode(returnValue)
	if err != nil {
		return nil, err
	}

	results, ok := decodedResults.(map[string]interface{})
	if !ok {
		return nil, errors.New("failed type assertion from decodedResults to map")
	}

	web3Addresses, ok := results["0"].([]web3.Address)

	if !ok {
		return nil, errors.New("failed type assertion from results[0] to []web3.Address")
	}

	addresses := make([]types.Address, len(web3Addresses))
	for idx, waddr := range web3Addresses {
		addresses[idx] = types.Address(waddr)
	}

	return addresses, nil
}

type TxQueryHandler interface {
	Apply(*types.Transaction) (*runtime.ExecutionResult, error)
	GetNonce(types.Address) uint64
	GetTxContext() runtime.TxContext
}

func QueryValidators(t TxQueryHandler, from types.Address) ([]types.Address, error) {
	method, ok := abis.StakingABI.Methods["validators"]
	if !ok {
		return nil, errors.New("validators method doesn't exist in Staking contract ABI")
	}

	selector := method.ID()
	res, err := t.Apply(&types.Transaction{
		From:     from,
		To:       &AddrStakingContract,
		Value:    big.NewInt(0),
		Input:    selector,
		GasPrice: big.NewInt(0),
		Gas:      queryGasLimit,
		Nonce:    t.GetNonce(from),
	})

	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, res.Err
	}

	return DecodeValidators(method, res.ReturnValue)
}

func BlockRewardsPayment(t TxQueryHandler) error {
	// set the Input to BlockRewardsInput so we can identify blockReward payments within t.apply()
	coinBase := t.GetTxContext().Coinbase
	res, err := t.Apply(&types.Transaction{
		From:     types.ZeroAddress,
		To:       &coinBase,
		Value:    big.NewInt(0),
		Input:    BlockRewardsInput,
		GasPrice: big.NewInt(0),
		Gas:      queryGasLimit,
		Nonce:    t.GetNonce(types.ZeroAddress),
	})

	if err != nil {
		return err
	}

	if res.Failed() {
		return res.Err
	}

	return nil
}
