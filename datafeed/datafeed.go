package datafeed

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
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

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer
}

// Config
type Config struct {
	MQConfig *MQConfig
}

type MQConfig struct {
	AMQPURI     string
	QueueConfig *QueueConfig
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(
	logger hclog.Logger,
	config *Config,
	grpcServer *grpc.Server,
	network *network.Server,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger: logger.Named("datafeed"),
		config: config,
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
func (d *DataFeed) addGossipMsg(obj interface{}) {
	dataFeedReportGossip, ok := obj.(*proto.DataFeedReport)
	if !ok {
		d.logger.Error("failed to cast gossiped message to DataFeedReport")

		return
	}

	d.logger.Debug("handling gossiped datafeed msg", "msg", dataFeedReportGossip.MarketHash)

	// Verify that the gossiped DataFeedReport message is not empty
	if dataFeedReportGossip == nil {
		d.logger.Error("malformed gossip DataFeedReport message received")

		return
	}

	// validate the payload
	if err := d.validateGossipedPayload(dataFeedReportGossip); err != nil {
		d.logger.Error("gossiped payload is invalid", err)

		return
	}

	// sign payload
	// TODO: we need validatorKey to sign
	signedPayload, isMajoritySigs, err := d.signPayload(dataFeedReportGossip)
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
	if strings.Contains(dataFeedReportGossip.Signatures, "validator-1") {
		return fmt.Errorf("we already signed this payload")
	}

	// check if payload too old
	if time.Since(time.Unix(dataFeedReportGossip.Timestamp, 0)).Seconds() > maxGossipTimestampDriftSeconds {
		return fmt.Errorf("proposed payload is too old")
	}

	return nil
}

// signPayload sings the payload by concatenating our own signature to the signatures field
func (d *DataFeed) signPayload(dataFeedReportGossip *proto.DataFeedReport) (string, bool, error) {
	isMajoritySigs := false

	signedReportOutcome := &types.ReportOutcomeGossip{
		MarketHash: dataFeedReportGossip.MarketHash,
		Outcome:    dataFeedReportGossip.Outcome,
		Signatures: dataFeedReportGossip.Signatures + ",validator-1",
		Epoch:      dataFeedReportGossip.Epoch,
		Timestamp:  dataFeedReportGossip.Timestamp,
	}

	numSigs := len(strings.Split(signedReportOutcome.Signatures, ","))

	// TODO: will need a reference to current validator Set
	// so we can pass to OptimalQuorumSize and use this number instead of 2
	if majorityThreshold := 2; numSigs < majorityThreshold {
		isMajoritySigs = true
	}

	reportOutcomeString, err := json.Marshal(signedReportOutcome)
	if err != nil {
		return "", false, err
	}

	return string(reportOutcomeString), isMajoritySigs, nil
}

// publishPayload
func (d *DataFeed) publishPayload(message string, isMajoritySigs bool) {
	//TODO: strings for now but eventually parse lsports payload here + process + sign + gossip
	//TODO: should call libp2p publish here after we validate the payload
	d.logger.Debug("Publishing message", "message", message)

	if isMajoritySigs {
		//TODO: write to SC
	} else {
		// broadcast the payload only if a topic
		// subscription is present
		if d.topic != nil {
			dataFeedReportGossip := &proto.DataFeedReport{
				MarketHash: "asdf",
				Outcome:    "asdf",
				Signatures: "asdf",
				Epoch:      1234,
				Timestamp:  1234,
			}

			if err := d.topic.Publish(dataFeedReportGossip); err != nil {
				d.logger.Error("failed to topic report", "err", err)
			}
		}
	}
}
