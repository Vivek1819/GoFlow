package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func executeAIPrompt(payload map[string]interface{}) (int, []byte, error) {

	provider, ok := payload["provider"].(string)
	if !ok || provider == "" {
		return 0, nil, fmt.Errorf("missing 'provider'")
	}

	apiKey, ok := payload["api_key"].(string)
	if !ok || apiKey == "" {
		return 0, nil, fmt.Errorf("missing 'api_key'")
	}

	model, ok := payload["model"].(string)
	if !ok || model == "" {
		return 0, nil, fmt.Errorf("missing 'model'")
	}

	prompt, ok := payload["prompt"].(string)
	if !ok || prompt == "" {
		return 0, nil, fmt.Errorf("missing 'prompt'")
	}

	extractContent := false
	if ec, ok := payload["extract_content"].(bool); ok {
		extractContent = ec
	}

	var endpoint string
	var bodyBytes []byte
	var err error

	switch provider {

	case "openai":
		endpoint = "https://api.openai.com/v1/chat/completions"
		bodyBytes, err = buildOpenAIRequest(model, prompt)

	case "groq":
		endpoint = "https://api.groq.com/openai/v1/chat/completions"
		bodyBytes, err = buildOpenAIRequest(model, prompt)

	case "anthropic":
		endpoint = "https://api.anthropic.com/v1/messages"
		bodyBytes, err = buildAnthropicRequest(model, prompt)

	case "gemini":
		endpoint = fmt.Sprintf(
			"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
			model,
			apiKey,
		)
		bodyBytes, err = buildGeminiRequest(prompt)

	default:
		return 0, nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return 0, nil, err
	}

	client := &http.Client{
		Timeout: 25 * time.Second,
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if provider != "gemini" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	if provider == "anthropic" {
		req.Header.Set("anthropic-version", "2023-06-01")
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	if resp.StatusCode >= 400 {
		return resp.StatusCode, responseBytes,
			fmt.Errorf("provider returned status %d", resp.StatusCode)
	}

	if extractContent {
		content, err := extractProviderContent(provider, responseBytes)
		if err != nil {
			return 0, nil, err
		}

		clean := map[string]string{"content": content}
		cleanBytes, _ := json.Marshal(clean)
		return 200, cleanBytes, nil
	}

	return resp.StatusCode, responseBytes, nil
}

func buildOpenAIRequest(model, prompt string) ([]byte, error) {
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	return json.Marshal(body)
}

func buildAnthropicRequest(model, prompt string) ([]byte, error) {
	body := map[string]interface{}{
		"model": model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	return json.Marshal(body)
}

func buildGeminiRequest(prompt string) ([]byte, error) {
	body := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}
	return json.Marshal(body)
}

func extractProviderContent(provider string, response []byte) (string, error) {

	var parsed map[string]interface{}
	if err := json.Unmarshal(response, &parsed); err != nil {
		return "", err
	}

	switch provider {

	case "openai", "groq":
		choices := parsed["choices"].([]interface{})
		first := choices[0].(map[string]interface{})
		message := first["message"].(map[string]interface{})
		return message["content"].(string), nil

	case "anthropic":
		contentArr := parsed["content"].([]interface{})
		first := contentArr[0].(map[string]interface{})
		return first["text"].(string), nil

	case "gemini":
		candidates := parsed["candidates"].([]interface{})
		first := candidates[0].(map[string]interface{})
		content := first["content"].(map[string]interface{})
		parts := content["parts"].([]interface{})
		part := parts[0].(map[string]interface{})
		return part["text"].(string), nil
	}

	return "", fmt.Errorf("unsupported provider for extraction")
}