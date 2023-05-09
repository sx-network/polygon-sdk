package datafeed

import (
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

//TODO: we should add a new queue struct similar to txpool's queue promoteRequest
//TODO: pop() should take off oldest event
//TODO: peek() should return oldest
//TODO: push() should add event

// StoreProcessor
type StoreProcessor struct {
	logger          hclog.Logger
	datafeedService *DataFeed
	store           *MarketItemStore
}

// MarketItemStore holds markets to be reported on after voting period
type MarketItemStore struct {
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

	go storeProcessor.startProcessingLoop()

	return storeProcessor, nil
}

func (s *StoreProcessor) startProcessingLoop() {

	for {
		time.Sleep(5 * time.Second)
		for marketHash, timestamp := range s.store.marketItems {
			if timestamp+s.datafeedService.config.OutcomeVotingPeriodSeconds <= uint64(time.Now().Unix()) {
				s.logger.Debug(
					"processing market item",
					"market", marketHash,
					"block ts", timestamp,
					"current ts", time.Now().Unix())
				s.datafeedService.reportOutcome(marketHash)
				s.store.remove(marketHash)
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

// addToStore adds a new market item specified by marketHash, outcome, timestamp to store to be processed later
func (d *DataFeed) addToStore(marketHash string, blockTimestamp uint64) {
	d.storeProcessor.store.add(marketHash, blockTimestamp)
	d.logger.Debug("added to store", "market", marketHash, "blockTimestamp", blockTimestamp)
}

// removeFromStore removes the market item from the store
func (d *DataFeed) removeFromStore(marketHash string) {
	d.storeProcessor.store.remove(marketHash)
	d.logger.Debug("removed from store", "market", marketHash)
}

func (m *MarketItemStore) add(marketHash string, blockTimestamp uint64) {
	m.Lock()
	defer m.Unlock()

	m.marketItems[marketHash] = blockTimestamp
}

func (m *MarketItemStore) remove(marketHash string) {
	m.Lock()
	defer m.Unlock()

	delete(m.marketItems, marketHash)
}
