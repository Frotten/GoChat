package aihelper

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func NewModelFromEnv(ctx context.Context) (AIModel, error) {
	if openAIType() == "ollama" && strings.TrimSpace(os.Getenv("OLLAMA_MODEL")) == "" {
		return nil, fmt.Errorf("OLLAMA_MODEL is required when OPENAI_TYPE=ollama")
	}
	return NewAgentModel(ctx)
}

func NewAIHelperFromEnv(ctx context.Context, sessionID string) (*AIHelper, error) {
	m, err := NewModelFromEnv(ctx)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(m, sessionID), nil
}
