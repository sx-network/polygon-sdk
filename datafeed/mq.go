package datafeed

import (
	"context"
	"fmt"

	"github.com/0xPolygon/polygon-edge/helper/common"
	"github.com/google/uuid"
	"github.com/hashicorp/go-hclog"
	amqp "github.com/rabbitmq/amqp091-go"
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
	messages, errors, err := mq.startConsumer(ctx, 1)

	if err != nil {
		panic(err)
	}

	for {
		select {
		case message := <-messages:
			mq.datafeedService.ProcessPayload(message)
		case err = <-errors:
			mq.logger.Error("got error while receiving event", "err", err)
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
// returns parsed deliveries within messages channel and any errors if they occurred within errors channel.
func (mq *MQService) startConsumer(
	ctx context.Context, concurrency int,
) (<-chan string, <-chan error, error) {
	mq.logger.Debug("Starting MQConsumerService...")
	// bind the queue to the routing key - optional
	// err = ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
	// if err != nil {
	// 	return err
	// }

	// prefetch 4x as many messages as we can handle at once
	prefetchCount := concurrency * 4

	err := mq.connection.Channel.Qos(prefetchCount, 0, false)
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

	messages := make(chan string)
	errors := make(chan error)

	for i := 0; i < concurrency; i++ {
		go func() {
			for delivery := range deliveries {
				message, err := mq.parseDelivery(delivery)
				if err != nil {
					errors <- err

					delivery.Nack(false, true) //nolint:errcheck
				} else {
					delivery.Ack(false) //nolint:errcheck
					messages <- message
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

	return messages, errors, nil
}

// parseDelivery returns body of message or error if one occurred during parsing
func (mq *MQService) parseDelivery(delivery amqp.Delivery) (string, error) {
	if delivery.Body == nil {
		err := fmt.Errorf("error, no message body")

		return "", err
	}

	body := string(delivery.Body)
	mq.logger.Debug("MQ message received", "message", body)

	return body, nil
}
