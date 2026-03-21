package jobs

import (
	"bytes"
	"context"   // ✅ ADD THIS
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func executeHTTPRequest(ctx context.Context, payload map[string]interface{}) (int, []byte, error) {

	url, ok := payload["url"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing url")
	}

	method := "GET"
	if m, ok := payload["method"].(string); ok {
		method = m
	}

	var bodyBytes []byte
	if body, ok := payload["body"]; ok {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second, // keep timeout as fallback
	}

	// ✅ CRITICAL CHANGE — CONTEXT-AWARE REQUEST
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {

		// 🔥 HANDLE CANCELLATION CLEANLY
		if ctx.Err() == context.Canceled {
			return 0, nil, fmt.Errorf("request cancelled")
		}

		return 0, nil, err
	}
	defer resp.Body.Close()

	responseBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return resp.StatusCode, responseBytes,
			fmt.Errorf("http status %d", resp.StatusCode)
	}

	return resp.StatusCode, responseBytes, nil
}