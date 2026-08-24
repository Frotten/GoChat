package evaluation

import (
	"testing"

	"GopherAI/common/aihelper"
	"GopherAI/common/rag"
)

func TestScoreExact(t *testing.T) {
	score, passed := Score(ScoreTypeExact, "Hello", "hello", nil)
	if !passed || score != 1 {
		t.Fatalf("expected exact match pass, got score=%v passed=%v", score, passed)
	}
}

func TestScoreToolCallsChecksNamesArgumentsOrderAndExtras(t *testing.T) {
	expected := []ToolCallExpectation{
		{Name: "search_files", Arguments: map[string]any{"keyword": "report"}},
		{Name: "read_file", Arguments: map[string]any{"file_path": "work/report.txt"}},
	}
	actual := []aihelper.ToolInvocation{
		{Name: "search_files", Arguments: `{"keyword":"report","limit":5}`},
		{Name: "read_file", Arguments: `{"file_path":"work/report.txt"}`},
	}
	metrics := ScoreToolCalls(expected, nil, actual)
	if !metrics.ExactMatch || metrics.CallF1 != 1 || metrics.ArgumentAccuracy != 1 || metrics.OrderAccuracy != 1 {
		t.Fatalf("expected exact tool match, got %+v", metrics)
	}

	actual[1].Arguments = `{"file_path":"work/other.txt"}`
	metrics = ScoreToolCalls(expected, nil, actual)
	if metrics.ExactMatch || metrics.MatchedCalls != 1 || metrics.CallRecall != 0.5 {
		t.Fatalf("expected argument mismatch, got %+v", metrics)
	}
}

func TestScoreRAGUsesSourceRecallPrecisionAndMRR(t *testing.T) {
	hits := []rag.RetrievalHit{
		{Source: "other.txt", Rank: 1},
		{Source: "Info/巴别塔.txt", Rank: 2},
		{Source: "世界语.txt", Rank: 3},
	}
	metrics := ScoreRAG([]string{"巴别塔.txt", "世界语.txt"}, hits, 3)
	if !metrics.Hit || metrics.RecallAtK != 1 || metrics.PrecisionAtK != 2.0/3.0 || metrics.MRR != 0.5 {
		t.Fatalf("unexpected RAG metrics: %+v", metrics)
	}
}

func TestScoreContains(t *testing.T) {
	score, passed := Score(ScoreTypeContains, "当前时间是 12:00", "12:00", nil)
	if !passed || score != 1 {
		t.Fatalf("expected contains pass, got score=%v passed=%v", score, passed)
	}
}

func TestScoreKeywords(t *testing.T) {
	score, passed := Score(ScoreTypeKeywords, "HTTP 服务已启动", "", []string{"HTTP", "服务"})
	if !passed || score != 1 {
		t.Fatalf("expected keywords pass, got score=%v passed=%v", score, passed)
	}
}

func TestScoreNone(t *testing.T) {
	score, passed := Score(ScoreTypeNone, "任意回复", "", nil)
	if !passed || score != 1 {
		t.Fatalf("expected none pass for non-empty output, got score=%v passed=%v", score, passed)
	}
	_, passed = Score(ScoreTypeNone, "", "", nil)
	if passed {
		t.Fatal("expected none fail for empty output")
	}
}
