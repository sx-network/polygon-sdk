package datafeed

import (
	"fmt"

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
	AMQPURI     string
	QueueConfig *QueueConfig
}

// NewDataFeedService returns the new datafeed service
func NewDataFeedService(logger hclog.Logger, config *Config) (*DataFeed, error) {
	datafeedService := &DataFeed{
		logger: logger.Named("datafeed"),
		config: config,
	}

	// configure and start mqService
	if config.MQConfig.AMQPURI != "" {
		if config.MQConfig.QueueConfig.QueueName == "" {
			return nil, fmt.Errorf("DataFeed AMQPURI provided without a valid QueueName")
		}

		mqService, err := newMQService(datafeedService.logger, config.MQConfig, datafeedService)
		if err != nil {
			return nil, err
		}

		datafeedService.mqService = mqService
	}
	//TODO: set up grpc listener
	//TODO: set up libp2p functions
	return datafeedService, nil
}

func (d *DataFeed) ProcessPayload(message string) {
	//TODO: eventually parse lsports payload here + process + sign + gossip
	d.logger.Debug("Processing message", "message", message)
}
