package evaluation

import (
	"context"
	"log"
	"time"

	"GopherAI/common/code"
	evalCore "GopherAI/common/evaluation"
	evalDao "GopherAI/dao/evaluation"
	"GopherAI/model"

	"github.com/google/uuid"
)

func ListDatasets() ([]model.EvalDatasetInfo, code.Code) {
	datasets, err := evalCore.ListDatasets()
	if err != nil {
		log.Println("ListDatasets error:", err)
		return nil, code.CodeServerBusy
	}
	out := make([]model.EvalDatasetInfo, 0, len(datasets))
	for _, ds := range datasets {
		out = append(out, model.EvalDatasetInfo{
			ID:          ds.ID,
			Name:        ds.Name,
			Description: ds.Description,
			Version:     ds.Version,
			CaseCount:   len(ds.Cases),
		})
	}
	return out, code.CodeSuccess
}

func GetDatasetCases(datasetID string) ([]evalCore.CaseDefinition, code.Code) {
	ds, err := evalCore.GetDataset(datasetID)
	if err != nil {
		return nil, code.CodeRecordNotFound
	}
	return ds.Cases, code.CodeSuccess
}

func CreateRun(userName, datasetID string) (string, code.Code) {
	ds, err := evalCore.GetDataset(datasetID)
	if err != nil {
		return "", code.CodeRecordNotFound
	}

	runID := uuid.New().String()
	run := &model.EvalRun{
		ID:          runID,
		UserName:    userName,
		DatasetID:   ds.ID,
		DatasetName: ds.Name,
		Status:      model.EvalRunStatusPending,
		TotalCases:  len(ds.Cases),
	}
	if err := evalDao.CreateRun(run); err != nil {
		log.Println("CreateRun error:", err)
		return "", code.CodeServerBusy
	}

	go executeRun(runID, userName, ds)
	return runID, code.CodeSuccess
}

func executeRun(runID, userName string, ds evalCore.DatasetDefinition) {
	ctx := context.Background()
	run, err := evalDao.GetRunByID(runID, userName)
	if err != nil {
		log.Println("executeRun GetRunByID error:", err)
		return
	}

	now := time.Now()
	run.Status = model.EvalRunStatusRunning
	run.StartedAt = &now
	if err := evalDao.UpdateRun(run); err != nil {
		log.Println("executeRun update running error:", err)
		return
	}

	runner, err := evalCore.NewRunner(ctx, nil)
	if err != nil {
		failRun(run, err.Error())
		return
	}
	run.ModelType = runner.Model.GetModelType()

	caseResults, summary, err := runner.RunDataset(ctx, ds)
	if err != nil {
		failRun(run, err.Error())
		return
	}

	dbResults := make([]model.EvalResult, 0, len(caseResults))
	for _, cr := range caseResults {
		dbResults = append(dbResults, model.EvalResult{
			ID:             uuid.New().String(),
			RunID:          runID,
			CaseID:         cr.CaseID,
			CaseName:       cr.CaseName,
			Category:       string(cr.Category),
			Input:          cr.Input,
			ActualOutput:   cr.ActualOutput,
			ExpectedOutput: cr.ExpectedOutput,
			Score:          cr.Score,
			Passed:         cr.Passed,
			LatencyMs:      cr.LatencyMs,
			Metrics:        toEvalCaseMetrics(cr.Metrics),
			ToolCalls:      toEvalToolCalls(cr.ToolCalls),
			RetrievalHits:  toEvalRetrievalHits(cr.RetrievalHits),
			RetrievalError: cr.RetrievalError,
			ErrorMessage:   cr.ErrorMessage,
		})
	}
	if err := evalDao.CreateResults(dbResults); err != nil {
		log.Println("executeRun CreateResults error:", err)
		failRun(run, err.Error())
		return
	}

	finished := time.Now()
	run.Status = model.EvalRunStatusCompleted
	run.TotalCases = summary.TotalCases
	run.PassedCases = summary.PassedCases
	run.FailedCases = summary.FailedCases
	run.AvgScore = summary.AvgScore
	run.AvgLatencyMs = summary.AvgLatencyMs
	run.AnswerCases = summary.AnswerCases
	run.AnswerAvgScore = summary.AnswerAvgScore
	run.ToolCases = summary.ToolCases
	run.ToolExactMatchRate = summary.ToolExactMatchRate
	run.ToolCallPrecision = summary.ToolCallPrecision
	run.ToolCallRecall = summary.ToolCallRecall
	run.ToolCallF1 = summary.ToolCallF1
	run.ToolArgumentAccuracy = summary.ToolArgumentAccuracy
	run.RAGCases = summary.RAGCases
	run.RAGHitRate = summary.RAGHitRate
	run.RAGRecallAtK = summary.RAGRecallAtK
	run.RAGPrecisionAtK = summary.RAGPrecisionAtK
	run.RAGMRR = summary.RAGMRR
	run.FinishedAt = &finished
	if err := evalDao.UpdateRun(run); err != nil {
		log.Println("executeRun update completed error:", err)
	}
}

func failRun(run *model.EvalRun, msg string) {
	finished := time.Now()
	run.Status = model.EvalRunStatusFailed
	run.ErrorMessage = msg
	run.FinishedAt = &finished
	if err := evalDao.UpdateRun(run); err != nil {
		log.Println("failRun error:", err)
	}
}

func GetRun(userName, runID string) (*model.EvalRunInfo, code.Code) {
	run, err := evalDao.GetRunByID(runID, userName)
	if err != nil {
		return nil, code.CodeRecordNotFound
	}
	return toRunInfo(run), code.CodeSuccess
}

func ListRuns(userName string) ([]model.EvalRunInfo, code.Code) {
	runs, err := evalDao.ListRunsByUser(userName, 50)
	if err != nil {
		log.Println("ListRuns error:", err)
		return nil, code.CodeServerBusy
	}
	out := make([]model.EvalRunInfo, 0, len(runs))
	for i := range runs {
		out = append(out, *toRunInfo(&runs[i]))
	}
	return out, code.CodeSuccess
}

func GetRunResults(userName, runID string) ([]model.EvalResult, code.Code) {
	if _, err := evalDao.GetRunByID(runID, userName); err != nil {
		return nil, code.CodeRecordNotFound
	}
	results, err := evalDao.ListResultsByRunID(runID)
	if err != nil {
		log.Println("GetRunResults error:", err)
		return nil, code.CodeServerBusy
	}
	return results, code.CodeSuccess
}

func toRunInfo(run *model.EvalRun) *model.EvalRunInfo {
	return &model.EvalRunInfo{
		ID:                   run.ID,
		DatasetID:            run.DatasetID,
		DatasetName:          run.DatasetName,
		ModelType:            run.ModelType,
		Status:               run.Status,
		TotalCases:           run.TotalCases,
		PassedCases:          run.PassedCases,
		FailedCases:          run.FailedCases,
		AvgScore:             run.AvgScore,
		AvgLatencyMs:         run.AvgLatencyMs,
		AnswerCases:          run.AnswerCases,
		AnswerAvgScore:       run.AnswerAvgScore,
		ToolCases:            run.ToolCases,
		ToolExactMatchRate:   run.ToolExactMatchRate,
		ToolCallPrecision:    run.ToolCallPrecision,
		ToolCallRecall:       run.ToolCallRecall,
		ToolCallF1:           run.ToolCallF1,
		ToolArgumentAccuracy: run.ToolArgumentAccuracy,
		RAGCases:             run.RAGCases,
		RAGHitRate:           run.RAGHitRate,
		RAGRecallAtK:         run.RAGRecallAtK,
		RAGPrecisionAtK:      run.RAGPrecisionAtK,
		RAGMRR:               run.RAGMRR,
		ErrorMessage:         run.ErrorMessage,
		StartedAt:            run.StartedAt,
		FinishedAt:           run.FinishedAt,
		CreatedAt:            run.CreatedAt,
	}
}

func toEvalCaseMetrics(metrics evalCore.CaseMetrics) model.EvalCaseMetrics {
	out := model.EvalCaseMetrics{AnswerScore: metrics.AnswerScore}
	if metrics.Tool != nil {
		m := metrics.Tool
		out.Tool = &model.EvalToolMetrics{
			ExpectedCalls: m.ExpectedCalls, ActualCalls: m.ActualCalls,
			MatchedNames: m.MatchedNames, MatchedCalls: m.MatchedCalls,
			NamePrecision: m.NamePrecision, NameRecall: m.NameRecall,
			CallPrecision: m.CallPrecision, CallRecall: m.CallRecall, CallF1: m.CallF1,
			ArgumentAccuracy: m.ArgumentAccuracy, OrderAccuracy: m.OrderAccuracy,
			ForbiddenCalls: m.ForbiddenCalls, ExactMatch: m.ExactMatch,
		}
	}
	if metrics.RAG != nil {
		m := metrics.RAG
		out.RAG = &model.EvalRAGMetrics{
			K: m.K, Hit: m.Hit, RecallAtK: m.RecallAtK,
			PrecisionAtK: m.PrecisionAtK, MRR: m.MRR,
		}
	}
	return out
}

func toEvalToolCalls(calls []evalCore.ToolInvocation) []model.EvalToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]model.EvalToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, model.EvalToolCall{Name: call.Name, Arguments: call.Arguments})
	}
	return out
}

func toEvalRetrievalHits(hits []evalCore.RetrievalHit) []model.EvalRetrievalHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]model.EvalRetrievalHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, model.EvalRetrievalHit{
			Text: hit.Text, Source: hit.Source, ChunkID: hit.ChunkID,
			Score: hit.Score, Rank: hit.Rank,
		})
	}
	return out
}
