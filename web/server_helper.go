package web

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"apitester/internal/models"
)

func fmt_sscanf(str, format string, args ...interface{}) (int, error) {
	return fmt.Sscanf(str, format, args...)
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(s))
}

func sortRecordsByTime(records []*models.HistoryRecord) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})
}
