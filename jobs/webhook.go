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

func executeWebhookDelivery(payload map[string]interface{}) (int, []byte, error) {

	url, ok := payload["url"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing url")
	}

	event, ok := payload["event"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing event")
	}

	data := payload["data"]

	secret, ok := payload["secret"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing secret")
	}

	bodyMap := map[string]interface{}{
		"event": event,
		"data":  data,
	}

	bodyBytes, err := json.Marshal(bodyMap)
	if err != nil {
		return 0, nil, err
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	client := &http.Client{Timeout: 5 * time.Second}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GoFlow-Signature", "sha256="+signature)

	resp, err := client.Do(req)
	if err != nil {
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