package datafeed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
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
