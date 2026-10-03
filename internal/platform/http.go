package platform

import (
	"fmt"
	"net/http"
	"time"
)

// WaitForHTTPUnavailable waits until an HTTP endpoint can no longer be reached.
// It is used by scenarios where the container should remain running while the
// application inside it is unavailable.
func WaitForHTTPUnavailable(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			return nil
		}

		_ = resp.Body.Close()
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf(
		"%s remained reachable within %s",
		url,
		timeout,
	)
}
