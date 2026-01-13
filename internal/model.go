package internal

import "time"

// NewsItem defines a normalized news struct
type NewsItem struct {
	ID        string    `json:"ID"`
	Title     string    `json:"Title"`
	Source    string    `json:"Source"`
	Scope     string    `json:"Scope"`
	Asset     string    `json:"Asset"`
	Impact    float64   `json:"Impact"`
	Sentiment float64   `json:"Sentiment"`
	Timestamp time.Time `json:"Timestamp"`

	// Phase 3: Decision Support
	TradingSignal string  `json:"TradingSignal"`
	RuleReason    string  `json:"RuleReason"`
	FinalScore    float64 `json:"FinalScore"`

	// Phase 7: AI Analysis
	AIAnalysis string `json:"AIAnalysis"`
	AIAdvice   string `json:"AIAdvice"`
	CoinSymbol string `json:"CoinSymbol"`
}
