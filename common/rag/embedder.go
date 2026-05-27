package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

type Embedder struct {
	cfg    Config
	client *http.Client
}

func NewEmbedder(cfg Config) *Embedder {
	return &Embedder{
		cfg: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	base := strings.TrimRight(e.cfg.EmbeddingBaseURL, "/")
	base = strings.TrimSuffix(base, "/v1")
	url := base + "/api/embed"

	body, err := json.Marshal(ollamaEmbedRequest{
		Model: e.cfg.EmbeddingModel,
		Input: text,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embedding API error %d (url=%s): %s", resp.StatusCode, url, string(respBody))
	}

	var result ollamaEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if len(result.Embeddings) == 0 || len(result.Embeddings[0]) == 0 {
		return nil, fmt.Errorf("empty ollama embedding response")
	}
	return result.Embeddings[0], nil
}
