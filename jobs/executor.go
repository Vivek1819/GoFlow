package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"goflow/workflow"
	"context"
)

var DB *sql.DB

func Execute(ctx context.Context, jobType string, payload map[string]interface{}) (int, []byte, error) {
	switch jobType {

	case "http_request":
		return executeHTTPRequest(ctx, payload)

	case "send_email":
		return executeSendEmail(ctx, payload)

	case "webhook_delivery":
		return executeWebhookDelivery(ctx, payload)

	case "delay":
		return executeDelay(ctx,payload)

	case "cron_schedule":
		return executeCronSchedule(ctx, payload)

	case "data_extract":
		return executeDataExtract(ctx, payload)

	case "ai_prompt":
		return executeAIPrompt(ctx, payload)

	case "db_query":
		return executeDBQuery(ctx, payload)

	case "callback":
		return executeCallback(ctx, payload)

	case "workflow":
		return workflow.Start(ctx, payload)

	default:
		return 0, nil, fmt.Errorf("unknown job type: %s", jobType)
	}
}

func jsonMarshalSafe(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}
