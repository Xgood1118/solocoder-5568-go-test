package websocket

import (
	"apitester/internal/assert"
	"apitester/internal/models"
	"apitester/internal/variables"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketMessageResult struct {
	MessageIndex   int
	MessageType string
	Content    string
	Sent       bool
	Received   bool
	Assertions []*models.AssertionResult
	Extracts   map[string]string
	Error      string
	Timestamp  time.Time
	Latency    time.Duration
}

type WebSocketTestResult struct {
	Connected      bool
	Disconnected bool
	Messages     []*WebSocketMessageResult
	HeartbeatResults []*HeartbeatResult
	Extracts     map[string]string
	Error        string
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
}

type WebSocketTester struct {
	client         *WebSocketClient
	heartbeat    *HeartbeatMonitor
	assertEngine  *assert.AssertionEngine
	interpolator *variables.Interpolator
}

func NewWebSocketTester(interpolator *variables.Interpolator) *WebSocketTester {
	return &WebSocketTester{
		assertEngine: assert.NewAssertionEngine(),
		interpolator: interpolator,
	}
}

func (t *WebSocketTester) Test(ctx context.Context, config *models.WebSocketConfig, testCase *models.TestCase) *WebSocketTestResult {
	result := &WebSocketTestResult{
		StartTime: time.Now(),
		Extracts:  make(map[string]string),
	}

	if config == nil {
		result.Error = "websocket config is nil"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	t.client = NewWebSocketClient(config)

	interpolatedURL, err := t.interpolateString(config.URL)
	if err != nil {
		result.Error = fmt.Sprintf("failed to interpolate URL: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}
	config.URL = interpolatedURL

	for k, v := range config.Headers {
		interpolatedV, err := t.interpolateString(v)
		if err != nil {
			result.Error = fmt.Sprintf("failed to interpolate header %s: %v", k, err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
		config.Headers[k] = interpolatedV
	}

	err = t.client.Connect(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("failed to connect: %v", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}
	result.Connected = true

	testCase.Assertions = t.interpolateAssertions(testCase.Assertions)

	if len(testCase.Assertions) > 0 {
		connectResponse := &models.Response{
			StatusCode: 101,
			Headers:    make(map[string]string),
			Body:       nil,
			BodyString: "",
			Latency:    0,
		}

		assertResults := t.assertEngine.Assert(connectResponse, testCase.Assertions)
		for _, ar := range assertResults {
			if !ar.Passed {
				result.Error = fmt.Sprintf("connection assertion failed: %s", ar.Error)
				t.client.Disconnect()
				result.Disconnected = true
				result.EndTime = time.Now()
				result.Duration = result.EndTime.Sub(result.StartTime)
				return result
			}
		}
	}

	if config.Heartbeat {
		t.heartbeat = NewHeartbeatMonitor(t.client, config.PingInterval, config.Timeout)
		heartbeatChan, err := t.heartbeat.Start(ctx)
		if err != nil {
			result.Error = fmt.Sprintf("failed to start heartbeat: %v", err)
			t.client.Disconnect()
			result.Disconnected = true
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}

		go func() {
			for hr := range heartbeatChan {
				result.HeartbeatResults = append(result.HeartbeatResults, hr)
			}
		}()
	}

	for i, msg := range config.Messages {
		msgResult := t.processMessage(ctx, i, msg, testCase)
		result.Messages = append(result.Messages, msgResult)

		for k, v := range msgResult.Extracts {
			result.Extracts[k] = v
		}

		if msgResult.Error != "" {
			result.Error = msgResult.Error
			break
		}

		hasFailedAssertion := false
		for _, ar := range msgResult.Assertions {
			if !ar.Passed {
				hasFailedAssertion = true
				break
			}
		}
		if hasFailedAssertion {
			break
		}
	}

	if t.heartbeat != nil && t.heartbeat.IsRunning() {
		t.heartbeat.Stop()
	}

	err = t.client.Disconnect()
	if err != nil {
		if result.Error == "" {
			result.Error = fmt.Sprintf("failed to disconnect: %v", err)
		}
	}
	result.Disconnected = true

	if len(testCase.Assertions) > 0 && result.Error == "" {
		disconnectResponse := &models.Response{
			StatusCode: 101,
			Headers:    make(map[string]string),
			Body:       nil,
			BodyString: "",
			Latency:    0,
		}

		assertResults := t.assertEngine.Assert(disconnectResponse, testCase.Assertions)
		for _, ar := range assertResults {
			if !ar.Passed {
				result.Error = fmt.Sprintf("disconnection assertion failed: %s", ar.Error)
				break
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

func (t *WebSocketTester) processMessage(ctx context.Context, index int, msg *models.WebSocketMessage, testCase *models.TestCase) *WebSocketMessageResult {
	result := &WebSocketMessageResult{
		MessageIndex: index,
		MessageType: msg.Type,
		Timestamp:    time.Now(),
		Extracts:   make(map[string]string),
	}

	interpolatedContent, err := t.interpolateString(msg.Content)
	if err != nil {
		result.Error = fmt.Sprintf("failed to interpolate content: %v", err)
		return result
	}

	msg.Content = interpolatedContent
	result.Content = interpolatedContent

	msg.Assertions = t.interpolateAssertions(msg.Assertions)

	msg.Extract = t.interpolateExtractMap(msg.Extract)

	msgType := strings.ToLower(strings.TrimSpace(msg.Type))

	switch msgType {
	case "send", "write":
		result.Sent = true
		startTime := time.Now()

		trimmed := strings.TrimSpace(interpolatedContent)
		isJSON := (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
			(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"))

		if isJSON {
			var jsonData interface{}
			if err := json.Unmarshal([]byte(interpolatedContent), &jsonData); err == nil {
				err = t.client.Send(websocket.TextMessage, []byte(interpolatedContent))
				if err != nil {
					result.Error = fmt.Sprintf("failed to send JSON message: %v", err)
					return result
				}
			} else {
				err = t.client.Send(websocket.TextMessage, []byte(interpolatedContent))
				if err != nil {
					result.Error = fmt.Sprintf("failed to send text message: %v", err)
					return result
				}
			}
		} else {
			err = t.client.Send(websocket.TextMessage, []byte(interpolatedContent))
			if err != nil {
				result.Error = fmt.Sprintf("failed to send text message: %v", err)
				return result
			}
		}

		result.Latency = time.Since(startTime)

	case "receive", "read":
		result.Received = true
		startTime := time.Now()

		receiveCtx := ctx
		if msg.WaitMs > 0 {
			var cancel context.CancelFunc
			receiveCtx, cancel = context.WithTimeout(ctx, time.Duration(msg.WaitMs)*time.Millisecond)
			defer cancel()
		}

		receivedText, err := t.client.ReceiveText(receiveCtx)
		if err != nil {
			result.Error = fmt.Sprintf("failed to receive message: %v", err)
			return result
		}
		result.Content = receivedText
		result.Latency = time.Since(startTime)

		if len(msg.Assertions) > 0 || len(msg.Extract) > 0 {
			response := &models.Response{
				StatusCode: 0,
				Headers:    make(map[string]string),
				Body:       []byte(receivedText),
				BodyString: receivedText,
				Latency:    result.Latency,
			}

			assertResults, extractedVars, err := t.assertEngine.AssertAndExtract(response, msg.Assertions, msg.Extract)
			if err != nil {
				result.Error = fmt.Sprintf("assert and extract failed: %v", err)
				return result
			}
			result.Assertions = assertResults
			for k, v := range extractedVars {
				result.Extracts[k] = v
			}
		}

	case "wait", "sleep":
		if msg.WaitMs > 0 {
			time.Sleep(time.Duration(msg.WaitMs) * time.Millisecond)
		}

	default:
		result.Error = fmt.Sprintf("unknown message type: %s", msg.Type)
		return result
	}

	if msg.WaitMs > 0 && msgType != "wait" {
		time.Sleep(time.Duration(msg.WaitMs) * time.Millisecond)
	}

	return result
}

func (t *WebSocketTester) interpolateString(s string) (string, error) {
	if t.interpolator == nil {
		return s, nil
	}
	return t.interpolator.InterpolateString(s)
}

func (t *WebSocketTester) interpolateAssertions(assertions []*models.Assertion) []*models.Assertion {
	if t.interpolator == nil || len(assertions) == 0 {
		return assertions
	}

	result := make([]*models.Assertion, len(assertions))
	for i, a := range assertions {
		na := &models.Assertion{
			Type:     a.Type,
			Property: a.Property,
			Operator: a.Operator,
			Message:  a.Message,
		}

		prop, err := t.interpolator.InterpolateString(a.Property)
		if err == nil {
			na.Property = prop
		} else {
			na.Property = a.Property
		}

		val, err := t.interpolator.InterpolateInterface(a.Value)
		if err == nil {
			na.Value = val
		} else {
			na.Value = a.Value
		}

		result[i] = na
	}
	return result
}

func (t *WebSocketTester) interpolateExtractMap(extract map[string]string) map[string]string {
	if t.interpolator == nil || len(extract) == 0 {
		return extract
	}

	result := make(map[string]string)
	for k, v := range extract {
		interpolatedV, err := t.interpolator.InterpolateString(v)
		if err == nil {
			result[k] = interpolatedV
		} else {
			result[k] = v
		}
	}
	return result
}

func (t *WebSocketTester) GetClient() *WebSocketClient {
	return t.client
}

func (t *WebSocketTester) GetHeartbeat() *HeartbeatMonitor {
	return t.heartbeat
}
