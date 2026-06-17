package websocket

import (
	"apitester/internal/models"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	url       string
	headers   http.Header
	protocols []string
	timeout   time.Duration
	isConnected bool
}

func NewWebSocketClient(config *models.WebSocketConfig) *WebSocketClient {
	headers := make(http.Header)
	for k, v := range config.Headers {
		headers.Set(k, v)
	}

	timeout := time.Duration(config.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &WebSocketClient{
		url:       config.URL,
		headers:   headers,
		protocols: config.Protocols,
		timeout:   timeout,
	}
}

func (c *WebSocketClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return nil
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: c.timeout,
		Subprotocols:     c.protocols,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, c.headers)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true
	return nil
}

func (c *WebSocketClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected || c.conn == nil {
		return nil
	}

	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		c.conn.Close()
		c.isConnected = false
		return err
	}

	c.conn.Close()
	c.isConnected = false
	return nil
}

func (c *WebSocketClient) Send(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.conn.WriteMessage(messageType, data)
}

func (c *WebSocketClient) SendText(data string) error {
	return c.Send(websocket.TextMessage, []byte(data))
}

func (c *WebSocketClient) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.conn.WriteJSON(v)
}

func (c *WebSocketClient) Receive(ctx context.Context) (int, []byte, error) {
	if !c.isConnected || c.conn == nil {
		return 0, nil, fmt.Errorf("websocket is not connected")
	}

	type result struct {
		messageType int
		data        []byte
		err         error
	}

	done := make(chan result, 1)

	go func() {
		c.conn.SetReadDeadline(time.Time{})
		messageType, data, err := c.conn.ReadMessage()
		done <- result{messageType, data, err}
	}()

	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case res := <-done:
		return res.messageType, res.data, res.err
	}
}

func (c *WebSocketClient) ReceiveText(ctx context.Context) (string, error) {
	_, data, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *WebSocketClient) ReceiveJSON(ctx context.Context, v interface{}) error {
	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	type result struct {
		err error
	}

	done := make(chan result, 1)

	go func() {
		c.conn.SetReadDeadline(time.Time{})
		err := c.conn.ReadJSON(v)
		done <- result{err}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case res := <-done:
		return res.err
	}
}

func (c *WebSocketClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.isConnected = false
	}
	return nil
}

func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

func (c *WebSocketClient) Ping() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("websocket is not connected")
	}

	c.conn.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.conn.WriteMessage(websocket.PingMessage, nil)
}

func (c *WebSocketClient) SetPongHandler(handler func(string) error) {
	if c.conn != nil {
		c.conn.SetPongHandler(handler)
	}
}

func (c *WebSocketClient) SetPingHandler(handler func(string) error) {
	if c.conn != nil {
		c.conn.SetPingHandler(handler)
	}
}
