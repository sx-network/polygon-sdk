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
	handleErr(err, " - 1 - ")
	
    c := contract.NewContract(
        ethgo.Address(ethgo.HexToAddress(d.config.SXNodeAddress)),
        abiContract,
        contract.WithJsonRPC(d.txService.client.Eth()),
    )

	res, err := c.Call("_votingPeriod", ethgo.Latest)
	handleErr(err, " - 3 - ")

	value := res["0"]
	fmt.Println("Value:", value, functionName, functionArgs)

	return nil
}

func handleErr(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		panic(err)
	}
}