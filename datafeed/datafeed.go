package datafeed

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/crypto"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/umbracle/ethgo"
	ethgoabi "github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"google.golang.org/grpc"
)

const (
	topicNameV1                    = "datafeed/0.1"
	maxGossipTimestampDriftSeconds = 10
	JSONRPCHost                    = "http://3.221.242.145:10002"
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

	// the last paload we published to SC, used to avoid posting dupes to SC
	lastPublishedPayload string

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer
}

// Config
type Config struct {
	MQConfig *MQConfig
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

	// configure and start mqService
	if config.MQConfig.AMQPURI != "" {
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

	//TODO: add 'Ethereum Signed Message' here?
	// 	prefixedHash := crypto.Keccak256Hash(
	// 		[]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%v", len(hash))),
	// 		hash.Bytes(),
	// )

	return crypto.Keccak256(bytes)
}

// validateSignatures checks if the current validator has already signed the payload
func (d *DataFeed) validateSignatures(payload *proto.DataFeedReport) (bool, error) {
	sigList := strings.Split(payload.Signatures, ",")
	for _, sig := range sigList {
		buf, err := hex.DecodeHex(sig)
		if err != nil {
			return false, err
		}

		pub, err := crypto.RecoverPubkey(buf, d.AbiEncode(payload))
		if err != nil {
			return false, err
		}

		// see if we signed it
		if d.consensusInfo().ValidatorAddress == crypto.PubKeyToAddress(pub) {
			return true, nil
		} else {
			// if we haven't signed it, see if a recognized validator from the current validator set signed it
			otherValidatorSigned := false
			for _, validator := range d.consensusInfo().Validators {
				if validator == crypto.PubKeyToAddress(pub) {
					otherValidatorSigned = true
				}
			}
			if !otherValidatorSigned {
				return false, fmt.Errorf("unrecognized signature detected")
			}
		}
	}

	return false, nil
}

// validateTimestamp  checks if payload too old
func (d *DataFeed) validateTimestamp(payload *proto.DataFeedReport) bool {
	d.logger.Debug("time", "time", time.Since(time.Unix(payload.Timestamp, 0)).Seconds())

	return time.Since(time.Unix(payload.Timestamp, 0)).Seconds() > maxGossipTimestampDriftSeconds
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

// GetSignatureForPayload derives the signature of the current validator
func (d *DataFeed) GetSignatureForPayload(payload *proto.DataFeedReport) (string, error) {
	signedData, err := crypto.Sign(d.consensusInfo().ValidatorKey, d.AbiEncode(payload))
	if err != nil {
		return "", err
	}

	return hex.EncodeToHex(signedData), nil
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
		"Publising new msg to topic",
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
			"Publising msg to topic",
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
	if d.lastPublishedPayload == payload.MarketHash+fmt.Sprint(payload.Timestamp) {
		d.logger.Debug("we've already tried to report this signed outcome ", "marketHash", payload.MarketHash)

		return
	}

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
		contract.WithSender(ethgo.Address(d.consensusInfo().ValidatorAddress)),
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

	//TODO: derive these gas params better
	txn.WithOpts(
		&contract.TxnOpts{
			GasLimit: 200000,
			GasPrice: 30,
		},
	)

	if err != nil {
		d.logger.Error("failed to create txn via ethgo", "err", err)

		return
	}

	err = txn.Do()
	if err != nil {
		d.logger.Error("failed to send raw txn via ethgo", "err", err)

		return
	}

	// TODO: investigating simultaneous tx failures due to nonce
	nonce, _ := client.Eth().GetNonce(ethgo.Address(d.consensusInfo().ValidatorAddress), ethgo.Latest)
	d.logger.Debug("sent tx", "market", payload.MarketHash, "from", ethgo.Address(d.consensusInfo().ValidatorAddress), "hash", txn.Hash(), "nonce", nonce)

	// do not wait for receipt as it blocks
	// receipt, err := txn.Wait()
	// if err != nil {
	// 	d.logger.Error("failed to get txn receipt via ethgo", "err", err)

	// 	return
	// }

	d.lastPublishedPayload = payload.MarketHash + fmt.Sprint(payload.Timestamp)
}
