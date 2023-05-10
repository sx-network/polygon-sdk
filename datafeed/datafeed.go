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

	// sends json-rpc txs
	txService *TxService

	// listens for ProposeOutcome and OutcomeReported events on OutcomeReporter
	eventListener *EventListener

	// consensus info function
	consensusInfo consensus.ConsensusInfoFn

	// processes marketItems once they are ready to be reported
	storeProcessor *StoreProcessor

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer

	lock sync.Mutex

	reportingTxChan chan *ReportingTx
}

// Config
type Config struct {
	MQConfig                   *MQConfig
	VerifyOutcomeURI           string
	OutcomeVotingPeriodSeconds uint64
	OutcomeReporterAddress     string
}

type ReportingTx struct {
	functionType string
	report       *proto.DataFeedReport
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(
	logger hclog.Logger,
	config *Config,
	grpcServer *grpc.Server,
	consensusInfoFn consensus.ConsensusInfoFn,
) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger:          logger.Named("datafeed"),
		config:          config,
		consensusInfo:   consensusInfoFn,
		reportingTxChan: make(chan *ReportingTx, 100),
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

	// start jsonRpcTxService
	txService, err := newTxService(datafeedService.logger)
	if err != nil {
		return nil, err
	}
	datafeedService.txService = txService

	if config.VerifyOutcomeURI == "" {
		datafeedService.logger.Warn("Datafeed 'verify_outcome_api_url' is missing but required for outcome voting and reporting.. we will avoid participating in outcome voting and reporting...") //nolint:lll

		return datafeedService, nil
	}

	// start txWorker
	go datafeedService.txWorker()

	// start eventListener
	eventListener, err := newEventListener(datafeedService.logger, datafeedService)
	if err != nil {
		return nil, err
	}
	datafeedService.eventListener = eventListener

	// start storeProcessor
	storeProcessor, err := newStoreProcessor(datafeedService.logger, datafeedService)
	if err != nil {
		return nil, err
	}
	datafeedService.storeProcessor = storeProcessor

	return datafeedService, nil
}

// proposeOutcome proposes new report outcome (from a datafeed source like MQ or GRPC)
func (d *DataFeed) proposeOutcome(report *proto.DataFeedReport) {

	reportingTx := &ReportingTx{
		functionType: ProposeOutcome,
		report: &proto.DataFeedReport{
			MarketHash: report.MarketHash,
			Outcome:    report.Outcome,
		},
	}

	d.reportingTxChan <- reportingTx
}

// voteOutcome adds vote for a previously proposed report outcome
func (d *DataFeed) voteOutcome(marketHash string) {

	outcome, err := d.verifyMarket(marketHash)
	if err != nil {
		d.logger.Error("Error encountered in verifying market, skipping vote tx", "err", err)
		return
	}

	reportingTx := &ReportingTx{
		functionType: VoteOutcome,
		report: &proto.DataFeedReport{
			MarketHash: marketHash,
			Outcome:    outcome,
		},
	}

	d.reportingTxChan <- reportingTx
}

// reportOutcome calls the reportOutcome function to publish the outcome once the voting period has ended
func (d *DataFeed) reportOutcome(marketHash string) {
	reportingTx := &ReportingTx{
		functionType: ReportOutcome,
		report: &proto.DataFeedReport{
			MarketHash: marketHash,
		},
	}

	d.reportingTxChan <- reportingTx
}

// txWorker processes each transaction separately
func (d *DataFeed) txWorker() {
	for reportingTx := range d.reportingTxChan {
		d.sendTxWithRetry(reportingTx.functionType, reportingTx.report)
	}
}
