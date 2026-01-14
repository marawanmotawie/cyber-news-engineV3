package main

import (
	"crypto-news-intelligence/internal"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

// Global Store (Thread-Safe)
type NewsStore struct {
	sync.RWMutex
	Items       []internal.NewsItem
	MarketState internal.MarketState
	SeenIDs     map[string]bool
}

var store = &NewsStore{
	Items:   []internal.NewsItem{},
	SeenIDs: make(map[string]bool),
}

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, relying on system environment variables")
	}

	fmt.Println("🚀 Crypto News Intelligence Engine (Server Mode) Starting...")
	fmt.Println("🌍 API Server running on http://localhost:8081")
	fmt.Println("📡 Scraper running in background (10s interval)...")
	fmt.Println("==================================================")

	// 1. Initialize Database
	internal.InitDB()

	// 2. Load History from DB
	history := internal.GetLatestNews(100)
	store.Lock()
	for _, item := range history {
		store.Items = append(store.Items, item)
		store.SeenIDs[item.ID] = true
	}
	fmt.Printf("📂 Loaded %d items from database.\n", len(store.Items))
	store.Unlock()

	// 3. Start Background Scraper
	go runBackgroundScraper()

	// 4. Setup HTTP Server
	http.HandleFunc("/api/news", handleGetNews)
	http.HandleFunc("/api/market", handleGetMarket)
	
	// Serve Static Dashboard (web folder)
	fs := http.FileServer(http.Dir("./web"))
	http.Handle("/", fs)

	// Dynamically get port from Environment (for Cloud Hosting like Render)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Printf("✅ Server is LIVE on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runBackgroundScraper() {
	feeds := map[string]string{
		"Binance Announcements": "HEADLESS", // Special marker
		"CoinDesk":              "https://www.coindesk.com/arc/outboundfeeds/rss/",
		"CoinTelegraph":         "https://cointelegraph.com/rss",
		"Decrypt":               "https://decrypt.co/feed",
	}

	for {
		var wg sync.WaitGroup
		resultsChan := make(chan internal.NewsItem, 100)

		// Fetch
		for name, url := range feeds {
			wg.Add(1)
			go func(sourceName, sourceUrl string) {
				defer wg.Done()
				var items []internal.NewsItem
				var err error

				if sourceName == "Binance Announcements" {
					// Use the new Headless Scraper
					items, err = internal.FetchBinanceHeadless()
				} else {
					items, err = internal.FetchRSS(sourceUrl, sourceName)
				}

				if err != nil {
					// Filter out common noise
					msg := err.Error()
					if !strings.Contains(msg, "193") && !strings.Contains(msg, "timeout") {
						log.Printf("⚠️  Error fetching from %s: %v", sourceName, err)
					}
					return
				}

				for _, item := range items {
					internal.AnalyzeNews(&item)
					resultsChan <- item
				}
			}(name, url)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		// Process & Update Store
		var newItems []internal.NewsItem
		store.Lock()
		for item := range resultsChan {
			if !store.SeenIDs[item.ID] {
				newItems = append(newItems, item)
				store.SeenIDs[item.ID] = true
				
				// Persist to DB
				internal.SaveNewsItem(item)
			}
		}
		
		// Prepend new items to the list (newest first)
		if len(newItems) > 0 {
			// Update Market Context
			store.MarketState = internal.CalculateMarketState(newItems)
			
			// Apply Rules & Calculate Score
			for i := range newItems {
				if newItems[i].Scope != "MARKET" {
					internal.ApplyTradingRules(&newItems[i], store.MarketState)
					newItems[i].FinalScore = internal.CalculateScore(newItems[i])
					
					// Update DB with Score/Signal
					internal.SaveNewsItem(newItems[i])
				}
			}

			store.Items = append(newItems, store.Items...)
			// Keep only last 100 items to prevent memory bloat
			if len(store.Items) > 100 {
				store.Items = store.Items[:100]
			}
			fmt.Printf("✓ Synced %d new items.\n", len(newItems))
		}

		// Memory Safety: Prevent SeenIDs from growing infinitely
		if len(store.SeenIDs) > 5000 {
			// Reset map and re-populate only with active items
			newMap := make(map[string]bool)
			for _, item := range store.Items {
				newMap[item.ID] = true
			}
			store.SeenIDs = newMap
			log.Println("🧹 Garbage Collection: Cleaned up old SeenIDs.")
		}
		
		store.Unlock()

		// Async: Process AI for new items (Non-blocking)
		if len(newItems) > 0 {
			go func(items []internal.NewsItem) {
				for _, item := range items {
					// FORCE AI ON EVERYTHING FOR TESTING
					// if item.FinalScore >= 0.05 || item.FinalScore <= -0.05 || item.TradingSignal == "STRONG_BUY" || item.Impact >= 0.7 {
					if true {
						fmt.Printf("🤖 Asking AI about: %s (Score: %.2f)...\n", item.Title, item.FinalScore)
						ctx, advice, coin, signal := internal.AnalyzeNewsAI(item)
						
						if ctx != "" {
							fmt.Printf("✅ AI Insight Ready: %s (Coin: %s, Signal: %s)\n", item.Title, coin, signal)
						}
						
						// Update Store Thread-Safely
						store.Lock()
						for i := range store.Items {
							if store.Items[i].ID == item.ID {
								store.Items[i].AIAnalysis = ctx
								store.Items[i].AIAdvice = advice
								store.Items[i].CoinSymbol = coin
								
								// OVERRIDE Signal with AI opinion if valid
								if signal != "" && signal != "WAIT" {
									store.Items[i].TradingSignal = signal
								}
								
								// Update DB with AI results
								internal.SaveNewsItem(store.Items[i])
								break
							}
						}
						store.Unlock()
					}
				}
			}(newItems)
		}

		time.Sleep(10 * time.Second)
	}
}

// API Handlers
func handleGetNews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	store.RLock()
	defer store.RUnlock()
	
	json.NewEncoder(w).Encode(store.Items)
}

func handleGetMarket(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	store.RLock()
	defer store.RUnlock()

	json.NewEncoder(w).Encode(store.MarketState)
}
