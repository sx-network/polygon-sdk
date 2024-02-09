package datafeed

import (
	"math/big"

	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
)

var functions = []string{
	"function _votingPeriod() view returns (uint256)",
}

const (
    VotingPeriod string = "_votingPeriod"
)

func (d *DataFeed) sendCall(
	functionType string,
) interface{} {
    var functionName string
    var functionArgs []interface{}

    switch functionType {
		case VotingPeriod:
			functionName = VotingPeriod
		}
		
    abiContract, err := ethgoabi.NewABIFromList(functions)
	if err != nil {
		d.txService.logger.Error(
			"failed to get abi contract via ethgo",
			"function", functionName,
			"functionArgs", functionArgs,
			"functionSig", abiContract,
			"err", err,
		)
		return nil
	}

	c := contract.NewContract(
        ethgo.Address(ethgo.HexToAddress("0x55b3d7c853aD2382f1c62dEc70056BD301CE5098")),
        abiContract,
        contract.WithJsonRPC(d.txService.client.Eth()),
    )

	res, err := c.Call(functionName, ethgo.Latest)
	if err != nil {
		d.txService.logger.Error(
			"failed to call via ethgo",
			"function", functionName,
			"functionArgs", functionArgs,
			"functionSig", abiContract,
			"err", err,
		)
		return nil
	}

	switch functionType {
	case VotingPeriod:
		votingPhase, ok := res["0"].(*big.Int)
		if !ok {
			d.txService.logger.Error(
				"failed to convert result to big int",
				"function", functionName,
				"functionArgs", functionArgs,
				"functionSig", abiContract,
				"err", err,
			)
			return nil
		}
		
		return votingPhase
	}
	
	return nil
}
