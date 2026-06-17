package datadriven

func LoadInlineData(data []map[string]any) ([]map[string]any, error) {
	if data == nil {
		return []map[string]any{}, nil
	}

	result := make([]map[string]any, len(data))
	for i, row := range data {
		rowCopy := make(map[string]any, len(row))
		for k, v := range row {
			rowCopy[k] = v
		}
		result[i] = rowCopy
	}

	return result, nil
}
