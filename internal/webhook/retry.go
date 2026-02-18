package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SendWithRetry sends an HTTP POST request with exponential backoff retry
func SendWithRetry(
	ctx context.Context,
	url string,
	payload []byte,
	headersJSON string,
	timeout time.Duration,
	maxRetries int,
	initialBackoff time.Duration,
	skipTLSVerify bool,
) (statusCode int, success bool, errMsg string, responseBody string, retryCount int) {
	var lastErr error
	var lastStatusCode int
	var lastResponseBody string

	backoff := initialBackoff
	if backoff <= 0 {
		backoff = 2 * time.Second
	}

	if maxRetries <= 0 {
		maxRetries = 1 // at least one attempt
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			slog.Error("Failed to create HTTP request",
				"url", url,
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err)
			continue
		}

		// Set default headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Sentinel-Webhook/1.0")

		// Parse and set custom headers
		if headersJSON != "" {
			var headers map[string]string
			if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
				slog.Warn("Failed to parse custom headers, using defaults",
					"error", err,
					"headers_json", headersJSON)
			} else {
				for key, value := range headers {
					req.Header.Set(key, value)
				}
			}
		}

		// Create client with timeout
		client := &http.Client{
			Timeout: timeout,
		}
		if skipTLSVerify {
			client.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			}
		}

		// Send request
		slog.Debug("Sending webhook request",
			"url", url,
			"attempt", attempt,
			"max_retries", maxRetries,
			"timeout", timeout)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			lastStatusCode = 0
			lastResponseBody = ""

			slog.Warn("Webhook request failed",
				"url", url,
				"attempt", attempt,
				"max_retries", maxRetries,
				"error", err)

			// If this is not the last attempt, wait and retry
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					return 0, false, ctx.Err().Error(), "", attempt - 1
				case <-time.After(backoff):
					// Double the backoff for next attempt (exponential backoff)
					backoff *= 2
					slog.Info("Retrying webhook after backoff",
						"url", url,
						"attempt", attempt+1,
						"backoff", backoff)
					continue
				}
			}

			continue
		}

		// Read response body
		body, err := io.ReadAll(io.LimitReader(resp.Body, 10240)) // limit to 10KB
		resp.Body.Close()

		if err != nil {
			slog.Warn("Failed to read response body",
				"url", url,
				"status", resp.StatusCode,
				"error", err)
			body = []byte(fmt.Sprintf("failed to read response: %v", err))
		}

		lastStatusCode = resp.StatusCode
		lastResponseBody = string(body)

		// Check if request was successful (2xx status code)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			slog.Info("Webhook delivered successfully",
				"url", url,
				"status", resp.StatusCode,
				"attempt", attempt)
			return resp.StatusCode, true, "", lastResponseBody, attempt - 1
		}

		// Request failed with non-2xx status
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))

		slog.Warn("Webhook request returned error status",
			"url", url,
			"status", resp.StatusCode,
			"attempt", attempt,
			"max_retries", maxRetries,
			"response", string(body))

		// If this is not the last attempt and it's a retriable error (5xx), wait and retry
		if attempt < maxRetries && resp.StatusCode >= 500 {
			select {
			case <-ctx.Done():
				return lastStatusCode, false, ctx.Err().Error(), lastResponseBody, attempt - 1
			case <-time.After(backoff):
				// Double the backoff for next attempt (exponential backoff)
				backoff *= 2
				slog.Info("Retrying webhook after backoff",
					"url", url,
					"attempt", attempt+1,
					"backoff", backoff,
					"previous_status", resp.StatusCode)
				continue
			}
		}

		// For 4xx errors, don't retry (client error)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			slog.Error("Webhook request failed with client error, not retrying",
				"url", url,
				"status", resp.StatusCode,
				"attempt", attempt)
			return resp.StatusCode, false, lastErr.Error(), lastResponseBody, attempt - 1
		}
	}

	// All retries exhausted
	errMessage := fmt.Sprintf("all %d attempts failed", maxRetries)
	if lastErr != nil {
		errMessage = fmt.Sprintf("%s: %v", errMessage, lastErr)
	}

	slog.Error("Webhook delivery failed after all retries",
		"url", url,
		"max_retries", maxRetries,
		"last_status", lastStatusCode,
		"error", errMessage)

	return lastStatusCode, false, errMessage, lastResponseBody, maxRetries
}
