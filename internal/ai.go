package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// OpenAI/DashScope Request Structure
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI/DashScope Response Structure
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error"`
}

// Internal state for key rotation
var (
	aiKeys            []string
	currentKeyIndex   = 0
	aiKeysOnce        sync.Once
)

func getAIKeys() []string {
	aiKeysOnce.Do(func() {
		rawKeys := os.Getenv("AI_KEYS")
		if rawKeys != "" {
			aiKeys = strings.Split(rawKeys, ",")
			for i := range aiKeys {
				aiKeys[i] = strings.TrimSpace(aiKeys[i])
			}
		}
		if len(aiKeys) == 0 {
			log.Println("⚠️  WARNING: No AI_KEYS found in environment!")
		}
	})
	return aiKeys
}

func GetNextKey() string {
	keys := getAIKeys()
	if len(keys) == 0 {
		return ""
	}
	key := keys[currentKeyIndex]
	currentKeyIndex = (currentKeyIndex + 1) % len(keys)
	return key
}

// AnalyzeNewsAI calls AI to analyze the news
func AnalyzeNewsAI(item NewsItem) (string, string, string, string) {
	searchQuery := fmt.Sprintf("%s %s crypto news", item.Title, item.Asset)
	fmt.Printf("🔍 Serper Searching: %s...\n", searchQuery)
	searchResults := SearchWeb(searchQuery)

	prompt := fmt.Sprintf(`
Analyze this crypto news headline: "%s" (Asset: %s).

Verified Web Search Context (Live Data):
%s

Respond in JSON:
{
  "context": "Hidden context in 1 Arabic sentence based on search results.",
  "advice": "Trading advice (Buy/Sell/Wait) in 1 Arabic sentence.",
  "coin": "The specific coin symbol (e.g. DOT, SOL, BTC) or 'GENERAL'.",
  "signal": "One of: STRONG_BUY, BUY, WAIT, CAUTION, SELL, STRONG_SELL"
}
`, item.Title, item.Asset, searchResults)

	ollamaUrl := os.Getenv("OLLAMA_URL")
	if ollamaUrl == "" {
		ollamaUrl = "https://ollama.com/api/generate"
	}
	
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "qwen3-coder:480b-cloud"
	}

	ollamaPayload := map[string]interface{}{
		"model":  model,
		"prompt": prompt + " Respond in JSON only.",
		"stream": false,
		"format": "json",
	}
	ollamaJson, _ := json.Marshal(ollamaPayload)

	keys := getAIKeys()
	for i := 0; i < len(keys); i++ {
		key := GetNextKey()
		
		client := &http.Client{Timeout: 15 * time.Second}
		req, _ := http.NewRequest("POST", ollamaUrl, bytes.NewBuffer(ollamaJson))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+key)

		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			var result ChatResponse
			if err := json.Unmarshal(body, &result); err == nil && len(result.Choices) > 0 {
				return parseRawResponse(result.Choices[0].Message.Content)
			}

			var ollamaRes struct {
				Response string `json:"response"`
			}
			if err := json.Unmarshal(body, &ollamaRes); err == nil && ollamaRes.Response != "" {
				return parseRawResponse(ollamaRes.Response)
			}
			
			return parseRawResponse(string(body))
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return "AI Exhausted", "All keys failed.", "", "WAIT"
}

func parseRawResponse(raw string) (string, string, string, string) {
	raw = strings.ReplaceAll(raw, "```json", "")
	raw = strings.ReplaceAll(raw, "```", "")
	raw = strings.TrimSpace(raw)

	var aiResult struct {
		Context string `json:"context"`
		Advice  string `json:"advice"`
		Coin    string `json:"coin"`
		Signal  string `json:"signal"`
	}
	if err := json.Unmarshal([]byte(raw), &aiResult); err != nil {
		return raw, "Check context", "", "WAIT"
	}
	if aiResult.Signal == "" { aiResult.Signal = "WAIT" }
	
	return aiResult.Context, aiResult.Advice, aiResult.Coin, aiResult.Signal
}
