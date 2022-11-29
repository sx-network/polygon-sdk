package datafeed

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	mqConsumerConcurrency = 1
)

// MQService
type MQService struct {
	logger          hclog.Logger
	config          *MQConfig
	connection      Connection
	datafeedService *DataFeed
}

// Connection
type Connection struct {
	Channel *amqp.Channel
}

type MQConfig struct {
	AMQPURI      string
	ExchangeName string
	QueueConfig  *QueueConfig
}

// QueueConfig
type QueueConfig struct {
	QueueName string
}

func newMQService(logger hclog.Logger, config *MQConfig, datafeedService *DataFeed) (*MQService, error) {
	conn, err := getConnection(
		config.AMQPURI,
	)
	if err != nil {
		return nil, err
	}

	mq := &MQService{
		logger:          logger.Named("mq"),
		config:          config,
		connection:      conn,
		datafeedService: datafeedService,
	}

	go mq.startConsumeLoop()

	return mq, nil
}

// startConsumeLoop
func (mq *MQService) startConsumeLoop() {
	mq.logger.Debug("listening for MQ messages...")

	ctx, cfunc := context.WithCancel(context.Background())

	reports, errors, err := mq.startConsumer(ctx, mqConsumerConcurrency)

	if err != nil {
		panic(err)
	}

	for {
		select {
		case report := <-reports:
			mq.datafeedService.addNewReport(report)
		case err = <-errors:
			mq.logger.Error("error while consuming from message queue", "err", err)
		case <-common.GetTerminationSignalCh():
			cfunc()
		}
	}
}

// getConnection establishes connection via TCP on provided rabbitMQURL (AMQP URI) and returns Connection and Channel
func getConnection(rabbitMQURL string) (Connection, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return Connection{}, err
	}

	ch, err := conn.Channel()

	return Connection{
		Channel: ch,
	}, err
}

// startConsumer start consuming queued messages, receiving deliveries on the 'deliveries' channel.
// returns parsed deliveries within reports channel and any errors if they occurred within errors channel.
func (mq *MQService) startConsumer(
	ctx context.Context, concurrency int,
) (<-chan *proto.DataFeedReport, <-chan error, error) {
	mq.logger.Debug("Starting MQConsumerService...")

	// create the queue if it doesn't already exist
	_, err := mq.connection.Channel.QueueDeclare(mq.config.QueueConfig.QueueName, true, false, false, false, nil)
	if err != nil {
		return nil, nil, err
	}

	// bind the queue to the routing key
	err = mq.connection.Channel.QueueBind(mq.config.QueueConfig.QueueName, "", mq.config.ExchangeName, false, nil)
	if err != nil {
		return nil, nil, err
	}

	// prefetch 4x as many messages as we can handle at once
	prefetchCount := concurrency * 4

	err = mq.connection.Channel.Qos(prefetchCount, 0, false)
	if err != nil {
		return nil, nil, err
	}

	uuid := uuid.New().String()
	deliveries, err := mq.connection.Channel.Consume(
		mq.config.QueueConfig.QueueName, // queue
		uuid,                            // consumer
		false,                           // auto-ack
		false,                           // exclusive
		false,                           // no-local
		false,                           // no-wait
		nil,                             // args
	)

	if err != nil {
		return nil, nil, err
	}

	reports := make(chan *proto.DataFeedReport)
	errors := make(chan error)

	for i := 0; i < concurrency; i++ {
		go func() {
			for delivery := range deliveries {
				report, err := mq.parseDelivery(delivery)
				if err != nil {
					errors <- err
					//delivery.Nack(false, true) //nolint:errcheck
					// nacking will avoid removing from queue, so we ack even so we've encountered an error
					delivery.Ack(false) //nolint:errcheck
				} else {
					delivery.Ack(false) //nolint:errcheck
					reports <- report
				}
			}
		}()
	}

	// stop the consumer upon sigterm
	go func() {
		<-ctx.Done()
		// stop consumer quickly
		mq.connection.Channel.Cancel(uuid, false) //nolint:errcheck
	}()

	return reports, errors, nil
}

// parseDelivery returns unmarshalled report or error if one occurred during parsing
func (mq *MQService) parseDelivery(delivery amqp.Delivery) (*proto.DataFeedReport, error) {
	if delivery.Body == nil {
		return &proto.DataFeedReport{}, fmt.Errorf("no message body")
	}

	var reportOutcome proto.DataFeedReport
	if err := json.Unmarshal(delivery.Body, &reportOutcome); err != nil {
		return &proto.DataFeedReport{}, fmt.Errorf("error during report outcome json unmarshaling, %w", err)
	}

	mq.logger.Debug("MQ message received", "marketHash", reportOutcome.MarketHash)

	return &reportOutcome, nil
}
