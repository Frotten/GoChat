package evaluation

import (
	"context"
	"testing"

	"GopherAI/common/aihelper"
	"GopherAI/common/rag"

	"github.com/cloudwego/eino/schema"
)

type stubAIModel struct {
	replies map[string]string
}

type stubRetriever struct {
	hits []rag.RetrievalHit
}

func (s *stubRetriever) RetrieveHits(context.Context, string, int) ([]rag.RetrievalHit, error) {
	return s.hits, nil
}

func (s *stubAIModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	content := messages[len(messages)-1].Content
	if reply, ok := s.replies[content]; ok {
		return &schema.Message{Content: reply}, nil
	}
	return &schema.Message{Content: "default reply"}, nil
}

func TestRunnerCapturesToolAndRAGMetrics(t *testing.T) {
	model := &stubAIModel{replies: map[string]string{
		"format": "formatted",
		"babel":  "巴别意为变乱",
	}}
	traced := &recordingModel{AIModel: model}
	runner := &Runner{
		Model:     traced,
		Retriever: &stubRetriever{hits: []rag.RetrievalHit{{Source: "巴别塔.txt", Rank: 1, Score: 0.9}}},
	}
	ds := DatasetDefinition{ID: "metrics", Name: "metrics", Cases: []CaseDefinition{
		{
			ID: "tool", Name: "tool", Category: CategoryToolCall, Input: "format",
			ExpectedTools: []ToolCallExpectation{{Name: "format_go_code", Arguments: map[string]any{"code": "x"}}},
		},
		{
			ID: "rag", Name: "rag", Category: CategoryRAG, Input: "babel",
			ExpectedSources: []string{"巴别塔.txt"}, RAGTopK: 1,
		},
	}}
	results, summary, err := runner.RunDataset(context.Background(), ds)
	if err != nil {
		t.Fatalf("RunDataset failed: %v", err)
	}
	if len(results) != 2 || !results[0].Passed || !results[1].Passed {
		t.Fatalf("unexpected results: %+v", results)
	}
	if summary.ToolExactMatchRate != 1 || summary.RAGHitRate != 1 || summary.RAGMRR != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

type recordingModel struct {
	aihelper.AIModel
}

func (m *recordingModel) GenerateResponse(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	if messages[len(messages)-1].Content == "format" {
		aihelper.RecordToolInvocation(ctx, "format_go_code", `{"code":"x"}`)
	}
	return m.AIModel.GenerateResponse(ctx, messages)
}

func (s *stubAIModel) StreamResponse(ctx context.Context, messages []*schema.Message, cb aihelper.StreamCallback) (string, error) {
	msg, err := s.GenerateResponse(ctx, messages)
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

func (s *stubAIModel) GetModelType() string { return "stub" }

func TestRunnerRunDataset(t *testing.T) {
	ds := DatasetDefinition{
		ID:   "test",
		Name: "test",
		Cases: []CaseDefinition{
			{ID: "c1", Name: "greet", Input: "hi", ScoreType: ScoreTypeNone},
			{ID: "c2", Name: "time", Input: "time", ExpectedKeywords: []string{"12"}, ScoreType: ScoreTypeKeywords},
		},
	}
	model := &stubAIModel{
		replies: map[string]string{
			"hi":   "hello there",
			"time": "now is 12:00",
		},
	}
	runner := &Runner{Model: model}
	results, summary, err := runner.RunDataset(context.Background(), ds)
	if err != nil {
		t.Fatalf("RunDataset failed: %v", err)
	}
	if len(results) != 2 || summary.PassedCases != 2 {
		t.Fatalf("unexpected results: %+v summary: %+v", results, summary)
	}
}
