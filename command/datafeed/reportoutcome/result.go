package reportoutcome

import (
	"bytes"
	"fmt"

	"github.com/0xPolygon/polygon-edge/command/helper"
)

type DataFeedReportOutcomeResult struct {
	MarketHash string `json:"market"`
	Outcome    string `json:"outcome"`
}

func newDataFeedReportOutcomeResult(marketHash, outcome string) *DataFeedReportOutcomeResult {
	res := &DataFeedReportOutcomeResult{
		MarketHash: marketHash,
		Outcome:    outcome,
	}

	return res
}

func (r *DataFeedReportOutcomeResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[DATAFEED REPORT SUBMITTED]\n")
	buffer.WriteString(helper.FormatKV([]string{
		fmt.Sprintf("MARKET HASH|%s", r.MarketHash),
		fmt.Sprintf("OUTCOME|%s", r.Outcome),
	}))
	buffer.WriteString("\n")

	return buffer.String()
}
