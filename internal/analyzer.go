package internal

import (
	"strings"
)

// AnalyzeNews performs rule-based analysis on a news item
func AnalyzeNews(item *NewsItem) {
	title := strings.ToLower(item.Title)

	// Default values
	item.Scope = "ASSET"
	item.Asset = "ALT"
	item.Impact = 0.3
	item.Sentiment = 0.0

	// 1. Market-wide detection
	marketKeywords := []string{"fed", "cpi", "sec", "etf", "regulation", "inflation", "interest rate", "macro", "economy"}
	for _, kw := range marketKeywords {
		if strings.Contains(title, kw) {
			item.Scope = "MARKET"
			item.Asset = "ALL"
			item.Impact = 0.7
			break
		}
	}

	// 2. Asset-specific detection - Improved with word boundaries
	assets := map[string][]string{
		"BTC":  {"btc", "bitcoin"},
		"ETH":  {"eth", "ethereum", "ether"},
		"SOL":  {"sol", "solana"},
		"BNB":  {"bnb", "binance"},
		"XRP":  {"xrp", "ripple"},
		"ADA":  {"ada", "cardano"},
		"DOGE": {"doge", "dogecoin"},
		"APT":  {"apt", "aptos"},
	}

	for asset, keywords := range assets {
		for _, kw := range keywords {
			if containsWord(title, kw) {
				item.Asset = asset
				break
			}
		}
		if item.Asset != "ALT" {
			break
		}
	}

	// 3. Keyword Impact Table (Sentiment & Event detection)
	bullishKeywords := []string{"surges", "jumps", "breakout", "adds", "record high", "moon", "rally", "gains", "bullish", "outperform", "upgrade", "listing", "listed", "partnership", "collaboration", "legalizes", "adoption", "pushes", "above"}
	bearishKeywords := []string{"loses", "falls", "exit", "withdrawn", "bloodbath", "crash", "bearish", "drop", "down", "delisting", "delisted", "hack", "exploit", "compromised", "selloff", "backlash", "left", "outflow", "ban", "restrict", "lose", "losing"}

	for _, kw := range bullishKeywords {
		if containsWord(title, kw) {
			item.Sentiment += 0.3
		}
	}

	for _, kw := range bearishKeywords {
		if containsWord(title, kw) {
			item.Sentiment -= 0.3
		}
	}

	// 4. Specific High Impact Events
	if strings.Contains(title, "listing") || strings.Contains(title, "listed") {
		item.Impact = 0.8
	}
	if strings.Contains(title, "delisting") || strings.Contains(title, "delisted") {
		item.Impact = 0.9
	}
	if strings.Contains(title, "hack") || strings.Contains(title, "exploit") || strings.Contains(title, "compromised") {
		item.Impact = 1.0
	}

	// 5. Price Action Noise Filter (Option 2)
	priceKeywords := []string{"surges", "jumps", "climbs", "pops", "falls", "drops", "slumps"}
	isPriceActionOnly := false
	for _, kw := range priceKeywords {
		if strings.Contains(title, kw) {
			isPriceActionOnly = true
			break
		}
	}

	if isPriceActionOnly {
		eventKeywords := []string{"listing", "delisting", "hack", "exploit", "partnership", "fed", "cpi", "sec", "etf", "regulation", "legalizes", "approves"}
		hasEvent := false
		for _, kw := range eventKeywords {
			if strings.Contains(title, kw) {
				hasEvent = true
				break
			}
		}

		if !hasEvent {
			item.Impact = 0.1 // Just price noise, lower impact
		}
	}

	// Clamp sentiment
	if item.Sentiment > 1.0 { item.Sentiment = 1.0 }
	if item.Sentiment < -1.0 { item.Sentiment = -1.0 }
}

func containsWord(s, word string) bool {
	index := strings.Index(s, word)
	if index == -1 {
		return false
	}

	// Check prev char
	if index > 0 {
		prev := s[index-1]
		if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
			return false
		}
	}

	// Check next char
	end := index + len(word)
	if end < len(s) {
		next := s[end]
		if (next >= 'z' && next <= 'z') || (next >= '0' && next <= '9') {
			return false
		}
	}

	return true
}
