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

type openAIEmbedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type openAIEmbedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

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
	if e.cfg.UseOllamaEmbedding() {
		return e.embedOllama(ctx, text)
	}
	return e.embedOpenAICompatible(ctx, text)
}

func (e *Embedder) embedOpenAICompatible(ctx context.Context, text string) ([]float32, error) {
	url := openAIEmbeddingURL(e.cfg.EmbeddingBaseURL)
	body, err := json.Marshal(openAIEmbedRequest{
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
	if e.cfg.OpenAIAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.cfg.OpenAIAPIKey)
	}

	return e.doOpenAIRequest(req, url)
}

func (e *Embedder) embedOllama(ctx context.Context, text string) ([]float32, error) {
	base := strings.TrimRight(e.cfg.EmbeddingBaseURL, "/")
	// 去掉误配的 /v1 后缀（Ollama 不使用 OpenAI 风格路径）
	base = strings.TrimSuffix(base, "/v1")
	url := base + "/api/embed"

	model := e.cfg.OllamaEmbeddingModel
	if model == "" {
		model = e.cfg.EmbeddingModel
	}

	body, err := json.Marshal(ollamaEmbedRequest{Model: model, Input: text})
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
		return nil, fmt.Errorf("embedding API error %d (url=%s): %s", resp.StatusCode, url, string(respBody))
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

func (e *Embedder) doOpenAIRequest(req *http.Request, url string) ([]float32, error) {
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
		return nil, fmt.Errorf("embedding API error %d (url=%s): %s", resp.StatusCode, url, string(respBody))
	}

	var result openAIEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return result.Data[0].Embedding, nil
}

func openAIEmbeddingURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(base, "/embeddings") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/embeddings"
	}
	return base + "/v1/embeddings"
}
