package evaluation

import (
	"GopherAI/common/mysql"
	"GopherAI/model"
)

func CreateRun(run *model.EvalRun) error {
	return mysql.DB.Create(run).Error
}

func UpdateRun(run *model.EvalRun) error {
	return mysql.DB.Save(run).Error
}

func GetRunByID(runID, userName string) (*model.EvalRun, error) {
	var run model.EvalRun
	err := mysql.DB.Where("id = ? AND user_name = ?", runID, userName).First(&run).Error
	return &run, err
}

func ListRunsByUser(userName string, limit int) ([]model.EvalRun, error) {
	var runs []model.EvalRun
	q := mysql.DB.Where("user_name = ?", userName).Order("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&runs).Error
	return runs, err
}

func CreateResults(results []model.EvalResult) error {
	if len(results) == 0 {
		return nil
	}
	return mysql.DB.Create(&results).Error
}

func ListResultsByRunID(runID string) ([]model.EvalResult, error) {
	var results []model.EvalResult
	err := mysql.DB.Where("run_id = ?", runID).Order("created_at ASC").Find(&results).Error
	return results, err
}
