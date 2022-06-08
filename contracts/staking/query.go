package staking

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"

	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"

	"github.com/umbracle/go-web3"
	"github.com/umbracle/go-web3/abi"
)

var (
	// staking contract address
	AddrStakingContract = types.StringToAddress("1001")

	// Gas limit used when querying the validator set
	queryGasLimit uint64 = 100000

	AddrUSDCContract = ethgo.HexToAddress("0x5147891461a7C81075950f8eE6384e019e39ab98")
	// types.StringToAddress("0x5147891461a7C81075950f8eE6384e019e39ab98")
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

func getContract(contractAddr ethgo.Address, contractABI *ethgoabi.ABI, client *jsonrpc.Client) *contract.Contract {
	return contract.NewContract(contractAddr, contractABI, contract.WithJsonRPC(client.Eth()))
}

func GetDecimals() (map[string]interface{}, error) {
	client, _ := jsonrpc.NewClient("https://rpc.toronto.sx.technology")
	usdcContract := getContract(AddrUSDCContract, abis.USDCContractABI, client)
	res, _ := usdcContract.Call("symbol", ethgo.Latest)
	for k, v := range res {
		fmt.Printf("%s value is %v", k, v)
	}
	return res, nil
}
