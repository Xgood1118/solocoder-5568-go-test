package engine

import (
	"sync"

	"apitester/internal/models"
)

type CaseStatus string

const (
	StatusPending CaseStatus = "pending"
	StatusRunning CaseStatus = "running"
	StatusPassed  CaseStatus = "passed"
	StatusFailed  CaseStatus = "failed"
	StatusSkipped CaseStatus = "skipped"
)

type DependencyManager struct {
	cases     map[string]*models.TestCase
	status    map[string]CaseStatus
	skipReasons map[string]string
	mu        sync.RWMutex
}

func NewDependencyManager(cases []*models.TestCase) *DependencyManager {
	dm := &DependencyManager{
		cases:      make(map[string]*models.TestCase),
		status:     make(map[string]CaseStatus),
		skipReasons: make(map[string]string),
	}

	for _, tc := range cases {
		if tc.ID == "" {
			tc.ID = tc.Name
		}
		dm.cases[tc.ID] = tc
		dm.status[tc.ID] = StatusPending
	}

	return dm
}

func (dm *DependencyManager) CheckDependencies(caseID string) (bool, string) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	tc, ok := dm.cases[caseID]
	if !ok {
		return false, "case not found"
	}

	for _, depID := range tc.DependsOn {
		depStatus, exists := dm.status[depID]
		if !exists {
			return false, "dependency not found: " + depID
		}

		switch depStatus {
		case StatusPending, StatusRunning:
			return false, "dependency not completed: " + depID
		case StatusFailed:
			return false, "dependency failed: " + depID
		case StatusSkipped:
			return false, "dependency skipped: " + depID
		}
	}

	return true, ""
}

func (dm *DependencyManager) MarkCompleted(caseID string, status CaseStatus, skipReason string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, ok := dm.cases[caseID]; ok {
		dm.status[caseID] = status
		if skipReason != "" {
			dm.skipReasons[caseID] = skipReason
		}
	}
}

func (dm *DependencyManager) GetStatus(caseID string) CaseStatus {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if status, ok := dm.status[caseID]; ok {
		return status
	}
	return StatusPending
}

func (dm *DependencyManager) GetSkipReason(caseID string) string {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return dm.skipReasons[caseID]
}

func (dm *DependencyManager) GetRunnableCases() []*models.TestCase {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	var runnable []*models.TestCase

	for caseID, tc := range dm.cases {
		if dm.status[caseID] != StatusPending {
			continue
		}

		canRun := true
		for _, depID := range tc.DependsOn {
			depStatus := dm.status[depID]
			if depStatus == StatusPending || depStatus == StatusRunning {
				canRun = false
				break
			}
			if depStatus == StatusFailed || depStatus == StatusSkipped {
				dm.mu.RUnlock()
				dm.mu.Lock()
				dm.status[caseID] = StatusSkipped
				dm.skipReasons[caseID] = "dependency failed: " + depID
				dm.mu.Unlock()
				dm.mu.RLock()
				canRun = false
				break
			}
		}

		if canRun {
			runnable = append(runnable, tc)
		}
	}

	return runnable
}

func (dm *DependencyManager) AllCompleted() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, status := range dm.status {
		if status == StatusPending || status == StatusRunning {
			return false
		}
	}
	return true
}

func (dm *DependencyManager) MarkRunning(caseID string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, ok := dm.cases[caseID]; ok {
		dm.status[caseID] = StatusRunning
	}
}
