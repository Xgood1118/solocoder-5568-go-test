package engine

import (
	"fmt"
	"time"

	"apitester/internal/auth"
	"apitester/internal/filter"
	"apitester/internal/models"
	"apitester/internal/variables"
)

type TestEngine struct {
	options       *models.ExecutionOptions
	variableStore *variables.VariableStore
	authManager *auth.AuthManager
	filterEngine *filter.FilterEngine
}

func NewTestEngine(options *models.ExecutionOptions) *TestEngine {
	if options == nil {
		options = &models.ExecutionOptions{}
	}

	variableStore := variables.NewVariableStore()
	variableStore.SetBuiltinVariables()

	if options.Environment != "" {
		variableStore.Set(variables.ScopeEnvironment, "env", options.Environment)
	}

	return &TestEngine{
		options:       options,
		variableStore: variableStore,
		authManager:   auth.NewAuthManager(),
		filterEngine:  filter.NewFilterEngine(),
	}
}

func (te *TestEngine) RunSuite(suite *models.TestSuite) *models.SuiteResult {
	startTime := time.Now()

	suiteResult := &models.SuiteResult{
		SuiteName: suite.Name,
		StartTime: startTime,
		Variables: te.variableStore.GetAll(),
	}

	if suite.Variables != nil {
		te.variableStore.SetMap(models.ScopeGlobal, suite.Variables)
	}

	filteredCases := te.filterEngine.FilterCases(suite.TestCases, te.options)

	concurrency := te.options.Concurrency
	if concurrency <= 0 {
		concurrency = suite.Concurrency
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	timeout := time.Duration(te.options.Timeout) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(suite.Timeout) * time.Second
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	retries := te.options.Retries
	if retries <= 0 {
		retries = suite.Retries
	}

	executor := NewCaseExecutor(
		te.variableStore,
		te.authManager,
		timeout,
		retries,
	)

	defer func() {
		te.runTeardown(suite, executor, suiteResult)
	}()

	setupResults := te.runSetup(suite, executor)
	for _, r := range setupResults {
		if r.Status == "failed" {
			suiteResult.TestResults = append(suiteResult.TestResults, setupResults...)
			suiteResult.Error = fmt.Sprintf("setup failed: %s", r.Error)
			suiteResult.EndTime = time.Now()
			suiteResult.Duration = suiteResult.EndTime.Sub(suiteResult.StartTime)
			suiteResult.Total = len(setupResults)
			suiteResult.Failed = 1
			suiteResult.PassRate = 0
			return suiteResult
		}
	}
	suiteResult.TestResults = append(suiteResult.TestResults, setupResults...)

	var callback ProgressCallback
	if te.options.DryRun {
		suiteResult.TestResults = append(suiteResult.TestResults, te.dryRunResults(filteredCases)...)
	} else {
		runner := NewSuiteRunner(
			filteredCases,
			executor,
			suite.BaseURL,
			suite.Auth,
			concurrency,
			callback,
		)
		testResults := runner.Run()
		suiteResult.TestResults = append(suiteResult.TestResults, testResults...)
	}

	suiteResult.EndTime = time.Now()
	suiteResult.Duration = suiteResult.EndTime.Sub(suiteResult.StartTime)

	total := len(suiteResult.TestResults)
	passed := 0
	failed := 0
	skipped := 0

	for _, r := range suiteResult.TestResults {
		switch r.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		case "skipped":
			skipped++
		}
	}

	suiteResult.Total = total
	suiteResult.Passed = passed
	suiteResult.Failed = failed
	suiteResult.Skipped = skipped

	if total > 0 {
		suiteResult.PassRate = float64(passed) / float64(total) * 100
	}

	suiteResult.Environment = te.options.Environment
	suiteResult.Variables = te.variableStore.GetAll()

	return suiteResult
}

func (te *TestEngine) runSetup(suite *models.TestSuite, executor *CaseExecutor) []*models.TestResult {
	if suite.Setup == nil || len(suite.Setup) == 0 {
		return []*models.TestResult{}
	}

	results := executor.ExecuteSetupTeardown(suite.Setup, suite.BaseURL, suite.Auth)
	return results
}

func (te *TestEngine) runTeardown(suite *models.TestSuite, executor *CaseExecutor, suiteResult *models.SuiteResult) {
	if suite.Teardown == nil || len(suite.Teardown) == 0 {
		return
	}

	teardownResults := executor.ExecuteSetupTeardown(suite.Teardown, suite.BaseURL, suite.Auth)
	suiteResult.TestResults = append(suiteResult.TestResults, teardownResults...)
}

func (te *TestEngine) dryRunResults(cases []*models.TestCase) []*models.TestResult {
	results := make([]*models.TestResult, 0, len(cases))

	for _, tc := range cases {
		if tc.ID == "" {
			tc.ID = tc.Name
		}
		results = append(results, &models.TestResult{
			CaseID:   tc.ID,
			CaseName: tc.Name,
			Status:   "skipped",
			SkipReason: "dry run",
		})
	}

	return results
}

func (te *TestEngine) GetVariableStore() *variables.VariableStore {
	return te.variableStore
}

func (te *TestEngine) GetAuthManager() *auth.AuthManager {
	return te.authManager
}
