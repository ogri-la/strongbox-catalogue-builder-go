package retry

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// DefaultConfig returns sensible defaults matching the Clojure version
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     8 * time.Second,
	}
}

// shouldRetry determines if we should retry based on the response or error
func shouldRetry(resp *http.Response, err error) bool {
	// Network errors: retry
	if err != nil {
		return true
	}

	// 429 (rate limit): retry
	if resp.StatusCode == 429 {
		return true
	}

	// 5xx (server errors): retry
	if resp.StatusCode >= 500 {
		return true
	}

	// Everything else (2xx, 3xx, 4xx except 429): don't retry
	return false
}

// getRetryDelay calculates the delay for the next retry
func getRetryDelay(resp *http.Response, attempt int, config Config) time.Duration {
	// Check for Retry-After header on 429 responses
	if resp != nil && resp.StatusCode == 429 {
		if retryAfter := resp.Headers["Retry-After"]; retryAfter != "" {
			// Try parsing as seconds
			if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
				delay := time.Duration(seconds) * time.Second
				// Cap at max delay
				if delay > config.MaxDelay {
					return config.MaxDelay
				}
				return delay
			}
		}
	}

	// Exponential backoff: initialDelay * 2^(attempt-1)
	delay := config.InitialDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > config.MaxDelay {
			return config.MaxDelay
		}
	}
	return delay
}

// WithRetry wraps an HTTP GET call with retry logic and exponential backoff
func WithRetry(ctx context.Context, client http.HTTPClient, url string, config Config) (*http.Response, error) {
	var lastErr error
	var lastResp *http.Response

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Log retry attempts after the first one
		if attempt > 1 {
			slog.Warn("retrying request", "url", url, "attempt", attempt, "max_attempts", config.MaxAttempts)
		}

		resp, err := client.Get(ctx, url)

		// Success case
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		// Store last response/error for potential return
		lastResp = resp
		lastErr = err

		// Check if we should retry
		if !shouldRetry(resp, err) {
			// Don't retry 4xx errors (except 429 which is handled above)
			if err == nil {
				return resp, nil
			}
			return nil, err
		}

		// If this was the last attempt, don't sleep
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay and sleep
		delay := getRetryDelay(resp, attempt, config)
		slog.Info("backing off before retry", "url", url, "delay", delay, "reason", getRetryReason(resp, err))

		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// All attempts exhausted
	if lastErr != nil {
		return nil, fmt.Errorf("request failed after %d attempts: %w", config.MaxAttempts, lastErr)
	}

	// Return the last response (non-200 status)
	return lastResp, nil
}

// getRetryReason returns a human-readable reason for the retry
func getRetryReason(resp *http.Response, err error) string {
	if err != nil {
		return "network_error"
	}
	if resp.StatusCode == 429 {
		return "rate_limited"
	}
	if resp.StatusCode >= 500 {
		return "server_error"
	}
	return "unknown"
}
