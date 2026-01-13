package internal

import (
	"math"
)

// MarketState represents the global market mood
type MarketState struct {
	Mood  string  // BULLISH, BEARISH, NEUTRAL
	Score float64 // Aggregate score of market news
}

// CalculateMarketState aggregates all MARKET scope news to determine global mood
func CalculateMarketState(items []NewsItem) MarketState {
	var totalScore float64
	var count int

	for _, item := range items {
		if item.Scope == "MARKET" {
			// Weight recent news more? For now flat weight.
			// Use the raw score calculated previously (Impact * Sentiment * Trust)
			score := CalculateScore(item)
			totalScore += score
			count++
		}
	}

	state := MarketState{Mood: "NEUTRAL", Score: 0}
	if count > 0 {
		state.Score = totalScore
		if totalScore > 0.2 {
			state.Mood = "BULLISH"
		} else if totalScore < -0.2 {
			state.Mood = "BEARISH"
		}
	}
	return state
}

// ApplyTradingRules applies context-aware logic to generate signals
func ApplyTradingRules(item *NewsItem, market MarketState) {
	// Calculate base asset score
	assetScore := CalculateScore(*item) 
	
	// Default
	item.TradingSignal = "WAIT"
	item.RuleReason = "Low impact or neutral signal"

	// 1. Filter Noise
	if math.Abs(assetScore) < 0.05 {
		item.TradingSignal = "IGNORE"
		item.RuleReason = "Noise / Insufficient Impact"
		return
	}

	// 2. The Golden Rule: Context Awareness
	// If Asset is Bullish...
	if assetScore > 0.1 {
		if market.Mood == "BEARISH" {
			item.TradingSignal = "CAUTION"
			item.RuleReason = "Asset Bullish but Market is Bearish (High Risk)"
		} else if market.Mood == "BULLISH" {
			item.TradingSignal = "STRONG_BUY"
			item.RuleReason = "Asset Bullish + Market Bullish (Trend Confirmation)"
		} else {
			item.TradingSignal = "BUY"
			item.RuleReason = "Asset Bullish in Neutral Market"
		}
	}

	// If Asset is Bearish...
	if assetScore < -0.1 {
		if market.Mood == "BULLISH" {
			item.TradingSignal = "CAUTION_SELL"
			item.RuleReason = "Asset Bearish but Market is Bullish (Potential Dip Buy?)"
		} else if market.Mood == "BEARISH" {
			item.TradingSignal = "STRONG_SELL"
			item.RuleReason = "Asset Bearish + Market Bearish (Trend Confirmation)"
		} else {
			item.TradingSignal = "SELL"
			item.RuleReason = "Asset Bearish in Neutral Market"
		}
	}
}
