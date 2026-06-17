package websocket

import (
	"apitester/internal/models"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

type SSEClient struct {
	url         string
	headers     http.Header
	client      *http.Client
	resp        *http.Response
	reader      *bufio.Reader
	mu          sync.Mutex
	connected   bool
	eventChan   chan *SSEEvent
	errorChan   chan error
	stopChan    chan struct{}
	lastEventID string
	retry       int
}

func NewSSEClient(config *models.SSEConfig) *SSEClient {
	headers := make(http.Header)
	for k, v := range config.Headers {
		headers.Set(k, v)
	}
	headers.Set("Accept", "text/event-stream")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Connection", "keep-alive")

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 0
	}

	return &SSEClient{
		url:     config.URL,
		headers: headers,
		client: &http.Client{
			Timeout: timeout,
		},
		retry: 3000,
	}
}

func (c *SSEClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, values := range c.headers {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}

	if c.lastEventID != "" {
		req.Header.Set("Last-Event-ID", c.lastEventID)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/event-stream") {
		resp.Body.Close()
		return fmt.Errorf("unexpected content type: %s, expected text/event-stream", contentType)
	}

	c.resp = resp
	c.reader = bufio.NewReader(resp.Body)
	c.connected = true
	c.eventChan = make(chan *SSEEvent, 100)
	c.errorChan = make(chan error, 1)
	c.stopChan = make(chan struct{})

	go c.readEvents()

	return nil
}

func (c *SSEClient) readEvents() {
	defer close(c.eventChan)
	defer close(c.errorChan)

	var eventBuf bytes.Buffer
	var currentEvent *SSEEvent

	for {
		select {
		case <-c.stopChan:
			return
		default:
		}

		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.mu.Lock()
			if c.connected {
				c.errorChan <- fmt.Errorf("read error: %w", err)
			}
			c.mu.Unlock()
			return
		}

		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")

		if line == "" {
			if currentEvent != nil && (currentEvent.Data != "" || currentEvent.Event != "") {
				if currentEvent.ID != "" {
					c.lastEventID = currentEvent.ID
				}
				select {
				case c.eventChan <- currentEvent:
				case <-c.stopChan:
					return
				}
				currentEvent = nil
			}
			eventBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		var field, value string
		if idx := strings.Index(line, ":"); idx != -1 {
			field = line[:idx]
			value = strings.TrimPrefix(line[idx+1:], " ")
		} else {
			field = line
			value = ""
		}

		if currentEvent == nil {
			currentEvent = &SSEEvent{}
		}

		switch field {
		case "id":
			currentEvent.ID = value
		case "event":
			currentEvent.Event = value
		case "data":
			if eventBuf.Len() > 0 {
				eventBuf.WriteString("\n")
			}
			eventBuf.WriteString(value)
			currentEvent.Data = eventBuf.String()
		case "retry":
			if retry, err := strconv.Atoi(value); err == nil {
				c.mu.Lock()
				c.retry = retry
				c.mu.Unlock()
				currentEvent.Retry = retry
			}
		}
	}
}

func (c *SSEClient) Subscribe(ctx context.Context, eventName string, count int) ([]*SSEEvent, error) {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return nil, fmt.Errorf("SSE client is not connected")
	}
	eventChan := c.eventChan
	errorChan := c.errorChan
	c.mu.Unlock()

	events := make([]*SSEEvent, 0)
	received := 0

	for {
		select {
		case <-ctx.Done():
			return events, ctx.Err()

		case err, ok := <-errorChan:
			if ok {
				return events, err
			}
			return events, fmt.Errorf("error channel closed")

		case event, ok := <-eventChan:
			if !ok {
				return events, fmt.Errorf("event channel closed")
			}

			if eventName == "" || event.Event == eventName {
				events = append(events, event)
				received++

				if count > 0 && received >= count {
					return events, nil
				}
			}
		}
	}
}

func (c *SSEClient) Events() <-chan *SSEEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.eventChan
}

func (c *SSEClient) Errors() <-chan error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.errorChan
}

func (c *SSEClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	close(c.stopChan)

	if c.resp != nil && c.resp.Body != nil {
		c.resp.Body.Close()
	}

	c.connected = false
	return nil
}

func (c *SSEClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

func (c *SSEClient) GetLastEventID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastEventID
}

func (c *SSEClient) GetRetry() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.retry
}

func (c *SSEClient) SetHeader(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.headers == nil {
		c.headers = make(http.Header)
	}
	c.headers.Set(key, value)
}
