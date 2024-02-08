package datafeed

import (
	"fmt"

	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
)

func (d *DataFeed) sendCall() {
	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------------------------------------------")
	var functions = []string{
		"function _votingPeriod() view returns (uint256)",
	}
	
	abiContract, err := ethgoabi.NewABIFromList(functions)
	handleErr(err, " - 1 - ")
	
	client, err := jsonrpc.NewClient("https://rpc.toronto.sx.technology")
	handleErr(err, " - 2 - ") 


	c := contract.NewContract(
		ethgo.Address(ethgo.HexToAddress("0x55b3d7c853aD2382f1c62dEc70056BD301CE5098")),
		abiContract,
		contract.WithJsonRPC(client.Eth()),
	)

	res, err := c.Call("_votingPeriod", ethgo.Latest)
	handleErr(err, " - 3 - ")

	value := res["0"]
	fmt.Println("Value:", value)
}

func handleErr(err error, msg string) {
	if err != nil {
		fmt.Println(msg)
		panic(err)
	}
}