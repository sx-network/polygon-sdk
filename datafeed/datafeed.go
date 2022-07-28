package datafeed

import (
	"fmt"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
)

// DataFeed
type DataFeed struct {
	logger    hclog.Logger
	config    *Config
	mqService *MQService

	// indicates which DataFeed operator commands should be implemented
	proto.UnimplementedDataFeedOperatorServer
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
func NewDataFeedService(logger hclog.Logger, config *Config, grpcServer *grpc.Server) (*DataFeed, error) {
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

	// configure grpc operator service
	if grpcServer != nil {
		proto.RegisterDataFeedOperatorServer(grpcServer, datafeedService)
	}

	//TODO: set up libp2p functions
	return datafeedService, nil
}

func (d *DataFeed) ProcessPayload(message string) {
	//TODO: strings for now but eventually parse lsports payload here + process + sign + gossip
	//TODO: should call libp2p publish here after we validate the payload
	d.logger.Debug("Processing message", "message", message)
}
