package datafeed

//TODO: we should add a new queue struct similar to txpool's queue promoteRequest
//TODO: pop() should take off oldest event
//TODO: peek() should return
//TODO: push() should add event

// addToQueue adds a new market item specified by marketHash, outcome, timestamp to queue to be processed later
func (d *DataFeed) addToQueue(marketHash string, blockTime uint64) {
	d.logger.Debug("added to queue", "market", marketHash, "blockTime", blockTime)
}

// removeFromQueue removes the market item from the queue
func (d *DataFeed) removeFromQueue(marketHash string) {
	d.logger.Debug("removed from queue", "market", marketHash)
}
