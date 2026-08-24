package evaluation

import (
	"context"

	"GopherAI/common/aihelper"
	"GopherAI/common/rag"
)

type ScoreType string

const (
	ScoreTypeNone     ScoreType = "none"
	ScoreTypeExact    ScoreType = "exact"
	ScoreTypeContains ScoreType = "contains"
	ScoreTypeKeywords ScoreType = "keywords"
)

type CaseCategory string

const (
	CategoryAnswer   CaseCategory = "answer"
	CategoryToolCall CaseCategory = "tool_call"
	CategoryRAG      CaseCategory = "rag"
)

// ToolCallExpectation matches a tool name and an expected JSON argument subset.
type ToolCallExpectation struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type ToolInvocation = aihelper.ToolInvocation
type RetrievalHit = rag.RetrievalHit

type CaseDefinition struct {
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Category         CaseCategory          `json:"category,omitempty"`
	Input            string                `json:"input"`
	ExpectedOutput   string                `json:"expected_output"`
	ExpectedKeywords []string              `json:"expected_keywords"`
	ScoreType        ScoreType             `json:"score_type"`
	ExpectedTools    []ToolCallExpectation `json:"expected_tools,omitempty"`
	ForbiddenTools   []string              `json:"forbidden_tools,omitempty"`
	ExpectedSources  []string              `json:"expected_sources,omitempty"`
	RAGTopK          int                   `json:"rag_top_k,omitempty"`
	PassThreshold    float64               `json:"pass_threshold,omitempty"`
}

type DatasetDefinition struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Version     string           `json:"version"`
	Cases       []CaseDefinition `json:"cases"`
}

type ToolMetrics struct {
	ExpectedCalls    int     `json:"expected_calls"`
	ActualCalls      int     `json:"actual_calls"`
	MatchedNames     int     `json:"matched_names"`
	MatchedCalls     int     `json:"matched_calls"`
	NamePrecision    float64 `json:"name_precision"`
	NameRecall       float64 `json:"name_recall"`
	CallPrecision    float64 `json:"call_precision"`
	CallRecall       float64 `json:"call_recall"`
	CallF1           float64 `json:"call_f1"`
	ArgumentAccuracy float64 `json:"argument_accuracy"`
	OrderAccuracy    float64 `json:"order_accuracy"`
	ForbiddenCalls   int     `json:"forbidden_calls"`
	ExactMatch       bool    `json:"exact_match"`
}

type RAGMetrics struct {
	K            int     `json:"k"`
	Hit          bool    `json:"hit"`
	RecallAtK    float64 `json:"recall_at_k"`
	PrecisionAtK float64 `json:"precision_at_k"`
	MRR          float64 `json:"mrr"`
}

type CaseMetrics struct {
	AnswerScore float64      `json:"answer_score"`
	Tool        *ToolMetrics `json:"tool,omitempty"`
	RAG         *RAGMetrics  `json:"rag,omitempty"`
}

type CaseResult struct {
	CaseID         string                    `json:"case_id"`
	CaseName       string                    `json:"case_name"`
	Category       CaseCategory              `json:"category"`
	Input          string                    `json:"input"`
	ActualOutput   string                    `json:"actual_output"`
	ExpectedOutput string                    `json:"expected_output"`
	Score          float64                   `json:"score"`
	Passed         bool                      `json:"passed"`
	LatencyMs      int64                     `json:"latency_ms"`
	Metrics        CaseMetrics               `json:"metrics"`
	ToolCalls      []aihelper.ToolInvocation `json:"tool_calls,omitempty"`
	RetrievalHits  []rag.RetrievalHit        `json:"retrieval_hits,omitempty"`
	RetrievalError string                    `json:"retrieval_error,omitempty"`
	ErrorMessage   string                    `json:"error_message,omitempty"`
}

type RunSummary struct {
	TotalCases           int     `json:"total_cases"`
	PassedCases          int     `json:"passed_cases"`
	FailedCases          int     `json:"failed_cases"`
	AvgScore             float64 `json:"avg_score"`
	AvgLatencyMs         int64   `json:"avg_latency_ms"`
	AnswerCases          int     `json:"answer_cases"`
	AnswerAvgScore       float64 `json:"answer_avg_score"`
	ToolCases            int     `json:"tool_cases"`
	ToolExactMatchRate   float64 `json:"tool_exact_match_rate"`
	ToolCallPrecision    float64 `json:"tool_call_precision"`
	ToolCallRecall       float64 `json:"tool_call_recall"`
	ToolCallF1           float64 `json:"tool_call_f1"`
	ToolArgumentAccuracy float64 `json:"tool_argument_accuracy"`
	RAGCases             int     `json:"rag_cases"`
	RAGHitRate           float64 `json:"rag_hit_rate"`
	RAGRecallAtK         float64 `json:"rag_recall_at_k"`
	RAGPrecisionAtK      float64 `json:"rag_precision_at_k"`
	RAGMRR               float64 `json:"rag_mrr"`
}

type RetrievalProvider interface {
	RetrieveHits(ctx context.Context, query string, limit int) ([]rag.RetrievalHit, error)
}
