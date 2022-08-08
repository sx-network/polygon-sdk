package datafeed

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/contracts/datafeed"
	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p-core/peer"
	"google.golang.org/grpc"
	protobuf "google.golang.org/protobuf/proto"
)

const (
	topicNameV1                    = "datafeed/0.1"
	maxGossipTimestampDriftSeconds = 10
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
	consensusInfoFn consensus.ConsensusInfoFn

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer
}

// Config
type Config struct {
	CustomContractAddress types.Address
	MQConfig              *MQConfig
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(
	logger hclog.Logger,
	config *Config,
	grpcServer *grpc.Server,
	network *network.Server,
	validatorInfoFn consensus.ConsensusInfoFn,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger:          logger.Named("datafeed"),
		config:          config,
		consensusInfoFn: validatorInfoFn,
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
	if network != nil {
		// subscribe to the gossip protocol
		topic, err := network.NewTopic(topicNameV1, &proto.DataFeedReport{})
		if err != nil {
			return nil, err
		}

		if subscribeErr := topic.Subscribe(datafeedService.addGossipMsg); subscribeErr != nil {
			return nil, fmt.Errorf("unable to subscribe to gossip topic, %w", subscribeErr)
		}

		datafeedService.topic = topic
	}

	return datafeedService, nil
}

// addGossipMsg handles receiving msgs
// gossiped by the network.
func (d *DataFeed) addGossipMsg(obj interface{}, _ peer.ID) {
	dataFeedReportGossip, ok := obj.(*proto.DataFeedReport)
	if !ok {
		d.logger.Error("failed to cast gossiped message to DataFeedReport")

		return
	}

	// Verify that the gossiped DataFeedReport message is not empty
	if dataFeedReportGossip == nil {
		d.logger.Error("malformed gossip DataFeedReport message received")

		return
	}

	d.logger.Debug("handling gossiped datafeed msg", "msg", dataFeedReportGossip.MarketHash)

	// validate the payload
	if err := d.validateGossipedPayload(dataFeedReportGossip); err != nil {
		d.logger.Warn("gossiped payload is invalid", "reason", err)

		return
	}

	// sign payload
	signedPayload, isMajoritySigs, err := d.signGossipedPayload(dataFeedReportGossip)
	if err != nil {
		d.logger.Error("unable to sign payload")

		return
	}

	// finally publish it
	d.publishPayload(signedPayload, isMajoritySigs)
}

// validateGossipedPayload performs validation steps on gossiped payload prior to signing
func (d *DataFeed) validateGossipedPayload(dataFeedReportGossip *proto.DataFeedReport) error {
	// check if we already signed
	isAlreadySigned, err := d.validateSignatures(dataFeedReportGossip)
	if err != nil {
		return err
	}

	if isAlreadySigned {
		return fmt.Errorf("we already signed this payload")
	}

	// check if payload too old
	if d.validateTimestamp(dataFeedReportGossip) {
		return fmt.Errorf("proposed payload is too old")
	}

	return nil
}

// validateSignatures checks if the current validator has already signed the payload
func (d *DataFeed) validateSignatures(dataFeedReportGossip *proto.DataFeedReport) (bool, error) {
	clonedMsg, ok := protobuf.Clone(dataFeedReportGossip).(*proto.DataFeedReport)
	if !ok {
		return false, fmt.Errorf("error while determining if we already signed")
	}

	clonedMsg.Signatures = ""
	data, err := protobuf.Marshal(clonedMsg)

	if err != nil {
		return false, err
	}

	sigList := strings.Split(dataFeedReportGossip.Signatures, ",")
	for _, sig := range sigList {
		buf, err := hex.DecodeHex(sig)
		if err != nil {
			return false, err
		}

		pub, err := crypto.RecoverPubkey(buf, crypto.Keccak256(data))
		if err != nil {
			return false, err
		}

		// see if we signed it
		if d.consensusInfoFn().ValidatorAddress == crypto.PubKeyToAddress(pub) {
			return true, nil
		} else {
			// if we haven't signed it, see if a recognized validator from the current validator set signed it
			otherValidatorSigned := false
			for _, validator := range d.consensusInfoFn().Validators {
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
func (d *DataFeed) validateTimestamp(dataFeedReportGossip *proto.DataFeedReport) bool {
	d.logger.Debug("time", "time", time.Since(time.Unix(dataFeedReportGossip.Timestamp, 0)).Seconds())

	return time.Since(time.Unix(dataFeedReportGossip.Timestamp, 0)).Seconds() > maxGossipTimestampDriftSeconds
}

// signGossipedPayload sings the payload by concatenating our own signature to the signatures field
func (d *DataFeed) signGossipedPayload(dataFeedReportGossip *proto.DataFeedReport) (*types.ReportOutcome, bool, error) {
	reportOutcome := &types.ReportOutcome{
		MarketHash: dataFeedReportGossip.MarketHash,
		Outcome:    dataFeedReportGossip.Outcome,
		Epoch:      dataFeedReportGossip.Epoch,
		Timestamp:  dataFeedReportGossip.Timestamp,
		IsGossip:   true,
	}

	sig, err := d.getSignatureForPayload(dataFeedReportGossip)
	if err != nil {
		return nil, false, err
	}

	reportOutcome.Signatures = dataFeedReportGossip.Signatures + "," + sig

	numSigs := len(strings.Split(reportOutcome.Signatures, ","))

	return reportOutcome, numSigs >= d.consensusInfoFn().QuorumSize, nil
}

// getSignatureForPayload derives the signature of the current validator
func (d *DataFeed) getSignatureForPayload(payload *proto.DataFeedReport) (string, error) {
	clonedMsg, ok := protobuf.Clone(payload).(*proto.DataFeedReport)
	if !ok {
		return "", fmt.Errorf("error while trying to clone datafeed report")
	}

	clonedMsg.Signatures = ""

	data, err := protobuf.Marshal(clonedMsg)
	if err != nil {
		return "", err
	}

	signedData, err := crypto.Sign(d.consensusInfoFn().ValidatorKey, crypto.Keccak256(data))
	if err != nil {
		return "", err
	}

	return hex.EncodeToHex(signedData), nil
}

// publishPayload
func (d *DataFeed) publishPayload(message *types.ReportOutcome, isMajoritySigs bool) {
	if isMajoritySigs {

		// TODO: can we only execute a publish payload if we are currently writing a block??? maybe we need to somehow
		// queue one until it gets picked up by a block proposer

		// every validator will have a queue that it will add to here and read from when reaching a hook triggered on every block
		// if the queue isn't empty, it will attempt to apply txn from queue
		// once validator writes to SC, it should gossip this so other validators remove from their queues
		header, _ := d.consensusInfoFn().Blockchain.GetHeaderByNumber(1)
		t, err := d.consensusInfoFn().Executor.BeginTxn(header.Hash, header, types.ZeroAddress)
		if err != nil {
			d.logger.Error("failed to begin txn", "err", err)
		}

		_, err = datafeed.ReportOutcome(t, d.consensusInfoFn().ValidatorAddress, d.config.CustomContractAddress)
		if err != nil {
			d.logger.Error("failed to call ReportOutcome", "err", err)
		}

		//TODO: write to SC
		d.logger.Debug(
			"Majority of sigs reached, writing payload to SC",
			"marketHash",
			message.MarketHash,
			"outcome",
			message.Outcome,
			"epoch",
			message.Epoch,
			"timestamp",
			message.Timestamp,
			"signatures",
			message.Signatures,
		)
	} else {
		if d.topic != nil {
			// broadcast the payload only if a topic subscription present
			dataFeedReportGossip := &proto.DataFeedReport{
				MarketHash: message.MarketHash,
				Outcome:    message.Outcome,
			}

			if message.IsGossip {
				dataFeedReportGossip.Epoch = message.Epoch
				dataFeedReportGossip.Timestamp = message.Timestamp
				dataFeedReportGossip.Signatures = message.Signatures
			} else {
				dataFeedReportGossip.Epoch = d.consensusInfoFn().Epoch
				dataFeedReportGossip.Timestamp = time.Now().Unix()

				mySig, err := d.getSignatureForPayload(dataFeedReportGossip)
				if err != nil {
					d.logger.Error("failed to get payload signature", "err", err)
				}

				dataFeedReportGossip.Signatures = mySig
			}

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
	}
}
