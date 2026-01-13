package internal

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/mmcdole/gofeed"
	"net/http"
	"strings"
	"time"
)

// FetchRSS fetches news from a standard RSS feed
func FetchRSS(url string, sourceName string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, item := range feed.Items {
		// Just take the last 10
		if len(items) >= 10 {
			break
		}
		
		pubDate := item.PublishedParsed
		if pubDate == nil {
			now := time.Now()
			pubDate = &now
		}

		newsItem := NewsItem{
			ID:        item.GUID,
			Title:     item.Title,
			Source:    sourceName,
			Timestamp: *pubDate,
		}
		
		// Fallback ID if GUID missing
		if newsItem.ID == "" {
			newsItem.ID = item.Link
		}

		items = append(items, newsItem)
	}
	return items, nil
}

// FetchBinanceHeadless scans Binance using a real hidden browser
func FetchBinanceHeadless() ([]NewsItem, error) {
	// 1. Setup Chromedp Context (Stealth Mode)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true), 
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-automation", false), // Key for stealth
		chromedp.Flag("disable-blink-features", "AutomationControlled"), // Key for stealth
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// 2. Set timeout (60s)
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 3. Navigate & Extract
	// We use a broader wait and extraction strategy
	var titles []string
	err := chromedp.Run(ctx,
		// Hide webdriver property
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := chromedp.Evaluate(`Object.defineProperty(navigator, 'webdriver', {get: () => undefined})`, nil).Do(ctx)
			return err
		}),
		chromedp.Navigate("https://www.binance.com/en/support/announcement/new-cryptocurrency-listing?c=48&navId=48"),
		// Wait for ANY main content
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(5 * time.Second), 
		// specific JS to grab the first few links that look like announcements
		chromedp.Evaluate(`
			Array.from(document.querySelectorAll('a'))
				.filter(a => a.href.includes('/support/announcement/'))
				.map(a => a.innerText)
				.filter(t => t.length > 20 && !t.includes("View More"))
				// deduplicate by text
				.filter((v, i, a) => a.indexOf(v) === i)
				.slice(0, 10)
		`, &titles),
	)

	if err != nil {
		return nil, fmt.Errorf("headless fetch failed: %v", err)
	}

	var items []NewsItem
	for _, title := range titles {
		// Cleanup title
		title = strings.TrimSpace(title)
		if title == "" { continue }

		items = append(items, NewsItem{
			ID:        "binance-" + title, // Simple ID
			Title:     title,
			Source:    "Binance",
			Timestamp: time.Now(), // Approx time
		})
	}

	return items, nil
}

// Fallback BAPI (Kept for legacy)
func FetchBinanceBAPI() ([]NewsItem, error) {
	url := "https://www.binance.com/bapi/composite/v1/public/cms/article/list/query"
	payload := strings.NewReader(`{"type":"catalogs","catalogId":48,"pageNo":1,"pageSize":10}`)

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("binance api returned status: %d", resp.StatusCode)
	}
	// ... decoding omitted for brevity as unused ...
	return nil, nil 
}
