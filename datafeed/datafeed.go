package datafeed

import (
	"github.com/hashicorp/go-hclog"
)

// DataFeed
type DataFeed struct {
	logger    hclog.Logger
	config    *Config
	mqService *MQService
}

// Config
type Config struct {
	MQConfig *MQConfig
}

type MQConfig struct {
	HostUrl     string
	QueueConfig *QueueConfig
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(logger hclog.Logger, config *Config) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger: logger.Named("datafeed"),
		config: config,
	}

	// configure and start mqService
	mqService, err := newMQService(datafeedService.logger, config.MQConfig, datafeedService)
	if err != nil {
		return nil, err
	}
	datafeedService.mqService = mqService

	//TODO: set up grpc listener

	//TODO: set up libp2p functions

	return datafeedService, nil
}

func (d *DataFeed) ProcessPayload(message string) {
	d.logger.Debug("Processing message", "message", message)

	//TODO: eventually parse lsports payload here + process + sign + gossip
}
