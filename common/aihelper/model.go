package aihelper

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type StreamCallback func(msg string)

// AIModel 定义AI模型接口
type AIModel interface {
	GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error)
	GetModelType() string
}

// AgentModel 统一的 Eino ReAct Agent 封装，底层可为 OpenAI 兼容 API 或火山方舟。
type AgentModel struct {
	llm       react.Agent
	modelType string
}

func openAIType() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_TYPE")))
}

func newChatModel(ctx context.Context) (model.ToolCallingChatModel, string, error) {
	switch openAIType() {
	case "ark":
		llm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   os.Getenv("OPENAI_MODEL"),
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
		})
		return llm, "ark", err
	default:
		llm, err := openai.NewChatModel(ctx, openaiCompatibleConfig())
		return llm, resolveModelType(), err
	}
}

// openaiCompatibleConfig 构建 OpenAI 兼容客户端配置。
// Ollama 通过 /v1 端点接入；DeepSeek 等思考模式模型默认关闭 thinking 以保证 ReAct 工具链可用。
func openaiCompatibleConfig() *openai.ChatModelConfig {
	if openAIType() == "ollama" {
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return &openai.ChatModelConfig{
			BaseURL: strings.TrimRight(baseURL, "/") + "/v1",
			APIKey:  "ollama",
			Model:   os.Getenv("OLLAMA_MODEL"),
		}
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	cfg := &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: baseURL,
		ByAzure: os.Getenv("OPENAI_BY_AZURE") == "true",
	}
	if shouldDisableThinking(baseURL) {
		cfg.ExtraFields = map[string]any{
			"thinking": map[string]string{"type": "disabled"},
		}
	}
	return cfg
}

func resolveModelType() string {
	switch openAIType() {
	case "ollama":
		return "ollama"
	case "ark":
		return "ark"
	default:
		return "openai"
	}
}

func shouldDisableThinking(baseURL string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_THINKING"))) {
	case "enabled":
		return false
	case "disabled":
		return true
	}
	return strings.Contains(strings.ToLower(baseURL), "deepseek.com")
}

func llmToAgent(ctx context.Context, llm model.ToolCallingChatModel) (*react.Agent, error) {
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: llm,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{
				&GetCurrentTimeTool{},
				&FormatGoCodeTool{},
				&SearchFilesTool{},
				&RenameFileTool{},
				&CreateFileTool{},
				&ReadFileTool{},
				&EditFileTool{},
			},
		},
		MaxStep: 15,
	})
	if err != nil {
		return nil, fmt.Errorf("create agent failed: %w", err)
	}
	return agent, nil
}

func NewAgentModel(ctx context.Context) (*AgentModel, error) {
	llm, modelType, err := newChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create chat model failed: %w", err)
	}
	agent, err := llmToAgent(ctx, llm)
	if err != nil {
		return nil, err
	}
	return &AgentModel{llm: *agent, modelType: modelType}, nil
}

func (m *AgentModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := m.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("%s generate failed: %w", m.modelType, err)
	}
	return resp, nil
}

func (m *AgentModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := m.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("%s stream failed: %w", m.modelType, err)
	}
	defer stream.Close()

	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%s stream recv failed: %w", m.modelType, err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (m *AgentModel) GetModelType() string {
	return m.modelType
}
