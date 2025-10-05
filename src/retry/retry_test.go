package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ogri-la/strongbox-catalogue-builder-go/src/http"
)

func TestWithRetry_Success(t *testing.T) {
	client := http.NewMockHTTPClient()
	client.SetResponse("http://example.com", &http.Response{
		StatusCode: 200,
		Body:       []byte("success"),
	})

	config := DefaultConfig()
	resp, err := WithRetry(context.Background(), client, "http://example.com", config)

	if err != nil {
		t.Fatalf("WithRetry() unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	// Should only make one call
	calls := client.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 call, got %d", len(calls))
	}
}

func TestWithRetry_ServerErrorThenSuccess(t *testing.T) {
	// First call returns 500, second returns 200
	callCount := 0
	client := &mockClientWithCounter{
		counter: &callCount,
		mock:    http.NewMockHTTPClient(),
	}

	client.mock.SetResponse("http://example.com", &http.Response{
		StatusCode: 200,
		Body:       []byte("success"),
	})

	// Use short delays for testing
	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	resp, err := WithRetry(context.Background(), client, "http://example.com", config)

	if err != nil {
		t.Fatalf("WithRetry() unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls (1 failure + 1 success), got %d", callCount)
	}
}

// mockClientWithCounter wraps mock client to allow conditional responses
type mockClientWithCounter struct {
	counter *int
	mock    *http.MockHTTPClient
}

func (m *mockClientWithCounter) Get(ctx context.Context, url string) (*http.Response, error) {
	*m.counter++
	if *m.counter == 1 {
		return &http.Response{StatusCode: 500}, nil
	}
	return m.mock.Get(ctx, url)
}

func TestWithRetry_RateLimit(t *testing.T) {
	// First call returns 429, second returns 200
	callCount := 0
	client := &mockClientWithRateLimit{
		counter: &callCount,
		mock:    http.NewMockHTTPClient(),
	}

	client.mock.SetResponse("http://example.com", &http.Response{
		StatusCode: 200,
		Body:       []byte("success"),
	})

	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	resp, err := WithRetry(context.Background(), client, "http://example.com", config)

	if err != nil {
		t.Fatalf("WithRetry() unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 calls (1 rate limit + 1 success), got %d", callCount)
	}
}

// mockClientWithRateLimit wraps mock client for rate limit testing
type mockClientWithRateLimit struct {
	counter *int
	mock    *http.MockHTTPClient
}

func (m *mockClientWithRateLimit) Get(ctx context.Context, url string) (*http.Response, error) {
	*m.counter++
	if *m.counter == 1 {
		return &http.Response{StatusCode: 429, Headers: map[string]string{"Retry-After": "1"}}, nil
	}
	return m.mock.Get(ctx, url)
}

func TestWithRetry_PermanentClientError(t *testing.T) {
	client := http.NewMockHTTPClient()
	client.SetResponse("http://example.com", &http.Response{
		StatusCode: 404,
		Body:       []byte("not found"),
	})

	config := DefaultConfig()
	resp, err := WithRetry(context.Background(), client, "http://example.com", config)

	if err != nil {
		t.Fatalf("WithRetry() unexpected error: %v", err)
	}

	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}

	// Should only make one call (no retry on 404)
	calls := client.GetCalls()
	if len(calls) != 1 {
		t.Errorf("Expected 1 call (no retry on 404), got %d", len(calls))
	}
}

func TestWithRetry_NetworkErrorExhaustsRetries(t *testing.T) {
	client := http.NewMockHTTPClient()
	client.SetError("http://example.com", errors.New("network error"))

	config := Config{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	_, err := WithRetry(context.Background(), client, "http://example.com", config)

	if err == nil {
		t.Fatal("WithRetry() expected error, got nil")
	}

	// Should make 3 attempts
	calls := client.GetCalls()
	if len(calls) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(calls))
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	client := &mockClientAlways500{}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	config := Config{
		MaxAttempts:  10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	_, err := WithRetry(ctx, client, "http://example.com", config)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// mockClientAlways500 always returns 500 errors
type mockClientAlways500 struct{}

func (m *mockClientAlways500) Get(ctx context.Context, url string) (*http.Response, error) {
	return &http.Response{StatusCode: 500}, nil
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
		want       bool
	}{
		{"Success 200", 200, nil, false},
		{"Success 201", 201, nil, false},
		{"Redirect 301", 301, nil, false},
		{"Client error 400", 400, nil, false},
		{"Not found 404", 404, nil, false},
		{"Rate limit 429", 429, nil, true},
		{"Server error 500", 500, nil, true},
		{"Bad gateway 502", 502, nil, true},
		{"Service unavailable 503", 503, nil, true},
		{"Network error", 0, errors.New("network error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp *http.Response
			if tt.err == nil {
				resp = &http.Response{StatusCode: tt.statusCode}
			}

			got := shouldRetry(resp, tt.err)
			if got != tt.want {
				t.Errorf("shouldRetry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRetryDelay(t *testing.T) {
	config := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     8 * time.Second,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"First retry", 1, 1 * time.Second},
		{"Second retry", 2, 2 * time.Second},
		{"Third retry", 3, 4 * time.Second},
		{"Fourth retry (capped)", 4, 8 * time.Second},
		{"Fifth retry (capped)", 5, 8 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := getRetryDelay(nil, tt.attempt, config)
			if delay != tt.expected {
				t.Errorf("getRetryDelay(attempt=%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestGetRetryDelay_WithRetryAfterHeader(t *testing.T) {
	config := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
	}

	resp := &http.Response{
		StatusCode: 429,
		Headers:    map[string]string{"Retry-After": "5"},
	}

	delay := getRetryDelay(resp, 1, config)
	expected := 5 * time.Second

	if delay != expected {
		t.Errorf("getRetryDelay() with Retry-After = %v, want %v", delay, expected)
	}
}

func TestGetRetryDelay_RetryAfterCapped(t *testing.T) {
	config := Config{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Second,
	}

	resp := &http.Response{
		StatusCode: 429,
		Headers:    map[string]string{"Retry-After": "100"},
	}

	delay := getRetryDelay(resp, 1, config)
	expected := 5 * time.Second // Should be capped at MaxDelay

	if delay != expected {
		t.Errorf("getRetryDelay() with large Retry-After = %v, want %v (capped)", delay, expected)
	}
}
