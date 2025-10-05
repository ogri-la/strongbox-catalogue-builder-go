package cache

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"time"
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	Directory       string
	DefaultTTLHours int
	SearchTTLHours  int
}

// FileCachingTransport implements http.RoundTripper with file-based caching
type FileCachingTransport struct {
	config    CacheConfig
	transport http.RoundTripper
	runStart  time.Time
}

// NewFileCachingTransport creates a new caching transport
func NewFileCachingTransport(config CacheConfig, transport http.RoundTripper) *FileCachingTransport {
	return &FileCachingTransport{
		config:    config,
		transport: transport,
		runStart:  time.Now(),
	}
}

// RoundTrip implements http.RoundTripper with caching
func (t *FileCachingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cacheKey := t.makeCacheKey(req)
	cachePath := t.cachePath(cacheKey)

	// Try to read from cache first
	if cachedResp, err := t.readCacheEntry(cacheKey); err == nil && !t.cacheExpired(cachePath) {
		slog.Info("cache hit", "url", req.URL.String())
		return cachedResp, nil
	}

	// Not in cache or expired, make real request
	slog.Info("fetching", "url", req.URL.String())
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Cache successful responses
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.writeCacheEntry(cacheKey, resp)
	}

	// Return a fresh response from cache to avoid body consumption issues
	if cachedResp, err := t.readCacheEntry(cacheKey); err == nil {
		return cachedResp, nil
	}

	return resp, nil
}

// makeCacheKey creates a cache key from the request
func (t *FileCachingTransport) makeCacheKey(req *http.Request) string {
	key := req.URL.String()
	md5sum := md5.Sum([]byte(key))
	cacheKey := hex.EncodeToString(md5sum[:])

	// Add suffix based on URL type
	if req.URL.Path == "/search" {
		return cacheKey + "-search"
	}
	if filepath.Ext(req.URL.Path) == ".zip" {
		return cacheKey + "-zip"
	}
	if filepath.Base(req.URL.Path) == "filelist.json" {
		return cacheKey + "-filelist"
	}

	return cacheKey
}

// cachePath returns the file path for a cache key
func (t *FileCachingTransport) cachePath(cacheKey string) string {
	return filepath.Join(t.config.Directory, cacheKey)
}

// cacheExpired checks if a cache file has expired
func (t *FileCachingTransport) cacheExpired(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return true // File doesn't exist or can't be read
	}

	// Determine TTL based on cache key suffix
	ttlHours := t.config.DefaultTTLHours
	base := filepath.Base(path)
	if base == "-search" || filepath.Ext(base) == "-search" {
		ttlHours = t.config.SearchTTLHours
	}

	age := t.runStart.Sub(stat.ModTime())
	return age >= time.Duration(ttlHours)*time.Hour
}

// readCacheEntry reads a cached HTTP response
func (t *FileCachingTransport) readCacheEntry(cacheKey string) (*http.Response, error) {
	path := t.cachePath(cacheKey)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return http.ReadResponse(bufio.NewReader(bytes.NewReader(data)), nil)
}

// writeCacheEntry writes an HTTP response to cache
func (t *FileCachingTransport) writeCacheEntry(cacheKey string, resp *http.Response) error {
	path := t.cachePath(cacheKey)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	dumpedBytes, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return fmt.Errorf("failed to dump response: %w", err)
	}

	if err := os.WriteFile(path, dumpedBytes, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}
