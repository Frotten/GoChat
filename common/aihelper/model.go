package aihelper

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
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

type OpenAIModel struct {
	llm react.Agent
}

type OllamaModel struct {
	llm react.Agent
}

func newChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_TYPE"))) == "ark" {
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("OPENAI_API_KEY"),
			Model:   os.Getenv("OPENAI_MODEL"),
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
		})
	} else if strings.ToLower(strings.TrimSpace(os.Getenv("OPENAI_TYPE"))) == "ollama" {
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			Model:   os.Getenv("OLLAMA_MODEL"),
			BaseURL: os.Getenv("OLLAMA_BASE_URL"),
		})
	}
	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: os.Getenv("OPENAI_BY_AZURE") == "true",
	})
}

func llmToAgent(llm model.ToolCallingChatModel) *react.Agent {
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: llm,

		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{
				&GetCurrentTimeTool{},
				&SearchFilesTool{},
				&ReadFileTool{},
			},
		},
		MaxStep: 6,
	})
	if err != nil {
		return nil
	}
	return agent
}

func NewOpenAIModel(ctx context.Context) (*OpenAIModel, error) {
	llm, err := newChatModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create AI model failed: %v", err)
	}
	agent := llmToAgent(llm)
	if agent == nil {
		return nil, fmt.Errorf("create agent failed")
	}
	return &OpenAIModel{llm: *agent}, nil
}

func NewOllamaModel(ctx context.Context, baseURL, modelName string) (*OpenAIModel, error) {
	llm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	})
	if err != nil {
		return nil, fmt.Errorf("create ollama model failed: %v", err)
	}
	agent := llmToAgent(llm)
	if agent == nil {
		return nil, fmt.Errorf("create agent failed")
	}
	return &OpenAIModel{llm: *agent}, nil
}

func (o *OpenAIModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("openai generate failed: %v", err)
	}
	return resp, nil
}

func (o *OpenAIModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("openai stream failed: %v", err)
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("openai stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content) // 聚合
			cb(msg.Content)                   // 实时调用cb函数，方便主动发送给前端
		}
	}
	return fullResp.String(), nil //返回完整内容，方便后续存储
}

func (o *OpenAIModel) GetModelType() string { return "openai" }

func (o *OllamaModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	resp, err := o.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ollama generate failed: %v", err)
	}
	return resp, nil
}

func (o *OllamaModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := o.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("ollama stream failed: %v", err)
	}
	defer stream.Close()
	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("openai stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}
	return fullResp.String(), nil
}

func (o *OllamaModel) GetModelType() string { return "ollama" }
