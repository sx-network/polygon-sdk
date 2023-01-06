package datafeed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type verifyAPIResponse struct {
	Outcome   int32
	Timestamp int64
}

// TODO: should we refactor verify API to just return the actual outcome instead of success/failure?

// verifyMarketOutcome compares the proposed market outcome with the verify api outcome,
// returning error if outcome does not match
func (d *DataFeed) verifyMarketOutcome(marketHash string) (int32, error) {
	requestURL := fmt.Sprintf("%s/%s", d.config.VerifyOutcomeURI, marketHash)
	response, err := http.Get(requestURL) //nolint:gosec

	if err != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to verify market outcome with server error", "error", err)

		return -1, err
	}

	defer response.Body.Close()
	body, parseErr := ioutil.ReadAll(response.Body)

	if parseErr != nil {
		d.logger.Error("[MARKET-VERIFICATION] Failed to parse response", "parseError", parseErr)

		return -1, parseErr
	}

	if response.StatusCode != 200 {
		d.logger.Error(
			"[MARKET-VERIFICATION] Got non-200 response from verify outcome",
			"market", marketHash,
			"parsedBody", body,
			"statusCode", response.StatusCode,
		)

		return -1, fmt.Errorf("got non-200 response from Verify Outcome API response")
	}

	var data verifyAPIResponse

	marshalErr := json.Unmarshal(body, &data)
	if marshalErr != nil {
		d.logger.Error(
			"[MARKET-VERIFICATION] Failed to unmarshal outcome",
			"body", body,
			"parseError", marshalErr,
		)

		return -1, marshalErr
	}

	return data.Outcome, nil
}
