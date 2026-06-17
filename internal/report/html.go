package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"time"

	"apitester/internal/models"
	"apitester/pkg/utils"
)

type HTMLReporter struct {
	HistoryRecords []*models.HistoryRecord
}

func NewHTMLReporter() *HTMLReporter {
	return &HTMLReporter{}
}

func (r *HTMLReporter) Generate(suite *models.SuiteResult, outputPath string) error {
	data := struct {
		Suite     *models.SuiteResult
		History   []*models.HistoryRecord
		Generated string
	}{
		Suite:     suite,
		History:   r.HistoryRecords,
		Generated: utils.FormatTime(time.Now()),
	}

	tmpl, err := template.New("html_report").Funcs(template.FuncMap{
		"formatTime":     utils.FormatTime,
		"formatDuration": formatDurationHTML,
		"prettyJSON":     prettyJSONHTML,
		"statusClass":    statusClass,
		"statusText":     statusText,
		"escapeJS":       escapeJS,
	}).Parse(htmlTemplate)

	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func formatDurationHTML(d time.Duration) string {
	return utils.FormatDuration(d)
}

var (
	jsonKeyRegex    = regexp.MustCompile(`"([^"\\]|\\.)*"(\s*):`)
	jsonStringRegex = regexp.MustCompile(`: "([^"\\]|\\.)*"`)
	jsonBoolRegex   = regexp.MustCompile(`: (true|false)`)
	jsonNullRegex   = regexp.MustCompile(`: (null)`)
	jsonNumberRegex = regexp.MustCompile(`: (-?\d+\.?\d*)`)
)

func prettyJSONHTML(v interface{}) template.HTML {
	var data []byte
	var err error

	switch val := v.(type) {
	case string:
		var parsed interface{}
		if json.Unmarshal([]byte(val), &parsed) == nil {
			data, err = json.MarshalIndent(parsed, "", "  ")
		} else {
			return template.HTML(template.HTMLEscapeString(val))
		}
	default:
		data, err = json.MarshalIndent(v, "", "  ")
	}

	if err != nil {
		return template.HTML(template.HTMLEscapeString(fmt.Sprintf("%v", v)))
	}

	highlighted := syntaxHighlight(string(data))
	return template.HTML(highlighted)
}

func syntaxHighlight(jsonStr string) string {
	lines := bytes.Split([]byte(jsonStr), []byte("\n"))
	var result bytes.Buffer

	for _, line := range lines {
		lineStr := string(line)
		lineStr = template.HTMLEscapeString(lineStr)
		lineStr = highlightJSONLine(lineStr)
		result.WriteString(lineStr)
		result.WriteString("\n")
	}

	return result.String()
}

func highlightJSONLine(line string) string {
	line = jsonKeyRegex.ReplaceAllString(line, `<span class="json-key">"$1"</span>:`)
	line = jsonStringRegex.ReplaceAllString(line, `: <span class="json-string">"$1"</span>`)
	line = jsonBoolRegex.ReplaceAllString(line, `: <span class="json-boolean">$1</span>`)
	line = jsonNullRegex.ReplaceAllString(line, `: <span class="json-null">$1</span>`)
	line = jsonNumberRegex.ReplaceAllString(line, `: <span class="json-number">$1</span>`)
	return line
}

func statusClass(passed, skipped bool) string {
	if skipped {
		return "skipped"
	}
	if passed {
		return "passed"
	}
	return "failed"
}

func statusText(passed, skipped bool) string {
	if skipped {
		return "跳过"
	}
	if passed {
		return "通过"
	}
	return "失败"
}

func escapeJS(s string) template.JS {
	return template.JS(template.HTMLEscapeString(s))
}

var htmlTemplate = `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>API 测试报告 - {{.Suite.Name}}</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background: #f5f7fa; color: #333; line-height: 1.6; }
.container { max-width: 1400px; margin: 0 auto; padding: 20px; }
.header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; border-radius: 12px; margin-bottom: 20px; box-shadow: 0 4px 20px rgba(102, 126, 234, 0.3); }
.header h1 { font-size: 28px; margin-bottom: 10px; }
.header .meta { opacity: 0.9; font-size: 14px; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; margin-bottom: 20px; }
.stat-card { background: white; padding: 20px; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); text-align: center; transition: transform 0.2s, box-shadow 0.2s; }
.stat-card:hover { transform: translateY(-2px); box-shadow: 0 4px 20px rgba(0,0,0,0.12); }
.stat-card .value { font-size: 36px; font-weight: bold; margin-bottom: 5px; }
.stat-card .label { color: #666; font-size: 14px; }
.stat-card.passed .value { color: #52c41a; }
.stat-card.failed .value { color: #ff4d4f; }
.stat-card.skipped .value { color: #faad14; }
.stat-card.rate .value { color: #1890ff; }
.stat-card.duration .value { color: #722ed1; font-size: 28px; }
.filters { background: white; padding: 20px; border-radius: 12px; margin-bottom: 20px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); }
.filter-group { display: flex; gap: 15px; flex-wrap: wrap; align-items: center; }
.filter-group input[type="text"] { flex: 1; min-width: 200px; padding: 10px 15px; border: 2px solid #e8e8e8; border-radius: 8px; font-size: 14px; transition: border-color 0.2s; }
.filter-group input[type="text"]:focus { outline: none; border-color: #667eea; }
.filter-btn { padding: 10px 20px; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; transition: all 0.2s; background: #f0f2f5; color: #333; }
.filter-btn:hover { background: #e8e8e8; }
.filter-btn.active { background: #667eea; color: white; }
.test-list { background: white; border-radius: 12px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); overflow: hidden; }
.test-item { border-bottom: 1px solid #f0f0f0; transition: background 0.2s; }
.test-item:hover { background: #fafafa; }
.test-header { padding: 15px 20px; cursor: pointer; display: flex; align-items: center; gap: 15px; }
.test-header .status { width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0; }
.test-header .status.passed { background: #52c41a; box-shadow: 0 0 8px rgba(82, 196, 26, 0.5); }
.test-header .status.failed { background: #ff4d4f; box-shadow: 0 0 8px rgba(255, 77, 79, 0.5); }
.test-header .status.skipped { background: #faad14; box-shadow: 0 0 8px rgba(250, 173, 20, 0.5); }
.test-header .name { flex: 1; font-weight: 500; }
.test-header .duration { color: #999; font-size: 13px; }
.test-header .toggle { color: #999; transition: transform 0.2s; }
.test-header.expanded .toggle { transform: rotate(90deg); }
.test-body { display: none; padding: 0 20px 20px; border-top: 1px solid #f0f0f0; background: #fafafa; }
.test-body.expanded { display: block; }
.test-body h4 { margin: 15px 0 10px; color: #555; font-size: 14px; }
.assertion-list { background: white; border-radius: 8px; overflow: hidden; margin-bottom: 15px; }
.assertion-item { padding: 12px 15px; border-bottom: 1px solid #f0f0f0; display: flex; align-items: center; gap: 10px; }
.assertion-item:last-child { border-bottom: none; }
.assertion-item .status-icon { width: 20px; height: 20px; border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 12px; color: white; flex-shrink: 0; }
.assertion-item.passed .status-icon { background: #52c41a; }
.assertion-item.failed .status-icon { background: #ff4d4f; }
.assertion-item .assertion-name { flex: 1; }
.assertion-item.failed .assertion-name { color: #ff4d4f; font-weight: 500; }
.assertion-details { background: #fff1f0; border-left: 3px solid #ff4d4f; padding: 15px; margin: 10px 0; border-radius: 4px; }
.assertion-details .expected, .assertion-details .actual { margin: 8px 0; }
.assertion-details .label { font-weight: bold; margin-bottom: 4px; }
.assertion-details .expected .label { color: #52c41a; }
.assertion-details .actual .label { color: #ff4d4f; }
.assertion-details .diff { background: #fff2e8; padding: 8px; border-radius: 4px; margin-top: 8px; font-family: monospace; }
.detail-section { background: white; border-radius: 8px; padding: 15px; margin-bottom: 15px; }
.detail-section .section-header { cursor: pointer; display: flex; justify-content: space-between; align-items: center; padding: 5px 0; }
.detail-section .section-header:hover { color: #667eea; }
.detail-section .section-content { display: none; margin-top: 10px; }
.detail-section.expanded .section-content { display: block; }
.detail-section .section-toggle { transition: transform 0.2s; }
.detail-section.expanded .section-toggle { transform: rotate(90deg); }
.request-method { display: inline-block; padding: 3px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; color: white; margin-right: 8px; }
.method-GET { background: #52c41a; }
.method-POST { background: #1890ff; }
.method-PUT { background: #faad14; }
.method-DELETE { background: #ff4d4f; }
.method-PATCH { background: #722ed1; }
pre { background: #1e1e1e; color: #d4d4d4; padding: 15px; border-radius: 8px; overflow-x: auto; font-family: 'Consolas', 'Monaco', monospace; font-size: 13px; line-height: 1.5; }
.json-key { color: #9cdcfe; }
.json-string { color: #ce9178; }
.json-number { color: #b5cea8; }
.json-boolean { color: #569cd6; }
.json-null { color: #569cd6; }
.headers-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.headers-table th, .headers-table td { padding: 8px 12px; text-align: left; border-bottom: 1px solid #f0f0f0; }
.headers-table th { background: #fafafa; font-weight: 600; }
.history-section { background: white; padding: 20px; border-radius: 12px; margin-top: 20px; box-shadow: 0 2px 12px rgba(0,0,0,0.08); }
.history-section h3 { margin-bottom: 15px; color: #333; }
.trend-chart { height: 200px; background: #fafafa; border-radius: 8px; margin-bottom: 15px; position: relative; overflow: hidden; }
.trend-bar { position: absolute; bottom: 30px; width: 30px; background: linear-gradient(to top, #667eea, #764ba2); border-radius: 4px 4px 0 0; transition: height 0.5s; }
.trend-label { position: absolute; bottom: 5px; font-size: 10px; color: #999; text-align: center; width: 50px; transform: translateX(-10px); }
.history-list { max-height: 300px; overflow-y: auto; }
.history-item { display: flex; justify-content: space-between; align-items: center; padding: 12px; border-bottom: 1px solid #f0f0f0; }
.history-item:last-child { border-bottom: none; }
.history-item .info { flex: 1; }
.history-item .date { color: #999; font-size: 12px; }
.history-item .pass-rate { font-weight: bold; }
.history-item .pass-rate.high { color: #52c41a; }
.history-item .pass-rate.low { color: #ff4d4f; }
.footer { text-align: center; padding: 20px; color: #999; font-size: 13px; margin-top: 20px; }
@media (max-width: 768px) { .stats { grid-template-columns: repeat(2, 1fr); } .header h1 { font-size: 22px; } .filter-group { flex-direction: column; align-items: stretch; } .filter-group input[type="text"] { width: 100%; } }
</style>
</head>
<body>
<div class="container">
    <div class="header">
        <h1>📋 API 测试报告</h1>
        <div class="meta">
            <strong>{{.Suite.Name}}</strong> &middot; 生成时间: {{.Generated}} &middot; 开始: {{formatTime .Suite.StartTime}} &middot; 结束: {{formatTime .Suite.EndTime}}
        </div>
    </div>

    <div class="stats">
        <div class="stat-card">
            <div class="value">{{.Suite.Total}}</div>
            <div class="label">总用例数</div>
        </div>
        <div class="stat-card passed">
            <div class="value">{{.Suite.Passed}}</div>
            <div class="label">通过</div>
        </div>
        <div class="stat-card failed">
            <div class="value">{{.Suite.Failed}}</div>
            <div class="label">失败</div>
        </div>
        <div class="stat-card skipped">
            <div class="value">{{.Suite.Skipped}}</div>
            <div class="label">跳过</div>
        </div>
        <div class="stat-card rate">
            <div class="value">{{printf "%.1f%%" .Suite.PassRate}}</div>
            <div class="label">通过率</div>
        </div>
        <div class="stat-card duration">
            <div class="value">{{formatDuration .Suite.Duration}}</div>
            <div class="label">总耗时</div>
        </div>
    </div>

    <div class="filters">
        <div class="filter-group">
            <input type="text" id="searchInput" placeholder="🔍 搜索测试用例名称...">
            <button class="filter-btn active" data-filter="all">全部 ({{.Suite.Total}})</button>
            <button class="filter-btn" data-filter="passed">✓ 通过 ({{.Suite.Passed}})</button>
            <button class="filter-btn" data-filter="failed">✗ 失败 ({{.Suite.Failed}})</button>
            <button class="filter-btn" data-filter="skipped">⏭ 跳过 ({{.Suite.Skipped}})</button>
        </div>
    </div>

    <div class="test-list" id="testList">
        {{range $index, $test := .Suite.Tests}}
        <div class="test-item" data-status="{{statusClass $test.Passed $test.Skipped}}" data-name="{{$test.Name}}">
            <div class="test-header" onclick="toggleTest({{$index}})">
                <div class="status {{statusClass $test.Passed $test.Skipped}}"></div>
                <div class="name">{{$test.Name}}</div>
                <div class="duration">{{formatDuration $test.Duration}}</div>
                <div class="toggle">▶</div>
            </div>
            <div class="test-body" id="testBody{{$index}}">
                {{if $test.Description}}
                <p style="color: #666; margin: 10px 0;">{{$test.Description}}</p>
                {{end}}

                {{if $test.Error}}
                <div class="assertion-details">
                    <div class="label">❌ 错误信息</div>
                    <pre style="margin-top: 8px;">{{$test.Error}}</pre>
                </div>
                {{end}}

                <h4>🔍 断言结果 ({{len $test.Assertions}})</h4>
                <div class="assertion-list">
                    {{range $assertIndex, $assert := $test.Assertions}}
                    <div class="assertion-item {{statusClass $assert.Passed false}}">
                        <div class="status-icon">{{if $assert.Passed}}✓{{else}}✗{{end}}</div>
                        <div class="assertion-name">{{$assert.Name}}</div>
                    </div>
                    {{if not $assert.Passed}}
                    <div class="assertion-details" style="margin: 0 15px 10px;">
                        {{if $assert.Description}}
                        <p style="margin-bottom: 10px; color: #666;">{{$assert.Description}}</p>
                        {{end}}
                        <div class="expected">
                            <div class="label">✓ 期望值:</div>
                            <pre>{{$assert.Expected}}</pre>
                        </div>
                        <div class="actual">
                            <div class="label">✗ 实际值:</div>
                            <pre>{{$assert.Actual}}</pre>
                        </div>
                    </div>
                    {{end}}
                    {{end}}
                </div>

                {{if $test.Request}}
                <div class="detail-section" id="requestSection{{$index}}">
                    <div class="section-header" onclick="toggleSection('requestSection{{$index}}')">
                        <span><strong>📤 请求详情</strong> <span class="request-method method-{{$test.Request.Method}}">{{$test.Request.Method}}</span> {{$test.Request.URL}}</span>
                        <span class="section-toggle">▶</span>
                    </div>
                    <div class="section-content">
                        <h4>请求头:</h4>
                        <table class="headers-table">
                            <thead><tr><th>名称</th><th>值</th></tr></thead>
                            <tbody>
                                {{range $key, $value := $test.Request.Headers}}
                                <tr><td>{{$key}}</td><td>{{$value}}</td></tr>
                                {{end}}
                            </tbody>
                        </table>
                        {{if $test.Request.Body}}
                        <h4>请求体:</h4>
                        <pre>{{prettyJSON $test.Request.Body}}</pre>
                        {{end}}
                    </div>
                </div>
                {{end}}

                {{if $test.Response}}
                <div class="detail-section" id="responseSection{{$index}}">
                    <div class="section-header" onclick="toggleSection('responseSection{{$index}}')">
                        <span><strong>📥 响应详情</strong> <span style="color: {{if ge $test.Response.StatusCode 400}}#ff4d4f{{else}}#52c41a{{end}}; font-weight: bold;">{{$test.Response.StatusCode}}</span> · {{formatDuration $test.Response.Time}}</span>
                        <span class="section-toggle">▶</span>
                    </div>
                    <div class="section-content">
                        <h4>响应头:</h4>
                        <table class="headers-table">
                            <thead><tr><th>名称</th><th>值</th></tr></thead>
                            <tbody>
                                {{range $key, $value := $test.Response.Headers}}
                                <tr><td>{{$key}}</td><td>{{$value}}</td></tr>
                                {{end}}
                            </tbody>
                        </table>
                        {{if $test.Response.Body}}
                        <h4>响应体:</h4>
                        <pre>{{prettyJSON $test.Response.Body}}</pre>
                        {{end}}
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}
    </div>

    {{if .History}}
    <div class="history-section">
        <h3>📊 历史执行趋势</h3>
        <div class="trend-chart" id="trendChart">
            {{range $i, $record := .History}}
            <div class="trend-bar" style="left: calc({{$i}} * 60px + 20px); height: {{$record.PassRate}}%;" title="{{$record.SuiteName}} - {{printf "%.1f%%" $record.PassRate}}"></div>
            <div class="trend-label" style="left: calc({{$i}} * 60px + 10px);">{{$record.Timestamp.Format "01-02"}}</div>
            {{end}}
        </div>
        <div class="history-list">
            {{range $record := .History}}
            <div class="history-item">
                <div class="info">
                    <div>{{$record.SuiteName}}</div>
                    <div class="date">{{formatTime $record.Timestamp}}</div>
                </div>
                <div>
                    <span>{{$record.Passed}}/{{$record.Total}}</span>
                    <span class="pass-rate {{if ge $record.PassRate 80}}high{{else}}low{{end}}" style="margin-left: 10px;">{{printf "%.1f%%" $record.PassRate}}</span>
                </div>
            </div>
            {{end}}
        </div>
    </div>
    {{end}}

    <div class="footer">
        <p>Generated by API Tester &middot; {{.Generated}}</p>
    </div>
</div>

<script>
function toggleTest(index) {
    const header = document.querySelectorAll('.test-header')[index];
    const body = document.getElementById('testBody' + index);
    header.classList.toggle('expanded');
    body.classList.toggle('expanded');
}

function toggleSection(id) {
    event.stopPropagation();
    const section = document.getElementById(id);
    section.classList.toggle('expanded');
}

const searchInput = document.getElementById('searchInput');
const filterBtns = document.querySelectorAll('.filter-btn');
const testItems = document.querySelectorAll('.test-item');
let currentFilter = 'all';

function applyFilters() {
    const searchTerm = searchInput.value.toLowerCase();
    testItems.forEach(item => {
        const status = item.dataset.status;
        const name = item.dataset.name.toLowerCase();
        const matchesFilter = currentFilter === 'all' || status === currentFilter;
        const matchesSearch = name.includes(searchTerm);
        item.style.display = matchesFilter && matchesSearch ? 'block' : 'none';
    });
}

searchInput.addEventListener('input', applyFilters);
filterBtns.forEach(btn => {
    btn.addEventListener('click', () => {
        filterBtns.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        currentFilter = btn.dataset.filter;
        applyFilters();
    });
});
</script>
</body>
</html>
`
