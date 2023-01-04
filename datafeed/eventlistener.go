package datafeed

import (
	"context"
	"encoding/hex"
	"math/big"
	"reflect"
	"strings"

	"github.com/0xPolygon/polygon-edge/contracts/abis"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hashicorp/go-hclog"
)

const (
	JSONRPCWsHost = "ws://localhost:10002/ws"
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

	client, err := ethclient.Dial(JSONRPCWsHost)
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

	e.logger.Debug("Listening for events...")

	for {
		select {
		case err := <-proposeOutcomeSub.Err():
			e.logger.Error("error listening to ProposeOutcome events", "err", err)
		case err := <-outcomeReportedSub.Err():
			e.logger.Error("error listening to OutocmeReported events", "err", err)
		case vLog := <-proposeOutcomeLogs:
			results, err := contractAbi.Unpack("ProposeOutcome", vLog.Data)

			if err != nil {
				e.logger.Error("error unpacking ProposeOutcome event", "err", err)
			}

			if len(results) == 0 {
				e.logger.Error("unexpected empty results for ProposeOutcome event")
			}

			marketHash, ok := results[0].([32]byte)
			if !ok { // type assertion failed
				e.logger.Error("type assertion failed for [32]byte", "marketHash", results[0], "got type", reflect.TypeOf(results[0]).String())
			}

			outcome, ok := results[1].(uint8)
			if !ok { // type assertion failed
				e.logger.Error("type assertion failed for int", "outcome", results[1], "got type", reflect.TypeOf(results[1]).String())
			}

			blockTimestamp, ok := results[2].(*big.Int)
			if !ok { // type assertion failed
				e.logger.Error("type assertion failed for int", "timestamp", results[2], "got type", reflect.TypeOf(results[2]).String())
			}

			// derive blockTimestamp from event's block
			// block, err := e.client.BlockByHash(context.Background(), vLog.BlockHash)
			// if err != nil {
			// 	e.logger.Error("could not get block by hash", "blockHash", vLog.BlockHash)
			// }
			// blockTimestamp := block.Time()

			e.logger.Debug("received ProposeOutcome event", "marketHash", marketHash, "outcome", outcome, "blockTime", blockTimestamp)

			e.datafeedService.voteOutcome(hex.EncodeToString(marketHash[:]), int32(outcome))
			e.datafeedService.addToQueue(hex.EncodeToString(marketHash[:]), uint64(blockTimestamp.Int64()))
		case vLog := <-outcomeReportedLogs:
			results, err := contractAbi.Unpack("OutcomeReported", vLog.Data)

			if err != nil {
				e.logger.Error("error unpacking OutcomeReported event", "err", err)
			}

			if len(results) == 0 {
				e.logger.Error("unexpected empty results for OutcomeReported event")
			}

			marketHash, ok := results[0].([32]byte)
			if !ok { // type assertion failed
				e.logger.Error("type assertion failed for [32]byte", "marketHash", results[0], "got type", reflect.TypeOf(results[0]).String())
			}

			outcome, ok := results[1].(uint8)
			if !ok { // type assertion failed
				e.logger.Error("type assertion failed for int", "outcome", results[1], "got type", reflect.TypeOf(results[1]).String())
			}

			//TODO: get blockTimestamp (vLog.BlockNumber?), outome, marketHash of ProposeOutcome event
			//TODO: remove from queue
			e.logger.Debug("received OutcomeReported event", "marketHash", marketHash, "outcome", outcome)

			e.datafeedService.removeFromQueue(hex.EncodeToString(marketHash[:]))
		}
	}
}
