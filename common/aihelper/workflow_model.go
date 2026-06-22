package aihelper

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type WorkflowAgentModel struct {
	runner    compose.Runnable[[]*schema.Message, *schema.Message]
	modelType string
}

func NewWorkflowAgentModel(ctx context.Context, base AIModel) (*WorkflowAgentModel, error) {
	wf := compose.NewWorkflow[[]*schema.Message, *schema.Message]()

	normalizeMessagesNode, err := compose.AnyLambda[[]*schema.Message, []*schema.Message, any](
		func(ctx context.Context, input []*schema.Message, opts ...any) ([]*schema.Message, error) {
			return normalizeMessages(input), nil
		},
		func(ctx context.Context, input []*schema.Message, opts ...any) (*schema.StreamReader[[]*schema.Message], error) {
			return schema.StreamReaderFromArray([][]*schema.Message{normalizeMessages(input)}), nil
		},
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build normalize node failed: %w", err)
	}

	wf.AddLambdaNode("normalize_messages", normalizeMessagesNode).AddInput(compose.START)

	agentNode, err := compose.AnyLambda[[]*schema.Message, *schema.Message, any](
		func(ctx context.Context, input []*schema.Message, opts ...any) (*schema.Message, error) {
			return base.GenerateResponse(ctx, input)
		},
		func(ctx context.Context, input []*schema.Message, opts ...any) (*schema.StreamReader[*schema.Message], error) {
			return streamModelResponse(ctx, base, input)
		},
		nil,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("build agent node failed: %w", err)
	}

	wf.AddLambdaNode("agent", agentNode).AddInput("normalize_messages")

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

func streamModelResponse(ctx context.Context, base AIModel, messages []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()

		_, err := base.StreamResponse(ctx, messages, func(chunk string) {
			if chunk == "" {
				return
			}
			sw.Send(&schema.Message{Content: chunk}, nil)
		})
		if err != nil {
			sw.Send(nil, err)
			return
		}
	}()

	return sr, nil
}

func (m *WorkflowAgentModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return m.runner.Invoke(ctx, messages)
}

func (m *WorkflowAgentModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	stream, err := m.runner.Stream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullResp strings.Builder
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if msg != nil && len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			cb(msg.Content)
		}
	}

	return fullResp.String(), nil
}

func (m *WorkflowAgentModel) GetModelType() string {
	return m.modelType
}
