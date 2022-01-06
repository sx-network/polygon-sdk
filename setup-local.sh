rm -rf genesis.json
rm -rf validator-1
rm -rf validator-2
rm -rf validator-3
rm -rf validator-4
rm -rf node-5
VAL_ONE=$(./main secrets init --data-dir validator-1 | grep -o '\b16\w*')
VAL_TWO=$(./main secrets init --data-dir validator-2 | grep -o '\b16\w*')
VAL_THREE=$(./main secrets init --data-dir validator-3 | grep -o '\b16\w*')
VAL_FOUR=$(./main secrets init --data-dir validator-4 | grep -o '\b16\w*')
VAL_FIVE=$(./main secrets init --data-dir node-5 | grep -o '\b16\w*')
./main genesis --premine 0x92235A23306Cd342582c80897aD9Cd16edC9a69a:0x33b2e3c9fd0803ce8000000 --consensus ibft --ibft-validators-prefix-path validator- --bootnode /ip4/127.0.0.1/tcp/10001/p2p/$VAL_ONE --bootnode /ip4/127.0.0.1/tcp/10001/p2p/$VAL_TWO --bootnode /ip4/127.0.0.1/tcp/10001/p2p/$VAL_THREE --bootnode /ip4/127.0.0.1/tcp/10001/p2p/$VAL_FOUR
cp genesis.json validator-1
cp genesis.json validator-2
cp genesis.json validator-3
cp genesis.json validator-4
cp genesis.json node-5

#./main server --log-level DEBUG --data-dir /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-1 --chain /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-1/genesis.json --grpc localhost:10000 -libp2p localhost:10001 --jsonrpc localhost:10002 --seal
#./main server --log-level DEBUG --data-dir /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-2 --chain /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-2/genesis.json --grpc localhost:20000 -libp2p localhost:20001 --jsonrpc localhost:20002 --seal
#./main server --log-level DEBUG --data-dir /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-3 --chain /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-3/genesis.json --grpc localhost:30000 -libp2p localhost:30001 --jsonrpc localhost:30002 --seal
#./main server --log-level DEBUG --data-dir /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-4 --chain /Users/dan/Documents/dev/nextgenbt/repos/sx-node/validator-4/genesis.json --grpc localhost:40000 -libp2p localhost:40001 --jsonrpc localhost:40002 --seal
#./main server --log-level DEBUG --data-dir /Users/dan/Documents/dev/nextgenbt/repos/sx-node/node-5 --chain /Users/dan/Documents/dev/nextgenbt/repos/sx-node/node-5/genesis.json --grpc localhost:50000 -libp2p localhost:50001 --jsonrpc localhost:50002