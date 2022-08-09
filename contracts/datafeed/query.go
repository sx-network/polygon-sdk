package datafeed

import (
	"errors"
	"math/big"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/types"
)

var (
	// Gas limit used
	queryGasLimit uint64 = 100000
)

type TxQueryHandler interface {
	Apply(*types.Transaction) (*runtime.ExecutionResult, error)
	GetNonce(types.Address) uint64
}

func SetValidators(t TxQueryHandler, from types.Address, to types.Address, validators []types.Address) ([]byte, error) {
	method, ok := abis.OutcomeReporterABI.Methods["setValidators"]
	if !ok {
		return nil, errors.New("setValidators method doesn't exist in OutcomeReporter contract ABI")
	}

	// TODO: figure out how to pass arguments using eth-go
	encoded, err := method.Encode(12)
	if err != nil {
		return nil, err
	}

	res, err := t.Apply(&types.Transaction{
		From:     from,
		To:       &to,
		Value:    big.NewInt(0),
		Input:    encoded,
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

	return res.ReturnValue, nil
}

func ReportOutcome(t TxQueryHandler, from types.Address, to types.Address, payload *types.ReportOutcome) ([]byte, error) {
	method, ok := abis.OutcomeReporterABI.Methods["reportOutcome"]
	if !ok {
		return nil, errors.New("reportOutcome method doesn't exist in OutcomeReporter contract ABI")
	}

	// TODO: figure out how to pass arguments using eth-go
	encoded, err := method.Encode(payload)
	if err != nil {
		return nil, err
	}

	res, err := t.Apply(&types.Transaction{
		From:     from,
		To:       &to,
		Value:    big.NewInt(0),
		Input:    encoded,
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

	return res.ReturnValue, nil
}
