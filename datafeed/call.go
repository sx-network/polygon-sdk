package datafeed

import (
	"github.com/hashicorp/go-hclog"
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
) interface{}{

	var logger hclog.Logger
	logger.Debug("------------------------------------------------------------------------------------------------------------------------------ 11")
	d.logger.Debug("------------ 1");
	var functionSig string
	var functionName string
	var functionArgs []interface{}
	d.logger.Debug("------------ 2");
	switch functionType {
	case VotingPeriod:
		functionSig = votingPeriod
		functionName = VotingPeriod
	}
	d.logger.Debug("------------ 3", functionSig, functionName);
	abiContract, err := ethgoabi.NewABIFromList([]string{functionSig})
	if err != nil {
		d.txService.logger.Error(
			"failed to retrieve ethgo ABI",
			"function", functionName,
			"err", err,
		)
		return nil
	}
	d.logger.Debug("------------ 4", abiContract);
	c := contract.NewContract(
		ethgo.Address(ethgo.HexToAddress(d.config.SXNodeAddress)),
		abiContract,
		contract.WithJsonRPC(d.txService.client.Eth()),
	)
	d.logger.Debug("------------ 5", c);
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

	d.logger.Debug("------------ CALL ", res["0"]);

	return res["0"]
}
