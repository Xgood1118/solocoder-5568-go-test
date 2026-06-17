package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"apitester/internal/models"
	"apitester/internal/report"

	"github.com/gin-gonic/gin"
)

//go:embed templates/* static/*
var embeddedFiles embed.FS

type Server struct {
	router         *gin.Engine
	historyManager *report.HistoryManager
}

func NewServer(historyManager *report.HistoryManager) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	s := &Server{
		router:         r,
		historyManager: historyManager,
	}

	s.setupRoutes()
	s.setupTemplates()

	return s
}

func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
		api.GET("/history", s.getHistory)
		api.GET("/history/trend", s.getTrend)
		api.GET("/health", s.healthCheck)
	}

	s.router.GET("/", s.index)
	s.router.GET("/trend", s.trendPage)

	staticFS, _ := fs.Sub(embeddedFiles, "static")
	s.router.StaticFS("/static", http.FS(staticFS))
}

func (s *Server) setupTemplates() {
	tmpl := template.New("")

	tmpl.Funcs(template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"formatDuration": func(ms int64) string {
			d := time.Duration(ms) * time.Millisecond
			if d < time.Second {
				return d.String()
			}
			return d.Round(time.Millisecond).String()
		},
		"statusClass": func(failed int) string {
			if failed > 0 {
				return "status-failed"
			}
			return "status-passed"
		},
		"trendData": func(records []*models.HistoryRecord) string {
			if len(records) == 0 {
				return "[]"
			}
			var data []string
			for _, r := range records {
				data = append(data, fmt.Sprintf("%.1f", r.PassRate))
			}
			return "[" + strings.Join(data, ",") + "]"
		},
		"trendLabels": func(records []*models.HistoryRecord) string {
			if len(records) == 0 {
				return "[]"
			}
			var labels []string
			for _, r := range records {
				labels = append(labels, "'"+r.Timestamp.Format("01-02 15:04")+"'")
			}
			return "[" + strings.Join(labels, ",") + "]"
		},
		"trendDurations": func(records []*models.HistoryRecord) string {
			if len(records) == 0 {
				return "[]"
			}
			var data []string
			for _, r := range records {
				data = append(data, fmt.Sprintf("%d", r.DurationMs))
			}
			return "[" + strings.Join(data, ",") + "]"
		},
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		"mult": func(a, b int) int {
			return a * b
		},
		"minus": func(a, b int) int {
			return a - b
		},
	})

	templates, _ := fs.Sub(embeddedFiles, "templates")
	tmpl, err := tmpl.ParseFS(templates, "*.html")
	if err != nil {
		panic(err)
	}

	s.router.SetHTMLTemplate(tmpl)
}

func (s *Server) index(c *gin.Context) {
	records, _ := s.historyManager.LoadRecords()

	if len(records) > 50 {
		records = records[:50]
	}

	stats := calculateStats(records)

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":   "API Tester - 测试报告",
		"records": records,
		"stats":   stats,
	})
}

func (s *Server) trendPage(c *gin.Context) {
	suite := c.Query("suite")
	records := s.historyManager.GetTrendData(suite, 20)

	c.HTML(http.StatusOK, "trend.html", gin.H{
		"title":   "API Tester - 趋势分析",
		"records": records,
		"suite":   suite,
	})
}

func (s *Server) getHistory(c *gin.Context) {
	suite := c.Query("suite")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := parseInt(l); err == nil {
			limit = n
		}
	}

	records, _ := s.historyManager.LoadRecords()

	if suite != "" {
		var filtered []*models.HistoryRecord
		for _, r := range records {
			if strings.Contains(r.SuiteName, suite) {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    records,
	})
}

func (s *Server) getTrend(c *gin.Context) {
	suite := c.Query("suite")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := parseInt(l); err == nil {
			limit = n
		}
	}

	records := s.historyManager.GetTrendData(suite, limit)

	var passRates []float64
	var durations []int64
	var labels []string

	for _, r := range records {
		passRates = append(passRates, r.PassRate)
		durations = append(durations, r.DurationMs)
		labels = append(labels, r.Timestamp.Format("01-02 15:04"))
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"labels":     labels,
			"pass_rates": passRates,
			"durations":  durations,
			"records":    records,
		},
	})
}

func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func calculateStats(records []*models.HistoryRecord) map[string]interface{} {
	if len(records) == 0 {
		return map[string]interface{}{
			"total_runs":    0,
			"avg_pass_rate": 0.0,
			"total_cases":   0,
			"avg_duration":  int64(0),
		}
	}

	var totalRuns = len(records)
	var totalPassRate float64
	var totalCases int
	var totalDuration int64

	for _, r := range records {
		totalPassRate += r.PassRate
		totalCases += r.Total
		totalDuration += r.DurationMs
	}

	return map[string]interface{}{
		"total_runs":    totalRuns,
		"avg_pass_rate": totalPassRate / float64(totalRuns),
		"total_cases":   totalCases,
		"avg_duration":  totalDuration / int64(totalRuns),
		"latest":        records[0],
		"best":          findBestRecord(records),
	}
}

func findBestRecord(records []*models.HistoryRecord) *models.HistoryRecord {
	if len(records) == 0 {
		return nil
	}
	best := records[0]
	for _, r := range records[1:] {
		if r.PassRate > best.PassRate || (r.PassRate == best.PassRate && r.DurationMs < best.DurationMs) {
			best = r
		}
	}
	return best
}
