package reportoutcome

import (
	"context"
	"fmt"

	"github.com/0xPolygon/polygon-edge/command"
	"github.com/0xPolygon/polygon-edge/command/helper"
	datafeedOp "github.com/0xPolygon/polygon-edge/datafeed/proto"
)

const (
	market  = "market"
	outcome = "outcome"
)

var (
	params = &reportOutcomeParams{}
)

type reportOutcomeParams struct {
	market  string
	outcome string
}

func (p *reportOutcomeParams) reportOutcome(grpcAddress string) error {
	dataFeedClient, err := helper.GetDatafeedClientConnection(grpcAddress)
	if err != nil {
		return err
	}

	resp, err := dataFeedClient.ReportOutcome(context.Background(), p.getReportOutcomeRequest())
	if err != nil {
		return err
	}

	if resp.MarketHash != p.market {
		return fmt.Errorf("report could not be processed")
	}

	return nil
}

func (p *reportOutcomeParams) getReportOutcomeRequest() *datafeedOp.ReportOutcomeReq {
	req := &datafeedOp.ReportOutcomeReq{
		Market:  p.market,
		Outcome: p.outcome,
	}

	return req
}

func (p *reportOutcomeParams) getRequiredFlags() []string {
	return []string{
		market,
		outcome,
	}
}

func (p *reportOutcomeParams) getResult() command.CommandResult {
	return newDataFeedReportOutcomeResult(p.market, p.outcome)
}
