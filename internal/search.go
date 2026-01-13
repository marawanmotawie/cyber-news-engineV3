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

// Internal state for key rotation
var (
	serperKeys            []string
	currentSerperKeyIndex = 0
	serperKeysOnce        sync.Once
)

func getSerperKeys() []string {
	serperKeysOnce.Do(func() {
		rawKeys := os.Getenv("SERPER_KEYS")
		if rawKeys != "" {
			serperKeys = strings.Split(rawKeys, ",")
			for i := range serperKeys {
				serperKeys[i] = strings.TrimSpace(serperKeys[i])
			}
		}
		if len(serperKeys) == 0 {
			log.Println("⚠️  WARNING: No SERPER_KEYS found in environment!")
		}
	})
	return serperKeys
}

func GetNextSerperKey() string {
	keys := getSerperKeys()
	if len(keys) == 0 {
		return ""
	}
	key := keys[currentSerperKeyIndex]
	currentSerperKeyIndex = (currentSerperKeyIndex + 1) % len(keys)
	return key
}

type SerperResponse struct {
	Organic []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
		Date    string `json:"date"`
	} `json:"organic"`
}

// SearchWeb performs a Google search using Serper.dev
func SearchWeb(query string) string {
	url := "https://google.serper.dev/search"
	
	payload := map[string]interface{}{
		"q":   query,
		"num": 3,
		"tbs": "qdr:d",
	}
	jsonPayload, _ := json.Marshal(payload)

	keys := getSerperKeys()
	for i := 0; i < len(keys); i++ {
		key := GetNextSerperKey()
		if key == "" {
			break
		}

		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
		req.Header.Set("X-API-KEY", key)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)

		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			var result SerperResponse
			if err := json.Unmarshal(body, &result); err != nil {
				continue
			}

			var sb strings.Builder
			sb.WriteString("Search Results (Verification Context):\n")
			for _, item := range result.Organic {
				sb.WriteString(fmt.Sprintf("- %s: %s (%s)\n", item.Title, item.Snippet, item.Date))
			}
			
			if len(result.Organic) == 0 {
				return "No relevant search results found."
			}

			return sb.String()
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return "Search API Unavailable."
}
