package types

// ReportOutcome represents a signed report payload, used for gossiping between validators
type ReportOutcome struct {
	MarketHash string `json:"market"`
	Outcome    int32  `json:"outcome"`
	Signatures string `json:"signatures"`
	Epoch      uint64 `json:"epoch"`
	Timestamp  int64  `json:"timestamp"`
	IsGossip   bool   `json:"is_gossip"`
}
