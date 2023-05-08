package datafeed

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"time"

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
	client          *ethclient.Client
}

func newEventListener(logger hclog.Logger, datafeedService *DataFeed) (*EventListener, error) {

	eventListener := &EventListener{
		logger:          logger.Named("eventListener"),
		datafeedService: datafeedService,
	}

	client, err := ethclient.Dial(JSONRPCWsHost)
	if err != nil {
		logger.Error("error while dialing ws rpc url", "err", err)
		return nil, err
	}
	eventListener.client = client

	go eventListener.startListeningLoop()

	return eventListener, nil
}

func (e EventListener) startListeningLoop() {

	contractAbi, err := abi.JSON(strings.NewReader(abis.OutcomeReporterJSONABI))
	if err != nil {
		e.logger.Error("error while parsing OutcomeReporter contract ABI", "err", err)

		return
	}

	outcomeReporterAddress := common.HexToAddress(e.datafeedService.config.OutcomeReporterAddress)

	proposeOutcomeSub, proposeOutcomeLogs, err := e.subscribeToProposeOutcome(contractAbi, outcomeReporterAddress)
	if err != nil {
		panic(fmt.Errorf("fatal error while subscribing to ProposeOutcome logs: %w", err))
	}

	outcomeReportedSub, outcomeReportedLogs, err := e.subscribeToOutcomeReported(contractAbi, outcomeReporterAddress)
	if err != nil {
		panic(fmt.Errorf("fatal error while subscribing to OutcomeReported logs: %w", err))
	}

	e.logger.Debug("Listening for events...")

	for {
		select {
		case err := <-proposeOutcomeSub.Err():
			e.logger.Error("error listening to ProposeOutcome events, re-connecting after 5 seconds..", "err", err)
			time.Sleep(5 * time.Second)
			proposeOutcomeSub, proposeOutcomeLogs, err = e.subscribeToProposeOutcome(contractAbi, outcomeReporterAddress)
			if err != nil {
				panic(fmt.Errorf("fatal error while re-subscribing to ProposeOutcome logs: %w", err))
			}

		case err := <-outcomeReportedSub.Err():
			e.logger.Error("error listening to OutcomeReported events, re-connecting after 5 seconds..", "err", err)

			time.Sleep(5 * time.Second)
			outcomeReportedSub, outcomeReportedLogs, err = e.subscribeToOutcomeReported(contractAbi, outcomeReporterAddress)
			if err != nil {
				panic(fmt.Errorf("fatal error while re-subscribing to OutcomeReported logs: %w", err))
			}

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

			marketHashStr := fmt.Sprintf("0x%s", hex.EncodeToString(marketHash[:]))
			e.logger.Debug("received ProposeOutcome event", "marketHash", marketHashStr, "outcome", outcome, "blockTime", blockTimestamp)

			e.datafeedService.voteOutcome(marketHashStr)
			e.datafeedService.addToStore(marketHashStr, uint64(blockTimestamp.Int64()))
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

			marketHashStr := fmt.Sprintf("0x%s", hex.EncodeToString(marketHash[:]))
			e.logger.Debug("received OutcomeReported event", "marketHash", marketHashStr, "outcome", outcome)

			e.datafeedService.removeFromStore(marketHashStr)
		}
	}
}

func (e EventListener) subscribeToProposeOutcome(contractAbi abi.ABI, outcomeReporterAddress common.Address) (ethereum.Subscription, <-chan types.Log, error) {
	proposeOutcomeEvent := contractAbi.Events["ProposeOutcome"].ID

	var proposeOutcomeEventTopics [][]common.Hash

	proposeOutcomeEventTopics = append(proposeOutcomeEventTopics, []common.Hash{proposeOutcomeEvent})

	proposeOutcomeQuery := ethereum.FilterQuery{
		Addresses: []common.Address{outcomeReporterAddress},
		Topics:    proposeOutcomeEventTopics,
	}

	proposeOutcomeLogs := make(chan types.Log)
	proposeOutcomeSub, err := e.client.SubscribeFilterLogs(context.Background(), proposeOutcomeQuery, proposeOutcomeLogs)
	if err != nil {
		e.logger.Error("error in SubscribeFilterLogs call", "err", err)

		return nil, nil, err
	}

	return proposeOutcomeSub, proposeOutcomeLogs, nil
}

func (e EventListener) subscribeToOutcomeReported(contractAbi abi.ABI, outcomeReporterAddress common.Address) (ethereum.Subscription, <-chan types.Log, error) {
	outcomeReportedEvent := contractAbi.Events["OutcomeReported"].ID

	var outcomeReportedEventTopics [][]common.Hash
	outcomeReportedEventTopics = append(outcomeReportedEventTopics, []common.Hash{outcomeReportedEvent})

	outcomeReportedQuery := ethereum.FilterQuery{
		Addresses: []common.Address{outcomeReporterAddress},
		Topics:    outcomeReportedEventTopics,
	}

	outcomeReportedLogs := make(chan types.Log)
	outcomeReportedSub, err := e.client.SubscribeFilterLogs(context.Background(), outcomeReportedQuery, outcomeReportedLogs)
	if err != nil {
		e.logger.Error("error in SubscribeFilterLogs call", "err", err)

		return nil, nil, err
	}

	return outcomeReportedSub, outcomeReportedLogs, nil
}
