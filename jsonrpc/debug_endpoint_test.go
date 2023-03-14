package jsonrpc

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/chain"
	"github.com/0xPolygon/polygon-edge/helper/progress"
	"github.com/0xPolygon/polygon-edge/state/runtime"
	"github.com/0xPolygon/polygon-edge/state/runtime/tracer"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/stretchr/testify/assert"
)

type debugEndpointMockStore struct {
	headerFn            func() *types.Header
	getHeaderByNumberFn func(uint64) (*types.Header, bool)
	readTxLookupFn      func(types.Hash) (types.Hash, bool)
	getBlockByHashFn    func(types.Hash, bool) (*types.Block, bool)
	getBlockByNumberFn  func(uint64, bool) (*types.Block, bool)
	traceBlockFn        func(*types.Block, tracer.Tracer) ([]interface{}, error)
	traceTxnFn          func(*types.Block, types.Hash, tracer.Tracer) (interface{}, error)
	traceCallFn         func(*types.Transaction, *types.Header, tracer.Tracer) (interface{}, error)
	getNonceFn          func(types.Address) uint64
	getAccountFn        func(types.Hash, types.Address) (*Account, error)
	addTx               func(tx *types.Transaction) error
	applyBlockTxn       func(parentHeader *types.Header, block *types.Block, hash types.Hash, tracerConfig runtime.TraceConfig) (result *runtime.ExecutionResult, err error)
	getAvgGasPrice      func() *big.Int
	applyTxn            func(header *types.Header, txn *types.Transaction) (*runtime.ExecutionResult, error)
	applyMessage        func(parentHeader *types.Header, header *types.Header, txn *types.Transaction, tracer runtime.TraceConfig) (*runtime.ExecutionResult, error)
	getCode             func(root types.Hash, addr types.Address) ([]byte, error)
	// getAccount          func(root types.Hash, addr types.Address) (*Account, error)
	getStorage         func(root types.Hash, addr types.Address, slot types.Hash) ([]byte, error)
	getForksInTime     func(blockNumber uint64) chain.ForksInTime
	getPendingTx       func(txHash types.Hash) (*types.Transaction, bool)
	getReceiptsByHash  func(hash types.Hash) ([]*types.Receipt, error)
	getSyncProgression func() *progress.Progression
	issIbftStateStale  func() bool
}

// AddTx(tx *types.Transaction) error

// 	// GetPendingTx gets the pending transaction from the transaction pool, if it's present
// 	GetPendingTx(txHash types.Hash) (*types.Transaction, bool)

// 	// GetNonce returns the next nonce for this address
// 	GetNonce(addr types.Address) uint64

func (m *debugEndpointMockStore) IsIbftStateStale() bool {
	return m.issIbftStateStale()
}

func (m *debugEndpointMockStore) GetSyncProgression() *progress.Progression {
	return m.getSyncProgression()
}

func (m *debugEndpointMockStore) GetReceiptsByHash(hash types.Hash) ([]*types.Receipt, error) {

	return m.getReceiptsByHash(hash)
}

func (m *debugEndpointMockStore) AddTx(tx *types.Transaction) error {

	return m.addTx(tx)
}

func (m *debugEndpointMockStore) ApplyBlockTxn(parentHeader *types.Header,
	block *types.Block,
	hash types.Hash,
	tracerConfig runtime.TraceConfig) (result *runtime.ExecutionResult, err error) {
	return m.applyBlockTxn(parentHeader, block, hash, tracerConfig)
}

func (s *debugEndpointMockStore) ApplyMessage(
	parentHeader *types.Header,
	header *types.Header,
	txn *types.Transaction,
	tracerConfig runtime.TraceConfig,
) (result *runtime.ExecutionResult, err error) {
	return s.applyMessage(parentHeader, header, txn, tracerConfig)
}

func (s *debugEndpointMockStore) ApplyTxn(
	header *types.Header,
	txn *types.Transaction,
) (result *runtime.ExecutionResult, err error) {

	return s.applyTxn(header, txn)
}

func (m *debugEndpointMockStore) GetAvgGasPrice() *big.Int {
	return m.getAvgGasPrice()
}

func (m *debugEndpointMockStore) GetCode(root types.Hash, addr types.Address) ([]byte, error) {
	return m.getCode(root, addr)
}

// func (m *debugEndpointMockStore) GetAccount(root types.Hash, addr types.Address) (*Account, error) {}

func (m *debugEndpointMockStore) GetStorage(root types.Hash, addr types.Address, slot types.Hash) ([]byte, error) {
	return m.getStorage(root, addr, slot)
}
func (m *debugEndpointMockStore) GetForksInTime(blockNumber uint64) chain.ForksInTime {
	return m.getForksInTime(blockNumber)
}

func (m *debugEndpointMockStore) GetPendingTx(txHash types.Hash) (*types.Transaction, bool) {
	return m.getPendingTx(txHash)
}

func (s *debugEndpointMockStore) Header() *types.Header {
	return s.headerFn()
}

func (s *debugEndpointMockStore) GetHeaderByNumber(num uint64) (*types.Header, bool) {
	return s.getHeaderByNumberFn(num)
}

func (s *debugEndpointMockStore) ReadTxLookup(txnHash types.Hash) (types.Hash, bool) {
	return s.readTxLookupFn(txnHash)
}

func (s *debugEndpointMockStore) GetBlockByHash(hash types.Hash, full bool) (*types.Block, bool) {
	return s.getBlockByHashFn(hash, full)
}

func (s *debugEndpointMockStore) GetBlockByNumber(num uint64, full bool) (*types.Block, bool) {
	return s.getBlockByNumberFn(num, full)
}

func (s *debugEndpointMockStore) TraceBlock(block *types.Block, tracer tracer.Tracer) ([]interface{}, error) {
	return s.traceBlockFn(block, tracer)
}

func (s *debugEndpointMockStore) TraceTxn(block *types.Block, targetTx types.Hash, tracer tracer.Tracer) (interface{}, error) {
	return s.traceTxnFn(block, targetTx, tracer)
}

func (s *debugEndpointMockStore) TraceCall(tx *types.Transaction, parent *types.Header, tracer tracer.Tracer) (interface{}, error) {
	return s.traceCallFn(tx, parent, tracer)
}

func (s *debugEndpointMockStore) GetNonce(acc types.Address) uint64 {
	return s.getNonceFn(acc)
}

func (s *debugEndpointMockStore) GetAccount(root types.Hash, addr types.Address) (*Account, error) {
	return s.getAccountFn(root, addr)
}

func TestDebugTraceConfigDecode(t *testing.T) {
	name := "callTracer"
	timeout15s := "15"
	tests := []struct {
		input    string
		expected TraceConfig
	}{
		{
			// default
			input:    `{}`,
			expected: TraceConfig{},
		},
		{
			input: `{
				"Tracer": "callTracer",
				"Timeout": "15"
			}`,
			expected: TraceConfig{
				Tracer:  &name,
				Timeout: &timeout15s,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := TraceConfig{}

			assert.NoError(
				t,
				json.Unmarshal(
					[]byte(test.input),
					&result,
				),
			)

			assert.Equal(
				t,
				test.expected,
				result,
			)
		})
	}
}

// func TestTraceTransaction(t *testing.T) {
// 	t.Parallel()

// 	blockWithTx := &types.Block{
// 		Header: testBlock10.Header,
// 		Transactions: []*types.Transaction{
// 			testTx1,
// 		},
// 	}

// 	tests := []struct {
// 		name   string
// 		txHash types.Hash
// 		config *TraceConfig
// 		store  *debugEndpointMockStore
// 		result interface{}
// 		err    bool
// 	}{
// 		{
// 			name:   "should trace the given transaction",
// 			txHash: testTxHash1,
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				readTxLookupFn: func(hash types.Hash) (types.Hash, bool) {
// 					assert.Equal(t, testTxHash1, hash)

// 					return testBlock10.Hash(), true
// 				},
// 				getBlockByHashFn: func(hash types.Hash, full bool) (*types.Block, bool) {
// 					assert.Equal(t, testBlock10.Hash(), hash)
// 					assert.True(t, full)

// 					return blockWithTx, true
// 				},
// 				traceTxnFn: func(block *types.Block, txHash types.Hash, tracer tracer.Tracer) (interface{}, error) {
// 					assert.Equal(t, blockWithTx, block)
// 					assert.Equal(t, testTxHash1, txHash)

// 					return testTraceResult, nil
// 				},
// 			},
// 			result: testTraceResult,
// 			err:    false,
// 		},
// 		{
// 			name:   "should return error if ReadTxLookup returns null",
// 			txHash: testTxHash1,
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				readTxLookupFn: func(hash types.Hash) (types.Hash, bool) {
// 					assert.Equal(t, testTxHash1, hash)

// 					return types.ZeroHash, false
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 		{
// 			name:   "should return error if block not found",
// 			txHash: testTxHash1,
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				readTxLookupFn: func(hash types.Hash) (types.Hash, bool) {
// 					assert.Equal(t, testTxHash1, hash)

// 					return testBlock10.Hash(), true
// 				},
// 				getBlockByHashFn: func(hash types.Hash, full bool) (*types.Block, bool) {
// 					assert.Equal(t, testBlock10.Hash(), hash)
// 					assert.True(t, full)

// 					return nil, false
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 		{
// 			name:   "should return error if the tx is not including the block",
// 			txHash: testTxHash1,
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				readTxLookupFn: func(hash types.Hash) (types.Hash, bool) {
// 					assert.Equal(t, testTxHash1, hash)

// 					return testBlock10.Hash(), true
// 				},
// 				getBlockByHashFn: func(hash types.Hash, full bool) (*types.Block, bool) {
// 					assert.Equal(t, testBlock10.Hash(), hash)
// 					assert.True(t, full)

// 					return testBlock10, true
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 		{
// 			name:   "should return error if the block is genesis",
// 			txHash: testTxHash1,
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				readTxLookupFn: func(hash types.Hash) (types.Hash, bool) {
// 					assert.Equal(t, testTxHash1, hash)

// 					return testBlock10.Hash(), true
// 				},
// 				getBlockByHashFn: func(hash types.Hash, full bool) (*types.Block, bool) {
// 					assert.Equal(t, testBlock10.Hash(), hash)
// 					assert.True(t, full)

// 					return &types.Block{
// 						Header: testGenesisHeader,
// 						Transactions: []*types.Transaction{
// 							testTx1,
// 						},
// 					}, true
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 	}

// 	for _, test := range tests {
// 		test := test

// 		t.Run(test.name, func(t *testing.T) {
// 			t.Parallel()

// 			endpoint := &Debug{test.store}

// 			res, err := endpoint.TraceTransaction(test.txHash, test.config)

// 			assert.Equal(t, test.result, res)

// 			if test.err {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// func TestTraceCall(t *testing.T) {
// 	t.Parallel()

// 	var (
// 		from     = types.StringToAddress("1")
// 		to       = types.StringToAddress("2")
// 		gas      = argUint64(10000)
// 		gasPrice = argBytes(new(big.Int).SetUint64(10).Bytes())
// 		value    = argBytes(new(big.Int).SetUint64(1000).Bytes())
// 		data     = argBytes([]byte("data"))
// 		input    = argBytes([]byte("input"))
// 		nonce    = argUint64(1)

// 		blockNumber = BlockNumber(testBlock10.Number())

// 		txArg = &txnArgs{
// 			From:     &from,
// 			To:       &to,
// 			Gas:      &gas,
// 			GasPrice: &gasPrice,
// 			Value:    &value,
// 			Data:     &data,
// 			Input:    &input,
// 			Nonce:    &nonce,
// 		}
// 		decodedTx = &types.Transaction{
// 			Nonce:    uint64(nonce),
// 			GasPrice: new(big.Int).SetBytes([]byte(gasPrice)),
// 			Gas:      uint64(gas),
// 			To:       &to,
// 			Value:    new(big.Int).SetBytes([]byte(value)),
// 			Input:    data,
// 			From:     from,
// 		}
// 	)

// 	decodedTx.ComputeHash()

// 	tests := []struct {
// 		name   string
// 		arg    *txnArgs
// 		filter BlockNumberOrHash
// 		config *TraceConfig
// 		store  *debugEndpointMockStore
// 		result interface{}
// 		err    bool
// 	}{
// 		{
// 			name: "should trace the given transaction",
// 			arg:  txArg,
// 			filter: BlockNumberOrHash{
// 				BlockNumber: &blockNumber,
// 			},
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				getHeaderByNumberFn: func(num uint64) (*types.Header, bool) {
// 					assert.Equal(t, testBlock10.Number(), num)

// 					return testHeader10, true
// 				},
// 				traceCallFn: func(tx *types.Transaction, header *types.Header, tracer tracer.Tracer) (interface{}, error) {
// 					assert.Equal(t, decodedTx, tx)
// 					assert.Equal(t, testHeader10, header)

// 					return testTraceResult, nil
// 				},
// 			},
// 			result: testTraceResult,
// 			err:    false,
// 		},
// 		{
// 			name: "should return error if block not found",
// 			arg:  txArg,
// 			filter: BlockNumberOrHash{
// 				BlockHash: &testHeader10.Hash,
// 			},
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				getBlockByHashFn: func(hash types.Hash, full bool) (*types.Block, bool) {
// 					assert.Equal(t, testHeader10.Hash, hash)
// 					assert.False(t, full)

// 					return nil, false
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 		{
// 			name: "should return error if decoding transaction fails",
// 			arg: &txnArgs{
// 				From:     &from,
// 				Gas:      &gas,
// 				GasPrice: &gasPrice,
// 				Value:    &value,
// 				Nonce:    &nonce,
// 			},
// 			filter: BlockNumberOrHash{},
// 			config: &TraceConfig{},
// 			store: &debugEndpointMockStore{
// 				headerFn: func() *types.Header {
// 					return testLatestHeader
// 				},
// 			},
// 			result: nil,
// 			err:    true,
// 		},
// 	}

// 	for _, test := range tests {
// 		test := test

// 		t.Run(test.name, func(t *testing.T) {
// 			t.Parallel()

// 			endpoint := &Debug{test.store}

// 			res, err := endpoint.TraceCall(test.arg, test.filter, test.config)

// 			assert.Equal(t, test.result, res)

// 			if test.err {
// 				assert.Error(t, err)
// 			} else {
// 				assert.NoError(t, err)
// 			}
// 		})
// 	}
// }

// func Test_newTracer(t *testing.T) {
// 	t.Parallel()

// 	t.Run("should create tracer", func(t *testing.T) {
// 		t.Parallel()

// 		tracer, cancel, err := newTracer(&TraceConfig{
// 			EnableMemory:     true,
// 			EnableReturnData: true,
// 			DisableStack:     false,
// 			DisableStorage:   false,
// 		})

// 		t.Cleanup(func() {
// 			cancel()
// 		})

// 		assert.NotNil(t, tracer)
// 		assert.NoError(t, err)
// 	})

// 	t.Run("GetResult should return errExecutionTimeout if timeout happens", func(t *testing.T) {
// 		t.Parallel()

// 		timeout := "0s"
// 		tracer, cancel, err := newTracer(&TraceConfig{
// 			EnableMemory:     true,
// 			EnableReturnData: true,
// 			DisableStack:     false,
// 			DisableStorage:   false,
// 			Timeout:          &timeout,
// 		})

// 		t.Cleanup(func() {
// 			cancel()
// 		})

// 		assert.NoError(t, err)

// 		// wait until timeout
// 		time.Sleep(100 * time.Millisecond)

// 		res, err := tracer.GetResult()
// 		assert.Nil(t, res)
// 		assert.Equal(t, ErrExecutionTimeout, err)
// 	})

// 	t.Run("GetResult should not return if cancel is called beforre timeout", func(t *testing.T) {
// 		t.Parallel()

// 		timeout := "5s"
// 		tracer, cancel, err := newTracer(&TraceConfig{
// 			EnableMemory:     true,
// 			EnableReturnData: true,
// 			DisableStack:     false,
// 			DisableStorage:   false,
// 			Timeout:          &timeout,
// 		})

// 		assert.NoError(t, err)

// 		cancel()

// 		res, err := tracer.GetResult()

// 		assert.NotNil(t, res)
// 		assert.NoError(t, err)
// 	})
// }
