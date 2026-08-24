package router

import (
	"GopherAI/controller/evaluation"

	"github.com/gin-gonic/gin"
)

func EvalRouter(r *gin.RouterGroup) {
	r.GET("/datasets", evaluation.ListDatasets)
	r.GET("/datasets/:id/cases", evaluation.GetDatasetCases)
	r.POST("/runs", evaluation.CreateRun)
	r.GET("/runs", evaluation.ListRuns)
	r.GET("/runs/:id", evaluation.GetRun)
	r.GET("/runs/:id/results", evaluation.GetRunResults)
}
