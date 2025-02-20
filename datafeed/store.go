package datafeed

import (
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

// StoreProcessor
type StoreProcessor struct {
	logger          hclog.Logger
	datafeedService *DataFeed
	store           *MarketItemStore
}

// MarketItemStore holds markets to be reported on after voting period
type MarketItemStore struct {
	logger      hclog.Logger
	marketItems map[string]uint64
	sync.Mutex
}

func newStoreProcessor(logger hclog.Logger, datafeedService *DataFeed) (*StoreProcessor, error) {
	storeProcessor := &StoreProcessor{
		logger:          logger.Named("storeProcessor"),
		datafeedService: datafeedService,
		store: &MarketItemStore{
			marketItems: make(map[string]uint64),
		},
	}
	storeProcessor.store.logger = storeProcessor.logger.Named("store")

	go storeProcessor.startProcessingLoop()

	return storeProcessor, nil
}

func (s *StoreProcessor) startProcessingLoop() {

	for {
		time.Sleep(5 * time.Second)
		for marketHash, timestamp := range s.store.marketItems {
			s.logger.Debug("current outcome voting period seconds", s.datafeedService.config.OutcomeVotingPeriodSeconds)
			if timestamp+s.datafeedService.config.OutcomeVotingPeriodSeconds <= uint64(time.Now().Unix()) {
				s.logger.Debug(
					"processing market item",
					"market", marketHash,
					"block ts", timestamp,
					"current ts", time.Now().Unix())
				s.datafeedService.queueReportingTx(ReportOutcome, marketHash, -1)
			} else {
				s.logger.Debug(
					"market item not yet ready for processing",
					"market", marketHash,
					"block ts", timestamp,
					"current ts", time.Now().Unix(),
					"remaining s", timestamp+s.datafeedService.config.OutcomeVotingPeriodSeconds-uint64(time.Now().Unix()))
				continue
			}
		}
	}
}

// add adds a new market item specified by marketHash, outcome, timestamp to store to be processed later
func (m *MarketItemStore) add(marketHash string, blockTimestamp uint64) {
	m.Lock()
	defer m.Unlock()

	m.marketItems[marketHash] = blockTimestamp
	m.logger.Debug("added to store", "market", marketHash, "blockTimestamp", blockTimestamp)
}

// remove removes the market item from the store
func (m *MarketItemStore) remove(marketHash string) {
	m.Lock()
	defer m.Unlock()

	delete(m.marketItems, marketHash)
	m.logger.Debug("removed from store", "market", marketHash)
}
