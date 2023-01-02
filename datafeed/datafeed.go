package datafeed

import (
	"fmt"
	"sync"

	"github.com/0xPolygon/polygon-edge/consensus"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

// DataFeed
type DataFeed struct {
	logger hclog.Logger
	config *Config

	// consumes amqp messages
	mqService *MQService

	// listens for ProposeOutcome and OutcomeReported events on OutcomeReporter
	eventListener *EventListener

	// consensus info function
	consensusInfo consensus.ConsensusInfoFn

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer

	lock sync.Mutex
}

// Config
type Config struct {
	MQConfig                   *MQConfig
	VerifyOutcomeURI           string
	OutcomeVotingPeriodSeconds uint64
	OutcomeReporterAddress     string
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
		logger.Warn("DataFeed 'verify_outcome_api_url' is missing but required for reporting - we will avoid participating in voting and reporting") //nolint:lll

		return datafeedService, nil
	}

	//TODO: if they aren't participating in voting, validators cannot report outcomes
	eventListener, err := newEventListener(datafeedService.logger, datafeedService)
	if err != nil {
		return nil, err
	}

	datafeedService.eventListener = eventListener

	return datafeedService, nil
}

// proposeOutcome proposes new report outcome (from a datafeed source like MQ or GRPC)
func (d *DataFeed) proposeOutcome(payload *proto.DataFeedReport) {
	d.sendTxWithRetry(ProposeOutcome, payload)
}

// voteOutcome adds vote for a previously proposed report outcome
func (d *DataFeed) voteOutcome(payload *proto.DataFeedReport) {
	d.sendTxWithRetry(VoteOutcome, payload)
}

// publishOutcome calls the reportOutcome function to publish the outcome once the voting period has ended
func (d *DataFeed) publishOutcome(payload *proto.DataFeedReport) {
	d.sendTxWithRetry(ReportOutcome, payload)
}
