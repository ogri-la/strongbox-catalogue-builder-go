package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"time"
)

// HTTPClient interface for mockable HTTP operations
type HTTPClient interface {
	Get(ctx context.Context, url string) (*Response, error)
}

// Response wraps HTTP response data
type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// RealHTTPClient implements HTTPClient using net/http
type RealHTTPClient struct {
	client    *http.Client
	userAgent string
}

// NewRealHTTPClient creates a new real HTTP client
func NewRealHTTPClient(transport http.RoundTripper, userAgent string) *RealHTTPClient {
	return &RealHTTPClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		userAgent: userAgent,
	}
}

// Get performs an HTTP GET request
func (c *RealHTTPClient) Get(ctx context.Context, url string) (*Response, error) {
	ctx = c.withTrace(ctx)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch '%s': %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Headers:    headers,
	}, nil
}

// withTrace adds HTTP connection tracing to context
func (c *RealHTTPClient) withTrace(ctx context.Context) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			// Logging would be injected here in a real implementation
		},
	})
}

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	responses map[string]*Response
	errors    map[string]error
	calls     []string
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*Response),
		errors:    make(map[string]error),
		calls:     make([]string, 0),
	}
}

// SetResponse sets a mock response for a URL
func (m *MockHTTPClient) SetResponse(url string, response *Response) {
	m.responses[url] = response
}

// SetError sets a mock error for a URL
func (m *MockHTTPClient) SetError(url string, err error) {
	m.errors[url] = err
}

// GetCalls returns all URLs that were called
func (m *MockHTTPClient) GetCalls() []string {
	return m.calls
}

// Get returns a mock response or error
func (m *MockHTTPClient) Get(ctx context.Context, url string) (*Response, error) {
	m.calls = append(m.calls, url)

	if err, exists := m.errors[url]; exists {
		return nil, err
	}

	if resp, exists := m.responses[url]; exists {
		return resp, nil
	}

	return nil, fmt.Errorf("no mock response configured for URL: %s", url)
}