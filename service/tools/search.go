package tools

import (
	"GopherAI/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func TavilySearch(ctx context.Context, Query string) (string, error) {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return "", nil
	}
	Body := &model.TavilyRequest{
		Query:         Query,
		SearchDepth:   "basic",
		IncludeAnswer: true,
		MaxResults:    5,
	}
	body, err := json.Marshal(Body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.tavily.com/search",
		bytes.NewReader(body),
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
	var result model.TavilyResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if result.Answer != "" {
		sb.WriteString("总结：\n")
		sb.WriteString(result.Answer)
		sb.WriteString("\n\n")
	}
	for i, item := range result.Results {
		_, _ = fmt.Fprintf(&sb,
			"结果%d：\n标题：%s\n内容：%s\n链接：%s\n\n",
			i+1,
			item.Title,
			item.Content,
			item.URL,
		)
	}
	return sb.String(), nil
}
