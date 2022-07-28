package types

// ReportOutcome used by DataFeedService's MQConsumer and GRPC reporting
type ReportOutcome struct {
	MarketHash string `json:"market"`
	Outcome    string `json:"outcome"`
}
