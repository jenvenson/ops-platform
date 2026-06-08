package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	browserHelperHTTPTimeout = 75 * time.Second
	browserHelperMaxAttempts = 2
)

type BrowserDiscoveryRequest struct {
	EntryURL string           `json:"entry_url"`
	Session  []AuthHeader     `json:"session_headers"`
	Options  DiscoveryOptions `json:"options"`
}

type BrowserDiscoveryResponse struct {
	Targets []DiscoveredTarget `json:"targets"`
}

func DiscoverWithBrowser(entry string, session *WebSession, opts DiscoveryOptions) ([]DiscoveredTarget, error) {
	helperURLSetting := strings.TrimSpace(os.Getenv("OPS_BROWSER_DISCOVERY_URL"))

	helperCmd := strings.TrimSpace(os.Getenv("OPS_BROWSER_DISCOVERY_HELPER"))
	var lastErr error

	req := BrowserDiscoveryRequest{
		EntryURL: sanitizeScanURL(entry),
		Options:  opts,
	}
	if session != nil {
		req.Session = session.Headers
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	for _, helperURL := range browserHelperURLs(helperURLSetting) {
		if targets, err := discoverWithBrowserHTTP(helperURL, payload); err == nil {
			return targets, nil
		} else {
			lastErr = err
		}
	}

	if helperCmd == "" {
		if lastErr != nil {
			return nil, fmt.Errorf("browser discovery helper unavailable: %w", lastErr)
		}
		return nil, fmt.Errorf("browser discovery helper is not configured")
	}

	parts := strings.Fields(helperCmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid browser discovery helper command")
	}

	timeout := 60 * time.Second
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewReader(payload)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			errText := strings.TrimSpace(stderr.String())
			if errText == "" {
				errText = err.Error()
			}
			if lastErr != nil {
				return nil, fmt.Errorf("browser discovery helper failed after http fallback %v: %s", lastErr, errText)
			}
			return nil, fmt.Errorf("browser discovery helper failed: %s", errText)
		}
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("browser discovery helper timed out after %s", timeout)
	}

	var resp BrowserDiscoveryResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("invalid browser discovery response: %w", err)
	}
	return resp.Targets, nil
}

func discoverWithBrowserHTTP(helperURL string, payload []byte) ([]DiscoveredTarget, error) {
	client := &http.Client{Timeout: browserHelperHTTPTimeout}
	var lastErr error

	for attempt := 1; attempt <= browserHelperMaxAttempts; attempt++ {
		req, err := http.NewRequest(http.MethodPost, helperURL, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < browserHelperMaxAttempts {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, err
		}

		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if attempt < browserHelperMaxAttempts {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, readErr
		}
		if resp.StatusCode >= 400 {
			lastErr = fmt.Errorf("browser discovery http helper returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			if attempt < browserHelperMaxAttempts && shouldRetryBrowserHelperStatus(resp.StatusCode) {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, lastErr
		}

		var result BrowserDiscoveryResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("invalid browser discovery response: %w", err)
		}
		return result.Targets, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("browser discovery http helper failed without response")
}

func shouldRetryBrowserHelperStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func browserHelperURLs(setting string) []string {
	if setting == "" {
		return nil
	}

	parts := strings.Split(setting, ",")
	urls := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			urls = append(urls, item)
		}
	}
	return urls
}
