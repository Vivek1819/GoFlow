package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

var DB *sql.DB

// ============================
// Start Workflow
// ============================

func Start(payload map[string]interface{}) (int, []byte, error) {

	rawSteps, ok := payload["steps"].([]interface{})
	if !ok || len(rawSteps) == 0 {
		return 0, nil, fmt.Errorf("missing or invalid 'steps'")
	}

	stepsJSON, err := json.Marshal(rawSteps)
	if err != nil {
		return 0, nil, err
	}

	var workflowID int

	err = DB.QueryRow(`
		INSERT INTO workflows (status, current_step, steps)
		VALUES ('running', 0, $1)
		RETURNING id
	`, stepsJSON).Scan(&workflowID)

	if err != nil {
		return 0, nil, err
	}

	// Spawn first step
	firstStep := rawSteps[0].(map[string]interface{})

	stepType := firstStep["type"].(string)
	stepPayload := firstStep["payload"].(map[string]interface{})

	// Inject workflow metadata
	stepPayload["workflow_id"] = workflowID
	stepPayload["step_index"] = 0
	stepPayload["step_id"] = firstStep["id"]

	payloadJSON, _ := json.Marshal(stepPayload)

	_, err = DB.Exec(`
		INSERT INTO jobs (type, payload, status)
		VALUES ($1, $2, 'pending')
	`, stepType, payloadJSON)

	if err != nil {
		return 0, nil, err
	}

	result := map[string]interface{}{
		"workflow_id": workflowID,
		"status":      "running",
	}

	respBytes, _ := json.Marshal(result)
	return workflowID, respBytes, nil
}

// ============================
// Advance Workflow (Sequential)
// ============================

func AdvanceIfNeeded(jobID int, payload map[string]interface{}, response []byte) {

	wfIDRaw, ok := payload["workflow_id"]
	if !ok {
		return
	}

	workflowID := int(wfIDRaw.(float64))
	stepIndex := int(payload["step_index"].(float64))
	stepID := payload["step_id"].(string)

	var stepsJSON []byte
	var contextJSON []byte

	err := DB.QueryRow(`
		SELECT steps, context
		FROM workflows
		WHERE id = $1
	`, workflowID).Scan(&stepsJSON, &contextJSON)

	if err != nil {
		log.Println("Workflow fetch failed:", err)
		return
	}

	// Parse context
	var contextMap map[string]interface{}
	if contextJSON == nil {
		contextMap = make(map[string]interface{})
	} else {
		json.Unmarshal(contextJSON, &contextMap)
	}

	// Store response in context
	var parsed interface{}
	json.Unmarshal(response, &parsed)

	contextMap[stepID] = map[string]interface{}{
		"response": parsed,
	}

	newContextJSON, _ := json.Marshal(contextMap)

	// Update workflow context + step index
	_, err = DB.Exec(`
		UPDATE workflows
		SET context = $2,
		    current_step = current_step + 1,
		    updated_at = NOW()
		WHERE id = $1
	`, workflowID, newContextJSON)

	if err != nil {
		log.Println("Workflow update failed:", err)
		return
	}

	// Parse steps
	var steps []map[string]interface{}
	json.Unmarshal(stepsJSON, &steps)

	nextIndex := stepIndex + 1

	if nextIndex >= len(steps) {
		// Final step completed
		_, err := DB.Exec(`
			UPDATE workflows
			SET status = 'completed',
			    updated_at = NOW()
			WHERE id = $1
		`, workflowID)

		if err != nil {
			log.Println("Workflow completion update failed:", err)
		}

		return
	}

	// Spawn next step
	nextStep := steps[nextIndex]

	nextType := nextStep["type"].(string)
	originalPayload := nextStep["payload"].(map[string]interface{})
	nextPayload := interpolatePayload(originalPayload, contextMap)

	nextPayload["workflow_id"] = workflowID
	nextPayload["step_index"] = nextIndex
	nextPayload["step_id"] = nextStep["id"]

	payloadJSON, _ := json.Marshal(nextPayload)

	_, err = DB.Exec(`
		INSERT INTO jobs (type, payload, status)
		VALUES ($1, $2, 'pending')
	`, nextType, payloadJSON)

	if err != nil {
		log.Println("Failed to insert next workflow job:", err)
	}
}

func interpolatePayload(payload map[string]interface{}, context map[string]interface{}) map[string]interface{} {
	interpolated := make(map[string]interface{})

	for key, value := range payload {
		switch v := value.(type) {

		case string:
			interpolated[key] = interpolateString(v, context)

		case map[string]interface{}:
			interpolated[key] = interpolatePayload(v, context)

		default:
			interpolated[key] = v
		}
	}

	return interpolated
}

var templateRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

func interpolateString(input string, context map[string]interface{}) string {

	matches := templateRegex.FindAllStringSubmatch(input, -1)

	result := input

	for _, match := range matches {
		fullMatch := match[0]
		expression := match[1] // e.g. step1.response.data.name

		value := resolvePath(expression, context)

		if value != "" {
			result = strings.ReplaceAll(result, fullMatch, value)
		}
	}

	return result
}

func resolvePath(path string, context map[string]interface{}) string {

	parts := strings.Split(path, ".")

	if len(parts) < 2 {
		return ""
	}

	stepID := parts[0]

	stepDataRaw, ok := context[stepID]
	if !ok {
		return ""
	}

	stepData, ok := stepDataRaw.(map[string]interface{})
	if !ok {
		return ""
	}

	current := stepData

	for _, part := range parts[1:] {

		next, ok := current[part]
		if !ok {
			return ""
		}

		switch typed := next.(type) {
		case map[string]interface{}:
			current = typed
		default:
			return stringify(typed)
		}
	}

	return ""
}

func stringify(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	default:
		bytes, _ := json.Marshal(val)
		return string(bytes)
	}
}
