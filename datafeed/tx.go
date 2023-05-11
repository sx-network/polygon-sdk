package datafeed

import (
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
)

const (
	JSONRPCHost              = "http://localhost:10002"
	proposeOutcomeSCFunction = "function proposeOutcome(bytes32 marketHash, uint8 outcome)"
	voteOutcomeSCFunction    = "function voteOutcome(bytes32 marketHash, uint8 outcome)"
	reportOutcomeSCFunction  = "function reportOutcome(bytes32 marketHash)"
)

const (
	ProposeOutcome string = "proposeOutcome"
	VoteOutcome           = "voteOutcome"
	ReportOutcome         = "reportOutcome"
)

// TxService
type TxService struct {
	logger hclog.Logger
	client *jsonrpc.Client
}

func newTxService(logger hclog.Logger) (*TxService, error) {

	client, err := jsonrpc.NewClient(JSONRPCHost)
	if err != nil {
		logger.Error("failed to initialize new ethgo client")

		return nil, err
	}

	txService := &TxService{
		logger: logger.Named("tx"),
		client: client,
	}

	return txService, nil
}

// sendTxWithRetry send tx with retry to SC specified by customContractAddress
func (d *DataFeed) sendTxWithRetry(
	functionType string,
	report *proto.DataFeedReport,
) {
	const (
		maxTxTries    = 4
		txGasPriceWei = 1000000000
		txGasLimitWei = 1000000
	)

	var functionSig string

	var functionName string

	var functionArgs []interface{}

	switch functionType {
	case ProposeOutcome:
		functionSig = proposeOutcomeSCFunction
		functionName = ProposeOutcome

		functionArgs = append(make([]interface{}, 0), types.StringToHash(report.MarketHash), report.Outcome)
	case VoteOutcome:
		functionSig = voteOutcomeSCFunction
		functionName = VoteOutcome

		functionArgs = append(make([]interface{}, 0), types.StringToHash(report.MarketHash), report.Outcome)
	case ReportOutcome:
		functionSig = reportOutcomeSCFunction
		functionName = ReportOutcome

		functionArgs = append(make([]interface{}, 0), types.StringToHash(report.MarketHash))
	}

	abiContract, err := ethgoabi.NewABIFromList([]string{functionSig})
	if err != nil {
		d.txService.logger.Error(
			"failed to retrieve ethgo ABI",
			"function", functionName,
			"err", err,
		)

		return
	}

	c := contract.NewContract(
		ethgo.Address(d.consensusInfo().CustomContractAddress),
		abiContract,
		contract.WithSender(wallet.NewKey(d.consensusInfo().ValidatorKey)),
		contract.WithJsonRPC(d.txService.client.Eth()),
	)

	txn, err := c.Txn(
		functionName,
		functionArgs...,
	)

	if err != nil {
		d.txService.logger.Error(
			"failed to create txn via ethgo",
			"function", functionName,
			"functionArgs", functionArgs,
			"functionSig", abiContract,
			"err", err,
		)

		return
	}

	txTry := uint64(0)
	currNonce := d.consensusInfo().Nonce

	for txTry < maxTxTries {
		d.txService.logger.Debug(
			"attempting tx with nonce",
			"function", functionName,
			"nonce", currNonce,
			"try #", txTry,
			"marketHash", report.MarketHash)

		//TODO: derive these gas params better, have it dynamic?
		txn.WithOpts(
			&contract.TxnOpts{
				GasPrice: txGasPriceWei + (txTry * txGasPriceWei),
				GasLimit: txGasLimitWei,
				Nonce:    currNonce,
			},
		)

		// TODO: consider adding directly to txpool txpool.AddTx() instead of over local jsonrpc
		// can use TxnPoolOperatorClient.AddTx()
		err = txn.Do()
		if err != nil {
			if strings.Contains(err.Error(), "nonce too low") {
				// if nonce too low, retry with higher nonce
				d.txService.logger.Debug(
					"encountered nonce too low error trying to send raw txn via ethgo, retrying...",
					"function", functionName,
					"try #", txTry,
					"nonce", currNonce,
					"marketHash", report.MarketHash,
				)
				currNonce = common.Max(d.consensusInfo().Nonce, d.consensusInfo().Nonce+1)
				txTry++

				continue
			} else {
				// if any other error, just log and return for now
				d.txService.logger.Error(
					"failed to send raw txn via ethgo due to non-recoverable error",
					"function", functionName,
					"err", err,
					"try #", txTry,
					"nonce", currNonce,
					"marketHash", report.MarketHash,
				)

				return
			}
		}

		d.txService.logger.Debug(
			"sent tx",
			"function", functionName,
			"hash", txn.Hash(),
			"from", ethgo.Address(d.consensusInfo().ValidatorAddress),
			"nonce", currNonce,
			"market", report.MarketHash,
			"outcome", report.Outcome,
		)

		// wait for tx to mine
		receipt := <-d.txService.waitTxConfirmed(txn.Hash())

		if receipt.Status == 1 {
			d.txService.logger.Debug(
				"got success receipt",
				"function", functionName,
				"nonce", currNonce,
				"txHash", txn.Hash(),
				"marketHash", report.MarketHash,
			)

			if functionName == ReportOutcome {
				d.storeProcessor.store.remove(report.MarketHash)
			}

			return
		} else {
			currNonce = common.Max(d.consensusInfo().Nonce, d.consensusInfo().Nonce+1)
			d.txService.logger.Debug(
				"got failed receipt, retrying with nextNonce and more gas",
				"function", functionName,
				"try #", txTry,
				"nonce", currNonce,
				"txHash", txn.Hash(),
				"marketHash", report.MarketHash,
			)
			txTry++
		}
	}
	d.txService.logger.Debug("could not get success tx receipt even after max tx retries",
		"function", functionName,
		"try #", txTry,
		"nonce", currNonce,
		"txHash", txn.Hash(),
		"marketHash", report.MarketHash)

		if functionName == ReportOutcome {
			d.storeProcessor.store.remove(report.MarketHash)
		}
}

// Returns a channel that blocks until the transaction is mined
func (t *TxService) waitTxConfirmed(hash ethgo.Hash) <-chan *ethgo.Receipt {
	ch := make(chan *ethgo.Receipt)
	go func() {
		for {
			var receipt *ethgo.Receipt
			t.client.Call("eth_getTransactionReceipt", &receipt, hash)
			if receipt != nil {
				ch <- receipt
			}

			time.Sleep(time.Millisecond * 500)
		}
	}()

	return ch
}
