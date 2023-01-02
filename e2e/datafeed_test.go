package e2e

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/0xPolygon/polygon-edge/consensus"
	cryptoutils "github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/datafeed"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/0xPolygon/polygon-edge/types"
	"github.com/0xPolygon/polygon-edge/validators"
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/hashicorp/go-hclog"
	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/abi"
	"github.com/umbracle/ethgo/contract"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/umbracle/ethgo/wallet"
)

// tests invoking reportOutcome() function on SC
func TestReportOutcome(t *testing.T) {
	jsonRPCURL := "http://34.225.14.139:10002"                                  //"http://127.0.0.1:8545"
	pk1 := "0x1cda74434f94025b01c74c34a1e913d07de4b7e653a9c534da1f6b1f1b97686f" // validator-1
	pk2 := "0x91abf5c93aada2af7b98ac3cccbcbc8e6b7cc2ad4b5540923ace3418eb76ac62" // validator-2
	pk3 := "0x5ec98cbbf3bdd1c175a12a9b3f91f10171712a236ae5004c8306da394bbe416a" // validator-3
	pk4 := "0x021dda5e6919eb47d633dd790578be4b0059ed73318a65e2bf333f3eb610eec2" // validator-4
	contractAddress := "0x671bb70b0b504E9Ac6981E347734dB07d3ef7562"             // SXNode.sol on hamilton

	// function params
	marketHashParam := "0x50ed19e2397382c3fa9130033534636d2e290b46e034aef10c0c6d7186f4f3ad"
	outcomeParam := int32(1)
	epochParam := uint64(8239)
	timestampParam := int64(1663711090)

	sig1, hashed1 := getSigAndHashedPayload(marketHashParam, outcomeParam, epochParam, timestampParam, pk1)
	sig1Decoded, _ := hex.DecodeHex(sig1)
	sig2, _ := getSigAndHashedPayload(marketHashParam, outcomeParam, epochParam, timestampParam, pk2)
	sig2Decoded, _ := hex.DecodeHex(sig2)
	sig3, _ := getSigAndHashedPayload(marketHashParam, outcomeParam, epochParam, timestampParam, pk3)
	sig3Decoded, _ := hex.DecodeHex(sig3)
	sig4, _ := getSigAndHashedPayload(marketHashParam, outcomeParam, epochParam, timestampParam, pk4)
	sig4Decoded, _ := hex.DecodeHex(sig4)

	t.Logf("sig1 %s", sig1)

	t.Logf("hashedReport1: %s", hex.EncodeToHex(hashed1))

	sig1Decoded[64] = sig1Decoded[64] - 27

	pub, err := cryptoutils.SigToPub(hashed1, sig1Decoded)
	if err != nil {
		t.Error(err)
	}

	t.Logf("derived address for sig1: %s", cryptoutils.PubKeyToAddress(pub))

	var functions = []string{
		`function reportOutcome(bytes32 marketHash, int32 outcome, uint64 epoch, uint256 timestamp, bytes[] signatures)`,
	}

	abiContract, err := abi.NewABIFromList(functions)
	if err != nil {
		t.Fatalf("failed to retrieve ethgo ABI, %v", err)

		return
	}

	client, err := jsonrpc.NewClient(jsonRPCURL)
	if err != nil {
		t.Fatalf("failed to initialize new ethgo client, %v", err)

		return
	}

	privateKeyBytes, _ := hex.DecodeHex(pk1)
	wallet, _ := wallet.NewWalletFromPrivKey(privateKeyBytes)

	t.Logf("sending tx from sender %s", wallet.Address().String())

	c := contract.NewContract(
		ethgo.Address(types.StringToAddress(contractAddress)),
		abiContract,
		contract.WithSender(wallet),
		contract.WithJsonRPC(client.Eth()),
	)

	txn, err := c.Txn(
		"reportOutcome",
		types.StringToHash(marketHashParam),
		outcomeParam,
		epochParam,
		new(big.Int).SetInt64(timestampParam),
		[][]byte{sig1Decoded, sig2Decoded, sig3Decoded, sig4Decoded},
	)
	if err != nil {
		t.Fatalf("failed to create txn via ethgo, %v", err)

		return
	}

	err = txn.Do()
	if err != nil {
		t.Fatalf("failed to send raw txn via ethgo, %v", err)

		return
	}

	receipt, err := txn.Wait()
	if err != nil {
		t.Fatalf("failed to get txn receipt via ethgo, %v", err)

		return
	}

	t.Log("txReceipt", "status", receipt.Status)
}

// helper function used in e2e test
func getSigAndHashedPayload(
	marketHash string,
	outcome int32,
	epoch uint64,
	timestamp int64,
	privateKey string,
) (string, []byte) {
	getPrivateKey := func(privateKeyStr string) *ecdsa.PrivateKey {
		privateKeyBytes, _ := hex.DecodeHex(privateKeyStr)
		privateKey, _ := wallet.ParsePrivateKey(privateKeyBytes)

		return privateKey
	}(privateKey)

	getConsensusInfoImpl := func() *consensus.ConsensusInfo {
		return &consensus.ConsensusInfo{
			Validators:       validators.NewBLSValidatorSet(),
			ValidatorKey:     getPrivateKey,
			ValidatorAddress: cryptoutils.PubKeyToAddress(&getPrivateKey.PublicKey),
			Epoch:            0,
			QuorumSize:       0,
		}
	}

	getConsensusInfo := func() consensus.ConsensusInfoFn {
		return getConsensusInfoImpl
	}

	dataFeedService, _ := datafeed.NewDataFeedService(
		hclog.NewNullLogger(),
		&datafeed.Config{
			MQConfig: &datafeed.MQConfig{
				AMQPURI:      "",
				ExchangeName: "",
				QueueConfig: &datafeed.QueueConfig{
					QueueName: "",
				},
			},
		},
		nil,
		getConsensusInfo(),
	)

	payload := &proto.DataFeedReport{
		MarketHash: marketHash,
		Outcome:    outcome,
	}

	signedDataGeth, _ := gethcrypto.Sign(dataFeedService.AbiEncode(payload), getConsensusInfoImpl().ValidatorKey)
	signedDataGeth[64] = signedDataGeth[64] + 27

	sig := hex.EncodeToHex(signedDataGeth)

	hashedPayload := dataFeedService.AbiEncode(payload)

	return sig, hashedPayload
}
