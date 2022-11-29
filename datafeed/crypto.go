package datafeed

import (
	"crypto/ecdsa"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/ethereum/go-ethereum/crypto"
)

func (d *DataFeed) GetSignatureForPayload(payload *proto.DataFeedReport) (string, error) {
	signedData, err := crypto.Sign(d.AbiEncode(payload), d.consensusInfo().ValidatorKey)
	if err != nil {
		return "", err
	}
	// add 27 to V since go-ethereum crypto.Sign() produces V as 0 or 1
	signedData[64] = signedData[64] + 27

	return hex.EncodeToHex(signedData), nil
}

func (d *DataFeed) signatureToAddress(payload *proto.DataFeedReport, signature string) (*ecdsa.PublicKey, error) {
	buf, err := hex.DecodeHex(signature)
	if err != nil {
		return nil, err
	}
	// subtract 27 from V since go-ethereum crypto.Ecrecover() expects V as 0 or 1
	buf[64] = buf[64] - 27

	pub, err := crypto.SigToPub(d.AbiEncode(payload), buf)
	if err != nil {
		return nil, err
	}

	return pub, nil
}
