package datafeed

import (
	"fmt"

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
			"failed to call via ethgo",
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
	handleErr(err, " - 3 - ")

	value := res["0"]
	fmt.Println("-------------- ---------------------- --------------------- Value:", functionName, value, functionName, functionArgs)

	return nil
}

func handleErr(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		panic(err)
	}
}