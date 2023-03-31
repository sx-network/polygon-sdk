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

// verifyMarket uses the verify market API to derive an outcome for the specified marketHash to vote on
func (d *DataFeed) verifyMarket(marketHash string) (int32, error) {

	requestURL := fmt.Sprintf("%s/%s", d.config.VerifyOutcomeURI, marketHash)

	d.logger.Debug("verifyMarket", "requestUrl", requestURL)
	response, err := http.Get(requestURL) //nolint:gosec

	if err != nil {
		d.logger.Error("failed to verify market with server error", "error", err)

		return -1, err
	}

	defer response.Body.Close()
	body, parseErr := ioutil.ReadAll(response.Body)

	if parseErr != nil {
		d.logger.Error("failed to parse response for verify market call", "parseError", parseErr)

		return -1, parseErr
	}

	if response.StatusCode != 200 {
		d.logger.Error(
			"got non-200 response for verify market call",
			"market", marketHash,
			"parsedBody", body,
			"statusCode", response.StatusCode,
		)

		return -1, fmt.Errorf("got non-200 response from verify market response with statusCode %d", response.StatusCode)
	}

	var data verifyAPIResponse

	marshalErr := json.Unmarshal(body, &data)
	if marshalErr != nil {
		d.logger.Error(
			"failed to unmarshal outcome for verify market response",
			"body", body,
			"parseError", marshalErr,
		)

		return -1, marshalErr
	}

	return data.Outcome, nil
}
