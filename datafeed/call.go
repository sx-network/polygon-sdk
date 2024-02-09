package datafeed

import (
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
)

const (
    votingPeriod string = "function _votingPeriod() view returns (uint256)"
)

const (
    VotingPeriod string = "_votingPeriod"
)

func (d *DataFeed) sendCall(
	functionType string,
) interface{} {
	var functionSig string
    var functionName string
    var functionArgs []interface{}

    switch functionType {
		case VotingPeriod:
			functionSig = votingPeriod
			functionName = VotingPeriod
		}
		
    abiContract, err := ethgoabi.NewABIFromList([]string{functionSig})
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
        ethgo.Address(ethgo.HexToAddress(d.config.OutcomeReporterAddress)),
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

	return res["0"]
}
