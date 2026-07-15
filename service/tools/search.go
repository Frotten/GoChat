package tools

import (
	"GopherAI/model"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	maxWebSearchResults      = 3
	maxWebSearchAnswerRunes  = 800
	maxWebSearchSnippetRunes = 400
	maxWebSearchTitleRunes   = 120
	maxWebSearchURLRunes     = 300
)

func TavilySearch(ctx context.Context, Query string) (string, error) {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return toolNoRetry("TAVILY_API_KEY is not configured"), nil
	}
	body := &model.TavilyRequest{
		Query:         Query,
		SearchDepth:   "basic",
		IncludeAnswer: true,
		MaxResults:    maxWebSearchResults,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.tavily.com/search",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return toolNoRetry("web search request failed"), nil
	}

	var result model.TavilyResponse
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	answer, truncated := truncateText(result.Answer, maxWebSearchAnswerRunes)

	results := make([]map[string]interface{}, 0, len(result.Results))
	for i, item := range result.Results {
		if i >= maxWebSearchResults {
			truncated = true
			break
		}
		title, titleTruncated := truncateText(item.Title, maxWebSearchTitleRunes)
		url, urlTruncated := truncateText(item.URL, maxWebSearchURLRunes)
		snippet, snippetTruncated := truncateText(item.Content, maxWebSearchSnippetRunes)
		truncated = truncated || titleTruncated || urlTruncated || snippetTruncated
		results = append(results, map[string]interface{}{
			"rank":    i + 1,
			"title":   title,
			"url":     url,
			"snippet": snippet,
			"score":   item.Score,
		})
	}

	query := result.Query
	if query == "" {
		query = Query
	}
	return toolJSONResult(map[string]interface{}{
		"success":   true,
		"query":     query,
		"answer":    answer,
		"results":   results,
		"truncated": truncated,
	}), nil
}

func truncateText(s string, maxRunes int) (string, bool) {
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return "", s != ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s, false
	}
	return string(runes[:maxRunes]) + "...", true
}
