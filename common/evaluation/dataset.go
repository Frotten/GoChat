package evaluation

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"
)

//go:embed datasets/default.json
var defaultDatasetJSON []byte

var (
	datasetOnce sync.Once
	datasetErr  error
	datasets    map[string]DatasetDefinition
)

func loadDatasets() {
	datasetOnce.Do(func() {
		datasets = make(map[string]DatasetDefinition)
		var ds DatasetDefinition
		if err := json.Unmarshal(defaultDatasetJSON, &ds); err != nil {
			datasetErr = fmt.Errorf("parse default dataset: %w", err)
			return
		}
		if err := ValidateDataset(ds); err != nil {
			datasetErr = fmt.Errorf("validate default dataset: %w", err)
			return
		}
		datasets[ds.ID] = ds
	})
}

// ValidateDataset fails fast on labels that would otherwise produce misleading metrics.
func ValidateDataset(ds DatasetDefinition) error {
	if ds.ID == "" || ds.Name == "" {
		return fmt.Errorf("dataset id and name are required")
	}
	seen := make(map[string]bool, len(ds.Cases))
	for i, c := range ds.Cases {
		if c.ID == "" || c.Name == "" || c.Input == "" {
			return fmt.Errorf("case %d requires id, name and input", i)
		}
		if seen[c.ID] {
			return fmt.Errorf("duplicate case id: %s", c.ID)
		}
		seen[c.ID] = true
		switch normalizeCategory(c.Category) {
		case CategoryToolCall:
			if len(c.ExpectedTools) == 0 && len(c.ForbiddenTools) == 0 {
				return fmt.Errorf("tool case %s requires expected_tools or forbidden_tools", c.ID)
			}
			for _, expected := range c.ExpectedTools {
				if expected.Name == "" {
					return fmt.Errorf("tool case %s contains an empty tool name", c.ID)
				}
			}
		case CategoryRAG:
			if len(c.ExpectedSources) == 0 {
				return fmt.Errorf("rag case %s requires expected_sources", c.ID)
			}
		}
		if c.PassThreshold < 0 || c.PassThreshold > 1 {
			return fmt.Errorf("case %s pass_threshold must be between 0 and 1", c.ID)
		}
	}
	return nil
}

// ListDatasets 返回所有内置测评集。
func ListDatasets() ([]DatasetDefinition, error) {
	loadDatasets()
	if datasetErr != nil {
		return nil, datasetErr
	}
	out := make([]DatasetDefinition, 0, len(datasets))
	for _, ds := range datasets {
		out = append(out, ds)
	}
	return out, nil
}

// GetDataset 按 ID 获取测评集。
func GetDataset(id string) (DatasetDefinition, error) {
	loadDatasets()
	if datasetErr != nil {
		return DatasetDefinition{}, datasetErr
	}
	ds, ok := datasets[id]
	if !ok {
		return DatasetDefinition{}, fmt.Errorf("dataset not found: %s", id)
	}
	return ds, nil
}
