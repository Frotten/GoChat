package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type QdrantClient struct {
	cfg    Config
	client *http.Client
}

func NewQdrantClient(cfg Config) *QdrantClient {
	return &QdrantClient{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (q *QdrantClient) EnsureCollection(ctx context.Context) error {
	url := fmt.Sprintf("%s/collections/%s", q.cfg.QdrantURL, q.cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := q.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	createBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     q.cfg.EmbeddingDim,
			"distance": "Cosine",
		},
	}
	data, _ := json.Marshal(createBody)
	req, err = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = q.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create collection failed %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

type qdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

func (q *QdrantClient) Upsert(ctx context.Context, points []qdrantPoint) error {
	if len(points) == 0 {
		return nil
	}
	body := map[string]interface{}{"points": points}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/collections/%s/points?wait=true", q.cfg.QdrantURL, q.cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upsert points failed %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

type searchResult struct {
	Result []struct {
		Payload map[string]interface{} `json:"payload"`
		Score   float64                `json:"score"`
	} `json:"result"`
}

func (q *QdrantClient) Search(ctx context.Context, vector []float32, limit int) ([]string, error) {
	body := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/collections/%s/points/search", q.cfg.QdrantURL, q.cfg.Collection)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed %d: %s", resp.StatusCode, string(respBody))
	}

	var result searchResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	chunks := make([]string, 0, len(result.Result))
	seen := make(map[string]bool)
	for _, hit := range result.Result {
		if text, ok := hit.Payload["text"].(string); ok && text != "" && !seen[text] {
			seen[text] = true
			chunks = append(chunks, text)
		}
	}
	return chunks, nil
}
