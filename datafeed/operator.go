package datafeed

import (
	"context"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
	"github.com/0xPolygon/polygon-edge/types"
)

// ReportOutcome reports an outcome to the datafeed consumer
func (d *DataFeed) ReportOutcome(
	ctx context.Context,
	request *proto.ReportOutcomeReq,
) (*proto.ReportOutcomeResp, error) {
	reportOutcome := &types.ReportOutcome{
		MarketHash: request.Market,
		Outcome:    request.Outcome,
	}

	d.publishPayload(reportOutcome, false)

	return &proto.ReportOutcomeResp{
		MarketHash: request.Market,
	}, nil
}
