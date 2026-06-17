package engine

import (
	"sync"

	"apitester/internal/models"
)

type ProgressCallback func(progress *models.CLIProgress)

type SuiteRunner struct {
	cases       []*models.TestCase
	executor    *CaseExecutor
	baseURL     string
	authConfig  *models.AuthConfig
	concurrency int
	callback    ProgressCallback
	mu          sync.Mutex
	results     map[string]*models.TestResult
}

func NewSuiteRunner(
	cases []*models.TestCase,
	executor *CaseExecutor,
	baseURL string,
	authConfig *models.AuthConfig,
	concurrency int,
	callback ProgressCallback,
) *SuiteRunner {
	if concurrency <= 0 {
		concurrency = 1
	}

	for _, tc := range cases {
		if tc.ID == "" {
			tc.ID = tc.Name
		}
	}

	return &SuiteRunner{
		cases:       cases,
		executor:    executor,
		baseURL:     baseURL,
		authConfig:  authConfig,
		concurrency: concurrency,
		callback:    callback,
		results:     make(map[string]*models.TestResult),
	}
}

func (sr *SuiteRunner) Run() []*models.TestResult {
	depManager := NewDependencyManager(sr.cases)

	total := len(sr.cases)
	completed := 0
	passed := 0
	failed := 0
	skipped := 0

	jobs := make(chan *models.TestCase, sr.concurrency)
	results := make(chan *models.TestResult, sr.concurrency)

	var wg sync.WaitGroup
	for i := 0; i < sr.concurrency; i++ {
		wg.Add(1)
		go sr.worker(jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for {
			runnable := depManager.GetRunnableCases()

			for _, tc := range runnable {
				depManager.MarkRunning(tc.ID)
				jobs <- tc
			}

			if depManager.AllCompleted() {
				close(jobs)
				break
			}
		}
	}()

	for result := range results {
		sr.mu.Lock()
		sr.results[result.CaseID] = result
		sr.mu.Unlock()

		completed++
		if result.Status == "passed" {
			passed++
			depManager.MarkCompleted(result.CaseID, StatusPassed, "")
		} else if result.Status == "skipped" {
			skipped++
			depManager.MarkCompleted(result.CaseID, StatusSkipped, result.SkipReason)
		} else {
			failed++
			depManager.MarkCompleted(result.CaseID, StatusFailed, result.Error)
		}

		if sr.callback != nil {
			sr.callback(&models.CLIProgress{
				Total:       total,
				Completed:   completed,
				Passed:      passed,
				Failed:      failed,
				Skipped:     skipped,
				CurrentCase: result.CaseName,
				Phase:       "executing",
			})
		}
	}

	allResults := make([]*models.TestResult, 0, len(sr.cases))
	for _, tc := range sr.cases {
		if result, ok := sr.results[tc.ID]; ok {
			allResults = append(allResults, result)
		} else {
			status := depManager.GetStatus(tc.ID)
			skipReason := depManager.GetSkipReason(tc.ID)
			if status == StatusSkipped || skipReason != "" {
				allResults = append(allResults, &models.TestResult{
					CaseID:     tc.ID,
					CaseName:   tc.Name,
					Status:     "skipped",
					SkipReason: skipReason,
				})
				skipped++
			}
		}
	}

	return allResults
}

func (sr *SuiteRunner) worker(jobs <-chan *models.TestCase, results chan<- *models.TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for tc := range jobs {
		result := sr.executor.ExecuteCase(tc, sr.baseURL, sr.authConfig)
		results <- &result
	}
}

func (sr *SuiteRunner) GetResults() map[string]*models.TestResult {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	results := make(map[string]*models.TestResult)
	for k, v := range sr.results {
		results[k] = v
	}
	return results
}
