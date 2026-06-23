package aihelper

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type WorkflowAgentModel struct {
	runner    compose.Runnable[[]*schema.Message, *schema.Message]
	modelType string
}

func (m *WorkflowAgentModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return m.runner.Invoke(ctx, messages) //触发工作流上的节点函数
}

func (m *WorkflowAgentModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	resp, err := m.GenerateResponse(ctx, messages)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	if resp.Content != "" {
		cb(resp.Content)
	}
	return resp.Content, nil
}

func (m *WorkflowAgentModel) GetModelType() string {
	return m.modelType
}

func NewWorkflowAgentModel(ctx context.Context, base AIModel) (*WorkflowAgentModel, error) {
	wf := compose.NewWorkflow[[]*schema.Message, *schema.Message]()

	wf.AddLambdaNode("normalize_messages", compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		return normalizeMessages(input), nil
	})).AddInput(compose.START)

	wf.AddLambdaNode("agent", compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (*schema.Message, error) {
		return base.GenerateResponse(ctx, input)
	})).AddInput("normalize_messages")

	wf.End().AddInput("agent")
	runner, err := wf.Compile(ctx, compose.WithGraphName("AIHelperWorkflow"))
	if err != nil {
		return nil, fmt.Errorf("compile workflow agent failed: %w", err)
	}
	return &WorkflowAgentModel{
		runner:    runner,
		modelType: base.GetModelType(),
	}, nil
}

func normalizeMessages(input []*schema.Message) []*schema.Message {
	if len(input) == 0 {
		return nil
	}
	messages := make([]*schema.Message, 0, len(input))
	for _, msg := range input {
		if msg != nil {
			messages = append(messages, msg)
		}
	}
	return messages
}
