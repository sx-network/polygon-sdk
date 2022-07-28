package datafeed

import (
	"context"
	"encoding/json"

	"github.com/0xPolygon/polygon-edge/datafeed/proto"
)

// ReportOutcome reports an outcome to the datafeed consumer
func (d *DataFeed) ReportOutcome(
	ctx context.Context,
	request *proto.ReportOutcomeReq,
) (*proto.ReportOutcomeResp, error) {
	//TODO: for now we are marshalling to a string but we want the payload to be processed as an object eventually
	reportOutcomeString, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	d.ProcessPayload(string(reportOutcomeString))

	return &proto.ReportOutcomeResp{
		MarketHash: request.Market,
	}, nil
}
