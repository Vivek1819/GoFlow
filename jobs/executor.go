package jobs

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

var DB *sql.DB

func Execute(jobType string, payload map[string]interface{}) (int, []byte, error) {
	switch jobType {

	case "http_request":
		return executeHTTPRequest(payload)

	case "send_email":
		return executeSendEmail(payload)

	case "webhook_delivery":
		return executeWebhookDelivery(payload)

	case "delay":
		return executeDelay(payload)

	case "cron_schedule":
		return executeCronSchedule(payload)

	case "data_extract":
		return executeDataExtract(payload)

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
