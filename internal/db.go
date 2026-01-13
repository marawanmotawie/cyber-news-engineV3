package internal

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB() {
	var err error
	DB, err = sql.Open("sqlite", "./news.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableSQL := `CREATE TABLE IF NOT EXISTS news_items (
		id TEXT PRIMARY KEY,
		title TEXT,
		source TEXT,
		scope TEXT,
		asset TEXT,
		impact REAL,
		sentiment REAL,
		timestamp DATETIME,
		trading_signal TEXT,
		rule_reason TEXT,
		final_score REAL,
		ai_analysis TEXT,
		ai_advice TEXT,
		coin_symbol TEXT
	);`

	_, err = DB.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

// SaveNewsItem inserts or updates a news item
func SaveNewsItem(item NewsItem) {
	stmt, err := DB.Prepare(`INSERT INTO news_items(
		id, title, source, scope, asset, impact, sentiment, timestamp,
		trading_signal, rule_reason, final_score, ai_analysis, ai_advice, coin_symbol
	) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		ai_analysis=excluded.ai_analysis,
		ai_advice=excluded.ai_advice,
		coin_symbol=excluded.coin_symbol,
		final_score=excluded.final_score,
		trading_signal=excluded.trading_signal
	;`)
	if err != nil {
		log.Println("DB Prepare Error:", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(
		item.ID, item.Title, item.Source, item.Scope, item.Asset,
		item.Impact, item.Sentiment, item.Timestamp,
		item.TradingSignal, item.RuleReason, item.FinalScore,
		item.AIAnalysis, item.AIAdvice, item.CoinSymbol,
	)
	if err != nil {
		log.Println("DB Save Error:", err)
	}
}

// GetLatestNews retrieves the last N items (for startup)
func GetLatestNews(limit int) []NewsItem {
	rows, err := DB.Query("SELECT * FROM news_items ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		log.Println("DB Query Error:", err)
		return nil
	}
	defer rows.Close()

	var items []NewsItem
	for rows.Next() {
		var item NewsItem
		var ts time.Time
		err = rows.Scan(
			&item.ID, &item.Title, &item.Source, &item.Scope, &item.Asset,
			&item.Impact, &item.Sentiment, &ts,
			&item.TradingSignal, &item.RuleReason, &item.FinalScore,
			&item.AIAnalysis, &item.AIAdvice, &item.CoinSymbol,
		)
		if err != nil {
			continue
		}
		item.Timestamp = ts
		items = append(items, item)
	}
	return items
}
