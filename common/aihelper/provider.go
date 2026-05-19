package aihelper

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func NewModelFromEnv(ctx context.Context) (AIModel, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_TYPE"))) {
	case "ark":
		return NewOpenAIModel(ctx)
	case "ollama":
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		modelName := os.Getenv("OLLAMA_MODEL")
		if modelName == "" {
			return nil, fmt.Errorf("OLLAMA_MODEL is required when OPENAI_TYPE=ollama")
		}
		return NewOllamaModel(ctx, baseURL, modelName)
	default:
		return NewOpenAIModel(ctx)
	}
}

func NewAIHelperFromEnv(ctx context.Context, sessionID string) (*AIHelper, error) {
	m, err := NewModelFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(m, sessionID), nil
}
