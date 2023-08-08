package datafeed

import (
	"context"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
)

// ReportOutcome reports an outcome to the datafeed consumer
func (d *DataFeed) ReportOutcome(
	ctx context.Context,
	request *proto.ReportOutcomeReq,
) (*proto.ReportOutcomeResp, error) {
	reportOutcome := &proto.DataFeedReport{
		MarketHash: request.Market,
		Outcome:    request.Outcome,
	}

	d.queueReportingTx(ProposeOutcome, reportOutcome.MarketHash, reportOutcome.Outcome)

	return &proto.ReportOutcomeResp{
		MarketHash: request.Market,
	}, nil
}
