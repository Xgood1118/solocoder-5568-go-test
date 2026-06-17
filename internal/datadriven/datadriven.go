package datadriven

import (
	"encoding/json"
	"fmt"
	"strings"

	"apitester/internal/models"
)

type ExecutionUnit struct {
	Case    *models.TestCase
	DataRow map[string]any
	Index   int
}

type DataDrivenEngine struct {
}

func NewDataDrivenEngine() *DataDrivenEngine {
	return &DataDrivenEngine{}
}

func (e *DataDrivenEngine) ExpandCase(testCase *models.TestCase) ([]*ExecutionUnit, error) {
	if testCase.DataDriven == nil {
		return []*ExecutionUnit{
			{
				Case:  testCase,
				Index: 0,
			},
		}, nil
	}

	dataRows, err := e.loadData(testCase.DataDriven)
	if err != nil {
		return nil, err
	}

	units := make([]*ExecutionUnit, len(dataRows))
	for i, row := range dataRows {
		expandedCase := e.expandTestCase(testCase, row)
		units[i] = &ExecutionUnit{
			Case:    expandedCase,
			DataRow: row,
			Index:   i,
		}
	}

	return units, nil
}

func (e *DataDrivenEngine) loadData(config *models.DataDrivenConfig) ([]map[string]any, error) {
	switch strings.ToLower(config.Format) {
	case "csv":
		if config.CSV == nil {
			return nil, fmt.Errorf("csv config is required for csv format")
		}
		return LoadCSV(config.CSV)
	case "json":
		if config.Source != "" {
			return LoadJSONArray(config.Source)
		}
		return nil, fmt.Errorf("source is required for json format")
	case "inline":
		return LoadInlineData(config.Data)
	default:
		return nil, fmt.Errorf("unsupported data format: %s", config.Format)
	}
}

func (e *DataDrivenEngine) expandTestCase(original *models.TestCase, data map[string]any) *models.TestCase {
	caseJSON, _ := json.Marshal(original)
	caseStr := string(caseJSON)

	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		var valueStr string
		switch v := value.(type) {
		case string:
			valueStr = v
		default:
			valueBytes, _ := json.Marshal(v)
			valueStr = string(valueBytes)
		}
		caseStr = strings.ReplaceAll(caseStr, placeholder, valueStr)
	}

	var expandedCase models.TestCase
	_ = json.Unmarshal([]byte(caseStr), &expandedCase)

	if expandedCase.Variables == nil {
		expandedCase.Variables = make(map[string]any)
	}
	for key, value := range data {
		expandedCase.Variables[key] = value
	}

	return &expandedCase
}
