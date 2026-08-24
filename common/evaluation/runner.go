package evaluation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"GopherAI/common/aihelper"
	"GopherAI/common/rag"

	"github.com/cloudwego/eino/schema"
)

const defaultEvaluationRAGTopK = 5

// Runner executes datasets against the production model, prompt and retrieval path.
type Runner struct {
	Model     aihelper.AIModel
	Retriever RetrievalProvider
}

func NewRunner(ctx context.Context, model aihelper.AIModel) (*Runner, error) {
	if model == nil {
		m, err := aihelper.NewModelFromEnv(ctx)
		if err != nil {
			return nil, err
		}
		model = m
	}
	return &Runner{Model: model, Retriever: rag.GetService()}, nil
}

func (r *Runner) RunDataset(ctx context.Context, ds DatasetDefinition) ([]CaseResult, RunSummary, error) {
	if err := ValidateDataset(ds); err != nil {
		return nil, RunSummary{}, err
	}
	results := make([]CaseResult, 0, len(ds.Cases))
	for _, c := range ds.Cases {
		if err := ctx.Err(); err != nil {
			return results, summarize(results), err
		}
		results = append(results, r.runCase(ctx, c))
	}
	return results, summarize(results), nil
}

func (r *Runner) runCase(ctx context.Context, c CaseDefinition) CaseResult {
	category := normalizeCategory(c.Category)
	result := CaseResult{
		CaseID: c.ID, CaseName: c.Name, Category: category,
		Input: c.Input, ExpectedOutput: c.ExpectedOutput,
	}

	start := time.Now()
	caseCtx, trace := aihelper.WithToolTrace(ctx)
	topK := c.RAGTopK
	if topK <= 0 {
		topK = defaultEvaluationRAGTopK
	}
	if r.Retriever != nil {
		hits, err := r.Retriever.RetrieveHits(caseCtx, c.Input, topK)
		if err != nil {
			result.RetrievalError = err.Error()
		} else {
			result.RetrievalHits = hits
		}
	}

	resp, err := r.Model.GenerateResponse(caseCtx, []*schema.Message{
		aihelper.BuildSystemPrompt(formatRetrievalContext(result.RetrievalHits)),
		{Role: schema.User, Content: c.Input},
	})
	result.LatencyMs = time.Since(start).Milliseconds()
	result.ToolCalls = trace.Calls()
	if err != nil {
		result.ErrorMessage = err.Error()
		return result
	}
	if resp != nil {
		result.ActualOutput = resp.Content
	}

	answerScore, _ := Score(defaultScoreType(c.ScoreType), result.ActualOutput, c.ExpectedOutput, c.ExpectedKeywords)
	result.Metrics.AnswerScore = answerScore
	switch category {
	case CategoryToolCall:
		metrics := ScoreToolCalls(c.ExpectedTools, c.ForbiddenTools, result.ToolCalls)
		result.Metrics.Tool = &metrics
		result.Score = metrics.CallF1
		if c.PassThreshold > 0 {
			result.Passed = metrics.CallF1 >= c.PassThreshold && metrics.ForbiddenCalls == 0
		} else {
			result.Passed = metrics.ExactMatch
		}
	case CategoryRAG:
		metrics := ScoreRAG(c.ExpectedSources, result.RetrievalHits, topK)
		result.Metrics.RAG = &metrics
		result.Score = metrics.RecallAtK
		threshold := c.PassThreshold
		if threshold <= 0 {
			threshold = 1
		}
		result.Passed = result.RetrievalError == "" && metrics.Hit && metrics.RecallAtK >= threshold
	default:
		result.Score, result.Passed = Score(defaultScoreType(c.ScoreType), result.ActualOutput, c.ExpectedOutput, c.ExpectedKeywords)
	}
	return result
}

func summarize(results []CaseResult) RunSummary {
	summary := RunSummary{TotalCases: len(results)}
	var totalScore float64
	var totalLatency int64
	var answerScore, toolExact, toolPrecision, toolRecall, toolF1, toolArgs float64
	var ragHits, ragRecall, ragPrecision, ragMRR float64
	for _, result := range results {
		totalScore += result.Score
		totalLatency += result.LatencyMs
		if result.Passed {
			summary.PassedCases++
		} else {
			summary.FailedCases++
		}
		switch result.Category {
		case CategoryToolCall:
			summary.ToolCases++
			if result.Metrics.Tool != nil {
				if result.Metrics.Tool.ExactMatch {
					toolExact++
				}
				toolPrecision += result.Metrics.Tool.CallPrecision
				toolRecall += result.Metrics.Tool.CallRecall
				toolF1 += result.Metrics.Tool.CallF1
				toolArgs += result.Metrics.Tool.ArgumentAccuracy
			}
		case CategoryRAG:
			summary.RAGCases++
			if result.Metrics.RAG != nil {
				if result.Metrics.RAG.Hit {
					ragHits++
				}
				ragRecall += result.Metrics.RAG.RecallAtK
				ragPrecision += result.Metrics.RAG.PrecisionAtK
				ragMRR += result.Metrics.RAG.MRR
			}
		default:
			summary.AnswerCases++
			answerScore += result.Metrics.AnswerScore
		}
	}
	if summary.TotalCases > 0 {
		summary.AvgScore = totalScore / float64(summary.TotalCases)
		summary.AvgLatencyMs = totalLatency / int64(summary.TotalCases)
	}
	if summary.AnswerCases > 0 {
		summary.AnswerAvgScore = answerScore / float64(summary.AnswerCases)
	}
	if summary.ToolCases > 0 {
		count := float64(summary.ToolCases)
		summary.ToolExactMatchRate = toolExact / count
		summary.ToolCallPrecision = toolPrecision / count
		summary.ToolCallRecall = toolRecall / count
		summary.ToolCallF1 = toolF1 / count
		summary.ToolArgumentAccuracy = toolArgs / count
	}
	if summary.RAGCases > 0 {
		count := float64(summary.RAGCases)
		summary.RAGHitRate = ragHits / count
		summary.RAGRecallAtK = ragRecall / count
		summary.RAGPrecisionAtK = ragPrecision / count
		summary.RAGMRR = ragMRR / count
	}
	return summary
}

func formatRetrievalContext(hits []rag.RetrievalHit) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	for _, hit := range hits {
		fmt.Fprintf(&b, "[%d] 来源: %s#%d\n%s\n\n", hit.Rank, hit.Source, hit.ChunkID, hit.Text)
	}
	return strings.TrimSpace(b.String())
}

func normalizeCategory(category CaseCategory) CaseCategory {
	switch category {
	case CategoryToolCall, CategoryRAG:
		return category
	default:
		return CategoryAnswer
	}
}

func defaultScoreType(scoreType ScoreType) ScoreType {
	if scoreType == "" {
		return ScoreTypeNone
	}
	return scoreType
}
