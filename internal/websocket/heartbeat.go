package websocket

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type HeartbeatMonitor struct {
	client       *WebSocketClient
	interval     time.Duration
	timeout      time.Duration
	stopChan     chan struct{}
	pongChan     chan struct{}
	mu           sync.Mutex
	running      bool
	lastPongTime time.Time
	failCount    int
	maxFailures  int
}

type HeartbeatResult struct {
	Success    bool
	Latency    time.Duration
	Error      string
	FailCount  int
	Timestamp  time.Time
}

func NewHeartbeatMonitor(client *WebSocketClient, pingIntervalSec int, timeoutSec int) *HeartbeatMonitor {
	interval := time.Duration(pingIntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &HeartbeatMonitor{
		client:      client,
		interval:    interval,
		timeout:     timeout,
		stopChan:    make(chan struct{}),
		pongChan:    make(chan struct{}, 1),
		maxFailures: 3,
	}
}

func (h *HeartbeatMonitor) SetMaxFailures(max int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.maxFailures = max
}

func (h *HeartbeatMonitor) Start(ctx context.Context) (<-chan *HeartbeatResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return nil, fmt.Errorf("heartbeat is already running")
	}

	if !h.client.IsConnected() {
		return nil, fmt.Errorf("websocket client is not connected")
	}

	h.running = true
	h.failCount = 0
	h.lastPongTime = time.Now()
	h.stopChan = make(chan struct{})
	h.pongChan = make(chan struct{}, 1)

	h.client.SetPongHandler(func(string) error {
		h.mu.Lock()
		h.lastPongTime = time.Now()
		h.failCount = 0
		h.mu.Unlock()

		select {
		case h.pongChan <- struct{}{}:
		default:
		}
		return nil
	})

	resultChan := make(chan *HeartbeatResult, 100)

	go h.run(ctx, resultChan)

	return resultChan, nil
}

func (h *HeartbeatMonitor) run(ctx context.Context, resultChan chan<- *HeartbeatResult) {
	defer close(resultChan)

	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			h.running = false
			h.mu.Unlock()
			return

		case <-h.stopChan:
			h.mu.Lock()
			h.running = false
			h.mu.Unlock()
			return

		case <-ticker.C:
			result := h.sendPing()
			resultChan <- result

			if !result.Success {
				h.mu.Lock()
				h.failCount++
				failCount := h.failCount
				h.mu.Unlock()

				if failCount >= h.maxFailures {
					h.mu.Lock()
					h.running = false
					h.mu.Unlock()
					return
				}
			}
		}
	}
}

func (h *HeartbeatMonitor) sendPing() *HeartbeatResult {
	startTime := time.Now()
	result := &HeartbeatResult{
		Timestamp: startTime,
	}

	err := h.client.Ping()
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("ping failed: %v", err)
		h.mu.Lock()
		result.FailCount = h.failCount + 1
		h.mu.Unlock()
		return result
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	select {
	case <-h.pongChan:
		result.Success = true
		result.Latency = time.Since(startTime)
		h.mu.Lock()
		result.FailCount = h.failCount
		h.mu.Unlock()
		return result

	case <-timeoutCtx.Done():
		result.Success = false
		result.Error = fmt.Sprintf("pong timeout after %v", h.timeout)
		h.mu.Lock()
		result.FailCount = h.failCount + 1
		h.mu.Unlock()
		return result
	}
}

func (h *HeartbeatMonitor) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		close(h.stopChan)
		h.running = false
	}
}

func (h *HeartbeatMonitor) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

func (h *HeartbeatMonitor) GetLastPongTime() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.lastPongTime
}

func (h *HeartbeatMonitor) GetFailCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.failCount
}

func (h *HeartbeatMonitor) GetInterval() time.Duration {
	return h.interval
}

func (h *HeartbeatMonitor) GetTimeout() time.Duration {
	return h.timeout
}
