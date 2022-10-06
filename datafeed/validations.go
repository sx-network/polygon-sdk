package datafeed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	cryptoutils "github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
	"github.com/ethereum/go-ethereum/crypto"
)

type verifyAPIResponse struct {
	Outcome   int32
	Timestamp int64
}

// verifyMarketOutcome compares the payload's proposed market outcome with the verify outcome api outcome,
// returning error if outcome does not match
func (d *DataFeed) verifyMarketOutcome(payload *proto.DataFeedReport, verifyAPIHost string) error {
	marketHash := payload.MarketHash
	requestURL := fmt.Sprintf("%s/%s", verifyAPIHost, marketHash)
	response, err := http.Get(requestURL) //nolint:gosec

	if err != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to verify market outcome with server error", "error", err)

		return err
	}

	defer response.Body.Close()
	body, parseErr := ioutil.ReadAll(response.Body)

	if parseErr != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to parse response", "parseError", parseErr)

		return parseErr
	}

	if response.StatusCode != 200 {
		d.logger.Error(
			"[MARKET-VERIFICATION] Got non-200 response from verify outcome",
			"market", payload.MarketHash,
			"outcome", payload.Outcome,
			"parsedBody", body,
			"statusCode", response.StatusCode,
		)

		return fmt.Errorf("got non-200 response from Verify Outcome API response")
	}

	var data verifyAPIResponse

	marshalErr := json.Unmarshal(body, &data)
	if marshalErr != nil {
		d.logger.Error(
			"[MARKET-VERIFICATION] Failed to unmarshal outcome",
			"body", body,
			"parseError", marshalErr,
		)

		return marshalErr
	}

	if data.Outcome != payload.Outcome {
		errorMsg := fmt.Sprintf(
			"failed to verify market %s, expected outcome %d got %d",
			marketHash,
			payload.Outcome,
			data.Outcome,
		)

		return fmt.Errorf(errorMsg)
	}

	return nil
}

// validateTimestamp  checks if payload too old
func (d *DataFeed) validateTimestamp(payload *proto.DataFeedReport) bool {
	d.logger.Debug("time", "time", time.Since(time.Unix(payload.Timestamp, 0)).Seconds())

	return time.Since(time.Unix(payload.Timestamp, 0)).Seconds() > maxGossipTimestampDriftSeconds
}

// validateSignatures checks if the current validator has already signed the payload
func (d *DataFeed) validateSignatures(payload *proto.DataFeedReport) (bool, error) {
	sigList := strings.Split(payload.Signatures, ",")
	for _, sig := range sigList {
		buf, err := hex.DecodeHex(sig)
		if err != nil {
			return false, err
		}
		// subtract 27 from V since go-ethereum crypto.Ecrecover() expects V as 0 or 1
		buf[64] = buf[64] - 27

		pub, err := crypto.SigToPub(d.AbiEncode(payload), buf)
		if err != nil {
			return false, err
		}

		// see if we signed it
		if d.consensusInfo().ValidatorAddress == cryptoutils.PubKeyToAddress(pub) {
			return true, nil
		} else {
			// if we haven't signed it, see if a recognized validator from the current validator set signed it
			otherValidatorSigned := false
			for _, validator := range d.consensusInfo().Validators {
				if validator == cryptoutils.PubKeyToAddress(pub) {
					otherValidatorSigned = true
				}
			}
			if !otherValidatorSigned {
				return false, fmt.Errorf("unrecognized signature detected, got address: %s", cryptoutils.PubKeyToAddress(pub))
			}
		}
	}

	return false, nil
}
