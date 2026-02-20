package jobs

import (
	"encoding/json"
	"fmt"
)

func executeDelay(payload map[string]interface{}) (int, []byte, error) {

	secondsFloat, ok := payload["seconds"].(float64)
	if !ok {
		return 0, nil, fmt.Errorf("missing or invalid 'seconds'")
	}
	seconds := int(secondsFloat)

	nextJobRaw, ok := payload["next_job"].(map[string]interface{})
	if !ok {
		return 0, nil, fmt.Errorf("missing or invalid 'next_job'")
	}

	nextType, ok := nextJobRaw["type"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("next_job missing type")
	}

	nextPayload, ok := nextJobRaw["payload"].(map[string]interface{})
	if !ok {
		return 0, nil, fmt.Errorf("next_job missing payload")
	}

	payloadJSON, err := json.Marshal(nextPayload)
	if err != nil {
		return 0, nil, err
	}

	_, err = DB.Exec(`
		INSERT INTO jobs (type, payload, status, run_at)
		VALUES ($1, $2, 'pending', NOW() + ($3 || ' seconds')::interval)
	`, nextType, payloadJSON, seconds)

	if err != nil {
		return 0, nil, err
	}

	result := map[string]interface{}{
		"scheduled_in_seconds": seconds,
		"next_job_type":        nextType,
	}

	jsonBytes, _ := json.Marshal(result)

	return 200, jsonBytes, nil
}