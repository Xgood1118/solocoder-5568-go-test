package models

import (
	"time"
)

type TestSuite struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string            `json:"version,omitempty" yaml:"version,omitempty"`
	Includes    []string          `json:"includes,omitempty" yaml:"includes,omitempty"`
	Variables   map[string]any    `json:"variables,omitempty" yaml:"variables,omitempty"`
	Auth        *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	BaseURL     string            `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Timeout     int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries     int               `json:"retries,omitempty" yaml:"retries,omitempty"`
	Concurrency int               `json:"concurrency,omitempty" yaml:"concurrency,omitempty"`
	Setup       []*TestCase       `json:"setup,omitempty" yaml:"setup,omitempty"`
	Teardown    []*TestCase       `json:"teardown,omitempty" yaml:"teardown,omitempty"`
	TestCases   []*TestCase       `json:"test_cases" yaml:"test_cases"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	MockRules   []*MockRule       `json:"mock_rules,omitempty" yaml:"mock_rules,omitempty"`
	DataDriven  *DataDrivenConfig `json:"data_driven,omitempty" yaml:"data_driven,omitempty"`
}

type TestCase struct {
	ID          string            `json:"id,omitempty" yaml:"id,omitempty"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Skip        bool              `json:"skip,omitempty" yaml:"skip,omitempty"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	DependsOn   []string          `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	Variables   map[string]any    `json:"variables,omitempty" yaml:"variables,omitempty"`
	Auth        *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	Retries     int               `json:"retries,omitempty" yaml:"retries,omitempty"`
	Timeout     int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Request     *Request          `json:"request" yaml:"request"`
	WebSocket   *WebSocketConfig  `json:"websocket,omitempty" yaml:"websocket,omitempty"`
	SSE         *SSEConfig        `json:"sse,omitempty" yaml:"sse,omitempty"`
	Extract     map[string]string `json:"extract,omitempty" yaml:"extract,omitempty"`
	Assertions  []*Assertion      `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	DataDriven  *DataDrivenConfig `json:"data_driven,omitempty" yaml:"data_driven,omitempty"`
}

type Request struct {
	Method         string            `json:"method" yaml:"method"`
	URL            string            `json:"url" yaml:"url"`
	Headers        map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	QueryParams    map[string]string `json:"query_params,omitempty" yaml:"query_params,omitempty"`
	Body           *RequestBody      `json:"body,omitempty" yaml:"body,omitempty"`
	FollowRedirect bool              `json:"follow_redirect,omitempty" yaml:"follow_redirect,omitempty"`
	Insecure       bool              `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	Proxy          string            `json:"proxy,omitempty" yaml:"proxy,omitempty"`
}

type RequestBody struct {
	Type        string                 `json:"type" yaml:"type"`
	JSON        any                    `json:"json,omitempty" yaml:"json,omitempty"`
	Form        map[string]string      `json:"form,omitempty" yaml:"form,omitempty"`
	Multipart   []*MultipartField      `json:"multipart,omitempty" yaml:"multipart,omitempty"`
	Raw         string                 `json:"raw,omitempty" yaml:"raw,omitempty"`
	ContentType string                 `json:"content_type,omitempty" yaml:"content_type,omitempty"`
	GraphQL     *GraphQLBody           `json:"graphql,omitempty" yaml:"graphql,omitempty"`
}

type MultipartField struct {
	Name        string `json:"name" yaml:"name"`
	Value       string `json:"value,omitempty" yaml:"value,omitempty"`
	File        string `json:"file,omitempty" yaml:"file,omitempty"`
	ContentType string `json:"content_type,omitempty" yaml:"content_type,omitempty"`
}

type GraphQLBody struct {
	Query         string         `json:"query" yaml:"query"`
	Variables     map[string]any `json:"variables,omitempty" yaml:"variables,omitempty"`
	OperationName string         `json:"operation_name,omitempty" yaml:"operation_name,omitempty"`
}

type AuthConfig struct {
	Type   string      `json:"type" yaml:"type"`
	Config interface{} `json:"config" yaml:"config"`
}

type BearerAuth struct {
	Token        string `json:"token" yaml:"token"`
	TokenURL     string `json:"token_url,omitempty" yaml:"token_url,omitempty"`
	RefreshURL   string `json:"refresh_url,omitempty" yaml:"refresh_url,omitempty"`
	ClientID     string `json:"client_id,omitempty" yaml:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty" yaml:"client_secret,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty" yaml:"expires_in,omitempty"`
}

type BasicAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

type DigestAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

type OAuth2Auth struct {
	GrantType    string            `json:"grant_type" yaml:"grant_type"`
	TokenURL     string            `json:"token_url" yaml:"token_url"`
	ClientID     string            `json:"client_id" yaml:"client_id"`
	ClientSecret string            `json:"client_secret" yaml:"client_secret"`
	Username     string            `json:"username,omitempty" yaml:"username,omitempty"`
	Password     string            `json:"password,omitempty" yaml:"password,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty" yaml:"refresh_token,omitempty"`
	Scopes       []string          `json:"scopes,omitempty" yaml:"scopes,omitempty"`
	Params       map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
	ExpiresIn    int               `json:"expires_in,omitempty" yaml:"expires_in,omitempty"`
}

type APIKeyAuth struct {
	Key   string `json:"key" yaml:"key"`
	Value string `json:"value" yaml:"value"`
	In    string `json:"in" yaml:"in"`
}

type Assertion struct {
	Type        string `json:"type" yaml:"type"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Property    string `json:"property,omitempty" yaml:"property,omitempty"`
	Value       any    `json:"value,omitempty" yaml:"value,omitempty"`
	Expected    any    `json:"expected,omitempty" yaml:"expected,omitempty"`
	Actual      any    `json:"actual,omitempty" yaml:"actual,omitempty"`
	Operator    string `json:"operator,omitempty" yaml:"operator,omitempty"`
	Message     string `json:"message,omitempty" yaml:"message,omitempty"`
	Passed      bool   `json:"passed,omitempty" yaml:"passed,omitempty"`
}

type AssertionResult struct {
	Passed     bool   `json:"passed"`
	Type       string `json:"type"`
	Name       string `json:"name,omitempty"`
	Property   string `json:"property,omitempty"`
	Expected   any    `json:"expected,omitempty"`
	Actual     any    `json:"actual,omitempty"`
	Operator   string `json:"operator,omitempty"`
	Message    string `json:"message,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	BodyString string            `json:"body_string,omitempty"`
	Latency    time.Duration     `json:"latency"`
	Time       time.Duration     `json:"time"`
	Protocol   string            `json:"protocol,omitempty"`
	TLS        bool              `json:"tls,omitempty"`
}

type TestResult struct {
	CaseID        string             `json:"case_id"`
	CaseName      string             `json:"case_name"`
	Name          string             `json:"name"`
	Status        string             `json:"status"`
	Passed        bool               `json:"passed"`
	Skipped       bool               `json:"skipped"`
	SkipReason    string             `json:"skip_reason,omitempty"`
	StartTime     time.Time          `json:"start_time"`
	EndTime       time.Time          `json:"end_time"`
	Duration      time.Duration      `json:"duration"`
	Retries       int                `json:"retries"`
	Assertions    []*AssertionResult `json:"assertions,omitempty"`
	Extracts      map[string]string  `json:"extracts,omitempty"`
	Request       *Request           `json:"request,omitempty"`
	Response      *Response          `json:"response,omitempty"`
	Error         string             `json:"error,omitempty"`
	DataRowIndex  int                `json:"data_row_index,omitempty"`
	DataRow       map[string]any     `json:"data_row,omitempty"`
}

type SuiteResult struct {
	SuiteName   string        `json:"suite_name"`
	Name        string        `json:"name"`
	StartTime   time.Time     `json:"start_time"`
	EndTime     time.Time     `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	Total       int           `json:"total"`
	Passed      int           `json:"passed"`
	Failed      int           `json:"failed"`
	Skipped     int           `json:"skipped"`
	PassRate    float64       `json:"pass_rate"`
	TestResults []*TestResult `json:"test_results"`
	Tests       []*TestResult `json:"tests"`
	Error       string        `json:"error,omitempty"`
	Environment string        `json:"environment,omitempty"`
	Variables   map[string]any `json:"variables,omitempty"`
}

type WebSocketConfig struct {
	URL          string                 `json:"url" yaml:"url"`
	Headers      map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Protocols    []string               `json:"protocols,omitempty" yaml:"protocols,omitempty"`
	Messages     []*WebSocketMessage    `json:"messages" yaml:"messages"`
	PingInterval int                    `json:"ping_interval,omitempty" yaml:"ping_interval,omitempty"`
	Heartbeat    bool                   `json:"heartbeat,omitempty" yaml:"heartbeat,omitempty"`
	Timeout      int                    `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

type WebSocketMessage struct {
	Type       string    `json:"type" yaml:"type"`
	Content    string    `json:"content" yaml:"content"`
	WaitMs     int       `json:"wait_ms,omitempty" yaml:"wait_ms,omitempty"`
	Assertions []*Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Extract    map[string]string `json:"extract,omitempty" yaml:"extract,omitempty"`
}

type SSEConfig struct {
	URL     string            `json:"url" yaml:"url"`
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Events  []*SSEEvent       `json:"events" yaml:"events"`
	Timeout int               `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

type SSEEvent struct {
	EventName  string    `json:"event_name,omitempty" yaml:"event_name,omitempty"`
	WaitFor    string    `json:"wait_for,omitempty" yaml:"wait_for,omitempty"`
	Count      int       `json:"count,omitempty" yaml:"count,omitempty"`
	Assertions []*Assertion `json:"assertions,omitempty" yaml:"assertions,omitempty"`
	Extract    map[string]string `json:"extract,omitempty" yaml:"extract,omitempty"`
}

type MockRule struct {
	ID           string            `json:"id,omitempty" yaml:"id,omitempty"`
	MatchPath    string            `json:"match_path" yaml:"match_path"`
	MatchMethod  string            `json:"match_method,omitempty" yaml:"match_method,omitempty"`
	MatchHeaders map[string]string `json:"match_headers,omitempty" yaml:"match_headers,omitempty"`
	Status       int               `json:"status" yaml:"status"`
	DelayMs      int               `json:"delay_ms,omitempty" yaml:"delay_ms,omitempty"`
	Headers      map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body         any               `json:"body,omitempty" yaml:"body,omitempty"`
	BodyFile     string            `json:"body_file,omitempty" yaml:"body_file,omitempty"`
}

type DataDrivenConfig struct {
	Source string            `json:"source" yaml:"source"`
	Format string            `json:"format" yaml:"format"`
	CSV    *CSVConfig        `json:"csv,omitempty" yaml:"csv,omitempty"`
	Data   []map[string]any  `json:"data,omitempty" yaml:"data,omitempty"`
}

type CSVConfig struct {
	File      string `json:"file" yaml:"file"`
	Delimiter string `json:"delimiter,omitempty" yaml:"delimiter,omitempty"`
	HasHeader bool   `json:"has_header,omitempty" yaml:"has_header,omitempty"`
}

type HistoryRecord struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	SuiteName  string        `json:"suite_name"`
	Total      int           `json:"total"`
	Passed     int           `json:"passed"`
	Failed     int           `json:"failed"`
	Skipped    int           `json:"skipped"`
	PassRate   float64       `json:"pass_rate"`
	Duration   time.Duration `json:"duration"`
	DurationMs int64         `json:"duration_ms"`
	Environment string       `json:"environment,omitempty"`
	ReportPath string        `json:"report_path,omitempty"`
}

type TrendData struct {
	Timestamps []string  `json:"timestamps"`
	PassRates  []float64 `json:"pass_rates"`
	Durations  []float64 `json:"durations"`
}

type ExecutionOptions struct {
	Environment   string
	Tags          []string
	SkipTags      []string
	Filter        string
	DryRun        bool
	Concurrency   int
	Retries       int
	Timeout       int
	ReportFormats []string
	ReportOutput  string
	MockMode      bool
	MockPort      int
}

type VariableScope int

const (
	ScopeBuiltin VariableScope = iota
	ScopeTestCase
	ScopeGlobal
	ScopeEnvironment
)

type TokenCache struct {
	Token        string
	ExpiresAt    time.Time
	RefreshToken string
}

type CLIProgress struct {
	Total       int
	Completed   int
	Current     int
	Passed      int
	Failed      int
	Skipped     int
	CurrentCase string
	CurrentName string
	Phase       string
	Status      string
}
