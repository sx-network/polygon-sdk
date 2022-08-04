package datafeed

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
)

func TestNewDataFeedService(t *testing.T) {
	datafeed, err := newTestDataFeedService("", "")
	if err != nil {
		t.Fatalf("cannot create datafeed - err: %v\n", err)
	}

	assert.True(t, datafeed.mqService == nil, "mqService should be nil")

	_, err = newTestDataFeedService("something", "")

	assert.True(t, err.Error() == "DataFeed AMQPURI provided without a valid QueueName", "expected error")
}

// TODO: test validateSignatures()
// TODO: test validateTimestamp()
// TODO: test getSignatureForPayload()

func newTestDataFeedService(amqpURI, queueName string) (*DataFeed, error) {
	return NewDataFeedService(
		hclog.NewNullLogger(),
		&Config{
			MQConfig: &MQConfig{
				AMQPURI: amqpURI,
				QueueConfig: &QueueConfig{
					QueueName: queueName,
				},
			},
		},
		nil,
		nil,
		nil,
	)
}
