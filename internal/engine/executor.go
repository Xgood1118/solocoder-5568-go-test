package engine

import (
	"context"
	"fmt"
	"time"

	"apitester/internal/assert"
	"apitester/internal/auth"
	"apitester/internal/httpclient"
	"apitester/internal/models"
	"apitester/internal/variables"
)

type CaseExecutor struct {
	variableStore *variables.VariableStore
	interpolator *variables.Interpolator
	authManager *auth.AuthManager
	assertionEngine *assert.AssertionEngine
	httpClient *httpclient.Client
	defaultTimeout time.Duration
	defaultRetries int
}

type ExecuteResult struct {
	Result    *models.TestResult
	Passed  bool
	Extracts map[string]string
}

func NewCaseExecutor(
	variableStore *variables.VariableStore,
	authManager *auth.AuthManager,
	defaultTimeout time.Duration,
	defaultRetries int,
) *CaseExecutor {
	clientOpt := &httpclient.ClientOption{
		Timeout: defaultTimeout,
	}

	return &CaseExecutor{
		variableStore:   variableStore,
		interpolator:  variables.NewInterpolator(variableStore),
		authManager:    authManager,
		assertionEngine: assert.NewAssertionEngine(),
		httpClient:     *httpclient.NewClient(clientOpt),
		defaultTimeout: defaultTimeout,
		defaultRetries: defaultRetries,
	}
}

func (ce *CaseExecutor) ExecuteCase(tc *models.TestCase, baseURL string, suiteAuth *models.AuthConfig) *models.TestResult {
	startTime := time.Now()

	result := &models.TestResult{
		CaseID:   tc.ID,
		CaseName: tc.Name,
		StartTime: startTime,
		Status:   "running",
	}

	if tc.ID == "" {
		tc.ID = tc.Name
	}

	retries := ce.defaultRetries
	if tc.Retries > 0 {
		retries = tc.Retries
	}

	timeout := ce.defaultTimeout
	if tc.Timeout > 0 {
		timeout = time.Duration(tc.Timeout) * time.Second
	}

	ce.variableStore.ClearScope(models.ScopeTestCase)
	if tc.Variables != nil {
		ce.variableStore.SetMap(models.ScopeTestCase, tc.Variables)
	}

	authConfig := tc.Auth
	if authConfig == nil {
		authConfig = suiteAuth
	}

	var retryResult *RetryResult
	var lastResult *models.TestResult

	retryConfig := &RetryConfig{
		MaxAttempts:  retries + 1,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}

	retryResult = RetryWithBackoff(func() error {
		execResult, err := ce.executeWithTimeout(tc, baseURL, authConfig, timeout)
		lastResult = execResult
		if err != nil {
			return err
		}

		allPassed := true
		for _, ar := range execResult.Assertions {
			if !ar.Passed {
				allPassed = false
				break
			}
		}

		if !allPassed {
			return fmt.Errorf("assertions failed")
		}

		return nil
	}, retryConfig)

	result.Retries = retryResult.Attempts - 1

	if lastResult != nil {
		result.Assertions = lastResult.Assertions
		result.Extracts = lastResult.Extracts
		result.Request = lastResult.Request
		result.Response = lastResult.Response
	}

	if retryResult.Success {
		result.Status = "passed"
	} else {
		result.Status = "failed"
		if retryResult.LastError != nil {
			result.Error = retryResult.LastError.Error()
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if result.Extracts != nil && len(result.Extracts) > 0 {
		for k, v := range result.Extracts {
			ce.variableStore.Set(models.ScopeGlobal, k, v)
		}
	}

	return *result
}

func (ce *CaseExecutor) executeWithTimeout(
	tc *models.TestCase,
	baseURL string,
	authConfig *models.AuthConfig,
	timeout time.Duration,
) (*models.TestResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan *ExecuteResult, 1)
	errChan := make(chan error, 1)

	go func() {
		result, err := ce.executeSingle(tc, baseURL, authConfig)
		if err != nil {
			errChan <- err
			return
		}
		done <- result
	}()

	select {
	case <-ctx.Done():
		return &models.TestResult{
			CaseID:   tc.ID,
			CaseName: tc.Name,
			Status:   "failed",
			Error:    fmt.Sprintf("timeout after %v", timeout),
		}, ctx.Err()
	case err := <-errChan:
		return &models.TestResult{
			CaseID:   tc.ID,
			CaseName: tc.Name,
			Status:   "failed",
			Error:    err.Error(),
		}, err
	case result := <-done:
		return result.Result, nil
	}
}

func (ce *CaseExecutor) executeSingle(
	tc *models.TestCase,
	baseURL string,
	authConfig *models.AuthConfig,
) (*ExecuteResult, error) {
	result := &models.TestResult{
		CaseID:   tc.ID,
		CaseName: tc.Name,
	}

	req, err := ce.buildRequest(tc, baseURL, authConfig)
	if err != nil {
		return &ExecuteResult{Result: result}, err}
	}
	result.Request = req

	response, err := ce.httpClient.Do(tc.Request)
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return &ExecuteResult{Result: result}, err
	}
	result.Response = response

	assertions, extracts, err := ce.assertionEngine.AssertAndExtract(
		response,
		tc.Assertions,
		tc.Extract,
	)
	if err != nil {
		result.Error = err.Error()
	}

	result.Assertions = assertions

	for _, a := range assertions {
		if !a.Passed {
			result.Status = "failed"
			return &ExecuteResult{
				Result:   result,
				Extracts: extracts,
			}, fmt.Errorf("assertion failed")
		}
	}

	result.Status = "passed"
	return &ExecuteResult{
		Result:   result,
		Extracts: extracts,
	}, nil
}

func (ce *CaseExecutor) buildRequest(
	tc *models.TestCase,
	baseURL string,
	authConfig *models.AuthConfig,
) (*models.Request, error) {
	req := &models.Request{}
	*req = *tc.Request

	if baseURL != "" && !isAbsoluteURL(req.URL) {
		req.URL = baseURL + req.URL
	}

	interpolatedURL, err := ce.interpolator.InterpolateString(req.URL)
	if err != nil {
		return nil, fmt.Errorf("interpolate url: %w", err)
	}
	req.URL = interpolatedURL

	if req.Headers != nil {
		interpolatedHeaders, err := ce.interpolator.InterpolateMap(req.Headers)
		if err != nil {
			return nil, fmt.Errorf("interpolate headers: %w", err)
		}
		req.Headers = interpolatedHeaders
	}

	if req.QueryParams != nil {
		interpolatedQuery, err := ce.interpolator.InterpolateMap(req.QueryParams)
		if err != nil {
			return nil, fmt.Errorf("interpolate query params: %w", err)
		}
		req.QueryParams = interpolatedQuery
	}

	if req.Body != nil && req.Body.JSON != nil {
		interpolatedJSON, err := ce.interpolator.InterpolateInterface(req.Body.JSON)
		if err != nil {
			return nil, fmt.Errorf("interpolate body: %w", err)
		}
		req.Body.JSON = interpolatedJSON
	}

	if req.Body != nil && req.Body.Form != nil {
		interpolatedForm, err := ce.interpolator.InterpolateMap(req.Body.Form)
		if err != nil {
			return nil, fmt.Errorf("interpolate form: %w", err)
		}
		req.Body.Form = interpolatedForm
	}

	return req, nil
}

func isAbsoluteURL(url string) bool {
	return len(url) >= 8 && (url[:7] == "http://" || url[:8] == "https://")
}

func (ce *CaseExecutor) ExecuteSetupTeardown(cases []*models.TestCase, baseURL string, authConfig *models.AuthConfig) []*models.TestResult {
	var results []*models.TestResult

	for _, tc := range cases {
		result := ce.ExecuteCase(tc, baseURL, authConfig)
		results = append(results, &result)
	}

	return results
}
