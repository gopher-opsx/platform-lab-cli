package platform

import (
	"fmt"
	"net/http"
	"time"
)

// WaitForHTTPUnavailable waits until an HTTP endpoint can no longer be reached.
func WaitForHTTPUnavailable(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			return nil
		}
		_ = resp.Body.Close()
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("%s remained reachable within %s", url, timeout)
}

// WaitForHTTPStatus waits until an endpoint returns the expected HTTP status.
func WaitForHTTPStatus(url string, expected int, timeout time.Duration) error {
	return WaitForHTTPStatusWithHeaders(url, nil, expected, timeout)
}

// WaitForHTTPStatusWithHeaders waits until an endpoint returns the expected HTTP
// status while sending the supplied request headers.
func WaitForHTTPStatusWithHeaders(
	url string,
	headers map[string]string,
	expected int,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastStatus int

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("create request for %s: %w", url, err)
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err == nil {
			lastStatus = resp.StatusCode
			_ = resp.Body.Close()
			if resp.StatusCode == expected {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("%s did not return HTTP %d within %s; last status: %d",
		url, expected, timeout, lastStatus)
}

// WaitForHTTPFailure waits until an endpoint is no longer successful.
// A non-2xx response or a request error/timeout both count as failure.
func WaitForHTTPFailure(url string, timeout time.Duration) error {
	return WaitForHTTPFailureWithHeaders(url, nil, timeout)
}

// WaitForHTTPFailureWithHeaders waits until an endpoint is no longer successful
// while sending the supplied request headers.
func WaitForHTTPFailureWithHeaders(
	url string,
	headers map[string]string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastStatus int

	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("create request for %s: %w", url, err)
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil
		}

		lastStatus = resp.StatusCode
		_ = resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("%s remained successful within %s; last status: %d",
		url, timeout, lastStatus)
}
