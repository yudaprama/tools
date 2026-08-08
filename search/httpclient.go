package search

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// httpDo executes an HTTP request and returns the response body reader and a
// cleanup function. It handles gzip decoding transparently.
func httpDo(ctx context.Context, client *http.Client, req *http.Request) (io.Reader, func(), error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to execute request: %w", err)
	}

	cleanup := func() { _ = resp.Body.Close() }

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		cleanup()
		return nil, func() {}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var reader io.Reader = resp.Body
	extraCleanup := func() {}
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		extraCleanup = func() { _ = gzReader.Close() }
		reader = gzReader
	}

	return reader, func() { extraCleanup(); cleanup() }, nil
}

// newGetRequest builds a GET request with standard headers.
func newGetRequest(ctx context.Context, fullURL string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}

// hostnameFromURL extracts the hostname from a URL string.
func hostnameFromURL(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil {
		return u.Hostname()
	}
	return ""
}

// elapsed returns milliseconds since startTime.
func elapsed(startTime time.Time) int64 {
	return time.Since(startTime).Milliseconds()
}
