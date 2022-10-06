package datafeed

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
	"google.golang.org/grpc"
)

const (
	topicNameV1                    = "datafeed/0.1"
	maxGossipTimestampDriftSeconds = 10
	JSONRPCHost                    = "http://localhost:10002"
	//nolint:lll
	reportOutcomeSCFunction = "function reportOutcome(bytes32 marketHash, int32 outcome, uint64 epoch, uint256 timestamp, bytes[] signatures)"
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

	// the last paload marketHash we published to SC, used to avoid posting dupes to SC
	lastPublishedMarketHash string

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
	network *network.Server,
	consensusInfoFn consensus.ConsensusInfoFn,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger:        logger.Named("datafeed"),
		config:        config,
		consensusInfo: consensusInfoFn,
	}

	if config.VerifyOutcomeURI == "" {
		return nil, fmt.Errorf("DataFeed 'verify_outcome_api_url' is not configured")
	}

	// configure and start mqService
	if config.MQConfig.AMQPURI != "" {
		if config.MQConfig.ExchangeName == "" {
			return nil, fmt.Errorf("DataFeed AMQPURI provided without a valid ExchangeName")
		}

		if config.MQConfig.QueueConfig.QueueName == "" {
			return nil, fmt.Errorf("DataFeed AMQPURI provided without a valid QueueName")
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

	// configure libp2p
	if network == nil {
		return nil, fmt.Errorf("network must be non-nil to start gossip protocol")
	}

	// subscribe to the gossip protocol
	topic, err := network.NewTopic(topicNameV1, &proto.DataFeedReport{})
	if err != nil {
		return nil, err
	}

	if subscribeErr := topic.Subscribe(datafeedService.addGossipMsg); subscribeErr != nil {
		return nil, fmt.Errorf("unable to subscribe to gossip topic, %w", subscribeErr)
	}

	datafeedService.topic = topic

	return datafeedService, nil
}

// addGossipMsg handler for messages on the gossip protocol
func (d *DataFeed) addGossipMsg(obj interface{}, _ peer.ID) {
	payload, ok := obj.(*proto.DataFeedReport)
	if !ok {
		d.logger.Error("failed to cast gossiped message to DataFeedReport")

		return
	}

	// Verify that the gossiped DataFeedReport message is not empty
	if payload == nil {
		d.logger.Error("malformed gossip DataFeedReport message received")

		return
	}

	d.logger.Debug("handling gossiped datafeed msg", "msg", payload.MarketHash)

	// validate the payload
	if err := d.validateGossipedPayload(payload); err != nil {
		d.logger.Warn("gossiped payload is invalid", "reason", err)

		return
	}

	// sign payload
	signedPayload, isMajoritySigs, err := d.signGossipedPayload(payload)
	if err != nil {
		d.logger.Error("unable to sign payload")

		return
	}

	// finally publish payload
	d.publishPayload(signedPayload, isMajoritySigs)
}

// validateGossipedPayload performs validation steps on gossiped payload prior to signing
func (d *DataFeed) validateGossipedPayload(payload *proto.DataFeedReport) error {
	// check if we already signed
	isAlreadySigned, err := d.validateSignatures(payload)
	if err != nil {
		return err
	}

	if isAlreadySigned {
		return fmt.Errorf("we already signed this payload")
	}

	// check if payload too old
	if d.validateTimestamp(payload) {
		return fmt.Errorf("proposed payload is too old")
	}

	verifyErr := d.verifyMarketOutcome(payload, d.config.VerifyOutcomeURI)

	if verifyErr != nil {
		return verifyErr
	}

	return nil
}

// encodes to conform to solidity's keccak256(abi.encode())
func (d *DataFeed) AbiEncode(payload *proto.DataFeedReport) []byte {
	bytes32Type, _ := abi.NewType("bytes32", "bytes32", nil)
	int32Type, _ := abi.NewType("int32", "int32", nil)
	uint64Type, _ := abi.NewType("uint64", "uint64", nil)
	uint256Type, _ := abi.NewType("uint256", "uint256", nil)

	arguments := abi.Arguments{
		{
			Type: bytes32Type,
		},
		{
			Type: int32Type,
		},
		{
			Type: uint64Type,
		},
		{
			Type: uint256Type,
		},
	}

	bytes, _ := arguments.Pack(
		types.StringToHash(payload.MarketHash),
		payload.Outcome,
		payload.Epoch,
		big.NewInt(payload.Timestamp),
	)

	hashedPayload := crypto.Keccak256(bytes)
	prefixedHash := crypto.Keccak256(
		[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(hashedPayload))),
		hashedPayload,
	)

	return prefixedHash
}

// signGossipedPayload sings the payload by concatenating our own signature to the signatures field
func (d *DataFeed) signGossipedPayload(payload *proto.DataFeedReport) (*proto.DataFeedReport, bool, error) {
	reportMinusSigs := &proto.DataFeedReport{
		MarketHash: payload.MarketHash,
		Outcome:    payload.Outcome,
		Epoch:      payload.Epoch,
		Timestamp:  payload.Timestamp,
	}

	sig, err := d.GetSignatureForPayload(payload)
	if err != nil {
		return nil, false, err
	}

	reportMinusSigs.Signatures = payload.Signatures + "," + sig

	numSigs := len(strings.Split(reportMinusSigs.Signatures, ","))

	return reportMinusSigs, numSigs >= d.consensusInfo().QuorumSize, nil
}

// addNewReport adds new report proposal (e.g. from MQ or GRPC)
func (d *DataFeed) addNewReport(payload *proto.DataFeedReport) {
	if d.topic == nil {
		d.logger.Error("Topic must be set in order to gossip new report payload")

		return
	}

	// broadcast the payload only if a topic subscription present
	dataFeedReportGossip := &proto.DataFeedReport{
		MarketHash: payload.MarketHash,
		Outcome:    payload.Outcome,
		Epoch:      d.consensusInfo().Epoch,
		Timestamp:  time.Now().Unix(),
	}

	mySig, err := d.GetSignatureForPayload(dataFeedReportGossip)
	if err != nil {
		d.logger.Error("failed to get payload signature", "err", err)

		return
	}

	dataFeedReportGossip.Signatures = mySig

	d.logger.Debug(
		"Publishing new msg to topic",
		"marketHash",
		dataFeedReportGossip.MarketHash,
		"outcome",
		dataFeedReportGossip.Outcome,
		"epoch",
		dataFeedReportGossip.Epoch,
		"timestamp",
		dataFeedReportGossip.Timestamp,
		"signatures",
		dataFeedReportGossip.Signatures,
	)

	if err := d.topic.Publish(dataFeedReportGossip); err != nil {
		d.logger.Error("failed to topic report", "err", err)
	}
}

// publishPayload
func (d *DataFeed) publishPayload(payload *proto.DataFeedReport, isMajoritySigs bool) {
	if isMajoritySigs {
		d.logger.Debug(
			"Majority of sigs reached, writing payload to SC",
			"marketHash",
			payload.MarketHash,
			"outcome",
			payload.Outcome,
			"epoch",
			payload.Epoch,
			"timestamp",
			payload.Timestamp,
			"signatures",
			payload.Signatures,
		)
		d.reportOutcomeToSC(payload)
	} else {
		if d.topic == nil {
			d.logger.Error("Topic must be set in order to use gossip protocol")

			return
		}

		d.logger.Debug(
			"Publishing msg to topic",
			"marketHash",
			payload.MarketHash,
			"outcome",
			payload.Outcome,
			"epoch",
			payload.Epoch,
			"timestamp",
			payload.Timestamp,
			"signatures",
			payload.Signatures,
		)

		if err := d.topic.Publish(payload); err != nil {
			d.logger.Error("failed to topic report", "err", err)
		}
	}
}

// reportOutcomeToSC write the report outcome to the current ibft fork's customContractAddress SC
func (d *DataFeed) reportOutcomeToSC(payload *proto.DataFeedReport) {
	const (
		maxTxTries        = 4
		txGasPriceWei     = 1000000000
		txGasLimitWei     = 1000000
		maxTxReceiptTries = 3
		txReceiptWaitMs   = 5000
	)

	d.lock.Lock()
	defer d.lock.Unlock()

	if d.lastPublishedMarketHash == payload.MarketHash {
		d.logger.Debug("skipping sending tx for payload we already reported", "marketHash", payload.MarketHash)

		return
	}

	d.lastPublishedMarketHash = payload.MarketHash

	abiContract, err := ethgoabi.NewABIFromList([]string{reportOutcomeSCFunction})
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

	sigList := strings.Split(payload.Signatures, ",")

	sigByteList := make([][]byte, len(sigList))

	for i, v := range sigList {
		decoded, err := hex.DecodeHex(v)
		if err != nil {
			d.logger.Error("failed to prepare signatures arg for reportOutcome() txn ", "err", err)

			return
		}

		sigByteList[i] = decoded
	}

	txn, err := c.Txn(
		"reportOutcome",
		types.StringToHash(payload.MarketHash),
		payload.Outcome,
		payload.Epoch,
		new(big.Int).SetInt64(payload.Timestamp),
		sigByteList,
	)

	if err != nil {
		d.logger.Error("failed to create txn via ethgo", "err", err)

		return
	}

	txTry := uint64(0)
	currNonce := d.consensusInfo().Nonce

	//TODO: we don't need this anymore as we are now waiting for receipt
	// in the event that the account's nonce hasn't been updated yet in memory or on state,
	// ensure we increment the nonce
	// if d.lastNonce == currNonce {
	// 	currNonce = currNonce + 1
	// }

	// d.lastNonce = currNonce

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
			"signatures", payload.Signatures,
			"timestamp", payload.Timestamp,
			"epoch", payload.Epoch,
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
			d.logger.Error(
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
