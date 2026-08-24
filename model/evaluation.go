package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	EvalRunStatusPending   = "pending"
	EvalRunStatusRunning   = "running"
	EvalRunStatusCompleted = "completed"
	EvalRunStatusFailed    = "failed"
)

// EvalRun 记录一次 Agent 测评运行。
type EvalRun struct {
	ID                   string         `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserName             string         `gorm:"index;not null" json:"username"`
	DatasetID            string         `gorm:"type:varchar(64);not null" json:"dataset_id"`
	DatasetName          string         `gorm:"type:varchar(128)" json:"dataset_name"`
	ModelType            string         `gorm:"type:varchar(64)" json:"model_type"`
	Status               string         `gorm:"type:varchar(20);index;not null" json:"status"`
	TotalCases           int            `json:"total_cases"`
	PassedCases          int            `json:"passed_cases"`
	FailedCases          int            `json:"failed_cases"`
	AvgScore             float64        `json:"avg_score"`
	AvgLatencyMs         int64          `json:"avg_latency_ms"`
	AnswerCases          int            `json:"answer_cases"`
	AnswerAvgScore       float64        `json:"answer_avg_score"`
	ToolCases            int            `json:"tool_cases"`
	ToolExactMatchRate   float64        `json:"tool_exact_match_rate"`
	ToolCallPrecision    float64        `json:"tool_call_precision"`
	ToolCallRecall       float64        `json:"tool_call_recall"`
	ToolCallF1           float64        `json:"tool_call_f1"`
	ToolArgumentAccuracy float64        `json:"tool_argument_accuracy"`
	RAGCases             int            `json:"rag_cases"`
	RAGHitRate           float64        `json:"rag_hit_rate"`
	RAGRecallAtK         float64        `json:"rag_recall_at_k"`
	RAGPrecisionAtK      float64        `json:"rag_precision_at_k"`
	RAGMRR               float64        `json:"rag_mrr"`
	ErrorMessage         string         `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt            *time.Time     `json:"started_at,omitempty"`
	FinishedAt           *time.Time     `json:"finished_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

// EvalResult 记录单条测评用例的执行结果。
type EvalResult struct {
	ID             string             `gorm:"primaryKey;type:varchar(36)" json:"id"`
	RunID          string             `gorm:"index;type:varchar(36);not null" json:"run_id"`
	CaseID         string             `gorm:"type:varchar(64)" json:"case_id"`
	CaseName       string             `gorm:"type:varchar(128)" json:"case_name"`
	Category       string             `gorm:"type:varchar(20);index" json:"category"`
	Input          string             `gorm:"type:text" json:"input"`
	ActualOutput   string             `gorm:"type:longtext" json:"actual_output"`
	ExpectedOutput string             `gorm:"type:text" json:"expected_output"`
	Score          float64            `json:"score"`
	Passed         bool               `json:"passed"`
	LatencyMs      int64              `json:"latency_ms"`
	Metrics        EvalCaseMetrics    `gorm:"serializer:json;type:longtext" json:"metrics"`
	ToolCalls      []EvalToolCall     `gorm:"serializer:json;type:longtext" json:"tool_calls,omitempty"`
	RetrievalHits  []EvalRetrievalHit `gorm:"serializer:json;type:longtext" json:"retrieval_hits,omitempty"`
	RetrievalError string             `gorm:"type:text" json:"retrieval_error,omitempty"`
	ErrorMessage   string             `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
	DeletedAt      gorm.DeletedAt     `gorm:"index" json:"-"`
}

type EvalToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type EvalRetrievalHit struct {
	Text    string  `json:"text"`
	Source  string  `json:"source"`
	ChunkID int     `json:"chunk_id"`
	Score   float64 `json:"score"`
	Rank    int     `json:"rank"`
}

type EvalToolMetrics struct {
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

type EvalRAGMetrics struct {
	K            int     `json:"k"`
	Hit          bool    `json:"hit"`
	RecallAtK    float64 `json:"recall_at_k"`
	PrecisionAtK float64 `json:"precision_at_k"`
	MRR          float64 `json:"mrr"`
}

type EvalCaseMetrics struct {
	AnswerScore float64          `json:"answer_score"`
	Tool        *EvalToolMetrics `json:"tool,omitempty"`
	RAG         *EvalRAGMetrics  `json:"rag,omitempty"`
}

type EvalRunInfo struct {
	ID                   string     `json:"id"`
	DatasetID            string     `json:"dataset_id"`
	DatasetName          string     `json:"dataset_name"`
	ModelType            string     `json:"model_type"`
	Status               string     `json:"status"`
	TotalCases           int        `json:"total_cases"`
	PassedCases          int        `json:"passed_cases"`
	FailedCases          int        `json:"failed_cases"`
	AvgScore             float64    `json:"avg_score"`
	AvgLatencyMs         int64      `json:"avg_latency_ms"`
	AnswerCases          int        `json:"answer_cases"`
	AnswerAvgScore       float64    `json:"answer_avg_score"`
	ToolCases            int        `json:"tool_cases"`
	ToolExactMatchRate   float64    `json:"tool_exact_match_rate"`
	ToolCallPrecision    float64    `json:"tool_call_precision"`
	ToolCallRecall       float64    `json:"tool_call_recall"`
	ToolCallF1           float64    `json:"tool_call_f1"`
	ToolArgumentAccuracy float64    `json:"tool_argument_accuracy"`
	RAGCases             int        `json:"rag_cases"`
	RAGHitRate           float64    `json:"rag_hit_rate"`
	RAGRecallAtK         float64    `json:"rag_recall_at_k"`
	RAGPrecisionAtK      float64    `json:"rag_precision_at_k"`
	RAGMRR               float64    `json:"rag_mrr"`
	ErrorMessage         string     `json:"error_message,omitempty"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
}

type EvalDatasetInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	CaseCount   int    `json:"case_count"`
}
