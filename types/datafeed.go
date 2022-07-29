package types

// ReportOutcome represents a report payload
type ReportOutcome struct {
	MarketHash string `json:"market"`
	Outcome    string `json:"outcome"`
}

// ReportOutcomeGossip represents a signed report payload, used for gossiping between validators
type ReportOutcomeGossip struct {
	MarketHash string `json:"marketHash"`
	Outcome    string `json:"outcome"`
	Signatures string `json:"signatures"`
	Epoch      uint64 `json:"epoch"`
	Timestamp  int64  `json:"timestamp"`
}
