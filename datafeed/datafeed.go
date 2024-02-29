package datafeed

import (
	"fmt"
	"math/big"
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

	// channel used to queue reporting txs for processing
	reportingTxChan chan *ReportingTx

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
	SXNodeAddress              string
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

	// start jsonRpc TxService
	txService, err := newTxService(datafeedService.logger)
	if err != nil {
		return nil, err
	}
	datafeedService.txService = txService

	// start txWorker
	go datafeedService.processTxsFromQueue()

	if config.VerifyOutcomeURI == "" {
		datafeedService.logger.Warn("Datafeed 'verify_outcome_api_url' is missing but required for outcome voting and reporting.. we will avoid participating in outcome voting and reporting...") //nolint:lll

		return datafeedService, nil
	}

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

// queueReportingTx Queues the reporting tx by writing to reportTxChan to be processed by txWorker
func (d *DataFeed) queueReportingTx(functionType string, marketHash string, outcome int32) {

	reportingTx := &ReportingTx{
		functionType: functionType,
		report: &proto.DataFeedReport{
			MarketHash: marketHash,
		},
	}

	switch functionType {
	case ProposeOutcome:
		reportingTx.report.Outcome = outcome
	case VoteOutcome:
		verifyOutcome, err := d.verifyMarket(marketHash)
		reportingTx.report.Outcome = verifyOutcome
		if err != nil {
			d.logger.Error("Error encountered in verifying market, skipping vote tx", "err", err)
			return
		}
	default:
		if functionType != ReportOutcome {
			d.logger.Error("Unrecognized function type, skipping tx..", "functionType", functionType)
			return
		}
	}

	d.logger.Debug("queueing reporting tx for processing", "function", functionType, "marketHash", marketHash)
	d.reportingTxChan <- reportingTx
}

// txWorker processes each tx in the queue
func (d *DataFeed) processTxsFromQueue() {
	for reportingTx := range d.reportingTxChan {
		d.logger.Debug("processing reporting tx", "function", reportingTx.functionType, "marketHash", reportingTx.report.MarketHash)
		d.sendTxWithRetry(reportingTx.functionType, reportingTx.report)
	}
}

// syncVotingPeriod synchronizes the outcome voting period onchain with the local configuration
func (d *DataFeed) syncVotingPeriod() {
	result := d.sendCall("_votingPeriod")
	if result == nil {
		d.logger.Error("voting period returned nil")
		return
	}

	votingPeriodOnchain, ok := result.(*big.Int)
	if !ok {
		d.logger.Error("failed to convert result to *big.Int")
		return
	}
	d.logger.Debug("retrieved onchain voting period", votingPeriodOnchain)

	d.logger.Debug("update voting period", votingPeriodOnchain)
	d.config.OutcomeVotingPeriodSeconds = votingPeriodOnchain.Uint64()
}
