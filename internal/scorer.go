package internal

import "strings"

// CalculateScore calculates the final score based on impact, sentiment, and trust
func CalculateScore(item NewsItem) float64 {
	trustWeight := 0.7 // Default for news sites
	source := strings.ToLower(item.Source)
	
	// Exchange announcements get higher trust
	if strings.Contains(source, "binance") || strings.Contains(source, "coinbase") || strings.Contains(source, "exchange") {
		trustWeight = 1.0
	}

	return item.Impact * item.Sentiment * trustWeight
}
