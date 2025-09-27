package http

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestMockHTTPClient(t *testing.T) {
	client := NewMockHTTPClient()
	ctx := context.Background()

	// Test setting and getting responses
	expectedResponse := &Response{
		StatusCode: 200,
		Body:       []byte("test response"),
		Headers:    map[string]string{"Content-Type": "text/html"},
	}

	client.SetResponse("https://example.com", expectedResponse)

	// Test successful response
	resp, err := client.Get(ctx, "https://example.com")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Status code = %d, want 200", resp.StatusCode)
	}

	if string(resp.Body) != "test response" {
		t.Errorf("Body = %s, want 'test response'", string(resp.Body))
	}

	// Test error response
	expectedError := errors.New("network error")
	client.SetError("https://error.com", expectedError)

	_, err = client.Get(ctx, "https://error.com")
	if err == nil {
		t.Error("Expected error but got none")
	}

	if err.Error() != "network error" {
		t.Errorf("Error message = %s, want 'network error'", err.Error())
	}

	// Test unconfigured URL
	_, err = client.Get(ctx, "https://unconfigured.com")
	if err == nil {
		t.Error("Expected error for unconfigured URL but got none")
	}

	// Test call tracking
	calls := client.GetCalls()
	expectedCalls := []string{
		"https://example.com",
		"https://error.com",
		"https://unconfigured.com",
	}

	if len(calls) != len(expectedCalls) {
		t.Errorf("Number of calls = %d, want %d", len(calls), len(expectedCalls))
	}

	for i, call := range calls {
		if call != expectedCalls[i] {
			t.Errorf("Call %d = %s, want %s", i, call, expectedCalls[i])
		}
	}
}

func TestMockHTTPClient_MultipleResponses(t *testing.T) {
	client := NewMockHTTPClient()
	ctx := context.Background()

	// Set up multiple responses for different URLs
	urls := []string{
		"https://api.example.com/data",
		"https://web.example.com/page",
		"https://files.example.com/download",
	}

	for i, url := range urls {
		response := &Response{
			StatusCode: 200,
			Body:       []byte(fmt.Sprintf("response %d", i)),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}
		client.SetResponse(url, response)
	}

	// Test that each URL returns its specific response
	for i, url := range urls {
		resp, err := client.Get(ctx, url)
		if err != nil {
			t.Errorf("Unexpected error for URL %s: %v", url, err)
			continue
		}

		expectedBody := fmt.Sprintf("response %d", i)
		if string(resp.Body) != expectedBody {
			t.Errorf("Body for %s = %s, want %s", url, string(resp.Body), expectedBody)
		}
	}

	// Verify all URLs were called
	calls := client.GetCalls()
	if len(calls) != len(urls) {
		t.Errorf("Number of calls = %d, want %d", len(calls), len(urls))
	}
}

func TestMockHTTPClient_OverrideResponse(t *testing.T) {
	client := NewMockHTTPClient()
	ctx := context.Background()
	url := "https://example.com"

	// Set initial response
	initialResponse := &Response{
		StatusCode: 200,
		Body:       []byte("initial response"),
	}
	client.SetResponse(url, initialResponse)

	// Get initial response
	resp, err := client.Get(ctx, url)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if string(resp.Body) != "initial response" {
		t.Errorf("Initial response body = %s, want 'initial response'", string(resp.Body))
	}

	// Override with new response
	newResponse := &Response{
		StatusCode: 404,
		Body:       []byte("not found"),
	}
	client.SetResponse(url, newResponse)

	// Get new response
	resp, err = client.Get(ctx, url)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("New response status = %d, want 404", resp.StatusCode)
	}
	if string(resp.Body) != "not found" {
		t.Errorf("New response body = %s, want 'not found'", string(resp.Body))
	}
}

