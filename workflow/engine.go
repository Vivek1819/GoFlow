package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
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
		INSERT INTO workflows (status, steps)
		VALUES ('running', $1)
		RETURNING id
	`, stepsJSON).Scan(&workflowID)

	if err != nil {
		return 0, nil, err
	}

	// Spawn first step
	firstStep := rawSteps[0].(map[string]interface{})

	stepType := firstStep["type"].(string)
	stepPayload := firstStep["payload"].(map[string]interface{})

	stepPayload["workflow_id"] = workflowID
	stepPayload["step_index"] = 0
	stepPayload["step_id"] = firstStep["id"]

	payloadJSON, _ := json.Marshal(stepPayload)

	var jobID int

	err = DB.QueryRow(`
		INSERT INTO jobs (type, payload, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
	`, stepType, payloadJSON).Scan(&jobID)

	if err != nil {
		return 0, nil, err
	}

	// Insert step run log
	_, err = DB.Exec(`
		INSERT INTO workflow_step_runs (workflow_id, step_id, job_id, status)
		VALUES ($1, $2, $3, 'running')
	`, workflowID, firstStep["id"].(string), jobID)

	if err != nil {
		log.Println("Failed to insert workflow_step_run for first step:", err)
	}

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
// Advance Workflow
// ============================

func AdvanceIfNeeded(jobID int, payload map[string]interface{}, response []byte) {

	wfIDRaw, ok := payload["workflow_id"]
	if !ok {
		return
	}

	workflowID := int(wfIDRaw.(float64))

	// Failure propagation
	var jobStatus string

	err := DB.QueryRow(`
		SELECT status FROM jobs WHERE id = $1
	`, jobID).Scan(&jobStatus)

	if err != nil {
		log.Println("Failed to fetch job status:", err)
		return
	}

	// Update step run
	_, updateErr := DB.Exec(`
		UPDATE workflow_step_runs
		SET status = $1,
			finished_at = NOW(),
			response_snapshot = $2,
			error = CASE WHEN $1 = 'failed' THEN 'Step execution failed' ELSE NULL END
		WHERE job_id = $3
	`, jobStatus, response, jobID)

	if updateErr != nil {
		log.Println("Failed to update workflow_step_run:", updateErr)
	}

	if jobStatus == "failed" {
		DB.Exec(`
			UPDATE workflows
			SET status = 'failed', updated_at = NOW()
			WHERE id = $1
		`, workflowID)
		return
	}

	if branchVal, exists := payload["branch"]; exists {
		if isBranch, ok := branchVal.(bool); ok && isBranch {

			DB.Exec(`
            UPDATE workflows
            SET status = 'completed', updated_at = NOW()
            WHERE id = $1
        `, workflowID)

			return
		}
	}

	stepIndex := int(payload["step_index"].(float64))
	stepID := payload["step_id"].(string)

	var stepsJSON []byte
	var contextJSON []byte

	err = DB.QueryRow(`
		SELECT steps, context FROM workflows WHERE id = $1
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

	// Store response
	var parsed interface{}
	json.Unmarshal(response, &parsed)

	contextMap[stepID] = map[string]interface{}{
		"response": parsed,
	}

	newContextJSON, _ := json.Marshal(contextMap)

	DB.Exec(`
		UPDATE workflows
		SET context = $2, updated_at = NOW()
		WHERE id = $1
	`, workflowID, newContextJSON)

	// Parse steps
	var steps []map[string]interface{}
	json.Unmarshal(stepsJSON, &steps)

	nextIndex := stepIndex + 1

	if nextIndex >= len(steps) {
		DB.Exec(`
			UPDATE workflows
			SET status = 'completed', updated_at = NOW()
			WHERE id = $1
		`, workflowID)
		return
	}

	nextStep := steps[nextIndex]
	nextType := nextStep["type"].(string)

	// Handle condition step
	if nextType == "condition" {
		handleCondition(workflowID, steps, nextStep, contextMap)
		return
	}

	// Normal step
	spawnStep(workflowID, steps, nextIndex, contextMap, false)
}

// ============================
// Condition Handling
// ============================

func handleCondition(workflowID int, steps []map[string]interface{}, step map[string]interface{}, context map[string]interface{}) {

	rawRules, ok := step["rules"].([]interface{})
	if !ok {
		log.Println("Invalid condition rules")
		return
	}

	for _, r := range rawRules {

		rule := r.(map[string]interface{})

		// ELSE rule
		if elseVal, exists := rule["else"]; exists && elseVal.(bool) {
			targetID := rule["next"].(string)
			spawnByID(workflowID, steps, targetID, context)
			return
		}

		if evaluateRule(rule, context) {
			targetID := rule["next"].(string)
			spawnByID(workflowID, steps, targetID, context)
			return
		}
	}
}

// ============================
// Spawn Helpers
// ============================

func spawnStep(workflowID int, steps []map[string]interface{}, index int, context map[string]interface{}, isBranch bool) {

	nextStep := steps[index]

	nextType := nextStep["type"].(string)
	originalPayload := nextStep["payload"].(map[string]interface{})
	nextPayload := interpolatePayload(originalPayload, context)

	nextPayload["workflow_id"] = workflowID
	nextPayload["step_index"] = index
	nextPayload["step_id"] = nextStep["id"]

	if isBranch {
		nextPayload["branch"] = true
	}

	payloadJSON, _ := json.Marshal(nextPayload)

	var jobID int

	err := DB.QueryRow(`
		INSERT INTO jobs (type, payload, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
	`, nextType, payloadJSON).Scan(&jobID)

	if err != nil {
		log.Println("Failed to spawn step:", err)
		return
	}

	_, err = DB.Exec(`
		INSERT INTO workflow_step_runs (workflow_id, step_id, job_id, status)
		VALUES ($1, $2, $3, 'running')
	`, workflowID, nextStep["id"].(string), jobID)

	if err != nil {
		log.Println("Failed to insert workflow_step_run:", err)
	}
}

func spawnByID(workflowID int, steps []map[string]interface{}, targetID string, context map[string]interface{}) {

	index := findStepIndexByID(steps, targetID)
	if index == -1 {
		log.Println("Target step not found:", targetID)
		return
	}

	spawnStep(workflowID, steps, index, context, true)
}

func findStepIndexByID(steps []map[string]interface{}, id string) int {
	for i, s := range steps {
		if s["id"].(string) == id {
			return i
		}
	}
	return -1
}

// ============================
// Rule Evaluation
// ============================

func evaluateRule(rule map[string]interface{}, context map[string]interface{}) bool {

	path := rule["path"].(string)
	operator := rule["operator"].(string)
	expected := rule["value"]

	actualStr := resolvePath(path, context)
	if actualStr == "" {
		return false
	}

	switch operator {
	case "==":
		return actualStr == stringify(expected)
	case "!=":
		return actualStr != stringify(expected)
	case "contains":
		return strings.Contains(actualStr, stringify(expected))
	case ">":
		return compareNumeric(actualStr, expected, ">")
	case "<":
		return compareNumeric(actualStr, expected, "<")
	case ">=":
		return compareNumeric(actualStr, expected, ">=")
	case "<=":
		return compareNumeric(actualStr, expected, "<=")
	}

	return false
}

func compareNumeric(actualStr string, expected interface{}, op string) bool {

	actualFloat, err := strconv.ParseFloat(actualStr, 64)
	if err != nil {
		return false
	}

	expectedFloat, ok := expected.(float64)
	if !ok {
		return false
	}

	switch op {
	case ">":
		return actualFloat > expectedFloat
	case "<":
		return actualFloat < expectedFloat
	case ">=":
		return actualFloat >= expectedFloat
	case "<=":
		return actualFloat <= expectedFloat
	}

	return false
}

// ============================
// Interpolation
// ============================

var templateRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

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

func interpolateString(input string, context map[string]interface{}) string {

	matches := templateRegex.FindAllStringSubmatch(input, -1)
	result := input

	for _, match := range matches {
		fullMatch := match[0]
		expression := match[1]
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

	stepData := stepDataRaw.(map[string]interface{})
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
