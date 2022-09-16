package datafeed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/0xPolygon/polygon-edge/crypto"
	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/helper/hex"
)

type verifyApiResponse struct {
	Outcome   int32
	Timestamp int64
}

func (d *DataFeed) verifyMarketOutcome(payload *proto.DataFeedReport, verifyApiHost string) (bool, error) {
	requestURL := fmt.Sprintf("%s/api/outcome/%s", verifyApiHost, payload.MarketHash)
	response, err := http.Get(requestURL)

	if err != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to verify market outcome with server error", "error", err)
		return false, err
	}

	defer response.Body.Close()
	body, parseErr := ioutil.ReadAll(response.Body)

	if parseErr != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to parse response", "parseError", parseErr)
		return false, parseErr
	}

	if response.StatusCode != 200 {
		d.logger.Error("[MARKET-VERIFICATION] Failed to verify market with invalid outcome/payload", "market", payload.MarketHash, "outcome", payload.Outcome, "parsedBody", body)
		return false, fmt.Errorf("Failed to verify market with invalid outcome/payload")
	}

	var data verifyApiResponse
	var marshalErr = json.Unmarshal(body, &data)

	if marshalErr != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to unmarshal outcome", "body", body, "parseError", marshalErr)
		return false, marshalErr
	}

	if data.Outcome == payload.Outcome {
		return true, nil
	}

	return false, fmt.Errorf("Failed to verify market")

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

		pub, err := crypto.RecoverPubkey(buf, d.AbiEncode(payload))
		if err != nil {
			return false, err
		}

		// see if we signed it
		if d.consensusInfo().ValidatorAddress == crypto.PubKeyToAddress(pub) {
			return true, nil
		} else {
			// if we haven't signed it, see if a recognized validator from the current validator set signed it
			otherValidatorSigned := false
			for _, validator := range d.consensusInfo().Validators {
				if validator == crypto.PubKeyToAddress(pub) {
					otherValidatorSigned = true
				}
			}
			if !otherValidatorSigned {
				return false, fmt.Errorf("unrecognized signature detected")
			}
		}
	}

	return false, nil
}
