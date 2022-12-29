package datafeed

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
	"google.golang.org/grpc"
)

const (
	JSONRPCHost              = "http://localhost:10002"
	proposeOutcomeSCFunction = "function proposeOutcome(bytes32 marketHash, int32 outcome)"
	voteOutcomeSCFunction    = "function voteOutcome(bytes32 marketHash, int32 outcome)"
	reportOutcomeSCFunction  = "function reportOutcome(bytes32 marketHash)"
)

// DataFeed
type DataFeed struct {
	logger hclog.Logger
	config *Config

	// consumes amqp messages
	mqService *MQService

	// networking stack
	topic *network.Topic

	// consensus info function
	consensusInfo consensus.ConsensusInfoFn

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer

	lock sync.Mutex
}

// Config
type Config struct {
	MQConfig         *MQConfig
	VerifyOutcomeURI string
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(
	logger hclog.Logger,
	config *Config,
	grpcServer *grpc.Server,
	consensusInfoFn consensus.ConsensusInfoFn,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger:        logger.Named("datafeed"),
		config:        config,
		consensusInfo: consensusInfoFn,
	}

	// configure and start mqService
	if config.MQConfig.AMQPURI != "" {
		if config.MQConfig.ExchangeName == "" {
			return nil, fmt.Errorf("DataFeed 'amqp_uri' provided but missing a valid 'amqp_exchange_name'")
		}

		if config.MQConfig.QueueConfig.QueueName == "" {
			return nil, fmt.Errorf("DataFeed 'amqp_uri' provided but missing a valid 'amqp_queue_name'")
		}

		mqService, err := newMQService(datafeedService.logger, config.MQConfig, datafeedService)
		if err != nil {
			return nil, err
		}

		datafeedService.mqService = mqService
	}

	// configure grpc operator service
	if grpcServer != nil {
		proto.RegisterDataFeedOperatorServer(grpcServer, datafeedService)
	}

	if config.VerifyOutcomeURI == "" && config.MQConfig.AMQPURI == "" {
		logger.Warn("DataFeed 'verify_outcome_api_url' is missing but required for reporting - we will avoid participating in voting") //nolint:lll

		return datafeedService, nil
	}

	return datafeedService, nil
}

// addNewReport adds new report proposal (e.g. from MQ or GRPC)
func (d *DataFeed) proposeOutcome(payload *proto.DataFeedReport) {
	d.proposeOutcomeTx(payload)
}

// proposeOutcomeTx send proposeOutcome tx to SC specified by customContractAddress
func (d *DataFeed) proposeOutcomeTx(payload *proto.DataFeedReport) {
	const (
		maxTxTries        = 4
		txGasPriceWei     = 1000000000
		txGasLimitWei     = 1000000
		maxTxReceiptTries = 3
		txReceiptWaitMs   = 5000
	)

	d.lock.Lock()
	defer d.lock.Unlock()

	abiContract, err := ethgoabi.NewABIFromList([]string{proposeOutcomeSCFunction})
	if err != nil {
		d.logger.Error("failed to retrieve ethgo ABI", "err", err)

		return
	}

	client, err := jsonrpc.NewClient(JSONRPCHost)
	if err != nil {
		d.logger.Error("failed to initialize new ethgo client", "err", err)

		return
	}

	c := contract.NewContract(
		ethgo.Address(d.consensusInfo().CustomContractAddress),
		abiContract,
		contract.WithSender(wallet.NewKey(d.consensusInfo().ValidatorKey)),
		contract.WithJsonRPC(client.Eth()),
	)

	txn, err := c.Txn(
		"proposeOutcome",
		types.StringToHash(payload.MarketHash),
		payload.Outcome,
	)

	if err != nil {
		d.logger.Error("failed to create txn via ethgo", "err", err)

		return
	}

	txTry := uint64(0)
	currNonce := d.consensusInfo().Nonce

	for txTry < maxTxTries {
		d.logger.Debug("attempting tx with nonce", "nonce", currNonce, "try", txTry)

		//TODO: derive these gas params better, have it dynamic?
		txn.WithOpts(
			&contract.TxnOpts{
				GasPrice: txGasPriceWei + (txTry * txGasPriceWei),
				GasLimit: txGasLimitWei,
				Nonce:    currNonce,
			},
		)

		// TODO: consider adding directly to txpool txpool.AddTx() instead of over local jsonrpc
		err = txn.Do()
		if err != nil {
			if strings.Contains(err.Error(), "nonce too low") {
				// if nonce too low, retry with higher nonce
				d.logger.Debug(
					"encountered nonce too low error trying to send raw txn via ethgo, retrying...",
					"try #", txTry,
					"nonce", currNonce,
					"marketHash", payload.MarketHash,
				)
				currNonce++
				txTry++

				continue
			} else {
				// if any other error, just log and return for now
				d.logger.Error(
					"failed to send raw txn via ethgo due to non-recoverable error",
					"err", err,
					"try #", txTry,
					"nonce", currNonce,
					"marketHash", payload.MarketHash,
				)

				return
			}
		}

		d.logger.Debug(
			"sent tx",
			"hash", txn.Hash(),
			"from", ethgo.Address(d.consensusInfo().ValidatorAddress),
			"nonce", currNonce,
			"market", payload.MarketHash,
			"outcome", payload.Outcome,
		)

		var receipt *ethgo.Receipt

		receiptTry := uint64(0)
		for receiptTry < maxTxReceiptTries {
			time.Sleep(txReceiptWaitMs * time.Millisecond)

			err := client.Call("eth_getTransactionReceipt", &receipt, txn.Hash())

			if receipt != nil {
				break
			}

			if err != nil {
				d.logger.Error("error while waiting for transaction receipt", "err", err.Error(), "try", receiptTry)
			}

			receiptTry++
		}

		if receipt == nil {
			d.logger.Warn(
				"failed to get txn receipt via ethgo, retry with same nonce and more gas",
				"try #", txTry,
				"nonce", currNonce,
				"txHash", txn.Hash(),
				"marketHash", payload.MarketHash,
			)
			txTry++

			continue
		}

		if receipt.Status == 1 {
			d.logger.Debug(
				"got success receipt",
				"nonce", currNonce,
				"txHash", txn.Hash(),
				"marketHash", payload.MarketHash,
			)

			return
		} else {
			currNonce++
			d.logger.Debug(
				"got failed receipt, retrying with nextNonce and more gas",
				"try #", txTry,
				"nonce", currNonce,
				"txHash", txn.Hash(),
				"marketHash", payload.MarketHash,
			)
			txTry++
		}
	}
	d.logger.Debug("could not get success tx receipt even after max tx retries",
		"try #", txTry,
		"nonce", currNonce,
		"txHash", txn.Hash(),
		"marketHash", payload.MarketHash)
}
