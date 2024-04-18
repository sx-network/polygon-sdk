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

// Is a struct type representing a reporting transaction.
// The ReportingTx struct holds information about individual transactions,
// while the DataFeedReport struct holds the detailed information about a report within those transactions.
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

	fmt.Println("[datafeed][NewDataFeedService] 1")

	datafeedService := &DataFeed{
		logger:          logger.Named("datafeed"), // @todo rename to reporter
		config:          config,

		// This allows the DataFeed instance to retrieve consensus information when needed by calling consensusInfoFn().
		consensusInfo:   consensusInfoFn, 

		/*
			Buffered channel named reportingTxChan that can hold up to 200 elements of type *ReportingTx
			If there are already 200 elements in the channel, any further attempts to send to the channel 
			will block until some elements are received by the receiver and space becomes available in the channel.
		*/
		reportingTxChan: make(chan *ReportingTx, 200), 
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
	/*
		In this context, gRPC is being used to facilitate communication between different components of a distributed system. 
		Specifically, it's used for communication between the gRPC server (which could be serving as an operator for the data 
		feed service) and its clients or other services. The proto.RegisterDataFeedOperatorServer function helps in this 
		context by registering the implementation of the data feed operator service with the gRPC server, allowing clients
		to make remote procedure calls to the methods provided by the data feed operator service over the gRPC protocol. 
		This registration process enables clients to interact with the data feed service seamlessly over the network 
		using gRPC.
	*/
	if grpcServer != nil {
		proto.RegisterDataFeedOperatorServer(grpcServer, datafeedService)
	}

	// start jsonRpc TxService
	/*
	This transaction service likely provides functionality for interacting with an Ethereum node or another
	 blockchain network via JSON-RPC calls.
	*/
	txService, err := newTxService(datafeedService.logger) 
	if err != nil {
		return nil, err
	}
	datafeedService.txService = txService

	// start txWorker
	/*
		In summary, datafeedService.processTxsFromQueue() starts a goroutine to continuously process reporting 
		transactions from the queue (reportingTxChan) within the DataFeed service. It retrieves each transaction 
		from the channel and logs its details before sending it for further processing, 
		likely involving interactions with external systems or services.
	*/
	go datafeedService.processTxsFromQueue() 

	if config.VerifyOutcomeURI == "" {
		datafeedService.logger.Warn("Datafeed 'verify_outcome_api_url' is missing but required for outcome voting and reporting.. we will avoid participating in outcome voting and reporting...") //nolint:lll

		return datafeedService, nil
	}

	// start eventListener
	/**
		initializes an event listener for a specific smart contract, establishes a connection to the JSON-RPC WebSocket server, 
		and starts listening for events. The startListeningLoop method continuously listens for events, handles errors, 
		unpacks event data, and performs corresponding actions based on the received events. This event listener plays a
		crucial role in synchronizing data and reacting to events emitted by the
		smart contract within the context of the data feed service.
	**/
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
	datafeedService.storeProcessor = storeProcessor // @note ?

	return datafeedService, nil
}


/*
	In summary, when a report message is received from the message queue, this part of the code 
	prepares and queues a reporting transaction based on the received report, setting the outcome 
	according to the function type, and then sends it to be processed by the txWorker of the DataFeed service.

*/
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
