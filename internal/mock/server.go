package mock

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"apitester/internal/models"
)

type MockServer struct {
	port    int
	server  *http.Server
	handler *Handler
	rules   []*models.MockRule
	mu      sync.RWMutex
}

func NewMockServer(port int) *MockServer {
	ms := &MockServer{
		port:  port,
		rules: make([]*models.MockRule, 0),
	}
	ms.handler = &Handler{server: ms}
	return ms
}

func (ms *MockServer) Start() error {
	ms.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ms.port),
		Handler: ms.handler,
	}

	errChan := make(chan error, 1)
	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	default:
		return nil
	}
}

func (ms *MockServer) Stop() error {
	if ms.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()
	return ms.server.Shutdown(ctx)
}

func (ms *MockServer) AddRule(rule *models.MockRule) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.rules = append(ms.rules, rule)
}

func (ms *MockServer) AddRules(rules []*models.MockRule) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.rules = append(ms.rules, rules...)
}

func (ms *MockServer) ClearRules() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.rules = make([]*models.MockRule, 0)
}

func (ms *MockServer) getRules() []*models.MockRule {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	rules := make([]*models.MockRule, len(ms.rules))
	copy(rules, ms.rules)
	return rules
}
