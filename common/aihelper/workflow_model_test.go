package aihelper

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"
)

type fakeAIModel struct {
	modelType string
	generate  func(ctx context.Context, messages []*schema.Message) (*schema.Message, error)
	stream    func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error)
}

func (f *fakeAIModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return f.generate(ctx, messages)
}

func (f *fakeAIModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	return f.stream(ctx, messages, cb)
}

func (f *fakeAIModel) GetModelType() string {
	return f.modelType
}

func TestWorkflowAgentModelGenerateResponse(t *testing.T) {
	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			return &schema.Message{Content: messages[0].Content + "-reply"}, nil
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			return "", nil
		},
	}

	model, err := NewWorkflowAgentModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewWorkflowAgentModel failed: %v", err)
	}

	resp, err := model.GenerateResponse(context.Background(), []*schema.Message{
		nil,
		{Role: schema.User, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("GenerateResponse failed: %v", err)
	}

	if resp.Content != "hello-reply" {
		t.Fatalf("unexpected response: %s", resp.Content)
	}
}

func TestWorkflowAgentModelStreamResponse(t *testing.T) {
	base := &fakeAIModel{
		modelType: "fake",
		generate: func(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
			return &schema.Message{Content: "hello"}, nil
		},
		stream: func(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
			cb("he")
			cb("llo")
			return "hello", nil
		},
	}

	model, err := NewWorkflowAgentModel(context.Background(), base)
	if err != nil {
		t.Fatalf("NewWorkflowAgentModel failed: %v", err)
	}

	var chunks []string
	resp, err := model.StreamResponse(context.Background(), []*schema.Message{
		{Role: schema.User, Content: "hello"},
	}, func(chunk string) {
		chunks = append(chunks, chunk)
	})
	if err != nil {
		t.Fatalf("StreamResponse failed: %v", err)
	}

	if resp != "hello" {
		t.Fatalf("unexpected streamed response: %s", resp)
	}

	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}
