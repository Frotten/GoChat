package evaluation

import (
	"GopherAI/common/code"
	evalCore "GopherAI/common/evaluation"
	"GopherAI/controller"
	"GopherAI/model"
	evalService "GopherAI/service/evaluation"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	ListDatasetsResponse struct {
		controller.Response
		Datasets []model.EvalDatasetInfo `json:"datasets,omitempty"`
	}

	DatasetCasesResponse struct {
		controller.Response
		Cases []evalCore.CaseDefinition `json:"cases,omitempty"`
	}

	CreateRunRequest struct {
		DatasetID string `json:"dataset_id" binding:"required"`
	}

	CreateRunResponse struct {
		controller.Response
		RunID string `json:"run_id,omitempty"`
	}

	ListRunsResponse struct {
		controller.Response
		Runs []model.EvalRunInfo `json:"runs,omitempty"`
	}

	GetRunResponse struct {
		controller.Response
		Run *model.EvalRunInfo `json:"run,omitempty"`
	}

	RunResultsResponse struct {
		controller.Response
		Results []model.EvalResult `json:"results,omitempty"`
	}
)

func ListDatasets(c *gin.Context) {
	res := new(ListDatasetsResponse)
	datasets, code_ := evalService.ListDatasets()
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.Datasets = datasets
	c.JSON(http.StatusOK, res)
}

func GetDatasetCases(c *gin.Context) {
	res := new(DatasetCasesResponse)
	datasetID := c.Param("id")
	cases, code_ := evalService.GetDatasetCases(datasetID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.Cases = cases
	c.JSON(http.StatusOK, res)
}

func CreateRun(c *gin.Context) {
	req := new(CreateRunRequest)
	res := new(CreateRunResponse)
	userName := c.GetString("userName")
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	runID, code_ := evalService.CreateRun(userName, req.DatasetID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.RunID = runID
	c.JSON(http.StatusOK, res)
}

func ListRuns(c *gin.Context) {
	res := new(ListRunsResponse)
	userName := c.GetString("userName")
	runs, code_ := evalService.ListRuns(userName)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.Runs = runs
	c.JSON(http.StatusOK, res)
}

func GetRun(c *gin.Context) {
	res := new(GetRunResponse)
	userName := c.GetString("userName")
	runID := c.Param("id")
	run, code_ := evalService.GetRun(userName, runID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.Run = run
	c.JSON(http.StatusOK, res)
}

func GetRunResults(c *gin.Context) {
	res := new(RunResultsResponse)
	userName := c.GetString("userName")
	runID := c.Param("id")
	results, code_ := evalService.GetRunResults(userName, runID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}
	res.Success()
	res.Results = results
	c.JSON(http.StatusOK, res)
}
