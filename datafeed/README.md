# DataFeed Service

Decentralized data feed oracle.

Current methods of consumption include:
- MQ
- GRPC Operator (used by validator operators)

## MQ Consumer

To enable the background MQ Consumer, `server` command must provide flags:
- --data-feed-amqp-uri (e.g. amqps://user:pass@domain.com:5671)
- --data-feed-amqp-queue-name (e.g. MyQueue)

For example:
```
./main server --log-level DEBUG --data-dir ./data-dir --chain ./genesis.json --grpc-address localhost:10000 --libp2p localhost:10001 --jsonrpc localhost:10002 --data-feed-amqp-uri  amqps://user:pass@domain.com:5671 --data-feed-amqp-queue-name MyQueue
```

Any new messages in the queue will be consumed and passed to the ProcessPayload() function.

## GRPC Operator

Users can manually interface with their validator to provide custom reporting outcomes for marketHashes, for example:

```
./main datafeed report --market asdf --outcome asdf --grpc-address localhost:10000
```

See `/sx-node/command/datafeed` for GRPC command files and `/sx-node/datafeed/operator.go` for operator.
### Generating protobuf files
```
protoc --go_out=. --go-grpc_out=. ./datafeed/proto/*.proto
```