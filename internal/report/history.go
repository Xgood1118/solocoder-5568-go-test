package report

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"apitester/internal/models"
	"apitester/pkg/utils"
)

const (
	historyDir      = "~/.apitester"
	historyFileName = "history.json"
	maxHistoryRecords = 100
)

type HistoryManager struct {
	historyPath string
}

func NewHistoryManager() *HistoryManager {
	historyDirPath := utils.ExpandPath(historyDir)
	return &HistoryManager{
		historyPath: filepath.Join(historyDirPath, historyFileName),
	}
}

func (m *HistoryManager) SaveRecord(record *models.HistoryRecord) error {
	if err := utils.EnsureDir(filepath.Dir(m.historyPath)); err != nil {
		return fmt.Errorf("ensure history dir: %w", err)
	}

	records, err := m.LoadRecords()
	if err != nil {
		records = []*models.HistoryRecord{}
	}

	if record.ID == "" {
		record.ID = generateID()
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	records = append(records, record)

	if len(records) > maxHistoryRecords {
		records = records[len(records)-maxHistoryRecords:]
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	if err := os.WriteFile(m.historyPath, data, 0644); err != nil {
		return fmt.Errorf("write history file: %w", err)
	}

	return nil
}

func (m *HistoryManager) SaveSuiteRecord(suite models.SuiteResult, reportPath string) error {
	record := &models.HistoryRecord{
		ID:         generateID(),
		Timestamp:  time.Now(),
		SuiteName:  suite.Name,
		Total:      suite.Total,
		Passed:     suite.Passed,
		Failed:     suite.Failed,
		Skipped:    suite.Skipped,
		PassRate:   suite.PassRate,
		Duration:   suite.Duration,
		DurationMs: suite.Duration.Milliseconds(),
		ReportPath: reportPath,
	}
	return m.SaveRecord(record)
}

func (m *HistoryManager) LoadRecords() ([]*models.HistoryRecord, error) {
	data, err := os.ReadFile(m.historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*models.HistoryRecord{}, nil
		}
		return nil, fmt.Errorf("read history file: %w", err)
	}

	var records []*models.HistoryRecord
	if len(data) == 0 {
		return []*models.HistoryRecord{}, nil
	}

	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("unmarshal records: %w", err)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.Before(records[j].Timestamp)
	})

	return records, nil
}

func (m *HistoryManager) GetTrendData(suite string, n int) []*models.HistoryRecord {
	records, err := m.LoadRecords()
	if err != nil {
		return []*models.HistoryRecord{}
	}

	if len(records) == 0 {
		return []*models.HistoryRecord{}
	}

	if suite != "" {
		var filtered []*models.HistoryRecord
		for _, r := range records {
			if r.SuiteName == suite {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	if n <= 0 || n > len(records) {
		n = len(records)
	}

	recentRecords := records[len(records)-n:]
	return recentRecords
}

func (m *HistoryManager) GetTrendDataModel(n int) (models.TrendData, error) {
	records, err := m.LoadRecords()
	if err != nil {
		return models.TrendData{}, err
	}

	if len(records) == 0 {
		return models.TrendData{}, nil
	}

	if n <= 0 || n > len(records) {
		n = len(records)
	}

	recentRecords := records[len(records)-n:]

	trend := models.TrendData{
		Timestamps: make([]string, 0, n),
		PassRates:  make([]float64, 0, n),
		Durations:  make([]float64, 0, n),
	}

	for _, record := range recentRecords {
		trend.Timestamps = append(trend.Timestamps, record.Timestamp.Format("01-02 15:04"))
		trend.PassRates = append(trend.PassRates, record.PassRate)
		trend.Durations = append(trend.Durations, record.Duration.Seconds())
	}

	return trend, nil
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("id-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
