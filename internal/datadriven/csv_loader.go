package datadriven

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"apitester/internal/models"
)

func LoadCSV(config *models.CSVConfig) ([]map[string]any, error) {
	file, err := os.Open(config.File)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = getDelimiter(config.Delimiter)
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv file: %w", err)
	}

	if len(records) == 0 {
		return []map[string]any{}, nil
	}

	var headers []string
	var dataStart int

	if config.HasHeader {
		headers = records[0]
		dataStart = 1
	} else {
		headers = make([]string, len(records[0]))
		for i := range records[0] {
			headers[i] = fmt.Sprintf("col%d", i+1)
		}
		dataStart = 0
	}

	result := make([]map[string]any, 0, len(records)-dataStart)
	for i := dataStart; i < len(records); i++ {
		row := make(map[string]any)
		for j, value := range records[i] {
			if j < len(headers) {
				row[headers[j]] = value
			}
		}
		result = append(result, row)
	}

	return result, nil
}

func getDelimiter(delimiter string) rune {
	if delimiter == "" {
		return ','
	}
	delimiter = strings.TrimSpace(delimiter)
	switch delimiter {
	case "\\t":
		return '\t'
	case "tab":
		return '\t'
	default:
		return rune(delimiter[0])
	}
}
