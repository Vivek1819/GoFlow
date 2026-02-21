package jobs

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func executeCallback(payload map[string]interface{}) (int, []byte, error) {

	url, ok := payload["url"].(string)
	if !ok || url == "" {
		return 0, nil, fmt.Errorf("missing 'url'")
	}

	includeResponse := false
	if ir, ok := payload["include_response"].(bool); ok {
		includeResponse = ir
	}

	secret, _ := payload["secret"].(string)

	jobIDFloat, ok := payload["job_id"].(float64)
	if !ok {
		return 0, nil, fmt.Errorf("missing 'job_id'")
	}
	jobID := int(jobIDFloat)

	// Fetch job from DB
	var status string
	var responseBody []byte
	var lastError *string

	err := DB.QueryRow(`
		SELECT status, response_body, last_error
		FROM jobs
		WHERE id = $1
	`, jobID).Scan(&status, &responseBody, &lastError)

	if err != nil {
		return 0, nil, err
	}

	body := map[string]interface{}{
		"job_id": jobID,
		"status": status,
	}

	if includeResponse {
		var parsed interface{}
		if responseBody != nil {
			json.Unmarshal(responseBody, &parsed)
		}
		body["response"] = parsed
	}

	if lastError != nil {
		body["error"] = *lastError
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	// Optional HMAC signing
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(bodyBytes)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-GoFlow-Signature", "sha256="+signature)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return resp.StatusCode, respBytes,
			fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return resp.StatusCode, respBytes, nil
}