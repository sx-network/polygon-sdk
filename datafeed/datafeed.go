package datafeed

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/network"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p-core/peer"
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

	// validator info
	validatorInfo *consensus.ValidatorInfo

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
	validatorInfo *consensus.ValidatorInfo,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger:        logger.Named("datafeed"),
		config:        config,
		validatorInfo: validatorInfo,
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

	//TODO: check if we already signed

	// sigList := strings.Split(dataFeedReportGossip.Signatures, ",")
	// for _, sig := range sigList {

	// 	crypto.SigToPub(sig, []byte(sig))

	// 	buffer := bytes.NewBuffer([]byte(sig))
	// 	dec := gob.NewDecoder(buffer)

	// 	var decoded types.ReportOutcomeGossip
	// 	err := dec.Decode(&decoded)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	//TODO: probably need to add special logic to compare current validatorKey with signatures
	// check if we already signed
	if strings.Contains(dataFeedReportGossip.Signatures, d.validatorInfo.ValidatorAddress) {
		return fmt.Errorf("we already signed this payload")
	}

	// check if payload too old
	d.logger.Debug("time", "time", time.Since(time.Unix(dataFeedReportGossip.Timestamp, 0)).Seconds())
	if time.Since(time.Unix(dataFeedReportGossip.Timestamp, 0)).Seconds() > maxGossipTimestampDriftSeconds {
		return fmt.Errorf("proposed payload is too old")
	}

	return nil
}

// signPayload sings the payload by concatenating our own signature to the signatures field
func (d *DataFeed) signPayload(dataFeedReportGossip *proto.DataFeedReport) (*types.ReportOutcome, bool, error) {

	reportOutcome := &types.ReportOutcome{
		MarketHash: dataFeedReportGossip.MarketHash,
		Outcome:    dataFeedReportGossip.Outcome,
		Epoch:      dataFeedReportGossip.Epoch,
		Timestamp:  dataFeedReportGossip.Timestamp,
		IsGossip:   true,
	}

	sig, err := d.deriveSignature(reportOutcome)
	if err != nil {
		return nil, false, err
	}

	reportOutcome.Signatures = dataFeedReportGossip.Signatures + "," + sig

	numSigs := len(strings.Split(reportOutcome.Signatures, ","))

	return reportOutcome, numSigs >= d.validatorInfo.QuorumSize, nil
}

// deriveSignature derives the signature of the current validator
func (d *DataFeed) deriveSignature(payload *types.ReportOutcome) (string, error) {

	//TODO: figure out how to sign

	// payload.Signatures = ""

	// var buffer bytes.Buffer
	// enc := gob.NewEncoder(&buffer)

	// // Encode the payload
	// err := enc.Encode(payload)
	// if err != nil {
	// 	return "", err
	// }

	// signedData, err := crypto.Sign(d.validatorInfo.ValidatorKey, crypto.Keccak256(buffer.Bytes()))
	// if err != nil {
	// 	return "", err
	// }

	// return string(signedData), nil

	return d.validatorInfo.ValidatorAddress, nil
}

// publishPayload
func (d *DataFeed) publishPayload(message *types.ReportOutcome, isMajoritySigs bool) {
	//TODO: strings for now but eventually parse lsports payload here + process + sign + gossip
	//TODO: should call libp2p publish here after we validate the payload
	d.logger.Debug("Publishing message", "message", message.MarketHash)

	d.logger.Debug(
		"Validator info",
		"privateKey",
		d.validatorInfo.ValidatorKey,
		"validator 1",
		d.validatorInfo.Validators[0],
		"epoch",
		d.validatorInfo.Epoch,
		"quorumSize",
		d.validatorInfo.QuorumSize,
	)

	if isMajoritySigs {
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
		// broadcast the payload only if a topic
		// subscription is present
		if d.topic != nil {

			reportOutcome := &types.ReportOutcome{}

			if !message.IsGossip {
				reportOutcome.Epoch = d.validatorInfo.Epoch
				reportOutcome.Timestamp = time.Now().Unix()
			} else {
				reportOutcome.Epoch = message.Epoch
				reportOutcome.Timestamp = message.Timestamp
			}

			mySig, err := d.deriveSignature(reportOutcome)
			if err != nil {
				d.logger.Error("failed derive signature", "err", err)
			}
			if !message.IsGossip {
				reportOutcome.Signatures = mySig
			} else {
				reportOutcome.Signatures = message.Signatures
			}

			dataFeedReportGossip := &proto.DataFeedReport{
				MarketHash: message.MarketHash,
				Outcome:    message.Outcome,
				Epoch:      reportOutcome.Epoch,
				Timestamp:  reportOutcome.Timestamp,
				Signatures: reportOutcome.Signatures,
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
