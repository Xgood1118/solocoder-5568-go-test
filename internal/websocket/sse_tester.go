package websocket

import (
	"apitester/internal/assert"
	"apitester/internal/models"
	"apitester/internal/variables"
	"context"
	"fmt"
	"time"
)

type SSEEventResult struct {
	EventIndex  int
	EventName string
	Data      string
	Assertions []*models.AssertionResult
	Extracts   map[string]string
	Error      string
	Timestamp  time.Time
}

type SSETestResult struct {
	Connected   bool
	Disconnected bool
	Events      []*SSEEventResult
	Extracts    map[string]string
	Error       string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
}

type SSETester struct {
	client       *SSEClient
	assertEngine  *assert.AssertionEngine
	interpolator *variables.Interpolator
}

func NewSSETester(interpolator *variables.Interpolator) *SSETester {
	return &SSETester{
		assertEngine: assert.NewAssertionEngine(),
		interpolator: interpolator,
	}
}

func (t *SSETester) Test(ctx context.Context, config *models.SSEConfig, testCase *models.TestCase) *SSETestResult {
	result := &SSETestResult{
		StartTime: time.Now(),
		Extracts:  make(map[string]string),
	}

	if config == nil {
		result.Error = "SSE config is nil"
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	t.client = NewSSEClient(config)

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
			StatusCode: 200,
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

	for i, event := range config.Events {
		eventResult := t.processEvent(ctx, i, event)
		result.Events = append(result.Events, eventResult)

		for k, v := range eventResult.Extracts {
			result.Extracts[k] = v
		}

		if eventResult.Error != "" {
			result.Error = eventResult.Error
			break
		}

		hasFailedAssertion := false
		for _, ar := range eventResult.Assertions {
			if !ar.Passed {
				hasFailedAssertion = true
				break
			}
		}
		if hasFailedAssertion {
			break
		}
	}

	err = t.client.Disconnect()
	if err != nil {
		if result.Error == "" {
			result.Error = fmt.Sprintf("failed to disconnect: %v", err)
		}
	}
	result.Disconnected = true

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	return result
}

func (t *SSETester) processEvent(ctx context.Context, index int, eventConfig *models.SSEEvent) *SSEEventResult {
	result := &SSEEventResult{
		EventIndex: index,
		EventName: eventConfig.EventName,
		Timestamp:  time.Now(),
		Extracts:   make(map[string]string),
	}

	eventConfig.Assertions = t.interpolateAssertions(eventConfig.Assertions)
	eventConfig.Extract = t.interpolateExtractMap(eventConfig.Extract)

	eventName := eventConfig.EventName
	if eventConfig.WaitFor != "" {
		interpolatedWaitFor, err := t.interpolateString(eventConfig.WaitFor)
		if err == nil {
			eventName = interpolatedWaitFor
		}
	}

	count := eventConfig.Count
	if count <= 0 {
		count = 1
	}

	subscribeCtx := ctx
	var cancel context.CancelFunc
	if eventConfig.Count > 0 {
		subscribeCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	events, err := t.client.Subscribe(subscribeCtx, eventName, count)
	if err != nil {
		result.Error = fmt.Sprintf("failed to subscribe to event '%s': %v", eventName, err)
		return result
	}

	if len(events) == 0 {
		result.Error = fmt.Sprintf("no events received for '%s'", eventName)
		return result
	}

	for _, event := range events {
		result.EventName = event.Event
		result.Data = event.Data

		if len(eventConfig.Assertions) > 0 || len(eventConfig.Extract) > 0 {
			response := &models.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Event":   event.Event,
					"Event-Id": event.ID,
				},
				Body:       []byte(event.Data),
				BodyString: event.Data,
				Latency:    time.Since(result.Timestamp),
			}

			assertResults, extractedVars, err := t.assertEngine.AssertAndExtract(response, eventConfig.Assertions, eventConfig.Extract)
			if err != nil {
				result.Error = fmt.Sprintf("assert and extract failed: %v", err)
				return result
			}
			result.Assertions = append(result.Assertions, assertResults...)
			for k, v := range extractedVars {
				result.Extracts[k] = v
			}
		}
	}

	return result
}

func (t *SSETester) interpolateString(s string) (string, error) {
	if t.interpolator == nil {
		return s, nil
	}
	return t.interpolator.InterpolateString(s)
}

func (t *SSETester) interpolateAssertions(assertions []*models.Assertion) []*models.Assertion {
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

func (t *SSETester) interpolateExtractMap(extract map[string]string) map[string]string {
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

func (t *SSETester) GetClient() *SSEClient {
	return t.client
}
