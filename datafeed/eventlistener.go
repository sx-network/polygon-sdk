package datafeed

import (
	"context"
	"strings"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/go-hclog"
)

// EventListener
type EventListener struct {
	logger          hclog.Logger
	datafeedService *DataFeed
}

func newEventListener(logger hclog.Logger, datafeedService *DataFeed) (*EventListener, error) {
	eventListener := &EventListener{
		logger:          logger.Named("eventListener"),
		datafeedService: datafeedService,
	}

	go eventListener.startListeningLoop()

	return eventListener, nil
}

func (e EventListener) startListeningLoop() {
	client, err := ethclient.Dial("ws://localhost:10002/ws")
	if err != nil {
		e.logger.Error("error while starting event listener", "err", err)

		return
	}

	contractAbi, err := abi.JSON(strings.NewReader(abis.OutcomeReporterJSONABI))
	if err != nil {
		e.logger.Error("error while parsing OutcomeReporter contract ABI", "err", err)

		return
	}

	outcomeReporterAddress := common.HexToAddress(e.datafeedService.config.OutcomeReporterAddress)

	proposeOutcomeEvent := contractAbi.Events["ProposeOutcome"].ID

	var proposeOutcomeEventTopics [][]common.Hash

	proposeOutcomeEventTopics = append(proposeOutcomeEventTopics, []common.Hash{proposeOutcomeEvent})

	proposeOutcomeQuery := ethereum.FilterQuery{
		Addresses: []common.Address{outcomeReporterAddress},
		//TODO: figure out how to listen for specific events here if using topics?
		// https://sourcegraph.com/github.com/obscuronet/go-obscuro/-/blob/integration/simulation/validate_chain.go?L225:20
		Topics: proposeOutcomeEventTopics,
	}

	proposeOutcomeLogs := make(chan types.Log)
	//TODO: might have to just use FilterLogs instead of ws SubscribeFilterLogs
	proposeOutcomeSub, err := client.SubscribeFilterLogs(context.Background(), proposeOutcomeQuery, proposeOutcomeLogs)
	if err != nil {
		e.logger.Error("error while subscribing to ProposeOutcome logs", "err", err)

		return
	}

	outcomeReportedEvent := contractAbi.Events["OutcomeReported"].ID

	var outcomeReportedEventTopics [][]common.Hash
	outcomeReportedEventTopics = append(outcomeReportedEventTopics, []common.Hash{outcomeReportedEvent})

	outcomeReportedQuery := ethereum.FilterQuery{
		Addresses: []common.Address{outcomeReporterAddress},
		//TODO: figure out how to listen for specific events here if using topics?
		// https://sourcegraph.com/github.com/obscuronet/go-obscuro/-/blob/integration/simulation/validate_chain.go?L225:20
		Topics: outcomeReportedEventTopics,
	}

	outcomeReportedLogs := make(chan types.Log)
	//TODO: might have to just use FilterLogs instead of ws SubscribeFilterLogs
	outcomeReportedSub, err := client.SubscribeFilterLogs(context.Background(), outcomeReportedQuery, outcomeReportedLogs)
	if err != nil {
		e.logger.Error("error while subscribing to ProposeOutcome logs", "err", err)

		return
	}

	for {
		select {
		case err := <-proposeOutcomeSub.Err():
			e.logger.Error("error listening to ProposeOutcome events", "err", err)
		case err := <-outcomeReportedSub.Err():
			e.logger.Error("error listening to OutocmeReported events", "err", err)
		case vLog := <-proposeOutcomeLogs:
			//TODO: how do we know the event is for ProposeOutcome?
			results, err := contractAbi.Unpack("ProposeOutcome", vLog.Data)

			if err != nil {
				e.logger.Error("error unpacking ProposeOutcome event", "err", err)
			}

			if len(results) == 0 {
				e.logger.Error("unexpected empty results for ProposeOutcome event")
			}

			marketHash, ok := results[0].([32]byte)
			if !ok { // type assertion failed

			}

			outcome, ok := results[1].(int32)
			if !ok { // type assertion failed

			}

			//TODO: get blockTimestamp (vLog.BlockNumber?), outome, marketHash of ProposeOutcome event
			//TODO: add to queue
			e.logger.Debug("received ProposeOutcome event", "marketHash", marketHash, "outcome", outcome)
		case vLog := <-outcomeReportedLogs:
			//TODO: how do we know the event is for ProposeOutcome?
			results, err := contractAbi.Unpack("OutcomeReported", vLog.Data)

			if err != nil {
				e.logger.Error("error unpacking OutcomeReported event", "err", err)
			}

			if len(results) == 0 {
				e.logger.Error("unexpected empty results for OutcomeReported event")
			}

			marketHash, ok := results[0].([32]byte)
			if !ok { // type assertion failed

			}

			outcome, ok := results[1].(int32)
			if !ok { // type assertion failed

			}

			//TODO: get blockTimestamp (vLog.BlockNumber?), outome, marketHash of ProposeOutcome event
			//TODO: remove from queue
			e.logger.Debug("received OutcomeReported event", "marketHash", marketHash, "outcome", outcome)
		}
	}
}
