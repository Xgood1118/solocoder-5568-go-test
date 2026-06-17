package engine

import (
	"time"
)

type RetryResult struct {
	Success      bool
	Attempts     int
	TotalDelay   time.Duration
	LastError    error
	AllErrors    []error
}

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
	}
}

func RetryWithBackoff(fn func() error, config *RetryConfig) *RetryResult {
	if config == nil {
		config = DefaultRetryConfig()
	}

	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 5 * time.Second
	}
	if config.Multiplier <= 1.0 {
		config.Multiplier = 2.0
	}

	result := &RetryResult{
		AllErrors: make([]error, 0, config.MaxAttempts),
	}

	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result.Attempts = attempt

		err := fn()
		if err == nil {
			result.Success = true
			return result
		}

		result.LastError = err
		result.AllErrors = append(result.AllErrors, err)

		if attempt < config.MaxAttempts {
			time.Sleep(delay)
			result.TotalDelay += delay

			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return result
}

func RetryWithBackoffAndCondition(fn func() error, shouldRetry func(error) bool, config *RetryConfig) *RetryResult {
	if config == nil {
		config = DefaultRetryConfig()
	}

	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 3
	}
	if config.InitialDelay <= 0 {
		config.InitialDelay = 100 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 5 * time.Second
	}
	if config.Multiplier <= 1.0 {
		config.Multiplier = 2.0
	}

	result := &RetryResult{
		AllErrors: make([]error, 0, config.MaxAttempts),
	}

	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		result.Attempts = attempt

		err := fn()
		if err == nil {
			result.Success = true
			return result
		}

		result.LastError = err
		result.AllErrors = append(result.AllErrors, err)

		if shouldRetry != nil && !shouldRetry(err) {
			return result
		}

		if attempt < config.MaxAttempts {
			time.Sleep(delay)
			result.TotalDelay += delay

			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return result
}
