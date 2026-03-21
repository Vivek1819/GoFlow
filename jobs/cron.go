package jobs

import (
	"context" // ✅ ADD
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

func executeCronSchedule(ctx context.Context, payload map[string]interface{}) (int, []byte, error) {

	// 🔴 CANCEL CHECK (EARLY EXIT)
	if ctx.Err() == context.Canceled {
		return 0, nil, fmt.Errorf("cron cancelled")
	}

	cronExpr, ok := payload["cron"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("missing cron expression")
	}

	jobRaw, ok := payload["job"].(map[string]interface{})
	if !ok {
		return 0, nil, fmt.Errorf("missing job definition")
	}

	jobType, ok := jobRaw["type"].(string)
	if !ok {
		return 0, nil, fmt.Errorf("job missing type")
	}

	jobPayload, ok := jobRaw["payload"].(map[string]interface{})
	if !ok {
		return 0, nil, fmt.Errorf("job missing payload")
	}

	// 🔴 CHECK WORKFLOW STATUS (CRITICAL)
	if wfIDRaw, exists := payload["workflow_id"]; exists {

		workflowID := int(wfIDRaw.(float64))

		var status string
		err := DB.QueryRow(`
			SELECT status FROM workflows WHERE id = $1
		`, workflowID).Scan(&status)

		if err == nil && status == "cancelled" {
			return 0, nil, fmt.Errorf("workflow cancelled")
		}
	}

	parser := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)

	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return 0, nil, err
	}

	now := time.Now().UTC()
	nextRun := schedule.Next(now)

	payloadJSON, err := json.Marshal(jobPayload)
	if err != nil {
		return 0, nil, err
	}

	// 🟢 Schedule actual job
	_, err = DB.Exec(`
		INSERT INTO jobs (type, payload, status, run_at)
		VALUES ($1, $2, 'pending', $3)
	`, jobType, payloadJSON, nextRun)

	if err != nil {
		return 0, nil, err
	}

	// 🔴 RECURSIVE CRON — ONLY IF NOT CANCELLED
	if ctx.Err() != context.Canceled {

		fullPayloadJSON, _ := json.Marshal(payload)

		_, err = DB.Exec(`
			INSERT INTO jobs (type, payload, status, run_at)
			VALUES ('cron_schedule', $1, 'pending', $2)
		`, fullPayloadJSON, nextRun)

		if err != nil {
			return 0, nil, err
		}
	}

	result := map[string]interface{}{
		"next_run_at":        nextRun,
		"scheduled_job_type": jobType,
	}

	jsonBytes, _ := json.Marshal(result)

	return 200, jsonBytes, nil
}